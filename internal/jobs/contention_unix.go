//go:build unix

package jobs

import "os"

func replaceFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func lockBusy(err error) bool {
	return os.IsExist(err)
}
