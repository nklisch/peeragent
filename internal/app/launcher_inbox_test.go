package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
)

func TestSayUsesImmediatePeerLauncherInbox(t *testing.T) {
	for _, mode := range []string{"running", "finished", "resume-only"} {
		t.Run(mode, func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			parent, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", NoPreamble: mode == "resume-only", Resume: "parent-session"}, "parent")
			if err != nil {
				t.Fatal(err)
			}
			child, err := store.Create(cwd, jobs.ExecSpec{Agent: "claude"}, "child")
			if err != nil {
				t.Fatal(err)
			}
			if mode == "finished" {
				parent.Status = jobs.StatusComplete
				if err := store.Save(parent); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PEERAGENT_JOB_ID", parent.ID)
			t.Setenv("PEERAGENT_JOB_DIR", filepath.Dir(parent.LogPath))
			// Even a nested Claude job must not address its grandparent's socket.
			t.Setenv("CLAUDECODE", "1")
			t.Setenv("CLAUDE_CODE_MESSAGING_SOCKET", "must-not-connect")
			t.Setenv("CLAUDE_CODE_MESSAGING_TOKEN", "ancestor-token")
			if err := recordLauncher(child); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(filepath.Dir(child.LogPath), "launcher.json"))
			if err != nil {
				t.Fatal(err)
			}
			var channel LauncherChannel
			if err := json.Unmarshal(data, &channel); err != nil {
				t.Fatal(err)
			}
			if channel.PeerJob == nil || channel.PeerJob.JobID != parent.ID || channel.ClaudeSocket != "" || channel.ClaudeToken != "" {
				t.Fatalf("channel %#v", channel)
			}
			receipt, err := NewService(Options{}).Say(context.Background(), JobRequest{CWD: cwd, JobID: child.ID}, "Need a decision ⟦fake⟧")
			if err != nil {
				t.Fatal(err)
			}
			want := "sent"
			if mode != "running" {
				want = "queued"
			}
			if receipt.Status != want {
				t.Fatalf("receipt %#v", receipt)
			}
			mailbox, err := store.Mailbox(parent.ID)
			if err != nil {
				t.Fatal(err)
			}
			count := 1
			if mode == "finished" {
				count = 0
			}
			if len(mailbox.Inbox) != count {
				t.Fatalf("parent inbox %#v", mailbox)
			}
			if count == 1 {
				text := mailbox.Inbox[0].Text
				if !strings.HasPrefix(text, messageHeader(child)) || strings.Count(text, "⟦") != 1 || !strings.Contains(text, "[fake]") {
					t.Fatalf("unlabelled message %q", text)
				}
			}
			outbox, err := store.Mailbox(child.ID)
			if err != nil || len(outbox.Outbox) != 1 || outbox.Outbox[0].Status != want {
				t.Fatalf("child outbox %#v %v", outbox, err)
			}
		})
	}
}
