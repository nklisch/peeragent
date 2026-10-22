//go:build linux || darwin

package executil

import (
	"errors"
	"os/exec"
	"sync"
	"syscall"
)

func prepareOwnedCommand(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// The unreaped leader reserves its PID and group number, even after exit. The
// same lock revokes group signalling before Wait can reap it and allow reuse.
type ownedGroup struct {
	mu      sync.Mutex
	pid     int
	reaping bool
	signal  func(int, syscall.Signal) error
}

func (g *ownedGroup) stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.reaping {
		return nil
	}
	err := g.signal(-g.pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func (g *ownedGroup) reap(wait func() error) error {
	g.mu.Lock()
	g.reaping = true
	g.mu.Unlock()
	return wait()
}

func ownProcess(cmd *exec.Cmd) (*ownedCommand, error) {
	group := &ownedGroup{pid: cmd.Process.Pid, signal: syscall.Kill}
	return &ownedCommand{
		stop:     group.stop,
		waitExit: func() error { return waitWithoutReaping(cmd.Process.Pid) },
		reap:     func() error { return group.reap(cmd.Wait) },
		release:  func() {},
	}, nil
}
