package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

var applicationService = app.NewService(app.Options{})

const jobWaitInterval = 500 * time.Millisecond

func main() {
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		fmt.Fprintln(os.Stderr, "peeragent MCP mode is not supported; invoke the bundled peer skill instead")
		os.Exit(2)
	}
	if err := run(os.Args[1:]); err != nil {
		var usage *usageError
		if errors.As(err, &usage) {
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type usageError struct{}

func (*usageError) Error() string { return "invalid command; see --help" }

const usageText = `peeragent — delegate a task pass to another local coding assistant.

Usage:
  peeragent [flags] <task text>
  peeragent --prompt-file <path> [flags]
  peeragent --status|--result|--wait|--cancel <job-id> [flags]

Flags:
  --agent <codex|claude|gemini|zai>
                                  Target assistant (default codex).
  --model <name>                  Model override (codex: astra|luna|terra|sol,
                                  sol selects gpt-6.1-sol; GPT-6 Astra
                                  flagship; claude: fable|sonnet|opus|haiku;
                                  gemini: flash|pro|supported family id;
                                  zai: glm-5.2 only).
  --effort <low|medium|high|xhigh>
                                  Reasoning effort (codex default high, accepts
                                  all four; gemini default high, accepts
                                  low|medium|high; zai default high, accepts medium+;
                                  claude default xhigh, accepts high|xhigh).
  --profile <name>                Codex profile override.
  --resume <agent-session>        Resume in its recorded directory unless --cwd is set.
  --sandbox                       Use the default target mode (Gemini still
                                  auto-approves tools but sandboxes terminals).
  --full-access                   Run the target CLI without sandboxing.
  --worktree                      Reserved; returns a clear failure today.
  --cwd <path>                    Override the target or indexed job/session directory.
  --prompt-file <path>            Read task text from a file.
  --async                         Start a background job and return its id.
  --retry-capacity <N>            Retry Codex capacity stops up to N times (async only).
  --capacity-backoff <duration>   Delay between capacity retries (default 10m).
  --status <job-id>               Show status of a background job.
  --result <job-id>               Print a background job's current/final result.
  --wait <job-id>                 Wait for and print a job's terminal result.
  --cancel <job-id>               Best-effort cancel a background job.
  --json | --text                 Output format (default --json).
  -h, --help                      Show this help.
  --                              Treat all following arguments as task text.

Job ids work from any directory through the per-user index. Unknown options and
empty/missing option values are usage errors (exit 2). Capacity stops report
capacity_exhausted with agent_session; retries resume that session under one job id.
PEERAGENT_STATE_DIR overrides the user cache directory used for the lookup index.
`

func run(args []string) error {
	req, err := input.Parse(args, os.Stdin, os.Getwd)
	if err != nil {
		if writeErr := writeResult(input.Request{JSON: true}, result.Result{
			Status:       result.StatusFailed,
			Summary:      err.Error(),
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				ExitCode: 2,
			},
		}); writeErr != nil {
			return writeErr
		}
		return &usageError{}
	}
	if req.Help {
		fmt.Print(usageText)
		return nil
	}
	if req.JobRunID != "" {
		return runAsyncJob(req)
	}
	if req.StatusJobID != "" {
		return showJobStatus(req)
	}
	if req.ResultJobID != "" {
		return showJobResult(req)
	}
	if req.WaitJobID != "" {
		return waitForJobResult(req)
	}
	if req.CancelJobID != "" {
		return cancelJob(req)
	}
	if req.Async {
		return launchAsync(req)
	}
	if req.Worktree {
		if err := writeResult(req, result.Result{
			Status:       result.StatusFailed,
			Summary:      "worktree mode is recognized but not implemented yet",
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				CWD:          req.CWD,
				Agent:        agentID(req),
				Access:       accessMode(req),
				Profile:      req.Profile,
				Effort:       req.Effort,
				Model:        req.Model,
				AgentSession: req.Resume,
				ExitCode:     2,
			},
		}); err != nil {
			return err
		}
		os.Exit(1)
	}

	delegation := delegationFromRequest(req)
	res, execResult, _ := applicationService.DelegateWithExecution(context.Background(), delegation)
	attachExecutionLog(req, &res, execResult, "")
	if err := writeResult(req, res); err != nil {
		return err
	}
	if res.Status != result.StatusSuccess {
		os.Exit(1)
	}
	return nil
}

