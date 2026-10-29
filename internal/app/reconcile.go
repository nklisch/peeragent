package app

import (
	"fmt"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
)

// Reconcile only after positive evidence of death. The second read under the
// writer lock lets a natural terminal publication win a concurrent observation.
func reconcileWorker(store jobs.Store, job jobs.Job) (jobs.Job, error) {
	if job.Status != jobs.StatusRunning {
		return job, nil
	}
	gone, err := store.WorkerGone(job)
	if err != nil {
		return job, fmt.Errorf("check job %s worker: %w", job.ID, err)
	}
	if !gone {
		return job, nil
	}
	err = store.WithJobLock(job.ID, func() error {
		current, err := store.Load(job.ID)
		if err != nil {
			return err
		}
		job = current
		if current.Status != jobs.StatusRunning {
			return nil
		}
		gone, err := store.WorkerGone(current)
		if err != nil || !gone {
			return err
		}
		prior, exists, err := readStoredResult(current.ResultPath)
		if err != nil {
			return err
		}
		if exists && isTerminalResultStatus(prior.Status) {
			current.Status = JobStatusFromResult(prior.Status)
		} else {
			current.Status = jobs.StatusInterrupted
			current.UpdatedAt = time.Now().UTC()
			current.CapacityBackoffUntil = nil
			current.FinalReceived = false
			res := jobStatusResult(JobRequest{CWD: current.CWD, JobID: current.ID}, current)
			res.Metadata.ExitCode = 1
			res.Metadata.StopReason = "worker_interrupted"
			res.Metadata.StopMessage = "recorded worker exited or belongs to an earlier boot or process"
			if err := enrichJobResult(&res, current); err != nil {
				return err
			}
			if err := addProgress(&res, current); err != nil {
				return err
			}
			// The last message remains progress, never an invented final handback.
			res.FinalMessage = nil
			if err := WriteJobResult(current.ResultPath, res); err != nil {
				return err
			}
		}
		if err := store.Save(current); err != nil {
			return err
		}
		job = current
		return store.RemovePID(job.ID)
	})
	return job, err
}
