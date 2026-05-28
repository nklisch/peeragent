//go:build windows

package jobs

import (
	"errors"
	"os"
	"os/exec"
)

func ApplyDetachAttrs(cmd *exec.Cmd) {
}

func SignalProcessGroup(pid int, sig os.Signal) error {
	return errors.New("process-group signalling not implemented on windows")
}

func ProcessGroupExists(pid int) bool {
	return false
}

func TerminateSignal() os.Signal {
	return os.Interrupt
}

func KillSignal() os.Signal {
	return os.Kill
}
