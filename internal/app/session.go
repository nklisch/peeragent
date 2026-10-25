package app

import (
	"fmt"

	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// resumeOptions reuses session discovery from the launch guard. Creation time
// orders turns; progress updates on an older turn must not make it the latest.
func resumeOptions(delegation input.Delegation, previous []jobs.Job) (input.Delegation, string, error) {
	if delegation.Resume == "" || !delegation.InheritEffort && !delegation.InheritModel {
		return delegation, "", nil
	}
	if len(previous) == 0 {
		return delegation, "previous session job not found; omitted --model/--effort use current defaults", nil
	}
	latest := previous[0]
	for _, job := range previous[1:] {
		if job.CreatedAt.After(latest.CreatedAt) || job.CreatedAt.Equal(latest.CreatedAt) && job.ID > latest.ID {
			latest = job
		}
	}
	if delegation.InheritEffort {
		delegation.Effort = latest.Spec.Effort
	}
	if delegation.InheritModel {
		delegation.Model = latest.Spec.Model
	}
	model, effort, err := input.NormalizeModelAndEffort(delegation.Agent, delegation.Model, delegation.Effort)
	if err != nil {
		return delegation, "", fmt.Errorf("resolve resumed model and effort: %w", err)
	}
	delegation.Effort, delegation.Model = effort, model
	delegation.InheritEffort, delegation.InheritModel = false, false
	return delegation, "", nil
}

// protectSession prevents lost/interleaved work from two agents writing the
// same session. A terminal job or a recorded process that has exited is harmless.
// --allow-concurrent-session is the explicit escape hatch.
func protectSession(delegation input.Delegation, launch func([]jobs.Job) (result.Result, error)) (result.Result, error) {
	if delegation.Resume == "" {
		return launch(nil)
	}
	if delegation.AllowConcurrentSession {
		matches, err := previousSessionJobs(delegation.Agent, delegation.Resume, delegation.CWD)
		if err != nil {
			return failedDelegationResult(delegation, err), err
		}
		return launch(matches)
	}
	var res result.Result
	conflict := false
	err := jobs.WithSessionLaunchLock(delegation.Agent, delegation.Resume, func() error {
		matches, err := previousSessionJobs(delegation.Agent, delegation.Resume, delegation.CWD)
		if err != nil {
			return err
		}
		job, found, err := jobs.LiveSessionJob(matches)
		if err != nil {
			return err
		}
		if found {
			conflict = true
			return fmt.Errorf("session %s already has running job %s in %q; wait with --wait %s --cwd %q or cancel with --cancel %s --cwd %q; override with --allow-concurrent-session", delegation.Resume, job.ID, job.CWD, job.ID, job.CWD, job.ID, job.CWD)
		}
		res, err = launch(matches)
		return err
	})
	if err != nil && res.Status == "" {
		err = fmt.Errorf("protect resumed session (override with --allow-concurrent-session): %w", err)
		res = failedDelegationResult(delegation, err)
		if conflict {
			res.Metadata.ExitCode = 2
		}
	}
	return res, err
}
