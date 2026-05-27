package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/result"
)

func TestResultFromExecutionSuccess(t *testing.T) {
	res := resultFromExecution(input.Request{CWD: "/repo"}, executil.Result{ExitCode: 0}, nil)
	if res.Status != result.StatusSuccess {
		t.Fatalf("Status = %q", res.Status)
	}
	if res.Metadata.Access != "default" {
		t.Fatalf("Access = %q", res.Metadata.Access)
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
