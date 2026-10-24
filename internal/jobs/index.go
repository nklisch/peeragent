package jobs

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Directory entries contain only working directories; terminal result snapshots
// are stored separately and never copy prompt files or raw logs.
// Separate files avoid lost updates between simultaneous launches.

func indexPath(kind, key string) (string, error) {
	root := os.Getenv("PEERAGENT_STATE_DIR")
	if root == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(base, "peeragent")
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	return filepath.Join(root, "index", kind, hash), nil
}

func writeIndex(kind, key, cwd string) error {
	path, err := indexPath(kind, key)
	if err != nil {
		return err
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return err
	}
	return writeIndexFile(path, []byte(cwd))
}

func writeIndexFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".index-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return replaceFile(file.Name(), path)
}

func readIndex(kind, key string) (string, error) {
	path, err := indexPath(kind, key)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func IndexJob(id, cwd string) error { return writeIndex("jobs", id, cwd) }

func LookupJob(id string) (string, error) { return readIndex("jobs", id) }

func IndexSession(agent, session, cwd string) error {
	return writeIndex("sessions", agent+"\x00"+session, cwd)
}

func LookupSession(agent, session string) (string, error) {
	return readIndex("sessions", agent+"\x00"+session)
}

// IndexedDirectories returns distinct job directories without depending on the
// hashed index filenames. Removed worktrees remain harmless stale entries.
func IndexedDirectories() ([]string, error) {
	path, err := indexPath("jobs", "")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list job index: %w", err)
	}
	seen := map[string]bool{}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".index-") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(filepath.Dir(path), entry.Name()))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read job index entry: %w", err)
		}
		cwd := string(data)
		if cwd != "" && !seen[cwd] {
			seen[cwd] = true
			dirs = append(dirs, cwd)
		}
	}
	return dirs, nil
}

func (s Store) ListIDs() ([]string, error) {
	entries, err := os.ReadDir(s.Root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() && ValidateID(entry.Name()) == nil {
			ids = append(ids, entry.Name())
		}
	}
	return ids, nil
}
