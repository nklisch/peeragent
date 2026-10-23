package prompt

import "strings"

func Build(taskText string) string {
	return BuildForAgent("Codex", taskText)
}

// Keep this introduction stable so older saved prompts remain evidence of
// messaging support when the instructions that follow it change.
const MessagingPreambleIntro = "Your launching session may send you messages. Run `peeragent inbox` after each long step and before your handback."

const MessagingPreamble = MessagingPreambleIntro + "\n" +
	"Use `peeragent say` only to ask a question you can't settle yourself, never for status or progress updates.\n" +
	"Your final handback carries the final status, so don't repeat it in a message.\n" +
	"After asking a question, watch your inbox for the answer: run `peeragent inbox` every few minutes, between steps,\n" +
	"until it arrives, and keep doing independent work meanwhile.\n" +
	"Never use terminal, desktop or keyboard input to contact the launching session."

func BuildForAgent(agentName string, taskText string) string {
	return BuildWithOptions(agentName, taskText, false)
}

func BuildWithOptions(agentName string, taskText string, noPreamble bool) string {
	if noPreamble {
		return strings.TrimSpace(taskText)
	}
	taskText = strings.TrimSpace(taskText)
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		agentName = "the selected agent"
	}
	return strings.TrimSpace(`You are ` + agentName + ` running as an autonomous implementation agent for another coding assistant.

Work in the current repository. Make the requested code changes directly. Follow project instructions and existing code patterns. Run relevant verification commands when practical. Keep the final response concise and include changed files, verification status, and any blockers.

If credentials, network access, permissions, or missing context block the work, stop and report the blocker instead of guessing or silently skipping required work.

` + MessagingPreamble + `

Implementation task:
` + taskText)
}
