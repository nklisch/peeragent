package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestJobStatusMapsPersistedStates(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status jobs.Status
		want   result.Status
	}{
		{name: "running", status: jobs.StatusRunning, want: result.StatusRunning},
		{name: "complete", status: jobs.StatusComplete, want: result.StatusSuccess},
		{name: "failed", status: jobs.StatusFailed, want: result.StatusFailed},
		{name: "cancelled", status: jobs.StatusCancelled, want: result.StatusCancelled},
		{name: "unknown is conservative", status: jobs.Status("future-state"), want: result.StatusRunning},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, testJobSpec(), "do work")
			if err != nil {
				t.Fatal(err)
			}
			job.Status = tt.status
			if err := store.Save(job); err != nil {
				t.Fatal(err)
			}

			service := NewService(Options{WorkingDirectory: func() (string, error) { return cwd, nil }})
			got, err := service.JobStatus(context.Background(), JobRequest{JobID: job.ID})
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tt.want || got.Metadata.CWD != cwd || got.Metadata.JobID != job.ID {
				t.Fatalf("result = %#v, want status=%q cwd=%q id=%q", got, tt.want, cwd, job.ID)
			}
		})
	}
}

func TestJobStatusMappingsUsePersistedLifecycleConstants(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status result.Status
		want   jobs.Status
	}{
		{name: "success", status: result.StatusSuccess, want: jobs.StatusComplete},
		{name: "cancelled", status: result.StatusCancelled, want: jobs.StatusCancelled},
		{name: "failed", status: result.StatusFailed, want: jobs.StatusFailed},
		{name: "running remains failed for persisted completion", status: result.StatusRunning, want: jobs.StatusFailed},
		{name: "blocked remains failed for persisted completion", status: result.StatusBlocked, want: jobs.StatusFailed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := JobStatusFromResult(tt.status); got != tt.want {
				t.Fatalf("JobStatusFromResult(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestJobResultReturnsRunningUntilResultExists(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})

	got, err := service.JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != result.StatusRunning || got.Metadata.JobID != job.ID {
		t.Fatalf("result = %#v, want running result", got)
	}

	stored := result.Result{
		Status:       result.StatusFailed,
		Summary:      "target failed",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID, ExitCode: 23},
	}
	if err := WriteJobResult(job.ResultPath, stored); err != nil {
		t.Fatal(err)
	}
	got, err = service.JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != stored.Status || got.Summary != stored.Summary || got.Metadata.ExitCode != 23 {
		t.Fatalf("decoded result = %#v, want %#v", got, stored)
	}
}

func TestJobLookupFailureIsStructuredExitCodeFour(t *testing.T) {
	cwd := t.TempDir()
	service := NewService(Options{})
	const missingJobID = "20260712T123456Z-deadbeef"
	for _, method := range []func() (result.Result, error){
		func() (result.Result, error) {
			return service.JobStatus(context.Background(), JobRequest{CWD: cwd, JobID: missingJobID})
		},
		func() (result.Result, error) {
			return service.JobResult(context.Background(), JobRequest{CWD: cwd, JobID: missingJobID})
		},
		func() (result.Result, error) {
			return service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: missingJobID})
		},
	} {
		got, err := method()
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != result.StatusFailed || got.Metadata.ExitCode != 4 || got.Metadata.JobID != missingJobID {
			t.Fatalf("missing-job result = %#v", got)
		}
	}
}

