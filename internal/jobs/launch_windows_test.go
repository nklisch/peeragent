//go:build windows

package jobs

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestWindowsJobHelperProcess(t *testing.T) {
	if os.Getenv("PEERAGENT_JOBS_TEST_CHILD") == "1" {
		time.Sleep(30 * time.Second)
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

	// A missing job must not cause cancellation to kill a possibly reused PID.
	if err := SignalProcessTree(jobID, pid, TerminateSignal()); err == nil {
		t.Fatal("cancellation accepted a live PID with no named job")
	}
	if !processExists(pid) || !ProcessTreeExists(jobID, pid) {
		t.Fatal("a live process was stopped or reported stopped before attachment")
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
	cancelDone := make(chan error, 1)
	go func() { cancelDone <- SignalProcessTree(jobID, pid, TerminateSignal()) }()
	select {
	case err := <-cancelDone:
		t.Fatalf("cancellation returned before the worker attached: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if !processExists(pid) || !ProcessTreeExists(jobID, pid) {
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
	select {
	case err := <-cancelDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not finish after worker attachment")
	}
	if state, err := syscall.WaitForSingleObject(process, 2000); err != nil || state != syscall.WAIT_OBJECT_0 {
		t.Fatalf("worker remained alive after job termination: state=%d err=%v", state, err)
	}
}
