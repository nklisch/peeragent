package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestDelegateSuccess(t *testing.T) {
	executor := &fakeExecutor{result: executil.Result{ExitCode: 0, Stdout: "done", AgentSession: "session-1"}}
	service := NewService(Options{Executor: executor})
	delegation := normalizedDelegation(t, input.Delegation{TaskText: "do work", CWD: "/repo"})

	got, err := service.Delegate(context.Background(), delegation)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != result.StatusSuccess || got.Metadata.AgentSession != "session-1" {
		t.Fatalf("result = %#v", got)
	}
	if !reflect.DeepEqual(executor.got, delegation) {
		t.Fatalf("executor request = %#v, want %#v", executor.got, delegation)
	}
}

func TestDelegateTargetFailureIsStructuredResult(t *testing.T) {
	executor := &fakeExecutor{result: executil.Result{ExitCode: 23, Stderr: "target failed"}}
	service := NewService(Options{Executor: executor})
	delegation := normalizedDelegation(t, input.Delegation{TaskText: "do work", CWD: "/repo"})

	got, err := service.Delegate(context.Background(), delegation)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != result.StatusFailed || got.Metadata.ExitCode != 23 {
		t.Fatalf("result = %#v", got)
	}
}

func TestDelegateInfrastructureFailureReturnsStructuredFailureAndError(t *testing.T) {
	expected := errors.New("runner unavailable")
	executor := &fakeExecutor{err: expected}
	service := NewService(Options{Executor: executor})
	delegation := normalizedDelegation(t, input.Delegation{TaskText: "do work", CWD: "/repo"})

	got, err := service.Delegate(context.Background(), delegation)
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want %v", err, expected)
	}
	if got.Status != result.StatusFailed || got.Summary != expected.Error() {
		t.Fatalf("result = %#v", got)
	}
}

func TestDelegateRejectsCancelledContextBeforeExecution(t *testing.T) {
	executor := &fakeExecutor{}
	service := NewService(Options{Executor: executor})
	delegation := normalizedDelegation(t, input.Delegation{TaskText: "do work", CWD: "/repo"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := service.Delegate(ctx, delegation)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if got.Status != result.StatusFailed || executor.called {
		t.Fatalf("result = %#v, executor called = %v", got, executor.called)
	}
}

func TestLaunchUsesInjectedLauncherAndPersistsRequest(t *testing.T) {
	launcher := &fakeLauncher{}
	service := NewService(Options{
		Launcher: launcher,
		Executable: func() (string, error) {
			return "/bin/peeragent", nil
		},
	})
	delegation := normalizedDelegation(t, input.Delegation{
		TaskText:   "do work",
		CWD:        t.TempDir(),
		Agent:      "claude",
		FullAccess: true,
		Profile:    "profile-a",
		Effort:     "xhigh",
		Model:      "fable",
		Resume:     "session-1",
	})

	got, err := service.Launch(context.Background(), delegation)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != result.StatusRunning || got.Metadata.JobID == "" {
		t.Fatalf("result = %#v", got)
	}
	if launcher.job.ID != got.Metadata.JobID {
		t.Fatalf("job = %#v, result = %#v", launcher.job, got)
	}
	if launcher.job.Spec.Agent != "claude" || launcher.job.Spec.Access != "full-access" || !launcher.job.Spec.FullAccess {
		t.Fatalf("spec = %#v", launcher.job.Spec)
	}
	if launcher.job.Spec.JSON != true {
		t.Fatal("async job must persist machine-readable result mode")
	}
}

func TestLaunchInfrastructureFailureReturnsErrorAndFailedResult(t *testing.T) {
	expected := errors.New("cannot start child")
	launcher := &fakeLauncher{err: expected}
	service := NewService(Options{
		Launcher: launcher,
		Executable: func() (string, error) {
			return "/bin/peeragent", nil
		},
	})
	delegation := normalizedDelegation(t, input.Delegation{TaskText: "do work", CWD: t.TempDir()})

	got, err := service.Launch(context.Background(), delegation)
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want %v", err, expected)
	}
	if got.Status != result.StatusFailed || got.Metadata.JobID == "" {
		t.Fatalf("result = %#v", got)
	}
}

func TestLaunchDoesNotCreateJobWhenExecutableResolutionFails(t *testing.T) {
	expected := errors.New("executable unavailable")
	service := NewService(Options{
		Executable: func() (string, error) { return "", expected },
	})
	delegation := normalizedDelegation(t, input.Delegation{TaskText: "do work", CWD: t.TempDir()})

	got, err := service.Launch(context.Background(), delegation)
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want %v", err, expected)
	}
	if got.Status != result.StatusFailed || got.Metadata.JobID == "" {
		t.Fatalf("result = %#v", got)
	}
}

type fakeExecutor struct {
	result executil.Result
	err    error
	got    input.Delegation
	called bool
}

func (f *fakeExecutor) Execute(_ context.Context, delegation input.Delegation) (executil.Result, error) {
	f.called = true
	f.got = delegation
	return f.result, f.err
}

type fakeLauncher struct {
	job jobs.Job
	err error
}

func (f *fakeLauncher) Launch(_ string, job jobs.Job) error {
	f.job = job
	return f.err
}

func normalizedDelegation(t *testing.T, raw input.Delegation) input.Delegation {
	t.Helper()
	got, err := input.NormalizeDelegation(raw, func() (string, error) { return "/repo", nil })
	if err != nil {
		t.Fatal(err)
	}
	return got
}
