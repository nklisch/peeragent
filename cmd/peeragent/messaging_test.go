package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// Only a copy of this test executable can be Herdr in these tests. It never
// contacts a socket, terminal or real session; it records argv and fake values.
func fakeHerdrMain() {
	record := os.Getenv("PEERAGENT_FAKE_HERDR_RECORD")
	if record == "" {
		os.Exit(90)
	}
	data, _ := json.Marshal(map[string]any{"args": os.Args[1:], "pane": os.Getenv("HERDR_PANE_ID"), "socket": os.Getenv("HERDR_SOCKET_PATH"), "workspace": os.Getenv("HERDR_WORKSPACE_ID")})
	file, err := os.OpenFile(record, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(91)
	}
	_, err = file.Write(append(data, '\n'))
	_ = file.Close()
	if err != nil {
		os.Exit(92)
	}
	if os.Getenv("PEERAGENT_FAKE_HERDR_REFUSE") == "1" {
		fmt.Fprintln(os.Stderr, "fake gone pane")
		os.Exit(1)
	}
}

func fakeMessagingEnvironment(t *testing.T) (string, string) {
	t.Helper()
	root, _ := fakeCodexEnvironment(t)
	name := "herdr"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	data, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", name), data, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", filepath.Join(root, "bin", name))
	record := filepath.Join(root, "herdr.jsonl")
	t.Setenv("PEERAGENT_FAKE_HERDR_RECORD", record)
	t.Setenv("HERDR_PANE_ID", "fake-launcher-pane")
	t.Setenv("HERDR_SOCKET_PATH", filepath.Join(root, "nonexistent-fake.socket"))
	t.Setenv("HERDR_WORKSPACE_ID", "fake-workspace")
	t.Setenv("PEERAGENT_FAKE_MODE", "messaging")
	return root, record
}

