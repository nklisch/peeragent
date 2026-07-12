package codex

import (
	"context"
	"reflect"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestExecWithRunnerBuildsArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0, Stdout: "ok"}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("Stdout = %q", result.Stdout)
	}
	if run.CWD != "/repo" {
		t.Fatalf("cwd = %q", run.CWD)
	}
	wantArgs := []string{
		"exec",
		"--json",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
	if run.Name == "" {
		t.Fatal("expected codex path")
	}
}

func TestExecWithRunnerBuildsFullAccessArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", FullAccess: true})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--json",
		"--cd", "/repo",
		"--dangerously-bypass-approvals-and-sandbox",
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsProfileArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Profile: "peeragent"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--json",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"--profile", "peeragent",
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsModelArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Model: "gpt-5.6-luna"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--json",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
		"--model", "gpt-5.6-luna",
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsLowEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Effort: "low"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--json",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
		"-c", `model_reasoning_effort="low"`,
		"do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsHighEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Effort: "high"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--json",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
		"-c", `model_reasoning_effort="high"`,
		"do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsXHighEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Effort: "xhigh"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"--json",
		"--cd", "/repo",
		"--sandbox", "workspace-write",
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
		"-c", `model_reasoning_effort="xhigh"`,
		"do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsResumeArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "continue work", Model: "gpt-5.6-sol", Resume: "019e6be9-b530-7ef3-96aa-989712db6ebb"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"exec",
		"resume",
		"--json",
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
		"--model", "gpt-5.6-sol",
		"-c", `model_reasoning_effort="high"`,
		"019e6be9-b530-7ef3-96aa-989712db6ebb",
		"continue work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerNormalizesJSONL(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0, Stdout: `{"type":"thread.started","thread_id":"thread-1"}
	{"type":"item.completed","item":{"type":"agent_message","text":"I am checking the repo."}}
	{"type":"item.completed","item":{"type":"tool_call","text":"ignored"}}
	{"type":"item.completed","item":{"type":"agent_message","text":"done"}}
	`}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.AgentSession != "thread-1" {
		t.Fatalf("AgentSession = %q", result.AgentSession)
	}
	if result.Stdout != "done" {
		t.Fatalf("Stdout = %q", result.Stdout)
	}
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
