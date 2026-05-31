package gemini

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
		"--print",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
	if run.name == "" {
		t.Fatal("expected agy path")
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
		"--print",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerIgnoresFixedModelArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work", Model: "gemini-3.5"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--print",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"do work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
	}
}

func TestExecWithRunnerBuildsResumeArgv(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{ExitCode: 0}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "continue work", Resume: "conversation-1"})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--print",
		"--add-dir", "/repo",
		"--print-timeout", "15m",
		"--conversation", "conversation-1",
		"continue work",
	}
	if !reflect.DeepEqual(run.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.args, wantArgs)
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
		{
			name: "successful output",
			input: Result{
				ExitCode: 0,
				Stdout:   "all looks good\n",
			},
			wantExit: 0,
		},
		{
			name: "print mode timeout in stdout",
			input: Result{
				ExitCode: 0,
				Stdout:   "some progress info\nError: timed out waiting for response\n",
			},
			wantExit: 1,
		},
		{
			name: "print mode timeout in stderr",
			input: Result{
				ExitCode: 0,
				Stderr:   "Error: timed out waiting for response\n",
			},
			wantExit: 1,
		},
		{
			name: "other error in output",
			input: Result{
				ExitCode: 0,
				Stdout:   "Error: failed to authenticate\n",
			},
			wantExit: 1,
		},
		{
			name: "non-zero exit code preserved",
			input: Result{
				ExitCode: 127,
				Stdout:   "Error: timed out waiting for response\n",
			},
			wantExit: 127,
		},
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


func TestExecWithRunnerFlagsPrintModeError(t *testing.T) {
	stubLookPath(t)
	run := &recordingRunner{result: Result{
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
	run := &recordingRunner{result: Result{ExitCode: 0, Stdout: "OK"}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit 0 for success, got %d", result.ExitCode)
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
