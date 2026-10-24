package jobs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestArchivedResultsExpireAndPruneOnlyOwnedSnapshots(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	const oldID = "20000101T000000Z-12345678"
	const newID = "20000101T000000Z-87654321"
	if err := ArchiveResult(oldID, []byte(`{"status":"success"}`)); err != nil {
		t.Fatal(err)
	}
	path, err := indexPath("results", oldID)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("archive permissions: %o", info.Mode().Perm())
		}
	}
	old := time.Now().Add(-31 * 24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if data, err := ReadArchivedResult(oldID); err != nil || data != nil {
		t.Fatalf("expired snapshot: %s %v", data, err)
	}
	unrelated := filepath.Join(filepath.Dir(path), ".index-unfinished")
	if err := os.WriteFile(unrelated, []byte("in flight"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(unrelated, old, old); err != nil {
		t.Fatal(err)
	}
	if err := ArchiveResult(newID, []byte(`{"status":"failed"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expired snapshot retained: %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unowned file pruned: %v", err)
	}
	if data, err := ReadArchivedResult(newID); err != nil || string(data) != `{"status":"failed"}` {
		t.Fatalf("new snapshot: %s %v", data, err)
	}
}
