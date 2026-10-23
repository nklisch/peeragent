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

func TestBuildForAgentIncludesAgentName(t *testing.T) {
	got := BuildForAgent("Gemini through Antigravity CLI", "task")
	if !strings.Contains(got, "Gemini through Antigravity CLI") {
		t.Fatalf("prompt does not include agent name: %q", got)
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

func TestMessagingPreambleQuestionsOnly(t *testing.T) {
	for _, agent := range []string{"Codex", "Claude", "Gemini through Antigravity CLI", "Z.AI GLM 5.2 through Pi"} {
		t.Run(agent, func(t *testing.T) {
			got := BuildForAgent(agent, "task")
			normalized := strings.Join(strings.Fields(got), " ")
			for _, want := range []string{
				"Use `peeragent say` only to ask a question you can't settle yourself, never for status or progress updates.",
				"Your final handback carries the final status, so don't repeat it in a message.",
				"After asking a question, watch your inbox for the answer: run `peeragent inbox` every few minutes, between steps, until it arrives, and keep doing independent work meanwhile.",
			} {
				if !strings.Contains(normalized, want) {
					t.Fatalf("prompt missing required instruction %q: %s", want, got)
				}
			}
			if strings.Contains(got, "short status") {
				t.Fatalf("outdated status permission: %s", got)
			}
			if !strings.Contains(got, "Never use terminal, desktop or keyboard input") {
				t.Fatalf("missing launcher transport instruction: %s", got)
			}
			if plain := BuildWithOptions(agent, " task ", true); plain != "task" {
				t.Fatalf("--no-preamble added instructions: %s", plain)
			}
		})
	}
}
