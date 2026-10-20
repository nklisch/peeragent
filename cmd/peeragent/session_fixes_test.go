package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// A copy of the test executable acts as Codex; no model or live job is used.
func fakeCodexMain() {
	path := os.Getenv("PEERAGENT_FAKE_RECORD")
	content, _ := os.ReadFile(path)
	var calls []struct {
		Args []string
		CWD  string
	}
	_ = json.Unmarshal(content, &calls)
	cwd, _ := os.Getwd()
	calls = append(calls, struct {
		Args []string
		CWD  string
	}{os.Args[1:], cwd})
	encoded, _ := json.Marshal(calls)
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		panic(err)
	}
	if os.Getenv("PEERAGENT_FAKE_MODE") != "no-session" {
		fmt.Println(`{"type":"thread.started","thread_id":"fake-session"}`)
	}
	if os.Getenv("PEERAGENT_FAKE_MODE") == "ordinary" {
		fmt.Println(`{"type":"error","message":"ordinary target failure"}`)
		os.Exit(1)
	}
	if os.Getenv("PEERAGENT_FAKE_MODE") == "ordinary-after-capacity" {
		fmt.Println(`{"type":"error","message":"Selected model is at capacity, retrying"}`)
		fmt.Println(`{"type":"turn.failed","error":{"message":"Authentication token expired"}}`)
		os.Exit(1)
	}
	if len(calls) == 1 || os.Getenv("PEERAGENT_FAKE_MODE") == "always" {
		fmt.Println(`{"type":"error","message":"Selected model is at capacity"}`)
		os.Exit(1)
	}
	fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"continued successfully"}}`)
	os.Exit(0)
}

func fakeCodexEnvironment(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", filepath.Join(root, "index"))
	record := filepath.Join(root, "calls.json")
	t.Setenv("PEERAGENT_FAKE_RECORD", record)
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "codex"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	content, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, name), content, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PEERAGENT_TEST_HELPER_MAIN", "1")
	return root, record
}

func cliResult(t *testing.T, dir string, args ...string) result.Result {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%q: %v\n%s", args, err, out)
	}
	return decodeResult(t, out)
}

func TestUnknownOptionsNeverStartAJob(t *testing.T) {
	cwd, record := fakeCodexEnvironment(t)
	for _, args := range [][]string{{"--list"}, {"--background", "--async"}, {"--resume", "", "--cwd", cwd}, {"--resume", "--cwd", cwd}} {
		cmd := exec.Command(os.Args[0], args...)
		cmd.Dir = cwd
		out, err := cmd.CombinedOutput()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 2 {
			t.Fatalf("%q exit = %v, output %s", args, err, out)
		}
		res := decodeResult(t, out)
		if res.Metadata.ExitCode != 2 {
			t.Fatalf("usage metadata: %#v", res)
		}
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("agent started: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".peeragent")); !os.IsNotExist(err) {
		t.Fatalf("job/log state created: %v", err)
	}
}

func TestAsyncCapacityRetryAndCrossDirectoryControls(t *testing.T) {
	for _, mode := range []string{"once", "always", "no-session", "default", "ordinary", "ordinary-after-capacity"} {
		t.Run(mode, func(t *testing.T) {
			cwd, record := fakeCodexEnvironment(t)
			t.Setenv("PEERAGENT_FAKE_MODE", mode)
			args := []string{"--async", "--full-access", "--model", "sol", "do work"}
			if mode != "default" {
				args = append(args, "--retry-capacity", "1", "--capacity-backoff", "1ms")
			}
			launched := cliResult(t, cwd, args...)
			other := t.TempDir()
			terminal := cliResult(t, other, "--wait", launched.Metadata.JobID)
			wantStatus, wantCalls := result.StatusSuccess, 2
			if mode != "once" {
				wantStatus = result.StatusCapacity
			}
			if mode == "no-session" || mode == "default" || mode == "ordinary" || mode == "ordinary-after-capacity" {
				wantCalls = 1
			}
			if mode == "ordinary" || mode == "ordinary-after-capacity" {
				wantStatus = result.StatusFailed
			}
			if terminal.Status != wantStatus || terminal.Metadata.JobID != launched.Metadata.JobID {
				t.Fatalf("terminal = %#v", terminal)
			}
			if terminal.Metadata.CapacityRetries != wantCalls-1 {
				t.Fatalf("retry count: %#v", terminal.Metadata)
			}
			for _, flag := range []string{"--status", "--result", "--cancel"} {
				got := cliResult(t, other, flag, launched.Metadata.JobID)
				if got.Status != wantStatus || got.Metadata.CWD != cwd {
					t.Fatalf("%s: %#v", flag, got)
				}
			}
			content, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			var calls []struct {
				Args []string
				CWD  string
			}
			if err := json.Unmarshal(content, &calls); err != nil {
				t.Fatal(err)
			}
			if len(calls) != wantCalls {
				t.Fatalf("calls: %s", content)
			}
			if wantCalls == 2 {
				args := strings.Join(calls[1].Args, " ")
				if calls[1].CWD != cwd || !strings.HasPrefix(args, "exec resume ") || !strings.Contains(args, "fake-session") || !strings.Contains(args, "--model gpt-6.1-sol") || !strings.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
					t.Fatalf("retry call: %#v", calls[1])
				}
			}
			if mode == "once" {
				resumed := cliResult(t, other, "--resume", terminal.Metadata.AgentSession, "continue")
				if resumed.Metadata.CWD != cwd {
					t.Fatalf("resumed in wrong folder: %#v", resumed)
				}
				override := cliResult(t, other, "--resume", terminal.Metadata.AgentSession, "--cwd", other, "continue")
				if override.Metadata.CWD != other {
					t.Fatalf("explicit override lost: %#v", override)
				}
			}
			job, err := jobs.NewStore(cwd).Load(launched.Metadata.JobID)
			if err != nil {
				t.Fatal(err)
			}
			if !jobs.IsTerminalStatus(job.Status) {
				t.Fatalf("nonterminal state: %#v", job)
			}
		})
	}
}
