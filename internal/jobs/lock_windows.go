//go:build windows

package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

const waitAbandoned = 0x00000080

var (
	createMutex  = kernel32.NewProc("CreateMutexW")
	releaseMutex = kernel32.NewProc("ReleaseMutex")
)

// WithJobLock uses a Windows mutex so process death releases ownership. The
// lock file is only a diagnostic marker and may be overwritten after a crash.
func (s Store) WithJobLock(id string, fn func() error) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name, err := syscall.UTF16PtrFromString("Local\\peeragent-lock-" + id)
	if err != nil {
		return err
	}
	h, _, callErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return fmt.Errorf("create job mutex %s: %w", id, callErr)
	}
	defer syscall.CloseHandle(syscall.Handle(h))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	state, err := syscall.WaitForSingleObject(syscall.Handle(h), uint32(jobLockTimeout.Milliseconds()))
	if err != nil {
		return fmt.Errorf("wait for job mutex %s: %w", id, err)
	}
	if state != syscall.WAIT_OBJECT_0 && state != waitAbandoned {
		return fmt.Errorf("acquire job lock %s: timed out after %s", id, jobLockTimeout)
	}
	defer releaseMutex.Call(h)
	lockPath := filepath.Join(dir, "lock")
	if err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o600); err != nil {
		return err
	}
	defer os.Remove(lockPath)
	return fn()
}
