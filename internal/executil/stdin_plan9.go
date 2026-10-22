package executil

import (
	"errors"
	"io/fs"
)

func ignoreStdinCopyError(err error) bool {
	var pathErr *fs.PathError
	return errors.As(err, &pathErr) && pathErr.Op == "write" && pathErr.Err.Error() == "i/o on hungup channel"
}
