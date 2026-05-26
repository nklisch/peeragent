package result

type Status string

const (
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
	CWD          string `json:"cwd,omitempty"`
	Agent        string `json:"agent,omitempty"`
	Access       string `json:"access,omitempty"`
	Profile      string `json:"profile,omitempty"`
	Effort       string `json:"effort,omitempty"`
	ExitCode     int    `json:"exit_code"`
	AgentSession string `json:"agent_session,omitempty"`
	JobID        string `json:"job_id,omitempty"`
}
