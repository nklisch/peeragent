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

type Options struct {
	KeepTerminalEnv bool
	CWD             string
	Prompt          string
	FullAccess      bool
	Effort          string
	Model           string
	Resume          string
}

type streamEvent struct {
	Event          string `json:"event"`
	ConversationID string `json:"conversation_id"`
	// Flat single-envelope fields for backwards compatibility with --output-format json.
	Status         string `json:"status"`
	Response       string `json:"response"`
	Error          string `json:"error"`
	FinishedReason string `json:"finished_reason"`
	NumTurns       int    `json:"num_turns"`
	StepUpdate     *struct {
		ConversationID string `json:"conversation_id"`
		StepIndex      int    `json:"step_index"`
		State          string `json:"state"`
		StepType       string `json:"step_type"`
		ToolName       string `json:"tool_name"`
		TextDelta      string `json:"text_delta"`
	} `json:"step_update"`
	Result *printEnvelope `json:"result"`
}

type printEnvelope struct {
	ConversationID string `json:"conversation_id"`
	Status         string `json:"status"`
	Response       string `json:"response"`
	Error          string `json:"error"`
	FinishedReason string `json:"finished_reason"`
	NumTurns       int    `json:"num_turns"`
}

var lookPath = exec.LookPath

func Exec(ctx context.Context, opts Options) (Result, error) {
	state := streamState{}
	return execWithRunner(ctx, executil.OSRunner{
		KeepTerminalEnv: opts.KeepTerminalEnv,
		ObserveStdout: func(line string) (executil.CompletionObservation, error) {
			progress, changed := state.consume(line)
			if changed {
				if err := executil.ReportProgress(ctx, progress); err != nil {
					return executil.CompletionUnchanged, err
				}
			}
			return state.observation, nil
		},
	}, opts, &state)
}

func ExecWithRunner(ctx context.Context, run executil.Runner, opts Options) (Result, error) {
	return execWithRunner(ctx, run, opts, nil)
}

func execWithRunner(ctx context.Context, run executil.Runner, opts Options, state *streamState) (Result, error) {
	path, err := lookPath("agy")
	if err != nil {
		return Result{ExitCode: 127}, errors.New("Antigravity CLI not found in PATH")
	}
	result, err := run.Run(ctx, path, buildArgs(opts), opts.CWD, "")
	result.AgentSession = opts.Resume
	if state == nil {
		normalizeResult(&result)
	} else {
		normalizeStreamResult(&result, *state)
	}
	return result, err
}

type streamState struct {
	started          bool
	session          string
	message          string
	preview          string
	terminal         bool
	success          bool
	authRequired     bool
	capacity         bool
	usageLimited     bool
	failureMessage   string
	activeStep       int
	hasActiveStep    bool
	stepText         strings.Builder
	observation      executil.CompletionObservation
}

func (s *streamState) consume(line string) (executil.Progress, bool) {
	s.observation = executil.CompletionUnchanged
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return executil.Progress{}, false
	}
	var event streamEvent
	if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
		return executil.Progress{}, false
	}

	oldSession := s.session
	oldMessage := s.message
	eventName := ""
	report := false

	switch event.Event {
	case "init":
		s.started = true
		if event.ConversationID != "" {
			s.session = event.ConversationID
		}
		eventName = "init"
		report = true
	case "step_update":
		if event.StepUpdate == nil {
			return executil.Progress{}, false
		}
		step := event.StepUpdate
		s.started = true
		if step.ConversationID != "" {
			s.session = step.ConversationID
		} else if event.ConversationID != "" {
			s.session = event.ConversationID
		}
		if s.terminal {
			s.terminal = false
			s.observation = executil.CompletionActive
		}
		if step.StepType == "agent_response" {
			if !s.hasActiveStep || s.activeStep != step.StepIndex {
				s.activeStep = step.StepIndex
				s.hasActiveStep = true
				s.stepText.Reset()
			}
			if step.TextDelta != "" {
				s.stepText.WriteString(step.TextDelta)
			}
			if strings.EqualFold(step.State, "DONE") {
				if text := s.stepText.String(); strings.TrimSpace(text) != "" {
					s.message = text
				}
				eventName = "agent_response"
				report = true
			}
		} else if step.StepType == "tool" {
			if step.ToolName != "" {
				eventName = "tool:" + step.ToolName
			} else {
				eventName = "tool"
			}
			report = true
		} else if strings.EqualFold(step.State, "DONE") && step.StepType != "" {
			eventName = step.StepType
			report = true
		}
	case "result":
		if event.Result == nil {
			return executil.Progress{}, false
		}
		s.applyEnvelope(*event.Result)
		eventName = "result"
		report = true
	case "":
		if event.Status == "" && event.ConversationID == "" {
			return executil.Progress{}, false
		}
		s.applyEnvelope(printEnvelope{
			ConversationID: event.ConversationID,
			Status:         event.Status,
			Response:       event.Response,
			Error:          event.Error,
			FinishedReason: event.FinishedReason,
			NumTurns:       event.NumTurns,
		})
		eventName = "result"
		report = true
	default:
		return executil.Progress{}, false
	}

	if oldMessage != s.message {
		s.preview = executil.MessagePreview(s.message)
	}
	if !report && s.session != oldSession {
		eventName = "session"
		report = true
	}
	if !report {
		return executil.Progress{}, false
	}
	progress := executil.Progress{
		AgentSession:  s.session,
		FinalReceived: s.terminal,
		Event:         eventName,
		LastMessage:   s.preview,
	}
	if oldMessage != s.message {
		progress.Message = s.message
	}
	return progress, true
}

