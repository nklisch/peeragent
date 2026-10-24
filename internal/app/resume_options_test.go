package app

import (
	"context"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
)

func TestResumeInheritsLatestJobOptions(t *testing.T) {
	for _, tc := range []struct {
		agent, model, effort string
	}{
		{"codex", "gpt-6-astra", "xhigh"},
		{"claude", "opus", "high"},
		{"gemini", "gemini-3.1-pro", "low"},
		{"zai", "glm-5.2", "xhigh"},
		{"codex", "", "medium"}, // No model override is itself a recorded choice.
	} {
		t.Run(tc.agent+"/"+tc.model, func(t *testing.T) {
			t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
			oldDir, latestDir, launchDir := t.TempDir(), t.TempDir(), t.TempDir()
			created := time.Date(2026, 10, 24, 1, 0, 0, 0, time.UTC)
			for i, dir := range []string{oldDir, latestDir} {
				store := jobs.NewStore(dir)
				job, err := store.Create(dir, jobs.ExecSpec{Agent: tc.agent, Resume: "same-session", Model: tc.model, Effort: tc.effort}, "old work")
				if err != nil {
					t.Fatal(err)
				}
				job.CreatedAt = created.Add(time.Duration(i) * time.Hour)
				job.Status = jobs.StatusComplete
				if i == 0 {
					job.Spec.Model, job.Spec.Effort = "obsolete-model", "obsolete-effort"
				}
				if err := store.Save(job); err != nil {
					t.Fatal(err)
				}
				if err := jobs.IndexJob(job.ID, dir); err != nil {
					t.Fatal(err)
				}
			}
			// The session index can still point at an older directory. Explicit
			// --cwd moves the resumed turn without changing its option inheritance.
			if err := jobs.IndexSession(tc.agent, "same-session", oldDir); err != nil {
				t.Fatal(err)
			}
			launcher := &fakeLauncher{}
			service := NewService(Options{Launcher: launcher, Executable: func() (string, error) { return "fake", nil }})
			delegation := normalizedDelegation(t, input.Delegation{Agent: tc.agent, CWD: launchDir, TaskText: "continue", Resume: "same-session"})
			res, err := service.Launch(context.Background(), delegation)
			if err != nil {
				t.Fatal(err)
			}
			if res.Metadata.Model != tc.model || res.Metadata.Effort != tc.effort || res.Metadata.ResumeWarning != "" || res.Metadata.CWD != launchDir {
				t.Fatalf("launch metadata = %#v", res.Metadata)
			}
			if launcher.job.Spec.Model != tc.model || launcher.job.Spec.Effort != tc.effort || len(launcher.job.InboxSources) != 2 {
				t.Fatalf("launched job = %#v", launcher.job)
			}
			loaded, err := jobs.NewStore(launchDir).Load(launcher.job.ID)
			if err != nil || loaded.Spec.Model != tc.model || loaded.Spec.Effort != tc.effort {
				t.Fatalf("saved job = %#v, %v", loaded, err)
			}
		})
	}
}

func TestResumeOverridesAndCombinedValidation(t *testing.T) {
	for _, tc := range []struct {
		name, agent, previousModel, previousEffort, model, effort, wantModel, wantEffort string
		wantError                                                                        bool
	}{
		{"effort-only", "codex", "gpt-6-astra", "xhigh", "", "high", "gpt-6-astra", "high", false},
		{"model-only", "codex", "gpt-6-astra", "xhigh", "luna", "", "gpt-6-luna", "xhigh", false},
		{"both", "codex", "gpt-6-astra", "xhigh", "sol", "low", "gpt-6.1-sol", "low", false},
		{"model-equals-default", "gemini", "gemini-3.1-pro", "low", "flash", "", "gemini-3.7-flash", "low", false},
		{"incompatible-effort-override", "gemini", "gemini-3.1-pro", "high", "", "medium", "", "", true},
		{"incompatible-model-override", "gemini", "gemini-3.7-flash", "medium", "pro", "", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			delegation := normalizedDelegation(t, input.Delegation{TaskText: "continue", CWD: t.TempDir(), Agent: tc.agent, Resume: "session", Model: tc.model, Effort: tc.effort})
			got, warning, err := resumeOptions(delegation, []jobs.Job{{Spec: jobs.ExecSpec{Model: tc.previousModel, Effort: tc.previousEffort}}})
			if tc.wantError {
				if err == nil {
					t.Fatal("accepted incompatible inherited options")
				}
				return
			}
			if err != nil || warning != "" || got.Model != tc.wantModel || got.Effort != tc.wantEffort || got.InheritEffort || got.InheritModel {
				t.Fatalf("resolved = %#v, warning %q, error %v", got, warning, err)
			}
		})
	}
}
