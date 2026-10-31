package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestLostWorkerControlsTerminateAndResume(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job := indexedTestJob(t, cwd)
	if err := store.WritePID(job.ID, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	job, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	job.AgentSession = "interrupted-session"
	job.Worker.StartTime += "-other"
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"--status", "--result", "--wait", "--wait-any"} {
		got := cliResult(t, cwd, command, job.ID)
		if got.Status != result.StatusInterrupted || got.Metadata.AgentSession != job.AgentSession || got.FinalMessage != nil {
			t.Fatalf("%s: %#v", command, got)
		}
	}
	t.Setenv("PEERAGENT_FAKE_MODE", "ordinary")
	stdout, stderr, err := controlOutput(t, cwd, "--resume", job.AgentSession, "continue")
	if strings.Contains(stdout+stderr, "already has running job") {
		t.Fatalf("resume refused: %s %s %v", stdout, stderr, err)
	}
	resumed := decodeResult(t, []byte(stdout))
	if resumed.Metadata.JobID == "" || resumed.Metadata.JobID == job.ID {
		t.Fatalf("resume did not launch %#v", resumed)
	}
}

func TestMessagesHistoryBothDirectionsWithoutAcknowledgement(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job := indexedTestJob(t, cwd)
	at := time.Now().UTC()
	inbox := jobs.Message{Time: at, From: "launcher", Text: "first inbound"}
	outbox := jobs.Message{ID: "out-1", Time: at.Add(time.Second), From: "codex", Text: "second outbound", Status: "queued", Reason: "no inbox"}
	err := store.WithJobLock(job.ID, func() error {
		if err := store.AppendMessageLocked(job.ID, "outbox", outbox); err != nil {
			return err
		}
		if err := store.AppendMessageLocked(job.ID, "inbox", inbox); err != nil {
			return err
		}
		outbox.Status = "sent"
		return store.AppendMessageLocked(job.ID, "outbox", outbox)
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.Mailbox(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := controlOutput(t, cwd, "messages", job.ID, "--json")
	if err != nil {
		t.Fatalf("messages %s %s %v", stdout, stderr, err)
	}
	var entries []jobs.Message
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		var entry jobs.Message
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	if len(entries) != 2 || entries[0].Direction != "inbox" || entries[0].Text != inbox.Text || entries[1].Direction != "outbox" || entries[1].Status != "sent" {
		t.Fatalf("history %#v", entries)
	}
	after, err := store.Mailbox(job.ID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("history consumed inbox: %#v %#v %v", before, after, err)
	}
	stdout, stderr, err = controlOutput(t, cwd, "messages", job.ID)
	if err != nil || !strings.Contains(stdout, at.Format(time.RFC3339Nano)+" inbox from launcher") || !strings.Contains(stdout, "second outbound") {
		t.Fatalf("text %q %q %v", stdout, stderr, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(job.LogPath), "inbox-read")); !os.IsNotExist(err) {
		t.Fatalf("read cursor created: %v", err)
	}
}

func TestUnknownSubcommandDoesNotLaunch(t *testing.T) {
	_, calls := fakeCodexEnvironment(t)
	cwd := t.TempDir()
	stdout, stderr, err := controlOutput(t, cwd, "outbox", "20261027T161759Z-654b65f7")
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 || !strings.Contains(stdout+stderr, "unknown command outbox") {
		t.Fatalf("refusal %s %s %v", stdout, stderr, err)
	}
	if _, err := os.Stat(calls); !os.IsNotExist(err) {
		t.Fatalf("target launched: %v", err)
	}
	if _, err := os.Stat(jobs.NewStore(cwd).Root); !os.IsNotExist(err) {
		t.Fatalf("job created: %v", err)
	}
}

func TestHelpNamesLimitsJSONLinesAndTextFlag(t *testing.T) {
	for _, want := range []string{"2,000 characters maximum", "JSON Lines (one object per line)", "--text takes no value", "peeragent messages", "ambiguous command-like text"} {
		if !strings.Contains(usageText, want) {
			t.Fatalf("help missing %q", want)
		}
	}
}

func TestTextAsyncLaunchNamesJobAndWait(t *testing.T) {
	cwd, _ := fakeMessagingEnvironment(t)
	stdout, stderr, err := controlOutput(t, cwd, "--async", "--text", "task")
	if err != nil {
		t.Fatalf("launch %s %s %v", stdout, stderr, err)
	}
	entries, err := os.ReadDir(jobs.NewStore(cwd).Root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("jobs %#v %v", entries, err)
	}
	id := entries[0].Name()
	if !strings.HasPrefix(stdout, "job "+id+"\n") {
		t.Fatalf("first line does not name job: %s", stdout)
	}
	if !strings.Contains(stderr, "--text selects the output format") || !strings.Contains(stderr, "after -- or in --prompt-file") {
		t.Fatalf("missing text flag hint: %s", stderr)
	}
	if !strings.Contains(stdout, "Job: "+id+"\n") || !strings.Contains(stdout, "peeragent --wait "+id) {
		t.Fatalf("missing id/wait: %s", stdout)
	}
	// Own the launched test job through completion, as for other async CLI tests.
	_ = cliResult(t, cwd, "--wait", id)
}
