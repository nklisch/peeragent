package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/nklisch/codex-implement/internal/codex"
	"github.com/nklisch/codex-implement/internal/input"
	"github.com/nklisch/codex-implement/internal/jobs"
	"github.com/nklisch/codex-implement/internal/prompt"
	"github.com/nklisch/codex-implement/internal/result"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

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
		return launchAsync(args, req)
	}
	if req.Worktree {
		if err := writeResult(req, result.Result{
			Status:       result.StatusFailed,
			Summary:      "worktree mode is recognized but not implemented yet",
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				CWD:      req.CWD,
				Access:   accessMode(req),
				Profile:  req.Profile,
				Effort:   req.Effort,
				ExitCode: 2,
			},
		}); err != nil {
			return err
		}
		os.Exit(1)
	}

	codexPrompt := prompt.Build(req.TaskText)
	execResult, execErr := codex.Exec(context.Background(), codex.Options{
		CWD:        req.CWD,
		Prompt:     codexPrompt,
		FullAccess: req.FullAccess,
		Profile:    req.Profile,
		Effort:     req.Effort,
	})

	res := resultFromExecution(req, execResult, execErr)
	if err := writeResult(req, res); err != nil {
		return err
	}
	if res.Status != result.StatusSuccess {
		os.Exit(1)
	}
	return nil
}

func launchAsync(args []string, req input.Request) error {
	store := jobs.NewStore(req.CWD)
	job, err := store.Create(req.TaskText)
	if err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	childArgs := []string{"--job-run", job.ID, "--cwd", req.CWD}
	childArgs = append(childArgs, stripAsync(args)...)
	cmd := exec.Command(executable, childArgs...)
	cmd.Dir = req.CWD

	logFile, err := os.OpenFile(job.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return err
	}
	job.PID = cmd.Process.Pid
	if err := store.Save(job); err != nil {
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		return err
	}

	return writeResult(req, result.Result{
		Status:       result.StatusRunning,
		Summary:      "Codex implementation job started",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			Access:   accessMode(req),
			Profile:  req.Profile,
			Effort:   req.Effort,
			ExitCode: 0,
			JobID:    job.ID,
		},
	})
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

	if isTerminalJobStatus(job.Status) {
		return writeResult(req, result.Result{
			Status:       resultStatusFromJob(job.Status),
			Summary:      fmt.Sprintf("Async job %s is already %s", job.ID, job.Status),
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				CWD:      req.CWD,
				ExitCode: 0,
				JobID:    job.ID,
			},
		})
	}

	if job.PID > 0 {
		process, err := os.FindProcess(job.PID)
		if err == nil {
			_ = process.Kill()
		}
	}

	res := result.Result{
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
	encoded, err := result.FormatJSON(res)
	if err != nil {
		return err
	}
	if err := os.WriteFile(job.ResultPath, append(encoded, '\n'), 0o644); err != nil {
		return err
	}
	job.Status = "cancelled"
	job.PID = 0
	if err := store.Save(job); err != nil {
		return err
	}
	return writeResult(req, res)
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
	if req.Worktree {
		res := result.Result{
			Status:       result.StatusFailed,
			Summary:      "worktree mode is recognized but not implemented yet",
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				CWD:      req.CWD,
				Access:   accessMode(req),
				Profile:  req.Profile,
				Effort:   req.Effort,
				ExitCode: 2,
				JobID:    job.ID,
			},
		}
		return finishAsyncJob(store, job, res)
	}

	codexPrompt := prompt.Build(req.TaskText)
	execResult, execErr := codex.Exec(context.Background(), codex.Options{
		CWD:        req.CWD,
		Prompt:     codexPrompt,
		FullAccess: req.FullAccess,
		Profile:    req.Profile,
		Effort:     req.Effort,
	})
	res := resultFromExecution(req, execResult, execErr)
	res.Metadata.JobID = job.ID
	return finishAsyncJob(store, job, res)
}

func finishAsyncJob(store jobs.Store, job jobs.Job, res result.Result) error {
	encoded, err := result.FormatJSON(res)
	if err != nil {
		return err
	}
	if err := os.WriteFile(job.ResultPath, append(encoded, '\n'), 0o644); err != nil {
		return err
	}
	if res.Status == result.StatusSuccess {
		job.Status = "complete"
	} else {
		job.Status = "failed"
	}
	job.PID = 0
	return store.Save(job)
}

func stripAsync(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--async" {
			continue
		}
		out = append(out, arg)
	}
	return out
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

func resultFromExecution(req input.Request, execResult codex.Result, execErr error) result.Result {
	status := result.StatusSuccess
	summary := "Codex implementation completed"
	if execErr != nil {
		status = result.StatusFailed
		summary = execErr.Error()
	} else if execResult.ExitCode != 0 {
		status = result.StatusFailed
		summary = "Codex exited with non-zero status"
	}

	return result.Result{
		Status:       status,
		Summary:      summary,
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Details:      details(execResult.Stdout, execResult.Stderr),
		Metadata: result.Metadata{
			CWD:      req.CWD,
			Access:   accessMode(req),
			Profile:  req.Profile,
			Effort:   req.Effort,
			ExitCode: execResult.ExitCode,
		},
	}
}
