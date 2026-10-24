package main

import (
	"encoding/json"
	"strings"

	"github.com/nklisch/peeragent/internal/version"
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

func TestInstalledShimFollowsActiveClaudeInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX installed shim requires a Unix shell")
	}
	for _, wrapper := range []string{"bin", "plugin/bin"} {
		t.Run(wrapper, func(t *testing.T) {
			shim, err := os.ReadFile(filepath.Join("..", "..", wrapper, "peeragent"))
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			config := filepath.Join(root, "Claude config")
			old := filepath.Join(config, "plugins", "cache", "test-market", "peeragent", "0.12.2")
			active := filepath.Join(config, "plugins", "cache", "test-market", "peeragent", `active "version"`)
			write := func(path, content string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for _, install := range []struct{ path, name string }{{old, "old"}, {active, "active"}} {
				write(filepath.Join(install.path, "bin", "peeragent"), string(shim))
				write(filepath.Join(install.path, "bin", "linux-amd64", "peeragent"), "#!/bin/sh\nprintf '%s\\n' '"+install.name+"' \"marker=${PEERAGENT_ACTIVE_INSTALL:-}\" \"$@\"\nexit 7\n")
			}
			t.Setenv("HOME", root)
			t.Setenv("CLAUDE_CONFIG_DIR", config)
			t.Setenv("PEERAGENT_BIN", "")
			t.Setenv("PEERAGENT_ACTIVE_INSTALL", "")
			t.Setenv("PEERAGENT_TARGET_OVERRIDE", "linux-amd64")
			record := filepath.Join(config, "plugins", "installed_plugins.json")
			invoked := old
			registry := func(install string) {
				t.Helper()
				data, err := json.Marshal(map[string]any{"version": 2, "plugins": map[string]any{"peeragent@test-market": []any{map[string]string{"scope": "user", "installPath": install}}}})
				if err != nil {
					t.Fatal(err)
				}
				write(record, string(data))
			}
			run := func(want string, forwarded bool) {
				t.Helper()
				cmd := exec.Command("sh", filepath.Join(invoked, "bin", "peeragent"), "one argument", "--literal", "")
				if forwarded {
					cmd = exec.Command("sh", "-c", `PEERAGENT_ACTIVE_INSTALL=$$; export PEERAGENT_ACTIVE_INSTALL; exec "$@"`, "sh", filepath.Join(old, "bin", "peeragent"), "one argument", "--literal", "")
				}
				out, err := cmd.CombinedOutput()
				exit, ok := err.(*exec.ExitError)
				if !ok || exit.ExitCode() != 7 || string(out) != want+"\nmarker=\none argument\n--literal\n\n" {
					t.Fatalf("want %s with unchanged args and exit 7: %v %q", want, err, out)
				}
			}
			run("old", false) // No record.
			write(record, "{broken")
			run("old", false)
			registry(old)
			run("old", false) // The active install is this one.
			registry(filepath.Join(root, "missing"))
			run("old", false)
			registry(active)
			run("active", false)
			invoked = filepath.Join(root, ".codex", "plugins", "cache", "test-market", "peeragent", "0.12.2")
			write(filepath.Join(invoked, "bin", "peeragent"), string(shim))
			write(filepath.Join(invoked, "bin", "linux-amd64", "peeragent"), "#!/bin/sh\nprintf '%s\\n' old \"marker=${PEERAGENT_ACTIVE_INSTALL:-}\" \"$@\"\nexit 7\n")
			run("old", false) // A Codex wrapper ignores Claude's record.
			invoked = old
			// Exercise jq without Python, then the dependency-free fallback.
			originalPath := os.Getenv("PATH")
			readers := filepath.Join(root, "readers")
			for _, name := range []string{"sh", "dirname", "jq"} {
				path, err := exec.LookPath(name)
				if err != nil {
					if name == "jq" {
						continue
					}
					t.Fatal(err)
				}
				if err := os.MkdirAll(readers, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path, filepath.Join(readers, name)); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", readers)
			if _, err := os.Stat(filepath.Join(readers, "jq")); err == nil {
				run("active", false)
				if err := os.Remove(filepath.Join(readers, "jq")); err != nil {
					t.Fatal(err)
				}
			}
			run("old", false)
			t.Setenv("PATH", originalPath)
			run("old", true) // An inconsistent registry cannot redirect twice.
			t.Setenv("PEERAGENT_ACTIVE_INSTALL", "1")
			run("active", false) // An inherited marker belongs to another process.
			t.Setenv("PEERAGENT_ACTIVE_INSTALL", "")
			// Developer builds and the explicit binary override still win.
			write(filepath.Join(old, "dist", "peeragent"), "#!/bin/sh\nprintf '%s\\n' old \"marker=${PEERAGENT_ACTIVE_INSTALL:-}\" \"$@\"\nexit 7\n")
			run("old", false)
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("PEERAGENT_BIN", binary)
			t.Setenv("PEERAGENT_TEST_HELPER_MAIN", "1")
			out, err := exec.Command("sh", filepath.Join(old, "bin", "peeragent"), "--version").CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != version.Version {
				t.Fatalf("override: %v %s", err, out)
			}
		})
	}
}
