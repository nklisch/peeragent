package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
)

type mailboxLauncher struct{ start func(jobs.Job) error }

func (l mailboxLauncher) Launch(_ string, job jobs.Job) error { return l.start(job) }

func TestResumeAcknowledgesSnapshotOnlyAfterSuccessfulStart(t *testing.T) {
	cwd := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", filepath.Join(cwd, "index"))
	t.Setenv("HERDR_PANE_ID", "")
	store := jobs.NewStore(cwd)
	old, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "session"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	req := JobRequest{CWD: cwd, JobID: old.ID}
	if err := service.Tell(context.Background(), req, "launcher", "first answer"); err != nil {
		t.Fatal(err)
	}
	old.Status = jobs.StatusComplete
	if err := store.Save(old); err != nil {
		t.Fatal(err)
	}
	fail := true
	service = NewService(Options{Executable: func() (string, error) { return "fake-worker", nil }, Launcher: mailboxLauncher{start: func(job jobs.Job) error {
		prompt, err := store.ReadPrompt(job.ID)
		if err != nil {
			return err
		}
		if !strings.Contains(prompt, "first answer") {
			t.Fatal("missing resume answer")
		}
		if fail {
			return errors.New("fake startup failure")
		}
		// Simulate a concurrent late append in a deliberately overlapping session.
		return store.WithJobLock(old.ID, func() error {
			return store.AppendMessageLocked(old.ID, "inbox", jobs.Message{Text: "late answer", From: "launcher"})
		})
	}}})
	delegation := input.Delegation{CWD: cwd, Agent: "codex", TaskText: "continue", Resume: "session"}
	if _, err := service.Launch(context.Background(), delegation); err == nil {
		t.Fatal("expected startup failure")
	}
	m, err := store.Mailbox(old.ID)
	if err != nil || m.Stats().InboxUnread != 1 {
		t.Fatalf("failed startup consumed inbox %#v %v", m, err)
	}
	fail = false
	if _, err := service.Launch(context.Background(), delegation); err != nil {
		t.Fatal(err)
	}
	m, err = store.Mailbox(old.ID)
	if err != nil || m.Read != 1 || len(m.Inbox) != 2 {
		t.Fatalf("late answer lost %#v %v", m, err)
	}
}

func TestCapacityContinuationCarriesUnreadInbox(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewService(Options{}).Tell(context.Background(), JobRequest{CWD: cwd, JobID: job.ID}, "launcher", "retry answer"); err != nil {
		t.Fatal(err)
	}
	text, err := CarryRetryInbox(job)
	if err != nil || !strings.Contains(text, "retry answer") {
		t.Fatalf("carry %q %v", text, err)
	}
	text, err = CarryRetryInbox(job)
	if err != nil || text != "" {
		t.Fatalf("duplicate carry %q %v", text, err)
	}
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
