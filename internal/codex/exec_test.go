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

func TestExecWithRunnerBuildsGPT6ModelArgv(t *testing.T) {
	stubLookPath(t)
	for _, model := range []string{"gpt-6-astra", "gpt-6-luna", "gpt-6-sol", "gpt-6.1-sol"} {
		t.Run(model, func(t *testing.T) {
			run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

			_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Model: model})
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
				"--model", model,
				"-c", `model_reasoning_effort="high"`,
				"do work",
			}
			if !reflect.DeepEqual(run.Args, wantArgs) {
				t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
			}
		})
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

func TestCapacityClassificationAndOrphanFiltering(t *testing.T) {
	raw := "Orphan custom tool call: call_123\nimportant diagnostic\n"
	res := Result{ExitCode: 1, Stdout: "{\"type\":\"thread.started\",\"thread_id\":\"session-capacity\"}\n{\"type\":\"turn.failed\",\"error\":{\"message\":\"Selected model is at capacity\"}}", Stderr: raw}
	normalizeJSONResult(&res)
	if !res.CapacityExhausted || res.AgentSession != "session-capacity" || res.Stderr != "important diagnostic" || res.RawStderr != raw {
		t.Fatalf("capacity normalization: %#v", res)
	}
	res = Result{Stdout: `{"type":"item.completed","item":{"type":"agent_message","text":"Added detection for Selected model is at capacity"}}`}
	normalizeJSONResult(&res)
	if res.CapacityExhausted {
		t.Fatal("ordinary final message classified as capacity failure")
	}
}

func TestRecoveredCapacityDiagnosticDoesNotRetryCompletedWork(t *testing.T) {
	res := Result{Stdout: `{"type":"error","message":"Selected model is at capacity"}
{"type":"item.completed","item":{"type":"agent_message","text":"done"}}
{"type":"turn.completed"}`, Stderr: "Selected model is at capacity"}
	normalizeJSONResult(&res)
	if res.CapacityExhausted || res.Stdout != "done" {
		t.Fatalf("completed turn: %#v", res)
	}
}

func TestResumeKeepsProfileOnParentCommand(t *testing.T) {
	args := buildResumeArgs(Options{Profile: "retry-profile", Resume: "session", Prompt: "continue"})
	wantPrefix := []string{"exec", "--profile", "retry-profile", "resume", "--json"}
	if !reflect.DeepEqual(args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("resume profile: %q", args)
	}
}
