package claude

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"

	"github.com/nklisch/peeragent/internal/executil"
)

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
	path, err := lookPath("claude")
	if err != nil {
		return Result{ExitCode: 127}, errors.New("Claude CLI not found in PATH")
	}
	result, err := run.Run(ctx, path, buildArgs(opts), opts.CWD)
	normalizeJSONResult(&result)
	return result, err
}

func buildArgs(opts Options) []string {
	args := []string{
		"--print",
		"--output-format", "json",
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
	if opts.Resume != "" {
		args = append(args, "--resume", opts.Resume)
	}
	if opts.Effort == "" {
		opts.Effort = "xhigh"
	}
	args = append(args, "--effort", opts.Effort)
	return append(args, opts.Prompt)
}

type claudeJSONResult struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

func normalizeJSONResult(result *Result) {
	trimmed := strings.TrimSpace(result.Stdout)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return
	}
	var parsed claudeJSONResult
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return
	}
	if parsed.SessionID != "" {
		result.AgentSession = parsed.SessionID
	}
	if parsed.Result != "" {
		result.Stdout = parsed.Result
	}
}
