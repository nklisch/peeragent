package jobs

import (
	"context"
	"testing"
	"time"
)

func TestOldestUnreadAgeUsesOnlyUnreadEntries(t *testing.T) {
	now := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC)
	mailbox := Mailbox{Inbox: []Message{
		{Time: now.Add(-24 * time.Hour)}, // already read
		{Time: now.Add(-time.Minute)},
		{Time: now.Add(-12 * time.Hour)}, // carried in a running prompt
		{Time: now.Add(-2*time.Hour - 14*time.Minute)},
	}, unread: []int{1, 3}}
	stats := mailbox.StatsAt(now)
	if stats.InboxUnread != 2 || stats.OldestUnreadAt == nil || !stats.OldestUnreadAt.Equal(mailbox.Inbox[3].Time) || stats.OldestUnreadAge != "2h14m" {
		t.Fatalf("oldest unread: %#v", stats)
	}
	if stats.LastMessageAt == nil || !stats.LastMessageAt.Equal(mailbox.Inbox[1].Time) {
		t.Fatalf("last-message time changed: %#v", stats)
	}
	if got := stats.UnreadSummary(); got != "unread 2, oldest 2h14m" {
		t.Fatal(got)
	}
}

func TestUnreadAgeFormatting(t *testing.T) {
	now := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		at   time.Time
		want string
	}{
		{"seconds", now.Add(-42 * time.Second), "42s"},
		{"minutes", now.Add(-14*time.Minute - 12*time.Second), "14m"},
		{"hours", now.Add(-2 * time.Hour), "2h0m"},
		{"days", now.Add(-26*time.Hour - 14*time.Minute), "26h14m"},
		{"future clock", now.Add(time.Hour), "0s"},
		{"unknown legacy timestamp", time.Time{}, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stats := (Mailbox{Inbox: []Message{{Time: tc.at}}, unread: []int{0}}).StatsAt(now)
			if stats.UnreadAge() != tc.want {
				t.Fatalf("age %#v, want %s", stats, tc.want)
			}
		})
	}
	mixed := (Mailbox{Inbox: []Message{{}, {Time: now.Add(-time.Hour)}}, unread: []int{0, 1}}).StatsAt(now)
	if mixed.OldestUnreadAt != nil || mixed.UnreadAge() != "unknown" {
		t.Fatalf("unknown timestamp hidden by known one: %#v", mixed)
	}
	stats := (Mailbox{}).StatsAt(now)
	if stats.OldestUnreadAt != nil || stats.OldestUnreadAge != "" || stats.UnreadSummary() != "unread 0" {
		t.Fatalf("empty mailbox: %#v", stats)
	}
}

func TestUnreadAgeFollowsReservationAndAcknowledgement(t *testing.T) {
	now := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC)
	cwd := t.TempDir()
	store := NewStore(cwd)
	job, err := store.Create(cwd, ExecSpec{}, "task")
	if err != nil {
		t.Fatal(err)
	}
	appendMessage := func(at time.Time) {
		t.Helper()
		if err := store.WithJobLock(job.ID, func() error { return store.AppendMessageLocked(job.ID, "inbox", Message{Time: at, Text: "reply"}) }); err != nil {
			t.Fatal(err)
		}
	}
	appendMessage(now.Add(-2*time.Hour - 14*time.Minute))
	attempt, err := BeginInboxAttempt(job)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReserveInbox(context.Background(), store, job.ID, attempt); err != nil {
		t.Fatal(err)
	}
	appendMessage(now.Add(-time.Minute))
	mailbox, err := store.Mailbox(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats := mailbox.StatsAt(now); stats.InboxUnread != 1 || stats.OldestUnreadAge != "1m" {
		t.Fatalf("reserved entry counted: %#v", stats)
	}
	outcome, err := FinishInboxAttempt(attempt, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SettleInboxAttempt(job.ID, attempt.ID, outcome); err != nil {
		t.Fatal(err)
	}
	mailbox, err = store.Mailbox(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats := mailbox.StatsAt(now); stats.InboxUnread != 2 || stats.OldestUnreadAge != "2h14m" {
		t.Fatalf("released entry missing: %#v", stats)
	}
	if err := store.ReadInbox(job.ID, func([]Message) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mailbox, err = store.Mailbox(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats := mailbox.StatsAt(now); stats.InboxUnread != 0 || stats.OldestUnreadAt != nil || stats.OldestUnreadAge != "" {
		t.Fatalf("read entry still counted: %#v", stats)
	}
}
