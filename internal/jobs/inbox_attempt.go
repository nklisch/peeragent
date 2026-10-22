package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type InboxAttemptStatus string

const (
	InboxAttemptRunning  InboxAttemptStatus = "running"
	InboxAttemptAccepted InboxAttemptStatus = "accepted"
	InboxAttemptReleased InboxAttemptStatus = "released"
)

// InboxAttempt identifies the worker that owns a carried prompt. Only that
// worker writes its terminal record; mailbox readers recover from that record
// or from its recorded process exiting, without signalling any process.
type InboxAttempt struct {
	ID   string `json:"id"`
	PID  int    `json:"pid"`
	Path string `json:"path"`
}
type inboxAttemptRecord struct {
	ID     string             `json:"id"`
	PID    int                `json:"pid"`
	Status InboxAttemptStatus `json:"status"`
}

type inboxEntryState struct {
	Read    bool          `json:"read,omitempty"`
	Attempt *InboxAttempt `json:"attempt,omitempty"`
}
type inboxState struct {
	Read    int                     `json:"read"`
	Entries map[int]inboxEntryState `json:"entries,omitempty"`
}

func BeginInboxAttempt(job Job) (InboxAttempt, error) {
	id, err := newID()
	if err != nil {
		return InboxAttempt{}, err
	}
	dir := filepath.Join(filepath.Dir(job.LogPath), "inbox-attempts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return InboxAttempt{}, err
	}
	attempt := InboxAttempt{ID: id, PID: os.Getpid(), Path: filepath.Join(dir, id+".json")}
	return attempt, writeInboxAttempt(attempt, InboxAttemptRunning)
}

func writeInboxAttempt(attempt InboxAttempt, status InboxAttemptStatus) error {
	data, err := json.Marshal(inboxAttemptRecord{attempt.ID, attempt.PID, status})
	if err != nil {
		return err
	}
	return AtomicWriteFile(attempt.Path, append(data, '\n'), 0o600)
}

func readInboxAttempt(attempt InboxAttempt) (InboxAttemptStatus, error) {
	data, err := os.ReadFile(attempt.Path)
	if os.IsNotExist(err) {
		return InboxAttemptRunning, nil
	}
	if err != nil {
		return "", fmt.Errorf("read inbox attempt %s: %w", attempt.ID, err)
	}
	var record inboxAttemptRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return "", fmt.Errorf("decode inbox attempt %s: %w", attempt.ID, err)
	}
	if record.ID != attempt.ID || record.PID != attempt.PID {
		return "", fmt.Errorf("inbox attempt %s record does not match reservation", attempt.ID)
	}
	switch record.Status {
	case InboxAttemptRunning, InboxAttemptAccepted, InboxAttemptReleased:
		return record.Status, nil
	default:
		return "", fmt.Errorf("unknown inbox attempt outcome %q", record.Status)
	}
}

func inboxAttemptOutcome(attempt InboxAttempt) (InboxAttemptStatus, error) {
	return resolveInboxAttempt(attempt, ProcessExists)
}

func resolveInboxAttempt(attempt InboxAttempt, processExists func(int) bool) (InboxAttemptStatus, error) {
	outcome, err := readInboxAttempt(attempt)
	if err != nil || outcome != InboxAttemptRunning {
		return outcome, err
	}
	if processExists(attempt.PID) {
		return InboxAttemptRunning, nil
	}
	// The worker may have published acceptance between the first read and exit.
	// Re-read after observing death so that terminal publication always wins.
	outcome, err = readInboxAttempt(attempt)
	if err != nil || outcome != InboxAttemptRunning {
		return outcome, err
	}
	return InboxAttemptReleased, nil
}

// FinishInboxAttempt records the outcome before any source mailbox is settled.
// That makes a crash halfway through settlement recoverable on the next read.
func FinishInboxAttempt(attempt InboxAttempt, accepted bool) (InboxAttemptStatus, error) {
	outcome, err := inboxAttemptOutcome(attempt)
	if err != nil {
		return "", err
	}
	if outcome != InboxAttemptRunning {
		return outcome, nil
	}
	outcome = InboxAttemptReleased
	if accepted {
		outcome = InboxAttemptAccepted
	}
	return outcome, writeInboxAttempt(attempt, outcome)
}

