package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestResumeGuardRunningTerminalGoneAndOverride(t *testing.T) {
	for _, mode := range []string{"running", "terminal", "gone", "missing-pid", "override", "different-agent"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
			cwd := t.TempDir()
			store := jobs.NewStore(cwd)
			job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "shared-session"}, "work")
			if err != nil {
				t.Fatal(err)
			}
			if mode == "terminal" {
				job.Status = jobs.StatusFailed
				if err := store.Save(job); err != nil {
					t.Fatal(err)
				}
			}
			pid := os.Getpid()
			if mode == "gone" {
				// A child we start and reap provides a positively known exited process.
				pid = exitedProcess(t)
			}
			if mode != "missing-pid" {
				if err := store.WritePID(job.ID, pid); err != nil {
					t.Fatal(err)
				}
			}
			if err := jobs.IndexJob(job.ID, cwd); err != nil {
				t.Fatal(err)
			}
			launched := false
			delegation := input.Delegation{Agent: "codex", Resume: "shared-session", CWD: t.TempDir(), AllowConcurrentSession: mode == "override"}
			if mode == "different-agent" {
				delegation.Agent = "claude"
			}
			res, err := protectSession(delegation, func() (result.Result, error) {
				launched = true
				return result.Result{Status: result.StatusRunning}, nil
			})
			if mode == "running" {
				if err == nil || launched || res.Metadata.ExitCode != 2 || !strings.Contains(res.Summary, job.ID) || !strings.Contains(res.Summary, "--wait") || !strings.Contains(res.Summary, "--cancel") || !strings.Contains(res.Summary, "--allow-concurrent-session") {
					t.Fatalf("guard: %#v %v launched=%t", res, err, launched)
				}
			} else if err != nil || !launched {
				t.Fatalf("valid launch refused: %#v %v", res, err)
			}
		})
	}
}

func TestResumeGuardSerializesConcurrentLaunches(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	service := NewService(Options{Launcher: livePIDLauncher{}, Executable: func() (string, error) { return "fake", nil }})
	cwd := t.TempDir()
	var wg sync.WaitGroup
	outcomes := make(chan result.Result, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, _ := service.Launch(context.Background(), input.Delegation{CWD: cwd, Agent: "codex", Resume: "same", TaskText: "work"})
			outcomes <- res
		}()
	}
	wg.Wait()
	close(outcomes)
	running, refused := 0, 0
	for res := range outcomes {
		if res.Status == result.StatusRunning {
			running++
		} else if res.Metadata.ExitCode == 2 {
			refused++
		}
	}
	if running != 1 || refused != 1 {
		t.Fatalf("running=%d refused=%d", running, refused)
	}
}

type livePIDLauncher struct{}

func (livePIDLauncher) Launch(_ string, job jobs.Job) error {
	return jobs.NewStore(job.CWD).WritePID(job.ID, os.Getpid())
}

func exitedProcess(t *testing.T) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestExitedProcessHelper$")
	cmd.Env = append(os.Environ(), "PEERAGENT_EXIT_HELPER=1")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
}
func TestExitedProcessHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_EXIT_HELPER") == "1" {
		os.Exit(0)
	}
}

func TestResumeGuardStorageFailureStaysInfrastructureError(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "same"}, "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Root, job.ID, "job.json"), []byte("invalid json"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := protectSession(input.Delegation{CWD: cwd, Agent: "codex", Resume: "same"}, func() (result.Result, error) { t.Fatal("launched with unreadable state"); return result.Result{}, nil })
	if err == nil || res.Metadata.ExitCode != 1 || !strings.Contains(err.Error(), "load job "+job.ID) {
		t.Fatalf("storage error became usage/conflict: %#v %v", res, err)
	}
}
