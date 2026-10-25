package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestRemovedWorktreeJobsAndResumeCLI(t *testing.T) {
	_, callsFile := fakeCodexEnvironment(t)
	cwd, caller, replacement := t.TempDir(), t.TempDir(), t.TempDir()
	job := indexedTestJob(t, cwd)
	job.Spec.Model, job.Spec.Effort = "gpt-6-astra", "xhigh"
	if err := jobs.NewStore(cwd).Save(job); err != nil {
		t.Fatal(err)
	}
	if err := app.RecordJobProgress(jobs.NewStore(cwd), job.ID, executil.Progress{Event: "thread.started", AgentSession: "removed-session"}); err != nil {
		t.Fatal(err)
	}
	terminalTestJob(t, job, result.StatusSuccess)
	if err := os.RemoveAll(cwd); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(os.Getenv("PEERAGENT_STATE_DIR"), "index", "results")); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, err := controlOutput(t, caller, "--jobs", "--all", "--json")
	if err != nil {
		t.Fatalf("listing: %s %s %v", out, diagnostic, err)
	}
	rows := decodeJobLines(t, out)
	if len(rows) != 1 || rows[0].JobID != job.ID || rows[0].AgentSession != "removed-session" || rows[0].Model != "gpt-6-astra" || rows[0].Effort != "xhigh" || rows[0].Summary != "finished" || rows[0].FinishedAt == nil || !rows[0].IndexedOnly {
		t.Fatalf("rows: %#v", rows)
	}
	for _, command := range []string{"--status", "--result", "--wait"} {
		res := cliResult(t, caller, command, job.ID)
		if res.Summary != "finished" || res.Metadata.Model != "gpt-6-astra" || res.Metadata.Effort != "xhigh" || res.Metadata.AgentSession != "removed-session" || !res.Metadata.IndexedOnly || res.Metadata.FinishedAt == nil || res.Metadata.IndexWarning != jobs.IndexedOnlyWarning {
			t.Fatalf("%s: %#v", command, res)
		}
	}
	out, diagnostic, err = controlOutput(t, caller, "--jobs", "--all", "--text")
	if err != nil || !strings.Contains(out, jobs.IndexedOnlyWarning) {
		t.Fatalf("text: %s %s %v", out, diagnostic, err)
	}
	t.Setenv("PEERAGENT_FAKE_MODE", "ordinary")
	out, diagnostic, err = controlOutput(t, caller, "--resume", "removed-session", "--cwd", replacement, "--no-preamble", "continue")
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 {
		t.Fatalf("resume: %s %s %v", out, diagnostic, err)
	}
	resumed := decodeResult(t, []byte(out))
	if resumed.Metadata.Model != "gpt-6-astra" || resumed.Metadata.Effort != "xhigh" || resumed.Metadata.ResumeWarning != "" {
		t.Fatalf("resumed options: %#v", resumed)
	}
	data, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatal(err)
	}
	var calls []struct {
		Args []string
		CWD  string
	}
	if err := json.Unmarshal(data, &calls); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].CWD != replacement || !strings.Contains(strings.Join(calls[0].Args, " "), "--model gpt-6-astra") || !strings.Contains(strings.Join(calls[0].Args, " "), `model_reasoning_effort="xhigh"`) {
		t.Fatalf("target invocation: %#v", calls)
	}
	if _, err := os.Stat(cwd); !os.IsNotExist(err) {
		t.Fatalf("removed directory recreated: %v", err)
	}
}

func TestWaitReturnsIndexedRunningHistoryWithoutPolling(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	job := indexedTestJob(t, cwd)
	if err := app.RecordJobProgress(jobs.NewStore(cwd), job.ID, executil.Progress{Event: "thread.started", AgentSession: "lost-running-session"}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(cwd); err != nil {
		t.Fatal(err)
	}
	// There is no worker or timer to settle: only last-known history remains.
	ctx := context.Background()
	prior := applicationService
	t.Cleanup(func() { applicationService = prior })
	applicationService = app.NewService(app.Options{})
	req := input.Request{WaitJobID: job.ID, CWD: cwd, JSON: true}
	if err := waitForJobResult(ctx, req); err != nil {
		t.Fatal(err)
	}
}
