package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestAuthRecoveryUsesOneResumeWithinSharedBudget(t *testing.T) {
	for _, tc := range []struct {
		name     string
		budget   int
		outcomes []result.Status
		want     result.Status
	}{
		{"disabled", 0, []result.Status{result.StatusAuthRequired}, result.StatusAuthRequired},
		{"current login resumes", 1, []result.Status{result.StatusAuthRequired, result.StatusSuccess}, result.StatusSuccess},
		{"auth retries only once", 5, []result.Status{result.StatusAuthRequired, result.StatusAuthRequired}, result.StatusAuthRequired},
		{"capacity consumes shared budget", 1, []result.Status{result.StatusCapacity, result.StatusAuthRequired}, result.StatusAuthRequired},
		{"auth shares capacity budget", 2, []result.Status{result.StatusCapacity, result.StatusAuthRequired, result.StatusSuccess}, result.StatusSuccess},
		{"interrupted stops", 5, []result.Status{result.StatusInterrupted}, result.StatusInterrupted},
		{"policy refusal never retries", 5, []result.Status{result.StatusFailed}, result.StatusFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", FullAccess: true, Model: "gpt-6.1-sol", RetryCapacity: tc.budget, CapacityBackoff: time.Nanosecond}, "original task")
			if err != nil {
				t.Fatal(err)
			}
			prior := applicationService
			t.Cleanup(func() { applicationService = prior })
			calls := 0
			applicationService = app.NewService(app.Options{Executor: messagingExecutor(func(ctx context.Context, req input.Delegation) (executil.Result, error) {
				if calls >= len(tc.outcomes) {
					t.Fatal("unexpected extra attempt")
				}
				if calls > 0 {
					if req.Resume != "saved-session" || !req.FullAccess || req.Model != "gpt-6.1-sol" || strings.Contains(req.TaskText, "original task") {
						t.Fatalf("resume contract: %#v", req)
					}
					if tc.outcomes[calls-1] == result.StatusAuthRequired && !strings.Contains(req.TaskText, "current login") {
						t.Fatal(req.TaskText)
					}
				}
				status := tc.outcomes[calls]
				calls++
				res := executil.Result{TurnStarted: true, AgentSession: "saved-session", AuthRequired: status == result.StatusAuthRequired, Interrupted: status == result.StatusInterrupted, CapacityExhausted: status == result.StatusCapacity, Stdout: "partial work", FinalMessage: "partial work"}
				if status == result.StatusFailed {
					res.ExitCode = 1
					res.Failure = "policy_refusal"
					res.FailureMessage = "This content was flagged for possible cybersecurity risk."
				}
				return res, nil
			})})
			if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: job.ID}); err != nil {
				t.Fatal(err)
			}
			res, err := applicationService.JobResult(context.Background(), app.JobRequest{CWD: cwd, JobID: job.ID})
			if err != nil || calls != len(tc.outcomes) || res.Status != tc.want || res.Metadata.CapacityRetries != calls-1 || res.Metadata.AgentSession != "saved-session" {
				t.Fatalf("outcome: calls=%d %#v %v", calls, res, err)
			}
			if tc.want == result.StatusAuthRequired || tc.want == result.StatusInterrupted {
				if res.FinalMessage != nil || res.Metadata.ExitCode != 1 || !strings.Contains(res.Summary, "--resume saved-session") {
					t.Fatalf("recovery: %#v", res)
				}
			}
			if tc.want == result.StatusFailed && (res.Failure != "policy_refusal" || !strings.HasPrefix(res.Summary, "This content was flagged")) {
				t.Fatalf("refusal hidden: %#v", res)
			}
			status, err := applicationService.JobStatus(context.Background(), app.JobRequest{CWD: cwd, JobID: job.ID})
			if err != nil || status.Status != tc.want {
				t.Fatalf("persisted status %#v %v", status, err)
			}
		})
	}
}

func TestAuthRecoveryWithoutSessionDoesNotRetry(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", RetryCapacity: 3}, "task")
	if err != nil {
		t.Fatal(err)
	}
	prior := applicationService
	t.Cleanup(func() { applicationService = prior })
	calls := 0
	applicationService = app.NewService(app.Options{Executor: messagingExecutor(func(context.Context, input.Delegation) (executil.Result, error) {
		calls++
		return executil.Result{AuthRequired: true}, nil
	})})
	if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: job.ID}); err != nil {
		t.Fatal(err)
	}
	res, err := applicationService.JobResult(context.Background(), app.JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || calls != 1 || res.Status != result.StatusAuthRequired || strings.Contains(res.Summary, "--resume") {
		t.Fatalf("missing session: %#v %v calls=%d", res, err, calls)
	}
}
