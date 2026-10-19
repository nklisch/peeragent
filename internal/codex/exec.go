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
	result, err := run.Run(ctx, path, buildArgs(opts), opts.CWD)
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
		return append(args, opts.Prompt)
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
	return append(args, opts.Prompt)
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
	return append(args, opts.Resume, opts.Prompt)
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
	session, text, capacity, completed := parseJSONL(result.Stdout)
	result.CapacityExhausted = capacity || (containsCapacity(result.Stderr) && (result.ExitCode != 0 || !completed))
	if result.RawStderr == "" {
		result.RawStderr = result.Stderr
	}
	lines := strings.Split(result.Stderr, "\n")
	filtered := lines[:0]
	for _, line := range lines {
		if !strings.Contains(line, "Orphan custom tool call") {
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

func containsCapacity(text string) bool {
	return strings.Contains(strings.ToLower(text), "selected model is at capacity")
}

func parseJSONL(output string) (session, message string, capacity, completed bool) {
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		var event codexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			if strings.HasPrefix(strings.ToLower(line), "selected model is at capacity") {
				capacity = true
				completed = false
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
		case "error", "turn.failed":
			if containsCapacity(event.Message) || containsCapacity(event.Error.Message) {
				capacity = true
				completed = false
			}
		case "turn.completed":
			// A transient capacity diagnostic followed by completion is not a stop.
			capacity = false
			completed = true
		}
	}
	return
}
