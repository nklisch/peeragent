//go:build windows

package jobs

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestStartDetachedRebuildsOnlyForDeniedBreakaway(t *testing.T) {
	first := exec.Command(os.Args[0])
	ApplyDetachAttrs(first)
	var retry *exec.Cmd
	build := func() *exec.Cmd {
		retry = exec.Command(os.Args[0])
		ApplyDetachAttrs(retry)
		return retry
	}
	starts := 0
	got, err := startDetached(first, build, func(cmd *exec.Cmd) error {
		starts++
		if starts == 1 {
			return syscall.ERROR_ACCESS_DENIED
		}
		if cmd == first || cmd.SysProcAttr.CreationFlags&createBreakawayFromJob != 0 {
			t.Fatal("retry reused the failed command or retained breakaway")
		}
		return nil
	}, func() bool { return true })
	if err != nil || got != retry || starts != 2 {
		t.Fatalf("fallback = %p, %v, starts %d", got, err, starts)
	}
	starts = 0
	got, err = startDetached(first, build, func(*exec.Cmd) error {
		starts++
		return syscall.ERROR_ACCESS_DENIED
	}, func() bool { return false })
	if got != nil || !errors.Is(err, syscall.ERROR_ACCESS_DENIED) || starts != 1 {
		t.Fatalf("outside-job fallback = %p, %v, starts %d", got, err, starts)
	}
}

func TestWindowsJobHelperProcess(t *testing.T) {
	if os.Getenv("PEERAGENT_JOBS_TEST_CHILD") == "1" {
		time.Sleep(30 * time.Second)
	}
}

func TestApplyDetachAttrsRequestsBreakaway(t *testing.T) {
	cmd := exec.Command(os.Args[0])
	ApplyDetachAttrs(cmd)
	if cmd.SysProcAttr.CreationFlags&(createNoWindow|createBreakawayFromJob) != createNoWindow|createBreakawayFromJob {
		t.Fatalf("creation flags = %#x", cmd.SysProcAttr.CreationFlags)
	}
}

func TestSignalProcessTreeWaitsForAttachment(t *testing.T) {
	jobID, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestWindowsJobHelperProcess$")
	cmd.Env = append(os.Environ(), "PEERAGENT_JOBS_TEST_CHILD=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	pid := cmd.Process.Pid
	identity, err := ProcessIdentity(pid)
	if err != nil {
		t.Fatal(err)
	}
	worker := PIDRecord{PID: pid, Identity: identity}

	// A missing job must not cause cancellation to kill a possibly reused PID.
	if err := SignalProcessTree(jobID, worker, TerminateSignal()); err == nil {
		t.Fatal("cancellation accepted a live PID with no named job")
	}
	if !processExists(pid) || !ProcessTreeExists(jobID, worker) {
		t.Fatal("a live process was stopped or reported stopped before attachment")
	}
	stale := PIDRecord{PID: pid, Identity: identity + 1}
	if err := SignalProcessTree(jobID, stale, TerminateSignal()); err != nil || ProcessTreeExists(jobID, stale) || !processExists(pid) {
		t.Fatalf("stale worker identity affected unrelated process: %v", err)
	}

	name, err := jobName(jobID)
	if err != nil {
		t.Fatal(err)
	}
	job, _, callErr := createJobObject.Call(0, uintptr(unsafe.Pointer(name)))
	if job == 0 {
		t.Fatalf("create empty job: %v", callErr)
	}
	defer syscall.CloseHandle(syscall.Handle(job))
	if err := SignalProcessTree(jobID, worker, TerminateSignal()); err == nil {
		t.Fatal("cancellation accepted a live worker while its named job was empty")
	}
	if !processExists(pid) || !ProcessTreeExists(jobID, worker) {
		t.Fatal("an empty job hid or stopped its live worker")
	}

	process, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE|syscall.SYNCHRONIZE|0x0100, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(process)
	ok, _, callErr := assignProcessToJobObject.Call(job, uintptr(process))
	if ok == 0 {
		t.Fatalf("assign worker to job: %v", callErr)
	}
	if err := SignalProcessTree(jobID, worker, TerminateSignal()); err != nil {
		t.Fatal(err)
	}
	if state, err := syscall.WaitForSingleObject(process, 2000); err != nil || state != syscall.WAIT_OBJECT_0 {
		t.Fatalf("worker remained alive after job termination: state=%d err=%v", state, err)
	}
}
