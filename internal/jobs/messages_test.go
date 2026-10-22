package jobs

import (
	"errors"
	"testing"
)

func TestInboxOutputFailureLeavesMessagesUnread(t *testing.T) {
	cwd := t.TempDir()
	store := NewStore(cwd)
	job, err := store.Create(cwd, ExecSpec{}, "mailbox")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WithJobLock(job.ID, func() error {
		return store.AppendMessageLocked(job.ID, "inbox", Message{From: "launcher", Text: "answer"})
	}); err != nil {
		t.Fatal(err)
	}
	want := errors.New("output closed")
	if err := store.ReadInbox(job.ID, func(entries []Message) error {
		if len(entries) != 1 {
			t.Fatal("missing answer")
		}
		return want
	}); !errors.Is(err, want) {
		t.Fatalf("delivery %v", err)
	}
	m, err := store.Mailbox(job.ID)
	if err != nil || m.Read != 0 || len(m.Inbox) != 1 {
		t.Fatalf("mailbox %#v %v", m, err)
	}
}
