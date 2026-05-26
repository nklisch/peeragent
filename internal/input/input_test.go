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

func TestParseDefaultsAgentCodex(t *testing.T) {
	req, err := Parse([]string{"task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Agent != "codex" {
		t.Fatalf("Agent = %q", req.Agent)
	}
}

func TestParseAgentGemini(t *testing.T) {
	req, err := Parse([]string{"--agent", "gemini", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Agent != "gemini" {
		t.Fatalf("Agent = %q", req.Agent)
	}
}

func TestParseAgentClaude(t *testing.T) {
	req, err := Parse([]string{"--agent", "claude", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Agent != "claude" {
		t.Fatalf("Agent = %q", req.Agent)
	}
}

func TestParseRejectsUnsupportedAgent(t *testing.T) {
	_, err := Parse([]string{"--agent", "llama", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
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
	req, err := Parse([]string{"--profile", "alt-subagent", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Profile != "alt-subagent" {
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

func TestParseClaudeModel(t *testing.T) {
	req, err := Parse([]string{"--agent", "claude", "--model", "opus", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Model != "opus" {
		t.Fatalf("Model = %q", req.Model)
	}
}

func TestParseClaudeModelBeforeAgent(t *testing.T) {
	req, err := Parse([]string{"--model", "sonnet", "--agent", "claude", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Model != "sonnet" {
		t.Fatalf("Model = %q", req.Model)
	}
}

func TestParseRejectsUnsupportedClaudeModel(t *testing.T) {
	_, err := Parse([]string{"--agent", "claude", "--model", "llama", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseGeminiFixedModel(t *testing.T) {
	req, err := Parse([]string{"--agent", "gemini", "--model", "3.5", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Model != "gemini-3.5" {
		t.Fatalf("Model = %q", req.Model)
	}
}

func TestParseRejectsUnsupportedGeminiModel(t *testing.T) {
	_, err := Parse([]string{"--agent", "gemini", "--model", "pro", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRejectsModelForCodex(t *testing.T) {
	_, err := Parse([]string{"--model", "opus", "task"}, nil, fixedCWD)
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

func TestParseCancel(t *testing.T) {
	req, err := Parse([]string{"--cancel", "job-1"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.CancelJobID != "job-1" {
		t.Fatalf("CancelJobID = %q", req.CancelJobID)
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
