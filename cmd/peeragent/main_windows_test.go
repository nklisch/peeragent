//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/prompt"
	"github.com/nklisch/peeragent/internal/result"
)

func TestWindowsCLIWithNpmShim(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "Program Files (x86)")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := "@echo off\r\nsetlocal\r\nset \"_prog=" + self + "\"\r\nendlocal & goto #_undefined_# 2>NUL || title %COMSPEC% & \"%_prog%\" -test.run=^TestWindowsCLIArgHelper$ -- %*\r\n"
	if err := os.WriteFile(filepath.Join(dir, "codex.cmd"), []byte(shim), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{1, 200, 2000} {
		task := strings.Repeat("Reply with \"OK\" & | ( ) ^ % !\r\nUnicode: café 日本語\n", size)
		for _, resume := range []string{"", "fixture-thread"} {
			for _, fullAccess := range []bool{false, true} {
				t.Run(fmt.Sprintf("size=%d/resume=%s/fullAccess=%t", size, resume, fullAccess), func(t *testing.T) {
					capture := filepath.Join(t.TempDir(), "args.json")
					args := []string{"--agent", "codex", "--cwd", dir}
					if resume != "" {
						args = append(args, "--resume", resume)
					}
					if fullAccess {
						args = append(args, "--full-access")
					}
					if size == 1 {
						args = append(args, task)
					} else {
						brief := filepath.Join(t.TempDir(), "brief.md")
						if err := os.WriteFile(brief, []byte(task), 0o600); err != nil {
							t.Fatal(err)
						}
						args = append(args, "--prompt-file", brief)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					cmd := exec.CommandContext(ctx, self, args...)
					cmd.Env = append(os.Environ(), "PEERAGENT_TEST_HELPER_MAIN=1", "PEERAGENT_CLI_CAPTURE="+capture, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
					output, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatalf("CLI failed: %v\n%s", err, output)
					}
					got := decodeResult(t, output)
					if got.Status != result.StatusSuccess || !strings.Contains(got.Details, "stdout:\nOK") {
						t.Fatalf("CLI result = %+v", got)
					}
					encoded, err := os.ReadFile(capture)
					if err != nil {
						t.Fatal(err)
					}
					var captured codexCLIInput
					if err := json.Unmarshal(encoded, &captured); err != nil {
						t.Fatal(err)
					}
					want := prompt.BuildForAgent("Codex", task)
					if captured.Stdin != want {
						t.Fatalf("stdin differs from prompt: got %d bytes, want %d", len(captured.Stdin), len(want))
					}
					forwarded := captured.Args
					if len(forwarded) == 0 || forwarded[len(forwarded)-1] != "-" {
						t.Fatalf("missing stdin marker: %q", forwarded)
					}
					if resume != "" && (forwarded[len(forwarded)-2] != resume || got.Metadata.AgentSession != resume) {
						t.Fatalf("resume ID missing from %q", forwarded)
					}
				})
			}
		}
	}
}

type codexCLIInput struct {
	Args  []string
	Stdin string
}

func TestWindowsCLIArgHelper(t *testing.T) {
	capture := os.Getenv("PEERAGENT_CLI_CAPTURE")
	if capture == "" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			stdin, err := io.ReadAll(os.Stdin)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(codexCLIInput{Args: os.Args[i+1:], Stdin: string(stdin)})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(capture, encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"OK"}}`)
			os.Exit(0)
		}
	}
	t.Fatal("missing argument marker")
}
