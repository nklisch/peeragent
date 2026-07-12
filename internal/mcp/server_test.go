package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/result"
)

func TestServerInitializesListsGeneratedDelegateSchemaAndInstructions(t *testing.T) {
	service := &fakeService{}
	session := connectTestClient(t, service)

	initialized := session.InitializeResult()
	if initialized == nil || initialized.Instructions != serverInstructions {
		t.Fatalf("initialize result = %#v", initialized)
	}
	if initialized.ServerInfo == nil || initialized.ServerInfo.Name != "peeragent" {
		t.Fatalf("server info = %#v", initialized.ServerInfo)
	}

	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 4 {
		t.Fatalf("tools = %#v", listed.Tools)
	}
	byName := make(map[string]*sdkmcp.Tool, len(listed.Tools))
	for _, tool := range listed.Tools {
		byName[tool.Name] = tool
	}
	for _, name := range []string{"delegate", "job_status", "job_result", "job_cancel"} {
		if byName[name] == nil {
			t.Fatalf("tool discovery missing %q: %#v", name, listed.Tools)
		}
	}
	if byName["job_status"].Annotations == nil || !byName["job_status"].Annotations.ReadOnlyHint || byName["job_status"].Annotations.DestructiveHint == nil || *byName["job_status"].Annotations.DestructiveHint {
		t.Fatalf("job_status annotations = %#v", byName["job_status"].Annotations)
	}
	if byName["job_result"].Annotations == nil || !byName["job_result"].Annotations.ReadOnlyHint || byName["job_result"].Annotations.DestructiveHint == nil || *byName["job_result"].Annotations.DestructiveHint {
		t.Fatalf("job_result annotations = %#v", byName["job_result"].Annotations)
	}
	cancelAnnotations := byName["job_cancel"].Annotations
	if cancelAnnotations == nil || cancelAnnotations.ReadOnlyHint || cancelAnnotations.DestructiveHint == nil || !*cancelAnnotations.DestructiveHint || !cancelAnnotations.IdempotentHint {
		t.Fatalf("job_cancel annotations = %#v", cancelAnnotations)
	}

	schema := decodeObject(t, byName["delegate"].InputSchema)
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("input schema properties = %#v", schema["properties"])
	}
	for _, field := range []string{"task", "agent", "cwd", "profile", "effort", "model", "resume", "full_access", "async"} {
		if _, ok := properties[field]; !ok {
			t.Fatalf("generated schema missing %q: %#v", field, properties)
		}
	}
	if _, ok := properties["worktree"]; ok {
		t.Fatal("worktree must not be exposed by MCP delegation")
	}
	for _, name := range []string{"job_status", "job_result", "job_cancel"} {
		jobSchema := decodeObject(t, byName[name].InputSchema)
		jobProperties, ok := jobSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s schema properties = %#v", name, jobSchema["properties"])
		}
		if _, ok := jobProperties["job_id"]; !ok {
			t.Fatalf("%s schema missing job_id: %#v", name, jobProperties)
		}
		if _, ok := jobProperties["cwd"]; !ok {
			t.Fatalf("%s schema missing cwd: %#v", name, jobProperties)
		}
	}
}

func TestDelegateBlockingReturnsStructuredPeeragentResult(t *testing.T) {
	service := &fakeService{
		delegateResult: result.Result{
			Status:       result.StatusSuccess,
			Summary:      "delegated",
			ChangedFiles: []string{"main.go"},
			Verification: []result.Verification{{Command: "go test ./...", Status: "passed"}},
			Metadata:     result.Metadata{CWD: "/repo", Agent: "claude", ExitCode: 0},
		},
	}
	session := connectTestClient(t, service)

	called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "delegate",
		Arguments: map[string]any{
			"task":  "do work",
			"agent": "claude",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called.IsError {
		t.Fatalf("unexpected tool error: %#v", called)
	}
	got := decodeResult(t, called.StructuredContent)
	if !reflect.DeepEqual(got, service.delegateResult) {
		t.Fatalf("structured result = %#v, want %#v", got, service.delegateResult)
	}
	if service.delegateCalls != 1 || service.launchCalls != 0 {
		t.Fatalf("calls: delegate=%d launch=%d", service.delegateCalls, service.launchCalls)
	}
	if service.lastDelegation.Agent != "claude" || service.lastDelegation.CWD != "/repo" {
		t.Fatalf("normalized delegation = %#v", service.lastDelegation)
	}
}

func TestDelegateAsyncReturnsRunningResult(t *testing.T) {
	service := &fakeService{
		launchResult: result.Result{
			Status:       result.StatusRunning,
			Summary:      "job started",
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata:     result.Metadata{CWD: "/repo", Agent: "codex", JobID: "job-1"},
		},
	}
	session := connectTestClient(t, service)

	called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name: "delegate",
		Arguments: map[string]any{
			"task":  "long work",
			"async": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called.IsError {
		t.Fatalf("unexpected tool error: %#v", called)
	}
	got := decodeResult(t, called.StructuredContent)
	if got.Status != result.StatusRunning || got.Metadata.JobID != "job-1" {
		t.Fatalf("structured result = %#v", got)
	}
	if service.delegateCalls != 0 || service.launchCalls != 1 {
		t.Fatalf("calls: delegate=%d launch=%d", service.delegateCalls, service.launchCalls)
	}
}

func TestDelegateRejectsInvalidInputBeforeCallingService(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "empty task", args: map[string]any{"task": "  "}},
		{name: "unknown agent", args: map[string]any{"task": "do work", "agent": "llama"}},
		{name: "invalid claude effort", args: map[string]any{"task": "do work", "agent": "claude", "effort": "medium"}},
		{name: "invalid zai model", args: map[string]any{"task": "do work", "agent": "zai", "model": "glm-5.1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeService{}
			session := connectTestClient(t, service)
			called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "delegate", Arguments: tt.args})
			if err != nil {
				t.Fatal(err)
			}
			if !called.IsError {
				t.Fatalf("expected tool error: %#v", called)
			}
			if service.delegateCalls != 0 || service.launchCalls != 0 {
				t.Fatalf("service called for invalid input: delegate=%d launch=%d", service.delegateCalls, service.launchCalls)
			}
		})
	}
}

