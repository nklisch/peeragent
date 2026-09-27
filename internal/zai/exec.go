package zai

import (
	"context"
	"errors"
	"os/exec"
	"runtime"

	"github.com/nklisch/peeragent/internal/executil"
)

const ModelGLM52 = "glm-5.2"

type Result = executil.Result

type Options struct {
	CWD        string
	Prompt     string
	FullAccess bool
	Effort     string
	Model      string
	Resume     string
}

var lookPath = exec.LookPath

func Exec(ctx context.Context, opts Options) (Result, error) {
	return ExecWithRunner(ctx, executil.OSRunner{}, opts)
}

func ExecWithRunner(ctx context.Context, run executil.Runner, opts Options) (Result, error) {
	path, err := lookPath("pi")
	if err != nil {
		return Result{ExitCode: 127}, errors.New("Pi CLI not found in PATH")
	}
	args := buildArgs(opts)
	var stdin string
	if runtime.GOOS == "windows" {
		args = args[:len(args)-1]
		stdin = opts.Prompt
	}
	result, err := run.Run(ctx, path, args, opts.CWD, stdin)
	result.AgentSession = opts.Resume
	return result, err
}

func buildArgs(opts Options) []string {
	model := opts.Model
	if model == "" {
		model = ModelGLM52
	}
	effort := opts.Effort
	if effort == "" {
		effort = "high"
	}

	args := []string{
		"--provider", "zai",
		"--model", model,
		"--thinking", effort,
	}
	if opts.Resume != "" {
		args = append(args, "--session", opts.Resume)
	} else {
		args = append(args, "--no-session")
	}
	args = append(args, "-p")
	return append(args, opts.Prompt)
}
