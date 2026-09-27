//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestMain(m *testing.M) {
	if strings.EqualFold(filepath.Base(os.Args[0]), "codex.exe") {
		fakeCodexMain()
		return
	}
	if os.Getenv("PEERAGENT_TEST_HELPER_MAIN") == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

func fakeCodexMain() {
	if os.Getenv("PEERAGENT_FAKE_CHILD") == "1" {
		for {
			time.Sleep(time.Hour)
		}
	}
	if os.Getenv("PEERAGENT_FAKE_MODE") == "quick" {
		if !strings.Contains(os.Args[len(os.Args)-1], `say "OK" & keep | literal`) {
			fmt.Fprintln(os.Stderr, "task text was altered before reaching Codex")
			os.Exit(1)
		}
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"WINDOWS_OK"}}`)
		return
	}
	child := exec.Command(os.Args[0])
	child.Env = append(os.Environ(), "PEERAGENT_FAKE_CHILD=1")
	if err := child.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	state := fmt.Sprintf("target=%d\nchild=%d\n", os.Getpid(), child.Process.Pid)
	if err := os.WriteFile(os.Getenv("PEERAGENT_FAKE_STATE"), []byte(state), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = child.Wait()
}

func TestWindowsBlockingAndAsyncFromPathsWithSpaces(t *testing.T) {
	cwd, fakeBin, state := windowsFixture(t, "quick")
	task := `say "OK" & keep | literal`
	blocking := runWindowsCLI(t, fakeBin, state, "quick", "--cwd", cwd, "--agent", "codex", task)
	if blocking.Status != result.StatusSuccess || !strings.Contains(blocking.Details, "WINDOWS_OK") {
		t.Fatalf("blocking result = %#v", blocking)
	}
	launch := runWindowsCLI(t, fakeBin, state, "quick", "--cwd", cwd, "--async", "--agent", "codex", task)
	if launch.Status != result.StatusRunning || launch.Metadata.JobID == "" {
		t.Fatalf("async launch = %#v", launch)
	}
	id := launch.Metadata.JobID
	_ = runWindowsCLI(t, fakeBin, state, "quick", "--cwd", cwd, "--status", id)
	waited := runWindowsCLI(t, fakeBin, state, "quick", "--cwd", cwd, "--wait", id)
	if waited.Status != result.StatusSuccess || !strings.Contains(waited.Details, "WINDOWS_OK") {
		t.Fatalf("wait result = %#v", waited)
	}
	fetched := runWindowsCLI(t, fakeBin, state, "quick", "--cwd", cwd, "--result", id)
	if fetched.Status != result.StatusSuccess {
		t.Fatalf("result = %#v", fetched)
	}
}

func TestWindowsCancelTerminatesTargetAndChild(t *testing.T) {
	cwd, fakeBin, state := windowsFixture(t, "slow")
	launch := runWindowsCLI(t, fakeBin, state, "slow", "--cwd", cwd, "--async", "--agent", "codex", "wait")
	id := launch.Metadata.JobID
	if launch.Status != result.StatusRunning || id == "" {
		t.Fatalf("launch = %#v", launch)
	}
	pids := waitForWindowsFakeState(t, state)
	if got := runWindowsCLI(t, fakeBin, state, "slow", "--cwd", cwd, "--status", id); got.Status != result.StatusRunning {
		t.Fatalf("status before cancel = %#v", got)
	}
	cancelled := runWindowsCLI(t, fakeBin, state, "slow", "--cwd", cwd, "--cancel", id)
	if cancelled.Status != result.StatusCancelled {
		t.Fatalf("cancel result = %#v", cancelled)
	}
	if got := runWindowsCLI(t, fakeBin, state, "slow", "--cwd", cwd, "--wait", id); got.Status != result.StatusCancelled {
		t.Fatalf("wait after cancel = %#v", got)
	}
	if got := runWindowsCLI(t, fakeBin, state, "slow", "--cwd", cwd, "--result", id); got.Status != result.StatusCancelled {
		t.Fatalf("result after cancel = %#v", got)
	}
	for _, pid := range pids {
		assertWindowsProcessGone(t, pid)
	}
	if _, err := jobs.NewStore(cwd).ReadPID(id); !os.IsNotExist(err) {
		t.Fatalf("pid after cancel = %v, want missing", err)
	}
}

func TestWindowsWorkerExitKillsDescendants(t *testing.T) {
	cwd, fakeBin, state := windowsFixture(t, "slow")
	launch := runWindowsCLI(t, fakeBin, state, "slow", "--cwd", cwd, "--async", "--agent", "codex", "wait")
	id := launch.Metadata.JobID
	pids := waitForWindowsFakeState(t, state)
	workerPID, err := jobs.NewStore(cwd).ReadPID(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, pid := range append(pids, workerPID) {
		pid := pid
		t.Cleanup(func() {
			if process, err := os.FindProcess(pid); err == nil {
				_ = process.Kill()
			}
		})
	}
	worker, err := os.FindProcess(workerPID)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Kill(); err != nil {
		t.Fatal(err)
	}
	for _, pid := range pids {
		assertWindowsProcessGone(t, pid)
	}
}

func windowsFixture(t *testing.T, mode string) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	cwd := filepath.Join(root, "repo with spaces")
	fakeBin := filepath.Join(root, "bin with spaces")
	for _, path := range []string{cwd, fakeBin} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	source, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeBin, "codex.exe"), source, 0o755); err != nil {
		t.Fatal(err)
	}
	return cwd, fakeBin, filepath.Join(root, mode+" state.txt")
}

func runWindowsCLI(t *testing.T, fakeBin, state, mode string, args ...string) result.Result {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(),
		"PEERAGENT_TEST_HELPER_MAIN=1",
		"PEERAGENT_FAKE_MODE="+mode,
		"PEERAGENT_FAKE_STATE="+state,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("peeragent %v: %v\n%s", args, err, output)
	}
	var got result.Result
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode peeragent output: %v\n%s", err, output)
	}
	return got
}

func waitForWindowsFakeState(t *testing.T, path string) []int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(path)
		if err == nil {
			lines := strings.Fields(string(content))
			if len(lines) == 2 {
				var pids []int
				for _, line := range lines {
					_, value, ok := strings.Cut(line, "=")
					if !ok {
						t.Fatalf("invalid state %q", content)
					}
					pid, err := strconv.Atoi(value)
					if err != nil {
						t.Fatal(err)
					}
					pids = append(pids, pid)
				}
				return pids
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("fake Codex target did not write %s", path)
	return nil
}

func assertWindowsProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
		if err != nil {
			return
		}
		state, waitErr := syscall.WaitForSingleObject(h, 0)
		_ = syscall.CloseHandle(h)
		if waitErr == nil && state == syscall.WAIT_OBJECT_0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("process %d survived cancellation", pid)
}
