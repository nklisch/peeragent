package codex

import (
	"context"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestOutputLimitOutcomes(t *testing.T) {
	stubLookPath(t)
	// The error + turn.failed pair is the actual diagnostic from the Voxlar
	// handback incident. No incident commands or repository output are copied.
	prefix := "{\"type\":\"thread.started\",\"thread_id\":\"limited-session\"}\n{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"partial handback\"}}\n"
	prefix += `{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"test output"}}` + "\n"
	const diagnostic = "Incomplete response returned, reason: max_output_tokens"
	for _, terminal := range []string{
		`{"type":"error","message":"` + diagnostic + `"}` + "\n" + `{"type":"turn.failed","error":{"message":"` + diagnostic + `"}}`,
		`{"type":"event_msg","payload":{"type":"turn.failed","error":{"message":"` + diagnostic + `"}}}`,
	} {
		t.Run(terminal, func(t *testing.T) {
			runner := &testsupport.RecordingRunner{Result: Result{Stdout: prefix + terminal, CompletionStopped: true, ExitCode: -1}}
			res, err := ExecWithRunner(context.Background(), runner, Options{Prompt: "task"})
			if err != nil || !res.OutputLimit || res.CapacityExhausted || res.ExitCode != 1 || res.AgentSession != "limited-session" || res.Stdout != "partial handback" {
				t.Fatalf("output limit: %#v %v", res, err)
			}
			state := streamState{}
			for _, line := range strings.Split(prefix+terminal, "\n") {
				state.consume(line)
			}
			if !state.outputLimit || state.observation != executil.CompletionPending || state.success {
				t.Fatalf("live parser: %#v", state)
			}
			for _, later := range []string{`{"type":"turn.failed","error":{"message":"Authentication failed"}}`, `{"type":"task_complete","last_agent_message":"done"}`} {
				res = Result{Stdout: prefix + terminal + "\n" + later}
				normalizeJSONResult(&res)
				if res.OutputLimit {
					t.Fatalf("stale diagnostic overrode outcome: %#v", res)
				}
			}
		})
	}
}

func TestOutputLimitDiagnosticsDoNotReadAssistantProse(t *testing.T) {
	for _, tc := range []struct {
		stdout, stderr string
		want           bool
	}{
		{`{"type":"item.completed","item":{"type":"agent_message","text":"Investigated max_output_tokens stops"}}`, "", false},
		{`{"type":"task_complete","last_agent_message":"Documented max_output_tokens"}`, "Error: Incomplete response returned, reason: max_output_tokens", false},
		{`{"type":"turn.failed","error":{"message":"ordinary failure"}}`, "Error: Incomplete response returned, reason: max_output_tokens", false},
		{"", "Error: Incomplete response returned, reason: max_output_tokens", true},
		{"", "quoted history: max_output_tokens", false},
		{"Incomplete response returned, reason: max_output_tokens", "", true},
		{`{"type":"turn.failed","error":{"message":"invalid configuration: max_output_tokens must be positive"}}`, "", false},
		{`{"type":"turn.failed","error":{"message":"Incomplete response returned, reason: max_output_tokens_extra"}}`, "", false},
	} {
		res := Result{Stdout: tc.stdout, Stderr: tc.stderr}
		normalizeJSONResult(&res)
		if res.OutputLimit != tc.want {
			t.Fatalf("%q / %q: %#v", tc.stdout, tc.stderr, res)
		}
	}
}
