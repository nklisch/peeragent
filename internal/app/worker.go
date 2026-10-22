package app

import (
	"context"
	"fmt"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// WorkerExited runs only after the owned handle has been waited on. Natural
// terminal results and cancellation keep their precedence. Without one, exit
// is the signal to record failure; no deadline or process-name search is used.
func (s *Service) WorkerExited(ctx context.Context, raw JobRequest, code int, waitErr error) (result.Result, error) {
	req, err := s.normalizeJobRequest(ctx, raw)
	if err != nil {
		return result.Result{}, err
	}
	store := jobs.NewStore(req.CWD)
	job, err := store.Load(req.JobID)
	if err != nil {
		return result.Result{}, fmt.Errorf("load exited worker job %s: %w", req.JobID, err)
	}
	res, exists, err := readStoredResult(job.ResultPath)
	if err != nil {
		return result.Result{}, err
	}
	if exists && isTerminalResultStatus(res.Status) && jobs.IsTerminalStatus(job.Status) {
		if err := store.RemovePID(job.ID); err != nil {
			return result.Result{}, err
		}
		if err := enrichJobResult(&res, job); err != nil {
			return result.Result{}, err
		}
		return res, nil
	}
	if !exists && jobs.IsTerminalStatus(job.Status) {
		if err := store.RemovePID(job.ID); err != nil {
			return result.Result{}, err
		}
		return jobStatusResult(req, job), nil
	}
	if !exists || !isTerminalResultStatus(res.Status) {
		res = result.Result{
			Status:       result.StatusFailed,
			Summary:      fmt.Sprintf("job worker exited with status %d before writing a terminal result; inspect %s", code, job.LogPath),
			ChangedFiles: []string{},
			Verification: []result.Verification{},
			Metadata: result.Metadata{
				JobID:           job.ID,
				CWD:             job.CWD,
				Agent:           job.Spec.Agent,
				Access:          job.Spec.Access,
				Profile:         job.Spec.Profile,
				Effort:          job.Spec.Effort,
				Model:           job.Spec.Model,
				AgentSession:    job.Spec.Resume,
				CapacityRetries: job.CapacityRetries,
				ExitCode:        code,
				LogPath:         job.LogPath,
			},
		}
		if waitErr != nil {
			res.Details = "wait for job worker: " + waitErr.Error()
		}
	}
	if err := enrichJobResult(&res, job); err != nil {
		return result.Result{}, err
	}
	if err := FinishJob(store, job, res); err != nil {
		return result.Result{}, err
	}
	// Re-read the guarded winner: cancellation may have committed during exit.
	return s.JobResult(ctx, req)
}
