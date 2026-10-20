//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	task := "Reply with \"OK\" & | ( ) ^ % !\nTake no other action."
	for _, resume := range []string{"", "fixture-thread"} {
		t.Run("resume="+resume, func(t *testing.T) {
			capture := filepath.Join(t.TempDir(), "args.json")
			args := []string{"--agent", "codex", "--cwd", dir}
			if resume != "" {
				args = append(args, "--resume", resume)
			}
			args = append(args, task)
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
			var forwarded []string
			if err := json.Unmarshal(encoded, &forwarded); err != nil {
				t.Fatal(err)
			}
			want := prompt.BuildForAgent("Codex", task)
			if len(forwarded) == 0 || forwarded[len(forwarded)-1] != want {
				t.Fatalf("forwarded prompt = %q, want %q", forwarded, want)
			}
			if resume != "" && (forwarded[len(forwarded)-2] != resume || got.Metadata.AgentSession != resume) {
				t.Fatalf("resume ID missing from %q", forwarded)
			}
		})
	}
}

func TestWindowsCLIArgHelper(t *testing.T) {
	capture := os.Getenv("PEERAGENT_CLI_CAPTURE")
	if capture == "" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			encoded, err := json.Marshal(os.Args[i+1:])
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
