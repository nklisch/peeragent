package main

import (
	"os"
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

func TestCapacityRetryPreservesMessagePointer(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "capacity-prior-message")
	launched := cliResult(t, cwd, "--async", "--retry-capacity", "1", "--capacity-backoff", "1ms", "work")
	terminal := cliResult(t, cwd, "--wait", launched.Metadata.JobID)
	if terminal.Status != result.StatusFailed || terminal.Metadata.CapacityRetries != 1 || terminal.Metadata.AgentSession != "fake-session" || terminal.Metadata.MessagePath == "" {
		t.Fatalf("retry lost message pointer: %#v", terminal.Metadata)
	}
	text, err := os.ReadFile(terminal.Metadata.MessagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(text) != strings.Repeat("complete report — ", 12000) {
		t.Fatalf("prior attempt report changed: %d bytes", len(text))
	}
}
