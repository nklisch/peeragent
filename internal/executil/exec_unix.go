//go:build unix

package executil

import (
	"context"
	"os/exec"
)

func commandContext(ctx context.Context, name string, args []string) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, name, args...), nil
}
