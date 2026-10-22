//go:build windows

package executil

import (
	"fmt"
	"os/exec"
	"syscall"
)

// A Windows Job Object owns the launched tree, including a native agent behind
// an npm shim. Forced completion ends only processes assigned to this job.
var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var createJobObject = kernel32.NewProc("CreateJobObjectW")
var assignJobProcess = kernel32.NewProc("AssignProcessToJobObject")
var terminateJobObject = kernel32.NewProc("TerminateJobObject")

func prepareOwnedCommand(cmd *exec.Cmd) {}

func ownProcess(cmd *exec.Cmd) (func() error, func(), error) {
	h, _, err := createJobObject.Call(0, 0)
	if h == 0 {
		return nil, nil, fmt.Errorf("CreateJobObject: %w", err)
	}
	closeJob := func() { _ = syscall.CloseHandle(syscall.Handle(h)) }
	process, err := syscall.OpenProcess(0x0100|0x0001|0x00100000, false, uint32(cmd.Process.Pid)) // SET_QUOTA | TERMINATE | SYNCHRONIZE
	if err != nil {
		closeJob()
		return nil, nil, err
	}
	ok, _, err := assignJobProcess.Call(h, uintptr(process))
	if ok == 0 {
		// A short command may have exited between Start and assignment. It
		// needs no cleanup; let Wait collect its real result.
		state, _ := syscall.WaitForSingleObject(process, 0)
		_ = syscall.CloseHandle(process)
		closeJob()
		if state == syscall.WAIT_OBJECT_0 {
			return func() error { return nil }, func() {}, nil
		}
		return nil, nil, fmt.Errorf("AssignProcessToJobObject: %w", err)
	}
	_ = syscall.CloseHandle(process)
	return func() error {
		ok, _, err := terminateJobObject.Call(h, 0)
		if ok == 0 {
			return fmt.Errorf("TerminateJobObject: %w", err)
		}
		return nil
	}, closeJob, nil
}
