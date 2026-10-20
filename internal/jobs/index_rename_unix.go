//go:build unix

package jobs

import "os"

func renameIndex(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
