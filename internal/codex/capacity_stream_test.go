package codex

import (
	"context"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestReportedCapacityStopStreams(t *testing.T) {
	stubLookPath(t)
	// Only the diagnostic/agent event shapes are copied from the incident logs;
	// commands and repository contents are deliberately excluded.
	for _, diagnostic := range []string{
		"Selected model is at capacity. Please try a different model.",
		"Error running remote compact task: Selected model is at capacity. Please try a different model.",
	} {
		t.Run(diagnostic, func(t *testing.T) {
			stream := `{"type":"thread.started","thread_id":"incident-session"}
{"type":"item.completed","item":{"type":"agent_message","text":"The baseline link build is still queued."}}
{"type":"error","message":"` + diagnostic + `"}
{"type":"turn.failed","error":{"message":"` + diagnostic + `"}}`
			runner := &testsupport.RecordingRunner{Result: Result{ExitCode: 1, Stdout: stream}}
			res, err := ExecWithRunner(context.Background(), runner, Options{CWD: "/repo", Prompt: "continue"})
			if err != nil || !res.CapacityExhausted || res.AgentSession != "incident-session" || res.Stdout != "The baseline link build is still queued." {
				t.Fatalf("incident outcome: %#v %v", res, err)
			}
			// A later explicit ordinary failure or success still decides the outcome.
			for _, terminal := range []string{`{"type":"turn.failed","error":{"message":"Authentication token expired"}}`, `{"type":"task_complete","last_agent_message":"done"}`} {
				res = Result{ExitCode: 1, Stdout: stream + "\n" + terminal}
				normalizeJSONResult(&res)
				if res.CapacityExhausted {
					t.Fatalf("history overrode terminal event: %#v", res)
				}
			}
		})
	}
}
