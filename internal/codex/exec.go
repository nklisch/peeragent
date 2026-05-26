package codex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type Options struct {
	CWD        string
	Prompt     string
	FullAccess bool
	Profile    string
}

type runner interface {
	Run(ctx context.Context, name string, args []string, cwd string) (Result, error)
}

func Exec(ctx context.Context, opts Options) (Result, error) {
	return ExecWithRunner(ctx, osExecRunner{}, opts)
}

func ExecWithRunner(ctx context.Context, run runner, opts Options) (Result, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return Result{ExitCode: 127}, errors.New("codex CLI not found in PATH")
	}
	return run.Run(ctx, path, buildArgs(opts), opts.CWD)
}

func buildArgs(opts Options) []string {
	profileArgs := profileArgs(opts.Profile)
	if opts.FullAccess {
		args := []string{
			"exec",
			"--cd", opts.CWD,
			"--dangerously-bypass-approvals-and-sandbox",
		}
		args = append(args, profileArgs...)
		return append(args, opts.Prompt)
	}
	args := []string{
		"exec",
		"--cd", opts.CWD,
		"--sandbox", "workspace-write",
		"--ask-for-approval", "on-request",
		"-c", "approvals_reviewer=auto_review",
	}
	args = append(args, profileArgs...)
	return append(args, opts.Prompt)
}

func profileArgs(profile string) []string {
	if profile == "" {
		return nil
	}
	return []string{"--profile", profile}
}

type osExecRunner struct{}

func (osExecRunner) Run(ctx context.Context, name string, args []string, cwd string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
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
	return result, fmt.Errorf("run codex: %w", err)
}
