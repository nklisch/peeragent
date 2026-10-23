package main

import (
	"context"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
)

func TestNoInboxRepliesPrependOnResumeAndFinalSummary(t *testing.T) {
	fakeCodexEnvironment(t)
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "legacy-session", NoPreamble: true}, "old task")
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range []string{"first reply", "second reply"} {
		out, _, err := controlOutput(t, cwd, "tell", job.ID, message)
		if err != nil || !strings.Contains(out, "queued for the next resume (this job has no inbox)") {
			t.Fatalf("receipt %q %v", out, err)
		}
	}
	mailbox, err := store.Mailbox(job.ID)
	if err != nil || mailbox.UnreadCount() != 2 {
		t.Fatalf("queued %#v %v", mailbox, err)
	}
	job.Status = jobs.StatusComplete
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	prior := applicationService
	t.Cleanup(func() { applicationService = prior })
	final := "Implemented the requested fixes; all focused checks passed."
	applicationService = app.NewService(app.Options{
		Executable: func() (string, error) { return "unused", nil },
		Launcher:   messagingLauncher(func(jobs.Job) error { return nil }),
		Executor: messagingExecutor(func(_ context.Context, request input.Delegation) (executil.Result, error) {
			for _, message := range []string{"first reply", "second reply"} {
				if strings.Count(request.TaskText, message) != 1 || strings.Index(request.TaskText, message) > strings.Index(request.TaskText, "resume work") {
					t.Fatalf("not prepended: %q", request.TaskText)
				}
			}
			if !strings.Contains(request.TaskText, "messages sent while the job ran") {
				t.Fatal(request.TaskText)
			}
			return executil.Result{TurnStarted: true, Stdout: final, AgentSession: "legacy-session"}, nil
		}),
	})
	launched, err := applicationService.Launch(context.Background(), input.Delegation{CWD: cwd, Agent: "codex", Resume: "legacy-session", TaskText: "resume work"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: launched.Metadata.JobID}); err != nil {
		t.Fatal(err)
	}
	mailbox, err = store.Mailbox(job.ID)
	if err != nil || mailbox.UnreadCount() != 0 {
		t.Fatalf("not acknowledged %#v %v", mailbox, err)
	}
	for _, command := range []string{"--result", "--wait"} {
		res := cliResult(t, cwd, command, launched.Metadata.JobID)
		if res.Summary != final || res.Details != "stdout:\n"+final {
			t.Fatalf("%s: %#v", command, res)
		}
		out, diagnostic, err := controlOutput(t, cwd, command, launched.Metadata.JobID, "--text")
		if err != nil || !strings.Contains(out, "Summary:\n"+final) {
			t.Fatalf("text %q %q %v", out, diagnostic, err)
		}
	}
}
