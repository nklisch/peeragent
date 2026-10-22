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
	state := streamState{}
	return execWithRunner(ctx, executil.OSRunner{ObserveStdout: func(line string) (bool, error) {
		progress, changed := state.consume(line)
		if changed {
			if err := executil.ReportProgress(ctx, progress); err != nil {
				return false, err
			}
		}
		return state.terminal, nil
	}}, opts, &state)
}

func ExecWithRunner(ctx context.Context, run executil.Runner, opts Options) (Result, error) {
	return execWithRunner(ctx, run, opts, nil)
}

func execWithRunner(ctx context.Context, run executil.Runner, opts Options, state *streamState) (Result, error) {
	path, err := lookPath("codex")
	if err != nil {
		return Result{ExitCode: 127}, errors.New("codex CLI not found in PATH")
	}
	// Codex reads the '-' prompt from stdin, avoiding command-line size limits
	// and shell quoting even when PATH resolves to a Windows npm shim.
	result, err := run.Run(ctx, path, buildArgs(opts), opts.CWD, opts.Prompt)
	if state == nil {
		normalizeJSONResult(&result)
	} else {
		normalizeStreamResult(&result, *state)
	}
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

type codexItem struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Role    string `json:"role"`
	Phase   string `json:"phase"`
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

type codexEvent struct {
	Type             string `json:"type"`
	ThreadID         string `json:"thread_id"`
	SessionID        string `json:"session_id"`
	Message          string `json:"message"`
	LastAgentMessage string `json:"last_agent_message"`
	Error            struct {
		Message string `json:"message"`
	} `json:"error"`
	Item    codexItem       `json:"item"`
	Payload json.RawMessage `json:"payload"`
}

// streamState accepts both exec JSONL and the terminal/session event envelopes.
// A completed agent_message without a final phase is also used for progress by
// Codex, so it must not start a shutdown timer on its own.
type streamState struct {
	session, message            string
	capacity, terminal, success bool
}

func (s *streamState) consume(line string) (executil.Progress, bool) {
	var event codexEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		if !s.terminal && isCapacityDiagnostic(line) {
			s.capacity = true
		}
		return executil.Progress{}, false
	}
	if event.Type == "event_msg" {
		if err := json.Unmarshal(event.Payload, &event); err != nil {
			return executil.Progress{}, false
		}
	}
	oldSession, oldMessage, oldTerminal := s.session, s.message, s.terminal
	switch event.Type {
	case "thread.started":
		if event.ThreadID != "" {
			s.session = event.ThreadID
		}
	case "session.started":
		if event.SessionID != "" {
			s.session = event.SessionID
		}
	case "item.completed", "response_item":
		item := event.Item
		if event.Type == "response_item" {
			if err := json.Unmarshal(event.Payload, &item); err != nil {
				break
			}
		}
		if item.Type == "agent_message" || (item.Type == "message" && item.Role == "assistant") {
			text := item.Text
			if text == "" {
				for _, part := range item.Content {
					text += part.Text
				}
			}
			if text != "" {
				s.message = text
			}
			if item.Phase == "final" || item.Phase == "final_answer" {
				s.terminal, s.success, s.capacity = true, true, false
			}
		}
	case "agent_message":
		if event.Message != "" {
			s.message = event.Message
		}
	case "error":
		if !s.terminal && (isCapacityDiagnostic(event.Message) || isCapacityDiagnostic(event.Error.Message)) {
			s.capacity = true
		}
	case "turn.failed":
		s.capacity = isCapacityDiagnostic(event.Message) || isCapacityDiagnostic(event.Error.Message)
		s.terminal, s.success = true, false
	case "turn.completed", "task_complete":
		if event.LastAgentMessage != "" {
			s.message = event.LastAgentMessage
		}
		s.capacity, s.terminal, s.success = false, true, true
	}
	changed := oldSession != s.session || oldMessage != s.message || oldTerminal != s.terminal
	progress := executil.Progress{AgentSession: s.session, FinalReceived: s.terminal}
	if oldMessage != s.message {
		progress.Message = s.message
	}
	return progress, changed
}

func normalizeJSONResult(result *Result) {
	// Classify the explicit Codex diagnostic before compacting JSONL to the final message.
	state := streamState{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		state.consume(line)
	}
	normalizeStreamResult(result, state)
}

func normalizeStreamResult(result *Result, state streamState) {
	session, text, capacity, terminal := state.session, state.message, state.capacity, state.terminal
	if result.CompletionStopped && terminal {
		result.ExitCode = 1
		if state.success {
			result.ExitCode = 0
		}
	}
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