func (s Store) readInboxStateLocked(id string) (inboxState, error) {
	dir, err := s.jobDir(id)
	if err != nil {
		return inboxState{}, err
	}
	state := inboxState{Entries: map[int]inboxEntryState{}}
	data, err := os.ReadFile(filepath.Join(dir, "inbox-state.json"))
	if err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return state, fmt.Errorf("decode inbox state: %w", err)
		}
		if state.Entries == nil {
			state.Entries = map[int]inboxEntryState{}
		}
	} else if os.IsNotExist(err) {
		// Read older cursor-only mailboxes, then migrate on the next state change.
		data, err = os.ReadFile(filepath.Join(dir, "inbox-read"))
		if err != nil && !os.IsNotExist(err) {
			return state, fmt.Errorf("read inbox cursor: %w", err)
		}
		if err == nil {
			state.Read, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				return state, fmt.Errorf("invalid inbox cursor %q", data)
			}
		}
	} else {
		return state, fmt.Errorf("read inbox state: %w", err)
	}
	if state.Read < 0 {
		return state, fmt.Errorf("negative inbox read cursor")
	}
	return state, nil
}

func (s Store) writeInboxStateLocked(id string, state inboxState) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(dir, "inbox-state.json"), append(data, '\n'), 0o600)
}

func (state *inboxState) advanceRead() {
	for state.Entries[state.Read].Read {
		delete(state.Entries, state.Read)
		state.Read++
	}
}

// recoverInboxLocked settles each distinct attempt once. Live reservations
// stay hidden. A terminal acceptance commits them even if its worker is dead;
// an abandoned attempt without an outcome returns them to unread.
func (s Store) recoverInboxLocked(id string, state *inboxState) error {
	outcomes := map[string]InboxAttemptStatus{}
	changed := false
	for index, entry := range state.Entries {
		if entry.Attempt == nil {
			continue
		}
		attempt := *entry.Attempt
		outcome, known := outcomes[attempt.ID]
		if !known {
			var err error
			outcome, err = inboxAttemptOutcome(attempt)
			if err != nil {
				return err
			}
			outcomes[attempt.ID] = outcome
		}
		switch outcome {
		case InboxAttemptAccepted:
			state.Entries[index] = inboxEntryState{Read: true}
			changed = true
		case InboxAttemptReleased:
			delete(state.Entries, index)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	state.advanceRead()
	return s.writeInboxStateLocked(id, *state)
}

// ReserveInbox snapshots and reserves under the writer/reader lock. The
// returned messages are delivered only in the prompt until this attempt ends.
func ReserveInbox(ctx context.Context, store Store, id string, attempt InboxAttempt) ([]Message, error) {
	var entries []Message
	err := store.WithJobLock(id, func() error {
		mailbox, err := MailboxSnapshotLocked(ctx, store, id)
		if err != nil {
			return err
		}
		entries = mailbox.UnreadMessages()
		if len(entries) == 0 {
			return nil
		}
		for _, index := range mailbox.unread {
			mailbox.state.Entries[index] = inboxEntryState{Attempt: &attempt}
		}
		return store.writeInboxStateLocked(id, mailbox.state)
	})
	return entries, err
}

// SettleInboxAttempt updates only entry state, not another history snapshot.
func (s Store) SettleInboxAttempt(id, attemptID string, outcome InboxAttemptStatus) error {
	return s.WithJobLock(id, func() error {
		state, err := s.readInboxStateLocked(id)
		if err != nil {
			return err
		}
		changed := false
		for index, entry := range state.Entries {
			if entry.Attempt == nil || entry.Attempt.ID != attemptID {
				continue
			}
			if outcome == InboxAttemptAccepted {
				state.Entries[index] = inboxEntryState{Read: true}
			} else {
				delete(state.Entries, index)
			}
			changed = true
		}
		if !changed {
			return nil
		}
		state.advanceRead()
		return s.writeInboxStateLocked(id, state)
	})
}
