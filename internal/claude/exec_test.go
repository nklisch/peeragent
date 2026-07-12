package claude

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

	wantArgs := []string{
		"--print",
		"--output-format", "json",
		"--add-dir", "/repo",
		"--permission-mode", "auto",
		"--effort", "xhigh",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
	if run.name == "" {
		t.Fatal("expected claude path")
	}
}

func TestExecWithRunnerBuildsFullAccessHighEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", FullAccess: true, Effort: "high"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--print",
		"--output-format", "json",
		"--add-dir", "/repo",
		"--dangerously-skip-permissions",
		"--effort", "high",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerBuildsModelArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Model: "fable"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--print",
		"--output-format", "json",
		"--add-dir", "/repo",
		"--permission-mode", "auto",
		"--model", "fable",
		"--effort", "xhigh",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerBuildsResumeArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "continue work", Resume: "session-1"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--print",
		"--output-format", "json",
		"--add-dir", "/repo",
		"--permission-mode", "auto",
		"--resume", "session-1",
		"--effort", "xhigh",
		"continue work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerNormalizesJSON(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0, Stdout: `{"result":"done","session_id":"session-1"}`}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentSession != "session-1" {
		t.Fatalf("AgentSession = %q", result.AgentSession)
	}
	if result.Stdout != "done" {
		t.Fatalf("Stdout = %q", result.Stdout)
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
