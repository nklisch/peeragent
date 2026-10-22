package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func indexedTestJob(t *testing.T, cwd string) jobs.Job {
	t.Helper()
	job, err := jobs.NewStore(cwd).Create(cwd, jobs.ExecSpec{Agent: "codex", Model: "gpt-6.1-sol", Effort: "high", JSON: true}, "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := jobs.IndexJob(job.ID, cwd); err != nil {
		t.Fatal(err)
	}
	return job
}

func terminalTestJob(t *testing.T, job jobs.Job, status result.Status) {
	t.Helper()
	if err := app.FinishJob(jobs.NewStore(job.CWD), job, result.Result{Status: status, Summary: "finished", ChangedFiles: []string{}, Verification: []result.Verification{}, Metadata: result.Metadata{JobID: job.ID, CWD: job.CWD, ExitCode: 7}}); err != nil {
		t.Fatal(err)
	}
}

func TestWaitAnyReturnsFirstTerminalAcrossDirectories(t *testing.T) {
	fakeCodexEnvironment(t)
	first := indexedTestJob(t, t.TempDir())
	second := indexedTestJob(t, t.TempDir())
	terminalTestJob(t, second, result.StatusFailed)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "--wait-any", first.ID, second.ID)
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wait: %v\n%s", err, out)
	}
	res := decodeResult(t, out)
	if res.Status != result.StatusFailed || res.Metadata.JobID != second.ID || res.Metadata.ExitCode != 7 {
		t.Fatalf("wrong terminal job: %#v", res)
	}
	// Failed results, like --wait, retain the historical wrapper exit 0.
	cmd = exec.CommandContext(ctx, os.Args[0], "--wait-any", first.ID, second.ID, "--text")
	out, err = cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "job_id: "+second.ID) {
		t.Fatalf("text winner id: %v %s", err, out)
	}
}

func TestWaitAnyWaitsForLaterCompletion(t *testing.T) {
	fakeCodexEnvironment(t)
	first := indexedTestJob(t, t.TempDir())
	second := indexedTestJob(t, t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "--wait-any", first.ID, second.ID+"@"+second.CWD)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	terminalTestJob(t, second, result.StatusSuccess)
	out, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait: %v %s", err, out)
	}
	if got := decodeResult(t, out); got.Metadata.JobID != second.ID || got.Status != result.StatusSuccess {
		t.Fatalf("winner: %#v", got)
	}
}

func TestWaitAnyValidatesAllIDsBeforeReturningTerminal(t *testing.T) {
	fakeCodexEnvironment(t)
	job := indexedTestJob(t, t.TempDir())
	terminalTestJob(t, job, result.StatusSuccess)
	for _, refs := range [][]string{{job.ID, "20000101T000000Z-00000000"}, {"20000101T000000Z-00000000", job.ID}} {
		cmd := exec.Command(os.Args[0], append([]string{"--wait-any"}, refs...)...)
		cmd.Dir = t.TempDir()
		out, err := cmd.CombinedOutput()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 2 {
			t.Fatalf("unknown id exit: %v %s", err, out)
		}
		if got := decodeResult(t, out); got.Metadata.ExitCode != 2 {
			t.Fatalf("usage result: %#v", got)
		}
	}
}

func TestMultipleStatusJSONAndText(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	for _, args := range [][]string{{"init"}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = cwd
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	first := indexedTestJob(t, cwd)
	second := indexedTestJob(t, t.TempDir())
	until := time.Now().Add(time.Hour)
	first.CapacityBackoffUntil = &until
	first.Spec.Resume = "live-session"
	if err := jobs.NewStore(cwd).Save(first); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "--status", first.ID, second.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status: %v %s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Fatalf("not one record per job: %s", out)
	}
	var r result.JobStatus
	if err := json.Unmarshal([]byte(lines[0]), &r); err != nil {
		t.Fatal(err)
	}
	if r.JobID != first.ID || r.AgentSession != "live-session" || !r.CapacityBackoff || r.Model != "gpt-6.1-sol" || r.Effort != "high" || !r.StartedAt.Equal(first.CreatedAt) || len(r.HeadCommit) != 40 {
		t.Fatalf("compact record: %#v", r)
	}
	cmd = exec.Command(os.Args[0], "--text", "--status", first.ID, second.ID+"@"+second.CWD)
	out, err = cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "AGENT_SESSION") || !strings.Contains(string(out), "live-session") || !strings.Contains(string(out), second.ID) {
		t.Fatalf("table: %v %s", err, out)
	}
}
