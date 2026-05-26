package codex

import (
	"context"
	"reflect"
	"testing"
)

func TestExecWithRunnerBuildsArgv(t *testing.T) {
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
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
	if run.name == "" {
		t.Fatal("expected codex path")
	}
}

type recordingRunner struct {
	name   string
	args   []string
	cwd    string
	result Result
}

func (r *recordingRunner) Run(_ context.Context, name string, args []string, cwd string) (Result, error) {
	r.name = name
	r.args = args
	r.cwd = cwd
	return r.result, nil
}
