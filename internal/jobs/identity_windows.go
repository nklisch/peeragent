//go:build windows

package jobs

import (
	"syscall"
)

const processQueryLimitedInformation = 0x1000

func ProcessIdentity(pid int) (uint64, error) {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(h)
	var created, exited, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &created, &exited, &kernel, &user); err != nil {
		return 0, err
	}
	return uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime), nil
}

func processMatchesIdentity(pid int, identity uint64) bool {
	if !processExists(pid) {
		return false
	}
	if identity == 0 {
		return true // legacy pid files have no process identity
	}
	current, err := ProcessIdentity(pid)
	return err != nil || current == identity
}
