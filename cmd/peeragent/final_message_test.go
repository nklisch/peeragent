package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/result"
)

// A long, multi-line handback past the 600-character summary cap.
var fakeHandback = "Handback — done.\n\n" + strings.Repeat("Every line of the report stays intact. ", 40) + "\nLast line."

// A copy of the test executable acts as Claude Code's print mode.
func fakeClaudeMain() {
	data, _ := json.Marshal(map[string]any{"type": "result", "subtype": "success", "result": fakeHandback, "session_id": "claude-session"})
	fmt.Println(string(data))
}

func installFakeClaude(t *testing.T, root string) {
	t.Helper()
	name := "claude"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	content, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", name), content, 0o755); err != nil {
		t.Fatal(err)
	}
}

// rawFinalMessage returns the final_message JSON value, failing if the key is absent.
func rawFinalMessage(t *testing.T, dir string, args ...string) json.RawMessage {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%q: %v\n%s", args, err, out)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(out, &fields); err != nil {
		t.Fatalf("decode %q: %v\n%s", args, err, out)
	}
	value, ok := fields["final_message"]
	if !ok {
		t.Fatalf("%q: final_message missing\n%s", args, out)
	}
	return value
}

func finalMessage(res result.Result) string {
	if res.FinalMessage == nil {
		return "<null>"
	}
	return *res.FinalMessage
}

func TestFinishedClaudeJobReportsFinalMessage(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	installFakeClaude(t, cwd)
	launched := cliResult(t, cwd, "--async", "--agent", "claude", "work")
	id := launched.Metadata.JobID
	for _, control := range []string{"--wait", "--result"} {
		res := cliResult(t, cwd, control, id)
		if res.Status != result.StatusSuccess || finalMessage(res) != fakeHandback {
			t.Fatalf("%s: status=%s final_message=%q", control, res.Status, finalMessage(res))
		}
		// Summary and details keep their established shapes.
		if res.Details != "stdout:\n"+fakeHandback || len([]rune(res.Summary)) != 600 || !strings.HasSuffix(res.Summary, "…") {
			t.Fatalf("%s: summary %q details %q", control, res.Summary, res.Details)
		}
	}
}

func TestCodexFinalMessageIsNullWhileRunningAndCompleteWhenFinished(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "live")
	message := strings.Repeat("complete report — ", 12000)
	launched := cliResult(t, cwd, "--async", "work")
	id := launched.Metadata.JobID
	// An interim agent message is visible in details but is not yet final.
	waitForObservation(t, cwd, id, "--result", func(res result.Result) bool { return res.Metadata.MessagePath != "" })
	running := cliResult(t, cwd, "--result", id)
	if running.Status != result.StatusRunning || running.FinalMessage != nil || running.Details != "stdout:\n"+message {
		t.Fatalf("running: status=%s final_message=%.40q details bytes=%d", running.Status, finalMessage(running), len(running.Details))
	}
	if raw := rawFinalMessage(t, cwd, "--result", id); string(raw) != "null" {
		t.Fatalf("running final_message = %s, want null", raw)
	}
	if err := os.WriteFile(os.Getenv("PEERAGENT_FAKE_GATE"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, control := range []string{"--wait", "--result"} {
		res := cliResult(t, cwd, control, id)
		if res.Status != result.StatusSuccess || finalMessage(res) != message || res.Details != "stdout:\n"+message {
			t.Fatalf("%s: status=%s final_message bytes=%d", control, res.Status, len(finalMessage(res)))
		}
	}
}

func TestRunningCodexReportsFinalMessageOnceReceived(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "final-hang")
	message := strings.Repeat("complete report — ", 12000)
	launched := cliResult(t, cwd, "--async", "work")
	id := launched.Metadata.JobID
	observed := waitForObservation(t, cwd, id, "--result", func(res result.Result) bool { return res.Metadata.FinalReceived })
	if observed.Status != result.StatusRunning || finalMessage(observed) != message {
		t.Fatalf("final received while running: status=%s final_message bytes=%d", observed.Status, len(finalMessage(observed)))
	}
	terminal := cliResult(t, cwd, "--wait", id)
	if terminal.Status != result.StatusSuccess || finalMessage(terminal) != message {
		t.Fatalf("terminal: status=%s final_message bytes=%d", terminal.Status, len(finalMessage(terminal)))
	}
}

func TestFailedCodexJobFinalMessage(t *testing.T) {
	for _, tc := range []struct {
		mode   string
		status result.Status
		want   string
	}{
		// The agent's last message before a failing exit is still its handback.
		{"failed-message", result.StatusFailed, strings.Repeat("complete report — ", 12000)},
		// Raw CLI events are not an agent message.
		{"ordinary", result.StatusFailed, "<null>"},
		// A capacity stop leaves an interim message, not a final one.
		{"capacity-message", result.StatusCapacity, "<null>"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			cwd, _ := fakeCodexEnvironment(t)
			t.Setenv("PEERAGENT_FAKE_MODE", tc.mode)
			launched := cliResult(t, cwd, "--async", "work")
			terminal := cliResult(t, cwd, "--wait", launched.Metadata.JobID)
			if terminal.Status != tc.status || finalMessage(terminal) != tc.want || !strings.HasPrefix(terminal.Details, "stdout:\n") {
				t.Fatalf("status=%s final_message=%.60q details=%.60q", terminal.Status, finalMessage(terminal), terminal.Details)
			}
		})
	}
}
