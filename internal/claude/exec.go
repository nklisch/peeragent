package claude

import (
	"context"
	"errors"
	"os/exec"

	"github.com/nklisch/alt-subagent/internal/executil"
)

type Result = executil.Result

type Options struct {
	CWD        string
	Prompt     string
	FullAccess bool
	Effort     string
	Model      string
}

var lookPath = exec.LookPath

func Exec(ctx context.Context, opts Options) (Result, error) {
	return ExecWithRunner(ctx, executil.OSRunner{}, opts)
}

func ExecWithRunner(ctx context.Context, run executil.Runner, opts Options) (Result, error) {
	path, err := lookPath("claude")
	if err != nil {
		return Result{ExitCode: 127}, errors.New("Claude CLI not found in PATH")
	}
	return run.Run(ctx, path, buildArgs(opts), opts.CWD)
}

func buildArgs(opts Options) []string {
	args := []string{
		"--print",
		"--output-format", "text",
		"--add-dir", opts.CWD,
	}
	if opts.FullAccess {
		args = append(args, "--dangerously-skip-permissions")
	} else {
		args = append(args, "--permission-mode", "auto")
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Effort == "" {
		opts.Effort = "medium"
	}
	args = append(args, "--effort", opts.Effort)
	return append(args, opts.Prompt)
}
