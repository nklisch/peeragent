//go:build !linux && !darwin && !windows

package executil

import "os/exec"

func prepareOwnedCommand(cmd *exec.Cmd) {}
func ownProcess(cmd *exec.Cmd) (*ownedCommand, error) {
	// Other platforms own only the direct os.Process handle, never a numeric group.
	var waitErr error
	return &ownedCommand{stop: cmd.Process.Kill, waitExit: func() error { waitErr = cmd.Wait(); return nil }, reap: func() error { return waitErr }, release: func() {}}, nil
}
