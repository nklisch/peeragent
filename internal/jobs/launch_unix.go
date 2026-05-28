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

func SignalProcessGroup(pid int, sig os.Signal) error {
	signal, ok := sig.(syscall.Signal)
	if !ok {
		return fmt.Errorf("unsupported process-group signal %T", sig)
	}
	return syscall.Kill(-pid, signal)
}

func ProcessGroupExists(pid int) bool {
	err := syscall.Kill(-pid, 0)
	return err == nil || err == syscall.EPERM
}

func TerminateSignal() os.Signal {
	return syscall.SIGTERM
}

func KillSignal() os.Signal {
	return syscall.SIGKILL
}
