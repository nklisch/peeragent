//go:build unix

package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WithJobLock serializes short result and state updates with O_EXCL on Unix.
func (s Store) WithJobLock(id string, fn func() error) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	lockPath := filepath.Join(dir, "lock")
	deadline := time.Now().Add(jobLockTimeout)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
				_ = file.Close()
				_ = os.Remove(lockPath)
				return err
			}
			if err := file.Close(); err != nil {
				_ = os.Remove(lockPath)
				return err
			}
			defer os.Remove(lockPath)
			return fn()
		}
		if !os.IsExist(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("acquire job lock %s: timed out after %s", id, jobLockTimeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
