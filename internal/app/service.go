// Package app contains peeragent's inbound-adapter-independent application
// services. It returns typed results and errors; formatting and process exit
// decisions remain in cmd/peeragent and the MCP adapter.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

type Options struct {
	Executor          TargetExecutor
	Launcher          JobLauncher
	Executable        func() (string, error)
	WorkingDirectory  func() (string, error)
	ProcessController ProcessController
}

type Service struct {
	executor          TargetExecutor
	launcher          JobLauncher
	executable        func() (string, error)
	workingDirectory  func() (string, error)
	processController ProcessController
}

type TargetExecutor interface {
	Execute(context.Context, input.Delegation) (executil.Result, error)
}

type JobLauncher interface {
	Launch(executable string, job jobs.Job) error
}

func NewService(opts Options) *Service {
	if opts.Executor == nil {
		opts.Executor = targetExecutor{}
	}
	if opts.Launcher == nil {
		opts.Launcher = ProcessLauncher{}
	}
	if opts.Executable == nil {
		opts.Executable = os.Executable
	}
	if opts.WorkingDirectory == nil {
		opts.WorkingDirectory = os.Getwd
	}
	if opts.ProcessController == nil {
		opts.ProcessController = processController{}
	}
	return &Service{
		executor:          opts.Executor,
		launcher:          opts.Launcher,
		executable:        opts.Executable,
		workingDirectory:  opts.WorkingDirectory,
		processController: opts.ProcessController,
	}
}

// Delegate executes one normalized delegation. A non-zero target exit is a
// valid failed result. An executor error is also represented in the result for
// CLI parity, then returned so protocol adapters can report infrastructure
// failures as tool errors rather than successful structured output.
func (s *Service) Delegate(ctx context.Context, delegation input.Delegation) (result.Result, error) {
	res, _, err := s.DelegateWithExecution(ctx, delegation)
	return res, err
}

// DelegateWithExecution is the CLI-facing form of Delegate. The raw execution
// value lets the CLI preserve its existing diagnostic log behavior without
// making raw target output part of the application result contract.
func (s *Service) DelegateWithExecution(ctx context.Context, delegation input.Delegation) (result.Result, executil.Result, error) {
	if s == nil || s.executor == nil {
		return failedDelegationResult(delegation, errors.New("delegation service has no executor")), executil.Result{ExitCode: 1}, errors.New("delegation service has no executor")
	}
	if ctx == nil {
		err := errors.New("delegation context is nil")
		return failedDelegationResult(delegation, err), executil.Result{ExitCode: 1}, err
	}
	if err := ctx.Err(); err != nil {
		return failedDelegationResult(delegation, err), executil.Result{ExitCode: 1}, err
	}

	execResult, execErr := s.executor.Execute(ctx, delegation)
	res := ResultFromExecution(delegation, execResult, execErr)
	return res, execResult, execErr
}

// Launch creates and starts a tracked async job. Job metadata is persisted
// before starting the child, preserving the existing cancellation and terminal
// race behavior while moving process startup behind an injectable port.
func (s *Service) Launch(ctx context.Context, delegation input.Delegation) (result.Result, error) {
	if s == nil || s.launcher == nil {
		err := errors.New("delegation service has no job launcher")
		return failedDelegationResult(delegation, err), err
	}
	if ctx == nil {
		err := errors.New("delegation context is nil")
		return failedDelegationResult(delegation, err), err
	}
	if err := ctx.Err(); err != nil {
		return failedDelegationResult(delegation, err), err
	}

	store := jobs.NewStore(delegation.CWD)
	job, err := store.Create(delegation.CWD, execSpecFromDelegation(delegation), delegation.TaskText)
	if err != nil {
		wrapped := fmt.Errorf("create async job: %w", err)
		return failedDelegationResult(delegation, wrapped), wrapped
	}

	executable, err := s.executable()
	if err != nil {
		wrapped := fmt.Errorf("resolve peeragent executable: %w", err)
		return failedJobResult(delegation, job, wrapped), wrapped
	}
	if err := s.launcher.Launch(executable, job); err != nil {
		wrapped := fmt.Errorf("launch async job: %w", err)
		return failedJobResult(delegation, job, wrapped), wrapped
	}

	return runningJobResult(delegation, job), nil
}

func execSpecFromDelegation(delegation input.Delegation) jobs.ExecSpec {
	return jobs.ExecSpec{
		Agent:      delegation.Agent,
		Access:     accessMode(delegation),
		Profile:    delegation.Profile,
		Effort:     delegation.Effort,
		Model:      delegation.Model,
		Resume:     delegation.Resume,
		JSON:       true,
		FullAccess: delegation.FullAccess,
	}
}

func runningJobResult(delegation input.Delegation, job jobs.Job) result.Result {
	return result.Result{
		Status:       result.StatusRunning,
		Summary:      fmt.Sprintf("%s implementation job started", agentDisplayName(delegation)),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      delegation.CWD,
			Agent:    delegation.Agent,
			Access:   accessMode(delegation),
			Profile:  delegation.Profile,
			Effort:   delegation.Effort,
			Model:    delegation.Model,
			ExitCode: 0,
			JobID:    job.ID,
		},
	}
}

func failedDelegationResult(delegation input.Delegation, err error) result.Result {
	return result.Result{
		Status:       result.StatusFailed,
		Summary:      err.Error(),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      delegation.CWD,
			Agent:    delegation.Agent,
			Access:   accessMode(delegation),
			Profile:  delegation.Profile,
			Effort:   delegation.Effort,
			Model:    delegation.Model,
			ExitCode: 1,
		},
	}
}

func failedJobResult(delegation input.Delegation, job jobs.Job, err error) result.Result {
	res := failedDelegationResult(delegation, err)
	res.Metadata.JobID = job.ID
	return res
}

func accessMode(delegation input.Delegation) string {
	if delegation.FullAccess {
		return "full-access"
	}
	return "default"
}

func agentDisplayName(delegation input.Delegation) string {
	switch delegation.Agent {
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

// ProcessLauncher is the production async process adapter. It intentionally
// owns only process setup; job state remains owned by the application service
// and jobs.Store so other inbound adapters can use the same state contract.
type ProcessLauncher struct{}

func (ProcessLauncher) Launch(executable string, job jobs.Job) error {
	cmd := exec.Command(executable, "--job-run", job.ID, "--cwd", job.CWD)
	cmd.Dir = job.CWD

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
	if err := jobs.NewStore(job.CWD).WritePID(job.ID, cmd.Process.Pid); err != nil {
		cleanupStartedProcess(cmd)
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		_ = jobs.NewStore(job.CWD).RemovePID(job.ID)
		cleanupStartedProcess(cmd)
		return err
	}
	return nil
}

func cleanupStartedProcess(cmd *exec.Cmd) {
	_ = jobs.SignalProcessGroup(cmd.Process.Pid, jobs.KillSignal())
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}
