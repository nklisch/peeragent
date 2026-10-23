package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/testsupport"
	"github.com/nklisch/peeragent/internal/testsupport/terminalenv"
)

func TestMain(m *testing.M) {
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "herdr" {
		fakeHerdrMain()
		return
	}
	if os.Getenv("PEERAGENT_FAKE_MODE") != "terminal-env-retry" && terminalenv.WriteTerminalEnvAgentSnapshot() {
		return
	}
	if filepath.Base(os.Args[0]) == "codex" || filepath.Base(os.Args[0]) == "codex.exe" {
		fakeCodexMain()
		return
	}
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "claude" {
		fakeClaudeMain()
		return
	}
	if testHelperMainRequested(os.Getenv("PEERAGENT_TEST_HELPER_MAIN"), os.Args) {
		recordTestWorkerPID()
		if len(os.Args) > 1 && os.Args[1] == "--job-run" && os.Getenv("PEERAGENT_FAKE_WORKER_EXIT") == "1" {
			fmt.Fprintln(os.Stderr, "fake worker exited before writing its result")
			os.Exit(23)
		}
		main()
		return
	}
	os.Exit(testsupport.RunWithPrivateState(m.Run))
}

func testHelperMainRequested(marker string, args []string) bool {
	if marker != "1" {
		return false
	}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-test.") {
			return false
		}
	}
	return true
}

func TestTestHelperMainRequestedRejectsInheritedGoTestArgs(t *testing.T) {
	if !testHelperMainRequested("1", []string{"peeragent.test", "task"}) {
		t.Fatal("intentional CLI subprocess should enter helper main")
	}
	if testHelperMainRequested("1", []string{"peeragent.test", "-test.run", "^$"}) {
		t.Fatal("go test subprocess should not enter helper main")
	}
	if testHelperMainRequested("", []string{"peeragent.test", "task"}) {
		t.Fatal("subprocess without helper marker should not enter helper main")
	}
}

func TestRemovedMCPModeFailsWithoutDelegating(t *testing.T) {
	cmd := exec.Command(os.Args[0], "mcp")
	cmd.Env = append(os.Environ(), "PEERAGENT_TEST_HELPER_MAIN=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected removed MCP mode to fail, output: %s", output)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("exit error = %v, want code 2; output: %s", err, output)
	}
	if !strings.Contains(string(output), "MCP mode is not supported") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestWaitReturnsTerminalAsyncResult(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	want := result.Result{
		Status:       result.StatusSuccess,
		Summary:      "completed",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, Agent: "codex", JobID: job.ID},
	}
	writeDone := make(chan error, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		writeDone <- app.WriteJobResult(job.ResultPath, want)
	}()

	cmd := exec.Command(os.Args[0], "--cwd", cwd, "--wait", job.ID)
	cmd.Env = append(os.Environ(), "PEERAGENT_TEST_HELPER_MAIN=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wait failed: %v\n%s", err, output)
	}
	if err := <-writeDone; err != nil {
		t.Fatal(err)
	}
	got := decodeResult(t, output)
	if got.Status != result.StatusSuccess || got.Summary != "completed" || got.Metadata.JobID != job.ID {
		t.Fatalf("wait result = %#v", got)
	}
}

func decodeResult(t *testing.T, output []byte) result.Result {
	t.Helper()
	var res result.Result
	if err := json.Unmarshal(output, &res); err != nil {
		t.Fatalf("decode result: %v\n%s", err, output)
	}
	return res
}
