//go:build unix

package main

import (
	"github.com/nklisch/peeragent/internal/result"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLiveSessionAndUntruncatedMessageSurviveCancellation(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "live")
	t.Setenv("PEERAGENT_FAKE_GATE", filepath.Join(cwd, "gate"))
	launched := cliResult(t, cwd, "--async", "work")
	id := launched.Metadata.JobID
	t.Cleanup(func() { cliResult(t, cwd, "--cancel", id) })
	waitForObservation(t, cwd, id, "--status", func(res result.Result) bool { return res.Metadata.AgentSession == "fake-session" })
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
