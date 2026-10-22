package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Message is a mailbox entry. Outbound entries also record delivery/refusal;
// pending reserves a rate-limit slot while delivery runs outside the job lock.
type Message struct {
	ID                string    `json:"id,omitempty"`
	Time              time.Time `json:"time"`
	From              string    `json:"from"`
	Text              string    `json:"text"`
	Direction         string    `json:"direction,omitempty"`
	Status            string    `json:"status,omitempty"`
	DeliveryAttempted bool      `json:"delivery_attempted,omitempty"`
	Reason            string    `json:"reason,omitempty"`
}

type MessageStats struct {
	InboxUnread   int        `json:"inbox_unread"`
	OutboxSent    int        `json:"outbox_sent"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
}

type Mailbox struct {
	Inbox  []Message
	Outbox []Message
	Read   int
}

func (m Mailbox) Stats() MessageStats {
	stats := MessageStats{InboxUnread: len(m.Inbox) - m.Read}
	for _, entries := range [][]Message{m.Inbox, m.Outbox} {
		for _, entry := range entries {
			if entry.Status == "sent" {
				stats.OutboxSent++
			}
			if stats.LastMessageAt == nil || entry.Time.After(*stats.LastMessageAt) {
				at := entry.Time
				stats.LastMessageAt = &at
			}
		}
	}
	return stats
}

func (m Mailbox) Conversation() []Message {
	entries := make([]Message, 0, len(m.Inbox)+len(m.Outbox))
	for _, entry := range m.Inbox {
		entry.Direction = "inbox"
		entries = append(entries, entry)
	}
	for _, entry := range m.Outbox {
		entry.Direction = "outbox"
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Time.Before(entries[j].Time) })
	return entries
}

// ReadMailboxLocked must be called under WithJobLock, including read-only
// snapshots, so readers never observe a partial append or a mismatched cursor.
func (s Store) ReadMailboxLocked(id string) (Mailbox, error) {
	dir, err := s.jobDir(id)
	if err != nil {
		return Mailbox{}, err
	}
	var m Mailbox
	for _, file := range []struct {
		name    string
		entries *[]Message
	}{{"inbox.jsonl", &m.Inbox}, {"outbox.jsonl", &m.Outbox}} {
		data, err := os.ReadFile(filepath.Join(dir, file.name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return m, fmt.Errorf("read %s: %w", file.name, err)
		}
		positions := map[string]int{}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var entry Message
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				return m, fmt.Errorf("decode %s: %w", file.name, err)
			}
			if file.name == "outbox.jsonl" && entry.ID != "" {
				if position, found := positions[entry.ID]; found {
					(*file.entries)[position] = entry
					continue
				}
				positions[entry.ID] = len(*file.entries)
			}
			*file.entries = append(*file.entries, entry)
		}
	}
	m.Read, err = s.ReadInboxCursorLocked(id)
	if err != nil {
		return m, err
	}
	if m.Read < 0 || m.Read > len(m.Inbox) {
		return m, fmt.Errorf("inbox cursor %d outside mailbox of %d", m.Read, len(m.Inbox))
	}
	return m, nil
}

// ReadInboxCursorLocked reads only the acknowledgement, without rescanning history.
func (s Store) ReadInboxCursorLocked(id string) (int, error) {
	dir, err := s.jobDir(id)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "inbox-read"))
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read inbox cursor: %w", err)
	}
	read, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || read < 0 {
		return 0, fmt.Errorf("invalid inbox cursor %q", data)
	}
	return read, nil
}

// Mailbox snapshots a known job. Callers resolve the job before this read.
func (s Store) Mailbox(id string) (Mailbox, error) {
	var m Mailbox
	err := s.WithJobLock(id, func() error {
		var err error
		m, err = s.ReadMailboxLocked(id)
		return err
	})
	return m, err
}

func (s Store) AppendMessageLocked(id, box string, entry Message) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, box+".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", box, err)
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("append %s: %w", box, writeErr)
	}
	return closeErr
}

func (s Store) MarkInboxReadLocked(id string, count int) error {
	dir, err := s.jobDir(id)
	if err != nil {
		return err
	}
	return AtomicWriteFile(filepath.Join(dir, "inbox-read"), []byte(strconv.Itoa(count)+"\n"), 0o600)
}

// ReadInbox holds the same lock as tell through delivery and acknowledgement.
// A failed output leaves the cursor unchanged; a concurrent append is either
// delivered in this batch or stays unread for the next call.
func (s Store) ReadInbox(id string, deliver func([]Message) error) error {
	return s.ReadInboxContext(context.Background(), id, deliver)
}

func (s Store) ReadInboxContext(ctx context.Context, id string, deliver func([]Message) error) error {
	return s.WithJobLock(id, func() error {
		if _, err := s.Load(id); err != nil {
			return err
		}
		m, err := MailboxSnapshotLocked(ctx, s, id)
		if err != nil {
			return err
		}
		if err := deliver(m.Inbox[m.Read:]); err != nil {
			return err
		}
		return s.MarkInboxReadLocked(id, len(m.Inbox))
	})
}
