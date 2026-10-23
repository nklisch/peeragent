package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestTellQueueReceiptAndMissingSession(t *testing.T) {
	fakeCodexEnvironment(t)
	for _, legacy := range []bool{false, true} {
		for _, sessionSource := range []string{"agent_session", "spec.resume", "none"} {
			t.Run(fmt.Sprintf("legacy=%t/%s", legacy, sessionSource), func(t *testing.T) {
				cwd := t.TempDir()
				store := jobs.NewStore(cwd)
				job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", NoPreamble: !legacy}, "old task")
				if err != nil {
					t.Fatal(err)
				}
				switch sessionSource {
				case "agent_session":
					job.AgentSession = "recorded-session"
				case "spec.resume":
					job.Spec.Resume = "resumed-session"
				}
				if err := store.Save(job); err != nil {
					t.Fatal(err)
				}
				if legacy {
					removeMessagingMarker(t, job)
				}
				for n := 1; n <= 2; n++ {
					message := fmt.Sprintf("reply %d", n)
					out, diagnostic, err := controlOutput(t, cwd, "tell", job.ID, message)
					mailbox, mailErr := store.Mailbox(job.ID)
					if mailErr != nil {
						t.Fatal(mailErr)
					}
					if sessionSource == "none" {
						if err == nil || !strings.Contains(out+diagnostic, "message not queued") || !strings.Contains(out+diagnostic, "no agent session to --resume") || strings.Contains(out, "queued for the next") {
							t.Fatalf("missing-session response: %q %q %v", out, diagnostic, err)
						}
						if mailbox.UnreadCount() != 0 || len(mailbox.Inbox) != 0 {
							t.Fatalf("undeliverable message stored: %#v", mailbox)
						}
					} else {
						want := fmt.Sprintf("queued for the next --resume (%d pending)", n)
						if err != nil || !strings.Contains(out, want) {
							t.Fatalf("receipt: %q %q %v", out, diagnostic, err)
						}
						if mailbox.UnreadCount() != n || len(mailbox.Inbox) != n || mailbox.Inbox[n-1].Text != message {
							t.Fatalf("receipt does not match stored messages: %#v", mailbox)
						}
					}
				}
			})
		}
	}
}

func TestCapacityWordingAcrossWaitResultAndStatus(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job := indexedTestJob(t, cwd)
	job.Status = jobs.StatusCapacity
	job.AgentSession = "capacity-session"
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	// Old persisted summaries must also be presented using the current wording.
	agentMessage := "Selected model is at capacity"
	res := result.Result{Status: result.StatusCapacity, Summary: "stopped: model at capacity", Details: "stderr:\n" + agentMessage, Metadata: result.Metadata{JobID: job.ID, CWD: cwd, AgentSession: job.AgentSession, ExitCode: 1}}
	if err := app.WriteJobResult(job.ResultPath, res); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(job.LogPath, []byte(agentMessage), 0600); err != nil {
		t.Fatal(err)
	}
	want := "stopped: capacity_exhausted; resumable with --resume capacity-session; use --retry-capacity N to retry automatically"
	for _, command := range []string{"--status", "--result", "--wait"} {
		got := cliResult(t, cwd, command, job.ID)
		if got.Status != result.StatusCapacity || got.Summary != want {
			t.Fatalf("%s: %#v", command, got)
		}
		if command != "--status" && got.Details != res.Details {
			t.Fatalf("agent text changed: %q", got.Details)
		}
		out, diagnostic, err := controlOutput(t, cwd, command, job.ID, "--text")
		// Query controls describe a terminal failure without failing the query.
		if err != nil || !strings.Contains(out, "Summary:\n"+want) {
			t.Fatalf("%s text: %q %q %v", command, out, diagnostic, err)
		}
	}
	data, err := os.ReadFile(job.LogPath)
	if err != nil || string(data) != agentMessage {
		t.Fatalf("agent log changed: %q %v", data, err)
	}
}
