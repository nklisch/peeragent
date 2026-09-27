//go:build windows

package executil

import (
	"context"
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
		if err := os.WriteFile(path, []byte("@echo off\r\necho %~1\r\n"), 0o644); err != nil {
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