func TestDelegateSchemaRejectsWrongTaskTypeBeforeCallingService(t *testing.T) {
	service := &fakeService{}
	session := connectTestClient(t, service)

	called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "delegate",
		Arguments: map[string]any{"task": 42},
	})
	if err == nil && (called == nil || !called.IsError) {
		t.Fatalf("expected schema rejection, result=%#v", called)
	}
	if service.delegateCalls != 0 || service.launchCalls != 0 {
		t.Fatalf("service called for schema-invalid input: delegate=%d launch=%d", service.delegateCalls, service.launchCalls)
	}
}

func TestDelegateCancellationReachesBlockingService(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	service := &fakeService{}
	service.delegateFn = func(ctx context.Context, _ input.Delegation) (result.Result, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return result.Result{Status: result.StatusFailed, Summary: "cancelled", ChangedFiles: []string{}, Verification: []result.Verification{}}, ctx.Err()
	}
	session := connectTestClient(t, service)

	ctx, cancel := context.WithCancel(context.Background())
	callDone := make(chan error, 1)
	go func() {
		_, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "delegate",
			Arguments: map[string]any{"task": "wait for cancellation"},
		})
		callDone <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("blocking service was not called")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("MCP cancellation did not reach service")
	}
	select {
	case <-callDone:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled MCP call did not return")
	}
}

func TestDelegatePropagatesInfrastructureErrorsAsToolErrors(t *testing.T) {
	service := &fakeService{delegateErr: errors.New("runner unavailable")}
	session := connectTestClient(t, service)

	called, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "delegate",
		Arguments: map[string]any{"task": "do work"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called.IsError {
		t.Fatalf("expected infrastructure tool error: %#v", called)
	}
	if service.delegateCalls != 1 {
		t.Fatalf("delegate calls = %d", service.delegateCalls)
	}
}

func connectTestClient(t *testing.T, service ServerService) *sdkmcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	server := NewServer(service, func() (string, error) { return "/repo", nil })
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Run(context.Background(), serverTransport) }()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
			t.Error("MCP server did not stop")
		}
	})
	return session
}

func decodeObject(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func decodeResult(t *testing.T, value any) result.Result {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded result.Result
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

type fakeService struct {
	delegateResult  result.Result
	launchResult    result.Result
	jobStatusResult result.Result
	jobResultResult result.Result
	jobCancelResult result.Result
	delegateErr     error
	launchErr       error
	jobStatusErr    error
	jobResultErr    error
	jobCancelErr    error
	delegateFn      func(context.Context, input.Delegation) (result.Result, error)
	launchFn        func(context.Context, input.Delegation) (result.Result, error)
	jobStatusFn     func(context.Context, app.JobRequest) (result.Result, error)
	jobResultFn     func(context.Context, app.JobRequest) (result.Result, error)
	jobCancelFn     func(context.Context, app.JobRequest) (result.Result, error)

	mu             sync.Mutex
	delegateCalls  int
	launchCalls    int
	jobStatusCalls int
	jobResultCalls int
	jobCancelCalls int
	lastDelegation input.Delegation
	lastJobRequest app.JobRequest
}

func (f *fakeService) Delegate(ctx context.Context, delegation input.Delegation) (result.Result, error) {
	f.mu.Lock()
	f.delegateCalls++
	f.lastDelegation = delegation
	f.mu.Unlock()
	if f.delegateFn != nil {
		return f.delegateFn(ctx, delegation)
	}
	return f.delegateResult, f.delegateErr
}

func (f *fakeService) Launch(ctx context.Context, delegation input.Delegation) (result.Result, error) {
	f.mu.Lock()
	f.launchCalls++
	f.lastDelegation = delegation
	f.mu.Unlock()
	if f.launchFn != nil {
		return f.launchFn(ctx, delegation)
	}
	return f.launchResult, f.launchErr
}

func (f *fakeService) JobStatus(ctx context.Context, request app.JobRequest) (result.Result, error) {
	f.mu.Lock()
	f.jobStatusCalls++
	f.lastJobRequest = request
	f.mu.Unlock()
	if f.jobStatusFn != nil {
		return f.jobStatusFn(ctx, request)
	}
	return f.jobStatusResult, f.jobStatusErr
}

func (f *fakeService) JobResult(ctx context.Context, request app.JobRequest) (result.Result, error) {
	f.mu.Lock()
	f.jobResultCalls++
	f.lastJobRequest = request
	f.mu.Unlock()
	if f.jobResultFn != nil {
		return f.jobResultFn(ctx, request)
	}
	return f.jobResultResult, f.jobResultErr
}

func (f *fakeService) CancelJob(ctx context.Context, request app.JobRequest) (result.Result, error) {
	f.mu.Lock()
	f.jobCancelCalls++
	f.lastJobRequest = request
	f.mu.Unlock()
	if f.jobCancelFn != nil {
		return f.jobCancelFn(ctx, request)
	}
	return f.jobCancelResult, f.jobCancelErr
}
