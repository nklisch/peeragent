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

// ProcessController is the narrow process-control port used during
// cancellation. It has no caller context: interruption must not strand a
// detached child after cleanup begins.
type ProcessController interface {
	TerminateAndWait(jobID string, worker jobs.PIDRecord, termGrace, killGrace time.Duration) error
}

type processController struct{}

// CancelJob commits cancelled only after the worker and its descendants have
// exited. Holding the job lock prevents a completion writer from racing the
// cleanup and preserves the existing terminal-winner rule.
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

	var cancelResult result.Result
	if err := store.WithJobLock(job.ID, func() error {
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
			cancelResult = prior
			return nil
		}

		worker, err := store.ReadPIDRecord(job.ID)
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("cannot confirm async job %q stopped: pid is missing", job.ID)
		}
		if err != nil {
			return fmt.Errorf("read async job %q pid: %w", job.ID, err)
		}
		if worker.PID <= 0 {
			return fmt.Errorf("cannot confirm async job %q stopped: invalid pid %d", job.ID, worker.PID)
		}
		// Once cleanup starts it completes even if the caller disconnects.
		if err := s.processController.TerminateAndWait(job.ID, worker, cancelTermGrace, cancelKillGrace); err != nil {
			return fmt.Errorf("terminate async job %q: %w", job.ID, err)
		}

		if exists && prior.Status == result.StatusCancelled {
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
		return nil
	}); err != nil {
		return result.Result{}, err
	}

	if cancelResult.Status == "" {
		cancelResult = jobStatusResult(req, job)
	}
	if err := store.RemovePID(job.ID); err != nil {
		return cancelResult, fmt.Errorf("remove async job %q pid: %w", job.ID, err)
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

func (processController) TerminateAndWait(jobID string, worker jobs.PIDRecord, termGrace, killGrace time.Duration) error {
	if worker.PID <= 0 {
		return nil
	}

	termErr := jobs.SignalProcessTree(jobID, worker, jobs.TerminateSignal())
	if !jobs.ProcessTreeExists(jobID, worker) {
		return nil
	}
	if waitForProcessTreeExit(jobID, worker, termGrace) {
		return nil
	}

	killErr := jobs.SignalProcessTree(jobID, worker, jobs.KillSignal())
	if !jobs.ProcessTreeExists(jobID, worker) {
		return nil
	}
	if waitForProcessTreeExit(jobID, worker, killGrace) {
		return nil
	}
	if killErr != nil {
		return fmt.Errorf("terminate process tree %d: TERM: %v; KILL: %w", worker.PID, termErr, killErr)
	}
	return fmt.Errorf("terminate process tree %d: did not exit after TERM/KILL", worker.PID)
}

func waitForProcessTreeExit(jobID string, worker jobs.PIDRecord, timeout time.Duration) bool {
	if !jobs.ProcessTreeExists(jobID, worker) {
		return true
	}
	if timeout <= 0 {
		return !jobs.ProcessTreeExists(jobID, worker)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !jobs.ProcessTreeExists(jobID, worker) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !jobs.ProcessTreeExists(jobID, worker)
}
