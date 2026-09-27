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
	ExitCode     int
	Stdout       string
	Stderr       string
	RawStdout    string
	RawStderr    string
	AgentSession string
}

type Runner interface {
	Run(ctx context.Context, name string, args []string, cwd, stdin string) (Result, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, name string, args []string, cwd, stdin string) (Result, error) {
	cmd, err := commandContext(ctx, name, args)
	if err != nil {
		return Result{ExitCode: 1}, err
	}
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader(stdin)

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
