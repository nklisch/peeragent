//go:build unix

package executil

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func prepareOwnedCommand(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error { return killOwnedGroup(cmd) }
}

func killOwnedGroup(cmd *exec.Cmd) error {
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}

func ownProcess(cmd *exec.Cmd) (func() error, func(), error) {
	return func() error {
		err := killOwnedGroup(cmd)
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}, func() {}, nil
}
