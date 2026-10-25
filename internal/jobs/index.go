package jobs

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Separate per-job files avoid lost updates between simultaneous launches.
// Job entries may be legacy directory strings or compact JSON metadata.

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

func LookupJob(id string) (string, error) {
	path, err := indexPath("jobs", id)
	if err != nil {
		return "", err
	}
	entry, err := readJobIndexFile(path)
	return entry.CWD, err
}

func decodeJobIndex(data []byte) (IndexedJob, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return IndexedJob{CWD: string(data)}, nil
	}
	var entry IndexedJob
	if err := json.Unmarshal(data, &entry); err != nil {
		return entry, fmt.Errorf("decode job index entry: %w", err)
	}
	if err := ValidateID(entry.ID); err != nil {
		return entry, fmt.Errorf("decode job index entry: %w", err)
	}
	return entry, nil
}

func IndexSession(agent, session, cwd string) error {
	return writeIndex("sessions", agent+"\x00"+session, cwd)
}

func LookupSession(agent, session string) (string, error) {
	cwd, err := readIndex("sessions", agent+"\x00"+session)
	if err != nil || cwd != "" {
		return cwd, err
	}
	entries, err := IndexedJobs()
	if err != nil {
		return "", err
	}
	var latest IndexedJob
	for _, entry := range entries {
		if entry.ID != "" && entry.Agent == agent && entry.AgentSession == session &&
			(latest.ID == "" || entry.StartedAt.After(latest.StartedAt) || entry.StartedAt.Equal(latest.StartedAt) && entry.ID > latest.ID) {
			latest = entry
		}
	}
	return latest.CWD, nil
}

// IndexedDirectories returns distinct job directories without depending on the
// hashed index filenames, including legacy path-only entries.
func IndexedDirectories() ([]string, error) {
	entries, err := IndexedJobs()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var dirs []string
	for _, entry := range entries {
		cwd := entry.CWD
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
