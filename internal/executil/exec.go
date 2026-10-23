package executil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	TurnStarted       bool
	OutputLimit       bool
	CapacityExhausted bool
	ExitCode          int
	Stdout            string
	Stderr            string
	RawStdout         string
	RawStderr         string
	AgentSession      string
	// FinalMessage is the target's final agent message when the adapter could
	// identify one; empty means the target produced none. Stdout may instead
	// hold raw CLI output when no message was found.
	FinalMessage      string
	CompletionStopped bool
}

type Runner interface {
	Run(ctx context.Context, name string, args []string, cwd string, stdin string) (Result, error)
}

type OSRunner struct {
	KeepTerminalEnv bool
	// ObserveStdout receives complete lines, without a scanner size limit.
	// Pending starts/restarts grace; Active cancels it; Unchanged preserves it.
	ObserveStdout   func(string) (CompletionObservation, error)
	CompletionGrace time.Duration
	// Test seam: pause at the drained-output decision before checking input/reap.
	afterOutputDrain func()
}

func (r OSRunner) Run(ctx context.Context, name string, args []string, cwd string, stdin string) (Result, error) {
	cmd, err := commandContext(ctx, name, args)
	if err != nil {
		return Result{ExitCode: 1}, fmt.Errorf("run %s: %w", name, err)
	}
	cmd.Dir = cwd
	cmd.Env = jobEnvironment(ctx, targetEnvironment(cmd.Environ(), r.KeepTerminalEnv))
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	var completionStopped bool
	if r.ObserveStdout == nil {
		err = cmd.Run()
	} else {
		completionStopped, err = r.runObserved(ctx, cmd, &stdout)
	}

	stdoutText := stdout.String()
	stderrText := stderr.String()
	result := Result{
		ExitCode:          0,
		Stdout:            stdoutText,
		Stderr:            stderrText,
		RawStdout:         stdoutText,
		RawStderr:         stderrText,
		CompletionStopped: completionStopped,
	}
	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	result.ExitCode = 1
	return result, fmt.Errorf("run %s: %w", name, err)
}
