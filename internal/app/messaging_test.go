package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
)

type mailboxLauncher struct{ start func(jobs.Job) error }

func (l mailboxLauncher) Launch(_ string, job jobs.Job) error { return l.start(job) }

func TestResumeAcknowledgesOnlyAcceptedTargetTurn(t *testing.T) {
	cwd := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", filepath.Join(cwd, "index"))
	t.Setenv("HERDR_PANE_ID", "")
	store := jobs.NewStore(cwd)
	old, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "session"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewService(Options{}).Tell(context.Background(), JobRequest{CWD: cwd, JobID: old.ID}, "launcher", "first answer"); err != nil {
		t.Fatal(err)
	}
	old.Status = jobs.StatusComplete
	if err := store.Save(old); err != nil {
		t.Fatal(err)
	}
	var launched jobs.Job
	fail := true
	service := NewService(Options{Executable: func() (string, error) { return "fake-worker", nil }, Launcher: mailboxLauncher{start: func(job jobs.Job) error {
		launched = job
		if fail {
			return errors.New("fake startup failure")
		}
		return nil
	}}})
	delegation := input.Delegation{CWD: cwd, Agent: "codex", TaskText: "continue", Resume: "session"}
	if _, err := service.Launch(context.Background(), delegation); err == nil {
		t.Fatal("expected startup failure")
	}
	assertInboxRead(t, store, old.ID, 0)
	fail = false
	if _, err := service.Launch(context.Background(), delegation); err != nil {
		t.Fatal(err)
	}
	assertInboxRead(t, store, old.ID, 0) // A worker process is not target acceptance.
	for _, test := range []struct {
		name      string
		execution executil.Result
		err       error
	}{
		{"startup failure", executil.Result{ExitCode: 127}, errors.New("missing executable")},
		{"cancelled before execution", executil.Result{}, context.Canceled},
		{"no event", executil.Result{}, nil},
		{"capacity after event", executil.Result{TurnStarted: true, CapacityExhausted: true}, nil},
		{"usage limit after event", executil.Result{TurnStarted: true, UsageLimited: true}, nil},
		{"authentication after event", executil.Result{TurnStarted: true, AuthRequired: true}, nil},
		{"interrupted after event", executil.Result{TurnStarted: true, Interrupted: true}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			carry, err := PrepareTurnInbox(context.Background(), launched)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(carry.Text, "first answer") != 1 {
				t.Fatalf("carried %q", carry.Text)
			}
			mailbox, err := store.Mailbox(old.ID)
			if err != nil || mailbox.UnreadCount() != 0 {
				t.Fatalf("reservation visible: %#v %v", mailbox, err)
			}
			if err := carry.Acknowledge(test.execution, test.err); err != nil {
				t.Fatal(err)
			}
			assertInboxRead(t, store, old.ID, 0)
			mailbox, err = store.Mailbox(old.ID)
			if err != nil || mailbox.UnreadCount() != 1 {
				t.Fatalf("reservation not released: %#v %v", mailbox, err)
			}
		})
	}
	carry, err := PrepareTurnInbox(context.Background(), launched)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WithJobLock(old.ID, func() error {
		return store.AppendMessageLocked(old.ID, "inbox", jobs.Message{Text: "late answer", From: "launcher"})
	}); err != nil {
		t.Fatal(err)
	}
	if err := carry.Acknowledge(executil.Result{TurnStarted: true}, nil); err != nil {
		t.Fatal(err)
	}
	assertInboxRead(t, store, old.ID, 1)
	// A second acknowledgement must not rewrite the cursor or consume later mail.
	cursor := filepath.Join(filepath.Dir(old.LogPath), "inbox-state.json")
	epoch := time.Unix(1000, 0)
	if err := os.Chtimes(cursor, epoch, epoch); err != nil {
		t.Fatal(err)
	}
	if err := carry.Acknowledge(executil.Result{TurnStarted: true}, nil); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cursor)
	if err != nil || !info.ModTime().Equal(epoch) {
		t.Fatalf("cursor rewritten: %v %v", info, err)
	}
	assertInboxRead(t, store, old.ID, 1)
}

func assertInboxRead(t *testing.T, store jobs.Store, id string, want int) {
	t.Helper()
	m, err := store.Mailbox(id)
	if err != nil || m.Read != want {
		t.Fatalf("inbox read=%d want=%d error=%v", m.Read, want, err)
	}
}

