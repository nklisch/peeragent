//go:build !windows

package jobs

import "os"

func ReadJobFile(path string) ([]byte, error) { return os.ReadFile(path) }

func renameJobFile(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }
