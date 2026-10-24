package codex

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestReportedProviderSafetyFilter(t *testing.T) {
	stubLookPath(t)
	// Real error and turn.failed envelopes from Voxlar job
	// 20261024T192222Z-113df796; session/progress are synthetic.
	fixture, err := os.ReadFile("testdata/provider-safety-filter.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	raw := string(fixture)
	got, err := ExecWithRunner(context.Background(), &testsupport.RecordingRunner{Result: Result{ExitCode: 1, Stdout: raw}}, Options{})
	if err != nil || got.Failure != "policy_refusal" || !strings.HasPrefix(got.FailureMessage, "This content was flagged for possible cybersecurity risk.") || !strings.HasSuffix(got.FailureMessage, "before retrying.") {
		t.Fatalf("filter outcome: %#v %v", got, err)
	}
	state := streamState{}
	for _, line := range strings.Split(raw, "\n") {
		state.consume(line)
	}
	live := Result{ExitCode: 1}
	normalizeStreamResult(&live, state)
	if live.Failure != got.Failure || live.FailureMessage != got.FailureMessage || live.AgentSession != "filtered-session" || live.FinalMessage != "Checks are continuing." {
		t.Fatalf("stream outcome: %#v", live)
	}
	// A later genuine success revokes the failed-turn classification.
	state.consume(`{"type":"task_complete","last_agent_message":"done"}`)
	normalizeStreamResult(&live, state)
	if live.Failure != "" || live.FailureMessage != "" {
		t.Fatalf("stale failure: %#v", live)
	}
}
