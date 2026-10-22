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
	if n := m.UnreadCount(); n > 0 {
		fmt.Fprintf(os.Stderr, "peeragent: %d unread message(s): run peeragent inbox\n", n)
	}
	return nil
}

func runMessageCommand(ctx context.Context, req input.Request) error {
	if req.MessageCommand == "tell" {
		from := os.Getenv("HERDR_PANE_ID")
		if from == "" {
			from = os.Getenv("PEERAGENT_JOB_ID")
		}
		return applicationService.Tell(ctx, app.JobRequest{CWD: req.CWD, JobID: req.MessageJobID}, from, req.TaskText)
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
		return applicationService.Say(ctx, app.JobRequest{CWD: job.CWD, JobID: id}, req.TaskText)
	}
	return store.ReadInboxContext(ctx, id, func(entries []jobs.Message) error {
		encoder := json.NewEncoder(os.Stdout)
		for _, entry := range entries {
			if err := encoder.Encode(entry); err != nil {
				return err
			}
		}
		return nil
	})
}

// Commands that read their current job emit the notice from that locked
// snapshot. Other commands emit it before handling the request. The fallback
// covers a lookup error before the handler reaches its mailbox read.
type commandNoticeKey struct{}

func ensureCommandNotice(ctx context.Context) error {
	if ensure, ok := ctx.Value(commandNoticeKey{}).(func() error); ok {
		return ensure()
	}
	return nil
}

func commandMessaging(ctx context.Context) (context.Context, func() error, error) {
	if os.Getenv("PEERAGENT_JOB_ID") == "" && os.Getenv("PEERAGENT_JOB_DIR") == "" {
		return jobs.WithMailboxCommand(ctx, nil), func() error { return nil }, nil
	}
	store, id, err := currentJob()
	if err != nil {
		return ctx, nil, err
	}
	seen := false
	ctx = jobs.WithMailboxCommand(ctx, func(jobID string, m jobs.Mailbox) {
		if jobID != id || seen {
			return
		}
		seen = true
		if n := m.UnreadCount(); n > 0 {
			fmt.Fprintf(os.Stderr, "peeragent: %d unread message(s): run peeragent inbox\n", n)
		}
	})
	ensure := func() error {
		if seen {
			return nil
		}
		_, err := jobs.MailboxSnapshot(ctx, store, id)
		return err
	}
	ctx = context.WithValue(ctx, commandNoticeKey{}, ensure)
	return ctx, ensure, nil
}

func commandReadsCurrentMailbox(req input.Request) bool {
	id := os.Getenv("PEERAGENT_JOB_ID")
	if id == "" {
		return false
	}
	if req.MessageCommand == "say" || req.MessageCommand == "inbox" {
		return true
	}
	for _, candidate := range []string{req.StatusJobID, req.ResultJobID, req.WaitJobID} {
		if candidate == id {
			return true
		}
	}
	for _, refs := range [][]input.JobReference{req.StatusJobs, req.WaitAnyJobs} {
		for _, ref := range refs {
			if ref.JobID == id {
				return true
			}
		}
	}
	return false
}
