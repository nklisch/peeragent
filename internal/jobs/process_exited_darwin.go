package jobs

import "syscall"

// A zero-time native exit observation also recognizes exited, unreaped workers.
// Closing the queue releases the observation without collecting another owner's
// child. Unsupported/denied observations fall back to the signal probe.
func processExited(pid int) bool {
	queue, err := syscall.Kqueue()
	if err != nil {
		return false
	}
	defer syscall.Close(queue)
	changes := []syscall.Kevent_t{{Ident: uint64(pid), Filter: syscall.EVFILT_PROC, Flags: syscall.EV_ADD | syscall.EV_ONESHOT, Fflags: syscall.NOTE_EXIT}}
	var events [1]syscall.Kevent_t
	zero := syscall.Timespec{}
	for {
		n, err := syscall.Kevent(queue, changes, events[:], &zero)
		if err == syscall.EINTR {
			continue
		}
		if err == syscall.ESRCH {
			return true
		}
		if err != nil || n == 0 {
			return false
		}
		event := events[0]
		if event.Flags&syscall.EV_ERROR != 0 {
			return syscall.Errno(event.Data) == syscall.ESRCH
		}
		return event.Fflags&syscall.NOTE_EXIT != 0
	}
}
