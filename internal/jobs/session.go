package jobs

import (
	"fmt"
	"os"
	"path/filepath"
)

// WithSessionLaunchLock serializes check/start, not the lifetime of an agent.
// The override can bypass this lock too if an interrupted launcher left it behind.
func WithSessionLaunchLock(agent, session string, fn func() error) error {
	path, err := indexPath("session-launch-locks", agent+"\x00"+session)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return withFileLock(path, fn)
}

// RunningSessionJob checks known worktrees, including older jobs that predate
// live session indexing. Each known directory is scanned once per resume;
// metadata-only entries can restore options but cannot prove a live process.
func RunningSessionJob(agent, session, cwd string) (Job, bool, error) {
	matches, err := SessionJobs(agent, session, cwd)
	if err != nil {
		return Job{}, false, err
	}
	return LiveSessionJob(matches)
}

func LiveSessionJob(matches []Job) (Job, bool, error) {
	for _, job := range matches {
		if job.IndexedOnly {
			continue
		}
		if IsTerminalStatus(job.Status) {
			continue
		}
		if job.Worker != nil {
			if job.Worker.PID > 1 && !job.Worker.Gone() {
				return job, true, nil
			}
			continue
		}
		store := NewStore(job.CWD)
		pid, err := store.ReadPID(job.ID)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Job{}, false, err
		}
		if ProcessExists(pid) {
			return job, true, nil
		}
	}
	return Job{}, false, nil
}

// SessionJobs locates previous jobs for both resume protection and inbox carryover.
func SessionJobs(agent, session, cwd string) ([]Job, error) {
	entries, err := IndexedJobs()
	if err != nil {
		return nil, err
	}
	return SessionJobsFromIndex(agent, session, cwd, entries)
}

// SessionJobsFromIndex lets the application reuse entries already read while
// promoting legacy archives. Local files still win over the supplied snapshots.
func SessionJobsFromIndex(agent, session, cwd string, entries []IndexedJob) ([]Job, error) {
	dirs := map[string]bool{cwd: true}
	indexed, err := readIndex("sessions", agent+"\x00"+session)
	if err != nil {
		return nil, fmt.Errorf("lookup session directory: %w", err)
	}
	if indexed != "" {
		dirs[indexed] = true
	}
	for _, entry := range entries {
		dirs[entry.CWD] = true
	}
	var matches []Job
	seen := map[string]bool{}
	for dir := range dirs {
		store := NewStore(dir)
		ids, err := store.ListIDs()
		if err != nil {
			return nil, fmt.Errorf("read jobs in %s: %w", dir, err)
		}
		for _, id := range ids {
			job, err := store.Load(id)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("load job %s: %w", id, err)
			}
			if job.Spec.Agent == agent && job.Session() == session && !seen[job.ID] {
				matches = append(matches, job)
			}
			seen[job.ID] = true
		}
	}
	for _, entry := range entries {
		if entry.ID != "" && !seen[entry.ID] && entry.Agent == agent && entry.AgentSession == session {
			matches = append(matches, Job{ID: entry.ID, IndexedOnly: true, CWD: entry.CWD,
				Status: entry.Status, CreatedAt: entry.StartedAt, AgentSession: entry.AgentSession,
				Spec: ExecSpec{Agent: entry.Agent, Model: entry.Model, Effort: entry.Effort, Resume: entry.AgentSession}})
		}
	}
	return matches, nil
}
