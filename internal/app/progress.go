package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		if progress.Event != "" || progress.Message != "" {
			preview := progress.LastMessage
			if preview == "" && progress.Message == "" && job.MessagePath != "" {
				previous, err := jobs.ReadProgress(job.LogPath)
				if err != nil {
					return fmt.Errorf("read previous job progress: %w", err)
				}
				preview = previous.LastMessage
			}
			if preview == "" && progress.Message != "" {
				preview = executil.MessagePreview(progress.Message)
			}
			now := time.Now().UTC()
			if err := jobs.AppendProgress(job.LogPath, jobs.Progress{At: &now, Event: progress.Event, LastMessage: preview}); err != nil {
				return fmt.Errorf("append job %s progress: %w", id, err)
			}
		}
		changed := progress.AgentSession != "" && progress.AgentSession != job.Spec.Resume || progress.Message != "" || job.FinalReceived != progress.FinalReceived
		if !changed {
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
	res.Metadata.KeepTerminalEnv = job.Spec.KeepTerminalEnv
	if res.Metadata.AgentSession == "" {
		res.Metadata.AgentSession = job.Spec.Resume
	}
	res.Metadata.MessagePath = job.MessagePath
	res.Metadata.FinalReceived = job.FinalReceived
	if res.Status == result.StatusCapacity {
		res.Summary = result.CapacitySummary(res.Metadata.AgentSession)
	}
	if res.Details == "" && job.MessagePath != "" {
		text, err := os.ReadFile(job.MessagePath)
		if err != nil {
			return err
		}
		res.Details = executionDetails(string(text), "")
	}
	return nil
}

func addProgress(res *result.Result, job jobs.Job) error {
	progress, err := jobs.ReadProgress(job.LogPath)
	if err != nil {
		return fmt.Errorf("read job %s progress: %w", job.ID, err)
	}
	res.Metadata.LastEventAt = progress.At
	res.Metadata.LastMessage = progress.LastMessage
	return nil
}
