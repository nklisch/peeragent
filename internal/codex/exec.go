package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nklisch/peeragent/internal/executil"
)

type Result = executil.Result

type Options struct {
	KeepTerminalEnv bool
	CWD             string
	Prompt          string
	FullAccess      bool
	Profile         string
	Effort          string
	Model           string
	Resume          string
}

var lookPath = exec.LookPath

func Exec(ctx context.Context, opts Options) (Result, error) {
	state := streamState{}
	return execWithRunner(ctx, executil.OSRunner{KeepTerminalEnv: opts.KeepTerminalEnv, ObserveStdout: func(line string) (executil.CompletionObservation, error) {
		progress, changed := state.consume(line)
		if changed {
			if err := executil.ReportProgress(ctx, progress); err != nil {
				return executil.CompletionUnchanged, err
			}
		}
		return state.observation, nil
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
	started                     bool
	session, message, preview   string
	capacity, terminal, success bool
	outputLimit                 bool
	usageLimited                bool
	authRequired                bool
	failure, failureMessage     string
	turnComplete                bool
	completedTurn               bool
	transportErrors             int
	transportExhausted          bool
	observation                 executil.CompletionObservation
}

func (s *streamState) consume(line string) (executil.Progress, bool) {
	s.observation = executil.CompletionUnchanged
	var event codexEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		if !s.terminal && (isCapacityDiagnostic(line) || isAuthDiagnostic(line) || isOutputLimitDiagnostic(line) || isUsageLimitDiagnostic(line)) {
			s.recordError(codexEvent{Message: line})
		}
		return executil.Progress{}, false
	}
	if event.Type == "event_msg" {
		if err := json.Unmarshal(event.Payload, &event); err != nil {
			return executil.Progress{}, false
		}
	}
	if event.Type != "" {
		s.started = true
	}
	oldMessage := s.message
	if s.turnComplete && isActivityEvent(event.Type) {
		s.turnComplete, s.terminal = false, false
		s.outputLimit, s.authRequired, s.usageLimited = false, false, false
		s.failure, s.failureMessage = "", ""
		s.observation = executil.CompletionActive
	}
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
				s.outputLimit, s.authRequired, s.usageLimited = false, false, false
				s.failure, s.failureMessage = "", ""
				s.turnComplete = false
				s.observation = executil.CompletionPending
			}
		}
	case "agent_message":
		if event.Message != "" {
			s.message = event.Message
		}
	case "error":
		if !s.terminal {
			s.recordError(event)
		}
	case "turn.failed":
		// A bare terminal marker closes the preceding error; only an explicit
		// replacement diagnostic may change its classification.
		s.recordError(event)
		s.terminal, s.success = true, false
		s.turnComplete = false
		s.observation = executil.CompletionPending
	case "turn.completed", "task_complete":
		s.completedTurn = true
		// A later turn marker must not downgrade an explicit session completion.
		if event.Type == "turn.completed" && s.terminal && !s.turnComplete {
			break
		}
		s.turnComplete = event.Type == "turn.completed"
		s.observation = executil.CompletionPending
		if event.LastAgentMessage != "" {
			s.message = event.LastAgentMessage
		}
		s.capacity, s.terminal, s.success = false, true, true
		s.outputLimit, s.authRequired, s.usageLimited = false, false, false
		s.failure, s.failureMessage = "", ""
	}
	if oldMessage != s.message {
		s.preview = executil.MessagePreview(s.message)
	}
	progress := executil.Progress{AgentSession: s.session, FinalReceived: s.terminal, Event: event.Type, LastMessage: s.preview}
	if oldMessage != s.message {
		progress.Message = s.message
	}
	return progress, event.Type != ""
}