func (s *streamState) applyEnvelope(envelope printEnvelope) {
	if envelope.ConversationID != "" {
		s.session = envelope.ConversationID
	}
	success := strings.EqualFold(envelope.Status, "SUCCESS")
	// Pre-turn failures (such as invalid model selection) emit a result envelope
	// with no conversation id and zero turns; keep started=false so reserved
	// inbox replies are released back to unread.
	if envelope.ConversationID != "" || envelope.NumTurns > 0 || success {
		s.started = true
	}
	s.terminal = true
	s.success = success
	s.observation = executil.CompletionPending
	if envelope.Response != "" {
		s.message = envelope.Response
	}
	if success {
		s.authRequired, s.capacity, s.usageLimited = false, false, false
		s.failureMessage = ""
		return
	}
	failure := strings.TrimSpace(envelope.Error)
	if failure == "" {
		failure = strings.TrimSpace(envelope.Response)
	}
	if failure != "" {
		s.recordDiagnostic(failure)
	}
}

func (s *streamState) recordDiagnostic(message string) {
	s.failureMessage = message
	s.authRequired = isAuthDiagnostic(message)
	s.capacity = isCapacityDiagnostic(message)
	s.usageLimited = isUsageLimitDiagnostic(message)
}

// normalizeResult translates agy's machine-readable print output (stream-json
// NDJSON or legacy single-object json) into the shared executor result and
// repairs false-success cases where agy exits 0 after a headless failure.
func normalizeResult(result *Result) {
	state := streamState{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		state.consume(line)
	}
	normalizeStreamResult(result, state)
}

func normalizeStreamResult(result *Result, state streamState) {
	result.TurnStarted = state.started
	if state.session != "" {
		result.AgentSession = state.session
	}
	if result.CompletionStopped && state.terminal {
		result.ExitCode = 1
		if state.success {
			result.ExitCode = 0
		}
	}
	if state.terminal {
		result.Stdout = state.message
		if state.success && state.message != "" {
			result.FinalMessage = state.message
		}
		if result.ExitCode == 0 && !state.success {
			result.ExitCode = 1
		}
	}
	result.AuthRequired = state.authRequired
	result.CapacityExhausted = state.capacity
	result.UsageLimited = state.usageLimited
	result.FailureMessage = state.failureMessage

	if !state.terminal {
		if result.ExitCode == 0 && (hasPrintModeError(result.Stdout) || hasPrintModeError(result.Stderr) || hasHeadlessPermissionDenial(result.Stderr)) {
			result.ExitCode = 1
		}
		if result.ExitCode != 0 {
			if isAuthDiagnostic(result.Stderr) || isAuthDiagnostic(result.Stdout) {
				result.AuthRequired = true
			}
			if result.FailureMessage == "" && hasHeadlessPermissionDenial(result.Stderr) {
				result.FailureMessage = strings.TrimSpace(result.Stderr)
			}
		}
	} else if result.ExitCode == 0 && hasHeadlessPermissionDenial(result.Stderr) {
		result.ExitCode = 1
		if result.FailureMessage == "" {
			result.FailureMessage = strings.TrimSpace(result.Stderr)
		}
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

func isAuthDiagnostic(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(lower, "authentication required") ||
		strings.Contains(lower, "authentication timed out") ||
		strings.Contains(lower, "unauthenticated") ||
		strings.Contains(lower, "login required")
}

func isCapacityDiagnostic(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(lower, "at capacity") ||
		strings.Contains(lower, "resource_exhausted") ||
		strings.Contains(lower, "model overloaded") ||
		strings.Contains(lower, "service unavailable")
}

func isUsageLimitDiagnostic(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(lower, "usage limit") ||
		strings.Contains(lower, "quota exceeded") ||
		strings.Contains(lower, "rate limit exceeded")
}

func buildArgs(opts Options) []string {
	args := []string{
		"--output-format", "stream-json",
		// Print mode cannot display edit-review prompts. accept-edits prevents
		// workspace file changes from waiting on an unavailable reviewer.
		"--mode", "accept-edits",
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Effort != "" {
		args = append(args, "--effort", opts.Effort)
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
	if opts.Resume != "" {
		args = append(args, "--conversation", opts.Resume)
	}
	// --print is a string-valued flag, not a mode toggle. Keep it after every
	// option so no later flag is accidentally consumed as the prompt.
	return append(args, "--print", opts.Prompt)
}
