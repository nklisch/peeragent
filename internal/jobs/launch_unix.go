//go:build unix

package jobs

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func ApplyDetachAttrs(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}

func StartDetached(cmd *exec.Cmd) error { return cmd.Start() }

func AttachCurrentProcess(string) (func(), error) {
	return func() {}, nil
}

func SignalProcessTree(_ string, worker PIDRecord, sig os.Signal) error {
	return SignalProcessGroup(worker.PID, sig)
}

func ProcessTreeExists(_ string, worker PIDRecord) bool {
	return ProcessGroupExists(worker.PID)
}

func SignalProcessGroup(pid int, sig os.Signal) error {
	if pid <= 1 {
		return fmt.Errorf("refusing to signal unsafe process group %d", pid)
	}
	signal, ok := sig.(syscall.Signal)
	if !ok {
		return fmt.Errorf("unsupported process-group signal %T", sig)
	}
	return syscall.Kill(-pid, signal)
}

func ProcessGroupExists(pid int) bool {
	if pid <= 1 {
		return false
	}
	err := syscall.Kill(-pid, 0)
	return err == nil || err == syscall.EPERM
}

func TerminateSignal() os.Signal {
	return syscall.SIGTERM
}

func KillSignal() os.Signal {
	return syscall.SIGKILL
}
