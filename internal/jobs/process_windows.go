package jobs

import "syscall"

func ProcessExists(pid int) bool {
	if pid <= 1 {
		return false
	}
	// SYNCHRONIZE permits a zero-time wait without terminating the process.
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err == syscall.ERROR_ACCESS_DENIED {
		return true
	}
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)
	status, err := syscall.WaitForSingleObject(handle, 0)
	return err == nil && status == uint32(syscall.WAIT_TIMEOUT)
}
