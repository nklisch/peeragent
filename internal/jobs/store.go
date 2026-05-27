package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Job struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	PID        int       `json:"pid,omitempty"`
	TaskText   string    `json:"task_text,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LogPath    string    `json:"log_path"`
	ResultPath string    `json:"result_path"`
}

type Store struct {
	Root string
}

func NewStore(cwd string) Store {
	return Store{Root: filepath.Join(cwd, ".peeragent", "jobs")}
}

func (s Store) Create(taskText string) (Job, error) {
	id, err := newID()
	if err != nil {
		return Job{}, err
	}
	now := time.Now().UTC()
	dir := s.jobDir(id)
	job := Job{
		ID:         id,
		Status:     "running",
		TaskText:   taskText,
		CreatedAt:  now,
		UpdatedAt:  now,
		LogPath:    filepath.Join(dir, "agent.log"),
		ResultPath: filepath.Join(dir, "result.json"),
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Job{}, err
	}
	if err := s.Save(job); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (s Store) Load(id string) (Job, error) {
	content, err := os.ReadFile(filepath.Join(s.jobDir(id), "job.json"))
	if err != nil {
		return Job{}, err
	}
	var job Job
	if err := json.Unmarshal(content, &job); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (s Store) Save(job Job) error {
	job.UpdatedAt = time.Now().UTC()
	content, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.jobDir(job.ID), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.jobDir(job.ID), "job.json"), append(content, '\n'), 0o644)
}

func (s Store) jobDir(id string) string {
	return filepath.Join(s.Root, id)
}

func newID() (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	return time.Now().UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(suffix[:]), nil
}
