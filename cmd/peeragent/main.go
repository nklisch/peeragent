package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nklisch/peeragent/internal/claude"
	"github.com/nklisch/peeragent/internal/codex"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/gemini"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/prompt"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/zai"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

const usageText = `peeragent — delegate a task pass to another local coding assistant.

Usage:
  peeragent [flags] <task text>
  peeragent --prompt-file <path> [flags]
  peeragent --status|--result|--cancel <job-id> [flags]

Flags:
  --agent <codex|claude|gemini|zai>
                                  Target assistant (default codex).
  --model <name>                  Model override (codex: luna|terra|sol,
                                  GPT-5.6 recommended for all Codex work;
                                  claude: fable|sonnet|opus|haiku;
                                  gemini: gemini-3.5; zai: glm-5.2 only).
  --effort <low|medium|high|xhigh>
                                  Reasoning effort (codex default high, accepts
                                  all four; zai default high, accepts medium+;
                                  claude default xhigh, accepts high|xhigh;
                                  unused by gemini).
  --profile <name>                Codex profile override.
  --resume <agent-session>        Resume a prior target-agent session when supported.
  --sandbox                       Use the default bounded target CLI mode.
  --full-access                   Run the target CLI without sandboxing.
  --worktree                      Reserved; returns a clear failure today.
  --cwd <path>                    Repo directory the target runs in.
  --prompt-file <path>            Read task text from a file.
  --async                         Start a background job and return its id.
  --status <job-id>               Show status of a background job.
  --result <job-id>               Print a background job's final result.
  --cancel <job-id>               Best-effort cancel a background job.
  --json | --text                 Output format (default --json).
  -h, --help                      Show this help.
`

func run(args []string) error {
	req, err := input.Parse(args, os.Stdin, os.Getwd)
	if err != nil {
		return writeResult(input.Request{JSON: true}, result.Result{
			Status:       result.StatusFailed,
			Summary:      err.Error(),
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				ExitCode: 2,
			},
		})
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

	execResult, execErr := executeRequest(context.Background(), req)
	res := resultFromExecution(req, execResult, execErr)
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
	store := jobs.NewStore(req.CWD)
	job, err := store.Create(req.CWD, execSpecFromRequest(req), req.TaskText)
	if err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, asyncJobRunArgs(job.ID, req.CWD)...)
	cmd.Dir = req.CWD

	logFile, err := os.OpenFile(job.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	jobs.ApplyDetachAttrs(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}
	if err := store.WritePID(job.ID, cmd.Process.Pid); err != nil {
		cleanupStartedAsyncProcess(cmd)
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		_ = store.RemovePID(job.ID)
		cleanupStartedAsyncProcess(cmd)
		return err
	}

	return writeResult(req, result.Result{
		Status:       result.StatusRunning,
		Summary:      fmt.Sprintf("%s implementation job started", agentDisplayName(req)),
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
			ExitCode:     0,
			JobID:        job.ID,
		},
	})
}

func cleanupStartedAsyncProcess(cmd *exec.Cmd) {
	_ = jobs.SignalProcessGroup(cmd.Process.Pid, jobs.KillSignal())
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func showJobStatus(req input.Request) error {
	store := jobs.NewStore(req.CWD)
	job, err := store.Load(req.StatusJobID)
	if err != nil {
		return writeJobLookupFailure(req, req.StatusJobID, err)
	}

	return writeResult(req, result.Result{
		Status:       resultStatusFromJob(job.Status),
		Summary:      fmt.Sprintf("Async job %s is %s", job.ID, job.Status),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			ExitCode: 0,
			JobID:    job.ID,
		},
	})
}

func showJobResult(req input.Request) error {
	store := jobs.NewStore(req.CWD)
	job, err := store.Load(req.ResultJobID)
	if err != nil {
		return writeJobLookupFailure(req, req.ResultJobID, err)
	}

	content, err := os.ReadFile(job.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return writeResult(req, result.Result{
				Status:       result.StatusRunning,
				Summary:      fmt.Sprintf("Async job %s is still running", job.ID),
				ChangedFiles: []string{},
				Verification: []result.Verification{},
				Metadata: result.Metadata{
					CWD:      req.CWD,
					ExitCode: 0,
					JobID:    job.ID,
				},
			})
		}
		return err
	}

	if req.JSON {
		_, err := os.Stdout.Write(content)
		if err != nil {
			return err
		}
		if len(content) == 0 || content[len(content)-1] != '\n' {
			_, err = fmt.Fprintln(os.Stdout)
		}
		return err
	}

	var res result.Result
	if err := json.Unmarshal(content, &res); err != nil {
		return err
	}
	return writeResult(req, res)
}