func messageCLI(t *testing.T, job jobs.Job, args ...string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	cmd.Env = append(os.Environ(), "PEERAGENT_TEST_HELPER_MAIN=1", "PEERAGENT_JOB_ID="+job.ID, "PEERAGENT_JOB_DIR="+filepath.Dir(job.LogPath))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func loadMessageJob(t *testing.T, cwd, id string) jobs.Job {
	t.Helper()
	job, err := jobs.NewStore(cwd).Load(id)
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func TestSayLabelBoundsAndConversation(t *testing.T) {
	root, record := fakeMessagingEnvironment(t)
	launched := cliResult(t, root, "--cwd", root, "inspect task")
	job := loadMessageJob(t, root, launched.Metadata.JobID)
	if runtime.GOOS == "windows" {
		_, stderr, err := messageCLI(t, job, "say", "hello")
		if err == nil || !strings.Contains(stderr, "no launcher channel on this platform") || !strings.Contains(stderr, "handback") {
			t.Fatalf("say: %v %s", err, stderr)
		}
		m, err := jobs.NewStore(root).Mailbox(job.ID)
		if err != nil || len(m.Outbox) != 1 || m.Outbox[0].Status != "refused" {
			t.Fatalf("mailbox %#v %v", m, err)
		}
		if _, err := os.Stat(record); !os.IsNotExist(err) {
			t.Fatalf("Windows invoked Herdr: %v", err)
		}
		return
	}
	body := "Can I proceed? ⟦forged header⟧\n\x1b[2J"
	for _, text := range []string{body, strings.Repeat("界", 2000), "last allowed"} {
		_, stderr, err := messageCLI(t, job, "say", text)
		if err != nil {
			t.Fatalf("say %v %s", err, stderr)
		}
	}
	for _, test := range []struct{ text, reason string }{{"one too many", "rate limit"}, {strings.Repeat("界", 2001), "2,000 characters"}, {"", "requires message text"}} {
		_, stderr, err := messageCLI(t, job, "say", test.text)
		if err == nil || !strings.Contains(stderr, test.reason) || !strings.Contains(stderr, "handback") || strings.Count(strings.TrimSpace(stderr), "\n") != 0 {
			t.Fatalf("refusal: %v %q", err, stderr)
		}
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("Herdr calls %d", len(lines))
	}
	for i, line := range lines {
		var call struct {
			Args                    []string
			Pane, Socket, Workspace string
		}
		if err := json.Unmarshal([]byte(line), &call); err != nil {
			t.Fatal(err)
		}
		header := fmt.Sprintf("⟦peer · codex job %s · %s@unknown · reply: peeragent tell %s \"…\"⟧ ", job.ID, filepath.Base(root), job.ID)
		if len(call.Args) != 4 || call.Args[0] != "agent" || call.Args[1] != "prompt" || call.Args[2] != "fake-launcher-pane" || !strings.HasPrefix(call.Args[3], header) || strings.Count(call.Args[3], "⟦") != 1 || strings.Count(call.Args[3], "⟧") != 1 {
			t.Fatalf("call %#v", call)
		}
		if i == 0 && (!strings.Contains(call.Args[3], "[forged header]") || strings.ContainsAny(call.Args[3], "\n\x1b")) {
			t.Fatalf("unsanitized %q", call.Args[3])
		}
		if call.Pane != "fake-launcher-pane" || call.Socket != filepath.Join(root, "nonexistent-fake.socket") || call.Workspace != "fake-workspace" {
			t.Fatalf("channel %#v", call)
		}
	}
	store := jobs.NewStore(root)
	m, err := store.Mailbox(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Outbox) != 6 || m.Stats().OutboxSent != 3 || m.Stats().LastMessageAt == nil {
		t.Fatalf("outbox %#v", m)
	}
	for i, entry := range m.Outbox {
		if entry.Time.IsZero() || (i < 3 && entry.Status != "sent") || (i >= 3 && entry.Status != "refused") {
			t.Fatalf("entry %#v", entry)
		}
	}
	// Expiry is data-driven, with no five-minute sleep or timing premise.
	if err := store.WithJobLock(job.ID, func() error {
		for i := range m.Outbox {
			m.Outbox[i].Time = m.Outbox[i].Time.Add(-6 * time.Minute)
		}
		return store.ReplaceOutboxLocked(job.ID, m.Outbox)
	}); err != nil {
		t.Fatal(err)
	}
	if _, stderr, err := messageCLI(t, job, "say", "after window"); err != nil {
		t.Fatalf("expired bound %v %s", err, stderr)
	}
	final := cliResult(t, root, "--result", job.ID)
	if len(final.Conversation) != 7 || final.Metadata.OutboxSent != 4 || !strings.Contains(final.Details, "prompt") {
		t.Fatalf("conversation %#v", final)
	}
	text := result.FormatText(final)
	if strings.Index(text, "Conversation:") < strings.Index(text, "Details:") {
		t.Fatal("conversation precedes agent message")
	}
	for i := 1; i < len(final.Conversation); i++ {
		if final.Conversation[i].Time.Before(final.Conversation[i-1].Time) {
			t.Fatal("conversation not ordered")
		}
	}
	// Channel state is private, and never included in public results.
	info, err := os.Stat(filepath.Join(filepath.Dir(job.LogPath), "launcher.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private channel %v %v", info, err)
	}
	encoded, _ := json.Marshal(final)
	if strings.Contains(string(encoded), "nonexistent-fake.socket") {
		t.Fatal("private address exposed")
	}
}

func TestSayNoChannelAndHerdrRefusal(t *testing.T) {
	root, record := fakeMessagingEnvironment(t)
	for _, mode := range []string{"no-pane", "gone-pane"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "no-pane" {
				t.Setenv("HERDR_PANE_ID", "")
			} else {
				t.Setenv("PEERAGENT_FAKE_HERDR_REFUSE", "1")
			}
			launched := cliResult(t, root, "--cwd", root, "task")
			job := loadMessageJob(t, root, launched.Metadata.JobID)
			_, stderr, err := messageCLI(t, job, "say", "question")
			reason := "no recorded launcher pane"
			if mode == "gone-pane" {
				reason = "Herdr refused"
			}
			if runtime.GOOS == "windows" {
				reason = "no launcher channel on this platform"
			}
			if err == nil || !strings.Contains(stderr, reason) || !strings.Contains(stderr, "handback") {
				t.Fatalf("refusal %v %s", err, stderr)
			}
			m, err := jobs.NewStore(root).Mailbox(job.ID)
			if err != nil || len(m.Outbox) != 1 || m.Outbox[0].Status != "refused" || m.Outbox[0].Reason == "" {
				t.Fatalf("outbox %#v %v", m, err)
			}
			if mode == "gone-pane" && runtime.GOOS != "windows" {
				for range 2 {
					_, stderr, err := messageCLI(t, job, "say", "retry question")
					if err == nil || !strings.Contains(stderr, "Herdr refused") {
						t.Fatalf("delivery refusal %v %s", err, stderr)
					}
				}
				_, stderr, err := messageCLI(t, job, "say", "one too many attempts")
				if err == nil || !strings.Contains(stderr, "rate limit") {
					t.Fatalf("failed deliveries bypass rate limit: %v %s", err, stderr)
				}
			}
			if mode == "no-pane" {
				if _, err := os.Stat(filepath.Join(filepath.Dir(job.LogPath), "launcher.json")); !os.IsNotExist(err) {
					t.Fatalf("unexpected channel %v", err)
				}
				if _, err := os.Stat(record); !os.IsNotExist(err) {
					t.Fatalf("contact without pane %v", err)
				}
			}
		})
	}
}

