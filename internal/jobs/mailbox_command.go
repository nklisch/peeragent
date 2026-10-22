package jobs

import (
	"context"
	"sync"
)

type mailboxCommandKey struct{}
type mailboxCommand struct {
	mu        sync.Mutex
	snapshots map[string]Mailbox
	observers []func(string, Mailbox)
}

// WithMailboxCommand scopes snapshots to one CLI command (or one target turn).
// The observer sees each job once, before any output based on that snapshot.
func WithMailboxCommand(ctx context.Context, observe func(string, Mailbox)) context.Context {
	if command, ok := ctx.Value(mailboxCommandKey{}).(*mailboxCommand); ok {
		if observe != nil {
			command.mu.Lock()
			command.observers = append(command.observers, observe)
			command.mu.Unlock()
		}
		return ctx
	}
	command := &mailboxCommand{snapshots: map[string]Mailbox{}}
	if observe != nil {
		command.observers = append(command.observers, observe)
	}
	return context.WithValue(ctx, mailboxCommandKey{}, command)
}

func MailboxSnapshot(ctx context.Context, store Store, id string) (Mailbox, error) {
	if command, ok := ctx.Value(mailboxCommandKey{}).(*mailboxCommand); ok {
		command.mu.Lock()
		m, found := command.snapshots[id]
		command.mu.Unlock()
		if found {
			return m, nil
		}
	}
	var m Mailbox
	err := store.WithJobLock(id, func() error { var err error; m, err = MailboxSnapshotLocked(ctx, store, id); return err })
	return m, err
}

// MailboxSnapshotLocked must be the first read for mutating commands: their
// unread notice is emitted here, inside the same lock as the operation.
func MailboxSnapshotLocked(ctx context.Context, store Store, id string) (Mailbox, error) {
	command, ok := ctx.Value(mailboxCommandKey{}).(*mailboxCommand)
	if !ok {
		return store.ReadMailboxLocked(id)
	}
	command.mu.Lock()
	if m, found := command.snapshots[id]; found {
		command.mu.Unlock()
		return m, nil
	}
	m, err := store.ReadMailboxLocked(id)
	if err != nil {
		command.mu.Unlock()
		return m, err
	}
	command.snapshots[id] = m
	observers := append([]func(string, Mailbox){}, command.observers...)
	command.mu.Unlock()
	for _, observe := range observers {
		observe(id, m)
	}
	return m, nil
}
