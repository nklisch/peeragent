//go:build windows

package jobs

import (
	"errors"
	"os"
	"syscall"
	"time"
)

const (
	fileIOGrace = 1500 * time.Millisecond
	fileIORetry = 20 * time.Millisecond
)

// ReadJobFile tolerates short Windows sharing locks while another process
// replaces a job sidecar or a scanner briefly holds it open.
func ReadJobFile(path string) ([]byte, error) {
	var content []byte
	err := retryJobFileIO(func() error {
		var err error
		content, err = os.ReadFile(path)
		return err
	})
	return content, err
}

func renameJobFile(oldPath, newPath string) error {
	return retryJobFileIO(func() error { return os.Rename(oldPath, newPath) })
}

func retryJobFileIO(operation func() error) error {
	deadline := time.Now().Add(fileIOGrace)
	for {
		err := operation()
		if err == nil || !retryableJobFileError(err) || !time.Now().Before(deadline) {
			return err
		}
		time.Sleep(fileIORetry)
	}
}

func retryableJobFileError(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED) ||
		errors.Is(err, syscall.Errno(32)) || // ERROR_SHARING_VIOLATION
		errors.Is(err, syscall.Errno(33)) // ERROR_LOCK_VIOLATION
}
