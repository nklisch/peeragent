package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestBlockingWorkerExitWithoutResult(t *testing.T) {
	for _, code := range []int{0, 23} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
			cwd := t.TempDir()
			launcher := ProcessLauncher{command: func(_ string, _ ...string) *exec.Cmd {
				cmd := exec.Command(os.Args[0], "-test.run=^TestWorkerExitHelper$")
				cmd.Env = append(os.Environ(), "PEERAGENT_WORKER_EXIT_HELPER="+strconv.Itoa(code))
				return cmd
			}, releaseProcess: func(*os.Process) error { t.Error("blocking worker released its native handle"); return nil }}
			service := NewService(Options{BlockingLauncher: launcher, Launcher: &fakeLauncher{}})
			delegation := input.Delegation{CWD: cwd, Agent: "codex", Resume: "dead-session", TaskText: "work"}
			launched, worker, err := service.LaunchBlocking(context.Background(), delegation)
			if err != nil {
				t.Fatal(err)
			}
			// Retained handles are collected before the missing-result failure is stored.
			gotCode, waitErr := worker.Wait()
			if gotCode != code || waitErr != nil {
				t.Fatalf("worker exit: %d %v", gotCode, waitErr)
			}
			req := JobRequest{CWD: cwd, JobID: launched.Metadata.JobID}
			terminal, err := service.WorkerExited(context.Background(), req, gotCode, waitErr)
			if err != nil || terminal.Status != result.StatusFailed || terminal.Metadata.ExitCode != code || terminal.Metadata.AgentSession != "dead-session" || terminal.Metadata.LogPath == "" || !strings.Contains(terminal.Summary, "before writing a terminal result") {
				t.Fatalf("missing-result outcome: %#v %v", terminal, err)
			}
			log, err := os.ReadFile(terminal.Metadata.LogPath)
			if err != nil || !strings.Contains(string(log), "worker diagnostic") {
				t.Fatalf("worker log: %q %v", log, err)
			}
			job, err := jobs.NewStore(cwd).Load(req.JobID)
			if err != nil || job.Status != jobs.StatusFailed {
				t.Fatalf("persisted status: %#v %v", job, err)
			}
			if _, err := jobs.NewStore(cwd).ReadPID(req.JobID); !os.IsNotExist(err) {
				t.Fatalf("exited worker retained pid: %v", err)
			}
			resumed, err := service.Launch(context.Background(), delegation)
			if err != nil || resumed.Status != result.StatusRunning {
				t.Fatalf("exited worker blocked resume: %#v %v", resumed, err)
			}
		})
	}
}

func TestWorkerExitHelper(t *testing.T) {
	if raw := os.Getenv("PEERAGENT_WORKER_EXIT_HELPER"); raw != "" {
		code, err := strconv.Atoi(raw)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(os.Stderr, "worker diagnostic")
		os.Exit(code)
	}
}

func TestWorkerExitPreservesTerminalWinner(t *testing.T) {
	for _, status := range []result.Status{result.StatusSuccess, result.StatusFailed, result.StatusCapacity, result.StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "same"}, "work")
			if err != nil {
				t.Fatal(err)
			}
			want := result.Result{Status: status, Summary: "original terminal result", Metadata: result.Metadata{JobID: job.ID, CWD: cwd, AgentSession: "same", ExitCode: 7}}
			if status == result.StatusCapacity {
				want.Summary = result.CapacitySummary("same")
			}
			if err := FinishJob(store, job, want); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(job.ResultPath)
			if err != nil {
				t.Fatal(err)
			}
			got, err := NewService(Options{}).WorkerExited(context.Background(), JobRequest{CWD: cwd, JobID: job.ID}, 23, nil)
			if err != nil || got.Status != want.Status || got.Metadata.ExitCode != 7 || got.Summary != want.Summary {
				t.Fatalf("terminal result overwritten: %#v %v", got, err)
			}
			after, err := os.ReadFile(job.ResultPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("worker exit rewrote an existing terminal result")
			}
		})
	}
}
