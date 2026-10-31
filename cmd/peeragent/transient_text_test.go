package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/result"
)

func TestTransportRetryBudget(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(map[bool]string{false: "no retry", true: "retry"}[retry], func(t *testing.T) {
			cwd, record := fakeCodexEnvironment(t)
			t.Setenv("PEERAGENT_FAKE_MODE", "transport-retry")
			args := []string{"--async"}
			if retry {
				args = append(args, "--retry-capacity", "1", "--capacity-backoff", "1ms")
			}
			launched := cliResult(t, cwd, append(args, "work")...)
			got := cliResult(t, cwd, "--wait", launched.Metadata.JobID)
			want, calls := result.StatusCapacity, 1
			if retry {
				want, calls = result.StatusSuccess, 2
			}
			if got.Status != want || got.Metadata.CapacityRetries != calls-1 {
				t.Fatalf("result: %#v", got)
			}
			var attempts []struct{ Args []string }
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &attempts); err != nil {
				t.Fatal(err)
			}
			if len(attempts) != calls || retry && !strings.Contains(strings.Join(attempts[1].Args, " "), "resume --json") {
				t.Fatalf("attempts: %s", data)
			}
		})
	}
}

func TestTextLaunchFirstLineWithCombinedOutput(t *testing.T) {
	cwd, _ := fakeCodexEnvironment(t)
	cmd := exec.Command(os.Args[0], "--async", "--text", "task")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("launch: %s %v", out, err)
	}
	first := strings.SplitN(string(out), "\n", 2)[0]
	if !strings.HasPrefix(first, "job ") || !strings.Contains(string(out), "--text selects the output format") {
		t.Fatalf("launch output: %s", out)
	}
	cliResult(t, cwd, "--wait", strings.TrimPrefix(first, "job "))
}

func TestTextHintOnlyForAmbiguousArgument(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"--async", "--text", "task"}, true},
		{[]string{"--async", "--text", "--", "task"}, false},
		{[]string{"--async", "--text", "task", "more"}, false},
		{[]string{"--async", "task", "--text"}, false},
	} {
		cwd, _ := fakeCodexEnvironment(t)
		out, diagnostic, err := controlOutput(t, cwd, tc.args...)
		if err != nil {
			t.Fatalf("launch: %s %s %v", out, diagnostic, err)
		}
		first := strings.SplitN(out, "\n", 2)[0]
		if !strings.HasPrefix(first, "job ") {
			t.Fatalf("first line: %q", out)
		}
		if strings.Contains(diagnostic, "--text selects the output format") != tc.want {
			t.Fatalf("hint: %q", diagnostic)
		}
		cliResult(t, cwd, "--wait", strings.TrimPrefix(first, "job "))
	}
}