func execSpecFromRequest(req input.Request) jobs.ExecSpec {
	return jobs.ExecSpec{
		Agent:      agentID(req),
		Access:     accessMode(req),
		Profile:    req.Profile,
		Effort:     req.Effort,
		Model:      req.Model,
		Resume:     req.Resume,
		JSON:       req.JSON,
		FullAccess: req.FullAccess,
		Worktree:   req.Worktree,
	}
}

func asyncJobRunArgs(jobID string, cwd string) []string {
	return []string{"--job-run", jobID, "--cwd", cwd}
}

func resultStatusFromJob(status string) result.Status {
	switch status {
	case "complete":
		return result.StatusSuccess
	case "failed":
		return result.StatusFailed
	case "cancelled":
		return result.StatusCancelled
	default:
		return result.StatusRunning
	}
}

func cancelJob(req input.Request) error {
	store := jobs.NewStore(req.CWD)
	job, err := store.Load(req.CancelJobID)
	if err != nil {
		return writeJobLookupFailure(req, req.CancelJobID, err)
	}

	res := cancelledJobResult(req, job)
	cancelApplied := false
	var priorResult result.Result
	priorResultSet := false
	// Keep the terminal result and job status as one serialized transition.
	if err := store.WithJobLock(job.ID, func() error {
		var err error
		job, err = store.Load(job.ID)
		if err != nil {
			return err
		}
		if isTerminalJobStatus(job.Status) {
			return nil
		}
		res = cancelledJobResult(req, job)
		if prior, ok := conflictingTerminalResult(job.ResultPath, res.Status); ok {
			job.Status = jobStatusFromResult(prior.Status)
			if _, err := store.SaveGuarded(job); err != nil {
				return err
			}
			priorResult = prior
			priorResultSet = true
			return nil
		}
		if err := writeJobResult(job.ResultPath, res); err != nil {
			return err
		}
		job.Status = "cancelled"
		if _, err := store.SaveGuarded(job); err != nil {
			return err
		}
		cancelApplied = true
		return nil
	}); err != nil {
		return err
	}
	// A completed result can appear before job.json is updated if the child
	// exits between the two writes. Preserve that winner and clean stale PID.
	if priorResultSet {
		_ = store.RemovePID(job.ID)
		return writeResult(req, priorResult)
	}
	if !cancelApplied {
		res = alreadyTerminalJobResult(req, job)
		if job.Status != "cancelled" {
			_ = store.RemovePID(job.ID)
			return writeResult(req, res)
		}
		// Another cancel command may have written cancelled and exited before
		// signalling. Continue so a live child is not stranded.
	}

	pid, err := store.ReadPID(job.ID)
	if err != nil {
		if os.IsNotExist(err) {
			return writeResult(req, res)
		}
		return err
	}
	if pid > 0 {
		_ = jobs.SignalProcessGroup(pid, jobs.TerminateSignal())
		if !waitForProcessGroupExit(pid, 5*time.Second) {
			_ = jobs.SignalProcessGroup(pid, jobs.KillSignal())
			_ = waitForProcessGroupExit(pid, 500*time.Millisecond)
		}
	}
	if err := store.RemovePID(job.ID); err != nil {
		return err
	}
	return writeResult(req, res)
}

func alreadyTerminalJobResult(req input.Request, job jobs.Job) result.Result {
	return result.Result{
		Status:       resultStatusFromJob(job.Status),
		Summary:      fmt.Sprintf("Async job %s is already %s", job.ID, job.Status),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			ExitCode: 0,
			JobID:    job.ID,
		},
	}
}

func cancelledJobResult(req input.Request, job jobs.Job) result.Result {
	return result.Result{
		Status:       result.StatusCancelled,
		Summary:      fmt.Sprintf("Async job %s cancelled", job.ID),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			ExitCode: 0,
			JobID:    job.ID,
		},
	}
}

func waitForProcessGroupExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !jobs.ProcessGroupExists(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !jobs.ProcessGroupExists(pid)
}

func writeJobLookupFailure(req input.Request, jobID string, err error) error {
	if writeErr := writeResult(req, jobLookupFailureResult(req, jobID, err)); writeErr != nil {
		return writeErr
	}
	os.Exit(4)
	return nil
}

func jobLookupFailureResult(req input.Request, jobID string, err error) result.Result {
	return result.Result{
		Status:       result.StatusFailed,
		Summary:      fmt.Sprintf("async job %s lookup failed: %v", jobID, err),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			ExitCode: 4,
			JobID:    jobID,
		},
	}
}

