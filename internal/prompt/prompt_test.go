package prompt

import (
	"strings"
	"testing"
)

func TestBuildIncludesTaskText(t *testing.T) {
	got := Build("add parser tests")
	if !strings.Contains(got, "add parser tests") {
		t.Fatalf("prompt does not include task text: %q", got)
	}
}

func TestBuildIncludesOperatingInstructions(t *testing.T) {
	got := Build("task")
	for _, want := range []string{
		"current repository",
		"Make the requested code changes directly",
		"Run relevant verification",
		"report the blocker",
		"changed files",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q: %q", want, got)
		}
	}
}