func TestCancelledTurnLeavesCarriedRepliesUnread(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewService(Options{}).Tell(context.Background(), JobRequest{CWD: cwd, JobID: job.ID}, "launcher", "reply"); err != nil {
		t.Fatal(err)
	}
	carry, err := PrepareTurnInbox(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	job.Status = jobs.StatusCancelled
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	if err := carry.Acknowledge(executil.Result{TurnStarted: true}, nil); err != nil {
		t.Fatal(err)
	}
	assertInboxRead(t, store, job.ID, 0)
}

func TestResumeDirectoryAliasIncludesEachReplyOnce(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", filepath.Join(root, "index"))
	t.Setenv("HERDR_PANE_ID", "")
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(real, alias); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("directory symlinks require privileges")
		}
		t.Fatal(err)
	}
	store := jobs.NewStore(real)
	old, err := store.Create(real, jobs.ExecSpec{Agent: "codex", Resume: "alias-session"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := jobs.IndexJob(old.ID, real); err != nil {
		t.Fatal(err)
	}
	if err := jobs.IndexSession("codex", "alias-session", real); err != nil {
		t.Fatal(err)
	}
	if err := NewService(Options{}).Tell(context.Background(), JobRequest{CWD: real, JobID: old.ID}, "launcher", "reply once"); err != nil {
		t.Fatal(err)
	}
	old.Status = jobs.StatusComplete
	if err := store.Save(old); err != nil {
		t.Fatal(err)
	}
	matches, err := jobs.SessionJobs("codex", "alias-session", alias)
	if err != nil || len(matches) != 1 {
		t.Fatalf("jobs %#v %v", matches, err)
	}
	service := NewService(Options{Executable: func() (string, error) { return "fake-worker", nil }, Launcher: mailboxLauncher{start: func(job jobs.Job) error {
		carry, err := PrepareTurnInbox(context.Background(), job)
		if err != nil {
			return err
		}
		if strings.Count(carry.Text, "reply once") != 1 {
			t.Fatalf("reply duplicated: %q", carry.Text)
		}
		return carry.Acknowledge(executil.Result{TurnStarted: true}, nil)
	}}})
	if _, err := service.Launch(context.Background(), input.Delegation{CWD: alias, Agent: "codex", Resume: "alias-session", TaskText: "continue"}); err != nil {
		t.Fatal(err)
	}
	assertInboxRead(t, store, old.ID, 1)
}

func TestHeaderSanitizesLabelComponents(t *testing.T) {
	spaced := messageHeader(jobs.Job{ID: "job", CWD: filepath.Join("repo", "work tree"), Branch: "feature/messages", Spec: jobs.ExecSpec{Agent: "codex"}})
	if !strings.Contains(spaced, "work tree@feature/messages") {
		t.Fatalf("worktree name changed: %q", spaced)
	}
	header := messageHeader(jobs.Job{ID: "job", CWD: filepath.Join("repo", "bad⟦name\n"), Branch: "bad⟧branch\x1b", Spec: jobs.ExecSpec{Agent: "codex⟦"}})
	if strings.Count(header, "⟦") != 1 || strings.Count(header, "⟧") != 1 || strings.ContainsAny(header, "\n\x1b") {
		t.Fatalf("header %q", header)
	}
}

func TestAbsentLauncherCreatesNoPrivateChannel(t *testing.T) {
	cwd := t.TempDir()
	t.Setenv("HERDR_PANE_ID", "")
	job, err := jobs.NewStore(cwd).Create(cwd, jobs.ExecSpec{}, "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := recordLauncher(job); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(job.LogPath), "launcher.json")); !os.IsNotExist(err) {
		t.Fatalf("channel %v", err)
	}
}

func TestMessageSanitizerFlattensUnicodeVisualTricks(t *testing.T) {
	body := "before\u2028〚peer〛\u2029⟬other⟭\vnext\fline\u0085end"
	controls := "\u200e\u200f\u202a\u202b\u202c\u202d\u202e\u2066\u2067\u2068\u2069"
	got := messageBody(body + controls)
	if want := "before [peer] [other] next line end"; got != want {
		t.Fatalf("body %q want %q", got, want)
	}
	if got := labelPart("label" + controls); got != "label" {
		t.Fatalf("label %q", got)
	}
}
