package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"

	"github.com/nklisch/peeragent/internal/executil"
)

type Result = executil.Result

const agyPrintTimeout = "15m"

type Options struct {
	CWD        string
	Prompt     string
	FullAccess bool
	Effort     string
	Model      string
	Resume     string
}

type printEnvelope struct {
	ConversationID string `json:"conversation_id"`
	Status         string `json:"status"`
	Response       string `json:"response"`
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

// normalizeResult translates agy's machine-readable print envelope into the
// shared executor result and repairs two false-success cases. Current agy can
// exit 0 after an internal print failure or after soft-denying a required tool
// because a headless run cannot display a permission prompt. In either case the
// delegated task is incomplete, so reporting success would mislead the host.
func normalizeResult(result *Result) {
	var envelope printEnvelope
	if json.Unmarshal([]byte(strings.TrimSpace(result.Stdout)), &envelope) == nil &&
		(envelope.Status != "" || envelope.ConversationID != "") {
		if envelope.ConversationID != "" {
			result.AgentSession = envelope.ConversationID
		}
		result.Stdout = envelope.Response
		if result.ExitCode == 0 && !strings.EqualFold(envelope.Status, "SUCCESS") {
			result.ExitCode = 1
		}
	}

	if result.ExitCode != 0 {
		return
	}
	if hasPrintModeError(result.Stdout) || hasPrintModeError(result.Stderr) || hasHeadlessPermissionDenial(result.Stderr) {
		result.ExitCode = 1
	}
}

// agyPrintFatalSignals are substrings older agy versions emitted on their final
// "Error: ..." line when print mode failed to produce a model response. Match
// these specifically rather than every Error line so a legitimate agent report
// about a bug remains a successful response.
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

func hasHeadlessPermissionDenial(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "headless mode cannot prompt for") &&
		strings.Contains(lower, "auto-denied")
}

func buildArgs(opts Options) []string {
	args := []string{
		"--output-format", "json",
		"--model", opts.Model,
		"--effort", opts.Effort,
		// Print mode cannot display edit-review prompts. accept-edits prevents
		// workspace file changes from waiting on an unavailable reviewer.
		"--mode", "accept-edits",
	}
	// Print mode has nobody to answer tool prompts. Auto-approval is therefore
	// required for an autonomous coding pass; without it agy soft-denies every
	// unlisted shell command and cannot run tests. The default still enables the
	// native terminal sandbox. Note that agy's sandbox contains terminal
	// processes only: --dangerously-skip-permissions can also approve direct file
	// tools outside the workspace, so callers must treat Gemini delegation as a
	// trusted local agent. --full-access additionally removes terminal isolation.
	if !opts.FullAccess {
		// agy currently applies these left-to-right: --sandbox resets the tool
		// approval mode, so it must precede the auto-approval flag.
		args = append(args, "--sandbox")
	}
	args = append(args, "--dangerously-skip-permissions")
	args = append(args, "--add-dir", opts.CWD)
	args = append(args, "--print-timeout", agyPrintTimeout)
	if opts.Resume != "" {
		args = append(args, "--conversation", opts.Resume)
	}
	// --print is a string-valued flag, not a mode toggle. Keep it after every
	// option so no later flag is accidentally consumed as the prompt.
	return append(args, "--print", opts.Prompt)
}
