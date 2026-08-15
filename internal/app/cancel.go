package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

const (
	cancelTermGrace = 5 * time.Second
	cancelKillGrace = 500 * time.Millisecond
)

// ProcessController is the narrow process-control port used after a
// cancellation transition commits. Its contract intentionally has no caller
// context: an interrupted CLI request must not strand a detached child after
// cancelled state is durable.
type ProcessController interface {
	TerminateAndWait(pid int, termGrace, killGrace time.Duration) error
}

type processController struct{}

// CancelJob atomically chooses cancellation or the already-persisted terminal
// winner, then performs best-effort process cleanup. The lock protects the
// result.json/job.json transition; cleanup happens after the lock so it cannot
// block a completion writer and is independent of ctx cancellation.
func (s *Service) CancelJob(ctx context.Context, raw JobRequest) (result.Result, error) {
	req, err := s.normalizeJobRequest(ctx, raw)
	if err != nil {
		return result.Result{}, err
	}
	if s.processController == nil {
		return result.Result{}, errors.New("job service has no process controller")
	}

	store := jobs.NewStore(req.CWD)
	job, err := store.Load(req.JobID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return JobLookupFailureResult(req, err), nil
		}
		return result.Result{}, fmt.Errorf("load async job %q: %w", req.JobID, err)
	}

	var (
		cancelResult       result.Result
		terminalWinner     result.Result
		terminalWinnerSet  bool
		cancellationStored bool
	)
	if err := store.WithJobLock(job.ID, func() error {
		// Before the terminal write, caller cancellation may safely abort the
		// operation. There is deliberately no context check after WriteJobResult.
		if err := ctx.Err(); err != nil {
			return err
		}

		current, err := store.Load(job.ID)
		if err != nil {
			return err
		}
		job = current

		if jobs.IsTerminalStatus(current.Status) {
			cancelResult, err = terminalJobResult(req, current)
			return err
		}

		prior, exists, err := readStoredResult(current.ResultPath)
		if err != nil {
			return err
		}
		if exists && isTerminalResultStatus(prior.Status) && prior.Status != result.StatusCancelled {
			current.Status = JobStatusFromResult(prior.Status)
			if _, err := store.SaveGuarded(current); err != nil {
				return err
			}
			terminalWinner = prior
			terminalWinnerSet = true
			return nil
		}

		if exists && prior.Status == result.StatusCancelled {
			// A prior cancel may have committed both terminal files and
			// disconnected before process cleanup. Repair job.json without
			// replacing its result, then continue cleanup below.
			cancelResult = prior
		} else {
			cancelResult = cancelledJobResult(req, current)
			if err := WriteJobResult(current.ResultPath, cancelResult); err != nil {
				return err
			}
		}
		current.Status = jobs.StatusCancelled
		if _, err := store.SaveGuarded(current); err != nil {
			return err
		}
		cancellationStored = true
		return nil
	}); err != nil {
		return result.Result{}, err
	}

	if terminalWinnerSet {
		if err := store.RemovePID(job.ID); err != nil {
			return terminalWinner, err
		}
		return terminalWinner, nil
	}

	if cancelResult.Status == "" {
		cancelResult = jobStatusResult(req, job)
	}
	if !cancellationStored && job.Status != jobs.StatusCancelled {
		if err := store.RemovePID(job.ID); err != nil {
			return cancelResult, err
		}
		return cancelResult, nil
	}

	pid, err := store.ReadPID(job.ID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cancelResult, nil
		}
		return cancelResult, fmt.Errorf("read async job %q pid: %w", job.ID, err)
	}
	var controllerErr error
	if pid > 0 {
		// This call intentionally does not receive ctx. Once cancellation is
		// persisted, TERM/KILL cleanup must finish after caller interruption.
		controllerErr = s.processController.TerminateAndWait(pid, cancelTermGrace, cancelKillGrace)
	}
	removeErr := store.RemovePID(job.ID)
	if controllerErr != nil {
		return cancelResult, controllerErr
	}
	if removeErr != nil {
		return cancelResult, fmt.Errorf("remove async job %q pid: %w", job.ID, removeErr)
	}
	return cancelResult, nil
}

func cancelledJobResult(req JobRequest, job jobs.Job) result.Result {
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

func (processController) TerminateAndWait(pid int, termGrace, killGrace time.Duration) error {
	if pid <= 0 {
		return nil
	}

	termErr := jobs.SignalProcessGroup(pid, jobs.TerminateSignal())
	if !jobs.ProcessGroupExists(pid) {
		return nil
	}
	if waitForProcessGroupExit(pid, termGrace) {
		return nil
	}

	killErr := jobs.SignalProcessGroup(pid, jobs.KillSignal())
	if !jobs.ProcessGroupExists(pid) {
		return nil
	}
	if waitForProcessGroupExit(pid, killGrace) {
		return nil
	}
	if killErr != nil {
		return fmt.Errorf("terminate process group %d: TERM: %v; KILL: %w", pid, termErr, killErr)
	}
	return fmt.Errorf("terminate process group %d: did not exit after TERM/KILL", pid)
}

func waitForProcessGroupExit(pid int, timeout time.Duration) bool {
	if !jobs.ProcessGroupExists(pid) {
		return true
	}
	if timeout <= 0 {
		return !jobs.ProcessGroupExists(pid)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !jobs.ProcessGroupExists(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !jobs.ProcessGroupExists(pid)
}
