package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// Keep the test worker's identity after production cancellation removes pid.
// This runs only in the fake CLI executable, before the worker can complete.
func recordTestWorkerPID() {
	if len(os.Args) != 5 || os.Args[1] != "--job-run" || os.Args[3] != "--cwd" || jobs.ValidateID(os.Args[2]) != nil {
		return
	}
	path := filepath.Join(jobs.NewStore(os.Args[4]).Root, os.Args[2], "test-worker.pid")
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		panic(err)
	}
}

// Register after launch, so this callback runs before TempDir and environment
// cleanup. Gate and job location are captured now, before later tests override
// environment variables. Never infer worker exit merely from terminal status.
func cleanupAsyncJob(t *testing.T, cwd, id string) {
	t.Helper()
	gate := os.Getenv("PEERAGENT_FAKE_GATE")
	t.Cleanup(func() { cancelAndWaitForTestJob(t, cwd, id, gate) })
}

func cancelAndWaitForTestJob(t *testing.T, cwd, id, gate string) {
	t.Helper()
	store := jobs.NewStore(cwd)
	pid, err := store.ReadPID(id)
	if os.IsNotExist(err) {
		var data []byte
		data, err = os.ReadFile(filepath.Join(store.Root, id, "test-worker.pid"))
		if err == nil {
			pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
		}
	}
	if err != nil || pid <= 1 {
		t.Errorf("cleanup job %s: cannot identify its owned worker: pid=%d err=%v", id, pid, err)
		return
	}
	// Cancellation is bounded separately from waiting for the worker to release
	// agent.log. The CLI test executable never launches a real target agent here.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "--cwd", cwd, "--cancel", id)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "PEERAGENT_TEST_HELPER_MAIN=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("cancel test job %s: %v: %s", id, err, output)
	}
	// Windows production cancel currently records the terminal state without
	// signalling the worker. Only after the test assertions/cancel, release the
	// fake target cooperatively. This changes no production cancellation rule.
	if gate != "" {
		if err := os.WriteFile(gate, []byte("cleanup"), 0o600); err != nil {
			t.Errorf("release fake target for job %s: %v", id, err)
		}
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		job, err := store.Load(id)
		if err != nil {
			t.Errorf("load test job %s during cleanup: %v", id, err)
			return
		}
		if jobs.IsTerminalStatus(job.Status) && !jobs.ProcessExists(pid) {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("cleanup job %s: worker %d still alive=%t, status=%s", id, pid, jobs.ProcessExists(pid), job.Status)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAsyncCleanupWaitsForWorkerBeforeTempDirRemoval(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "live")
	var id string
	// Register the assertion first: cleanupAsyncJob must finish before this check,
	// and both callbacks must run before the TempDir's RemoveAll callback.
	t.Cleanup(func() {
		data, err := os.ReadFile(filepath.Join(jobs.NewStore(cwd).Root, id, "test-worker.pid"))
		if err != nil {
			t.Error(err)
			return
		}
		pid, err := strconv.Atoi(string(data))
		if err != nil {
			t.Error(err)
			return
		}
		if jobs.ProcessExists(pid) {
			t.Errorf("cleanup returned while worker %d was alive", pid)
		}
		job, err := jobs.NewStore(cwd).Load(id)
		if err != nil {
			t.Error(err)
			return
		}
		if !jobs.IsTerminalStatus(job.Status) {
			t.Errorf("cleanup returned with job status %s", job.Status)
		}
		if err := os.Remove(job.LogPath); err != nil {
			t.Errorf("worker still holds agent.log: %v", err)
		}
	})
	launched := cliResult(t, cwd, "--async", "work")
	id = launched.Metadata.JobID
	waitForObservation(t, cwd, id, "--status", func(res result.Result) bool { return res.Metadata.LastMessage != "" })
}
