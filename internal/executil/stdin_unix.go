//go:build !windows && !plan9

package executil

import (
	"errors"
	"syscall"
)

// Match os/exec: a target may close stdin before consuming the whole prompt.
func ignoreStdinCopyError(err error) bool { return errors.Is(err, syscall.EPIPE) }
