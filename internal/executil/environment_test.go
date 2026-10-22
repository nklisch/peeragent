package executil

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport/terminalenv"
)

func TestTargetEnvironmentPolicy(t *testing.T) {
	preserved := []string{"PATH=/fake/bin", "OPENAI_API_KEY=fixture", "ANTHROPIC_API_KEY=fixture", "ZAI_API_KEY=fixture", "TERM=xterm-256color", "COLORTERM=truecolor", "TERMINFO=/fake/info", "TERM_PROGRAM=fixture", "HERDR=ordinary", "HERDRX_PANE=ordinary", "ZELLIJX=ordinary", "KONSOLE_PROFILE=ordinary", "DISPLAY=:99", "DBUS_SESSION_BUS_ADDRESS=fixture", "VALUE=embedded=equals", "=C:=C:\\fixture"}
	env := append([]string(nil), preserved...)
	for _, name := range terminalenv.TerminalEnvNames() {
		env = append(env, name+"=fixture")
	}
	original := append([]string(nil), env...)
	if got := targetEnvironment(env, false); !reflect.DeepEqual(got, preserved) {
		t.Fatalf("filtered environment: %q", got)
	}
	if !reflect.DeepEqual(env, original) {
		t.Fatal("filter changed parent environment")
	}
	if got := targetEnvironment(env, true); !reflect.DeepEqual(got, original) {
		t.Fatalf("override changed environment: %q", got)
	}
	lower := []string{"herdr_pane_id=fixture", "tMuX=fixture"}
	if runtime.GOOS == "windows" {
		if len(targetEnvironment(lower, false)) != 0 {
			t.Fatal("Windows case-insensitive names survived")
		}
	} else if got := targetEnvironment(lower, false); !reflect.DeepEqual(got, lower) {
		t.Fatal("Unix case-sensitive names changed")
	}
}

func TestOSRunnerTerminalEnvironment(t *testing.T) {
	for _, name := range terminalenv.TerminalEnvNames() {
		t.Setenv(name, "fixture")
	}
	t.Setenv("PEERAGENT_ENV_SENTINEL", "ordinary=value")
	t.Setenv("PEERAGENT_ENV_DUMP_HELPER", "1")
	for _, observed := range []bool{false, true} {
		for _, keep := range []bool{false, true} {
			name := "ordinary"
			if observed {
				name = "observed"
			}
			if keep {
				name += "-keep"
			}
			t.Run(name, func(t *testing.T) {
				runner := OSRunner{KeepTerminalEnv: keep}
				if observed {
					runner.ObserveStdout = func(string) (CompletionObservation, error) { return CompletionUnchanged, nil }
				}
				res, err := runner.Run(context.Background(), os.Args[0], []string{"-test.run=^TestTerminalEnvironmentHelper$"}, t.TempDir(), "")
				if err != nil || res.ExitCode != 0 {
					t.Fatalf("fake agent: %#v %v", res, err)
				}
				var env map[string]string
				if err := json.Unmarshal([]byte(res.Stdout), &env); err != nil {
					t.Fatal(err)
				}
				if env["PEERAGENT_ENV_SENTINEL"] != "ordinary=value" {
					t.Fatal("ordinary environment lost")
				}
				for _, key := range terminalenv.TerminalEnvNames() {
					value, exists := env[key]
					if exists != keep || keep && value != "fixture" {
						t.Fatalf("%s exists=%t value=%q keep=%t", key, exists, value, keep)
					}
				}
			})
		}
	}
	for _, name := range terminalenv.TerminalEnvNames() {
		if os.Getenv(name) != "fixture" {
			t.Fatal("target launch changed parent environment")
		}
	}
}

func TestTerminalEnvironmentHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_ENV_DUMP_HELPER") != "1" {
		return
	}
	data, err := json.Marshal(terminalenv.TerminalEnvSnapshot())
	if err != nil {
		panic(err)
	}
	os.Stdout.WriteString(strings.TrimSpace(string(data)))
	os.Exit(0)
}
