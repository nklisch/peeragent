package app

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestTerminalResultsSurviveRemovedJobDirectory(t *testing.T) {
	for _, status := range []result.Status{result.StatusSuccess, result.StatusFailed, result.StatusAuthRequired, result.StatusInterrupted, result.StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, testJobSpec(), "private task prompt")
			if err != nil {
				t.Fatal(err)
			}
			if err := jobs.IndexJob(job.ID, cwd); err != nil {
				t.Fatal(err)
			}
			message := "complete handback"
			res := result.Result{Status: status, Summary: "outcome", FinalMessage: &message, Details: "retained details", Metadata: result.Metadata{AgentSession: "archived-session", ExitCode: 1}}
			if status == result.StatusAuthRequired || status == result.StatusInterrupted {
				res.FinalMessage = nil
			}
			if status == result.StatusFailed {
				res.Failure = "policy_refusal"
				res.Summary = "This content was flagged for possible cybersecurity risk."
			}
			if err := FinishJob(store, job, res); err != nil {
				t.Fatal(err)
			}
			data, err := jobs.ReadArchivedResult(job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "private task prompt") {
				t.Fatal("prompt copied into snapshot")
			}
			// This test owns the whole temporary repository; removing it exercises the
			// same lookup as removing an implementation worktree after completion.
			if err := os.RemoveAll(cwd); err != nil {
				t.Fatal(err)
			}
			s := NewService(Options{WorkingDirectory: func() (string, error) { return t.TempDir(), nil }})
			for _, read := range []func(context.Context, JobRequest) (result.Result, error){s.JobResult, s.PollJobResult, s.CancelJob} {
				got, err := read(context.Background(), JobRequest{JobID: job.ID})
				if err != nil || got.Status != status || got.Metadata.CWD != cwd || got.Metadata.AgentSession != "archived-session" || got.Details != "retained details" || !reflect.DeepEqual(got.FinalMessage, res.FinalMessage) || got.Failure != res.Failure {
					t.Fatalf("archived result: %#v %v", got, err)
				}
			}
			got, err := s.JobStatus(context.Background(), JobRequest{JobID: job.ID})
			if err != nil || got.Status != status || got.FinalMessage != nil || got.Details != "" || got.Failure != res.Failure {
				t.Fatalf("archived status: %#v %v", got, err)
			}
			got, err = s.JobResult(context.Background(), JobRequest{CWD: t.TempDir(), JobID: job.ID})
			if err != nil || got.Metadata.ExitCode != 4 {
				t.Fatalf("explicit cwd ignored: %#v %v", got, err)
			}
		})
	}
}

func TestArchiveFailureKeepsRepositoryCompletion(t *testing.T) {
	state := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", state)
	if err := os.MkdirAll(filepath.Join(state, "index"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "index", "results"), []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := FinishJob(store, job, result.Result{Status: result.StatusSuccess, Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	got, err := NewService(Options{}).JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || got.Status != result.StatusSuccess || !strings.Contains(got.Metadata.IndexWarning, "archive terminal result") {
		t.Fatalf("completion lost: %#v %v", got, err)
	}
}

func TestArchiveDoesNotMaskCorruptLocalJob(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := FinishJob(store, job, result.Result{Status: result.StatusSuccess, Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Root, job.ID, "job.json"), []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = NewService(Options{}).JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err == nil || !strings.Contains(err.Error(), "load async job") {
		t.Fatalf("corrupt local state masked: %v", err)
	}
}
