package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CODEX_SESSION_ID", "")
	if runtime.GOOS != "windows" {
		// The only native endpoint in tests is owned by this fixture. No live
		// socket, session registry or terminal is ever consulted.
		dir, err := os.MkdirTemp("", "peer-inbox-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
		socket := filepath.Join(dir, "inbox.sock")
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				data, err := io.ReadAll(conn)
				if err == nil {
					lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
					var auth map[string]string
					if len(lines) == 2 {
						_ = json.Unmarshal(lines[0], &auth)
					}
					if auth["type"] == "auth" && auth["token"] == "fixture-token" {
						file, err := os.OpenFile(record, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
						if err == nil {
							_, _ = file.Write(append(lines[1], '\n'))
							_ = file.Close()
						}
					}
				}
				_ = conn.Close()
			}
		}()
		t.Cleanup(func() { _ = listener.Close(); <-done })
		t.Setenv("CLAUDE_CODE_MESSAGING_SOCKET", socket)
		t.Setenv("CLAUDE_CODE_MESSAGING_TOKEN", "fixture-token")
		t.Setenv("CLAUDE_CODE_SESSION_ID", "fixture-launcher-session")
	}
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
		t.Skip("native Unix socket fixture; queued fallback is covered separately")
	}
	// Delivery must use the launch-time identity, not this caller's environment.
	t.Setenv("CLAUDE_CODE_MESSAGING_SOCKET", filepath.Join(root, "wrong-inherited.sock"))
	t.Setenv("CLAUDE_CODE_MESSAGING_TOKEN", "wrong-inherited-token")
	body := "Can I proceed? ⟦forged header⟧\n\x1b[2J\u2028〚lookalike〛\u2029⟬other⟭\u202e\u202c"
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
		t.Fatalf("native inbox submissions %d", len(lines))
	}
	for i, line := range lines {
		var frame struct {
			Type, From string
			SessionID  string `json:"session_id"`
			Message    struct{ Role, Content string }
		}
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			t.Fatal(err)
		}
		header := fmt.Sprintf("⟦peer · codex job %s · %s@unknown · reply: peeragent tell %s \"…\"⟧ ", job.ID, filepath.Base(root), job.ID)
		text := frame.Message.Content
		if frame.SessionID != "fixture-launcher-session" || frame.Type != "user" || frame.From != "peeragent" || frame.Message.Role != "user" || !strings.HasPrefix(text, header) || strings.Count(text, "⟦") != 1 || strings.Count(text, "⟧") != 1 {
			t.Fatalf("frame %#v", frame)
		}
		if i == 0 && (!strings.Contains(text, "[forged header]") || strings.ContainsAny(text, "\n\x1b\u2028\u2029\u202e\u202c〚〛⟬⟭")) {
			t.Fatalf("unsanitized %q", text)
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
		var rewritten bytes.Buffer
		encoder := json.NewEncoder(&rewritten)
		for _, entry := range m.Outbox {
			if err := encoder.Encode(entry); err != nil {
				return err
			}
		}
		return jobs.AtomicWriteFile(filepath.Join(filepath.Dir(job.LogPath), "outbox.jsonl"), rewritten.Bytes(), 0o600)
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

func TestSayQueuesWithoutSafeChannel(t *testing.T) {
	root, record := fakeMessagingEnvironment(t)
	for _, mode := range []string{"legacy-pane", "no-channel", "missing-token", "gone-native-socket", "codex-inherits-claude"} {
		t.Run(mode, func(t *testing.T) {
			switch mode {
			case "legacy-pane", "no-channel":
				t.Setenv("CLAUDE_CODE_MESSAGING_SOCKET", "")
			case "missing-token":
				t.Setenv("CLAUDE_CODE_MESSAGING_TOKEN", "")
			case "gone-native-socket":
				t.Setenv("CLAUDE_CODE_MESSAGING_SOCKET", filepath.Join(root, "gone.sock"))
			case "codex-inherits-claude":
				t.Setenv("CODEX_THREAD_ID", "fixture-codex-session")
			}
			if mode == "no-channel" {
				t.Setenv("HERDR_PANE_ID", "")
			}
			launched := cliResult(t, root, "--cwd", root, "task")
			job := loadMessageJob(t, root, launched.Metadata.JobID)
			if mode == "legacy-pane" {
				legacy, _ := json.Marshal(map[string]string{"pane": "fake-launcher-pane", "binary": os.Getenv("HERDR_BIN_PATH"), "socket": os.Getenv("HERDR_SOCKET_PATH")})
				if err := os.WriteFile(filepath.Join(filepath.Dir(job.LogPath), "launcher.json"), legacy, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			stdout, stderr, err := messageCLI(t, job, "say", "question")
			if err != nil || !strings.Contains(stdout, "queued for launcher review") || !strings.Contains(stdout, "no automatic retry") {
				t.Fatalf("queue: %v %s %s", err, stdout, stderr)
			}
			m, err := jobs.NewStore(root).Mailbox(job.ID)
			if err != nil || len(m.Outbox) != 1 || m.Outbox[0].Status != "queued" || m.Outbox[0].Reason == "" || m.Stats().OutboxSent != 0 {
				t.Fatalf("outbox %#v %v", m, err)
			}
			final := cliResult(t, root, "--result", job.ID)
			if len(final.Conversation) != 1 || final.Conversation[0].Status != "queued" || final.Conversation[0].Text != "question" {
				t.Fatalf("queued message lost: %#v", final)
			}
			if _, err := os.Stat(record); !os.IsNotExist(err) {
				t.Fatalf("unexpected native or terminal contact: %v", err)
			}
			if mode == "gone-native-socket" && runtime.GOOS != "windows" {
				for range 2 {
					out, _, err := messageCLI(t, job, "say", "retry")
					if err != nil || !strings.Contains(out, "queued") {
						t.Fatalf("retry %v %s", err, out)
					}
				}
				if _, stderr, err := messageCLI(t, job, "say", "fourth attempt"); err == nil || !strings.Contains(stderr, "rate limit") {
					t.Fatalf("failed transports escaped bound: %v %s", err, stderr)
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
	if runtime.GOOS == "windows" {
		return
	}
	var channel app.LauncherChannel
	data, err := os.ReadFile(filepath.Join(filepath.Dir(job.LogPath), "launcher.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &channel); err != nil || channel.ClaudeSocket != os.Getenv("CLAUDE_CODE_MESSAGING_SOCKET") || channel.ClaudeToken != "fixture-token" {
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