func launchAsync(req input.Request) error {
	res, err := applicationService.Launch(context.Background(), delegationFromRequest(req))
	if writeErr := writeResult(req, res); writeErr != nil {
		return writeErr
	}
	if err != nil {
		os.Exit(1)
	}
	return nil
}

func showJobStatus(req input.Request) error {
	res, err := applicationService.JobStatus(context.Background(), app.JobRequest{
		CWD:   req.CWD,
		JobID: req.StatusJobID,
	})
	if err != nil {
		return err
	}
	return writeJobControlResult(req, res)
}

func showJobResult(req input.Request) error {
	res, err := applicationService.JobResult(context.Background(), app.JobRequest{
		CWD:   req.CWD,
		JobID: req.ResultJobID,
	})
	if err != nil {
		return err
	}
	return writeJobControlResult(req, res)
}

func waitForJobResult(req input.Request) error {
	ticker := time.NewTicker(jobWaitInterval)
	defer ticker.Stop()

	for {
		res, err := applicationService.JobResult(context.Background(), app.JobRequest{
			CWD:   req.CWD,
			JobID: req.WaitJobID,
		})
		if err != nil {
			return err
		}
		if res.Status != result.StatusRunning {
			return writeJobControlResult(req, res)
		}
		<-ticker.C
	}
}

func cancelJob(req input.Request) error {
	res, err := applicationService.CancelJob(context.Background(), app.JobRequest{
		CWD:   req.CWD,
		JobID: req.CancelJobID,
	})
	if err != nil {
		return err
	}
	return writeJobControlResult(req, res)
}

func writeJobControlResult(req input.Request, res result.Result) error {
	if err := writeResult(req, res); err != nil {
		return err
	}
	if res.Metadata.ExitCode == 4 {
		os.Exit(4)
	}
	return nil
}

func runAsyncJob(req input.Request) error {
	store := jobs.NewStore(req.CWD)
	job, err := store.Load(req.JobRunID)
	if err != nil {
		return err
	}
	prompt, err := store.ReadPrompt(job.ID)
	if err != nil {
		return err
	}
	jobReq := requestFromJob(job, prompt)
	if jobReq.Worktree {
		res := result.Result{
			Status:       result.StatusFailed,
			Summary:      "worktree mode is recognized but not implemented yet",
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				CWD:          jobReq.CWD,
				Agent:        agentID(jobReq),
				Access:       accessMode(jobReq),
				Profile:      jobReq.Profile,
				Effort:       jobReq.Effort,
				Model:        jobReq.Model,
				AgentSession: jobReq.Resume,
				ExitCode:     2,
				JobID:        job.ID,
			},
		}
		return finishAsyncJob(store, job, res)
	}

	for attempt := 0; ; attempt++ {
		current, err := store.Load(job.ID)
		if err != nil {
			return err
		}
		if jobs.IsTerminalStatus(current.Status) {
			return nil
		}
		res, execResult, _ := applicationService.DelegateWithExecution(context.Background(), delegationFromRequest(jobReq))
		res.Metadata.JobID = job.ID
		res.Metadata.CapacityRetries = attempt
		logPath := jobTargetLogPath(job)
		if attempt > 0 {
			logPath = filepath.Join(filepath.Dir(job.LogPath), fmt.Sprintf("target-retry-%d.log", attempt))
		}
		attachExecutionLog(jobReq, &res, execResult, logPath)
		if res.Status != result.StatusCapacity || attempt >= job.Spec.RetryCapacity || res.Metadata.AgentSession == "" {
			return finishAsyncJob(store, job, res)
		}
		// Never start a fresh conversation on retry. Preserve all execution options,
		// continuing the same session with a continuation prompt instead of replaying work.
		jobReq.Resume = res.Metadata.AgentSession
		jobReq.TaskText = "Continue the task from where you stopped when model capacity was exhausted."
		proceed := false
		err = store.WithJobLock(job.ID, func() error {
			current, err := store.Load(job.ID)
			if err != nil {
				return err
			}
			if jobs.IsTerminalStatus(current.Status) {
				return nil
			}
			current.Spec.Resume = jobReq.Resume
			if err := store.Save(current); err != nil {
				return err
			}
			proceed = true
			return nil
		})
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}
		backoff := job.Spec.CapacityBackoff
		if backoff <= 0 {
			backoff = 10 * time.Minute
		}
		time.Sleep(backoff)
	}
}

