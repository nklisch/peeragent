//go:build unix && !linux && !darwin

package jobs

func processExited(pid int) bool { return false }
