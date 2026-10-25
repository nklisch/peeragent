package jobs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const resultRetention = 30 * 24 * time.Hour

// VisitUnindexedArchives promotes legacy snapshots only when no compact job
// metadata exists. Read at most the newest metadata-retention budget, one
// archive at a time, so listing does not repeatedly load full handbacks.
func VisitUnindexedArchives(known []IndexedJob, visit func([]byte, time.Time) error) error {
	path, err := indexPath("results", "")
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, job := range known {
		if job.ID != "" && IsTerminalStatus(job.Status) {
			key, err := indexPath("results", job.ID)
			if err != nil {
				return err
			}
			seen[filepath.Base(key)] = true
		}
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var candidates []os.FileInfo
	before := time.Now().Add(-resultRetention)
	for _, file := range files {
		if file.IsDir() || !indexFileName(file.Name()) {
			continue
		}
		info, err := file.Info()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.ModTime().Before(before) {
			candidates = append(candidates, info)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ModTime().Equal(candidates[j].ModTime()) {
			return candidates[i].Name() < candidates[j].Name()
		}
		return candidates[i].ModTime().After(candidates[j].ModTime())
	})
	for i, file := range candidates {
		if i >= maxIndexedJobs {
			break
		}
		if seen[file.Name()] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(filepath.Dir(path), file.Name()))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := visit(data, file.ModTime()); err != nil {
			return err
		}
	}
	return nil
}

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
		if entry.IsDir() || !indexFileName(entry.Name()) {
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
