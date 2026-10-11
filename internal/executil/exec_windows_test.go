//go:build windows

package executil

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOSRunnerBatchShim(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PEERAGENT_ARG_HELPER", "1")
	t.Setenv("PEERAGENT_EXPAND_TEST", "must not expand")
	args := []string{"", "task with spaces", `say "OK" & | ( ) ^ % !`, `%PEERAGENT_EXPAND_TEST% !PEERAGENT_EXPAND_TEST!`, `" & echo injected>injected.txt & "`, `C:\path with spaces\`, `slash\"quote`, "unicode: café 日本語", "tab\there", `%% %1 %* %PATH% ^"!"^`, `"""`, `backslashes\\"and\\`, `< > , ; = [ ] * ?`}
	for _, folder := range []string{"bin", "Program Files (x86)"} {
		for _, ext := range []string{".cmd", ".bat", ".CMD"} {
			t.Run(folder+ext, func(t *testing.T) {
				dir := filepath.Join(t.TempDir(), folder)
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				shim := filepath.Join(dir, "codex"+ext)
				// npm shims forward %* inside a compound command, after endlocal.
				body := "@echo off\r\nsetlocal\r\nset \"_prog=" + self + "\"\r\nendlocal & goto #_undefined_# 2>NUL || title %COMSPEC% & \"%_prog%\" -test.run=^TestBatchArgHelper$ -- %*\r\n"
				if err := os.WriteFile(shim, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
				path, err := exec.LookPath("codex")
				if err != nil || !strings.EqualFold(path, shim) {
					t.Fatalf("LookPath = %q, %v; want %q", path, err, shim)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				got, err := (OSRunner{}).Run(ctx, path, args, dir)
				if err != nil || got.ExitCode != 0 {
					t.Fatalf("Run = %+v, %v", got, err)
				}
				var forwarded []string
				if err := json.Unmarshal([]byte(got.Stdout), &forwarded); err != nil || !reflect.DeepEqual(forwarded, args) {
					t.Fatalf("argv = %q, want %q (%v); stderr = %q", forwarded, args, err, got.Stderr)
				}
				if _, err := os.Stat(filepath.Join(dir, "injected.txt")); !os.IsNotExist(err) {
					t.Fatalf("shell injection marker: %v", err)
				}
				for _, arg := range []string{"before\rafter", "before\nafter"} {
					got, err := (OSRunner{}).Run(ctx, path, []string{arg}, dir)
					if err == nil || got.ExitCode != 1 || got.Stdout != "" {
						t.Fatalf("command boundary argument launched: %+v, %v", got, err)
					}
				}
			})
		}
	}
}
