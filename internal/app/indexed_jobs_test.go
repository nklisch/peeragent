package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestIndexedMetadataSurvivesRemovedWorktreeWithoutArchive(t *testing.T) {
	for _, status := range []result.Status{result.StatusSuccess, result.StatusFailed, result.StatusCapacity, result.StatusUsageLimited, result.StatusOutputLimit, result.StatusAuthRequired, result.StatusInterrupted, result.StatusCancelled, result.StatusRunning} {
		t.Run(string(status), func(t *testing.T) {
			state, cwd, replacement := t.TempDir(), t.TempDir(), t.TempDir()
			t.Setenv("PEERAGENT_STATE_DIR", state)
			launcher := &fakeLauncher{}
			service := NewService(Options{Launcher: launcher, Executable: func() (string, error) { return "fake", nil }})
			delegation := normalizedDelegation(t, input.Delegation{Agent: "codex", CWD: cwd, TaskText: "work", Model: "astra", Effort: "xhigh"})
			started, err := service.Launch(context.Background(), delegation)
			if err != nil {
				t.Fatal(err)
			}
			id := started.Metadata.JobID
			store := jobs.NewStore(cwd)
			if err := RecordJobProgress(store, id, executil.Progress{Event: "thread.started", AgentSession: "saved-conversation"}); err != nil {
				t.Fatal(err)
			}
			original, err := store.Load(id)
			if err != nil {
				t.Fatal(err)
			}
			if status != result.StatusRunning {
				if err := FinishJob(store, original, result.Result{Status: status, Summary: "final outcome", Details: "full details"}); err != nil {
					t.Fatal(err)
				}
			}
			// Remove only this test's worktree and full result archive. Metadata must
			// independently support discovery, summary reads and option inheritance.
			if err := os.RemoveAll(cwd); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(state, "index", "results")); err != nil {
				t.Fatal(err)
			}
			records, err := service.ListJobs(context.Background(), replacement, true)
			if err != nil || len(records) != 1 {
				t.Fatalf("list: %#v %v", records, err)
			}
			row := records[0]
			if row.JobID != id || row.Agent != "codex" || row.Model != "gpt-6-astra" || row.Effort != "xhigh" || row.AgentSession != "saved-conversation" || row.CWD != cwd || row.Status != status || !row.IndexedOnly || !row.StartedAt.Equal(original.CreatedAt) || row.IndexWarning != jobs.IndexedOnlyWarning {
				t.Fatalf("indexed row: %#v", row)
			}
			if (row.FinishedAt != nil) != (status != result.StatusRunning) {
				t.Fatalf("finish time: %#v", row)
			}
			if status != result.StatusRunning && row.Summary != "final outcome" {
				t.Fatal(row.Summary)
			}
			if !strings.Contains(result.FormatJobStatuses(records), jobs.IndexedOnlyWarning) {
				t.Fatal("text list hid metadata-only provenance")
			}
			for _, read := range []func(context.Context, JobRequest) (result.Result, error){service.JobResult, service.JobStatus} {
				got, err := read(context.Background(), JobRequest{JobID: id})
				if err != nil || got.Status != status || !got.Metadata.IndexedOnly || got.Metadata.AgentSession != row.AgentSession || got.Metadata.Model != row.Model || got.Metadata.Effort != row.Effort || got.Details != "" || got.FinalMessage != nil || got.Metadata.IndexWarning != jobs.IndexedOnlyWarning {
					t.Fatalf("read: %#v %v", got, err)
				}
			}
			// Reads and failed default-directory resumes must not recreate removed work.
			bad := normalizedDelegation(t, input.Delegation{Agent: "codex", CWD: cwd, TaskText: "continue", Resume: "saved-conversation"})
			if _, err := service.Launch(context.Background(), bad); err == nil || !strings.Contains(err.Error(), "--cwd") {
				t.Fatalf("removed-directory resume: %v", err)
			}
			resumed, err := service.Launch(context.Background(), normalizedDelegation(t, input.Delegation{Agent: "codex", CWD: replacement, TaskText: "continue", Resume: "saved-conversation"}))
			if err != nil || resumed.Metadata.Model != row.Model || resumed.Metadata.Effort != row.Effort || resumed.Metadata.ResumeWarning != "" || len(launcher.job.InboxSources) != 0 {
				t.Fatalf("resume: %#v %#v %v", resumed, launcher.job, err)
			}
			if _, err := os.Stat(cwd); !os.IsNotExist(err) {
				t.Fatalf("removed worktree recreated: %v", err)
			}
		})
	}
}

