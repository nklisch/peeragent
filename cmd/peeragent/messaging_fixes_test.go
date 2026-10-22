package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

type messagingExecutor func(context.Context, input.Delegation) (executil.Result, error)

func (e messagingExecutor) Execute(ctx context.Context, request input.Delegation) (executil.Result, error) {
	return e(ctx, request)
}

type messagingLauncher func(jobs.Job) error

func (l messagingLauncher) Launch(_ string, job jobs.Job) error { return l(job) }

func TestRetryStartupFailurePreservesUnreadReply(t *testing.T) {
	cwd := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", filepath.Join(cwd, "index"))
	t.Setenv("HERDR_PANE_ID", "")
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", RetryCapacity: 1, CapacityBackoff: time.Nanosecond}, "task")
	if err != nil {
		t.Fatal(err)
	}
	prior := applicationService
	t.Cleanup(func() { applicationService = prior })
	calls := 0
	applicationService = app.NewService(app.Options{Executor: messagingExecutor(func(ctx context.Context, request input.Delegation) (executil.Result, error) {
		calls++
		if calls == 1 {
			if err := app.NewService(app.Options{}).Tell(ctx, app.JobRequest{CWD: cwd, JobID: job.ID}, "launcher", "reply to preserve on startup failure"); err != nil {
				t.Fatal(err)
			}
			return executil.Result{TurnStarted: true, CapacityExhausted: true, ExitCode: 1, AgentSession: "retry-session"}, nil
		}
		if !strings.Contains(request.TaskText, "reply to preserve on startup failure") {
			t.Fatal("retry missing reply")
		}
		return executil.Result{ExitCode: 127}, errors.New("synthetic missing executable")
	})})
	if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: job.ID}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls %d", calls)
	}
	mailbox, err := store.Mailbox(job.ID)
	if err != nil || mailbox.Read != 0 {
		t.Fatalf("startup consumed reply %#v %v", mailbox, err)
	}
	resumed := false
	service := app.NewService(app.Options{Executable: func() (string, error) { return "fake-worker", nil }, Launcher: messagingLauncher(func(next jobs.Job) error {
		resumed = true
		carry, err := app.PrepareTurnInbox(context.Background(), next)
		if err != nil {
			return err
		}
		if strings.Count(carry.Text, "reply to preserve on startup failure") != 1 {
			t.Fatalf("recovery lost/duplicated reply: %q", carry.Text)
		}
		return carry.Acknowledge(executil.Result{TurnStarted: true}, nil)
	})})
	if _, err := service.Launch(context.Background(), input.Delegation{CWD: cwd, Agent: "codex", Resume: "retry-session", TaskText: "recover"}); err != nil {
		t.Fatal(err)
	}
	mailbox, err = store.Mailbox(job.ID)
	if err != nil || mailbox.Read != 1 || !resumed {
		t.Fatalf("recovery %#v %v", mailbox, err)
	}
}

func TestCapacityRepliesReachSuccessfulTurn(t *testing.T) {
	cwd := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", filepath.Join(cwd, "index"))
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", RetryCapacity: 2, CapacityBackoff: time.Nanosecond}, "task")
	if err != nil {
		t.Fatal(err)
	}
	prior := applicationService
	t.Cleanup(func() { applicationService = prior })
	calls := 0
	applicationService = app.NewService(app.Options{Executor: messagingExecutor(func(ctx context.Context, request input.Delegation) (executil.Result, error) {
		calls++
		if calls == 1 {
			if err := app.NewService(app.Options{}).Tell(ctx, app.JobRequest{CWD: cwd, JobID: job.ID}, "launcher", "reply must reach accepted turn"); err != nil {
				t.Fatal(err)
			}
		} else {
			if strings.Count(request.TaskText, "reply must reach accepted turn") != 1 {
				t.Fatalf("turn %d lost/duplicated reply: %q", calls, request.TaskText)
			}
			mailbox, err := store.Mailbox(job.ID)
			if err != nil || mailbox.Read != 0 {
				t.Fatalf("premature acknowledgement on turn %d: %#v %v", calls, mailbox, err)
			}
		}
		if calls <= 2 {
			return executil.Result{TurnStarted: true, CapacityExhausted: true, ExitCode: 1, AgentSession: "retry-session"}, nil
		}
		return executil.Result{TurnStarted: true, Stdout: "accepted", AgentSession: "retry-session"}, nil
	})})
	if err := runAsyncJob(input.Request{CWD: cwd, JobRunID: job.ID}); err != nil {
		t.Fatal(err)
	}
	mailbox, err := store.Mailbox(job.ID)
	if err != nil || mailbox.Read != 1 || calls != 3 {
		t.Fatalf("successful turn calls=%d mailbox=%#v err=%v", calls, mailbox, err)
	}
}

func TestCommandsSnapshotEachMailboxOnce(t *testing.T) {
	cwd, _ := fakeMessagingEnvironment(t)
	launched := cliResult(t, cwd, "--cwd", cwd, "task")
	job := loadMessageJob(t, cwd, launched.Metadata.JobID)
	t.Setenv("PEERAGENT_JOB_ID", job.ID)
	t.Setenv("PEERAGENT_JOB_DIR", filepath.Dir(job.LogPath))
	for _, args := range [][]string{{"inbox"}, {"say", "question"}, {"--status", job.ID}, {"--result", job.ID}, {"--wait", job.ID}, {"--status", job.ID, job.ID}, {"--help"}, {"--unknown-option"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			store := jobs.NewStore(cwd)
			if err := store.WithJobLock(job.ID, func() error {
				return store.AppendMessageLocked(job.ID, "inbox", jobs.Message{Text: "unread reply", From: "launcher"})
			}); err != nil {
				t.Fatal(err)
			}
			reads := map[string]int{}
			ctx := jobs.WithMailboxCommand(context.Background(), func(id string, _ jobs.Mailbox) { reads[id]++ })
			err := captureStdout(func() error { return runCommand(ctx, args) })
			if err != nil && args[0] != "--unknown-option" && !(args[0] == "say" && runtime.GOOS == "windows") {
				t.Fatal(err)
			}
			if reads[job.ID] != 1 || len(reads) != 1 {
				t.Fatalf("mailbox snapshots %#v", reads)
			}
		})
	}
	// Wait-any may select a different terminal job before visiting this one.
	// Its own snapshot and the current-job notice must each be read once.
	store := jobs.NewStore(cwd)
	other, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex"}, "other")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.FinishJob(store, other, result.Result{Status: result.StatusSuccess, Summary: "other done"}); err != nil {
		t.Fatal(err)
	}
	reads := map[string]int{}
	ctx := jobs.WithMailboxCommand(context.Background(), func(id string, _ jobs.Mailbox) { reads[id]++ })
	if err := captureStdout(func() error { return runCommand(ctx, []string{"--wait-any", other.ID, job.ID, "--cwd", cwd}) }); err != nil {
		t.Fatal(err)
	}
	if reads[other.ID] != 1 || reads[job.ID] != 1 || len(reads) != 2 {
		t.Fatalf("wait-any snapshots %#v", reads)
	}
	// A caller using a different command snapshot sees the newly sent entry.
	m, err := jobs.NewStore(cwd).Mailbox(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && (len(m.Outbox) != 1 || m.Outbox[0].Status != "sent") {
		t.Fatalf("outbox journal %#v", m.Outbox)
	}
	if _, err := os.Stat(job.LogPath); err != nil {
		t.Fatal(err)
	}
}
