package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestResumeOptionsCLI(t *testing.T) {
	cwd, record := fakeCodexEnvironment(t)
	t.Setenv("PEERAGENT_FAKE_MODE", "messaging") // Always succeeds, including the first turn.
	first := cliResult(t, cwd, "--model", "luna", "--effort", "xhigh", "work")
	if first.Metadata.AgentSession != "fake-session" {
		t.Fatalf("initial session = %#v", first.Metadata)
	}
	for _, tc := range []struct {
		name, session, model, effort string
		flags                        []string
		async, warning               bool
	}{
		{"inherit-blocking", "fake-session", "gpt-6-luna", "xhigh", nil, false, false},
		{"override-effort-async", "fake-session", "gpt-6-luna", "high", []string{"--effort", "high"}, true, false},
		{"inherit-most-recent", "fake-session", "gpt-6-luna", "high", nil, true, false},
		{"override-model-blocking", "fake-session", "gpt-6-astra", "high", []string{"--model", "astra"}, false, false},
		{"inherit-updated-model", "fake-session", "gpt-6-astra", "high", nil, true, false},
		{"unknown-defaults", "unknown-session", "", "high", nil, true, true},
		{"unknown-partial-override", "another-unknown", "gpt-6.1-sol", "high", []string{"--model", "sol"}, true, true},
		{"unknown-explicit-options", "third-unknown", "gpt-6.1-sol", "medium", []string{"--model", "sol", "--effort", "medium"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--resume", tc.session, "continue"}, tc.flags...)
			if tc.async {
				args = append(args, "--async")
			}
			// Lookup from another directory exercises the session index. Unknown
			// sessions have no index and use this directory as today's fallback.
			launched := cliResult(t, t.TempDir(), args...)
			assertMetadata := func(res result.Result) {
				t.Helper()
				if res.Metadata.Model != tc.model || res.Metadata.Effort != tc.effort || (res.Metadata.ResumeWarning != "") != tc.warning {
					t.Fatalf("metadata = %#v; want model %q, effort %q, warning %t", res.Metadata, tc.model, tc.effort, tc.warning)
				}
				if tc.warning && !strings.Contains(res.Metadata.ResumeWarning, "current defaults") {
					t.Fatalf("fallback note = %q", res.Metadata.ResumeWarning)
				}
			}
			assertMetadata(launched)
			if tc.async {
				assertMetadata(cliResult(t, cwd, "--wait", launched.Metadata.JobID))
			}
			assertMetadata(cliResult(t, cwd, "--status", launched.Metadata.JobID))
			job, err := jobs.NewStore(launched.Metadata.CWD).Load(launched.Metadata.JobID)
			if err != nil || job.Spec.Model != tc.model || job.Spec.Effort != tc.effort || (job.ResumeWarning != "") != tc.warning {
				t.Fatalf("saved job = %#v, %v", job, err)
			}
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			var calls []struct{ Args []string }
			if err := json.Unmarshal(data, &calls); err != nil || len(calls) == 0 {
				t.Fatalf("calls = %s, %v", data, err)
			}
			argv := strings.Join(calls[len(calls)-1].Args, " ")
			if !strings.Contains(argv, "resume --json") || !strings.Contains(argv, tc.session) || !strings.Contains(argv, `model_reasoning_effort="`+tc.effort+`"`) {
				t.Fatalf("resume arguments = %s", argv)
			}
			if tc.model != "" && !strings.Contains(argv, "--model "+tc.model) || tc.model == "" && strings.Contains(argv, "--model") {
				t.Fatalf("model arguments = %s", argv)
			}
		})
	}
}
