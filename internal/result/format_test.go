package result

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	encoded, err := FormatJSON(Result{Status: StatusSuccess, Summary: "done", Metadata: Metadata{JobID: "job-id"}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["status"] != "success" {
		t.Fatalf("status = %v", decoded["status"])
	}
	if decoded["job_id"] != "job-id" || decoded["metadata"].(map[string]any)["job_id"] != "job-id" {
		t.Fatalf("job id aliases differ: %s", encoded)
	}
}

func TestFormatText(t *testing.T) {
	text := FormatText(Result{
		Status:       StatusFailed,
		Summary:      "failed",
		ChangedFiles: []string{"main.go"},
		Verification: []Verification{{Command: "go test ./...", Status: "failed"}},
		Details:      "stderr",
		Metadata: Metadata{
			CWD:          "/repo",
			Agent:        "codex",
			Access:       "default",
			Model:        "opus",
			AgentSession: "session-1",
			LogPath:      "/repo/.peeragent/runs/run.log",
			ExitCode:     1,
		},
	})
	for _, want := range []string{"Alt Subagent: failed", "Changed Files:", "Verification:", "Details:", "Metadata:", "agent: codex", "model: opus", "agent_session: session-1", "log_path: /repo/.peeragent/runs/run.log"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q: %s", want, text)
		}
	}
}

func TestTextResultIncludesMessagePointer(t *testing.T) {
	text := FormatText(Result{Status: StatusFailed, Metadata: Metadata{JobID: "job-id", MessagePath: "/job/message.txt", FinalReceived: true}})
	for _, want := range []string{"message_path: /job/message.txt", "final_received: true"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
}
