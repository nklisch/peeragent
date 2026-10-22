package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/testsupport/terminalenv"
)

func TestEveryTargetFiltersTerminalEnvironment(t *testing.T) {
	root := t.TempDir()
	data, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"codex", "claude", "agy", "pi"} {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PEERAGENT_TERMINAL_ENV_HELPER", "1")
	t.Setenv("PEERAGENT_ENV_SENTINEL", "ordinary=value")
	for _, name := range terminalenv.TerminalEnvNames() {
		t.Setenv(name, "fixture")
	}
	for _, agent := range []string{"codex", "claude", "gemini", "zai"} {
		for _, keep := range []bool{false, true} {
			for _, resume := range []string{"", "existing-session"} {
				name := agent
				if keep {
					name += "-keep"
				}
				if resume != "" {
					name += "-resume"
				}
				t.Run(name, func(t *testing.T) {
					delegation := input.Delegation{CWD: root, Agent: agent, TaskText: "dump fixture environment", Resume: resume, KeepTerminalEnv: keep}
					res, err := (targetExecutor{}).Execute(executil.WithJobContext(context.Background(), "fixture-job", root), delegation)
					if err != nil || res.ExitCode != 0 {
						t.Fatalf("target execution: %#v %v", res, err)
					}
					var env map[string]string
					if err := json.Unmarshal([]byte(res.Stdout), &env); err != nil {
						t.Fatalf("fake target stdout: %q %v", res.Stdout, err)
					}
					if env["PEERAGENT_JOB_ID"] != "fixture-job" || env["PEERAGENT_JOB_DIR"] != root {
						t.Fatalf("job identity missing: %#v", env)
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
	}
}
