package codex

import (
	"context"
	"errors"
	"os/exec"

	"github.com/nklisch/peeragent/internal/executil"
)

type Result = executil.Result

type Options struct {
	CWD        string
	Prompt     string
	FullAccess bool
	Profile    string
	Effort     string
}

var lookPath = exec.LookPath

func Exec(ctx context.Context, opts Options) (Result, error) {
	return ExecWithRunner(ctx, executil.OSRunner{}, opts)
}

func ExecWithRunner(ctx context.Context, run executil.Runner, opts Options) (Result, error) {
	path, err := lookPath("codex")
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
		args = append(args, effortArgs(opts.Effort)...)
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
	args = append(args, effortArgs(opts.Effort)...)
	return append(args, opts.Prompt)
}

func profileArgs(profile string) []string {
	if profile == "" {
		return nil
	}
	return []string{"--profile", profile}
}

func effortArgs(effort string) []string {
	if effort == "" {
		effort = "high"
	}
	return []string{"-c", `model_reasoning_effort="` + effort + `"`}
}