func TestJobServicesRejectInvalidOrCancelledRequestsBeforeStoreAccess(t *testing.T) {
	service := NewService(Options{
		WorkingDirectory: func() (string, error) {
			t.Fatal("working directory should not be requested for an empty job id")
			return "", nil
		},
	})
	for _, call := range []func(context.Context, JobRequest) (result.Result, error){
		service.JobStatus,
		service.JobResult,
		service.CancelJob,
	} {
		if _, err := call(context.Background(), JobRequest{CWD: t.TempDir(), JobID: "  "}); err == nil || !strings.Contains(err.Error(), "job id is required") {
			t.Fatalf("error = %v, want required job id", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.JobStatus(ctx, JobRequest{CWD: t.TempDir(), JobID: "job-1"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context error = %v", err)
	}
}

func TestJobServicesRejectMalformedIDsBeforeStoreAccess(t *testing.T) {
	service := NewService(Options{
		WorkingDirectory: func() (string, error) {
			t.Fatal("working directory should not be requested for a malformed job id")
			return "", nil
		},
	})
	invalidIDs := []string{
		"../escape",
		"..",
		".",
		"/absolute/path",
		"a/b",
		`a\b`,
		"not-a-job",
		"20260712T123456Z-deadbeeg",
		"20260712T12345Z-deadbeef",
		"20261340T123456Z-deadbeef",
		"20260712T123456Z-deadbeef\x00",
		"20260712T123456Z-😀😀😀😀",
	}
	for _, id := range invalidIDs {
		for _, call := range []func(context.Context, JobRequest) (result.Result, error){
			service.JobStatus,
			service.JobResult,
			service.CancelJob,
		} {
			if _, err := call(context.Background(), JobRequest{CWD: t.TempDir(), JobID: id}); !errors.Is(err, jobs.ErrInvalidID) {
				t.Errorf("job id %q error = %v, want ErrInvalidID", id, err)
			}
		}
	}
}

func TestJobServicesReportCorruptPersistedState(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Root, job.ID, "job.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	if _, err := service.JobStatus(context.Background(), JobRequest{CWD: cwd, JobID: job.ID}); err == nil {
		t.Fatal("expected corrupt job.json error")
	}

	job, err = store.Create(cwd, testJobSpec(), "another")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(job.ResultPath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID}); err == nil {
		t.Fatal("expected corrupt result.json error")
	}
}

func TestFinishJobPreservesCompetingTerminalResult(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	winner := result.Result{
		Status:       result.StatusCancelled,
		Summary:      "cancel won",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID},
	}
	if err := WriteJobResult(job.ResultPath, winner); err != nil {
		t.Fatal(err)
	}
	if err := FinishJob(store, job, result.Result{
		Status:       result.StatusSuccess,
		Summary:      "late completion",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID},
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != jobs.StatusCancelled {
		t.Fatalf("job status = %q, want cancelled", loaded.Status)
	}
	content, err := os.ReadFile(job.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	var got result.Result
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatal(err)
	}
	if got.Summary != winner.Summary || got.Status != winner.Status {
		t.Fatalf("stored result = %#v, want %#v", got, winner)
	}
}

func TestFinishJobKeepsPIDWhenResultWriteFails(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 999999999); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(job.ResultPath+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := FinishJob(store, job, result.Result{Status: result.StatusSuccess}); err == nil {
		t.Fatal("expected result write failure")
	}
	if pid, err := store.ReadPID(job.ID); err != nil || pid != 999999999 {
		t.Fatalf("pid after failed finish = %d, %v", pid, err)
	}
	got, err := NewService(Options{}).JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || got.Status != result.StatusFailed {
		t.Fatalf("abandoned job result = %#v, %v", got, err)
	}
}

func TestJobResultFailsAfterMissingPIDGrace(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	job.CreatedAt = time.Now().Add(-missingPIDGrace - time.Second)
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	req := JobRequest{CWD: cwd, JobID: job.ID}
	got, err := service.JobResult(context.Background(), req)
	if err != nil || got.Status != result.StatusFailed {
		t.Fatalf("missing worker result = %#v, %v", got, err)
	}
	loaded, err := store.Load(job.ID)
	if err != nil || loaded.Status != jobs.StatusFailed {
		t.Fatalf("persisted abandoned job = %#v, %v", loaded, err)
	}
	status, err := service.JobStatus(context.Background(), req)
	if err != nil || status.Status != result.StatusFailed {
		t.Fatalf("status after abandonment = %#v, %v", status, err)
	}
	cancelled, err := service.CancelJob(context.Background(), req)
	if err != nil || cancelled.Status != result.StatusFailed || cancelled.Summary != got.Summary {
		t.Fatalf("cancel after abandonment = %#v, %v", cancelled, err)
	}
}

func TestJobStatusUsesTerminalResultAfterInterruptedStateWrite(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteJobResult(job.ResultPath, result.Result{Status: result.StatusFailed, Summary: "worker stopped"}); err != nil {
		t.Fatal(err)
	}
	got, err := NewService(Options{}).JobStatus(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || got.Status != result.StatusFailed {
		t.Fatalf("status after interrupted state write = %#v, %v", got, err)
	}
}

func TestJobResultRechecksResultAfterCompletionLock(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	job.CreatedAt = time.Now().Add(-missingPIDGrace - time.Second)
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	want := result.Result{Status: result.StatusSuccess, Summary: "completed while result was checked"}
	done := make(chan struct {
		value result.Result
		err   error
	}, 1)
	if err := store.WithJobLock(job.ID, func() error {
		go func() {
			value, err := NewService(Options{}).JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
			done <- struct {
				value result.Result
				err   error
			}{value, err}
		}()
		select {
		case early := <-done:
			t.Fatalf("result returned before completion lock released: %#v, %v", early.value, early.err)
		case <-time.After(50 * time.Millisecond):
		}
		return WriteJobResult(job.ResultPath, want)
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got.err != nil || got.value.Status != want.Status || got.value.Summary != want.Summary {
			t.Fatalf("result after completion = %#v, %v", got.value, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("result remained blocked after completion")
	}
}

func testJobSpec() jobs.ExecSpec {
	return jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}
}
