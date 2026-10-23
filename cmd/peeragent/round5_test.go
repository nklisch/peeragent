package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/prompt"
	"github.com/nklisch/peeragent/internal/result"
)

func TestCaptureStdoutHandlesLargeOutput(t *testing.T) {
	if os.Getenv("PEERAGENT_CAPTURE_HELPER") == "1" {
		original := os.Stdout
		want := errors.New("command outcome")
		err := captureStdout(func() error {
			data := strings.Repeat("x", 1<<20)
			n, err := io.WriteString(os.Stdout, data)
			if err != nil || n != len(data) {
				t.Fatalf("large write: %d %v", n, err)
			}
			return want
		})
		if !errors.Is(err, want) || os.Stdout != original {
			t.Fatalf("capture did not restore stdout/outcome: %v", err)
		}
		return
	}
	// The old helper blocks even on Linux once output exceeds pipe capacity.
	// Bound the owned child so this regression reports a useful failure rather
	// than hanging the whole suite; this is a safety timeout, not a timing probe.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCaptureStdoutHandlesLargeOutput$")
	cmd.Env = append(os.Environ(), "PEERAGENT_CAPTURE_HELPER=1")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("stdout capture blocked on output exceeding pipe capacity: %v\n%s", ctx.Err(), out)
	}
	if err != nil {
		t.Fatalf("large-output helper: %v\n%s", err, out)
	}
}

func removeMessagingMarker(t *testing.T, job jobs.Job) {
	t.Helper()
	path := filepath.Join(filepath.Dir(job.LogPath), "job.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "messaging_enabled")
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyMessagingEvidenceAcrossControls(t *testing.T) {
	fakeCodexEnvironment(t)
	for _, tc := range []struct {
		name, text, outbox, source string
		enabled                    bool
	}{
		{"prompt", prompt.Build("task"), "", "inferred from prompt", true},
		{"outbox", "task", "sent", "inferred from outbox", true},
		{"before messaging", "task", "", "no messaging evidence", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job := indexedTestJob(t, cwd)
			if err := store.WritePrompt(job.ID, tc.text); err != nil {
				t.Fatal(err)
			}
			removeMessagingMarker(t, job)
			if tc.outbox != "" {
				if err := store.WithJobLock(job.ID, func() error {
					return store.AppendMessageLocked(job.ID, "outbox", jobs.Message{Time: time.Now().UTC(), Text: "question", Status: tc.outbox})
				}); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{{"--status", job.ID}, {"--status", job.ID, "--text"}, {"--status", job.ID, job.ID}, {"--jobs"}, {"--jobs", "--text"}} {
				out, diagnostic, err := controlOutput(t, cwd, args...)
				if err != nil {
					t.Fatalf("%v: %q %q %v", args, out, diagnostic, err)
				}
				if strings.Contains(out, jobs.LegacyMessagingWarning) == tc.enabled {
					t.Fatalf("wrong warning in %v: %s", args, out)
				}
				if args[len(args)-1] == "--text" {
					if !strings.Contains(out, tc.source) && tc.enabled {
						t.Fatalf("missing source: %s", out)
					}
				} else if args[0] == "--status" && len(args) == 2 {
					res := decodeResult(t, []byte(out))
					if res.Metadata.MessagingEnabled != tc.enabled || res.Metadata.MessagingSource != tc.source {
						t.Fatalf("single label: %#v", res.Metadata)
					}
				} else {
					for _, r := range decodeJobLines(t, out) {
						if r.MessagingEnabled != tc.enabled || r.MessagingSource != tc.source {
							t.Fatalf("list label: %#v", r)
						}
					}
				}
			}
			out, diagnostic, err := controlOutput(t, cwd, "tell", job.ID, "answer")
			if err != nil || strings.Contains(diagnostic, jobs.LegacyMessagingWarning) == tc.enabled || !strings.Contains(out, "unread 1, oldest 0s") {
				t.Fatalf("tell: %q %q %v", out, diagnostic, err)
			}
			if tc.enabled && !strings.Contains(out, tc.source) {
				t.Fatalf("tell lacks source: %s", out)
			}
		})
	}
}

func TestUnreadAgeAcrossStatusJobsAndTell(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job := indexedTestJob(t, cwd)
	oldest := time.Now().UTC().Add(-2*time.Hour - 14*time.Minute)
	if err := store.WithJobLock(job.ID, func() error {
		for _, at := range []time.Time{oldest.Add(time.Hour), oldest} {
			if err := store.AppendMessageLocked(job.ID, "inbox", jobs.Message{Time: at, From: "launcher", Text: "direction change"}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"--status", job.ID}, {"--status", job.ID, job.ID}, {"--jobs"}, {"--jobs", "--text"}, {"--status", job.ID, "--text"}} {
		out, diagnostic, err := controlOutput(t, cwd, args...)
		if err != nil || !strings.Contains(out, "2h14m") {
			t.Fatalf("%v: %q %q %v", args, out, diagnostic, err)
		}
		if args[len(args)-1] == "--text" {
			if !strings.Contains(out, jobs.InboxReadContext) {
				t.Fatalf("missing read context: %s", out)
			}
		} else if args[0] == "--status" && len(args) == 2 {
			res := decodeResult(t, []byte(out))
			if res.Metadata.OldestUnreadAt == nil || !res.Metadata.OldestUnreadAt.Equal(oldest) || res.Metadata.OldestUnreadAge != "2h14m" {
				t.Fatalf("oldest metadata: %#v", res.Metadata)
			}
		} else {
			for _, r := range decodeJobLines(t, out) {
				if r.OldestUnreadAt == nil || !r.OldestUnreadAt.Equal(oldest) || r.OldestUnreadAge != "2h14m" {
					t.Fatalf("oldest row: %#v", r)
				}
			}
		}
	}
	out, diagnostic, err := controlOutput(t, cwd, "tell", job.ID, "new instruction")
	want := fmt.Sprintf("delivered to job %s (codex, running); unread 3, oldest 2h14m; %s\n", job.ID, jobs.InboxReadContext)
	if err != nil || out != want || diagnostic != "" {
		t.Fatalf("age receipt: %q %q %v", out, diagnostic, err)
	}
	if err := store.ReadInbox(job.ID, func([]jobs.Message) error { return nil }); err != nil {
		t.Fatal(err)
	}
	res, err := app.NewService(app.Options{}).JobStatus(context.Background(), app.JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || res.Metadata.OldestUnreadAt != nil || res.Metadata.OldestUnreadAge != "" || res.Metadata.InboxUnread != 0 {
		t.Fatalf("stale unread age: %#v %v", res, err)
	}
	if text := result.FormatText(res); strings.Contains(text, "oldest_unread_age") {
		t.Fatalf("empty inbox has age: %s", text)
	}
}
