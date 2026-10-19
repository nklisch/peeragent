package jobs

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// The index contains only working directories, never prompts or credentials.
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
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".index-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(cwd); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
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
