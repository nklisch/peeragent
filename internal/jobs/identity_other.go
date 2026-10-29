//go:build !linux && !darwin && !windows

package jobs

func bootID() string                  { return "" }
func processStartTime(pid int) string { return "" }