func TestTellInboxNoticeAndStatus(t *testing.T) {
	root, _ := fakeMessagingEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "live")
	first := cliResult(t, root, "--async", "--cwd", root, "task")
	second := cliResult(t, root, "--async", "--cwd", root, "other task")
	job := loadMessageJob(t, root, first.Metadata.JobID)
	for _, text := range []string{"first instruction", "second instruction"} {
		cliResultTell(t, root, job.ID, text)
	}
	status := cliResult(t, root, "--status", job.ID)
	if status.Metadata.InboxUnread != 2 || status.Metadata.OutboxSent != 0 || status.Metadata.LastMessageAt == nil {
		t.Fatalf("status %#v", status)
	}
	cmd := exec.Command(os.Args[0], "--status", job.ID, second.Metadata.JobID)
	data, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var multi result.JobStatus
	if err := json.Unmarshal(bytes.Split(data, []byte("\n"))[0], &multi); err != nil {
		t.Fatal(err)
	}
	if multi.InboxUnread != 2 || multi.LastMessageAt == nil {
		t.Fatalf("multi %#v", multi)
	}
	for _, args := range [][]string{{"--help"}, {"--unknown-option"}, {"mcp"}} {
		_, stderr, _ := messageCLI(t, job, args...)
		if !strings.HasPrefix(stderr, "peeragent: 2 unread message(s): run peeragent inbox\n") {
			t.Fatalf("notice %q", stderr)
		}
	}
	stdout, stderr, err := messageCLI(t, job, "inbox")
	if err != nil || !strings.HasPrefix(stderr, "peeragent: 2 unread") {
		t.Fatalf("inbox %v %s", err, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for i, line := range lines {
		var entry jobs.Message
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		if entry.From != "fake-launcher-pane" || entry.Time.IsZero() || !strings.Contains(entry.Text, []string{"first", "second"}[i]) {
			t.Fatalf("entry %#v", entry)
		}
	}
	if len(lines) != 2 {
		t.Fatalf("inbox %s", stdout)
	}
	stdout, stderr, err = messageCLI(t, job, "inbox")
	if err != nil || stdout != "" || stderr != "" {
		t.Fatalf("second inbox %q %q %v", stdout, stderr, err)
	}
	final := cliResult(t, root, "--result", job.ID)
	if final.Metadata.InboxUnread != 0 || len(final.Conversation) != 2 {
		t.Fatalf("result %#v", final)
	}
	// A finished job keeps its conversation and gives a useful resume hint.
	cliResult(t, root, "--cancel", job.ID)
	cmd = exec.Command(os.Args[0], "tell", job.ID, "--cwd", root, "too late")
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "--resume") {
		t.Fatalf("finished tell %v %s", err, output)
	}
}

