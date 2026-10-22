package executil

import (
	"syscall"
	"unsafe"
)

// WNOWAIT observes exit but leaves the leader unreaped, pinning the group id
// until ownedGroup.reap revokes all future group signals. No other waiter may
// reap this child in the meantime.
func waitWithoutReaping(pid int) error {
	var info [16]uint64 // Linux siginfo_t is 128 bytes; only exit readiness is needed.
	for {
		_, _, errno := syscall.Syscall6(syscall.SYS_WAITID, 1 /* P_PID */, uintptr(pid), uintptr(unsafe.Pointer(&info)), syscall.WEXITED|syscall.WNOWAIT, 0, 0)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			return errno
		}
		return nil
	}
}
