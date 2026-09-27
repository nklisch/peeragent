//go:build windows

package executil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatchRunnerQuotesPathAndArguments(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bin with spaces")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".cmd", ".bat"} {
		path := filepath.Join(dir, "echo"+ext)
		if err := os.WriteFile(path, []byte("@echo off\r\necho %~1\r\nif not \"%~2\"==\"\" echo %~2\r\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := (OSRunner{}).Run(context.Background(), path, []string{"arg with spaces"}, dir, "")
		if err != nil || got.ExitCode != 0 || strings.TrimSpace(got.Stdout) != "arg with spaces" {
			t.Fatalf("%s launch = %#v, %v", ext, got, err)
		}
		if _, err := (OSRunner{}).Run(context.Background(), path, []string{`bad" & echo injected`}, dir, ""); err == nil {
			t.Fatalf("%s launcher accepted an argument that can break quoting", ext)
		}
	}
}

func TestBatchRunnerPreservesForwardedArguments(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "bin with spaces")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".cmd", ".bat"} {
		path := filepath.Join(dir, "forward"+ext)
		content := "@echo off\r\n\"" + self + "\" -test.run=TestBatchArgHelper -- %*\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		args := []string{`C:\`, `C:\path with spaces\`, "tail"}
		got, err := (OSRunner{}).Run(context.Background(), path, args, dir, "")
		if err != nil || got.ExitCode != 0 {
			t.Fatalf("%s forwarded launch = %#v, %v", ext, got, err)
		}
		var forwarded []string
		if err := json.Unmarshal([]byte(got.Stdout), &forwarded); err != nil || !equalStringSlices(forwarded, args) {
			t.Fatalf("%s forwarded args = %q, want %q (%v)", ext, forwarded, args, err)
		}
	}
}

func TestBatchArgHelper(t *testing.T) {
	marker := -1
	for i, arg := range os.Args {
		if arg == "--" {
			marker = i
			break
		}
	}
	if marker == -1 {
		return
	}
	encoded, err := json.Marshal(os.Args[marker+1:])
	if err != nil {
		t.Fatal(err)
	}
	fmt.Print(string(encoded))
	os.Exit(0)
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
