package zai

import (
	"context"
	"reflect"
	"testing"
)

func TestExecWithRunnerBuildsDefaultArgv(t *testing.T) {
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
		"--provider", "zai",
		"--model", "glm-5.2",
		"--thinking", "high",
		"--no-session",
		"-p",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
	if run.name == "" {
		t.Fatal("expected pi path")
	}
}

func TestExecWithRunnerBuildsEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Effort: "xhigh"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--provider", "zai",
		"--model", "glm-5.2",
		"--thinking", "xhigh",
		"--no-session",
		"-p",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerBuildsResumeArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "continue work", Resume: "session-1"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--provider", "zai",
		"--model", "glm-5.2",
		"--thinking", "high",
		"--session", "session-1",
		"-p",
		"continue work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
	if result.AgentSession != "session-1" {
		t.Fatalf("AgentSession = %q", result.AgentSession)
	}
}

func TestExecWithRunnerAcceptsExplicitFixedModel(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Model: "glm-5.2"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--provider", "zai",
		"--model", "glm-5.2",
		"--thinking", "high",
		"--no-session",
		"-p",
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
