package app

import (
	"os"
	"path/filepath"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// RecordJobProgress shares cancellation's lock, so an observation can never
// overwrite a terminal winner. Messages live separately to keep status cheap.
func RecordJobProgress(store jobs.Store, id string, progress executil.Progress) error {
	return store.WithJobLock(id, func() error {
		job, err := store.Load(id)
		if err != nil {
			return err
		}
		if jobs.IsTerminalStatus(job.Status) {
			return nil
		}
		if progress.AgentSession != "" {
			job.Spec.Resume = progress.AgentSession
		}
		if progress.Message != "" {
			job.MessagePath = filepath.Join(filepath.Dir(job.LogPath), "message.txt")
			if err := jobs.AtomicWriteFile(job.MessagePath, []byte(progress.Message), 0o644); err != nil {
				return err
			}
		}
		job.FinalReceived = progress.FinalReceived
		return store.Save(job)
	})
}

func enrichJobResult(res *result.Result, job jobs.Job) error {
	if res.Metadata.AgentSession == "" {
		res.Metadata.AgentSession = job.Spec.Resume
	}
	res.Metadata.MessagePath = job.MessagePath
	res.Metadata.FinalReceived = job.FinalReceived
	if res.Details == "" && job.MessagePath != "" {
		text, err := os.ReadFile(job.MessagePath)
		if err != nil {
			return err
		}
		res.Details = executionDetails(string(text), "")
	}
	return nil
}
