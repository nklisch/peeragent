package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestMain(m *testing.M) {
	if os.Getenv("PEERAGENT_TEST_HELPER_MAIN") == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

func TestAsyncCancelKillsDetachedProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group cancellation is unix-only")
	}
	cwd := t.TempDir()
	statePath := filepath.Join(cwd, "fake-codex-state")
	fakeBin := filepath.Join(cwd, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	codexPath := filepath.Join(fakeBin, "codex")
	script := `#!/bin/sh
printf 'wrapper=%s\n' "$$" > "$PEERAGENT_FAKE_CODEX_STATE"
trap '' TERM
( trap '' TERM; sleep 100 ) &
child=$!
printf 'child=%s\n' "$child" >> "$PEERAGENT_FAKE_CODEX_STATE"
wait "$child"
`
	if err := os.WriteFile(codexPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	launch := exec.Command(os.Args[0], "--cwd", cwd, "--async", "--agent", "codex", "sleep until cancelled")
	launch.Env = helperEnv(fakeBin, statePath)
	output, err := launch.CombinedOutput()
	if err != nil {
		t.Fatalf("launch failed: %v\n%s", err, output)
	}
	launchResult := decodeResult(t, output)
	if launchResult.Status != result.StatusRunning {
		t.Fatalf("launch status = %q, output:\n%s", launchResult.Status, output)
	}
	jobID := launchResult.Metadata.JobID
	if jobID == "" {
		t.Fatalf("missing job id in %#v", launchResult.Metadata)
	}

	store := jobs.NewStore(cwd)
	wrapperPID, err := store.ReadPID(jobID)
	if err != nil {
		t.Fatal(err)
	}
	if pgid, err := syscall.Getpgid(wrapperPID); err != nil || pgid != wrapperPID {
		t.Fatalf("wrapper pgid = %d, err = %v, want pgid == pid %d", pgid, err, wrapperPID)
	}
	childPID := waitForFakeChildPID(t, statePath)

	start := time.Now()
	cancel := exec.Command(os.Args[0], "--cwd", cwd, "--cancel", jobID)
	cancel.Env = helperEnv(fakeBin, statePath)
	cancelOutput, err := cancel.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("cancel failed: %v\n%s", err, cancelOutput)
	}
	if elapsed > 6500*time.Millisecond {
		t.Fatalf("cancel took %s, want roughly 5s + 500ms", elapsed)
	}
	cancelResult := decodeResult(t, cancelOutput)
	if cancelResult.Status != result.StatusCancelled {
		t.Fatalf("cancel status = %q, output:\n%s", cancelResult.Status, cancelOutput)
	}
	if _, err := store.ReadPID(jobID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID after cancel err = %v, want missing pid", err)
	}
	assertProcessGone(t, wrapperPID)
	assertProcessGone(t, childPID)
}

func helperEnv(fakeBin string, statePath string) []string {
	env := os.Environ()
	env = append(env,
		"PEERAGENT_TEST_HELPER_MAIN=1",
		"PEERAGENT_FAKE_CODEX_STATE="+statePath,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	return env
}

func decodeResult(t *testing.T, output []byte) result.Result {
	t.Helper()
	var res result.Result
	if err := json.Unmarshal(output, &res); err != nil {
		t.Fatalf("decode result: %v\n%s", err, output)
	}
	return res
}

func waitForFakeChildPID(t *testing.T, statePath string) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(statePath)
		if err == nil {
			for _, line := range strings.Split(string(content), "\n") {
				key, value, ok := strings.Cut(line, "=")
				if ok && key == "child" {
					pid, err := strconv.Atoi(strings.TrimSpace(value))
					if err != nil {
						t.Fatalf("invalid child pid in %q: %v", line, err)
					}
					return pid
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("fake codex did not record a child pid at %s", statePath)
	return 0
}

func assertProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal(fmt.Sprintf("process %d still exists", pid))
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