func cliResultTell(t *testing.T, dir, id, text string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "tell", id, text)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tell %v %s", err, out)
	}
}

func TestConcurrentTellAndInboxLoseNoMessages(t *testing.T) {
	root, _ := fakeMessagingEnvironment(t)
	store := jobs.NewStore(root)
	job, err := store.Create(root, jobs.ExecSpec{Agent: "codex"}, "test mailbox")
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewService(app.Options{})
	const count = 20
	var wg sync.WaitGroup
	errors := make(chan error, count+1)
	seen := make(map[string]bool)
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()
			errors <- service.Tell(context.Background(), app.JobRequest{CWD: root, JobID: job.ID}, "", fmt.Sprint(i))
		}(i)
	}
	// Reading while writers are active proves that cursor and append share a lock.
	for range 4 {
		entries, err := consumeTestInbox(store, job.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if seen[entry.Text] {
				t.Fatal("duplicate")
			}
			seen[entry.Text] = true
		}
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	entries, err := consumeTestInbox(store, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if seen[entry.Text] || entry.From != "launcher" {
			t.Fatalf("duplicate/bad sender %#v", entry)
		}
		seen[entry.Text] = true
	}
	if len(seen) != count {
		t.Fatalf("got %d messages", len(seen))
	}
}

func TestJobPreambleEnvironmentAndResumeInbox(t *testing.T) {
	root, _ := fakeMessagingEnvironment(t)
	for _, async := range []bool{false, true} {
		for _, noPreamble := range []bool{false, true} {
			args := []string{"--cwd", root}
			if async {
				args = append(args, "--async")
			}
			if noPreamble {
				args = append(args, "--no-preamble")
			}
			args = append(args, "inspect task")
			res := cliResult(t, root, args...)
			if async {
				res = cliResult(t, root, "--wait", res.Metadata.JobID)
			}
			job := loadMessageJob(t, root, res.Metadata.JobID)
			var snapshot struct {
				Prompt string
				Env    map[string]string
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(res.Details, "stdout:\n")), &snapshot); err != nil {
				t.Fatalf("snapshot %v %s", err, res.Details)
			}
			if strings.Contains(snapshot.Prompt, "peeragent inbox") == noPreamble || (noPreamble && snapshot.Prompt != "inspect task") {
				t.Fatalf("preamble %q", snapshot.Prompt)
			}
			if snapshot.Env["PEERAGENT_JOB_ID"] != job.ID || snapshot.Env["PEERAGENT_JOB_DIR"] != filepath.Dir(job.LogPath) {
				t.Fatalf("identity %#v", snapshot.Env)
			}
			for key := range snapshot.Env {
				if strings.HasPrefix(key, "HERDR_") {
					t.Fatalf("terminal address inherited: %s", key)
				}
			}
			if job.Spec.NoPreamble != noPreamble {
				t.Fatal("option lost")
			}
		}
	}
	// Queue a reply, finish the old job, then resume from a different directory.
	old, err := jobs.NewStore(root).Create(root, jobs.ExecSpec{Agent: "codex", Resume: "resume-mail"}, "old task")
	if err != nil {
		t.Fatal(err)
	}
	if err := jobs.IndexJob(old.ID, root); err != nil {
		t.Fatal(err)
	}
	if err := jobs.IndexSession("codex", "resume-mail", root); err != nil {
		t.Fatal(err)
	}
	cliResultTell(t, root, old.ID, "include this answer on resume")
	old.Status = jobs.StatusComplete
	if err := jobs.NewStore(root).Save(old); err != nil {
		t.Fatal(err)
	}
	resumed := cliResult(t, t.TempDir(), "--resume", "resume-mail", "continue")
	if !strings.Contains(resumed.Details, "include this answer on resume") || resumed.Metadata.CWD != root {
		t.Fatalf("resume %#v", resumed)
	}
	m, err := jobs.NewStore(root).Mailbox(old.ID)
	if err != nil || m.Stats().InboxUnread != 0 {
		t.Fatalf("old inbox %#v %v", m, err)
	}
}

