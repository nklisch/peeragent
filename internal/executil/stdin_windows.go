package executil

import (
	"errors"
	"syscall"
)

func ignoreStdinCopyError(err error) bool {
	const errorNoData = syscall.Errno(232)
	return errors.Is(err, syscall.ERROR_BROKEN_PIPE) || errors.Is(err, errorNoData)
}
