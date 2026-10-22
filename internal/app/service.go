// Package app contains peeragent's inbound-adapter-independent application
// services. It returns typed results and errors; formatting and process exit
// decisions remain in cmd/peeragent.
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
	BlockingLauncher  BlockingJobLauncher
	Executable        func() (string, error)
	WorkingDirectory  func() (string, error)
	ProcessController ProcessController
}

type Service struct {
	executor          TargetExecutor
	launcher          JobLauncher
	blockingLauncher  BlockingJobLauncher
	executable        func() (string, error)
	workingDirectory  func() (string, error)
	processController ProcessController
}

type TargetExecutor interface {
	Execute(context.Context, input.Delegation) (executil.Result, error)
}

// JobWorker retains the native child handle until Wait collects its exit.
type JobWorker interface {
	Wait() (int, error)
}

type BlockingJobLauncher interface {
	Start(executable string, job jobs.Job) (JobWorker, error)
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
	if opts.BlockingLauncher == nil {
		opts.BlockingLauncher = ProcessLauncher{}
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
		blockingLauncher:  opts.BlockingLauncher,
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
	if res.Metadata.AgentSession != "" {
		if err := jobs.IndexSession(delegation.Agent, res.Metadata.AgentSession, delegation.CWD); err != nil {
			res.Metadata.IndexWarning = err.Error()
		}
	}
	return res, execResult, execErr
}

// Launch creates and starts a tracked async job. Job metadata is persisted
// before starting the child, preserving the existing cancellation and terminal
// race behavior while moving process startup behind an injectable port.
func (s *Service) Launch(ctx context.Context, delegation input.Delegation) (result.Result, error) {
	var start func(string, jobs.Job) error
	if s != nil && s.launcher != nil {
		start = s.launcher.Launch
	}
	return s.launchJob(ctx, delegation, start)
}

// LaunchBlocking keeps the worker handle instead of releasing it. The caller
// must Wait exactly once, on every successful launch, including failure results.
func (s *Service) LaunchBlocking(ctx context.Context, delegation input.Delegation) (result.Result, JobWorker, error) {
	var worker JobWorker
	var start func(string, jobs.Job) error
	if s != nil && s.blockingLauncher != nil {
		start = func(executable string, job jobs.Job) error {
			var err error
			worker, err = s.blockingLauncher.Start(executable, job)
			return err
		}
	}
	res, err := s.launchJob(ctx, delegation, start)
	return res, worker, err
}

func (s *Service) launchJob(ctx context.Context, delegation input.Delegation, start func(string, jobs.Job) error) (result.Result, error) {
	if s == nil || start == nil {
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
	return protectSession(delegation, func() (result.Result, error) { return s.launch(delegation, start) })
}

func (s *Service) launch(delegation input.Delegation, start func(string, jobs.Job) error) (result.Result, error) {
	store := jobs.NewStore(delegation.CWD)
	job, err := store.Create(delegation.CWD, execSpecFromDelegation(delegation), delegation.TaskText)
	if err != nil {
		wrapped := fmt.Errorf("create async job: %w", err)
		return failedDelegationResult(delegation, wrapped), wrapped
	}

	indexErr := jobs.IndexJob(job.ID, job.CWD)
	executable, err := s.executable()
	if err != nil {
		wrapped := fmt.Errorf("resolve peeragent executable: %w", err)
		return failedJobResult(delegation, job, wrapped), wrapped
	}
	if err := start(executable, job); err != nil {
		wrapped := fmt.Errorf("launch async job: %w", err)
		return failedJobResult(delegation, job, wrapped), wrapped
	}

	res := runningJobResult(delegation, job)
	if indexErr != nil {
		res.Metadata.IndexWarning = indexErr.Error()
	}
	return res, nil
}

func execSpecFromDelegation(delegation input.Delegation) jobs.ExecSpec {
	return jobs.ExecSpec{
		KeepTerminalEnv: delegation.KeepTerminalEnv,
		RetryCapacity:   delegation.RetryCapacity,
		CapacityBackoff: delegation.CapacityBackoff,
		Agent:           delegation.Agent,
		Access:          accessMode(delegation),
		Profile:         delegation.Profile,
		Effort:          delegation.Effort,
		Model:           delegation.Model,
		Resume:          delegation.Resume,
		JSON:            true,
		FullAccess:      delegation.FullAccess,
	}
}

func runningJobResult(delegation input.Delegation, job jobs.Job) result.Result {
	return result.Result{
		Status:       result.StatusRunning,
		Summary:      fmt.Sprintf("%s implementation job started", agentDisplayName(delegation)),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			KeepTerminalEnv: delegation.KeepTerminalEnv,
			CWD:             delegation.CWD,
			Agent:           delegation.Agent,
			Access:          accessMode(delegation),
			Profile:         delegation.Profile,
			Effort:          delegation.Effort,
			Model:           delegation.Model,
			ExitCode:        0,
			JobID:           job.ID,
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
			KeepTerminalEnv: delegation.KeepTerminalEnv,
			CWD:             delegation.CWD,
			Agent:           delegation.Agent,
			Access:          accessMode(delegation),
			Profile:         delegation.Profile,
			Effort:          delegation.Effort,
			Model:           delegation.Model,
			ExitCode:        1,
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
	return agentDefinition(delegation.Agent).DisplayName
}

// ProcessLauncher is the production worker process adapter. It intentionally
// owns only process setup; job state remains owned by the application service
// and jobs.Store so other inbound adapters can use the same state contract.
//
// The dependencies are nil for normal production construction and default to
// the operating-system operations below. Keeping them as function dependencies
// makes post-start cleanup failures deterministic to test without using file
// permissions or replacing process-global functions.
type ProcessLauncher struct {
	command        func(string, ...string) *exec.Cmd
	writePID       func(jobs.Store, string, int) error
	releaseProcess func(*os.Process) error
}

// Launch detaches the handle for async callers; Start retains it for blocking
// callers. Both paths persist the PID inside the session check/start lock.
func (l ProcessLauncher) Launch(executable string, job jobs.Job) error {
	worker, err := l.start(executable, job)
	if err != nil {
		return err
	}
	release := l.releaseProcess
	if release == nil {
		release = func(process *os.Process) error { return process.Release() }
	}
	if err := release(worker.cmd.Process); err != nil {
		_ = jobs.NewStore(job.CWD).RemovePID(job.ID)
		cleanupStartedProcess(worker.cmd)
		return err
	}
	return nil
}

func (l ProcessLauncher) Start(executable string, job jobs.Job) (JobWorker, error) {
	worker, err := l.start(executable, job)
	if err != nil {
		return nil, err
	}
	return worker, nil
}

type processWorker struct{ cmd *exec.Cmd }

func (w *processWorker) Wait() (int, error) {
	err := w.cmd.Wait()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	if err != nil {
		return 1, err
	}
	return w.cmd.ProcessState.ExitCode(), nil
}

func (l ProcessLauncher) start(executable string, job jobs.Job) (*processWorker, error) {
	command := l.command
	if command == nil {
		command = exec.Command
	}
	writePID := l.writePID
	if writePID == nil {
		writePID = func(store jobs.Store, id string, pid int) error { return store.WritePID(id, pid) }
	}
	cmd := command(executable, "--job-run", job.ID, "--cwd", job.CWD)
	cmd.Dir = job.CWD
	logFile, err := os.OpenFile(job.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()
	cmd.Stdout, cmd.Stderr = logFile, logFile
	jobs.ApplyDetachAttrs(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	store := jobs.NewStore(job.CWD)
	if err := writePID(store, job.ID, cmd.Process.Pid); err != nil {
		cleanupStartedProcess(cmd)
		return nil, err
	}
	return &processWorker{cmd: cmd}, nil
}

func cleanupStartedProcess(cmd *exec.Cmd) {
	_ = jobs.SignalProcessGroup(cmd.Process.Pid, jobs.KillSignal())
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}
