package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	StatusRunning      Status = "running"
	StatusComplete     Status = "complete"
	StatusFailed       Status = "failed"
	StatusCancelled    Status = "cancelled"
	StatusAuthRequired Status = "auth_required"
	StatusInterrupted  Status = "interrupted"
	StatusCapacity     Status = "capacity_exhausted"
	StatusUsageLimited Status = "usage_limited"
	StatusOutputLimit  Status = "output_limit"
)

type InboxSource struct {
	JobID string `json:"job_id"`
	CWD   string `json:"cwd"`
}

type Job struct {
	Worker               *ProcessIdentity `json:"worker,omitempty"`
	IndexedOnly          bool             `json:"-"`
	IndexWarning         string           `json:"index_warning,omitempty"`
	messagingRecorded    bool
	MessagingEnabled     bool          `json:"messaging_enabled"`
	AgentSession         string        `json:"agent_session,omitempty"`
	ResumeWarning        string        `json:"resume_warning,omitempty"`
	InboxSources         []InboxSource `json:"inbox_sources,omitempty"`
	Branch               string        `json:"branch,omitempty"`
	MessagePath          string        `json:"message_path,omitempty"`
	FinalReceived        bool          `json:"final_received,omitempty"`
	CapacityBackoffUntil *time.Time    `json:"capacity_backoff_until,omitempty"`
	CapacityRetries      int           `json:"capacity_retries,omitempty"`
	ID                   string        `json:"id"`
	Status               Status        `json:"status"`
	CWD                  string        `json:"cwd"`
	Spec                 ExecSpec      `json:"spec"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
	LogPath              string        `json:"log_path"`
	ResultPath           string        `json:"result_path"`
	PromptPath           string        `json:"prompt_path"`
}

// A pointer distinguishes an absent legacy marker from an explicitly disabled
// preamble. Preserve absence when older job metadata is saved for other reasons.
type jobRecord Job
type storedJob struct {
	*jobRecord
	MessagingEnabled *bool `json:"messaging_enabled,omitempty"`
}

type ExecSpec struct {
	NoPreamble      bool          `json:"no_preamble,omitempty"`
	KeepTerminalEnv bool          `json:"keep_terminal_env,omitempty"`
	RetryCapacity   int           `json:"retry_capacity,omitempty"`
	CapacityBackoff time.Duration `json:"capacity_backoff,omitempty"`
	Agent           string        `json:"agent"`
	Access          string        `json:"access"`
	Profile         string        `json:"profile,omitempty"`
	Effort          string        `json:"effort,omitempty"`
	Model           string        `json:"model,omitempty"`
	Resume          string        `json:"resume,omitempty"`
	JSON            bool          `json:"json"`
	FullAccess      bool          `json:"full_access,omitempty"`
	Worktree        bool          `json:"worktree,omitempty"`
}

type Store struct {
	Root       string
	legacyRoot string
}

const jobLockTimeout = 5 * time.Second

func NewStore(cwd string) Store {
	legacy := filepath.Join(cwd, ".peeragent", "jobs")
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--absolute-git-dir").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return Store{Root: legacy}
	}
	return Store{Root: filepath.Join(strings.TrimSpace(string(out)), "peeragent", "jobs"), legacyRoot: legacy}
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
		messagingRecorded: true,
		MessagingEnabled:  !spec.NoPreamble,
		AgentSession:      spec.Resume,
		ID:                id,
		Status:            StatusRunning,
		CWD:               cwd,
		Spec:              spec,
		CreatedAt:         now,
		UpdatedAt:         now,
		LogPath:           filepath.Join(dir, "agent.log"),
		ResultPath:        filepath.Join(dir, "result.json"),
		PromptPath:        filepath.Join(dir, "prompt.txt"),
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
	record := storedJob{jobRecord: (*jobRecord)(&job)}
	if err := json.Unmarshal(content, &record); err != nil {
		return Job{}, err
	}
	job.messagingRecorded = record.MessagingEnabled != nil
	if job.messagingRecorded {
		job.MessagingEnabled = *record.MessagingEnabled
	}
	return job, nil
}

func (s Store) Save(job Job) error {
	dir, err := s.jobDir(job.ID)
	if err != nil {
		return err
	}
	job.UpdatedAt = time.Now().UTC()
	record := storedJob{jobRecord: (*jobRecord)(&job)}
	if job.messagingRecorded || job.MessagingEnabled {
		record.MessagingEnabled = &job.MessagingEnabled
	}
	content, err := json.MarshalIndent(record, "", "  ")
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
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return s.WithJobLock(id, func() error {
		job, err := s.Load(id)
		if err != nil {
			return err
		}
		identity := IdentifyProcess(pid)
		job.Worker = &identity
		if job.Status == StatusRunning {
			if _, err := WriteIndexedJob(JobIndexEntry(job)); err != nil {
				job.IndexWarning = "index worker identity: " + err.Error()
			}
		}
		if err := s.Save(job); err != nil {
			return err
		}
		return AtomicWriteFile(filepath.Join(dir, "pid"), []byte(strconv.Itoa(pid)+"\n"), 0o644)
	})
}

func (s Store) ReadPID(id string) (int, error) {
	dir, err := s.jobDir(id)
	if err != nil {
		return 0, err
	}
	content, err := os.ReadFile(filepath.Join(dir, "pid"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(content)))
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

// WithJobLock serializes short updates that keep job.json and result.json
// consistent. A kernel lock releases on death; the removable owner file also
// coordinates with older binaries and diagnoses a live holder.
func (s Store) WithJobLock(id string, fn func() error) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return withFileLock(filepath.Join(dir, "lock"), fn)
}

func withFileLock(lockPath string, fn func() error) error {
	// The guard inode stays in place: unlinking a kernel-locked inode would
	// let a second opener lock a different inode and enter the same critical section.
	guard, err := os.OpenFile(lockPath+".guard", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer guard.Close()
	deadline := time.Now().Add(jobLockTimeout)
	for {
		acquired, err := tryLockFile(guard)
		if err != nil {
			return err
		}
		if acquired {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("acquire job lock %s: timed out after %s", lockPath, jobLockTimeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			owner, _ := json.Marshal(IdentifyProcess(os.Getpid()))
			if _, err := file.Write(append(owner, '\n')); err != nil {
				_ = file.Close()
				_ = os.Remove(lockPath)
				return err
			}
			if err := file.Close(); err != nil {
				_ = os.Remove(lockPath)
				return err
			}
			defer os.Remove(lockPath)
			return fn()
		}
		if !lockBusy(err) {
			return err
		}
		// Only a proven dead owner is recoverable. The kernel guard serializes
		// stale-file removal so two recoverers cannot unlink a new holder.
		data, readErr := os.ReadFile(lockPath)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return readErr
		}
		var owner ProcessIdentity
		if json.Unmarshal(data, &owner) != nil {
			owner.PID, _ = strconv.Atoi(strings.TrimSpace(string(data)))
		}
		if owner.Gone() {
			if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("acquire job lock %s: timed out after %s (holder pid %d); remove the stale lock file manually if its identity cannot be checked", lockPath, jobLockTimeout, owner.PID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s Store) jobDir(id string) (string, error) {
	if err := ValidateID(id); err != nil {
		return "", err
	}
	primary := filepath.Join(s.Root, id)
	if s.legacyRoot == "" {
		return primary, nil
	}
	// Keep all reads and writes for an existing job in its original folder,
	// including locks and mailboxes used by workers from an older binary.
	if _, err := os.Stat(primary); !os.IsNotExist(err) {
		return primary, err
	}
	legacy := filepath.Join(s.legacyRoot, id)
	if _, err := os.Stat(legacy); err == nil {
		return legacy, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return primary, nil
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
	if err := replaceFile(tmp, path); err != nil {
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
	case StatusComplete, StatusFailed, StatusCancelled, StatusCapacity, StatusUsageLimited, StatusOutputLimit, StatusAuthRequired, StatusInterrupted:
		return true
	default:
		return false
	}
}

// Session retains compatibility with jobs written before agent_session was explicit.
func (j Job) Session() string {
	if j.AgentSession != "" {
		return j.AgentSession
	}
	return j.Spec.Resume
}

const LegacyMessagingWarning = "this job started before messaging; it will see replies only on --resume"

func (j Job) MessagingWarning() string {
	if j.MessagingEnabled {
		return ""
	}
	if j.Spec.NoPreamble {
		return "this job started without a messaging preamble; it will see replies only on --resume"
	}
	return LegacyMessagingWarning
}