// recordError keeps the last provider error verbatim. Terminal markers often
// omit it, especially on resumed attempts, so absence does not erase the cause.
func (s *streamState) recordError(event codexEvent) {
	message := event.Error.Message
	if message == "" {
		message = event.Message
	}
	if message == "" {
		return
	}
	if isTransportDiagnostic(message) && (event.Type != "turn.failed" || message != s.failureMessage) {
		s.transportErrors++
	}
	if exhaustedReconnect(message) {
		s.transportExhausted = true
	}
	s.failureMessage = message
	s.failure = ""
	if policyRefusalMessage(event) != "" {
		s.failure = "policy_refusal"
	}
	s.capacity = isCapacityDiagnostic(message)
	s.outputLimit = isOutputLimitDiagnostic(message)
	s.authRequired = isAuthDiagnostic(message)
	s.usageLimited = isUsageLimitDiagnostic(message)
}

// Turn completion is provisional: stop hooks and goals can continue the same
// session. Agent/tool activity revokes that candidate, while task_complete and
// explicitly final messages remain session completion observations.
func isActivityEvent(kind string) bool {
	return kind == "turn.started" || kind == "response_item" ||
		strings.HasPrefix(kind, "item.") || strings.HasPrefix(kind, "agent_") ||
		strings.HasPrefix(kind, "agent.") || strings.HasPrefix(kind, "response.") ||
		strings.HasPrefix(kind, "function_call") ||
		strings.Contains(kind, "tool") || strings.HasPrefix(kind, "exec_command_") ||
		strings.HasPrefix(kind, "web_search_") || strings.HasPrefix(kind, "patch_apply_")
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
	result.TurnStarted = state.started
	session, text, capacity, terminal := state.session, state.message, state.capacity, state.terminal
	if result.CompletionStopped && terminal {
		result.ExitCode = 1
		if state.success {
			result.ExitCode = 0
		}
	}
	result.CapacityExhausted = capacity
	result.UsageLimited = state.usageLimited
	result.OutputLimit = state.outputLimit
	result.AuthRequired = state.authRequired
	result.Failure, result.FailureMessage = state.failure, state.failureMessage
	if result.RawStderr == "" {
		result.RawStderr = result.Stderr
	}
	lines := strings.Split(result.Stderr, "\n")
	filtered := lines[:0]
	transportLogErrors := 0
	for _, line := range lines {
		if isTransportDiagnostic(codexLogDiagnostic(line)) {
			transportLogErrors++
		}
		// A structured terminal outcome is authoritative over stderr history.
		if !terminal && !result.CapacityExhausted && isCapacityDiagnostic(line) {
			result.CapacityExhausted = true
		}
		if !terminal && isAuthDiagnostic(line) {
			result.AuthRequired = true
		}
		if !terminal && isRolloutInterruption(line) {
			result.Interrupted = true
		}
		if !terminal && !result.OutputLimit && isOutputLimitDiagnostic(line) {
			result.OutputLimit = true
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "Orphan custom tool call:") {
			filtered = append(filtered, line)
		}
	}
	// Transport history can explain a failed compact operation, but never a
	// completed turn or an explicit unrelated provider failure.
	if result.ExitCode != 0 && !state.success && !state.completedTurn &&
		(state.transportExhausted || state.transportErrors >= 2 || transportLogErrors >= 2) &&
		isTransportTerminal(state.failureMessage) && !result.UsageLimited && !result.AuthRequired && result.Failure == "" {
		result.CapacityExhausted = true
	}
	result.Stderr = strings.TrimSpace(strings.Join(filtered, "\n"))
	if session != "" {
		result.AgentSession = session
	}
	if text != "" {
		result.Stdout = text
		result.FinalMessage = text
	}
}

var reconnectDiagnostic = regexp.MustCompile(`(?i)^reconnecting\.\.\.\s+(\d+)/(\d+)(?:\s|\(|$)`)

func exhaustedReconnect(message string) bool {
	match := reconnectDiagnostic.FindStringSubmatch(strings.TrimSpace(message))
	if match == nil {
		return false
	}
	n, err1 := strconv.Atoi(match[1])
	total, err2 := strconv.Atoi(match[2])
	return err1 == nil && err2 == nil && total > 0 && n == total
}

