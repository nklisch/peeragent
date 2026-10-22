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

func TestProgressPreviewSurvivesToolActivityAndRetry(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "work")
	if err != nil {
		t.Fatal(err)
	}
	reports := []executil.Progress{
		{Event: "item.completed", AgentSession: "same", Message: "Building\n  the library."},
		{Event: "item.started", AgentSession: "same"},
		{Event: "thread.started", AgentSession: "same"}, // fresh retry parser has no message yet
	}
	for _, report := range reports {
		if err := RecordJobProgress(store, job.ID, report); err != nil {
			t.Fatal(err)
		}
	}
	progress, err := jobs.ReadProgress(job.LogPath)
	if err != nil || progress.At == nil || progress.Event != "thread.started" || progress.LastMessage != "Building the library." {
		t.Fatalf("retry preview: %#v %v", progress, err)
	}
	// Status never opens the full report, even when the report disappears.
	job, err = store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(job.MessagePath); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	single, err := service.JobStatus(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || single.Metadata.LastMessage != progress.LastMessage || single.Metadata.LastEventAt == nil {
		t.Fatalf("single status: %#v %v", single, err)
	}
	multi, err := service.JobStatuses(context.Background(), []JobRequest{{CWD: cwd, JobID: job.ID}, {CWD: cwd, JobID: job.ID}})
	if err != nil || len(multi) != 2 || multi[0].LastMessage != progress.LastMessage {
		t.Fatalf("multi status: %#v %v", multi, err)
	}
}
