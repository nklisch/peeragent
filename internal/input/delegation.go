package input

import (
	"errors"
	"fmt"
	"strings"
)

// Delegation is the shared request contract used by every inbound delegation
// adapter. CLI job-control fields intentionally live on Request instead: they
// describe how to inspect a job, not work to send to a target agent.
type Delegation struct {
	TaskText   string
	CWD        string
	Agent      string
	FullAccess bool
	Profile    string
	Effort     string
	Model      string
	Resume     string
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

	agent, err := normalizeAgent(raw.Agent)
	if err != nil {
		return Delegation{}, err
	}
	raw.Agent = agent

	effort, err := normalizeEffort(raw.Agent, raw.Effort)
	if err != nil {
		return Delegation{}, err
	}
	raw.Effort = effort

	model, err := normalizeModel(raw.Agent, raw.Model)
	if err != nil {
		return Delegation{}, err
	}
	raw.Model = model
	if raw.Agent == "gemini" && raw.Model == "gemini-3.1-pro" && raw.Effort == "medium" {
		return Delegation{}, errors.New("--effort for Gemini 3.1 Pro must be low or high")
	}
	raw.Profile = strings.TrimSpace(raw.Profile)
	raw.Resume = strings.TrimSpace(raw.Resume)
	return raw, nil
}
