package input

import (
	"os"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	req, err := Parse([]string{"implement", "the", "thing"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "implement the thing" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
	if req.CWD != "/repo" {
		t.Fatalf("CWD = %q", req.CWD)
	}
}

func TestParsePromptFile(t *testing.T) {
	file := writeTempPrompt(t, "from file\n")
	req, err := Parse([]string{"--prompt-file", file}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "from file" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
}

func TestParseStdin(t *testing.T) {
	req, err := Parse(nil, strings.NewReader("from stdin\n"), fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "from stdin" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
}

func TestParseCombinesInputs(t *testing.T) {
	file := writeTempPrompt(t, "from file")
	req, err := Parse([]string{"prefix", "--prompt-file", file}, strings.NewReader("from stdin"), fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	want := "prefix\n\nfrom file\n\nfrom stdin"
	if req.TaskText != want {
		t.Fatalf("TaskText = %q, want %q", req.TaskText, want)
	}
}

func TestParseCWDOverride(t *testing.T) {
	req, err := Parse([]string{"--cwd", "/other", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.CWD != "/other" {
		t.Fatalf("CWD = %q", req.CWD)
	}
}

func TestParseFullAccess(t *testing.T) {
	req, err := Parse([]string{"--full-access", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if !req.FullAccess {
		t.Fatal("expected FullAccess")
	}
}

func TestParseWorktree(t *testing.T) {
	req, err := Parse([]string{"--worktree", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if !req.Worktree {
		t.Fatal("expected Worktree")
	}
}

func TestParseProfile(t *testing.T) {
	req, err := Parse([]string{"--profile", "codex-subagent", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Profile != "codex-subagent" {
		t.Fatalf("Profile = %q", req.Profile)
	}
}

func TestParseProfileRequiresValue(t *testing.T) {
	_, err := Parse([]string{"--profile"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDefaultsEffortMedium(t *testing.T) {
	req, err := Parse([]string{"task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "medium" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseEffortHigh(t *testing.T) {
	req, err := Parse([]string{"--effort", "high", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "high" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseRejectsUnsupportedEffort(t *testing.T) {
	_, err := Parse([]string{"--effort", "xhigh", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAsync(t *testing.T) {
	req, err := Parse([]string{"--async", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if !req.Async {
		t.Fatal("expected Async")
	}
}

func TestParseJobRun(t *testing.T) {
	req, err := Parse([]string{"--job-run", "job-1", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.JobRunID != "job-1" {
		t.Fatalf("JobRunID = %q", req.JobRunID)
	}
}

func TestParseStatus(t *testing.T) {
	req, err := Parse([]string{"--status", "job-1", "ignored"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.StatusJobID != "job-1" {
		t.Fatalf("StatusJobID = %q", req.StatusJobID)
	}
}

func TestParseResult(t *testing.T) {
	req, err := Parse([]string{"--result", "job-1", "ignored"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.ResultJobID != "job-1" {
		t.Fatalf("ResultJobID = %q", req.ResultJobID)
	}
}

func TestParseRequiresTaskText(t *testing.T) {
	_, err := Parse(nil, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func fixedCWD() (string, error) {
	return "/repo", nil
}

func writeTempPrompt(t *testing.T, content string) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "prompt-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}
