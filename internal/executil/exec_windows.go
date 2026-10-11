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
	ext := filepath.Ext(name)
	if !strings.EqualFold(ext, ".cmd") && !strings.EqualFold(ext, ".bat") {
		return exec.CommandContext(ctx, name, args...), nil
	}
	if strings.ContainsAny(name, "\x00\r\n\"%") {
		return nil, fmt.Errorf("invalid Windows batch launcher path")
	}
	parts := []string{escapeBatchMeta(name)}
	for _, arg := range args {
		quoted, err := quoteBatchArg(arg)
		if err != nil {
			return nil, err
		}
		parts = append(parts, quoted)
	}
	// /s /c strips this outer quote pair. The caret-escaped launcher path
	// survives that pass; arguments also survive the shim's %* forwarding.
	// Disable AutoRun and delayed expansion so neither alters the invocation.
	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:    `cmd.exe /d /e:on /v:off /s /c "` + strings.Join(parts, " ") + `"`,
		HideWindow: true,
	}
	return cmd, nil
}