func requestFromJob(job jobs.Job, prompt string) input.Request {
	fullAccess, worktree := accessFlagsFromSpec(job.Spec)
	req := input.Request{
		TaskText:   prompt,
		CWD:        job.CWD,
		JSON:       job.Spec.JSON,
		Agent:      job.Spec.Agent,
		Profile:    job.Spec.Profile,
		Effort:     job.Spec.Effort,
		Model:      job.Spec.Model,
		Resume:     job.Spec.Resume,
		FullAccess: fullAccess,
		Worktree:   worktree,
	}
	return req
}

func accessFlagsFromSpec(spec jobs.ExecSpec) (fullAccess bool, worktree bool) {
	switch spec.Access {
	case "worktree":
		return false, true
	case "full-access":
		return true, false
	default:
		return spec.FullAccess, spec.Worktree
	}
}

func finishAsyncJob(store jobs.Store, job jobs.Job, res result.Result) error {
	return app.FinishJob(store, job, res)
}

func accessMode(req input.Request) string {
	switch {
	case req.Worktree:
		return "worktree"
	case req.FullAccess:
		return "full-access"
	default:
		return "default"
	}
}

func writeResult(req input.Request, res result.Result) error {
	if req.JSON {
		encoded, err := result.FormatJSON(res)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, string(encoded))
		return err
	}
	_, err := fmt.Fprintln(os.Stdout, result.FormatText(res))
	return err
}

func details(stdout string, stderr string) string {
	switch {
	case stdout != "" && stderr != "":
		return "stdout:\n" + stdout + "\n\nstderr:\n" + stderr
	case stdout != "":
		return "stdout:\n" + stdout
	case stderr != "":
		return "stderr:\n" + stderr
	default:
		return ""
	}
}

func attachExecutionLog(req input.Request, res *result.Result, execResult executil.Result, logPath string) {
	path, err := writeExecutionLog(req, execResult, logPath)
	if err == nil && path != "" {
		res.Metadata.LogPath = path
	}
}

func writeExecutionLog(req input.Request, execResult executil.Result, logPath string) (string, error) {
	stdout, stderr := rawOutput(execResult)
	content := details(stdout, stderr)
	if content == "" {
		return "", nil
	}
	if logPath == "" {
		logPath = runLogPath(req.CWD, agentID(req))
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", err
	}
	if err := jobs.AtomicWriteFile(logPath, []byte(content+"\n"), 0o644); err != nil {
		return "", err
	}
	return logPath, nil
}

func rawOutput(execResult executil.Result) (string, string) {
	stdout := execResult.RawStdout
	if stdout == "" {
		stdout = execResult.Stdout
	}
	stderr := execResult.RawStderr
	if stderr == "" {
		stderr = execResult.Stderr
	}
	return stdout, stderr
}

func runLogPath(cwd string, agent string) string {
	name := fmt.Sprintf("%s-%d-%s.log", time.Now().UTC().Format("20060102T150405.000000000Z"), os.Getpid(), agent)
	return filepath.Join(cwd, ".peeragent", "runs", name)
}

func jobTargetLogPath(job jobs.Job) string {
	return filepath.Join(filepath.Dir(job.LogPath), "target.log")
}

func resultFromExecution(req input.Request, execResult executil.Result, execErr error) result.Result {
	return app.ResultFromExecution(delegationFromRequest(req), execResult, execErr)
}

func delegationFromRequest(req input.Request) input.Delegation {
	return input.Delegation{
		RetryCapacity:   req.RetryCapacity,
		CapacityBackoff: req.CapacityBackoff,
		TaskText:        req.TaskText,
		CWD:             req.CWD,
		Agent:           agentID(req),
		FullAccess:      req.FullAccess,
		Profile:         req.Profile,
		Effort:          req.Effort,
		Model:           req.Model,
		Resume:          req.Resume,
	}
}

func agentID(req input.Request) string {
	if req.Agent == "" {
		return "codex"
	}
	return req.Agent
}
