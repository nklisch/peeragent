package jobs

import (
	"errors"
	"os"
	"syscall"
	"time"
)

func renameIndex(oldpath, newpath string) error {
	return renameIndexWithRetry(oldpath, newpath, os.Rename, time.Sleep)
}

func renameIndexWithRetry(oldpath, newpath string, rename func(string, string) error, sleep func(time.Duration)) error {
	// Windows can briefly deny replacement while another writer or reader has
	// the destination open. Retry only these errors, for at most 450ms of delay;
	// permanent permission errors still return to the caller as index warnings.
	const sharingViolation syscall.Errno = 32 // ERROR_SHARING_VIOLATION
	for attempt := 0; ; attempt++ {
		err := rename(oldpath, newpath)
		if attempt == 9 || (!errors.Is(err, syscall.ERROR_ACCESS_DENIED) && !errors.Is(err, sharingViolation)) {
			return err
		}
		sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
}
