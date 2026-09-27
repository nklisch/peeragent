package zai

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestWindowsPromptUsesStdin(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows prompt transport")
	}
	stubLookPath(t)
	prompt := strings.Repeat("a", 40000) + "\n" + `odd " & | >`
	run := &testsupport.RecordingRunner{}
	if _, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: prompt}); err != nil {
		t.Fatal(err)
	}
	if run.Stdin != prompt || strings.Contains(strings.Join(run.Args, " "), prompt) {
		t.Fatal("Pi prompt was not sent intact through stdin")
	}
}

func TestExecWithRunnerBuildsDefaultArgv(t *testing.T) {
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
		"--provider", "zai",
		"--model", "glm-5.2",
		"--thinking", "high",
		"--no-session",
		"-p",
		"do work",
	}
	if !reflect.DeepEqual(run.Args, testsupport.ExpectedPromptArgs(wantArgs, "pi")) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
	if run.Name == "" {
		t.Fatal("expected pi path")
	}
	if runtime.GOOS == "windows" && run.Stdin != "do work" {
		t.Fatalf("stdin prompt = %q", run.Stdin)
	}
}

func TestExecWithRunnerBuildsEffortArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

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
	if !reflect.DeepEqual(run.Args, testsupport.ExpectedPromptArgs(wantArgs, "pi")) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerBuildsResumeArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

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
	if !reflect.DeepEqual(run.Args, testsupport.ExpectedPromptArgs(wantArgs, "pi")) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
	if result.AgentSession != "session-1" {
		t.Fatalf("AgentSession = %q", result.AgentSession)
	}
}

func TestExecWithRunnerAcceptsExplicitFixedModel(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

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
	if !reflect.DeepEqual(run.Args, testsupport.ExpectedPromptArgs(wantArgs, "pi")) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
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
