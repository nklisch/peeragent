package result

func CapacitySummary(session string) string {
	summary := "stopped: " + string(StatusCapacity)
	if session != "" {
		summary += "; resumable with --resume " + session
	}
	return summary + "; use --retry-capacity N to retry automatically"
}

func OutputLimitSummary(session string) string {
	summary := "stopped: Codex max_output_tokens limit"
	if session != "" {
		summary += "; resumable with peeragent --agent codex --resume " + session + " \"Continue.\""
	}
	return summary
}
