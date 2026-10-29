package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/version"
)

func setResultJobMetadata(res *result.Result, job jobs.Job) {
	res.Metadata.StartedAt = &job.CreatedAt
	if res.Metadata.Agent == "" {
		res.Metadata.Agent = job.Spec.Agent
	}
	if res.Metadata.Model == "" {
		res.Metadata.Model = job.Spec.Model
	}
	if res.Metadata.Effort == "" {
		res.Metadata.Effort = job.Spec.Effort
	}
	if res.Metadata.AgentSession == "" {
		res.Metadata.AgentSession = job.Session()
	}
	if res.Metadata.IndexWarning == "" {
		res.Metadata.IndexWarning = job.IndexWarning
	}
	if isTerminalResultStatus(res.Status) && res.Metadata.FinishedAt == nil {
		finished := job.UpdatedAt
		if !jobs.IsTerminalStatus(job.Status) {
			finished = time.Now().UTC()
		}
		res.Metadata.FinishedAt = &finished
	}
}

func importArchivedJobMetadata(known []jobs.IndexedJob) ([]jobs.IndexedJob, error) {
	var imported []jobs.IndexedJob
	byID := map[string]int{}
	for i, entry := range known {
		if entry.ID != "" {
			byID[entry.ID] = i
		}
	}
	wrote := false
	err := jobs.VisitUnindexedArchives(known, func(data []byte, modified time.Time) error {
		var res result.Result
		if err := json.Unmarshal(data, &res); err != nil {
			return fmt.Errorf("decode archived job metadata: %w", err)
		}
		if !isTerminalResultStatus(res.Status) || res.Metadata.JobID == "" {
			return nil
		}
		if res.Metadata.FinishedAt == nil {
			res.Metadata.FinishedAt = &modified
		}
		entry := indexedEntryFromResult(res)
		prior, exists := byID[entry.ID]
		if exists && res.Metadata.StartedAt == nil {
			entry.StartedAt = known[prior].StartedAt
		}
		saved, err := jobs.WriteIndexedJob(entry)
		if err != nil {
			return fmt.Errorf("index archived job metadata: %w", err)
		}
		if exists {
			known[prior] = saved
		} else {
			imported = append(imported, saved)
		}
		wrote = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if wrote {
		if err := jobs.PruneJobIndex(); err != nil {
			return nil, err
		}
	}
	return imported, nil
}

func indexedEntryFromResult(res result.Result) jobs.IndexedJob {
	entry := jobs.IndexedJob{ID: res.Metadata.JobID, Agent: res.Metadata.Agent,
		Model: res.Metadata.Model, Effort: res.Metadata.Effort, CWD: res.Metadata.CWD,
		AgentSession: res.Metadata.AgentSession, Status: JobStatusFromResult(res.Status),
		Summary: res.Summary, Failure: res.Failure, FinishedAt: res.Metadata.FinishedAt,
		ExitCode: res.Metadata.ExitCode, StopReason: res.Metadata.StopReason, StopMessage: res.Metadata.StopMessage}
	if res.Metadata.StartedAt != nil {
		entry.StartedAt = *res.Metadata.StartedAt
	} else {
		entry.StartedAt, _ = time.Parse("20060102T150405Z", strings.SplitN(entry.ID, "-", 2)[0])
	}
	if entry.FinishedAt == nil && isTerminalResultStatus(res.Status) {
		finished := time.Now().UTC()
		entry.FinishedAt = &finished
	}
	return entry
}

func previousSessionJobs(agent, session, cwd string) ([]jobs.Job, error) {
	known, err := jobs.IndexedJobs()
	if err != nil {
		return nil, err
	}
	imported, err := importArchivedJobMetadata(known)
	if err != nil {
		return nil, err
	}
	matches, err := jobs.SessionJobsFromIndex(agent, session, cwd, append(known, imported...))
	if err != nil {
		return nil, err
	}
	for i, job := range matches {
		if job.IndexedOnly {
			continue
		}
		matches[i], err = reconcileWorker(jobs.NewStore(job.CWD), job)
		if err != nil {
			return nil, err
		}
	}
	return matches, nil
}

func readIndexedJobResult(req JobRequest) (result.Result, bool, error) {
	entry, found, err := jobs.ReadIndexedJob(req.JobID)
	if err != nil {
		return result.Result{}, false, fmt.Errorf("read indexed job %q: %w", req.JobID, err)
	}
	if !found || !sameJobDirectory(entry.CWD, req.CWD) {
		return result.Result{}, false, nil
	}
	entry = reconcileIndexedWorker(entry)
	summary := entry.Summary
	if summary == "" {
		summary = fmt.Sprintf("Last indexed state for job %s: %s", entry.ID, entry.Status)
	}
	return result.Result{Status: ResultStatusFromJob(string(entry.Status)), Failure: entry.Failure,
		Summary: summary, ChangedFiles: []string{}, Verification: []result.Verification{},
		Metadata: result.Metadata{Version: version.Version, JobID: entry.ID, Agent: entry.Agent,
			CWD: entry.CWD, Model: entry.Model, Effort: entry.Effort, AgentSession: entry.AgentSession,
			ExitCode: entry.ExitCode, StopReason: entry.StopReason, StopMessage: entry.StopMessage,
			StartedAt: &entry.StartedAt, FinishedAt: entry.FinishedAt,
			IndexedOnly: true, IndexWarning: jobs.IndexedOnlyWarning}}, true, nil
}

func indexedJobStatus(entry jobs.IndexedJob) result.JobStatus {
	return result.JobStatus{Version: version.Version, JobID: entry.ID, Agent: entry.Agent,
		Model: entry.Model, Effort: entry.Effort, CWD: entry.CWD,
		Status: ResultStatusFromJob(string(entry.Status)), AgentSession: entry.AgentSession,
		StartedAt: entry.StartedAt, FinishedAt: entry.FinishedAt, Summary: entry.Summary,
		Failure: entry.Failure, IndexedOnly: true, IndexWarning: jobs.IndexedOnlyWarning}
}

// Missing worktrees have no local terminal writer. Retained worker identity can
// still distinguish a live job from a process lost since the last snapshot.
func reconcileIndexedWorker(entry jobs.IndexedJob) jobs.IndexedJob {
	if entry.Status == jobs.StatusRunning && entry.Worker != nil && entry.Worker.Gone() {
		entry.Status = jobs.StatusInterrupted
		entry.Summary = result.RecoverySummary(result.StatusInterrupted, entry.AgentSession)
		entry.ExitCode = 1
		entry.StopReason = "worker_interrupted"
	}
	return entry
}