func isTransportDiagnostic(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	return reconnectDiagnostic.MatchString(message) ||
		strings.HasPrefix(message, "failed to connect to responses websocket") ||
		strings.Contains(message, ": failed to connect to responses websocket") ||
		strings.HasPrefix(message, "stream disconnected") ||
		strings.Contains(message, ": stream disconnected")
}

func isTransportTerminal(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	if rest, ok := strings.CutPrefix(message, "error running remote compact task:"); ok {
		return strings.HasPrefix(strings.TrimSpace(rest), "unexpected status ") || isTransportDiagnostic(rest)
	}
	return message == "" || isTransportDiagnostic(message) ||
		strings.HasPrefix(message, "failed to run pre-sampling compact")
}

func isUsageLimitDiagnostic(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.ReplaceAll(text, "’", "'")
	return strings.HasPrefix(text, "you've hit your usage limit.")
}

func isCapacityDiagnostic(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.TrimSpace(strings.TrimPrefix(text, "error:"))
	// Codex compaction propagates the same capacity error with this context.
	// Only unwrap its known diagnostic prefix; quoted/history mentions stay inert.
	text = strings.TrimSpace(strings.TrimPrefix(text, "error running remote compact task:"))
	const diagnostic = "selected model is at capacity"
	if !strings.HasPrefix(text, diagnostic) {
		return false
	}
	// Require the diagnostic's own prefix, not a quoted or historical mention.
	suffix := strings.TrimPrefix(text, diagnostic)
	return suffix == "" || strings.ContainsRune(" \t.,:;!?", rune(suffix[0]))
}

func isOutputLimitDiagnostic(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.TrimSpace(strings.TrimPrefix(text, "error:"))
	const diagnostic = "incomplete response returned, reason: max_output_tokens"
	if !strings.HasPrefix(text, diagnostic) {
		return false
	}
	suffix := strings.TrimPrefix(text, diagnostic)
	return suffix == "" || strings.ContainsRune(" \t.,:;!?", rune(suffix[0]))
}

// Match target diagnostics only, never an assistant message or tool output.
func isAuthDiagnostic(text string) bool {
	text = strings.TrimSpace(text)
	lower := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(text), "error:"))
	if strings.HasPrefix(lower, "your access token could not be refreshed because you have since logged out or signed in to another account.") {
		return true
	}
	if strings.HasPrefix(lower, "401 token_revoked") {
		suffix := strings.TrimPrefix(lower, "401 token_revoked")
		return suffix == "" || strings.ContainsRune(" \t.,:;!?", rune(suffix[0]))
	}
	diagnostic := codexLogDiagnostic(text)
	return strings.Contains(diagnostic, "auth error code: token_revoked")
}

// A valid timestamp/logger prefix distinguishes Codex stderr from quoted prose.
func codexLogDiagnostic(text string) string {
	text = strings.TrimSpace(text)
	if stamp, rest, ok := strings.Cut(text, " "); ok {
		if _, err := time.Parse(time.RFC3339Nano, stamp); err == nil {
			text = rest
		}
	}
	if !strings.HasPrefix(text, "ERROR codex_") {
		return ""
	}
	return text
}

func isRolloutInterruption(text string) bool {
	diagnostic := codexLogDiagnostic(text)
	const prefix = "ERROR codex_core::session: failed to record rollout items: thread "
	return strings.HasPrefix(diagnostic, prefix) && strings.HasSuffix(diagnostic, " not found") &&
		strings.TrimSuffix(strings.TrimPrefix(diagnostic, prefix), " not found") != ""
}

func policyRefusalMessage(event codexEvent) string {
	for _, text := range []string{event.Error.Message, event.Message} {
		if strings.HasPrefix(text, "This content was flagged for possible cybersecurity risk.") {
			return text
		}
	}
	return ""
}
