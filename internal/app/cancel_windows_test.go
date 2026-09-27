//go:build windows

package app

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestWindowsUnrelatedProcess(t *testing.T) {
	if os.Getenv("PEERAGENT_UNRELATED_TEST_PROCESS") == "1" {
		time.Sleep(30 * time.Second)
	}
}

func TestCancelIgnoresReusedWorkerPID(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestWindowsUnrelatedProcess$")
	cmd.Env = append(os.Environ(), "PEERAGENT_UNRELATED_TEST_PROCESS=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	identity, err := jobs.ProcessIdentity(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePIDRecord(job.ID, jobs.PIDRecord{PID: cmd.Process.Pid, Identity: identity + 1}); err != nil {
		t.Fatal(err)
	}
	got, err := NewService(Options{}).CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil || got.Status != result.StatusCancelled {
		t.Fatalf("cancel with reused PID = %#v, %v", got, err)
	}
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(cmd.Process.Pid))
	if err != nil {
		t.Fatalf("unrelated process was killed: %v", err)
	}
	defer syscall.CloseHandle(h)
	if state, err := syscall.WaitForSingleObject(h, 0); err != nil || state != syscall.WAIT_TIMEOUT {
		t.Fatalf("unrelated process exited: state=%d err=%v", state, err)
	}
}
