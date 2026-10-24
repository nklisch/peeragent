package input

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Delegation is the shared request contract used by every inbound delegation
// adapter. CLI job-control fields intentionally live on Request instead: they
// describe how to inspect a job, not work to send to a target agent.
type Delegation struct {
	NoPreamble             bool
	KeepTerminalEnv        bool
	AllowConcurrentSession bool
	RetryCapacity          int
	CapacityBackoff        time.Duration
	TaskText               string
	CWD                    string
	Agent                  string
	FullAccess             bool
	Profile                string
	Effort                 string
	Model                  string
	InheritEffort          bool
	InheritModel           bool
	Resume                 string
}

// NormalizeDelegation applies boundary defaults and validation for direct and
// persisted delegation requests. Keeping this at one boundary prevents async
// jobs from accepting a target/model combination the CLI rejects.
func NormalizeDelegation(raw Delegation, getwd func() (string, error)) (Delegation, error) {
	raw.TaskText = strings.TrimSpace(raw.TaskText)
	if raw.TaskText == "" {
		return Delegation{}, errors.New("no task text supplied")
	}

	raw.CWD = strings.TrimSpace(raw.CWD)
	if raw.CWD == "" {
		if getwd == nil {
			return Delegation{}, errors.New("resolve cwd: working-directory resolver is nil")
		}
		cwd, err := getwd()
		if err != nil {
			return Delegation{}, fmt.Errorf("resolve cwd: %w", err)
		}
		raw.CWD = cwd
	}

	absolute, err := filepath.Abs(raw.CWD)
	if err != nil {
		return Delegation{}, fmt.Errorf("resolve absolute cwd: %w", err)
	}
	raw.CWD = absolute

	agent, err := normalizeAgent(raw.Agent)
	if err != nil {
		return Delegation{}, err
	}
	raw.Agent = agent

	raw.Resume = strings.TrimSpace(raw.Resume)
	// Remember omissions before defaults erase the distinction between an
	// explicit override and a resume that should inherit its previous options.
	raw.InheritEffort = raw.InheritEffort || raw.Resume != "" && strings.TrimSpace(raw.Effort) == ""
	raw.InheritModel = raw.InheritModel || raw.Resume != "" && strings.TrimSpace(raw.Model) == ""
	raw.Model, raw.Effort, err = NormalizeModelAndEffort(raw.Agent, raw.Model, raw.Effort)
	if err != nil {
		return Delegation{}, err
	}
	raw.Profile = strings.TrimSpace(raw.Profile)
	return raw, nil
}

// NormalizeModelAndEffort validates the pair after resume inheritance as well
// as at the input boundary, since a partial override can change its validity.
func NormalizeModelAndEffort(agent, model, effort string) (string, string, error) {
	effort, err := normalizeEffort(agent, effort)
	if err != nil {
		return "", "", err
	}

	model, err = normalizeModel(agent, model)
	if err != nil {
		return "", "", err
	}
	if agent == "gemini" && model == "gemini-3.1-pro" && effort == "medium" {
		return "", "", errors.New("--effort for Gemini 3.1 Pro must be low or high")
	}
	return model, effort, nil
}
