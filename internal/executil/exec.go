package executil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Result struct {
	CapacityExhausted bool
	ExitCode          int
	Stdout            string
	Stderr            string
	RawStdout         string
	RawStderr         string
	AgentSession      string
}

type Runner interface {
	Run(ctx context.Context, name string, args []string, cwd string, stdin string) (Result, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, name string, args []string, cwd string, stdin string) (Result, error) {
	cmd, err := commandContext(ctx, name, args)
	if err != nil {
		return Result{ExitCode: 1}, fmt.Errorf("run %s: %w", name, err)
	}
	cmd.Dir = cwd
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	stdoutText := stdout.String()
	stderrText := stderr.String()
	result := Result{
		ExitCode:  0,
		Stdout:    stdoutText,
		Stderr:    stderrText,
		RawStdout: stdoutText,
		RawStderr: stderrText,
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
