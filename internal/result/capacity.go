package result

func UsageLimitSummary(message string) string {
	summary := "stopped: usage_limited"
	if message != "" {
		summary += "; " + message
	}
	return summary + "; not retried by --retry-capacity; usage limits clear at the reset or on an account switch"
}

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

// RecoverySummary keeps resumable non-capacity stops consistent across result,
// single-job status and multi-job status responses.
func RecoverySummary(status Status, session string) string {
	summary := "stopped: " + string(status)
	if status == StatusAuthRequired {
		summary += "; Codex login needs refreshing; sign in again if needed"
	}
	if status == StatusInterrupted {
		summary += "; Codex could not record its rollout"
	}
	if session != "" {
		summary += "; resumable with --resume " + session
	}
	return summary
}
