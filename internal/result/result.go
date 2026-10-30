package result

import (
	"time"

	"github.com/nklisch/peeragent/internal/heavy"
	"github.com/nklisch/peeragent/internal/jobs"
)

type Status string

const (
	StatusNotFound     Status = "not_found"
	StatusAuthRequired Status = "auth_required"
	StatusInterrupted  Status = "interrupted"
	StatusCapacity     Status = "capacity_exhausted"
	StatusUsageLimited Status = "usage_limited"
	StatusOutputLimit  Status = "output_limit"
	StatusSuccess      Status = "success"
	StatusFailed       Status = "failed"
	StatusBlocked      Status = "blocked"
	StatusCancelled    Status = "cancelled"
	StatusRunning      Status = "running"
)

// Result is the public result contract. FinalMessage is the job's complete
// final agent message, or null while running, after a failed or resumable stop, or when
// the target produced no message. Summary and Details keep their established shapes.
type Result struct {
	JobID             string         `json:"job_id,omitempty"`
	FailureReason     string         `json:"failure_reason,omitempty"`
	Failure           string         `json:"failure,omitempty"`
	PendingHeavySteps []heavy.Step   `json:"pending_heavy_steps,omitempty"`
	Status            Status         `json:"status"`
	Summary           string         `json:"summary"`
	FinalMessage      *string        `json:"final_message"`
	ChangedFiles      []string       `json:"changed_files"`
	Verification      []Verification `json:"verification"`
	Details           string         `json:"details,omitempty"`
	Conversation      []jobs.Message `json:"conversation,omitempty"`
	Metadata          Metadata       `json:"metadata"`
}

type Verification struct {
	Command string `json:"command"`
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
}

type Metadata struct {
	LookupWarning    string     `json:"lookup_warning,omitempty"`
	IndexedOnly      bool       `json:"indexed_only,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	StopReason       string     `json:"stop_reason,omitempty"`
	StopMessage      string     `json:"stop_message,omitempty"`
	MessagingSource  string     `json:"messaging_source,omitempty"`
	MessagingWarning string     `json:"messaging_warning,omitempty"`
	Version          string     `json:"version"`
	MessagingEnabled bool       `json:"messaging_enabled"`
	*jobs.MessageStats
	KeepTerminalEnv bool       `json:"keep_terminal_env,omitempty"`
	LastEventAt     *time.Time `json:"last_event_at,omitempty"`
	LastMessage     string     `json:"last_message,omitempty"`
	MessagePath     string     `json:"message_path,omitempty"`
	FinalReceived   bool       `json:"final_received,omitempty"`
	IndexWarning    string     `json:"index_warning,omitempty"`
	ResumeWarning   string     `json:"resume_warning,omitempty"`
	CapacityRetries int        `json:"capacity_retries,omitempty"`
	CWD             string     `json:"cwd,omitempty"`
	Agent           string     `json:"agent,omitempty"`
	Access          string     `json:"access,omitempty"`
	Profile         string     `json:"profile,omitempty"`
	Effort          string     `json:"effort,omitempty"`
	Model           string     `json:"model,omitempty"`
	ExitCode        int        `json:"exit_code"`
	AgentSession    string     `json:"agent_session,omitempty"`
	JobID           string     `json:"job_id,omitempty"`
	LogPath         string     `json:"log_path,omitempty"`
}
