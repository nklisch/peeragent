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
	args = append(args, effortArgs(opts.Effort)...)
	return append(args, opts.Prompt)
}

func buildResumeArgs(opts Options) []string {
	args := []string{
		"exec",
		"resume",
		"--json",
	}
	if opts.FullAccess {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	} else {
		args = append(args, approvalArgs()...)
	}
	args = append(args, effortArgs(opts.Effort)...)
	return append(args, opts.Resume, opts.Prompt)
}

func profileArgs(profile string) []string {
	if profile == "" {
		return nil
	}
	return []string{"--profile", profile}
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
	Item     struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
}

func normalizeJSONResult(result *Result) {
	session, text := parseJSONL(result.Stdout)
	if session != "" {
		result.AgentSession = session
	}
	if text != "" {
		result.Stdout = text
	}
}

func parseJSONL(output string) (string, string) {
	var session string
	var messages []string
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var event codexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Type == "thread.started" && event.ThreadID != "" {
			session = event.ThreadID
		}
		if event.Type == "item.completed" && event.Item.Type == "agent_message" && event.Item.Text != "" {
			messages = append(messages, event.Item.Text)
		}
	}
	return session, strings.Join(messages, "\n\n")
}
