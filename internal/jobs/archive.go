package jobs

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const resultRetention = 30 * 24 * time.Hour

// ArchiveResult keeps just the terminal result JSON outside the worktree.
// The application owns terminal membership and encoding, avoiding a dependency
// from job storage back to the public result package.
func ArchiveResult(id string, content []byte) error {
	if err := ValidateID(id); err != nil {
		return err
	}
	path, err := indexPath("results", id)
	if err != nil {
		return err
	}
	if err := writeIndexFile(path, content); err != nil {
		return err
	}
	return pruneArchivedResults(filepath.Dir(path), time.Now().Add(-resultRetention))
}

func ReadArchivedResult(id string) ([]byte, error) {
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	path, err := indexPath("results", id)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.ModTime().Before(time.Now().Add(-resultRetention)) {
		return nil, nil
	}
	return io.ReadAll(file)
}

// Only the hash-named snapshots this package owns are eligible for pruning.
// Temporary writes are ignored, so concurrent completions need no global lock.
func pruneArchivedResults(dir string, before time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || len(entry.Name()) != 64 || strings.ToLower(entry.Name()) != entry.Name() {
			continue
		}
		if _, err := hex.DecodeString(entry.Name()); err != nil {
			continue
		}
		info, err := entry.Info()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect archived result: %w", err)
		}
		if info.ModTime().Before(before) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("prune archived result: %w", err)
			}
		}
	}
	return nil
}
