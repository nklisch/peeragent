package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/version"
)

func controlOutput(t *testing.T, cwd string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = cwd
	var out, diagnostic bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &diagnostic
	err := cmd.Run()
	return out.String(), diagnostic.String(), err
}

func TestTellReceiptAndLegacyStatus(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	modern := indexedTestJob(t, cwd)
	legacy := indexedTestJob(t, cwd)
	// Remove the field entirely, as a pre-v0.11 job would have been written.
	path := filepath.Join(filepath.Dir(legacy.LogPath), "job.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "messaging_enabled")
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		job     jobs.Job
		warning bool
	}{{modern, false}, {legacy, true}} {
		for n := 1; n <= 2; n++ {
			out, diagnostic, err := controlOutput(t, cwd, "tell", tc.job.ID, "answer")
			contextNote := ""
			if !tc.warning {
				contextNote = "; " + jobs.InboxReadContext
			}
			prefix := fmt.Sprintf("delivered to job %s (codex, running); unread %d, oldest ", tc.job.ID, n)
			suffix := contextNote + "\n"
			age := strings.TrimSuffix(strings.TrimPrefix(out, prefix), suffix)
			_, ageErr := time.ParseDuration(age)
			if err != nil || !strings.HasPrefix(out, prefix) || !strings.HasSuffix(out, suffix) || ageErr != nil || strings.Contains(diagnostic, jobs.LegacyMessagingWarning) != tc.warning {
				t.Fatalf("receipt: %q %q %v (age %v)", out, diagnostic, err, ageErr)
			}
		}
		res := cliResult(t, cwd, "--status", tc.job.ID)
		if res.Metadata.MessagingEnabled == tc.warning || (res.Metadata.MessagingWarning != "") != tc.warning || res.Metadata.Version != version.Version || res.Metadata.InboxUnread != 2 {
			t.Fatalf("status: %#v", res)
		}
		out, diagnostic, err := controlOutput(t, cwd, "--status", tc.job.ID, "--text")
		if err != nil || strings.Contains(out, jobs.LegacyMessagingWarning) != tc.warning {
			t.Fatalf("text status: %s %s %v", out, diagnostic, err)
		}
	}
	legacy.FinalReceived = true
	legacy.MessagingEnabled = false
	if err := jobs.NewStore(cwd).Save(legacy); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, err := controlOutput(t, cwd, "--status", modern.ID, legacy.ID, "--text")
	if err != nil || !strings.Contains(out, jobs.LegacyMessagingWarning) || !strings.Contains(out, "final message received") {
		t.Fatalf("multi text: %q %q %v", out, diagnostic, err)
	}
	out, diagnostic, err = controlOutput(t, cwd, "--status", modern.ID, legacy.ID)
	if err != nil {
		t.Fatalf("multi status: %s %s %v", out, diagnostic, err)
	}
	records := decodeJobLines(t, out)
	if len(records) != 2 || records[0].MessagingWarning != "" || records[1].MessagingWarning != jobs.LegacyMessagingWarning || records[1].Version != version.Version {
		t.Fatalf("multi status: %#v", records)
	}
}

