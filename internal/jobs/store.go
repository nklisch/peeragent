package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Status is the persisted lifecycle vocabulary for an asynchronous job.
// Unknown values may still be loaded from disk so older/newer writers remain
// readable; callers must use IsTerminalStatus for terminal membership.
type Status string

const (
	StatusRunning   Status = "running"
	StatusComplete  Status = "complete"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Job struct {
	ID         string    `json:"id"`
	Status     Status    `json:"status"`
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

// PIDRecord includes the worker's process creation identity on Windows. A PID
// alone may later refer to an unrelated process after the worker exits.
type PIDRecord struct {
	PID      int
	Identity uint64
}

const jobLockTimeout = 5 * time.Second

func NewStore(cwd string) Store {
	return Store{Root: filepath.Join(cwd, ".peeragent", "jobs")}
}

func (s Store) Create(cwd string, spec ExecSpec, prompt string) (Job, error) {
	id, err := newID()
	if err != nil {
		return Job{}, err
	}
	now := time.Now().UTC()
	dir, err := s.jobDir(id)
	if err != nil {
		return Job{}, err
	}
	job := Job{
		ID:         id,
		Status:     StatusRunning,
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
	dir, err := s.jobDir(id)
	if err != nil {
		return Job{}, err
	}
	content, err := os.ReadFile(filepath.Join(dir, "job.json"))
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
	dir, err := s.jobDir(job.ID)
	if err != nil {
		return err
	}
	job.UpdatedAt = time.Now().UTC()
	content, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(dir, "job.json"), append(content, '\n'), 0o644)
}

func (s Store) SaveGuarded(job Job) (Status, error) {
	prior, err := s.Load(job.ID)
	if err != nil {
		return Status(""), err
	}
	if IsTerminalStatus(prior.Status) && prior.Status != job.Status {
		return prior.Status, nil
	}
	return prior.Status, s.Save(job)
}

func (s Store) WritePrompt(id string, prompt string) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(dir, "prompt.txt"), []byte(prompt), 0o644)
}

func (s Store) ReadPrompt(id string) (string, error) {
	dir, err := s.jobDir(id)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.Join(dir, "prompt.txt"))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (s Store) WritePID(id string, pid int) error {
	return s.writePIDContent(id, strconv.Itoa(pid)+"\n")
}

func (s Store) WritePIDRecord(id string, worker PIDRecord) error {
	return s.writePIDContent(id, fmt.Sprintf("%d\n%d\n", worker.PID, worker.Identity))
}

func (s Store) writePIDContent(id, content string) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(dir, "pid"), []byte(content), 0o644)
}

func (s Store) ReadPID(id string) (int, error) {
	worker, err := s.ReadPIDRecord(id)
	return worker.PID, err
}

func (s Store) ReadPIDRecord(id string) (PIDRecord, error) {
	dir, err := s.jobDir(id)
	if err != nil {
		return PIDRecord{}, err
	}
	content, err := os.ReadFile(filepath.Join(dir, "pid"))
	if err != nil {
		return PIDRecord{}, err
	}
	lines := strings.Fields(string(content))
	if len(lines) < 1 || len(lines) > 2 {
		return PIDRecord{}, fmt.Errorf("invalid worker pid record")
	}
	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return PIDRecord{}, err
	}
	worker := PIDRecord{PID: pid}
	if len(lines) == 2 {
		worker.Identity, err = strconv.ParseUint(lines[1], 10, 64)
	}
	return worker, err
}

func (s Store) RemovePID(id string) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, "pid"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Store) jobDir(id string) (string, error) {
	if err := ValidateID(id); err != nil {
		return "", err
	}
	return filepath.Join(s.Root, id), nil
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

// IsTerminalStatus is the authoritative persisted terminal-state membership
// check. Unknown values intentionally remain non-terminal so a newer writer
// cannot be accidentally overwritten by an older process.
func IsTerminalStatus(status Status) bool {
	switch status {
	case StatusComplete, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}
