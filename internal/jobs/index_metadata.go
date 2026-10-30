package jobs

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxIndexedJobs = 10000

const IndexedOnlyWarning = "Indexed metadata only; local job files and logs are unavailable."

// IndexedJob is enough to find and resume a conversation after its worktree is
// removed. Prompts, messages, mailboxes, and raw logs stay in their existing stores.
type IndexedJob struct {
	Worker        *ProcessIdentity `json:"worker,omitempty"`
	ID            string           `json:"id"`
	Agent         string           `json:"agent"`
	Model         string           `json:"model"`
	Effort        string           `json:"effort"`
	CWD           string           `json:"cwd"`
	Status        Status           `json:"status"`
	AgentSession  string           `json:"agent_session"`
	Summary       string           `json:"summary,omitempty"`
	FailureReason string           `json:"failure_reason,omitempty"`
	Failure       string           `json:"failure,omitempty"`
	ExitCode      int              `json:"exit_code"`
	StopReason    string           `json:"stop_reason,omitempty"`
	StopMessage   string           `json:"stop_message,omitempty"`
	StartedAt     time.Time        `json:"started_at"`
	FinishedAt    *time.Time       `json:"finished_at,omitempty"`
}

func JobIndexEntry(job Job) IndexedJob {
	entry := IndexedJob{Worker: job.Worker, ID: job.ID, Agent: job.Spec.Agent, Model: job.Spec.Model,
		Effort: job.Spec.Effort, CWD: job.CWD, Status: job.Status,
		AgentSession: job.Session(), StartedAt: job.CreatedAt}
	if IsTerminalStatus(job.Status) {
		entry.FinishedAt = &job.UpdatedAt
	}
	return entry
}

// WriteIndexedJob only writes one snapshot. Launch/completion and legacy import
// prune once per operation; live session observations never scan the whole index.
func WriteIndexedJob(entry IndexedJob) (IndexedJob, error) {
	if err := ValidateID(entry.ID); err != nil {
		return entry, err
	}
	var err error
	entry.CWD, err = filepath.Abs(entry.CWD)
	if err != nil {
		return entry, err
	}
	if len(entry.Summary) > 600 {
		text := []rune(entry.Summary)
		if len(text) > 600 {
			entry.Summary = string(text[:599]) + "…"
		}
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return entry, err
	}
	path, err := indexPath("jobs", entry.ID)
	if err != nil {
		return entry, err
	}
	return entry, writeIndexFile(path, data)
}

func ReadIndexedJob(id string) (IndexedJob, bool, error) {
	if err := ValidateID(id); err != nil {
		return IndexedJob{}, false, err
	}
	path, err := indexPath("jobs", id)
	if err != nil {
		return IndexedJob{}, false, err
	}
	entry, err := readJobIndexFile(path)
	if err != nil {
		return entry, false, err
	}
	return entry, entry.ID != "" && entry.ID == id, nil
}

func readJobIndexFile(path string) (IndexedJob, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return IndexedJob{}, nil
	}
	if err != nil {
		return IndexedJob{}, fmt.Errorf("read job index entry: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return IndexedJob{}, err
	}
	if info.ModTime().Before(time.Now().Add(-resultRetention)) {
		return IndexedJob{}, nil
	}
	// The reader accepts old directory-only entries as well as current JSON.
	data, err := io.ReadAll(file)
	if err != nil {
		return IndexedJob{}, fmt.Errorf("read job index entry: %w", err)
	}
	return decodeJobIndex(data)
}

func IndexedJobs() ([]IndexedJob, error) {
	path, err := indexPath("jobs", "")
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list job index: %w", err)
	}
	var entries []IndexedJob
	for _, file := range files {
		if file.IsDir() || !indexFileName(file.Name()) {
			continue
		}
		entry, err := readJobIndexFile(filepath.Join(filepath.Dir(path), file.Name()))
		if err != nil {
			return nil, err
		}
		if entry.CWD != "" {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func indexFileName(name string) bool {
	if len(name) != 64 || strings.ToLower(name) != name {
		return false
	}
	_, err := hex.DecodeString(name)
	return err == nil
}

func PruneJobIndex() error {
	path, err := indexPath("jobs", "")
	if err != nil {
		return err
	}
	return pruneJobIndex(filepath.Dir(path), time.Now().Add(-resultRetention), maxIndexedJobs)
}

func pruneJobIndex(dir string, before time.Time, limit int) error {
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var entries []os.FileInfo
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
		entries = append(entries, info)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ModTime().Equal(entries[j].ModTime()) {
			return entries[i].Name() < entries[j].Name()
		}
		return entries[i].ModTime().After(entries[j].ModTime())
	})
	for i, entry := range entries {
		if i < limit && !entry.ModTime().Before(before) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("prune job index entry: %w", err)
		}
	}
	return nil
}
