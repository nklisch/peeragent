package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestIncidentCapacityStopsReportAndRetry(t *testing.T) {
	for _, mode := range []string{"incident-direct", "incident-compaction"} {
		for _, retry := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "-stop", true: "-retry"}[retry], func(t *testing.T) {
				cwd, _ := fakeCodexEnvironment(t)
				t.Setenv("PEERAGENT_FAKE_MODE", mode)
				args := []string{"--async"}
				if retry {
					args = append(args, "--retry-capacity", "1", "--capacity-backoff", "1ms")
				}
				launched := cliResult(t, cwd, append(args, "work")...)
				terminal := cliResult(t, cwd, "--wait", launched.Metadata.JobID)
				want := result.StatusCapacity
				if retry {
					want = result.StatusSuccess
				}
				if terminal.Status != want {
					t.Fatalf("terminal: %#v", terminal)
				}
				if retry {
					if terminal.Metadata.CapacityRetries != 1 || terminal.Details != "stdout:\ncontinued successfully" {
						t.Fatalf("retry: %#v", terminal)
					}
				} else {
					status := cliResult(t, cwd, "--status", launched.Metadata.JobID)
					fetched := cliResult(t, cwd, "--result", launched.Metadata.JobID)
					for _, res := range []result.Result{terminal, status, fetched} {
						if res.Status != result.StatusCapacity || res.Summary != "stopped: model at capacity; resumable with --resume fake-session" {
							t.Fatalf("capacity hidden: %#v", res)
						}
					}
					if fetched.Details != "stdout:\nThe baseline link build is still queued." {
						t.Fatalf("message lost: %#v", fetched)
					}
				}
			})
		}
	}
}

func TestStatusShowsLiveProgressInSingleAndMultipleOutput(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "live")
	launched := cliResult(t, cwd, "--async", "work")
	id := launched.Metadata.JobID
	t.Cleanup(func() { cliResult(t, cwd, "--cancel", id) })
	live := waitForObservation(t, cwd, id, "--status", func(res result.Result) bool { return res.Metadata.LastMessage != "" })
	if live.Status != result.StatusRunning || live.Metadata.LastEventAt == nil || !strings.HasPrefix(live.Metadata.LastMessage, "complete report") || len([]rune(live.Metadata.LastMessage)) > 200 || strings.Contains(live.Metadata.LastMessage, "\n") {
		t.Fatalf("live progress: %#v", live)
	}
	job, err := jobs.NewStore(cwd).Load(id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(job.LogPath)
	if err != nil || !strings.Contains(string(data), "complete report") {
		t.Fatalf("empty live agent.log: %q %v", data, err)
	}
	output := runCLIOutput(t, cwd, "--status", id, id)
	decoder := json.NewDecoder(strings.NewReader(output))
	for i := 0; i < 2; i++ {
		var record result.JobStatus
		if err := decoder.Decode(&record); err != nil {
			t.Fatal(err)
		}
		if record.LastEventAt == nil || record.LastMessage != live.Metadata.LastMessage {
			t.Fatalf("multi progress: %#v", record)
		}
	}
	text := runCLIOutput(t, cwd, "--status", id, id, "--text")
	if !strings.Contains(text, "LAST_EVENT") || !strings.Contains(text, "complete report") {
		t.Fatalf("text progress missing: %s", text)
	}
}

func runCLIOutput(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI: %v %s", err, output)
	}
	return string(output)
}

func TestEveryLaunchPrintsJobID(t *testing.T) {
	for _, resume := range []bool{false, true} {
		for _, async := range []bool{false, true} {
			t.Run(map[bool]string{false: "new", true: "resume"}[resume]+map[bool]string{false: "-blocking", true: "-async"}[async], func(t *testing.T) {
				cwd, _ := fakeCodexEnvironment(t)
				t.Setenv("PEERAGENT_FAKE_MODE", "incident-direct")
				// First fake call fails; seed its call history so this launch succeeds.
				if err := os.WriteFile(os.Getenv("PEERAGENT_FAKE_RECORD"), []byte(`[{}]`), 0o600); err != nil {
					t.Fatal(err)
				}
				args := []string{"--text"}
				if resume {
					args = append(args, "--resume", "previous-session")
				}
				if async {
					args = append(args, "--async")
				}
				args = append(args, "work")
				cmd := exec.Command(os.Args[0], args...)
				cmd.Dir = cwd
				output, err := cmd.Output()
				if err != nil {
					t.Fatalf("launch: %v %s", err, output)
				}
				text := string(output)
				if !strings.HasPrefix(text, "Alt Subagent: ") {
					t.Fatalf("text result prefix changed: %s", text)
				}
				var id string
				for _, line := range strings.Split(text, "\n") {
					if strings.HasPrefix(line, "- job_id: ") {
						id = strings.TrimPrefix(line, "- job_id: ")
					}
				}
				if jobs.ValidateID(id) != nil {
					t.Fatalf("launch omitted job id: %s", text)
				}
				terminal := cliResult(t, cwd, "--wait", id)
				if terminal.Metadata.JobID != id || terminal.Status != result.StatusSuccess {
					t.Fatalf("job identity: %#v", terminal)
				}
			})
		}
	}
}

func TestResumeGuardCLIExitAndOverride(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "live")
	launched := cliResult(t, cwd, "--async", "work")
	id := launched.Metadata.JobID
	t.Cleanup(func() { cliResult(t, cwd, "--cancel", id) })
	waitForObservation(t, cwd, id, "--status", func(res result.Result) bool { return res.Metadata.AgentSession == "fake-session" })
	other := t.TempDir()
	for _, async := range []bool{false, true} {
		args := []string{"--resume", "fake-session", "--cwd", other}
		if async {
			args = append(args, "--async")
		}
		args = append(args, "work")
		cmd := exec.Command(os.Args[0], args...)
		cmd.Dir = other
		output, err := cmd.CombinedOutput()
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
			t.Fatalf("guard exit: %v %s", err, output)
		}
		res := decodeResult(t, output)
		if res.Metadata.ExitCode != 2 || !strings.Contains(res.Summary, id) || !strings.Contains(res.Summary, "--cancel") {
			t.Fatalf("guard explanation: %#v", res)
		}
	}
	// Override intentionally permits the second job without touching the first.
	t.Setenv("PEERAGENT_FAKE_RECORD", filepath.Join(other, "calls.json"))
	second := cliResult(t, other, "--resume", "fake-session", "--cwd", other, "--async", "--allow-concurrent-session", "work")
	t.Cleanup(func() { cliResult(t, other, "--cancel", second.Metadata.JobID) })
	if second.Status != result.StatusRunning || second.Metadata.JobID == id {
		t.Fatalf("override: %#v", second)
	}
}

func TestCapacityRetryHelpAndResumeUsage(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	cmd := exec.Command(os.Args[0], "--resume", "prior", "--retry-capacity", "1", "work")
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("usage exit: %v %s", err, output)
	}
	if !strings.Contains(string(output), "add --async (also with --resume)") {
		t.Fatalf("unhelpful error: %s", output)
	}
	help := runCLIOutput(t, cwd, "--help")
	if !strings.Contains(help, "requires --async --agent codex,") || !strings.Contains(help, "including with --resume.") {
		t.Fatalf("missing prerequisite: %s", help)
	}
}
