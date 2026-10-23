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
	if all {
		indexed, err := jobs.IndexedDirectories()
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, indexed...)
	}
	seen := map[string]bool{}
	var requests []JobRequest
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
			requests = append(requests, JobRequest{CWD: dir, JobID: id})
		}
	}
	records, err := s.jobStatuses(ctx, requests, true)
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
