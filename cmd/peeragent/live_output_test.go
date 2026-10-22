package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/result"
)

func waitForObservation(t *testing.T, cwd, id string, predicate func(result.Result) bool) result.Result {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second) // Safety bound; readiness decides success.
	for time.Now().Before(deadline) {
		res := cliResult(t, cwd, "--status", id)
		if predicate(res) {
			return res
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job never exposed expected observation")
	return result.Result{}
}

func TestLiveSessionAndUntruncatedMessageSurviveCancellation(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "live")
	t.Setenv("PEERAGENT_FAKE_GATE", filepath.Join(cwd, "gate"))
	launched := cliResult(t, cwd, "--async", "work")
	id := launched.Metadata.JobID
	t.Cleanup(func() { cliResult(t, cwd, "--cancel", id) })
	waitForObservation(t, cwd, id, func(res result.Result) bool { return res.Metadata.AgentSession == "fake-session" })
	// The session can arrive before the following message is persisted.
	deadline := time.Now().Add(20 * time.Second)
	var running result.Result
	for time.Now().Before(deadline) {
		running = cliResult(t, cwd, "--result", id)
		if running.Metadata.MessagePath != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	want := "stdout:\n" + strings.Repeat("complete report — ", 12000)
	if running.Status != result.StatusRunning || running.Details != want || running.Metadata.FinalReceived {
		t.Fatalf("running message: status=%s session=%s details bytes=%d", running.Status, running.Metadata.AgentSession, len(running.Details))
	}
	cancelled := cliResult(t, cwd, "--cancel", id)
	if cancelled.Status != result.StatusCancelled || cancelled.Metadata.AgentSession != "fake-session" || cancelled.Details != want {
		t.Fatalf("cancelled message/session missing: %#v", cancelled.Metadata)
	}
	stored := cliResult(t, cwd, "--result", id)
	if stored.Metadata.AgentSession != "fake-session" || stored.Details != want {
		t.Fatal("cancelled result lost output")
	}
}

func TestTaskCompleteStopsOwnedHangingCodex(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "final-hang")
	launched := cliResult(t, cwd, "--async", "work")
	id := launched.Metadata.JobID
	t.Cleanup(func() { cliResult(t, cwd, "--cancel", id) })
	observed := waitForObservation(t, cwd, id, func(res result.Result) bool { return res.Metadata.FinalReceived })
	if observed.Status != result.StatusRunning || observed.Summary != "final message received, process still alive" {
		t.Fatalf("grace status: %#v", observed)
	}
	terminal := cliResult(t, cwd, "--wait", id)
	if terminal.Status != result.StatusSuccess || terminal.Metadata.AgentSession != "fake-session" || terminal.Metadata.ExitCode != 0 {
		t.Fatalf("terminal outcome: %#v", terminal.Metadata)
	}
	if terminal.Details != "stdout:\n"+strings.Repeat("complete report — ", 12000) {
		t.Fatalf("report truncated: %d bytes", len(terminal.Details))
	}
	if _, err := os.Stat(terminal.Metadata.LogPath); err != nil {
		t.Fatal(err)
	}
}

func TestFailedCodexRetainsFinalMessage(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "failed-message")
	launched := cliResult(t, cwd, "--async", "work")
	terminal := cliResult(t, cwd, "--wait", launched.Metadata.JobID)
	if terminal.Status != result.StatusFailed || terminal.Metadata.AgentSession != "fake-session" || terminal.Details != "stdout:\n"+strings.Repeat("complete report — ", 12000) {
		t.Fatalf("failure output: status=%s details bytes=%d", terminal.Status, len(terminal.Details))
	}
}
