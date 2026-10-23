package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestLaunchReportsRecordedMessagingState(t *testing.T) {
	for _, noPreamble := range []bool{false, true} {
		cwd := t.TempDir()
		s := NewService(Options{Launcher: &fakeLauncher{}, Executable: func() (string, error) { return "unused", nil }})
		res, err := s.Launch(context.Background(), input.Delegation{CWD: cwd, Agent: "codex", TaskText: "task", NoPreamble: noPreamble})
		if err != nil {
			t.Fatal(err)
		}
		job, err := jobs.NewStore(cwd).Load(res.Metadata.JobID)
		if err != nil {
			t.Fatal(err)
		}
		if res.Metadata.MessagingEnabled != job.MessagingEnabled || job.MessagingEnabled == noPreamble {
			t.Fatalf("launch %#v job %#v", res, job)
		}
		data, _ := result.FormatJSON(res)
		if !strings.Contains(string(data), `"messaging_enabled":`+map[bool]string{true: "true", false: "false"}[!noPreamble]) {
			t.Fatal(string(data))
		}
	}
}

func TestFinalMessageSummaryAndDetails(t *testing.T) {
	for _, text := range []string{" Fixed the issue.\nVerified with tests. ", strings.Repeat("界", 650)} {
		res := ResultFromExecution(input.Delegation{Agent: "codex"}, executil.Result{Stdout: text, Stderr: "diagnostic"}, nil)
		if len([]rune(res.Summary)) > 600 || !utf8.ValidString(res.Summary) {
			t.Fatal(res.Summary)
		}
		if len([]rune(strings.TrimSpace(text))) > 600 {
			if res.Summary != strings.Repeat("界", 599)+"…" {
				t.Fatal(res.Summary)
			}
		} else if res.Summary != strings.TrimSpace(text) {
			t.Fatal(res.Summary)
		}
		if res.Details != "stdout:\n"+text+"\n\nstderr:\ndiagnostic" {
			t.Fatal(res.Details)
		}
		if !strings.Contains(result.FormatText(res), text) {
			t.Fatal("text lost final handback")
		}
	}
}

func TestFinishPersistsPendingHeavyWarning(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // No real client: deterministic dashboard fixture.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"queued":[{"id":"queued-build","state":"queued","owners":["session"],"argv":["go","build"]}]}`))
	}))
	defer server.Close()
	t.Setenv("HEAVYD_DASHBOARD", server.URL)
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "session"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	res := result.Result{Status: result.StatusSuccess, Summary: "done", Metadata: result.Metadata{AgentSession: "session"}}
	if err := FinishJob(store, job, res); err != nil {
		t.Fatal(err)
	}
	got, err := NewService(Options{}).JobResult(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(got)
	if len(got.PendingHeavySteps) != 1 || !strings.Contains(string(data), `"pending_heavy_steps":[{"job":"queued-build","state":"queued","command":"go build"}]`) {
		t.Fatal(string(data))
	}
	if !strings.Contains(result.FormatText(got), "Warning: pending heavy step queued-build (queued): go build") {
		t.Fatal(result.FormatText(got))
	}
}

func TestProcessLauncherExportsHeavyLineage(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HEAVY_SESSION", "launcher")
	t.Setenv("HEAVY_PARENT_SESSIONS", "ancestor")
	cwd := t.TempDir()
	job, err := jobs.NewStore(cwd).Create(cwd, jobs.ExecSpec{}, "task")
	if err != nil {
		t.Fatal(err)
	}
	var cmd *exec.Cmd
	launcher := ProcessLauncher{command: func(string, ...string) *exec.Cmd { cmd = exec.Command(os.Args[0], "-test.run=^$"); return cmd }}
	worker, err := launcher.Start("unused", job)
	if err != nil {
		t.Fatal(err)
	}
	if code, err := worker.Wait(); err != nil || code != 0 {
		t.Fatalf("worker %d %v", code, err)
	}
	found := false
	for _, entry := range cmd.Env {
		if entry == "HEAVY_PARENT_SESSIONS=ancestor:launcher" {
			found = true
		}
	}
	if !found {
		t.Fatal("worker did not receive lineage")
	}
}
