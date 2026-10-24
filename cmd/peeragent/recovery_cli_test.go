package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestRecoveryCLIAndRemovedWorktreeResult(t *testing.T) {
	for _, tc := range []struct {
		mode   string
		budget int
		want   result.Status
		calls  int
	}{
		{"auth-recovery", 0, result.StatusAuthRequired, 1},
		{"auth-recovery", 1, result.StatusSuccess, 2},
		{"auth-persistent", 5, result.StatusAuthRequired, 2},
		{"rollout-lost", 5, result.StatusInterrupted, 1},
		{"refusal-rollout", 5, result.StatusFailed, 1},
	} {
		t.Run(tc.mode+"-"+string(tc.want), func(t *testing.T) {
			cwd, record := fakeCodexEnvironment(t)
			repo := filepath.Join(cwd, "worktree")
			if err := os.Mkdir(repo, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PEERAGENT_FAKE_MODE", tc.mode)
			args := []string{"--async", "--full-access"}
			if tc.budget > 0 {
				args = append(args, "--retry-capacity", strconv.Itoa(tc.budget))
			}
			var launched, terminal result.Result
			if !t.Run("owned worker", func(t *testing.T) {
				launched = cliResult(t, repo, append(args, "work")...)
				terminal = cliResult(t, cwd, "--wait", launched.Metadata.JobID)
			}) {
				return
			}
			if terminal.Status != tc.want || terminal.Metadata.AgentSession != "fake-session" {
				t.Fatalf("terminal: %#v", terminal)
			}
			var calls []struct {
				Args []string
				CWD  string
			}
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &calls); err != nil {
				t.Fatal(err)
			}
			if len(calls) != tc.calls {
				t.Fatalf("calls=%d want %d", len(calls), tc.calls)
			}
			if tc.calls > 1 && (!strings.Contains(strings.Join(calls[1].Args, " "), "exec resume") || !strings.Contains(strings.Join(calls[1].Args, " "), "fake-session")) {
				t.Fatalf("fresh session on retry: %#v", calls)
			}
			status := cliResult(t, cwd, "--status", launched.Metadata.JobID)
			if tc.mode == "refusal-rollout" && (terminal.Failure != "policy_refusal" || status.Failure != "policy_refusal" || !strings.HasPrefix(status.Summary, "This content was flagged")) {
				t.Fatalf("refusal hidden: %#v %#v", terminal, status)
			}
			// The nested test has waited for the owned worker to exit. Remove only
			// the completed job metadata we created. Leaving cwd and the
			// fake CLI intact lets a fresh process resolve the archived result by id.
			if err := os.RemoveAll(filepath.Join(repo, ".peeragent")); err != nil {
				t.Fatal(err)
			}
			archived := cliResult(t, cwd, "--result", launched.Metadata.JobID)
			if archived.Status != tc.want || archived.Metadata.AgentSession != "fake-session" || archived.Details != terminal.Details || archived.Failure != terminal.Failure {
				t.Fatalf("archive: %#v terminal %#v", archived, terminal)
			}
			if _, err := os.Stat(filepath.Join(jobs.NewStore(repo).Root, launched.Metadata.JobID)); !os.IsNotExist(err) {
				t.Fatalf("lookup recreated deleted job: %v", err)
			}
		})
	}
}

func TestArchivedStatusSeparatesTargetExitCodeFromLookup(t *testing.T) {
	cwd := t.TempDir()
	t.Setenv("PEERAGENT_TEST_HELPER_MAIN", "1")
	const id = "20260712T123456Z-deadbeef"
	if err := app.WriteJobResult(filepath.Join(cwd, "result.json"), result.Result{Status: result.StatusFailed, Summary: "target exited with four", Metadata: result.Metadata{JobID: id, CWD: cwd, ExitCode: 4}}); err != nil {
		t.Fatal(err)
	}
	// No repository job metadata exists, so both controls use the snapshot.
	status := cliResult(t, cwd, "--status", id)
	if status.Status != result.StatusFailed || status.Metadata.ExitCode != 0 {
		t.Fatalf("status treated target exit as lookup failure: %#v", status)
	}
	res, err := app.NewService(app.Options{}).JobResult(context.Background(), app.JobRequest{CWD: cwd, JobID: id})
	if err != nil || res.Metadata.ExitCode != 4 {
		t.Fatalf("target exit lost: %#v %v", res, err)
	}
	cmd := exec.Command(os.Args[0], "--status", "20260712T123456Z-00000000")
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 4 || decodeResult(t, output).Metadata.ExitCode != 4 {
		t.Fatalf("missing lookup exit: %v %s", err, output)
	}
}
