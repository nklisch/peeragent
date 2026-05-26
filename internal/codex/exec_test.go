package codex

import (
	"context"
	"reflect"
	"testing"
)

func TestExecWithRunnerBuildsArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0, Stdout: "ok"}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("Stdout = %q", result.Stdout)
	}
	if run.cwd != "/repo" {
		t.Fatalf("cwd = %q", run.cwd)
	}
	wantArgs := []string{
		"exec",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"--ask-for-approval", "on-request",
		"-c", "approvals_reviewer=auto_review",
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
	if run.name == "" {
		t.Fatal("expected codex path")
	}
}

func TestExecWithRunnerBuildsFullAccessArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", FullAccess: true})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--cd", "/repo",
		"--dangerously-bypass-approvals-and-sandbox",
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerBuildsProfileArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Profile: "alt-subagent"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"--ask-for-approval", "on-request",
		"-c", "approvals_reviewer=auto_review",
		"--profile", "alt-subagent",
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerBuildsHighEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Effort: "high"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"--ask-for-approval", "on-request",
		"-c", "approvals_reviewer=auto_review",
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerBuildsXHighEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Effort: "xhigh"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"--ask-for-approval", "on-request",
		"-c", "approvals_reviewer=auto_review",
		"-c", `model_reasoning_effort="xhigh"`,
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

type recordingRunner struct {
	name   string
	args   []string
	cwd    string
	result Result
}

func stubLookPath(t *testing.T) {
	t.Helper()
	previous := lookPath
	lookPath = func(name string) (string, error) {
		return "/test/bin/" + name, nil
	}
	t.Cleanup(func() {
		lookPath = previous
	})
}

func (r *recordingRunner) Run(_ context.Context, name string, args []string, cwd string) (Result, error) {
	r.name = name
	r.args = args
	r.cwd = cwd
	return r.result, nil
}
