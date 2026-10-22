package codex

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestExecWithRunnerSendsPromptOnStdin(t *testing.T) {
	stubLookPath(t)
	prompt := strings.Repeat("quotes: \" & | % !\r\nUnicode: café 日本語\n", 2000)
	for _, resume := range []string{"", "fixture-thread"} {
		for _, fullAccess := range []bool{false, true} {
			run := &testsupport.RecordingRunner{}
			_, err := ExecWithRunner(context.Background(), run, Options{
				CWD: "/repo", Prompt: prompt, Resume: resume, FullAccess: fullAccess,
			})
			if err != nil {
				t.Fatal(err)
			}
			if run.Stdin != prompt {
				t.Fatal("prompt changed on stdin")
			}
			if run.Args[len(run.Args)-1] != "-" {
				t.Fatalf("missing stdin marker: %q", run.Args)
			}
			for _, arg := range run.Args {
				if strings.Contains(arg, "quotes:") {
					t.Fatal("prompt leaked into argv")
				}
			}
		}
	}
}

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
		"-",
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
		"-",
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
		"-",
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
		"-",
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
				"-",
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
		"-",
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
		"-",
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
		"-",
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
		"-",
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

func TestCapacityClassificationUsesTerminalOutcome(t *testing.T) {
	for _, tt := range []struct {
		name, stdout, stderr string
		wantCapacity         bool
	}{
		{"ordinary failure after capacity history", `{"type":"thread.started","thread_id":"review-session"}
{"type":"error","message":"Selected model is at capacity, retrying"}
{"type":"turn.failed","error":{"message":"Authentication token expired"}}`, "", false},
		{"ordinary failure with historical stderr", `{"type":"turn.failed","error":{"message":"Authentication token expired"}}`, "WARN recovered earlier error: Selected model is at capacity", false},
		{"ordinary failure with capacity diagnostic on stderr", `{"type":"turn.failed","error":{"message":"Authentication token expired"}}`, "Selected model is at capacity", false},
		{"stderr historical mention without terminal event", "", "WARN recovered earlier error: Selected model is at capacity", false},
		{"stderr diagnostic without terminal event", "", "Selected model is at capacity", true},
		{"stderr error diagnostic without terminal event", "", "error: Selected model is at capacity, try again later", true},
		{"ordinary quoted diagnostic", `{"type":"turn.failed","error":{"message":"Failed to write 'Selected model is at capacity.log'"}}`, "", false},
		{"later capacity terminal event", `{"type":"turn.failed","error":{"message":"Authentication token expired"}}
{"type":"turn.failed","error":{"message":"Selected model is at capacity"}}`, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := Result{ExitCode: 1, Stdout: tt.stdout, Stderr: tt.stderr}
			normalizeJSONResult(&res)
			if res.CapacityExhausted != tt.wantCapacity {
				t.Fatalf("capacity = %v, want %v: %#v", res.CapacityExhausted, tt.wantCapacity, res)
			}
		})
	}
}

func TestOrphanFilteringPreservesQuotedFilename(t *testing.T) {
	raw := "Orphan custom tool call: call_123\nERROR failed to write diagnostic file '/repo/Orphan custom tool call.log': Permission denied\n"
	res := Result{Stderr: raw}
	normalizeJSONResult(&res)
	want := "ERROR failed to write diagnostic file '/repo/Orphan custom tool call.log': Permission denied"
	if res.Stderr != want || res.RawStderr != raw {
		t.Fatalf("stderr normalization: %#v", res)
	}
}

func TestResumeKeepsProfileOnParentCommand(t *testing.T) {
	args := buildResumeArgs(Options{Profile: "retry-profile", Resume: "session", Prompt: "continue"})
	wantPrefix := []string{"exec", "--profile", "retry-profile", "resume", "--json"}
	if !reflect.DeepEqual(args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("resume profile: %q", args)
	}
}

func TestTerminalStreamEnvelopes(t *testing.T) {
	for _, stream := range []string{
		`{"type":"task_complete","last_agent_message":"final from terminal"}`,
		`{"type":"event_msg","payload":{"type":"task_complete","last_agent_message":"final from terminal"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","phase":"final_answer","content":[{"type":"output_text","text":"final from terminal"}]}}`,
		`{"type":"item.completed","item":{"type":"agent_message","phase":"final_answer","text":"final from terminal"}}`,
	} {
		state := streamState{}
		progress, changed := state.consume(stream)
		if !changed || !progress.FinalReceived || progress.Message != "final from terminal" || !state.success {
			t.Fatalf("terminal stream %s: %#v", stream, state)
		}
		res := Result{Stdout: stream, ExitCode: -1, CompletionStopped: true}
		normalizeJSONResult(&res)
		if res.ExitCode != 0 || res.Stdout != "final from terminal" {
			t.Fatalf("completion outcome: %#v", res)
		}
	}
}

func TestProgressMessageDoesNotEndTurn(t *testing.T) {
	state := streamState{}
	progress, _ := state.consume(`{"type":"item.completed","item":{"type":"agent_message","text":"Still working"}}`)
	if progress.FinalReceived || state.terminal {
		t.Fatal("progress started shutdown")
	}
	state.consume(`{"type":"turn.failed","error":{"message":"Selected model is at capacity"}}`)
	res := Result{Stdout: `{"type":"turn.failed","error":{"message":"Selected model is at capacity"}}`, CompletionStopped: true}
	normalizeJSONResult(&res)
	if !res.CapacityExhausted || res.ExitCode != 1 {
		t.Fatalf("capacity shutdown outcome: %#v", res)
	}
}
