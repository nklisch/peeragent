package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// live session indexing. Index entries contain directories only; each directory
// is scanned once per resume, never as part of status polling.
func RunningSessionJob(agent, session, cwd string) (Job, bool, error) {
	matches, err := SessionJobs(agent, session, cwd)
	if err != nil {
		return Job{}, false, err
	}
	return LiveSessionJob(matches)
}

func LiveSessionJob(matches []Job) (Job, bool, error) {
	for _, job := range matches {
		if IsTerminalStatus(job.Status) {
			continue
		}
		pid, err := NewStore(job.CWD).ReadPID(job.ID)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Job{}, false, fmt.Errorf("read job %s pid: %w", job.ID, err)
		}
		if ProcessExists(pid) {
			return job, true, nil
		}
	}
	return Job{}, false, nil
}

// SessionJobs locates previous jobs for both resume protection and inbox carryover.
func SessionJobs(agent, session, cwd string) ([]Job, error) {
	dirs := map[string]bool{cwd: true}
	indexed, err := LookupSession(agent, session)
	if err != nil {
		return nil, fmt.Errorf("lookup session directory: %w", err)
	}
	if indexed != "" {
		dirs[indexed] = true
	}
	path, err := indexPath("jobs", "")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read job index: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(filepath.Dir(path), entry.Name()))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read job index entry: %w", err)
		}
		if len(data) > 0 {
			dirs[string(data)] = true
		}
	}
	var matches []Job
	seen := map[string]bool{}
	for dir := range dirs {
		store := NewStore(dir)
		entries, err := os.ReadDir(store.Root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read jobs in %s: %w", dir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() || ValidateID(entry.Name()) != nil {
				continue
			}
			job, err := store.Load(entry.Name())
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("load job %s: %w", entry.Name(), err)
			}
			if job.Spec.Agent == agent && job.Spec.Resume == session && !seen[job.ID] {
				seen[job.ID] = true
				matches = append(matches, job)
			}
		}
	}
	return matches, nil
}
