//go:build windows

package jobs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestWindowsLockHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_LOCK_HELPER") != "1" {
		return
	}
	store := NewStore(os.Getenv("PEERAGENT_LOCK_ROOT"))
	err := store.WithJobLock(os.Getenv("PEERAGENT_LOCK_ID"), func() error {
		if err := os.WriteFile(os.Getenv("PEERAGENT_LOCK_READY"), []byte("ready"), 0o644); err != nil {
			return err
		}
		time.Sleep(30 * time.Second)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWindowsLockRecoversAfterHolderIsKilled(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	job, err := store.Create(root, ExecSpec{Agent: "codex"}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(root, "lock-ready")
	cmd := exec.Command(os.Args[0], "-test.run=^TestWindowsLockHelper$")
	cmd.Env = append(os.Environ(),
		"PEERAGENT_LOCK_HELPER=1",
		"PEERAGENT_LOCK_ROOT="+root,
		"PEERAGENT_LOCK_ID="+job.ID,
		"PEERAGENT_LOCK_READY="+ready,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper did not acquire job lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := store.WithJobLock(job.ID, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, job.ID, "lock")); !os.IsNotExist(err) {
		t.Fatalf("diagnostic lock marker remains: %v", err)
	}
}
