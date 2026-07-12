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

func TestParseCombinesArgsAndStdin(t *testing.T) {
	req, err := Parse([]string{"task"}, strings.NewReader("ctx"), fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	want := "task\n\nctx"
	if req.TaskText != want {
		t.Fatalf("TaskText = %q, want %q", req.TaskText, want)
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

func TestParseReadsNonTTYStdinFile(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	if _, err := writer.WriteString("from pipe\n"); err != nil {
		writer.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := Parse(nil, reader, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "from pipe" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
}

func TestParseSkipsNonTTYStdinFileWhenTaskTextExists(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	req, err := Parse([]string{"task"}, reader, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "task" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
}

func TestParseSkipsTTYStdinFile(t *testing.T) {
	tty := openTestTTY(t)
	defer tty.Close()

	req, err := Parse([]string{"task"}, tty, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "task" {
		t.Fatalf("TaskText = %q", req.TaskText)
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

func TestParseSandbox(t *testing.T) {
	req, err := Parse([]string{"--sandbox", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "task" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
	if req.FullAccess || req.Worktree {
		t.Fatalf("unexpected access flags: FullAccess=%v Worktree=%v", req.FullAccess, req.Worktree)
	}
}

func TestParseRejectsSandboxWithFullAccess(t *testing.T) {
	_, err := Parse([]string{"--sandbox", "--full-access", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRejectsSandboxWithWorktree(t *testing.T) {
	_, err := Parse([]string{"--sandbox", "--worktree", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
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

func TestParseAgentZAI(t *testing.T) {
	req, err := Parse([]string{"--agent", "z.ai", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Agent != "zai" {
		t.Fatalf("Agent = %q", req.Agent)
	}
	if req.Effort != "high" {
		t.Fatalf("Effort = %q", req.Effort)
	}
	if req.Model != "glm-5.2" {
		t.Fatalf("Model = %q", req.Model)
	}
}

func TestParseAgentGLMAlias(t *testing.T) {
	req, err := Parse([]string{"--agent", "glm-5.2", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Agent != "zai" {
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
	req, err := Parse([]string{"--profile", "peeragent", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Profile != "peeragent" {
		t.Fatalf("Profile = %q", req.Profile)
	}
}

func TestParseProfileRequiresValue(t *testing.T) {
	_, err := Parse([]string{"--profile"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDefaultsEffortHighForCodex(t *testing.T) {
	req, err := Parse([]string{"task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "high" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseDefaultsEffortXHighForClaude(t *testing.T) {
	req, err := Parse([]string{"--agent", "claude", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "xhigh" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseDefaultsNoEffortForGemini(t *testing.T) {
	req, err := Parse([]string{"--agent", "gemini", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "" {
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

func TestParseEffortXHighForCodex(t *testing.T) {
	req, err := Parse([]string{"--effort", "xhigh", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "xhigh" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseEffortXHighBeforeAgentForCodex(t *testing.T) {
	req, err := Parse([]string{"--effort", "xhigh", "--agent", "codex", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "xhigh" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseEffortLowForCodex(t *testing.T) {
	req, err := Parse([]string{"--effort", "low", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "low" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseRejectsUnsupportedEffort(t *testing.T) {
	_, err := Parse([]string{"--effort", "minimal", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEffortXHighForClaude(t *testing.T) {
	req, err := Parse([]string{"--agent", "claude", "--effort", "xhigh", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "xhigh" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseEffortHighForClaude(t *testing.T) {
	req, err := Parse([]string{"--agent", "claude", "--effort", "high", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "high" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseRejectsMediumEffortForClaude(t *testing.T) {
	_, err := Parse([]string{"--agent", "claude", "--effort", "medium", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRejectsEffortForGemini(t *testing.T) {
	_, err := Parse([]string{"--agent", "gemini", "--effort", "high", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEffortXHighForZAI(t *testing.T) {
	req, err := Parse([]string{"--agent", "zai", "--effort", "xhigh", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Effort != "xhigh" {
		t.Fatalf("Effort = %q", req.Effort)
	}
}

func TestParseRejectsUnsupportedEffortForZAI(t *testing.T) {
	_, err := Parse([]string{"--agent", "zai", "--effort", "low", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseClaudeModel(t *testing.T) {
	for _, model := range []string{"fable", "opus"} {
		t.Run(model, func(t *testing.T) {
			req, err := Parse([]string{"--agent", "claude", "--model", model, "task"}, nil, fixedCWD)
			if err != nil {
				t.Fatal(err)
			}
			if req.Model != model {
				t.Fatalf("Model = %q", req.Model)
			}
		})
	}
}

func TestParseResume(t *testing.T) {
	req, err := Parse([]string{"--resume", "session-1", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Resume != "session-1" {
		t.Fatalf("Resume = %q", req.Resume)
	}
}

func TestParseResumeRequiresValue(t *testing.T) {
	_, err := Parse([]string{"--resume"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
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

func TestParseZAIModel(t *testing.T) {
	req, err := Parse([]string{"--agent", "zai", "--model", "zai/glm-5.2", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Model != "glm-5.2" {
		t.Fatalf("Model = %q", req.Model)
	}
}

func TestParseRejectsUnsupportedZAIModel(t *testing.T) {
	_, err := Parse([]string{"--agent", "zai", "--model", "glm-5.1", "task"}, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCodexModels(t *testing.T) {
	tests := map[string]string{
		"luna":          "gpt-5.6-luna",
		"gpt5.6-luna":   "gpt-5.6-luna",
		"gpt-5.6-luna":  "gpt-5.6-luna",
		"terra":         "gpt-5.6-terra",
		"gpt5.6-terra":  "gpt-5.6-terra",
		"gpt-5.6-terra": "gpt-5.6-terra",
		"sol":           "gpt-5.6-sol",
		"gpt5.6-sol":    "gpt-5.6-sol",
		"gpt-5.6-sol":   "gpt-5.6-sol",
	}
	for inputModel, want := range tests {
		t.Run(inputModel, func(t *testing.T) {
			req, err := Parse([]string{"--model", inputModel, "task"}, nil, fixedCWD)
			if err != nil {
				t.Fatal(err)
			}
			if req.Model != want {
				t.Fatalf("Model = %q, want %q", req.Model, want)
			}
		})
	}
}

func TestParseRejectsUnsupportedModelForCodex(t *testing.T) {
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
	req, err := Parse([]string{"--job-run", validJobID, "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.JobRunID != validJobID {
		t.Fatalf("JobRunID = %q", req.JobRunID)
	}
}

func TestParseJobRunAllowsEmptyTaskText(t *testing.T) {
	req, err := Parse([]string{"--job-run", validJobID}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
	if req.JobRunID != validJobID {
		t.Fatalf("JobRunID = %q", req.JobRunID)
	}
}

func TestParseStatus(t *testing.T) {
	req, err := Parse([]string{"--status", validJobID, "ignored"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.StatusJobID != validJobID {
		t.Fatalf("StatusJobID = %q", req.StatusJobID)
	}
}

func TestParseResult(t *testing.T) {
	req, err := Parse([]string{"--result", validJobID, "ignored"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.ResultJobID != validJobID {
		t.Fatalf("ResultJobID = %q", req.ResultJobID)
	}
}

func TestParseCancel(t *testing.T) {
	req, err := Parse([]string{"--cancel", validJobID}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.CancelJobID != validJobID {
		t.Fatalf("CancelJobID = %q", req.CancelJobID)
	}
}

func TestParseRejectsMalformedJobIDsAtCLIInputBoundary(t *testing.T) {
	invalidIDs := []string{
		"../escape",
		"..",
		".",
		"/absolute/path",
		`a\b`,
		"a/b",
		"not-a-job",
		"20260712T123456Z-deadbeeg",
		"20260712T12345Z-deadbeef",
		"20261340T123456Z-deadbeef",
		"20260712T123456Z-deadbeef\x00",
		"20260712T123456Z-😀😀😀😀",
	}
	for _, flag := range []string{"--job-run", "--status", "--result", "--cancel"} {
		for _, id := range invalidIDs {
			if _, err := Parse([]string{flag, id}, nil, fixedCWD); err == nil {
				t.Errorf("Parse(%s, %q) accepted malformed id", flag, id)
			}
		}
	}
}

func TestParseRequiresTaskText(t *testing.T) {
	_, err := Parse(nil, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

const validJobID = "20260712T123456Z-deadbeef"

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

func openTestTTY(t *testing.T) *os.File {
	t.Helper()

	for _, path := range []string{"/dev/ptmx", "/dev/tty"} {
		file, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			continue
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			continue
		}
		if info.Mode()&os.ModeCharDevice == 0 {
			file.Close()
			continue
		}
		return file
	}

	t.Skip("no TTY-like test device available")
	return nil
}
