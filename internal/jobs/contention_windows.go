package jobs

import (
	"errors"
	"os"
	"syscall"
	"time"
)

const errorSharingViolation syscall.Errno = 32 // ERROR_SHARING_VIOLATION

// windowsContention reports the errors Windows returns while another process
// briefly holds a file: a reader without delete sharing, or a just-removed file
// still pending deletion.
func windowsContention(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED) || errors.Is(err, errorSharingViolation)
}

func replaceFile(oldpath, newpath string) error {
	return replaceFileWithRetry(oldpath, newpath, os.Rename, time.Sleep)
}

func replaceFileWithRetry(oldpath, newpath string, rename func(string, string) error, sleep func(time.Duration)) error {
	// Windows can briefly deny replacement while another writer or reader has
	// the destination open. Retry only these errors, for at most 450ms of delay;
	// permanent permission errors still return to the caller.
	for attempt := 0; ; attempt++ {
		err := rename(oldpath, newpath)
		if attempt == 9 || !windowsContention(err) {
			return err
		}
		sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
}

// lockBusy treats a lock file that is still pending deletion after its holder
// released it as held: creating it again fails with access denied until the
// last handle closes. withFileLock's deadline bounds a permanent denial.
func lockBusy(err error) bool {
	return os.IsExist(err) || windowsContention(err)
}
