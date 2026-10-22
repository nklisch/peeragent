package jobs

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func appendInboxForTest(t *testing.T, store Store, id, text string) {
	t.Helper()
	if err := store.WithJobLock(id, func() error { return store.AppendMessageLocked(id, "inbox", Message{From: "launcher", Text: text}) }); err != nil {
		t.Fatal(err)
	}
}

func TestInboxAttemptCrashRecovery(t *testing.T) {
	// An owned subprocess reserves replies, then exits without mailbox settlement.
	// Parent waits on that exact child handle before observing recovery.
	if mode := os.Getenv("PEERAGENT_TEST_INBOX_CRASH_MODE"); mode != "" {
		cwd, id := os.Getenv("PEERAGENT_TEST_INBOX_CWD"), os.Getenv("PEERAGENT_TEST_INBOX_JOB")
		store := NewStore(cwd)
		job, err := store.Load(id)
		if err != nil {
			t.Fatal(err)
		}
		attempt, err := BeginInboxAttempt(job)
		if err != nil {
			t.Fatal(err)
		}
		entries, err := ReserveInbox(context.Background(), store, id, attempt)
		if err != nil || len(entries) != 1 {
			t.Fatalf("reserve %#v %v", entries, err)
		}
		if mode == "accepted" {
			if _, err := FinishInboxAttempt(attempt, true); err != nil {
				t.Fatal(err)
			}
		}
		if mode == "missing-record" {
			if err := os.Remove(attempt.Path); err != nil {
				t.Fatal(err)
			}
		}
		data, err := json.Marshal(attempt)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cwd, "attempt-ref.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		os.Exit(0)
	}
	for _, mode := range []string{"abandoned", "missing-record", "accepted"} {
		t.Run(mode, func(t *testing.T) {
			cwd := t.TempDir()
			store := NewStore(cwd)
			job, err := store.Create(cwd, ExecSpec{}, "mailbox")
			if err != nil {
				t.Fatal(err)
			}
			appendInboxForTest(t, store, job.ID, "recover this reply")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInboxAttemptCrashRecovery$")
			cmd.Env = append(os.Environ(), "PEERAGENT_TEST_INBOX_CRASH_MODE="+mode, "PEERAGENT_TEST_INBOX_CWD="+cwd, "PEERAGENT_TEST_INBOX_JOB="+job.ID)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("helper %v %s", err, output)
			}
			data, err := os.ReadFile(filepath.Join(cwd, "attempt-ref.json"))
			if err != nil {
				t.Fatal(err)
			}
			var attempt InboxAttempt
			if err := json.Unmarshal(data, &attempt); err != nil {
				t.Fatal(err)
			}
			if attempt.PID != cmd.Process.Pid || ProcessExists(attempt.PID) {
				t.Fatalf("owned helper still alive or wrong handle: %#v", attempt)
			}
			mailbox, err := store.Mailbox(job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "accepted" {
				if mailbox.Read != 1 || mailbox.UnreadCount() != 0 {
					t.Fatalf("accepted reply replayed %#v", mailbox)
				}
			} else if mailbox.Read != 0 || mailbox.UnreadCount() != 1 {
				t.Fatalf("crash stranded reply %#v", mailbox)
			}
			// Recovery is persisted; a second read has the same result.
			again, err := store.Mailbox(job.ID)
			if err != nil || again.Read != mailbox.Read || again.UnreadCount() != mailbox.UnreadCount() {
				t.Fatalf("unstable recovery %#v %v", again, err)
			}
		})
	}
}

func TestInboxReservationsPreserveIndependentDeliveryOrder(t *testing.T) {
	cwd := t.TempDir()
	store := NewStore(cwd)
	source, err := store.Create(cwd, ExecSpec{}, "source")
	if err != nil {
		t.Fatal(err)
	}
	first, err := BeginInboxAttempt(source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BeginInboxAttempt(source)
	if err != nil {
		t.Fatal(err)
	}
	appendInboxForTest(t, store, source.ID, "first")
	entries, err := ReserveInbox(context.Background(), store, source.ID, first)
	if err != nil || len(entries) != 1 || entries[0].Text != "first" {
		t.Fatalf("first %#v %v", entries, err)
	}
	appendInboxForTest(t, store, source.ID, "second")
	entries, err = ReserveInbox(context.Background(), store, source.ID, second)
	if err != nil || len(entries) != 1 || entries[0].Text != "second" {
		t.Fatalf("second %#v %v", entries, err)
	}
	mailbox, err := store.Mailbox(source.ID)
	if err != nil || mailbox.UnreadCount() != 0 {
		t.Fatalf("live reservations visible %#v %v", mailbox, err)
	}
	if _, err := FinishInboxAttempt(second, true); err != nil {
		t.Fatal(err)
	}
	// Recovery commits the second reply while the first remains reserved.
	mailbox, err = store.Mailbox(source.ID)
	if err != nil || mailbox.Read != 0 || mailbox.UnreadCount() != 0 {
		t.Fatalf("read across reservation %#v %v", mailbox, err)
	}
	if _, err := FinishInboxAttempt(first, false); err != nil {
		t.Fatal(err)
	}
	var delivered []Message
	if err := store.ReadInbox(source.ID, func(entries []Message) error { delivered = entries; return nil }); err != nil {
		t.Fatal(err)
	}
	if len(delivered) != 1 || delivered[0].Text != "first" {
		t.Fatalf("released/read entries mixed: %#v", delivered)
	}
	mailbox, err = store.Mailbox(source.ID)
	if err != nil || mailbox.Read != 2 || mailbox.UnreadCount() != 0 {
		t.Fatalf("final state %#v %v", mailbox, err)
	}
}

func TestInboxAttemptAcceptanceWinsProcessExitRace(t *testing.T) {
	cwd := t.TempDir()
	job, err := NewStore(cwd).Create(cwd, ExecSpec{}, "attempt")
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := BeginInboxAttempt(job)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := resolveInboxAttempt(attempt, func(pid int) bool {
		if pid != attempt.PID {
			t.Fatalf("wrong recorded process %d", pid)
		}
		// Publish exactly after the reader saw running, before it observes exit.
		if err := writeInboxAttempt(attempt, InboxAttemptAccepted); err != nil {
			t.Fatal(err)
		}
		return false
	})
	if err != nil || outcome != InboxAttemptAccepted {
		t.Fatalf("published acceptance lost: %s %v", outcome, err)
	}
}
