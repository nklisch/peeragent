package codex

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestFailureDiagnosticsSurviveBareTerminal(t *testing.T) {
	stubLookPath(t)
	const usage = "You've hit your usage limit. Visit https://chatgpt.com/settings/usage to purchase more credits or try again at Nov 3rd, 2026 11:48 PM."
	const capacity = "Selected model is at capacity. Please try a different model."
	for _, tc := range []struct {
		name, diagnostic, later, want string
		capacity, usage               bool
	}{
		{name: "usage limit", diagnostic: usage, want: usage, usage: true},
		{name: "typographic apostrophe", diagnostic: strings.ReplaceAll(usage, "'", "’"), want: strings.ReplaceAll(usage, "'", "’"), usage: true},
		{name: "capacity", diagnostic: capacity, want: capacity, capacity: true},
		{name: "ordinary failure", diagnostic: "first error", later: `{"type":"turn.failed","error":{"message":"last error"}}`, want: "last error"},
		{name: "explicit failure replaces capacity", diagnostic: capacity, later: `{"type":"turn.failed","error":{"message":"last error"}}`, want: "last error"},
		{name: "success clears usage", diagnostic: usage, later: `{"type":"task_complete","last_agent_message":"done"}`},
		{name: "success clears capacity", diagnostic: capacity, later: `{"type":"task_complete","last_agent_message":"done"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			message, _ := json.Marshal(tc.diagnostic)
			raw := "{\"type\":\"thread.started\",\"thread_id\":\"incident-session\"}\n{\"type\":\"error\",\"message\":" + string(message) + "}\n{\"type\":\"turn.failed\"}"
			if tc.later != "" {
				raw += "\n" + tc.later
			}
			parsed, err := ExecWithRunner(context.Background(), &testsupport.RecordingRunner{Result: Result{Stdout: raw, ExitCode: 1}}, Options{Resume: "incident-session"})
			if err != nil {
				t.Fatal(err)
			}
			state := streamState{}
			for _, line := range strings.Split(raw, "\n") {
				state.consume(line)
			}
			live := Result{ExitCode: 1}
			normalizeStreamResult(&live, state)
			for _, got := range []Result{parsed, live} {
				if got.FailureMessage != tc.want || got.CapacityExhausted != tc.capacity || got.UsageLimited != tc.usage {
					t.Fatalf("diagnostic lost or misclassified: %#v", got)
				}
			}
		})
	}
}

func TestUsageLimitMentionsAreNotProviderStops(t *testing.T) {
	for _, raw := range []string{
		`{"type":"item.completed","item":{"type":"agent_message","text":"You've hit your usage limit."}}`,
		`{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"You've hit your usage limit."}}`,
		`{"type":"error","message":"quoted history: You've hit your usage limit."}`,
	} {
		got := Result{Stdout: raw}
		normalizeJSONResult(&got)
		if got.UsageLimited {
			t.Fatalf("prose classified as usage stop: %#v", got)
		}
	}
}
