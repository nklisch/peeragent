package codex

import (
	"context"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

const revokedLogin = "Your access token could not be refreshed because you have since logged out or signed in to another account. Please sign in again."
const rolloutLost = "2026-10-24T03:41:03.708607Z ERROR codex_core::session: failed to record rollout items: thread incident-session not found"
const policyRefusal = "This content was flagged for possible cybersecurity risk. If this seems wrong, try rephrasing your request."

func TestCodexRecoveryDiagnosticsAndTerminalPrecedence(t *testing.T) {
	stubLookPath(t)
	const prefix = "{\"type\":\"thread.started\",\"thread_id\":\"incident-session\"}\n"
	for _, tc := range []struct {
		name, stdout, stderr string
		auth, interrupted    bool
		failure              string
	}{
		{name: "revoked login events", stdout: `{"type":"error","message":"` + revokedLogin + `"}` + "\n" + `{"type":"turn.failed","error":{"message":"` + revokedLogin + `"}}`, auth: true},
		{name: "revoked login stderr", stderr: "2026-10-23T22:09:14.309462Z ERROR codex_models_manager::manager: request failed: auth error: 401, auth error code: token_revoked", auth: true},
		{name: "rollout loss without terminal", stderr: rolloutLost, interrupted: true},
		{name: "policy refusal is authoritative", stdout: `{"type":"turn.failed","error":{"message":"` + policyRefusal + `"}}`, stderr: rolloutLost, failure: "policy_refusal"},
		{name: "ordinary failure wins", stdout: `{"type":"turn.failed","error":{"message":"ordinary target failure"}}`, stderr: rolloutLost},
		{name: "success wins", stdout: `{"type":"task_complete","last_agent_message":"done"}`, stderr: rolloutLost + "\n" + revokedLogin},
		{name: "new turn revokes prior success", stdout: `{"type":"turn.completed"}` + "\n" + `{"type":"turn.started"}`, stderr: rolloutLost, interrupted: true},
		{name: "agent prose is inert", stdout: `{"type":"item.completed","item":{"type":"agent_message","text":"` + revokedLogin + `"}}`},
		{name: "tool output is inert", stdout: `{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"401 token_revoked"}}`},
		{name: "quoted log is inert", stderr: "quoted history: " + rolloutLost},
		{name: "unrelated thread failure", stderr: "thread incident-session not found"},
		{name: "generic HTTP error", stderr: "401 Unauthorized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &testsupport.RecordingRunner{Result: Result{ExitCode: 1, Stdout: prefix + tc.stdout, Stderr: tc.stderr}}
			got, err := ExecWithRunner(context.Background(), runner, Options{})
			if err != nil || got.AuthRequired != tc.auth || got.Interrupted != tc.interrupted || got.Failure != tc.failure || got.AgentSession != "incident-session" {
				t.Fatalf("outcome: %#v %v", got, err)
			}
			if tc.failure != "" && got.FailureMessage != policyRefusal {
				t.Fatalf("refusal reason: %q", got.FailureMessage)
			}
			if got.RawStderr != tc.stderr {
				t.Fatal("raw diagnostics lost")
			}
			// Exec's incremental parser and the recording runner share one classifier.
			state := streamState{}
			for _, line := range strings.Split(prefix+tc.stdout, "\n") {
				state.consume(line)
			}
			live := Result{ExitCode: 1, Stdout: prefix + tc.stdout, Stderr: tc.stderr}
			normalizeStreamResult(&live, state)
			if live.AuthRequired != got.AuthRequired || live.Interrupted != got.Interrupted || live.Failure != got.Failure {
				t.Fatalf("live parser disagrees: %#v", live)
			}
		})
	}
}
