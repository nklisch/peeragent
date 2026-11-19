package gemini

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestExecWithRunnerBuildsDefaultArgv(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0, Stdout: "ok"}}

	result, err := ExecWithRunner(context.Background(), run, Options{
		CWD: "/repo", Prompt: "do work", Effort: "high",
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
		"--output-format", "stream-json",
		"--mode", "accept-edits",
		"--effort", "high",
		"--sandbox",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
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
		Model: "gemini-4-neon", Effort: "xhigh",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--output-format", "stream-json",
		"--mode", "accept-edits",
		"--model", "gemini-4-neon",
		"--effort", "xhigh",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
		"--print", "do work",
	}
	if !reflect.DeepEqual(run.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", run.Args, wantArgs)
	}
}

func TestExecWithRunnerOmitsEffortWhenEmpty(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{ExitCode: 0}}

	_, err := ExecWithRunner(context.Background(), run, Options{
		CWD: "/repo", Prompt: "do work", Model: "gemini-4-neon",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--output-format", "stream-json",
		"--mode", "accept-edits",
		"--model", "gemini-4-neon",
		"--sandbox",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
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
		CWD: "/repo", Prompt: "continue work", Model: "gemini-4-carbon", Effort: "high",
		Resume: "conversation-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"--output-format", "stream-json",
		"--mode", "accept-edits",
		"--model", "gemini-4-carbon",
		"--effort", "high",
		"--sandbox",
		"--dangerously-skip-permissions",
		"--add-dir", "/repo",
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

func TestStreamStateConsumesStreamJSONLifecycle(t *testing.T) {
	var state streamState
	lines := []string{
		`{"event":"init","conversation_id":"conv-123","init":{"model":"gemini-4-neon","cwd":"/repo"}}`,
		`{"event":"step_update","step_update":{"conversation_id":"conv-123","step_index":0,"state":"DONE","step_type":"user_input"}}`,
		`{"event":"step_update","step_update":{"conversation_id":"conv-123","step_index":1,"state":"ACTIVE","step_type":"tool","tool_name":"run_command"}}`,
		`{"event":"step_update","step_update":{"conversation_id":"conv-123","step_index":1,"state":"DONE","step_type":"tool","tool_name":"run_command"}}`,
		`{"event":"step_update","step_update":{"conversation_id":"conv-123","step_index":2,"state":"ACTIVE","step_type":"agent_response","text_delta":"All "}}`,
		`{"event":"step_update","step_update":{"conversation_id":"conv-123","step_index":2,"state":"ACTIVE","step_type":"agent_response","text_delta":"done"}}`,
		`{"event":"step_update","step_update":{"conversation_id":"conv-123","step_index":2,"state":"DONE","step_type":"agent_response","text_delta":"."}}`,
		`{"event":"result","result":{"conversation_id":"conv-123","status":"SUCCESS","response":"All done.\n","num_turns":1}}`,
	}
	var events []executil.Progress
	for _, line := range lines {
		progress, changed := state.consume(line)
		if changed {
			events = append(events, progress)
		}
	}

	// Token deltas while ACTIVE should not emit separate progress records.
	wantEvents := []string{"init", "user_input", "tool:run_command", "tool:run_command", "agent_response", "result"}
	var gotEvents []string
	for _, ev := range events {
		gotEvents = append(gotEvents, ev.Event)
	}
	if !reflect.DeepEqual(gotEvents, wantEvents) {
		t.Fatalf("events = %#v, want %#v", gotEvents, wantEvents)
	}
	if events[0].AgentSession != "conv-123" {
		t.Fatalf("init session = %q", events[0].AgentSession)
	}
	if events[4].Message != "All done." || events[4].FinalReceived {
		t.Fatalf("agent_response progress = %#v", events[4])
	}
	if !events[5].FinalReceived || state.observation != executil.CompletionPending {
		t.Fatalf("result progress = %#v, observation = %v", events[5], state.observation)
	}

	res := Result{ExitCode: 0, Stdout: strings.Join(lines, "\n")}
	normalizeStreamResult(&res, state)
	if !res.TurnStarted || res.AgentSession != "conv-123" || res.FinalMessage != "All done.\n" || res.Stdout != "All done.\n" {
		t.Fatalf("normalized = %#v", res)
	}
}

func TestStreamStateRevokesCompletionOnSubsequentActivity(t *testing.T) {
	var state streamState
	state.consume(`{"event":"result","result":{"conversation_id":"conv-1","status":"SUCCESS","response":"draft","num_turns":1}}`)
	if !state.terminal || state.observation != executil.CompletionPending {
		t.Fatalf("expected CompletionPending, got terminal=%t obs=%v", state.terminal, state.observation)
	}
	state.consume(`{"event":"step_update","step_update":{"conversation_id":"conv-1","step_index":3,"state":"ACTIVE","step_type":"tool","tool_name":"run_command"}}`)
	if state.terminal || state.observation != executil.CompletionActive {
		t.Fatalf("expected CompletionActive, got terminal=%t obs=%v", state.terminal, state.observation)
	}
}

func TestStreamJSONPreTurnErrorDoesNotMarkTurnStarted(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{
		ExitCode: 1,
		Stdout:   `{"event":"result","result":{"conversation_id":"","status":"ERROR","response":"","error":"invalid model selection","num_turns":0,"finished_reason":"error"}}`,
	}}

	res, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TurnStarted {
		t.Fatal("pre-turn error should not mark TurnStarted")
	}
	if res.FailureMessage != "invalid model selection" {
		t.Fatalf("FailureMessage = %q", res.FailureMessage)
	}
	if res.FinalMessage != "" {
		t.Fatalf("FinalMessage = %q, want empty", res.FinalMessage)
	}
}

func TestStreamJSONErrorClassifications(t *testing.T) {
	tests := []struct {
		name     string
		errText  string
		auth     bool
		capacity bool
		usage    bool
	}{
		{"auth", "Authentication required. Run agy to sign in.", true, false, false},
		{"capacity", "RESOURCE_EXHAUSTED: selected model is at capacity", false, true, false},
		{"usage", "Quota exceeded for this account", false, false, true},
		{"generic", "tool execution failed", false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Result{
				ExitCode: 0,
				Stdout:   `{"event":"result","result":{"conversation_id":"conv-1","status":"ERROR","response":"","error":"` + tt.errText + `","num_turns":1}}`,
			}
			normalizeResult(&res)
			if res.ExitCode != 1 {
				t.Fatalf("ExitCode = %d, want 1", res.ExitCode)
			}
			if res.AuthRequired != tt.auth || res.CapacityExhausted != tt.capacity || res.UsageLimited != tt.usage {
				t.Fatalf("flags = auth:%t cap:%t usage:%t", res.AuthRequired, res.CapacityExhausted, res.UsageLimited)
			}
			if res.FailureMessage != tt.errText {
				t.Fatalf("FailureMessage = %q", res.FailureMessage)
			}
		})
	}
}

