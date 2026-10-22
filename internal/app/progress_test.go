package app

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestProgressPreservesTerminalWinner(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := RecordJobProgress(store, job.ID, executil.Progress{AgentSession: "live-session", Message: "first message"}); err != nil {
		t.Fatal(err)
	}
	job, err = store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := FinishJob(store, job, result.Result{Status: result.StatusCancelled, Metadata: result.Metadata{AgentSession: "live-session"}}); err != nil {
		t.Fatal(err)
	}
	if err := RecordJobProgress(store, job.ID, executil.Progress{AgentSession: "late-session", Message: "late message", FinalReceived: true}); err != nil {
		t.Fatal(err)
	}
	job, err = store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(job.MessagePath)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != jobs.StatusCancelled || job.Spec.Resume != "live-session" || string(text) != "first message" {
		t.Fatalf("terminal state overwritten: %#v message %q", job, text)
	}
}

func TestWaitPollDoesNotReadRunningMessage(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := RecordJobProgress(store, job.ID, executil.Progress{AgentSession: "live-session", Message: strings.Repeat("message", 20000)}); err != nil {
		t.Fatal(err)
	}
	job, err = store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(job.MessagePath); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	res, err := service.PollJobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || res.Status != result.StatusRunning || res.Metadata.AgentSession != "live-session" || res.Details != "" {
		t.Fatalf("wait poll read message: %#v %v", res, err)
	}
	if _, err := service.JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID}); err == nil {
		t.Fatal("explicit result did not report missing message file")
	}
}
