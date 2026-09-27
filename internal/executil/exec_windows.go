//go:build windows

package executil

import (
	"context"
	"fmt"
	"os"
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
	for index, arg := range append([]string{name}, args...) {
		if unsafeAt := strings.IndexAny(arg, "\r\n\"%^!"); unsafeAt >= 0 {
			return nil, fmt.Errorf("Windows batch launcher %s cannot pass argument %d: character %q is unsafe for cmd.exe", name, index, arg[unsafeAt])
		}
		// Batch launchers such as npm's forward %* to a native executable.
		// Double trailing backslashes for that executable's argv parser so
		// the closing quote stays a delimiter rather than becoming literal.
		parts = append(parts, `"`+arg+strings.Repeat(`\`, trailingBackslashes(arg))+`"`)
	}
	// cmd /s removes the outer pair of quotes, leaving each path and option
	// quoted for the batch launcher. The prompt itself is sent through stdin.
	commandLine := `"` + strings.Join(parts, " ") + `"`
	interpreter := os.Getenv("ComSpec")
	if !filepath.IsAbs(interpreter) {
		interpreter = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}
	if !filepath.IsAbs(interpreter) {
		return nil, fmt.Errorf("cannot locate System32 cmd.exe for Windows batch launcher %s", name)
	}
	cmd := exec.CommandContext(ctx, interpreter)
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