func TestCompletionStoppedRespectsTerminalOutcome(t *testing.T) {
	var state streamState
	state.consume(`{"event":"result","result":{"conversation_id":"conv-1","status":"SUCCESS","response":"finished","num_turns":1}}`)
	res := Result{ExitCode: 137, CompletionStopped: true}
	normalizeStreamResult(&res, state)
	if res.ExitCode != 0 || res.FinalMessage != "finished" {
		t.Fatalf("res = %#v", res)
	}

	var failedState streamState
	failedState.consume(`{"event":"result","result":{"conversation_id":"conv-1","status":"ERROR","error":"boom","num_turns":1}}`)
	failedRes := Result{ExitCode: 0, CompletionStopped: true}
	normalizeStreamResult(&failedRes, failedState)
	if failedRes.ExitCode != 1 || failedRes.FailureMessage != "boom" {
		t.Fatalf("failedRes = %#v", failedRes)
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

func TestExecWithRunnerNormalizesLegacyJSONEnvelopeAndCapturesSession(t *testing.T) {
	stubLookPath(t)
	run := &testsupport.RecordingRunner{Result: Result{
		ExitCode: 0,
		Stdout:   `{"conversation_id":"conversation-new","status":"SUCCESS","response":"completed"}`,
	}}

	result, err := ExecWithRunner(context.Background(), run, Options{CWD: "/repo", Prompt: "do work"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "completed" || result.FinalMessage != "completed" {
		t.Fatalf("Stdout = %q, FinalMessage = %q", result.Stdout, result.FinalMessage)
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
	if result.FailureMessage != "could not complete" {
		t.Fatalf("FailureMessage = %q", result.FailureMessage)
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

func TestTurnStartedRequiresPrintEnvelope(t *testing.T) {
	for _, test := range []struct {
		output  string
		started bool
	}{
		{"", false},
		{"startup failed", false},
		{`{"status":"SUCCESS","conversation_id":"session","response":"answer"}`, true},
		{`{"event":"init","conversation_id":"session"}`, true},
		{`{"event":"result","result":{"status":"ERROR","error":"bad model","num_turns":0}}`, false},
	} {
		res := Result{Stdout: test.output}
		normalizeResult(&res)
		if res.TurnStarted != test.started {
			t.Fatalf("output %q started %t", test.output, res.TurnStarted)
		}
	}
}
