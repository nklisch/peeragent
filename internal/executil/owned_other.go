//go:build !unix && !windows

package executil

import "os/exec"

func prepareOwnedCommand(cmd *exec.Cmd) {}
func ownProcess(cmd *exec.Cmd) (func() error, func(), error) {
	return cmd.Process.Kill, func() {}, nil
}
