package gemini

import (
	"context"
	"errors"
	"os/exec"
	"strings"

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
	normalizeResult(&result)
	return result, err
}

// normalizeResult repairs agy print-mode's habit of exiting 0 even when it
// failed to produce a model response. agy prints a final "Error: ..." line on
// such failures (print-mode timeouts, auth), so a zero exit alongside that line
// is a false success; remap it to a non-zero exit so the wrapper reports the
// failure instead.
func normalizeResult(result *Result) {
	if result.ExitCode != 0 {
		return
	}
	if hasPrintModeError(result.Stdout) || hasPrintModeError(result.Stderr) {
		result.ExitCode = 1
	}
}

// agyPrintFatalSignals are substrings agy emits on its final "Error: ..." line
// when print mode fails to produce a model response (timeout or auth). Matching
// these specifically — rather than any line starting with "Error: " — avoids
// misclassifying a peer agent's own legitimate final line that merely happens to
// begin with "Error: " (e.g. reporting a bug it found).
var agyPrintFatalSignals = []string{
	"timed out waiting for response",
	"authentication required",
	"authentication timed out",
}

func hasPrintModeError(output string) bool {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return false
	}
	if idx := strings.LastIndexByte(trimmed, '\n'); idx != -1 {
		trimmed = strings.TrimSpace(trimmed[idx+1:])
	}
	if !strings.HasPrefix(trimmed, "Error: ") {
		return false
	}
	lower := strings.ToLower(trimmed)
	for _, sig := range agyPrintFatalSignals {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

func buildArgs(opts Options) []string {
	args := []string{"--print"}
	if opts.FullAccess {
		args = append(args, "--dangerously-skip-permissions")
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