func decodeJobLines(t *testing.T, out string) []result.JobStatus {
	t.Helper()
	var records []result.JobStatus
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var record result.JobStatus
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func TestJobsDirectoryScopeLiveSessionAndVersion(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd, other := t.TempDir(), t.TempDir()
	first := indexedTestJob(t, cwd)
	second := indexedTestJob(t, other)
	// An unindexed local job must also be discovered.
	third, err := jobs.NewStore(cwd).Create(cwd, jobs.ExecSpec{Agent: "claude", Model: "opus"}, "work")
	if err != nil {
		t.Fatal(err)
	}
	first.CreatedAt = first.CreatedAt.Add(-time.Hour)
	if err := jobs.NewStore(cwd).Save(first); err != nil {
		t.Fatal(err)
	}
	if err := app.RecordJobProgress(jobs.NewStore(other), second.ID, executil.Progress{Event: "thread.started", AgentSession: "live-session"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(second.LogPath), "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"agent_session": "live-session"`)) {
		t.Fatalf("session absent from live job: %s", data)
	}
	if err := app.NewService(app.Options{}).Tell(context.Background(), app.JobRequest{CWD: other, JobID: second.ID}, "launcher", "latest reply"); err != nil {
		t.Fatal(err)
	}
	terminalTestJob(t, first, result.StatusOutputLimit)
	if err := jobs.IndexJob("20000101T000000Z-00000000", filepath.Join(t.TempDir(), "removed")); err != nil {
		t.Fatal(err)
	}
	// Multiple index entries pointing at one directory must not duplicate rows.
	if err := jobs.IndexJob("20000101T000000Z-00000001", cwd); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args  []string
		count int
	}{
		{[]string{"--jobs"}, 2},
		{[]string{"--jobs", "--cwd", other}, 1},
		{[]string{"--jobs", "--all"}, 3},
	} {
		out, diagnostic, err := controlOutput(t, cwd, tc.args...)
		if err != nil {
			t.Fatalf("list: %s %s %v", out, diagnostic, err)
		}
		records := decodeJobLines(t, out)
		if len(records) != tc.count {
			t.Fatalf("%v: %#v", tc.args, records)
		}
		for _, r := range records {
			if r.Version != version.Version || r.Agent == "" || r.StartedAt.IsZero() {
				t.Fatalf("missing list fields: %#v", r)
			}
			if r.JobID == second.ID && (r.AgentSession != "live-session" || r.LastMessageAt == nil || r.InboxUnread != 1) {
				t.Fatalf("live session: %#v", r)
			}
			if r.JobID == first.ID && (r.Status != result.StatusOutputLimit || !strings.Contains(r.Summary, "output")) {
				t.Fatalf("terminal list: %#v", r)
			}
		}
		if tc.count == 2 && (records[0].JobID != third.ID || records[1].JobID != first.ID) {
			t.Fatalf("order: %#v", records)
		}
	}
	out, diagnostic, err := controlOutput(t, cwd, "--jobs", "--all", "--text")
	for _, want := range []string{"AGENT", "MODEL", "LAST_MESSAGE_AT", "AGENT_SESSION", second.ID, third.ID, "live-session"} {
		if err != nil || !strings.Contains(out, want) {
			t.Fatalf("text list missing %s: %s %s %v", want, out, diagnostic, err)
		}
	}
	out, diagnostic, err = controlOutput(t, cwd, "--version")
	if err != nil || out != version.Version+"\n" || diagnostic != "" {
		t.Fatalf("version: %q %q %v", out, diagnostic, err)
	}
	empty := t.TempDir()
	out, diagnostic, err = controlOutput(t, empty, "--jobs")
	if err != nil || out != "" {
		t.Fatalf("empty list: %q %q %v", out, diagnostic, err)
	}
	out, diagnostic, err = controlOutput(t, cwd, "--jobs", "--unknown")
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 {
		t.Fatalf("unknown flag: %q %q %v", out, diagnostic, err)
	}
}

func TestOutputLimitRecoveryBudget(t *testing.T) {
	for _, tc := range []struct {
		name     string
		budget   int
		outcomes []result.Status
		calls    int
		want     result.Status
	}{
		{"disabled", 0, []result.Status{result.StatusOutputLimit}, 1, result.StatusOutputLimit},
		{"continue once", 1, []result.Status{result.StatusOutputLimit, result.StatusSuccess}, 2, result.StatusSuccess},
		{"limit stays stopped", 5, []result.Status{result.StatusOutputLimit, result.StatusOutputLimit}, 2, result.StatusOutputLimit},
		{"capacity shares budget", 1, []result.Status{result.StatusCapacity, result.StatusOutputLimit}, 2, result.StatusOutputLimit},
		{"capacity then limit", 2, []result.Status{result.StatusCapacity, result.StatusOutputLimit, result.StatusSuccess}, 3, result.StatusSuccess},
		{"limit then capacity", 2, []result.Status{result.StatusOutputLimit, result.StatusCapacity, result.StatusSuccess}, 3, result.StatusSuccess},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Model: "gpt-6.1-sol", FullAccess: true, RetryCapacity: tc.budget, CapacityBackoff: time.Nanosecond}, "original task")
			if err != nil {
				t.Fatal(err)
			}
			prior := applicationService
			t.Cleanup(func() { applicationService = prior })
			calls := 0
			applicationService = app.NewService(app.Options{Executor: messagingExecutor(func(ctx context.Context, req input.Delegation) (executil.Result, error) {
				if calls >= len(tc.outcomes) {
					t.Fatal("unexpected extra attempt")
				}
				if calls > 0 {
					if req.Resume != "limited-session" || !req.FullAccess || req.Model != "gpt-6.1-sol" || strings.Contains(req.TaskText, "original task") {
						t.Fatalf("retry lost options/replayed task: %#v", req)
					}
					if tc.outcomes[calls-1] == result.StatusOutputLimit && !strings.Contains(req.TaskText, "Finish your handback concisely") {
						t.Fatalf("wrong continuation: %s", req.TaskText)
					}
				}
				outcome := tc.outcomes[calls]
				calls++
				return executil.Result{TurnStarted: true, OutputLimit: outcome == result.StatusOutputLimit, CapacityExhausted: outcome == result.StatusCapacity, AgentSession: "limited-session", Stdout: "partial handback"}, nil
			})})
			if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: job.ID}); err != nil {
				t.Fatal(err)
			}
			res, err := app.NewService(app.Options{}).JobResult(context.Background(), app.JobRequest{CWD: cwd, JobID: job.ID})
			if err != nil || calls != tc.calls || res.Status != tc.want || res.Metadata.CapacityRetries != tc.calls-1 || res.Metadata.AgentSession != "limited-session" {
				t.Fatalf("result calls=%d: %#v %v", calls, res, err)
			}
			if tc.want == result.StatusOutputLimit && (!strings.Contains(res.Summary, `peeragent --agent codex --resume limited-session "Continue."`) || res.Metadata.ExitCode != 1 || !strings.Contains(res.Details, "partial handback")) {
				t.Fatalf("stop has no recovery: %#v", res)
			}
		})
	}
}

func TestOutputLimitWithoutSessionDoesNotRetry(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", RetryCapacity: 5}, "task")
	if err != nil {
		t.Fatal(err)
	}
	prior := applicationService
	t.Cleanup(func() { applicationService = prior })
	calls := 0
	applicationService = app.NewService(app.Options{Executor: messagingExecutor(func(context.Context, input.Delegation) (executil.Result, error) {
		calls++
		return executil.Result{TurnStarted: true, OutputLimit: true}, nil
	})})
	if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: job.ID}); err != nil {
		t.Fatal(err)
	}
	res, err := applicationService.JobResult(context.Background(), app.JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || calls != 1 || res.Status != result.StatusOutputLimit || res.Metadata.AgentSession != "" || strings.Contains(res.Summary, "--resume") {
		t.Fatalf("missing-session recovery: calls=%d %#v %v", calls, res, err)
	}
}
