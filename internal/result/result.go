package result

import "time"

type Status string

const (
	StatusCapacity  Status = "capacity_exhausted"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusBlocked   Status = "blocked"
	StatusCancelled Status = "cancelled"
	StatusRunning   Status = "running"
)

type Result struct {
	Status       Status         `json:"status"`
	Summary      string         `json:"summary"`
	ChangedFiles []string       `json:"changed_files"`
	Verification []Verification `json:"verification"`
	Details      string         `json:"details,omitempty"`
	Metadata     Metadata       `json:"metadata"`
}

type Verification struct {
	Command string `json:"command"`
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
}

type Metadata struct {
	KeepTerminalEnv bool       `json:"keep_terminal_env,omitempty"`
	LastEventAt     *time.Time `json:"last_event_at,omitempty"`
	LastMessage     string     `json:"last_message,omitempty"`
	MessagePath     string     `json:"message_path,omitempty"`
	FinalReceived   bool       `json:"final_received,omitempty"`
	IndexWarning    string     `json:"index_warning,omitempty"`
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
