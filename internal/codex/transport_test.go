package codex

import (
	"context"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestTransportStopClassification(t *testing.T) {
	stubLookPath(t)
	const start = `{"type":"thread.started","thread_id":"incident-session"}` + "\n"
	const reconnects = `{"type":"error","message":"Reconnecting... 1/2 (connection reset)"}` + "\n" + `{"type":"error","message":"Reconnecting... 2/2 (connection reset)"}` + "\n"
	const compact = `{"type":"turn.failed","error":{"message":"Error running remote compact task: unexpected status 401 Unauthorized"}}`
	const logs = "2026-10-31T05:13:36Z ERROR codex_api::endpoint::responses_websocket: failed to connect to Responses WebSocket stage=Unknown\n2026-10-31T05:13:37Z ERROR codex_api::endpoint::responses_websocket: failed to connect to Responses WebSocket stage=Unknown"
	for _, tc := range []struct {
		name, stdout, stderr string
		exit                 int
		want                 bool
	}{
		{"incident", start + reconnects + compact, logs, 1, true},
		{"exhausted reconnect", start + `{"type":"turn.failed","error":{"message":"Reconnecting... 2/2 (connection reset)"}}`, "", 1, true},
		{"repeated disconnect", start + `{"type":"error","message":"stream disconnected before completion"}` + "\n" + `{"type":"turn.failed","error":{"message":"stream disconnected before completion"}}`, logs, 1, true},
		{"repeated disconnect events", start + `{"type":"error","message":"stream disconnected before completion"}` + "\n" + `{"type":"error","message":"stream disconnected before completion"}` + "\n" + `{"type":"turn.failed"}`, "", 1, true},
		{"stderr connects", start + `{"type":"turn.failed"}`, logs, 1, true},
		{"single failure", start + `{"type":"error","message":"stream disconnected before completion"}` + "\n" + `{"type":"turn.failed","error":{"message":"stream disconnected before completion"}}`, "", 1, false},
		{"same failure in both streams", start + `{"type":"turn.failed","error":{"message":"failed to connect to Responses WebSocket"}}`, "2026-10-31T05:13:36Z ERROR codex_api::endpoint::responses_websocket: failed to connect to Responses WebSocket stage=Unknown", 1, false},
		{"unrelated terminal failure", start + reconnects + `{"type":"turn.failed","error":{"message":"tool failed"}}`, logs, 1, false},
		{"usage takes precedence", start + reconnects + `{"type":"turn.failed","error":{"message":"You've hit your usage limit. Try again tomorrow."}}`, logs, 1, false},
		{"success", start + reconnects + `{"type":"task_complete","last_agent_message":"done"}`, logs, 0, false},
		{"earlier completed turn", start + `{"type":"turn.completed"}` + "\n" + `{"type":"turn.started"}` + "\n" + reconnects + compact, logs, 1, false},
		{"quoted history", start + `{"type":"item.completed","item":{"type":"agent_message","text":"Reconnecting... 2/2"}}` + "\n" + `{"type":"turn.failed","error":{"message":"tool failed"}}`, "", 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &testsupport.RecordingRunner{Result: Result{Stdout: tc.stdout, Stderr: tc.stderr, ExitCode: tc.exit}}
			got, err := ExecWithRunner(context.Background(), runner, Options{})
			if err != nil || got.CapacityExhausted != tc.want {
				t.Fatalf("classification: %#v, %v", got, err)
			}
		})
	}
}
