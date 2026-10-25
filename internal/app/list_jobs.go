package app

import (
	"context"
	"sort"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// ListJobs includes all lifecycle states, newest first. --all broadens directory
// scope using the per-user index; it does not crawl unrelated filesystem trees.
func (s *Service) ListJobs(ctx context.Context, cwd string, all bool) ([]result.JobStatus, error) {
	dirs := []string{cwd}
	var requests []JobRequest
	var indexed []jobs.IndexedJob
	metadata := map[string]jobs.IndexedJob{}
	if all {
		var err error
		indexed, err = jobs.IndexedJobs()
		if err != nil {
			return nil, err
		}
		imported, err := importArchivedJobMetadata(indexed)
		if err != nil {
			return nil, err
		}
		indexed = append(indexed, imported...)
		for _, entry := range indexed {
			dirs = append(dirs, entry.CWD)
			if entry.ID != "" {
				requests = append(requests, JobRequest{CWD: entry.CWD, JobID: entry.ID})
				metadata[entry.ID] = entry
			}
		}
	}
	seen := map[string]bool{}
	seenJobs := map[JobRequest]bool{}
	for _, req := range requests {
		seenJobs[req] = true
	}
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		store := jobs.NewStore(dir)
		ids, err := store.ListIDs()
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			req := JobRequest{CWD: dir, JobID: id}
			if !seenJobs[req] {
				requests = append(requests, req)
				seenJobs[req] = true
			}
		}
	}
	records, err := s.jobStatuses(ctx, requests, true, metadata)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].StartedAt.Equal(records[j].StartedAt) {
			if records[i].JobID == records[j].JobID {
				return records[i].CWD < records[j].CWD
			}
			return records[i].JobID < records[j].JobID
		}
		return records[i].StartedAt.After(records[j].StartedAt)
	})
	return records, nil
}
