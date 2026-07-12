package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestResultFromExecutionSuccess(t *testing.T) {
	res := resultFromExecution(input.Request{CWD: "/repo"}, executil.Result{ExitCode: 0, AgentSession: "session-1"}, nil)
	if res.Status != result.StatusSuccess {
		t.Fatalf("Status = %q", res.Status)
	}
	if res.Metadata.Access != "default" {
		t.Fatalf("Access = %q", res.Metadata.Access)
	}
	if res.Metadata.AgentSession != "session-1" {
		t.Fatalf("AgentSession = %q", res.Metadata.AgentSession)
	}
}

func TestResultFromExecutionNonZero(t *testing.T) {
	res := resultFromExecution(input.Request{CWD: "/repo"}, executil.Result{ExitCode: 2, Stderr: "bad"}, nil)
	if res.Status != result.StatusFailed {
		t.Fatalf("Status = %q", res.Status)
	}
	if res.Summary == "" || res.Details == "" {
		t.Fatalf("expected summary and details: %#v", res)
	}
}

func TestResultFromExecutionUsesResumeSessionFallback(t *testing.T) {
	res := resultFromExecution(input.Request{CWD: "/repo", Resume: "session-1"}, executil.Result{ExitCode: 1}, nil)
	if res.Metadata.AgentSession != "session-1" {
		t.Fatalf("AgentSession = %q", res.Metadata.AgentSession)
	}
}

func TestAttachExecutionLogWritesRawOutput(t *testing.T) {
	cwd := t.TempDir()
	logPath := filepath.Join(cwd, ".peeragent", "jobs", "job-1", "target.log")
	req := input.Request{CWD: cwd, Agent: "codex"}
	execResult := executil.Result{
		ExitCode:  0,
		Stdout:    "final message",
		RawStdout: `{"type":"item.completed","item":{"type":"agent_message","text":"thinking"}}`,
	}
	res := resultFromExecution(req, execResult, nil)

	attachExecutionLog(req, &res, execResult, logPath)

	if res.Metadata.LogPath != logPath {
		t.Fatalf("LogPath = %q, want %q", res.Metadata.LogPath, logPath)
	}
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(content)
	if !strings.Contains(got, execResult.RawStdout) {
		t.Fatalf("log missing raw stdout: %s", got)
	}
	if strings.Contains(got, "final message") {
		t.Fatalf("log used normalized stdout instead of raw stdout: %s", got)
	}
}

func TestResultFromExecutionError(t *testing.T) {
	res := resultFromExecution(input.Request{FullAccess: true, Profile: "p", Effort: "high", Model: "opus"}, executil.Result{ExitCode: 127}, errors.New("missing"))
	if res.Status != result.StatusFailed {
		t.Fatalf("Status = %q", res.Status)
	}
	if res.Metadata.Access != "full-access" {
		t.Fatalf("Access = %q", res.Metadata.Access)
	}
	if res.Metadata.Profile != "p" {
		t.Fatalf("Profile = %q", res.Metadata.Profile)
	}
	if res.Metadata.Effort != "high" {
		t.Fatalf("Effort = %q", res.Metadata.Effort)
	}
	if res.Metadata.Model != "opus" {
		t.Fatalf("Model = %q", res.Metadata.Model)
	}
}

func TestResultFromExecutionGemini(t *testing.T) {
	res := resultFromExecution(input.Request{CWD: "/repo", Agent: "gemini"}, executil.Result{ExitCode: 0}, nil)
	if res.Metadata.Agent != "gemini" {
		t.Fatalf("Agent = %q", res.Metadata.Agent)
	}
	if !strings.Contains(res.Summary, "Gemini") {
		t.Fatalf("Summary = %q", res.Summary)
	}
}

func TestResultFromExecutionZAI(t *testing.T) {
	res := resultFromExecution(input.Request{CWD: "/repo", Agent: "zai", Effort: "xhigh", Model: "glm-5.2"}, executil.Result{ExitCode: 0}, nil)
	if res.Metadata.Agent != "zai" {
		t.Fatalf("Agent = %q", res.Metadata.Agent)
	}
	if res.Metadata.Model != "glm-5.2" {
		t.Fatalf("Model = %q", res.Metadata.Model)
	}
	if res.Metadata.Effort != "xhigh" {
		t.Fatalf("Effort = %q", res.Metadata.Effort)
	}
	if !strings.Contains(res.Summary, "GLM 5.2") {
		t.Fatalf("Summary = %q", res.Summary)
	}
}

