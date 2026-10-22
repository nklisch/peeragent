package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/testsupport/terminalenv"
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
	mode := os.Getenv("PEERAGENT_FAKE_MODE")
	if mode == "messaging" || mode == "messaging-retry" {
		input, _ := io.ReadAll(os.Stdin)
		data, _ := json.Marshal(map[string]any{"prompt": string(input), "env": terminalenv.TerminalEnvSnapshot()})
		if mode == "messaging-retry" && len(calls) == 1 {
			for {
				if _, err := os.Stat(filepath.Join(os.Getenv("PEERAGENT_JOB_DIR"), "inbox.jsonl")); err == nil {
					break
				}
				if _, err := os.Stat(os.Getenv("PEERAGENT_FAKE_GATE")); err == nil {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Println(`{"type":"turn.failed","error":{"message":"Selected model is at capacity"}}`)
			os.Exit(1)
		}
		fmt.Printf("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":%q}}\n", string(data))
		fmt.Println(`{"type":"turn.completed"}`)
		return
	}
	if mode == "terminal-env-retry" {
		snapshot := terminalenv.TerminalEnvSnapshot()
		data, _ := json.Marshal(snapshot)
		path := fmt.Sprintf("%s-%d.json", os.Getenv("PEERAGENT_ENV_RECORD"), len(calls))
		if err := os.WriteFile(path, data, 0o600); err != nil {
			panic(err)
		}
		if len(calls) == 1 {
			fmt.Println(`{"type":"turn.failed","error":{"message":"Selected model is at capacity"}}`)
			os.Exit(1)
		}
		event, _ := json.Marshal(map[string]any{"type": "item.completed", "item": map[string]string{"type": "agent_message", "text": string(data)}})
		fmt.Println(string(event))
		fmt.Println(`{"type":"turn.completed"}`)
		return
	}

	if mode == "incident-direct" || mode == "incident-compaction" {
		if len(calls) == 1 {
			fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"The baseline link build is still queued."}}`)
			diagnostic := "Selected model is at capacity. Please try a different model."
			if mode == "incident-compaction" {
				diagnostic = "Error running remote compact task: " + diagnostic
			}
			fmt.Printf("{\"type\":\"error\",\"message\":%q}\n", diagnostic)
			fmt.Printf("{\"type\":\"turn.failed\",\"error\":{\"message\":%q}}\n", diagnostic)
			os.Exit(1)
		}
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"continued successfully"}}`)
		fmt.Println(`{"type":"turn.completed"}`)
		return
	}
	if mode == "continuing-turn" {
		fmt.Println(`{"type":"turn.completed"}`)
		fmt.Println(`{"type":"turn.started"}`)
		fmt.Println(`{"type":"agent_message","message":"continuation is running"}`)
		// Continue beyond the original five-second grace, then announce true
		// completion and remain alive to exercise the fresh watchdog deadline.
		time.Sleep(6 * time.Second)
		fmt.Println(`{"type":"task_complete","last_agent_message":"finished continuing"}`)
		waitForFakeTargetStop()
		return
	}
	if mode == "live" || mode == "final-hang" || mode == "failed-message" || mode == "capacity-message" || mode == "capacity-prior-message" {
		if mode == "capacity-prior-message" && len(calls) > 1 {
			fmt.Println(`{"type":"turn.failed","error":{"message":"ordinary target failure"}}`)
			os.Exit(1)
		}
		message := strings.Repeat("complete report — ", 12000)
		event, _ := json.Marshal(map[string]any{"type": "item.completed", "item": map[string]string{"type": "agent_message", "text": message}})
		fmt.Println(string(event))
		if mode == "failed-message" {
			os.Exit(1)
		}
		if (mode == "capacity-message" || mode == "capacity-prior-message") && len(calls) == 1 {
			fmt.Println(`{"type":"turn.failed","error":{"message":"Selected model is at capacity"}}`)
			os.Exit(1)
		}
		if mode == "final-hang" {
			fmt.Println(`{"type":"event_msg","payload":{"type":"task_complete"}}`)
		}
		waitForFakeTargetStop()
		fmt.Println(`{"type":"turn.completed"}`)
		return
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
	t.Setenv("PEERAGENT_FAKE_GATE", filepath.Join(root, "fake-target-stop"))
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
	res := decodeResult(t, out)
	for _, arg := range args {
		if arg == "--async" && res.Metadata.JobID != "" {
			cleanupAsyncJob(t, res.Metadata.CWD, res.Metadata.JobID)
			break
		}
	}
	return res
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

func waitForFakeTargetStop() {
	for {
		if _, err := os.Stat(os.Getenv("PEERAGENT_FAKE_GATE")); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
