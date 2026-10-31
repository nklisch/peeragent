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
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestFrictionRetryOutcomes(t *testing.T) {
	for _, tc := range []struct {
		mode   string
		status result.Status
		calls  int
		reason string
	}{
		{"friction-usage", result.StatusUsageLimited, 1, "You’ve hit your usage limit. Visit https://chatgpt.com/settings/usage to purchase more credits or try again at Nov 3rd, 2026 11:48 PM."},
		{"friction-capacity-usage", result.StatusUsageLimited, 2, "You’ve hit your usage limit. Visit https://chatgpt.com/settings/usage to purchase more credits or try again at Nov 3rd, 2026 11:48 PM."},
		{"friction-capacity-always", result.StatusCapacity, 4, "Selected model is at capacity. Please try a different model."},
		{"friction-capacity-resumes", result.StatusSuccess, 4, ""},
		{"friction-last-error", result.StatusFailed, 1, "last error"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			cwd, record := fakeCodexEnvironment(t)
			t.Setenv("PEERAGENT_FAKE_MODE", tc.mode)
			var launched result.Result
			other := t.TempDir()
			if !t.Run("owned worker", func(t *testing.T) {
				launched = cliResult(t, cwd, "--async", "--retry-capacity", "3", "--capacity-backoff", "1ms", "work")

				for _, command := range []string{"--wait", "--status", "--result"} {
					got := cliResult(t, other, command, launched.Metadata.JobID)
					if got.Status != tc.status || got.FailureReason != tc.reason {
						t.Fatalf("%s: %#v", command, got)
					}
					if tc.status == result.StatusUsageLimited && (!strings.Contains(got.Summary, tc.reason) || got.FinalMessage != nil) {
						t.Fatalf("usage stop hidden: %#v", got)
					}
				}
				terminal := cliResult(t, other, "--result", launched.Metadata.JobID)
				if terminal.Metadata.CapacityRetries != tc.calls-1 {
					t.Fatalf("retry count: %#v", terminal)
				}
				var calls []struct {
					Args []string
					CWD  string
				}
				data, err := os.ReadFile(record)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &calls); err != nil {
					t.Fatal(err)
				}
				if len(calls) != tc.calls {
					t.Fatalf("attempts: %s", data)
				}
				for _, call := range calls[1:] {
					if !strings.Contains(strings.Join(call.Args, " "), "exec resume --json") || !strings.Contains(strings.Join(call.Args, " "), "fake-session") {
						t.Fatalf("retry lost session: %#v", call)
					}
				}
			}) {
				return
			}
			// Retained metadata must preserve the diagnostic even when local records
			// and the richer archived result are no longer available.
			if err := os.RemoveAll(filepath.Join(jobs.NewStore(cwd).Root, launched.Metadata.JobID)); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(os.Getenv("PEERAGENT_STATE_DIR"), "index", "results")); err != nil {
				t.Fatal(err)
			}
			got := cliResult(t, other, "--result", launched.Metadata.JobID)
			if got.Status != tc.status || got.FailureReason != tc.reason || !got.Metadata.IndexedOnly {
				t.Fatalf("indexed diagnostic: %#v", got)
			}
		})
	}
}

