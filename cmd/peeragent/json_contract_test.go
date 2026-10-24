package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/result"
)

func TestLaunchAndStatusJSONJobID(t *testing.T) {
	cwd, _ := fakeMessagingEnvironment(t)
	launched := cliResult(t, cwd, "--async", "--cwd", cwd, "task")
	if launched.JobID == "" || launched.JobID != launched.Metadata.JobID {
		t.Fatalf("launch aliases %#v", launched)
	}
	status := cliResult(t, cwd, "--status", launched.JobID)
	if status.JobID != launched.JobID || status.Metadata.JobID != launched.JobID {
		t.Fatalf("status aliases %#v", status)
	}
}

func TestWaitJSONSingleDocumentWithUnreadNoticeAndFailedResult(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	job := indexedTestJob(t, cwd)
	// A legacy result has only metadata.job_id; formatting must mirror it.
	want := result.Result{Status: result.StatusFailed, Summary: "failed task", Metadata: result.Metadata{JobID: job.ID, ExitCode: 23}}
	stored, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(job.ResultPath, stored, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.NewService(app.Options{}).Tell(context.Background(), app.JobRequest{CWD: cwd, JobID: job.ID}, "launcher", "unread answer"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PEERAGENT_JOB_ID", job.ID)
	t.Setenv("PEERAGENT_JOB_DIR", filepath.Dir(job.LogPath))
	stdout, stderr, err := controlOutput(t, cwd, "--wait", job.ID)
	if err != nil {
		t.Fatalf("wait %v %s", err, stderr)
	}
	got := decodeResult(t, []byte(stdout)) // json.Unmarshal rejects a second document.
	if got.Status != result.StatusFailed || got.Summary != want.Summary || got.JobID != job.ID || got.Metadata.JobID != job.ID {
		t.Fatalf("wait result %#v", got)
	}
	if strings.Count(stderr, "unread message(s)") != 1 {
		t.Fatalf("diagnostics %q", stderr)
	}
}