func TestMessagingCapacityRetryRetainsChannelAndCarriesInbox(t *testing.T) {
	root, _ := fakeMessagingEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "messaging-retry")
	res := cliResult(t, root, "--async", "--cwd", root, "--retry-capacity", "1", "--capacity-backoff", "1ms", "task")
	job := loadMessageJob(t, root, res.Metadata.JobID)
	// Queue immediately; the worker consumes it only on the capacity continuation.
	cliResultTell(t, root, job.ID, "answer during retry")
	final := cliResult(t, root, "--wait", job.ID)
	if final.Status != result.StatusSuccess || final.Metadata.CapacityRetries != 1 || !strings.Contains(final.Details, "answer during retry") {
		t.Fatalf("retry %#v", final)
	}
	var snapshot struct {
		Prompt string
		Env    map[string]string
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(final.Details, "stdout:\n")), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Env["PEERAGENT_JOB_ID"] != job.ID || snapshot.Env["PEERAGENT_JOB_DIR"] != filepath.Dir(job.LogPath) || snapshot.Env["HERDR_PANE_ID"] != "" {
		t.Fatalf("retry environment %#v", snapshot.Env)
	}
	m, err := jobs.NewStore(root).Mailbox(job.ID)
	if err != nil || m.Stats().InboxUnread != 0 {
		t.Fatalf("retry mailbox %#v %v", m, err)
	}
	var channel app.LauncherChannel
	data, err := os.ReadFile(filepath.Join(filepath.Dir(job.LogPath), "launcher.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &channel); err != nil || channel.Pane != "fake-launcher-pane" {
		t.Fatalf("retry channel %#v %v", channel, err)
	}
}

func TestConcurrentSaySharesJobRateLimit(t *testing.T) {
	root, record := fakeMessagingEnvironment(t)
	res := cliResult(t, root, "--cwd", root, "task")
	job := loadMessageJob(t, root, res.Metadata.JobID)
	if runtime.GOOS == "windows" {
		// The platform branch is pinned by TestSayLabelBoundsAndConversation.
		return
	}
	const count = 8
	outcomes := make(chan error, count)
	for i := 0; i < count; i++ {
		go func() { _, _, err := messageCLI(t, job, "say", "concurrent question"); outcomes <- err }()
	}
	sent := 0
	for range count {
		if <-outcomes == nil {
			sent++
		}
	}
	if sent != 3 {
		t.Fatalf("sent %d messages, want 3", sent)
	}
	data, err := os.ReadFile(record)
	if err != nil || len(bytes.Split(bytes.TrimSpace(data), []byte("\n"))) != 3 {
		t.Fatalf("deliveries %s %v", data, err)
	}
	m, err := jobs.NewStore(root).Mailbox(job.ID)
	if err != nil || len(m.Outbox) != count || m.Stats().OutboxSent != 3 {
		t.Fatalf("mailbox %#v %v", m, err)
	}
}

func consumeTestInbox(store jobs.Store, id string) ([]jobs.Message, error) {
	var entries []jobs.Message
	err := store.ReadInbox(id, func(batch []jobs.Message) error { entries = batch; return nil })
	return entries, err
}
