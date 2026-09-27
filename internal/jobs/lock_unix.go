//go:build unix

package jobs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// WithJobLock uses an OS file lock that is released when its holder exits.
// The file remains in place so waiters and future callers lock the same inode.
func (s Store) WithJobLock(id string, fn func() error) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	lockPath := filepath.Join(dir, "lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	deadline := time.Now().Add(jobLockTimeout)
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
			if err := file.Truncate(0); err != nil {
				return err
			}
			if _, err := file.Seek(0, 0); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
				return err
			}
			return fn()
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("acquire job lock %s: timed out after %s", id, jobLockTimeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
