package jobs

import (
	"os"
	"runtime"
	"sync"
	"testing"
)

func TestIndexConcurrentWritesAndPermissions(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := IndexSession("codex", "same-session", cwd); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := LookupSession("codex", "same-session")
	if err != nil || got != cwd {
		t.Fatalf("lookup: %q %v", got, err)
	}
	if got, err := LookupSession("claude", "same-session"); err != nil || got != "" {
		t.Fatalf("agent namespace: %q %v", got, err)
	}
	if runtime.GOOS != "windows" {
		path, _ := indexPath("sessions", "codex\x00same-session")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("index mode: %o", info.Mode().Perm())
		}
	}
}
