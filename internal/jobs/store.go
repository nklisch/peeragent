package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Job struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	CWD        string    `json:"cwd"`
	Spec       ExecSpec  `json:"spec"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LogPath    string    `json:"log_path"`
	ResultPath string    `json:"result_path"`
	PromptPath string    `json:"prompt_path"`
}

type ExecSpec struct {
	Agent      string `json:"agent"`
	Access     string `json:"access"`
	Profile    string `json:"profile,omitempty"`
	Effort     string `json:"effort,omitempty"`
	Model      string `json:"model,omitempty"`
	Resume     string `json:"resume,omitempty"`
	JSON       bool   `json:"json"`
	FullAccess bool   `json:"full_access,omitempty"`
	Worktree   bool   `json:"worktree,omitempty"`
}

type Store struct {
	Root string
}

func NewStore(cwd string) Store {
	return Store{Root: filepath.Join(cwd, ".peeragent", "jobs")}
}

func (s Store) Create(cwd string, spec ExecSpec, prompt string) (Job, error) {
	id, err := newID()
	if err != nil {
		return Job{}, err
	}
	now := time.Now().UTC()
	dir := s.jobDir(id)
	job := Job{
		ID:         id,
		Status:     "running",
		CWD:        cwd,
		Spec:       spec,
		CreatedAt:  now,
		UpdatedAt:  now,
		LogPath:    filepath.Join(dir, "agent.log"),
		ResultPath: filepath.Join(dir, "result.json"),
		PromptPath: filepath.Join(dir, "prompt.txt"),
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Job{}, err
	}
	if err := s.Save(job); err != nil {
		return Job{}, err
	}
	if err := s.WritePrompt(job.ID, prompt); err != nil {
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
	return AtomicWriteFile(filepath.Join(s.jobDir(job.ID), "job.json"), append(content, '\n'), 0o644)
}

func (s Store) SaveGuarded(job Job) (string, error) {
	prior, err := s.Load(job.ID)
	if err != nil {
		return "", err
	}
	if isTerminalStatus(prior.Status) && prior.Status != job.Status {
		return prior.Status, nil
	}
	return prior.Status, s.Save(job)
}

func (s Store) WritePrompt(id string, prompt string) error {
	if err := os.MkdirAll(s.jobDir(id), 0o755); err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(s.jobDir(id), "prompt.txt"), []byte(prompt), 0o644)
}

func (s Store) ReadPrompt(id string) (string, error) {
	content, err := os.ReadFile(filepath.Join(s.jobDir(id), "prompt.txt"))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (s Store) WritePID(id string, pid int) error {
	if err := os.MkdirAll(s.jobDir(id), 0o755); err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(s.jobDir(id), "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func (s Store) ReadPID(id string) (int, error) {
	content, err := os.ReadFile(filepath.Join(s.jobDir(id), "pid"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(content)))
}

func (s Store) RemovePID(id string) error {
	err := os.Remove(filepath.Join(s.jobDir(id), "pid"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
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

func AtomicWriteFile(path string, content []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, perm); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func isTerminalStatus(status string) bool {
	switch status {
	case "complete", "failed", "cancelled":
		return true
	default:
		return false
	}
}
