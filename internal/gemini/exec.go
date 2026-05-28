package gemini

import (
	"context"
	"errors"
	"os/exec"

	"github.com/nklisch/peeragent/internal/executil"
)

type Result = executil.Result

type Options struct {
	CWD          string
	Prompt       string
	FullAccess   bool
	Model        string
	Resume       string
	PrintTimeout string
}

var lookPath = exec.LookPath

func Exec(ctx context.Context, opts Options) (Result, error) {
	return ExecWithRunner(ctx, executil.OSRunner{}, opts)
}

func ExecWithRunner(ctx context.Context, run executil.Runner, opts Options) (Result, error) {
	path, err := lookPath("agy")
	if err != nil {
		return Result{ExitCode: 127}, errors.New("Antigravity CLI not found in PATH")
	}
	result, err := run.Run(ctx, path, buildArgs(opts), opts.CWD)
	result.AgentSession = opts.Resume
	return result, err
}

func buildArgs(opts Options) []string {
	args := []string{"--print"}
	if opts.FullAccess {
		args = append(args, "--dangerously-skip-permissions")
	} else {
		args = append(args, "--sandbox")
	}
	args = append(args, "--add-dir", opts.CWD)
	if opts.PrintTimeout == "" {
		opts.PrintTimeout = "15m"
	}
	args = append(args, "--print-timeout", opts.PrintTimeout)
	if opts.Resume != "" {
		args = append(args, "--conversation", opts.Resume)
	}
	return append(args, opts.Prompt)
}
