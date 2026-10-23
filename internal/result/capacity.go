package result

func CapacitySummary(session string) string {
	summary := "stopped: model at capacity"
	if session != "" {
		summary += "; resumable with --resume " + session
	}
	return summary
}

func OutputLimitSummary(session string) string {
	summary := "stopped: Codex max_output_tokens limit"
	if session != "" {
		summary += "; resumable with peeragent --agent codex --resume " + session + " \"Continue.\""
	}
	return summary
}
