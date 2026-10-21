package codex

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
	Profile    string
	Effort     string
	Model      string
	Resume     string
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
	// Codex reads the '-' prompt from stdin, avoiding command-line size limits
	// and shell quoting even when PATH resolves to a Windows npm shim.
	result, err := run.Run(ctx, path, buildArgs(opts), opts.CWD, opts.Prompt)
	normalizeJSONResult(&result)
	return result, err
}

func buildArgs(opts Options) []string {
	if opts.Resume != "" {
		return buildResumeArgs(opts)
	}

	profileArgs := profileArgs(opts.Profile)
	if opts.FullAccess {
		args := []string{
			"exec",
			"--json",
			"--cd", opts.CWD,
			"--dangerously-bypass-approvals-and-sandbox",
		}
		args = append(args, profileArgs...)
		args = append(args, modelArgs(opts.Model)...)
		args = append(args, effortArgs(opts.Effort)...)
		return append(args, "-")
	}
	args := []string{
		"exec",
		"--json",
		"--cd", opts.CWD,
		"--sandbox", "workspace-write",
	}
	args = append(args, profileArgs...)
	args = append(args, approvalArgs()...)
	args = append(args, modelArgs(opts.Model)...)
	args = append(args, effortArgs(opts.Effort)...)
	return append(args, "-")
}

func buildResumeArgs(opts Options) []string {
	args := append([]string{"exec"}, profileArgs(opts.Profile)...)
	args = append(args, "resume", "--json")
	if opts.FullAccess {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	} else {
		args = append(args, approvalArgs()...)
	}
	args = append(args, modelArgs(opts.Model)...)
	args = append(args, effortArgs(opts.Effort)...)
	return append(args, opts.Resume, "-")
}

func profileArgs(profile string) []string {
	if profile == "" {
		return nil
	}
	return []string{"--profile", profile}
}

func modelArgs(model string) []string {
	if model == "" {
		return nil
	}
	return []string{"--model", model}
}

func effortArgs(effort string) []string {
	if effort == "" {
		effort = "high"
	}
	return []string{"-c", `model_reasoning_effort="` + effort + `"`}
}

func approvalArgs() []string {
	return []string{
		"-c", `approval_policy="on-request"`,
		"-c", `approvals_reviewer="auto_review"`,
	}
}

type codexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Error    struct {
		Message string `json:"message"`
	} `json:"error"`
	Item struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
}

func normalizeJSONResult(result *Result) {
	// Classify the explicit Codex diagnostic before compacting JSONL to the final message.
	session, text, capacity, terminal := parseJSONL(result.Stdout)
	result.CapacityExhausted = capacity
	if result.RawStderr == "" {
		result.RawStderr = result.Stderr
	}
	lines := strings.Split(result.Stderr, "\n")
	filtered := lines[:0]
	for _, line := range lines {
		// A structured terminal outcome is authoritative over stderr history.
		if !terminal && !result.CapacityExhausted && isCapacityDiagnostic(line) {
			result.CapacityExhausted = true
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "Orphan custom tool call:") {
			filtered = append(filtered, line)
		}
	}
	result.Stderr = strings.TrimSpace(strings.Join(filtered, "\n"))
	if session != "" {
		result.AgentSession = session
	}
	if text != "" {
		result.Stdout = text
	}
}

func isCapacityDiagnostic(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.TrimSpace(strings.TrimPrefix(text, "error:"))
	const diagnostic = "selected model is at capacity"
	if !strings.HasPrefix(text, diagnostic) {
		return false
	}
	// Require the diagnostic's own prefix, not a quoted or historical mention.
	suffix := strings.TrimPrefix(text, diagnostic)
	return suffix == "" || strings.ContainsRune(" \t.,:;!?", rune(suffix[0]))
}

func parseJSONL(output string) (session, message string, capacity, terminal bool) {
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		var event codexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			if !terminal && isCapacityDiagnostic(line) {
				capacity = true
			}
			continue
		}
		switch event.Type {
		case "thread.started":
			if event.ThreadID != "" {
				session = event.ThreadID
			}
		case "item.completed":
			if event.Item.Type == "agent_message" && event.Item.Text != "" {
				message = event.Item.Text
			}
		case "error":
			if !terminal && (isCapacityDiagnostic(event.Message) || isCapacityDiagnostic(event.Error.Message)) {
				capacity = true
			}
		case "turn.failed":
			capacity = isCapacityDiagnostic(event.Message) || isCapacityDiagnostic(event.Error.Message)
			terminal = true
		case "turn.completed":
			// A transient capacity diagnostic followed by completion is not a stop.
			capacity = false
			terminal = true
		}
	}
	return
}
