package jobs

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestReplaceFileRetriesWindowsContention(t *testing.T) {
	for _, code := range []syscall.Errno{syscall.ERROR_ACCESS_DENIED, 32} {
		t.Run(code.Error(), func(t *testing.T) {
			calls, waits := 0, 0
			err := replaceFileWithRetry("source", "destination", func(oldpath, newpath string) error {
				if oldpath != "source" || newpath != "destination" {
					t.Fatalf("rename paths = %q, %q", oldpath, newpath)
				}
				calls++
				if calls <= 2 {
					return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: code}
				}
				return nil
			}, func(delay time.Duration) {
				if delay <= 0 {
					t.Fatal("retry must wait before trying again")
				}
				waits++
			})
			if err != nil || calls != 3 || waits != 2 {
				t.Fatalf("rename: %v, calls=%d waits=%d", err, calls, waits)
			}
		})
	}
}

func TestReplaceFileReturnsPermanentWindowsErrors(t *testing.T) {
	for _, code := range []syscall.Errno{syscall.ERROR_ACCESS_DENIED, syscall.ERROR_FILE_NOT_FOUND} {
		t.Run(code.Error(), func(t *testing.T) {
			want := &os.LinkError{Op: "rename", Err: code}
			calls := 0
			var totalDelay time.Duration
			err := replaceFileWithRetry("source", "destination", func(string, string) error {
				calls++
				return want
			}, func(delay time.Duration) { totalDelay += delay })
			if !errors.Is(err, want) {
				t.Fatalf("error = %v, want %v", err, want)
			}
			if code == syscall.ERROR_FILE_NOT_FOUND {
				if calls != 1 || totalDelay != 0 {
					t.Fatalf("unrelated error retried: calls=%d delay=%s", calls, totalDelay)
				}
			} else if calls != 10 || totalDelay > 500*time.Millisecond {
				t.Fatalf("retry budget: calls=%d delay=%s", calls, totalDelay)
			}
		})
	}
}

func TestLockBusyTreatsPendingDeletionAsHeld(t *testing.T) {
	for _, code := range []syscall.Errno{syscall.ERROR_FILE_EXISTS, syscall.ERROR_ACCESS_DENIED, 32} {
		if !lockBusy(&os.PathError{Op: "open", Path: "lock", Err: code}) {
			t.Fatalf("lockBusy(%v) = false, want retry", code)
		}
	}
	for _, code := range []syscall.Errno{syscall.ERROR_PATH_NOT_FOUND, syscall.ERROR_FILE_NOT_FOUND} {
		if lockBusy(&os.PathError{Op: "open", Path: "lock", Err: code}) {
			t.Fatalf("lockBusy(%v) = true, want immediate error", code)
		}
	}
}
