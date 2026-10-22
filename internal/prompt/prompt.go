package prompt

import "strings"

func Build(taskText string) string {
	return BuildForAgent("Codex", taskText)
}

const MessagingPreamble = "Your launching session may send you messages. Run `peeragent inbox` after each long step and before your handback.\n" +
	"To reach the launching session, use `peeragent say \"<text>\"`; never use terminal, desktop or keyboard input, and\n" +
	"keep messages for questions you can't settle and short status the launcher asked for."

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
