package main

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

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

func TestResultStatusFromJob(t *testing.T) {
	cases := map[string]result.Status{
		"running":   result.StatusRunning,
		"complete":  result.StatusSuccess,
		"failed":    result.StatusFailed,
		"cancelled": result.StatusCancelled,
		"unknown":   result.StatusRunning,
	}
	for status, want := range cases {
		if got := resultStatusFromJob(status); got != want {
			t.Fatalf("resultStatusFromJob(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestIsTerminalJobStatus(t *testing.T) {
	for _, status := range []string{"complete", "failed", "cancelled"} {
		if !isTerminalJobStatus(status) {
			t.Fatalf("expected %q to be terminal", status)
		}
	}
	if isTerminalJobStatus("running") {
		t.Fatal("running should not be terminal")
	}
}

func TestJobLookupFailureResult(t *testing.T) {
	res := jobLookupFailureResult(input.Request{CWD: "/repo"}, "job-1", errors.New("missing"))
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

func TestAsyncJobRunArgsAreExact(t *testing.T) {
	got := asyncJobRunArgs("job-1", "/repo")
	want := []string{"--job-run", "job-1", "--cwd", "/repo"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("args = %#v, want %#v", got, want)
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
