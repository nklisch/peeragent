package gemini

import (
	"context"
	"reflect"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestExecWithRunnerBuildsDefaultArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0, Stdout: "ok"}}

	result, err := ExecWithRunner(context.Background(), run, Options{
		CWD: "/repo", Prompt: "do work", Model: "gemini-3.7-flash", Effort: "high",
	})
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
		"--output-format", "json",
		"--model", "gemini-3.7-flash",
		"--effort", "high",
		"--mode", "accept-edits",
		"--sandbox",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"--print", "do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
	if run.Name == "" {
		t.Fatal("expected agy path")
	}
}

func TestExecWithRunnerBuildsFullAccessArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{
		CWD: "/repo", Prompt: "do work", FullAccess: true,
		Model: "gemini-3.7-flash", Effort: "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--output-format", "json",
		"--model", "gemini-3.7-flash",
		"--effort", "high",
		"--mode", "accept-edits",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"--print", "do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerPassesModelAndEffort(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{
		CWD: "/repo", Prompt: "do work", Model: "gemini-3.1-pro", Effort: "low",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--output-format", "json",
		"--model", "gemini-3.1-pro",
		"--effort", "low",
		"--mode", "accept-edits",
		"--sandbox",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"--print", "do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsResumeArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	result, err := ExecWithRunner(context.Background(), run, Options{
		CWD: "/repo", Prompt: "continue work", Model: "gemini-3.7-flash", Effort: "high",
		Resume: "conversation-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--output-format", "json",
		"--model", "gemini-3.7-flash",
		"--effort", "high",
		"--mode", "accept-edits",
		"--sandbox",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"--conversation", "conversation-1",
		"--print", "continue work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
	if result.AgentSession != "conversation-1" {
		t.Fatalf("AgentSession = %q", result.AgentSession)
	}
}

func TestNormalizeResult(t *testing.T) {
	tests := []struct {
		name     string
		input    Result
		wantExit int
	}{
		{name: "successful output", input: Result{ExitCode: 0, Stdout: "all looks good\n"}, wantExit: 0},
		{name: "print mode timeout in stdout", input: Result{ExitCode: 0, Stdout: "some progress info\nError: timed out waiting for response\n"}, wantExit: 1},
		{name: "print mode timeout in stderr", input: Result{ExitCode: 0, Stderr: "Error: timed out waiting for response\n"}, wantExit: 1},
		{name: "auth failure in output", input: Result{ExitCode: 0, Stdout: "Error: Authentication required\n"}, wantExit: 1},
		{name: "non-zero exit code preserved", input: Result{ExitCode: 127, Stdout: "Error: timed out waiting for response\n"}, wantExit: 127},
		{name: "legitimate agent error report stays success", input: Result{ExitCode: 0, Stdout: "Reviewed the change.\nError: missing validation in foo.go\n"}, wantExit: 0},
		{name: "headless permission denial", input: Result{ExitCode: 0, Stderr: `a tool required the "command" permission that headless mode cannot prompt for, so it was auto-denied.`}, wantExit: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.input
			normalizeResult(&res)
			if res.ExitCode != tt.wantExit {
				t.Errorf("ExitCode = %d, want %d", res.ExitCode, tt.wantExit)
			}
		})
	}
}

func TestExecWithRunnerNormalizesStructuredOutputAndCapturesSession(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{
		ExitCode: 0,
		Stdout:   `{"conversation_id":"conversation-new","status":"SUCCESS","response":"completed"}`,
	}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "completed" {
		t.Fatalf("Stdout = %q", result.Stdout)
	}
	if result.AgentSession != "conversation-new" {
		t.Fatalf("AgentSession = %q", result.AgentSession)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d", result.ExitCode)
	}
}

func TestExecWithRunnerFlagsStructuredFailure(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{
		ExitCode: 0,
		Stdout:   `{"conversation_id":"conversation-new","status":"FAILED","response":"could not complete"}`,
	}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == 0 {
		t.Fatal("expected structured failure to produce a non-zero exit")
	}
}

func TestExecWithRunnerFlagsPrintModeError(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{
		ExitCode: 0,
		Stdout:   "I will explore the repo.\nError: timed out waiting for response",
	}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == 0 {
		t.Fatal("expected non-zero exit for agy print-mode error, got 0")
	}
}

func TestExecWithRunnerKeepsSuccessExitCode(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0, Stdout: "OK"}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit 0 for success, got %d", result.ExitCode)
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
