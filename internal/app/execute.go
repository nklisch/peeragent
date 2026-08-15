package app

import (
	"context"
	"fmt"

	"github.com/nklisch/peeragent/internal/agent"
	"github.com/nklisch/peeragent/internal/claude"
	"github.com/nklisch/peeragent/internal/codex"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/gemini"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/prompt"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/zai"
)

type targetExecutor struct{}

func (targetExecutor) Execute(ctx context.Context, delegation input.Delegation) (executil.Result, error) {
	targetID := agent.ID(delegation.Agent)
	if targetID == "" {
		targetID = agent.DefaultID()
	}
	definition, ok := agent.Lookup(targetID)
	if !ok {
		return executil.Result{ExitCode: 2}, fmt.Errorf("unsupported delegation agent %q", delegation.Agent)
	}
	agentPrompt := prompt.BuildForAgent(definition.PromptIdentity, delegation.TaskText)
	switch definition.ID {
	case agent.GeminiID:
		return gemini.Exec(ctx, gemini.Options{
			CWD:        delegation.CWD,
			Prompt:     agentPrompt,
			FullAccess: delegation.FullAccess,
			Effort:     delegation.Effort,
			Model:      delegation.Model,
			Resume:     delegation.Resume,
		})
	case agent.ClaudeID:
		return claude.Exec(ctx, claude.Options{
			CWD:        delegation.CWD,
			Prompt:     agentPrompt,
			FullAccess: delegation.FullAccess,
			Effort:     delegation.Effort,
			Model:      delegation.Model,
			Resume:     delegation.Resume,
		})
	case agent.ZAIID:
		return zai.Exec(ctx, zai.Options{
			CWD:        delegation.CWD,
			Prompt:     agentPrompt,
			FullAccess: delegation.FullAccess,
			Effort:     delegation.Effort,
			Model:      delegation.Model,
			Resume:     delegation.Resume,
		})
	case agent.CodexID:
		return codex.Exec(ctx, codex.Options{
			CWD:        delegation.CWD,
			Prompt:     agentPrompt,
			FullAccess: delegation.FullAccess,
			Profile:    delegation.Profile,
			Effort:     delegation.Effort,
			Model:      delegation.Model,
			Resume:     delegation.Resume,
		})
	default:
		return executil.Result{ExitCode: 2}, fmt.Errorf("unsupported delegation agent %q", delegation.Agent)
	}
}

// ResultFromExecution preserves the CLI's established result semantics while
// keeping target-specific execution behind the application port.
func ResultFromExecution(delegation input.Delegation, execResult executil.Result, execErr error) result.Result {
	status := result.StatusSuccess
	summary := fmt.Sprintf("%s implementation completed", agentDisplayName(delegation))
	if execErr != nil {
		status = result.StatusFailed
		summary = execErr.Error()
	} else if execResult.ExitCode != 0 {
		status = result.StatusFailed
		summary = fmt.Sprintf("%s exited with non-zero status", agentDisplayName(delegation))
	}

	return result.Result{
		Status:       status,
		Summary:      summary,
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Details:      executionDetails(execResult.Stdout, execResult.Stderr),
		Metadata: result.Metadata{
			CWD:          delegation.CWD,
			Agent:        delegation.Agent,
			Access:       accessMode(delegation),
			Profile:      delegation.Profile,
			Effort:       delegation.Effort,
			Model:        delegation.Model,
			AgentSession: agentSession(delegation, execResult),
			ExitCode:     execResult.ExitCode,
		},
	}
}

func executionDetails(stdout string, stderr string) string {
	switch {
	case stdout != "" && stderr != "":
		return "stdout:\n" + stdout + "\n\nstderr:\n" + stderr
	case stdout != "":
		return "stdout:\n" + stdout
	case stderr != "":
		return "stderr:\n" + stderr
	default:
		return ""
	}
}

func agentSession(delegation input.Delegation, execResult executil.Result) string {
	if execResult.AgentSession != "" {
		return execResult.AgentSession
	}
	return delegation.Resume
}

func agentPromptName(delegation input.Delegation) string {
	return agentDefinition(delegation.Agent).PromptIdentity
}

func agentDefinition(id string) agent.Definition {
	definition, ok := agent.Lookup(agent.ID(id))
	if ok {
		return definition
	}
	definition, _ = agent.Lookup(agent.DefaultID())
	return definition
}