func isTerminalJobStatus(status string) bool {
	switch status {
	case "complete", "failed", "cancelled":
		return true
	default:
		return false
	}
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

	execResult, execErr := executeRequest(context.Background(), jobReq)
	res := resultFromExecution(jobReq, execResult, execErr)
	res.Metadata.JobID = job.ID
	attachExecutionLog(jobReq, &res, execResult, jobTargetLogPath(job))
	return finishAsyncJob(store, job, res)
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
	targetStatus := jobStatusFromResult(res.Status)
	err := store.WithJobLock(job.ID, func() error {
		current, err := store.Load(job.ID)
		if err != nil {
			return err
		}
		if isTerminalJobStatus(current.Status) && current.Status != targetStatus {
			return nil
		}
		if prior, ok := conflictingTerminalResult(current.ResultPath, res.Status); ok {
			current.Status = jobStatusFromResult(prior.Status)
			_, err := store.SaveGuarded(current)
			return err
		}
		if err := writeJobResult(current.ResultPath, res); err != nil {
			return err
		}
		current.Status = targetStatus
		_, err = store.SaveGuarded(current)
		return err
	})
	removeErr := store.RemovePID(job.ID)
	if err != nil {
		return err
	}
	return removeErr
}

func jobStatusFromResult(status result.Status) string {
	switch status {
	case result.StatusSuccess:
		return "complete"
	case result.StatusCancelled:
		return "cancelled"
	default:
		return "failed"
	}
}

func conflictingTerminalResult(path string, status result.Status) (result.Result, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return result.Result{}, false
	}
	var prior result.Result
	if err := json.Unmarshal(content, &prior); err != nil {
		return result.Result{}, false
	}
	if isTerminalResultStatus(prior.Status) && prior.Status != status {
		return prior, true
	}
	return result.Result{}, false
}

func isTerminalResultStatus(status result.Status) bool {
	switch status {
	case result.StatusSuccess, result.StatusFailed, result.StatusCancelled:
		return true
	default:
		return false
	}
}

func writeJobResult(path string, res result.Result) error {
	encoded, err := result.FormatJSON(res)
	if err != nil {
		return err
	}
	return jobs.AtomicWriteFile(path, append(encoded, '\n'), 0o644)
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

func executeRequest(ctx context.Context, req input.Request) (executil.Result, error) {
	agentPrompt := prompt.BuildForAgent(agentPromptName(req), req.TaskText)
	switch agentID(req) {
	case "gemini":
		return gemini.Exec(ctx, gemini.Options{
			CWD:        req.CWD,
			Prompt:     agentPrompt,
			FullAccess: req.FullAccess,
			Model:      req.Model,
			Resume:     req.Resume,
		})
	case "claude":
		return claude.Exec(ctx, claude.Options{
			CWD:        req.CWD,
			Prompt:     agentPrompt,
			FullAccess: req.FullAccess,
			Effort:     req.Effort,
			Model:      req.Model,
			Resume:     req.Resume,
		})
	case "zai":
		return zai.Exec(ctx, zai.Options{
			CWD:        req.CWD,
			Prompt:     agentPrompt,
			FullAccess: req.FullAccess,
			Effort:     req.Effort,
			Model:      req.Model,
			Resume:     req.Resume,
		})
	default:
		return codex.Exec(ctx, codex.Options{
			CWD:        req.CWD,
			Prompt:     agentPrompt,
			FullAccess: req.FullAccess,
			Profile:    req.Profile,
			Effort:     req.Effort,
			Model:      req.Model,
			Resume:     req.Resume,
		})
	}
}

func resultFromExecution(req input.Request, execResult executil.Result, execErr error) result.Result {
	status := result.StatusSuccess
	summary := fmt.Sprintf("%s implementation completed", agentDisplayName(req))
	if execErr != nil {
		status = result.StatusFailed
		summary = execErr.Error()
	} else if execResult.ExitCode != 0 {
		status = result.StatusFailed
		summary = fmt.Sprintf("%s exited with non-zero status", agentDisplayName(req))
	}

	return result.Result{
		Status:       status,
		Summary:      summary,
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Details:      details(execResult.Stdout, execResult.Stderr),
		Metadata: result.Metadata{
			CWD:          req.CWD,
			Agent:        agentID(req),
			Access:       accessMode(req),
			Profile:      req.Profile,
			Effort:       req.Effort,
			Model:        req.Model,
			AgentSession: agentSession(req, execResult),
			ExitCode:     execResult.ExitCode,
		},
	}
}

func agentSession(req input.Request, execResult executil.Result) string {
	if execResult.AgentSession != "" {
		return execResult.AgentSession
	}
	return req.Resume
}

func agentID(req input.Request) string {
	if req.Agent == "" {
		return "codex"
	}
	return req.Agent
}

func agentDisplayName(req input.Request) string {
	switch agentID(req) {
	case "gemini":
		return "Gemini"
	case "claude":
		return "Claude"
	case "zai":
		return "Z.AI GLM 5.2"
	default:
		return "Codex"
	}
}

func agentPromptName(req input.Request) string {
	switch agentID(req) {
	case "gemini":
		return "Gemini through Antigravity CLI"
	case "claude":
		return "Claude Code"
	case "zai":
		return "Z.AI GLM 5.2 through Pi"
	default:
		return "Codex"
	}
}
