package result

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	encoded, err := FormatJSON(Result{Status: StatusSuccess, Summary: "done"})
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
}

func TestFormatText(t *testing.T) {
	text := FormatText(Result{
		Status:       StatusFailed,
		Summary:      "failed",
		ChangedFiles: []string{"main.go"},
		Verification: []Verification{{Command: "go test ./...", Status: "failed"}},
		Details:      "stderr",
		Metadata: Metadata{
			CWD:      "/repo",
			Agent:    "codex",
			Access:   "default",
			ExitCode: 1,
		},
	})
	for _, want := range []string{"Alt Subagent: failed", "Changed Files:", "Verification:", "Details:", "Metadata:", "agent: codex"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q: %s", want, text)
		}
	}
}
