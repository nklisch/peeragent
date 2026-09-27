//go:build windows

package executil

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func commandContext(ctx context.Context, name string, args []string) (*exec.Cmd, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".cmd" && ext != ".bat" {
		return exec.CommandContext(ctx, name, args...), nil
	}
	parts := make([]string, 0, len(args)+1)
	for _, arg := range append([]string{name}, args...) {
		if strings.ContainsAny(arg, "\r\n\"%^!") {
			return nil, fmt.Errorf("unsafe argument for Windows batch launcher %s", name)
		}
		// Batch launchers such as npm's forward %* to a native executable.
		// Double trailing backslashes for that executable's argv parser so
		// the closing quote stays a delimiter rather than becoming literal.
		parts = append(parts, `"`+arg+strings.Repeat(`\`, trailingBackslashes(arg))+`"`)
	}
	// cmd /s removes the outer pair of quotes, leaving each path and option
	// quoted for the batch launcher. The prompt itself is sent through stdin.
	commandLine := `"` + strings.Join(parts, " ") + `"`
	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /v:off /s /c ` + commandLine, HideWindow: true}
	return cmd, nil
}

func trailingBackslashes(value string) int {
	n := 0
	for i := len(value) - 1; i >= 0 && value[i] == '\\'; i-- {
		n++
	}
	return n
}
