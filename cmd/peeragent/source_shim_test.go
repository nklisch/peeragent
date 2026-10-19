package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSourceShimPreservesUsageAndLookupExitCodes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX source shim requires a Unix shell")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(filepath.Join(root, "cmd", "peeragent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	shim, err := os.ReadFile(filepath.Join("..", "..", "bin", "peeragent"))
	if err != nil {
		t.Fatal(err)
	}
	shimPath := filepath.Join(bin, "peeragent")
	if err := os.WriteFile(shimPath, shim, 0o755); err != nil {
		t.Fatal(err)
	}
	// Match Go's process contract: build produces an executable, while go run
	// maps a non-zero program exit to 1. The helper executable runs our real CLI.
	fakeGo := `#!/bin/sh
case "$1" in
 env) printf '\n' ;;
 build) cp "$PEERAGENT_TEST_EXE" "$3" ;;
 run)
  shift 2
  "$PEERAGENT_TEST_EXE" "$@"
  code=$?
  [ "$code" -eq 0 ] && exit 0
  exit 1
  ;;
 *) exit 99 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(fakeGo), 0o755); err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PEERAGENT_TEST_EXE", executable)
	t.Setenv("PEERAGENT_TEST_HELPER_MAIN", "1")
	t.Setenv("PEERAGENT_BIN", "")
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	t.Setenv("GOTMPDIR", temporary)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"--list"}, 2},
		{[]string{"--background"}, 2},
		{[]string{"--status", "20000101T000000Z-00000000"}, 4},
	} {
		cmd := exec.Command("sh", append([]string{shimPath}, tc.args...)...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != tc.code {
			t.Fatalf("%q: exit %v, want %d; %s", tc.args, err, tc.code, out)
		}
		res := decodeResult(t, out)
		if res.Metadata.ExitCode != tc.code {
			t.Fatalf("%q: metadata %#v", tc.args, res.Metadata)
		}
		entries, err := os.ReadDir(temporary)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("source shim left temporary binaries: %v", entries)
		}
	}
}
