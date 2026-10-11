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
	ext := filepath.Ext(name)
	if !strings.EqualFold(ext, ".cmd") && !strings.EqualFold(ext, ".bat") {
		return exec.CommandContext(ctx, name, args...), nil
	}
	if strings.ContainsAny(name, "\x00\r\n\"%") {
		return nil, fmt.Errorf("invalid Windows batch launcher path")
	}
	launcher := strings.ReplaceAll(escapeBatchMeta(name), "^!", "^^^!")
	parts := []string{launcher}
	env := os.Environ()
	for i, arg := range args {
		quoted, err := quoteBatchArg(arg)
		if err != nil {
			return nil, err
		}
		variable := fmt.Sprintf("_PEERAGENT_BATCH_ARG_%d", i)
		env = append(env, variable+"="+quoted)
		// Carets defer expansion while invoking the shim. When its native
		// command expands %*, delayed expansion inserts the quoted value
		// after shell parsing, without interpreting any of the task text.
		parts = append(parts, "^^!"+variable+"^^!")
	}
	// /s /c strips the outer quote pair; escaped spaces keep the path intact.
	// AutoRun is disabled. npm shims preserve this delayed expansion setting
	// across setlocal/endlocal before forwarding %* to the native executable.
	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:    `cmd.exe /d /e:on /v:on /s /c "` + strings.Join(parts, " ") + `"`,
		HideWindow: true,
	}
	return cmd, nil
}
