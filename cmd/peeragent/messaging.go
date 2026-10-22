package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
)

func currentJob() (jobs.Store, string, error) {
	id, dir := os.Getenv("PEERAGENT_JOB_ID"), os.Getenv("PEERAGENT_JOB_DIR")
	if id == "" || dir == "" {
		return jobs.Store{}, "", fmt.Errorf("no current peeragent job; use your handback instead")
	}
	if err := jobs.ValidateID(id); err != nil {
		return jobs.Store{}, "", err
	}
	if !filepath.IsAbs(dir) || filepath.Base(dir) != id {
		return jobs.Store{}, "", fmt.Errorf("invalid PEERAGENT_JOB_DIR; use your handback instead")
	}
	store := jobs.Store{Root: filepath.Dir(dir)}
	if _, err := store.Load(id); err != nil {
		return store, "", fmt.Errorf("load current job: %w", err)
	}
	return store, id, nil
}

func printUnreadNotice() error {
	if os.Getenv("PEERAGENT_JOB_ID") == "" && os.Getenv("PEERAGENT_JOB_DIR") == "" {
		return nil
	}
	store, id, err := currentJob()
	if err != nil {
		return err
	}
	m, err := store.Mailbox(id)
	if err != nil {
		return fmt.Errorf("read current job inbox: %w", err)
	}
	if n := len(m.Inbox) - m.Read; n > 0 {
		fmt.Fprintf(os.Stderr, "peeragent: %d unread message(s): run peeragent inbox\n", n)
	}
	return nil
}

func runMessageCommand(req input.Request) error {
	if req.MessageCommand == "tell" {
		from := os.Getenv("HERDR_PANE_ID")
		if from == "" {
			from = os.Getenv("PEERAGENT_JOB_ID")
		}
		return applicationService.Tell(context.Background(), app.JobRequest{CWD: req.CWD, JobID: req.MessageJobID}, from, req.TaskText)
	}
	store, id, err := currentJob()
	if err != nil {
		return err
	}
	if req.MessageCommand == "say" {
		job, err := store.Load(id)
		if err != nil {
			return err
		}
		return applicationService.Say(context.Background(), app.JobRequest{CWD: job.CWD, JobID: id}, req.TaskText)
	}
	return store.ReadInbox(id, func(entries []jobs.Message) error {
		encoder := json.NewEncoder(os.Stdout)
		for _, entry := range entries {
			if err := encoder.Encode(entry); err != nil {
				return err
			}
		}
		return nil
	})
}
