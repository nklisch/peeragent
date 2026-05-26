package result

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResultJSONFields(t *testing.T) {
	res := Result{
		Status:       StatusSuccess,
		Summary:      "done",
		ChangedFiles: []string{"main.go"},
		Verification: []Verification{{Command: "go test ./...", Status: "passed"}},
		Metadata: Metadata{
			CWD:      "/repo",
			Agent:    "codex",
			Access:   "default",
			Profile:  "alt-subagent",
			Effort:   "medium",
			ExitCode: 0,
		},
	}

	encoded, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	got := string(encoded)
	for _, want := range []string{
		`"status"`,
		`"summary"`,
		`"changed_files"`,
		`"verification"`,
		`"metadata"`,
		`"agent"`,
		`"exit_code"`,
		`"effort"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("encoded result missing %s: %s", want, got)
		}
	}
}
