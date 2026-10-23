package jobs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/prompt"
)

func withoutMessagingMarker(t *testing.T, store Store, job Job) Job {
	t.Helper()
	path := filepath.Join(filepath.Dir(job.LogPath), "job.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "messaging_enabled")
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestMessagingSupportUsesRecordedOrObservedEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, text, outbox, source              string
		legacy, noPreamble, missingPrompt, want bool
	}{
		{name: "recorded enabled", want: true, source: "recorded in job.json"},
		{name: "recorded disabled remains explicit", noPreamble: true, text: prompt.MessagingPreamble, outbox: "sent", source: "recorded in job.json"},
		{name: "legacy prompt", legacy: true, text: prompt.Build("work"), want: true, source: "inferred from prompt"},
		{name: "original v0.11 preamble", legacy: true, text: "Your launching session may send you messages. Run `peeragent inbox` after each long step and before your handback.\nTo reach the launching session, use `peeragent say \"<text>\"`; never use terminal, desktop or keyboard input, and\nkeep messages for questions you can't settle and short status the launcher asked for.", want: true, source: "inferred from prompt"},
		{name: "legacy Windows line endings", legacy: true, text: strings.ReplaceAll(prompt.Build("work"), "\n", "\r\n"), want: true, source: "inferred from prompt"},
		{name: "legacy say", legacy: true, text: "old task", outbox: "sent", want: true, source: "inferred from outbox"},
		{name: "legacy refused say", legacy: true, missingPrompt: true, outbox: "refused", want: true, source: "inferred from outbox"},
		{name: "true pre-messaging job", legacy: true, text: "old task", source: "no messaging evidence"},
		{name: "missing prompt without evidence", legacy: true, missingPrompt: true, source: "no messaging evidence"},
		{name: "mention is not a preamble", legacy: true, text: "fix peeragent inbox", source: "no messaging evidence"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			store := NewStore(cwd)
			job, err := store.Create(cwd, ExecSpec{Agent: "codex", NoPreamble: tc.noPreamble}, tc.text)
			if err != nil {
				t.Fatal(err)
			}
			if tc.legacy {
				job = withoutMessagingMarker(t, store, job)
			}
			if tc.missingPrompt {
				if err := os.Remove(job.PromptPath); err != nil {
					t.Fatal(err)
				}
			}
			mailbox := Mailbox{}
			if tc.outbox != "" {
				mailbox.Outbox = []Message{{From: "codex", Text: "question", Status: tc.outbox}}
			}
			support, err := store.MessagingSupport(job, mailbox)
			if err != nil || support.Enabled != tc.want || support.Source != tc.source || (support.Warning != "") == tc.want {
				t.Fatalf("support %#v %v", support, err)
			}
		})
	}
}

func TestLegacyMessagingMarkerStaysAbsentAcrossLifecycleSaves(t *testing.T) {
	cwd := t.TempDir()
	store := NewStore(cwd)
	job, err := store.Create(cwd, ExecSpec{Agent: "codex"}, prompt.Build("work"))
	if err != nil {
		t.Fatal(err)
	}
	job = withoutMessagingMarker(t, store, job)
	job.FinalReceived = true
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(job.LogPath), "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"messaging_enabled"`) {
		t.Fatalf("save invented a legacy flag: %s", data)
	}
	job, err = store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	support, err := store.MessagingSupport(job, Mailbox{})
	if err != nil || !support.Enabled || support.Source != "inferred from prompt" || !job.FinalReceived {
		t.Fatalf("lost legacy evidence/lifecycle: %#v %#v %v", job, support, err)
	}
}

func TestMessagingEvidenceReportsUnreadablePrompt(t *testing.T) {
	cwd := t.TempDir()
	store := NewStore(cwd)
	job, err := store.Create(cwd, ExecSpec{}, "old task")
	if err != nil {
		t.Fatal(err)
	}
	job = withoutMessagingMarker(t, store, job)
	if err := os.Remove(job.PromptPath); err != nil {
		t.Fatal(err)
	}
	// A directory instead of a file fails on all supported platforms, including
	// when tests run as a privileged user who could bypass file permissions.
	if err := os.Mkdir(job.PromptPath, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MessagingSupport(job, Mailbox{}); err == nil || !strings.Contains(err.Error(), "messaging evidence") {
		t.Fatalf("hidden evidence read error: %v", err)
	}
}
