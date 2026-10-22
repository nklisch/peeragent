package app

import (
	"fmt"

	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// protectSession prevents lost/interleaved work from two agents writing the
// same session. A terminal job or a recorded process that has exited is harmless.
// --allow-concurrent-session is the explicit escape hatch.
func protectSession(delegation input.Delegation, launch func([]jobs.Job) (result.Result, error)) (result.Result, error) {
	if delegation.Resume == "" {
		return launch(nil)
	}
	if delegation.AllowConcurrentSession {
		matches, err := jobs.SessionJobs(delegation.Agent, delegation.Resume, delegation.CWD)
		if err != nil {
			return failedDelegationResult(delegation, err), err
		}
		return launch(matches)
	}
	var res result.Result
	conflict := false
	err := jobs.WithSessionLaunchLock(delegation.Agent, delegation.Resume, func() error {
		matches, err := jobs.SessionJobs(delegation.Agent, delegation.Resume, delegation.CWD)
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
