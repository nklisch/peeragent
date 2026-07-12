package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

const validJobID = "20260712T123456Z-deadbeef"

func TestJobToolsReturnStructuredResultsAndNormalizeRequests(t *testing.T) {
	service := &fakeService{
		jobStatusResult: result.Result{Status: result.StatusRunning, Summary: "running", ChangedFiles: []string{}, Verification: []result.Verification{}},
		jobResultResult: result.Result{Status: result.StatusSuccess, Summary: "done", ChangedFiles: []string{}, Verification: []result.Verification{}},
		jobCancelResult: result.Result{Status: result.StatusCancelled, Summary: "cancelled", ChangedFiles: []string{}, Verification: []result.Verification{}},
	}
	session := connectTestClient(t, service)

	for _, test := range []struct {
		name string
		want result.Status
	}{
		{name: "job_status", want: result.StatusRunning},
		{name: "job_result", want: result.StatusSuccess},
		{name: "job_cancel", want: result.StatusCancelled},
	} {
		t.Run(test.name, func(t *testing.T) {
			called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
				Name:      test.name,
				Arguments: map[string]any{"job_id": " " + validJobID + " ", "cwd": " /repo "},
			})
			if err != nil {
				t.Fatal(err)
			}
			if called.IsError {
				t.Fatalf("unexpected tool error: %#v", called)
			}
			got := decodeResult(t, called.StructuredContent)
			if got.Status != test.want {
				t.Fatalf("result = %#v, want status %q", got, test.want)
			}
		})
	}
	if service.lastJobRequest != (app.JobRequest{JobID: validJobID, CWD: "/repo"}) {
		t.Fatalf("job request = %#v, want normalized request", service.lastJobRequest)
	}
	if service.jobStatusCalls != 1 || service.jobResultCalls != 1 || service.jobCancelCalls != 1 {
		t.Fatalf("calls = status:%d result:%d cancel:%d", service.jobStatusCalls, service.jobResultCalls, service.jobCancelCalls)
	}
}

func TestJobToolsRejectEmptyIDBeforeApplicationService(t *testing.T) {
	service := &fakeService{}
	session := connectTestClient(t, service)
	for _, name := range []string{"job_status", "job_result", "job_cancel"} {
		called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
			Name:      name,
			Arguments: map[string]any{"job_id": "  "},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !called.IsError {
			t.Fatalf("%s accepted empty id: %#v", name, called)
		}
	}
	if service.jobStatusCalls != 0 || service.jobResultCalls != 0 || service.jobCancelCalls != 0 {
		t.Fatalf("application service called for invalid ids: %#v", service)
	}
}

func TestMCPJobToolsRejectMalformedIDsThroughApplicationBoundary(t *testing.T) {
	cwd := filepath.Join(t.TempDir(), "repository")
	service := app.NewService(app.Options{WorkingDirectory: func() (string, error) { return cwd, nil }})
	session := connectTestClient(t, service)
	invalidIDs := []string{
		"../escape",
		"..",
		".",
		"/absolute/path",
		`a\b`,
		"a/b",
		"not-a-job",
		"20260712T123456Z-deadbeeg",
		"20260712T12345Z-deadbeef",
		"20261340T123456Z-deadbeef",
		"20260712T123456Z-deadbeef\x00",
		"20260712T123456Z-😀😀😀😀",
	}
	for _, name := range []string{"job_status", "job_result", "job_cancel"} {
		for _, id := range invalidIDs {
			called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
				Name:      name,
				Arguments: map[string]any{"job_id": id, "cwd": cwd},
			})
			if err != nil {
				t.Fatal(err)
			}
			if !called.IsError {
				t.Fatalf("%s accepted malformed id %q: %#v", name, id, called)
			}
		}
	}

	for _, name := range []string{"job_status", "job_result", "job_cancel"} {
		called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
			Name:      name,
			Arguments: map[string]any{"job_id": validJobID, "cwd": cwd},
		})
		if err != nil {
			t.Fatal(err)
		}
		if called.IsError {
			t.Fatalf("%s rejected syntactically valid missing id: %#v", name, called)
		}
		got := decodeResult(t, called.StructuredContent)
		if got.Metadata.ExitCode != 4 || got.Metadata.JobID != validJobID || got.Status != result.StatusFailed {
			t.Fatalf("%s missing result = %#v, want structured exit 4", name, got)
		}
	}
}

func TestMCPJobToolsUseApplicationCWDNormalizer(t *testing.T) {
	defaultCWD := t.TempDir()
	store := jobs.NewStore(defaultCWD)
	job, err := store.Create(defaultCWD, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewService(app.Options{WorkingDirectory: func() (string, error) { return defaultCWD, nil }})
	session := connectTestClient(t, service)

	called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "job_status",
		Arguments: map[string]any{"job_id": job.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called.IsError || decodeResult(t, called.StructuredContent).Metadata.CWD != defaultCWD {
		t.Fatalf("default cwd result = %#v", called)
	}

	explicitCWD := t.TempDir()
	explicitStore := jobs.NewStore(explicitCWD)
	explicitJob, err := explicitStore.Create(explicitCWD, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	called, err = session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "job_status",
		Arguments: map[string]any{"job_id": explicitJob.ID, "cwd": explicitCWD},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called.IsError || decodeResult(t, called.StructuredContent).Metadata.CWD != explicitCWD {
		t.Fatalf("explicit cwd result = %#v", called)
	}
}

func TestMCPJobToolsConcurrentCallsPreserveTerminalState(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewService(app.Options{})
	session := connectTestClient(t, service)

	names := []string{"job_status", "job_result", "job_status", "job_result", "job_cancel", "job_status", "job_cancel", "job_result"}
	results := make(chan *sdkmcp.CallToolResult, len(names))
	errs := make(chan error, len(names))
	var wg sync.WaitGroup
	for _, name := range names {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
				Name:      name,
				Arguments: map[string]any{"job_id": job.ID, "cwd": cwd},
			})
			if err != nil {
				errs <- err
				return
			}
			results <- called
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	for called := range results {
		if called.IsError {
			t.Fatalf("concurrent tool error: %#v", called)
		}
		got := decodeResult(t, called.StructuredContent)
		if got.Status != result.StatusRunning && got.Status != result.StatusCancelled {
			t.Fatalf("unexpected concurrent status: %#v", got)
		}
	}

	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "cancelled" {
		t.Fatalf("job status = %q, want one cancellation winner", loaded.Status)
	}
	content, err := os.ReadFile(job.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	var stored result.Result
	if err := json.Unmarshal(content, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Status != result.StatusCancelled || stored.Metadata.JobID != job.ID {
		t.Fatalf("stored result = %#v, want cancelled result for %s", stored, job.ID)
	}
}
