package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestBlockingCLIWorkerExitReturnsFailureAndAllowsResume(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_WORKER_EXIT", "1")
	// A safety bound catches a regression in the test; production observes exit,
	// with no deadline. The fake worker immediately exits with no result file.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "--resume", "dead-session", "--cwd", cwd, "work")
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("blocking call hung after worker exit: %s", output)
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		t.Fatalf("blocking wrapper exit: %v %s", err, output)
	}
	terminal := decodeResult(t, output)
	if terminal.Status != result.StatusFailed || terminal.Metadata.ExitCode != 23 || terminal.Metadata.AgentSession != "dead-session" || terminal.Metadata.JobID == "" || terminal.Metadata.LogPath == "" {
		t.Fatalf("worker failure: %#v", terminal)
	}
	log, err := os.ReadFile(terminal.Metadata.LogPath)
	if err != nil || !strings.Contains(string(log), "fake worker exited") {
		t.Fatalf("missing worker logs: %q %v", log, err)
	}
	stored := cliResult(t, cwd, "--result", terminal.Metadata.JobID)
	if stored.Status != result.StatusFailed || stored.Metadata.ExitCode != 23 {
		t.Fatalf("failure not persisted: %#v", stored)
	}
	job, err := jobs.NewStore(cwd).Load(terminal.Metadata.JobID)
	if err != nil || job.Status != jobs.StatusFailed {
		t.Fatalf("job still running: %#v %v", job, err)
	}
	t.Setenv("PEERAGENT_FAKE_WORKER_EXIT", "")
	t.Setenv("PEERAGENT_FAKE_MODE", "incident-direct")
	// This next fake Codex call fails with capacity; it must reach the target,
	// rather than being refused by the dead session's guard.
	next := exec.CommandContext(ctx, os.Args[0], "--resume", "dead-session", "--cwd", cwd, "work")
	next.Dir = cwd
	output, err = next.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("resume hung: %s", output)
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		t.Fatalf("resumed wrapper exit: %v %s", err, output)
	}
	resumed := decodeResult(t, output)
	if resumed.Status != result.StatusCapacity || resumed.Metadata.JobID == terminal.Metadata.JobID {
		t.Fatalf("dead worker blocked next resume: %#v", resumed)
	}
}
