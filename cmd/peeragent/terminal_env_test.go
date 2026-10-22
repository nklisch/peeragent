package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/testsupport/terminalenv"
)

func TestTerminalEnvironmentLaunchPathsAndMetadata(t *testing.T) {
	for _, mode := range []string{"blocking", "blocking-resume", "async", "async-resume", "retry"} {
		for _, keep := range []bool{false, true} {
			name := mode
			if keep {
				name += "-keep"
			}
			t.Run(name, func(t *testing.T) {
				cwd, _ := fakeCodexEnvironment(t)
				for _, key := range terminalenv.TerminalEnvNames() {
					t.Setenv(key, "fixture")
				}
				t.Setenv("PEERAGENT_ENV_SENTINEL", "ordinary=value")
				t.Setenv("PEERAGENT_TERMINAL_ENV_HELPER", "1")
				args := []string{}
				async := strings.HasPrefix(mode, "async") || mode == "retry"
				if async {
					args = append(args, "--async")
				}
				if strings.HasSuffix(mode, "resume") {
					args = append(args, "--resume", "existing-session")
				}
				if keep {
					args = append(args, "--keep-terminal-env")
				}
				if mode == "retry" {
					t.Setenv("PEERAGENT_FAKE_MODE", "terminal-env-retry")
					t.Setenv("PEERAGENT_ENV_RECORD", filepath.Join(cwd, "attempt"))
					args = append(args, "--retry-capacity", "1", "--capacity-backoff", "1ms")
				}
				launched := cliResult(t, cwd, append(args, "dump fixture environment")...)
				if launched.Metadata.KeepTerminalEnv != keep {
					t.Fatalf("launch metadata lost policy: %#v", launched.Metadata)
				}
				terminal := launched
				if async {
					terminal = cliResult(t, cwd, "--wait", launched.Metadata.JobID)
				}
				if terminal.Status != result.StatusSuccess || terminal.Metadata.KeepTerminalEnv != keep {
					t.Fatalf("terminal metadata: %#v", terminal)
				}
				job, err := jobs.NewStore(cwd).Load(terminal.Metadata.JobID)
				if err != nil || job.Spec.KeepTerminalEnv != keep {
					t.Fatalf("persisted policy: %#v %v", job, err)
				}
				status := cliResult(t, cwd, "--status", job.ID)
				if status.Metadata.KeepTerminalEnv != keep {
					t.Fatalf("status metadata lost policy: %#v", status)
				}
				multiText := runCLIOutput(t, cwd, "--status", job.ID, job.ID, "--text")
				if strings.Contains(multiText, "terminal environment retained") != keep {
					t.Fatalf("multi-status lost policy: %s", multiText)
				}
				multiJSON := runCLIOutput(t, cwd, "--status", job.ID, job.ID)
				if strings.Contains(multiJSON, `"keep_terminal_env":true`) != keep {
					t.Fatalf("multi-status JSON lost policy: %s", multiJSON)
				}
				text := result.FormatText(terminal)
				if strings.Contains(text, "keep_terminal_env: true") != keep {
					t.Fatalf("text metadata lost policy: %s", text)
				}
				var env map[string]string
				if err := json.Unmarshal([]byte(strings.TrimPrefix(terminal.Details, "stdout:\n")), &env); err != nil {
					t.Fatal(err)
				}
				checkFixtureEnvironment(t, env, keep)
				if mode == "retry" {
					if terminal.Metadata.CapacityRetries != 1 {
						t.Fatalf("retry missing: %#v", terminal)
					}
					for attempt := 1; attempt <= 2; attempt++ {
						data, err := os.ReadFile(fmt.Sprintf("%s-%d.json", os.Getenv("PEERAGENT_ENV_RECORD"), attempt))
						if err != nil {
							t.Fatal(err)
						}
						if err := json.Unmarshal(data, &env); err != nil {
							t.Fatal(err)
						}
						checkFixtureEnvironment(t, env, keep)
					}
				}
			})
		}
	}
}

func checkFixtureEnvironment(t *testing.T, env map[string]string, keep bool) {
	t.Helper()
	if env["PEERAGENT_ENV_SENTINEL"] != "ordinary=value" {
		t.Fatal("ordinary environment lost")
	}
	for _, key := range terminalenv.TerminalEnvNames() {
		value, exists := env[key]
		if exists != keep || keep && value != "fixture" {
			t.Fatalf("%s exists=%t value=%q keep=%t", key, exists, value, keep)
		}
	}
}
