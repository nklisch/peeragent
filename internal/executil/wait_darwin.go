package executil

import "syscall"

// EVFILT_PROC observes exit without reaping. Unlike Darwin waitid, it cannot
// mistake a stopped child for an exited child (Go issue 19314).
func waitWithoutReaping(pid int) error {
	queue, err := syscall.Kqueue()
	if err != nil {
		return err
	}
	defer syscall.Close(queue)
	changes := []syscall.Kevent_t{{Ident: uint64(pid), Filter: syscall.EVFILT_PROC, Flags: syscall.EV_ADD | syscall.EV_ONESHOT, Fflags: syscall.NOTE_EXIT}}
	var events [1]syscall.Kevent_t
	for {
		n, err := syscall.Kevent(queue, changes, events[:], nil)
		if err == syscall.EINTR {
			continue
		}
		// An already exited, unreaped child cannot be registered on some kernels.
		// Its number is still reserved until our explicit reap.
		if err == syscall.ESRCH {
			return nil
		}
		if err != nil {
			return err
		}
		changes = nil
		if n > 0 {
			event := events[0]
			if event.Flags&syscall.EV_ERROR != 0 {
				if syscall.Errno(event.Data) == syscall.ESRCH {
					return nil
				}
				return syscall.Errno(event.Data)
			}
			if event.Fflags&syscall.NOTE_EXIT != 0 {
				return nil
			}
		}
	}
}
