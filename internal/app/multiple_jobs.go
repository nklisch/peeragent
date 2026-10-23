package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/version"
)

// ResolveJobs validates the whole wait set before any terminal result can win.
// The returned directories avoid repeated index lookup on each wait iteration.
func (s *Service) ResolveJobs(ctx context.Context, requests []JobRequest) ([]JobRequest, error) {
	resolved := make([]JobRequest, 0, len(requests))
	for _, raw := range requests {
		req, err := s.normalizeJobRequest(ctx, raw)
		if err != nil {
			return nil, err
		}
		if _, err := jobs.NewStore(req.CWD).Load(req.JobID); err != nil {
			return nil, fmt.Errorf("load async job %q: %w", req.JobID, err)
		}
		resolved = append(resolved, req)
	}
	return resolved, nil
}

// JobStatuses loads each job once and reads HEAD once per distinct directory.
// Missing git or a non-repository directory simply omits the optional commit.
func (s *Service) JobStatuses(ctx context.Context, requests []JobRequest) ([]result.JobStatus, error) {
	return s.jobStatuses(ctx, requests, false)
}

func (s *Service) jobStatuses(ctx context.Context, requests []JobRequest, skipMissing bool) ([]result.JobStatus, error) {
	records := make([]result.JobStatus, 0, len(requests))
	heads := map[string]string{}
	for _, raw := range requests {
		req, err := s.normalizeJobRequest(ctx, raw)
		if err != nil {
			return nil, err
		}
		job, err := jobs.NewStore(req.CWD).Load(req.JobID)
		if err != nil {
			if skipMissing && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("load async job %q: %w", req.JobID, err)
		}
		head, known := heads[req.CWD]
		if !known {
			out, err := exec.CommandContext(ctx, "git", "-C", req.CWD, "rev-parse", "--verify", "HEAD").Output()
			if err == nil {
				head = strings.TrimSpace(string(out))
			}
			heads[req.CWD] = head
		}
		progress, err := jobs.ReadProgress(job.LogPath)
		if err != nil {
			return nil, fmt.Errorf("read job %s progress: %w", job.ID, err)
		}
		summary := ""
		if job.Status == jobs.StatusCapacity {
			summary = result.CapacitySummary(job.Session())
		}
		if job.Status == jobs.StatusOutputLimit {
			summary = result.OutputLimitSummary(job.Session())
		}
		mailbox, err := jobs.MailboxSnapshot(ctx, jobs.NewStore(req.CWD), job.ID)
		if err != nil {
			return nil, fmt.Errorf("read job %s mailbox: %w", job.ID, err)
		}
		support, err := jobs.NewStore(req.CWD).MessagingSupport(job, mailbox)
		if err != nil {
			return nil, err
		}
		records = append(records, result.JobStatus{
			MessageStats: mailbox.Stats(),
			Version:      version.Version, Agent: job.Spec.Agent, MessagingEnabled: support.Enabled, MessagingSource: support.Source, MessagingWarning: support.Warning,
			KeepTerminalEnv: job.Spec.KeepTerminalEnv,
			LastEventAt:     progress.At, LastMessage: progress.LastMessage, Summary: summary,
			JobID: job.ID, CWD: req.CWD, Status: ResultStatusFromJob(string(job.Status)),
			Model: job.Spec.Model, Effort: job.Spec.Effort, AgentSession: job.Session(),
			StartedAt: job.CreatedAt, CapacityBackoff: !jobs.IsTerminalStatus(job.Status) && job.CapacityBackoffUntil != nil && time.Now().Before(*job.CapacityBackoffUntil),
			HeadCommit: head, FinalReceived: job.FinalReceived && !jobs.IsTerminalStatus(job.Status),
		})
	}
	return records, nil
}