func TestResultStatusFromJob(t *testing.T) {
	cases := map[string]result.Status{
		"running":   result.StatusRunning,
		"complete":  result.StatusSuccess,
		"failed":    result.StatusFailed,
		"cancelled": result.StatusCancelled,
		"unknown":   result.StatusRunning,
	}
	for status, want := range cases {
		if got := app.ResultStatusFromJob(status); got != want {
			t.Fatalf("ResultStatusFromJob(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestIsTerminalJobStatus(t *testing.T) {
	for _, status := range []string{"complete", "failed", "cancelled"} {
		if !app.IsTerminalJobStatus(status) {
			t.Fatalf("expected %q to be terminal", status)
		}
	}
	if app.IsTerminalJobStatus("running") {
		t.Fatal("running should not be terminal")
	}
}

func TestJobLookupFailureResult(t *testing.T) {
	res := app.JobLookupFailureResult(app.JobRequest{CWD: "/repo", JobID: "job-1"}, errors.New("missing"))
	if res.Status != result.StatusFailed {
		t.Fatalf("Status = %q", res.Status)
	}
	if res.Metadata.ExitCode != 4 {
		t.Fatalf("ExitCode = %d", res.Metadata.ExitCode)
	}
	if res.Metadata.JobID != "job-1" {
		t.Fatalf("JobID = %q", res.Metadata.JobID)
	}
	if !strings.Contains(res.Summary, "missing") {
		t.Fatalf("Summary = %q", res.Summary)
	}
}

func TestRequestFromJobReconstructsRequest(t *testing.T) {
	job := jobs.Job{
		ID:  "job-1",
		CWD: "/repo",
		Spec: jobs.ExecSpec{
			Agent:   "claude",
			Access:  "worktree",
			Profile: "profile-a",
			Effort:  "xhigh",
			Model:   "opus",
			Resume:  "session-1",
			JSON:    true,
		},
	}
	req := requestFromJob(job, "do work")
	if req.TaskText != "do work" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
	if req.CWD != "/repo" {
		t.Fatalf("CWD = %q", req.CWD)
	}
	if req.Agent != "claude" || req.Profile != "profile-a" || req.Effort != "xhigh" || req.Model != "opus" || req.Resume != "session-1" {
		t.Fatalf("request = %#v", req)
	}
	if !req.JSON {
		t.Fatal("JSON = false")
	}
	if !req.Worktree {
		t.Fatal("Worktree = false")
	}
	if req.FullAccess {
		t.Fatal("FullAccess = true")
	}
}

func TestRequestFromJobPreservesCompatibilityBooleans(t *testing.T) {
	job := jobs.Job{
		CWD: "/repo",
		Spec: jobs.ExecSpec{
			Agent:      "codex",
			Access:     "default",
			FullAccess: true,
			Worktree:   true,
		},
	}
	req := requestFromJob(job, "task")
	if !req.FullAccess || !req.Worktree {
		t.Fatalf("request = %#v", req)
	}
}

func TestRequestFromJobUsesCanonicalAccess(t *testing.T) {
	job := jobs.Job{
		CWD: "/repo",
		Spec: jobs.ExecSpec{
			Agent:    "codex",
			Access:   "full-access",
			Worktree: true,
		},
	}
	req := requestFromJob(job, "task")
	if !req.FullAccess || req.Worktree {
		t.Fatalf("request = %#v", req)
	}
}

func TestRunAsyncJobWorktreeFailureUsesPersistedJobRequest(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{
		Agent:    "claude",
		Access:   "worktree",
		Effort:   "xhigh",
		Model:    "opus",
		Resume:   "session-1",
		JSON:     true,
		Worktree: true,
	}, "do work")
	if err != nil {
		t.Fatal(err)
	}

	if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: job.ID}); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "failed" {
		t.Fatalf("Status = %q", loaded.Status)
	}
	content, err := os.ReadFile(job.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	var res result.Result
	if err := json.Unmarshal(content, &res); err != nil {
		t.Fatal(err)
	}
	if res.Status != result.StatusFailed {
		t.Fatalf("result status = %q", res.Status)
	}
	if res.Metadata.Agent != "claude" || res.Metadata.Access != "worktree" || res.Metadata.AgentSession != "session-1" {
		t.Fatalf("metadata = %#v", res.Metadata)
	}
}

func TestFinishAsyncJobDoesNotOverwriteCancelledJobOrResult(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	cancelled := result.Result{
		Status:       result.StatusCancelled,
		Summary:      "cancel won",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID},
	}
	if err := app.WriteJobResult(job.ResultPath, cancelled); err != nil {
		t.Fatal(err)
	}
	job.Status = "cancelled"
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}

	err = finishAsyncJob(store, job, result.Result{
		Status:       result.StatusSuccess,
		Summary:      "natural finish",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "cancelled" {
		t.Fatalf("Status = %q, want cancelled", loaded.Status)
	}
	got := readStoredResult(t, job.ResultPath)
	if got.Status != result.StatusCancelled || got.Summary != "cancel won" {
		t.Fatalf("result = %#v, want cancelled result untouched", got)
	}
}

func TestFinishAsyncJobDoesNotOverwriteCancelledResult(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 99999); err != nil {
		t.Fatal(err)
	}
	if err := app.WriteJobResult(job.ResultPath, result.Result{
		Status:       result.StatusCancelled,
		Summary:      "cancel result",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID},
	}); err != nil {
		t.Fatal(err)
	}

	if err := finishAsyncJob(store, job, result.Result{
		Status:       result.StatusFailed,
		Summary:      "late failure",
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
	if loaded.Status != "cancelled" {
		t.Fatalf("Status = %q, want cancelled", loaded.Status)
	}
	got := readStoredResult(t, job.ResultPath)
	if got.Status != result.StatusCancelled || got.Summary != "cancel result" {
		t.Fatalf("result = %#v, want cancelled result untouched", got)
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID err = %v, want removed pid", err)
	}
}

func TestFinishAsyncJobRemovesPIDAfterNaturalFinish(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 99999); err != nil {
		t.Fatal(err)
	}

	if err := finishAsyncJob(store, job, result.Result{
		Status:       result.StatusSuccess,
		Summary:      "done",
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
	if loaded.Status != "complete" {
		t.Fatalf("Status = %q, want complete", loaded.Status)
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID err = %v, want removed pid", err)
	}
}

func TestCancelJobWithoutPIDSidecarWritesTerminalState(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}

	if err := captureStdout(func() error {
		return cancelJob(input.Request{CWD: cwd, CancelJobID: job.ID, JSON: true})
	}); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "cancelled" {
		t.Fatalf("Status = %q, want cancelled", loaded.Status)
	}
	got := readStoredResult(t, job.ResultPath)
	if got.Status != result.StatusCancelled {
		t.Fatalf("result status = %q, want cancelled", got.Status)
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID err = %v, want missing pid", err)
	}
}

func TestCancelJobRepairsExistingTerminalResultWithoutOverwrite(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 99999); err != nil {
		t.Fatal(err)
	}
	if err := app.WriteJobResult(job.ResultPath, result.Result{
		Status:       result.StatusSuccess,
		Summary:      "finish won",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID},
	}); err != nil {
		t.Fatal(err)
	}

	if err := captureStdout(func() error {
		return cancelJob(input.Request{CWD: cwd, CancelJobID: job.ID, JSON: true})
	}); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "complete" {
		t.Fatalf("Status = %q, want complete", loaded.Status)
	}
	got := readStoredResult(t, job.ResultPath)
	if got.Status != result.StatusSuccess || got.Summary != "finish won" {
		t.Fatalf("result = %#v, want success result untouched", got)
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID err = %v, want removed pid", err)
	}
}

func readStoredResult(t *testing.T, path string) result.Result {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var res result.Result
	if err := json.Unmarshal(content, &res); err != nil {
		t.Fatal(err)
	}
	return res
}

func captureStdout(fn func() error) error {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	os.Stdout = w
	defer func() {
		os.Stdout = orig
	}()
	err = fn()
	closeErr := w.Close()
	_, _ = io.ReadAll(r)
	readCloseErr := r.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return readCloseErr
}