// Every command resolves the same store, including legacy folders while a new
// launch in the same checkout goes to the Git directory.
func TestGitJobStorageControls(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "git-directory"
		if legacy {
			name = "legacy-directory"
		}
		t.Run(name, func(t *testing.T) {
			cwd, _ := fakeCodexEnvironment(t)
			cmd := exec.Command("git", "init", "--quiet", cwd)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git init: %v %s", err, out)
			}
			store := jobs.NewStore(cwd)
			if legacy {
				store = jobs.Store{Root: filepath.Join(cwd, ".peeragent", "jobs")}
			}
			job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh", Resume: "fake-session"}, "prior task")
			if err != nil {
				t.Fatal(err)
			}
			if err := jobs.IndexJob(job.ID, cwd); err != nil {
				t.Fatal(err)
			}
			if err := jobs.IndexSession("codex", "fake-session", cwd); err != nil {
				t.Fatal(err)
			}
			caller := t.TempDir()
			cliResultTell(t, caller, job.ID, "retained reply")
			for _, args := range [][]string{{"messages", job.ID}, {"messages", job.ID, "--cwd", cwd}} {
				out, diagnostic, err := controlOutput(t, caller, args...)
				if err != nil || !strings.Contains(out, "retained reply") {
					t.Fatalf("messages: %s %s %v", out, diagnostic, err)
				}
			}
			status := cliResult(t, caller, "--status", job.ID)
			if status.Status != result.StatusRunning || status.Metadata.InboxUnread != 1 {
				t.Fatalf("running job: %#v", status)
			}
			if err := app.FinishJob(jobs.NewStore(cwd), job, result.Result{Status: result.StatusSuccess, Summary: "prior result", Metadata: result.Metadata{AgentSession: "fake-session"}}); err != nil {
				t.Fatal(err)
			}
			for _, flag := range []string{"--status", "--result", "--wait"} {
				for _, extra := range [][]string{nil, {"--cwd", cwd}} {
					got := cliResult(t, caller, append([]string{flag, job.ID}, extra...)...)
					if got.Status != result.StatusSuccess || got.Metadata.CWD != cwd {
						t.Fatalf("%s: %#v", flag, got)
					}
					if flag != "--status" && (got.Summary != "prior result" || len(got.Conversation) != 1) {
						t.Fatalf("result: %#v", got)
					}
				}
			}
			for _, args := range [][]string{{"--jobs", "--cwd", cwd}, {"--jobs", "--all"}} {
				out, diagnostic, err := controlOutput(t, caller, args...)
				rows := decodeJobLines(t, out)
				if err != nil || len(rows) != 1 || rows[0].JobID != job.ID {
					t.Fatalf("listing: %s %s %v", out, diagnostic, err)
				}
			}
			t.Setenv("PEERAGENT_FAKE_MODE", "friction-success")
			resumed := cliResult(t, caller, "--resume", "fake-session", "continue")
			if resumed.Status != result.StatusSuccess || resumed.Metadata.CWD != cwd || resumed.Metadata.Model != "gpt-6-astra" || resumed.Metadata.Effort != "xhigh" || resumed.Metadata.ResumeWarning != "" {
				t.Fatalf("resume: %#v", resumed)
			}
			newJob, err := jobs.NewStore(cwd).Load(resumed.Metadata.JobID)
			if err != nil {
				t.Fatal(err)
			}
			// Git reports canonical paths: macOS temp folders sit behind a /private link and Windows ones can use 8.3 names.
			canonical, err := filepath.EvalSymlinks(cwd)
			if err != nil {
				t.Fatal(err)
			}
			expected := filepath.Join(canonical, ".git", "peeragent", "jobs", newJob.ID)
			if filepath.Dir(newJob.LogPath) != expected {
				t.Fatalf("new job folder: %s, want %s", newJob.LogPath, expected)
			}
			old, err := jobs.NewStore(cwd).Load(job.ID)
			if err != nil || old.LogPath != job.LogPath {
				t.Fatalf("original folder changed: %#v %v", old, err)
			}
			matches, err := jobs.SessionJobs("codex", "fake-session", cwd)
			if err != nil || len(matches) != 2 {
				t.Fatalf("session jobs: %#v %v", matches, err)
			}
			// Local listing must find both locations without duplicate IDs.
			rows, err := app.NewService(app.Options{}).ListJobs(context.Background(), cwd, false)
			if err != nil || len(rows) != 2 {
				t.Fatalf("both locations: %#v %v", rows, err)
			}
			nested := filepath.Join(cwd, "nested")
			if err := os.Mkdir(nested, 0o755); err != nil {
				t.Fatal(err)
			}
			nestedJob := cliResult(t, caller, "--cwd", nested, "nested task")
			status = cliResult(t, cwd, "--status", nestedJob.Metadata.JobID, "--cwd", cwd)
			if status.Metadata.CWD != nested {
				t.Fatalf("lookup changed target cwd: %#v", status)
			}
			for _, all := range []bool{false, true} {
				rows, err = app.NewService(app.Options{}).ListJobs(context.Background(), cwd, all)
				if err != nil || len(rows) != 3 {
					t.Fatalf("same checkout scanned twice: %#v %v", rows, err)
				}
			}

			if !legacy {
				if _, err := os.Stat(filepath.Join(cwd, ".peeragent")); !os.IsNotExist(err) {
					t.Fatalf("working tree polluted: %v", err)
				}
			} else if _, err := os.Stat(filepath.Join(jobs.NewStore(cwd).Root, job.ID)); !os.IsNotExist(err) {
				t.Fatalf("legacy job was duplicated: %v", err)
			}
		})
	}
}
