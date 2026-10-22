package result

func CapacitySummary(session string) string {
	summary := "stopped: model at capacity"
	if session != "" {
		summary += "; resumable with --resume " + session
	}
	return summary
}
