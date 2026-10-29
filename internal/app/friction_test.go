package app

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestDeadWorkerBecomesInterruptedAcrossReadersAndResume(t *testing.T) {
	for _, read := range []string{"status", "result", "poll", "multi", "resume"} {
		t.Run(read, func(t *testing.T) {
			t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Model: "gpt-6.1-sol", Effort: "high"}, "work")
			if err != nil {
				t.Fatal(err)
			}
			if err := store.WritePID(job.ID, os.Getpid()); err != nil {
				t.Fatal(err)
			}
			if err := RecordJobProgress(store, job.ID, executil.Progress{AgentSession: "recover-me", Message: "last message before reboot", Event: "item.completed"}); err != nil {
				t.Fatal(err)
			}
			job, _ = store.Load(job.ID)
			job.Worker.StartTime += "-reused"
			if err := store.Save(job); err != nil {
				t.Fatal(err)
			}
			observedAfter := time.Now().UTC()
			service := NewService(Options{})
			req := JobRequest{CWD: cwd, JobID: job.ID}
			ctx := context.Background()
			var res result.Result
			switch read {
			case "status":
				res, err = service.JobStatus(ctx, req)
			case "result":
				res, err = service.JobResult(ctx, req)
			case "poll":
				res, err = service.PollJobResult(ctx, req)
			case "multi":
				rows, e := service.JobStatuses(ctx, []JobRequest{req})
				err = e
				if err == nil && (len(rows) != 1 || rows[0].Status != result.StatusInterrupted || rows[0].AgentSession != "recover-me" || rows[0].LastMessage != "last message before reboot") {
					t.Fatalf("rows %#v", rows)
				}
			case "resume":
				matches, e := previousSessionJobs("codex", "recover-me", cwd)
				err = e
				if err == nil {
					_, live, e := jobs.LiveSessionJob(matches)
					err = e
					if live {
						t.Fatal("dead worker blocks resume")
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if read != "multi" && read != "resume" && (res.Status != result.StatusInterrupted || res.Metadata.AgentSession != "recover-me" || res.FinalMessage != nil) {
				t.Fatalf("%#v", res)
			}
			loaded, _ := store.Load(job.ID)
			if loaded.Status != jobs.StatusInterrupted {
				t.Fatalf("not persisted: %#v", loaded)
			}
			terminal, err := service.JobResult(ctx, req)
			if err != nil || !strings.Contains(terminal.Details, "last message before reboot") || terminal.Metadata.StopReason != "worker_interrupted" || terminal.Metadata.FinishedAt == nil || terminal.Metadata.FinishedAt.Before(observedAfter) {
				t.Fatalf("lost recovery context: %#v %v", terminal, err)
			}
		})
	}
}

func TestLiveWorkerAndTerminalWinnerArePreserved(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	req := JobRequest{CWD: cwd, JobID: job.ID}
	got, err := service.JobResult(context.Background(), req)
	if err != nil || got.Status != result.StatusRunning {
		t.Fatalf("live job: %#v %v", got, err)
	}
	job, _ = store.Load(job.ID)
	job.Worker.StartTime += "-old"
	store.Save(job)
	if err := WriteJobResult(job.ResultPath, result.Result{Status: result.StatusSuccess, Summary: "natural winner", Metadata: result.Metadata{CWD: cwd, JobID: job.ID}}); err != nil {
		t.Fatal(err)
	}
	got, err = service.JobResult(context.Background(), req)
	if err != nil || got.Status != result.StatusSuccess || got.Summary != "natural winner" {
		t.Fatalf("winner: %#v %v", got, err)
	}
}

func TestCancelNeverSignalsMismatchedOrLegacyPID(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{true: "legacy", false: "reused"}[legacy], func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, testJobSpec(), "work")
			if err != nil {
				t.Fatal(err)
			}
			if err := store.WritePID(job.ID, os.Getpid()); err != nil {
				t.Fatal(err)
			}
			job, _ = store.Load(job.ID)
			if legacy {
				job.Worker = nil
			} else {
				job.Worker.StartTime += "-reused"
			}
			if err := store.Save(job); err != nil {
				t.Fatal(err)
			}
			controller := &fakeProcessController{}
			service := NewService(Options{ProcessController: controller})
			res, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
			if err != nil || res.Status != result.StatusCancelled || controller.calls() != 0 {
				t.Fatalf("unsafe cleanup: %#v %v %d", res, err, controller.calls())
			}
		})
	}
}

func TestWrongDirectoryUsesIndexOrReportsNotFound(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cwd, wrong := t.TempDir(), t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "work")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	req := JobRequest{CWD: wrong, JobID: job.ID}
	got, err := service.JobStatus(context.Background(), req)
	if err != nil || got.Status != result.StatusNotFound || got.Metadata.ExitCode != 4 || !strings.Contains(got.Summary, wrong) {
		t.Fatalf("lookup %#v %v", got, err)
	}
	if _, err := jobs.WriteIndexedJob(jobs.JobIndexEntry(job)); err != nil {
		t.Fatal(err)
	}
	for _, read := range []func(context.Context, JobRequest) (result.Result, error){service.JobStatus, service.JobResult} {
		got, err = read(context.Background(), req)
		if err != nil || got.Status != result.StatusRunning || got.Metadata.CWD != cwd || !strings.Contains(got.Metadata.LookupWarning, wrong) || !strings.Contains(got.Metadata.LookupWarning, cwd) {
			t.Fatalf("indexed lookup %#v %v", got, err)
		}
	}
}

func TestSayRateRefusalNamesExactNextSend(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "work")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	times := []time.Time{now.Add(-time.Minute), now.Add(-3 * time.Minute), now.Add(-2 * time.Minute)}
	err = store.WithJobLock(job.ID, func() error {
		for _, at := range times {
			if err := store.AppendMessageLocked(job.ID, "outbox", jobs.Message{Time: at, DeliveryAttempted: true, Status: "sent"}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewService(Options{}).Say(context.Background(), JobRequest{CWD: cwd, JobID: job.ID}, "question")
	next := times[1].Add(5 * time.Minute).Format(time.RFC3339Nano)
	if err == nil || !strings.Contains(err.Error(), "next send allowed at "+next) {
		t.Fatalf("refusal %v, want %s", err, next)
	}
}

func TestRemovedWorktreeRetainsWorkerRecovery(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "retained-session"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	job, _ = store.Load(job.ID)
	entry := jobs.JobIndexEntry(job)
	if err := os.RemoveAll(cwd); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	res, err := service.JobStatus(context.Background(), JobRequest{JobID: job.ID})
	if err != nil || res.Status != result.StatusRunning || !res.Metadata.IndexedOnly {
		t.Fatalf("live retained worker %#v %v", res, err)
	}
	entry.Worker.StartTime += "-reused"
	if _, err := jobs.WriteIndexedJob(entry); err != nil {
		t.Fatal(err)
	}
	for _, read := range []func(context.Context, JobRequest) (result.Result, error){service.JobStatus, service.JobResult} {
		res, err = read(context.Background(), JobRequest{JobID: job.ID})
		if err != nil || res.Status != result.StatusInterrupted || res.Metadata.AgentSession != "retained-session" {
			t.Fatalf("lost retained worker %#v %v", res, err)
		}
	}
	if _, err := os.Stat(cwd); !os.IsNotExist(err) {
		t.Fatalf("removed directory recreated: %v", err)
	}
}