func TestLegacyArchivedMetadataImportedForListingAndResume(t *testing.T) {
	for _, listingFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "listing", false: "resume"}[listingFirst], func(t *testing.T) {
			t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
			oldDir, replacement := filepath.Join(t.TempDir(), "removed"), t.TempDir()
			const id = "20261024T010203Z-12345678"
			const session = "legacy-session"
			old := result.Result{Status: result.StatusSuccess, Summary: strings.Repeat("界", 700), Details: "full retained output", Metadata: result.Metadata{JobID: id, Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh", CWD: oldDir, AgentSession: session}}
			data, err := result.FormatJSON(old)
			if err != nil {
				t.Fatal(err)
			}
			// Use the old writer directly: there is deliberately no compact metadata.
			if err := jobs.ArchiveResult(id, data); err != nil {
				t.Fatal(err)
			}
			if err := jobs.IndexJob(id, oldDir); err != nil {
				t.Fatal(err)
			}
			if err := jobs.IndexSession("codex", session, oldDir); err != nil {
				t.Fatal(err)
			}
			launcher := &fakeLauncher{}
			service := NewService(Options{Launcher: launcher, Executable: func() (string, error) { return "fake", nil }})
			if listingFirst {
				// A legacy worker can finish after another command observed its
				// running state. Promote the archive without duplicating its row.
				if _, err := jobs.WriteIndexedJob(jobs.IndexedJob{ID: id, Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh", CWD: oldDir, Status: jobs.StatusRunning, AgentSession: session, StartedAt: time.Date(2026, 10, 24, 1, 2, 3, 123, time.UTC)}); err != nil {
					t.Fatal(err)
				}
				rows, err := service.ListJobs(context.Background(), replacement, true)
				if err != nil || len(rows) != 1 || rows[0].Summary != strings.Repeat("界", 599)+"…" || rows[0].AgentSession != session || rows[0].FinishedAt == nil || rows[0].StartedAt.IsZero() {
					t.Fatalf("legacy listing: %#v %v", rows, err)
				}
				// Once promoted, repeated discovery no longer needs to decode the full archive.
				if err := os.RemoveAll(filepath.Join(os.Getenv("PEERAGENT_STATE_DIR"), "index", "results")); err != nil {
					t.Fatal(err)
				}
			}
			res, err := service.Launch(context.Background(), normalizedDelegation(t, input.Delegation{Agent: "codex", CWD: replacement, TaskText: "continue", Resume: session}))
			if err != nil || res.Metadata.Model != "gpt-6-astra" || res.Metadata.Effort != "xhigh" || res.Metadata.ResumeWarning != "" || len(launcher.job.InboxSources) != 0 {
				t.Fatalf("legacy resume: %#v %v", res, err)
			}
			if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
				t.Fatalf("legacy directory recreated: %v", err)
			}
		})
	}
}

func TestLiveMetadataWinsAndCorruptionIsNotMasked(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh"}, "work")
	if err != nil {
		t.Fatal(err)
	}
	entry := jobs.JobIndexEntry(job)
	entry.Model = "gpt-6-luna"
	entry.Summary = "stale"
	entry.StartedAt = time.Now()
	if _, err := jobs.WriteIndexedJob(entry); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	rows, err := service.ListJobs(context.Background(), cwd, true)
	if err != nil || len(rows) != 1 || rows[0].IndexedOnly || rows[0].Model != job.Spec.Model {
		t.Fatalf("live precedence: %#v %v", rows, err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(job.LogPath), "job.json"), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListJobs(context.Background(), cwd, true); err == nil {
		t.Fatal("corrupt local metadata was hidden")
	}
}
