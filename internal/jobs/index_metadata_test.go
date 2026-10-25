package jobs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIndexedMetadataLegacyAndRetentionBounds(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	const id = "20261024T010203Z-12345678"
	if err := IndexJob(id, cwd); err != nil {
		t.Fatal(err)
	}
	if got, err := LookupJob(id); err != nil || got != cwd {
		t.Fatalf("legacy directory: %q %v", got, err)
	}
	entry := IndexedJob{ID: id, Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh", CWD: cwd, Status: StatusComplete, AgentSession: "saved-session", Summary: strings.Repeat("界", 700), StartedAt: time.Now().UTC()}
	finished := entry.StartedAt.Add(time.Minute)
	entry.FinishedAt = &finished
	if _, err := WriteIndexedJob(entry); err != nil {
		t.Fatal(err)
	}
	got, found, err := ReadIndexedJob(id)
	if err != nil || !found || got.AgentSession != entry.AgentSession || got.Model != entry.Model || got.Effort != entry.Effort || got.FinishedAt == nil || !got.FinishedAt.Equal(finished) || len([]rune(got.Summary)) != 600 || !strings.HasSuffix(got.Summary, "…") {
		t.Fatalf("metadata: %#v %v %v", got, found, err)
	}
	if got, err := LookupJob(id); err != nil || got != cwd {
		t.Fatalf("JSON directory: %q %v", got, err)
	}
	if got, err := LookupSession(entry.Agent, entry.AgentSession); err != nil || got != cwd {
		t.Fatalf("session directory: %q %v", got, err)
	}
	path, _ := indexPath("jobs", id)
	dir := filepath.Dir(path)
	now := time.Now()
	if err := os.Chtimes(path, now.Add(-31*24*time.Hour), now.Add(-31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, found, err := ReadIndexedJob(id); err != nil || found {
		t.Fatalf("expired: %t %v", found, err)
	}
	for i, key := range []string{"20261024T010203Z-11111111", "20261024T010203Z-22222222", "20261024T010203Z-33333333"} {
		if _, err := WriteIndexedJob(IndexedJob{ID: key, CWD: cwd, Status: StatusRunning}); err != nil {
			t.Fatal(err)
		}
		p, _ := indexPath("jobs", key)
		stamp := now.Add(time.Duration(i) * time.Second)
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	unfinished := filepath.Join(dir, ".index-pending")
	if err := os.WriteFile(unfinished, []byte("unowned"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := pruneJobIndex(dir, now.Add(-30*24*time.Hour), 2); err != nil {
		t.Fatal(err)
	}
	entries, err := IndexedJobs()
	if err != nil || len(entries) != 2 {
		t.Fatalf("bounded metadata: %#v %v", entries, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expired file kept: %v", err)
	}
	if _, err := os.Stat(unfinished); err != nil {
		t.Fatalf("temporary file pruned: %v", err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupJob(id); err == nil {
		t.Fatal("corrupt JSON treated as a directory")
	}
}
