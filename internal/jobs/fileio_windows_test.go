//go:build windows

package jobs

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestStoreLoadRetriesSharingViolation(t *testing.T) {
	cwd := t.TempDir()
	store := NewStore(cwd)
	job, err := store.Create(cwd, ExecSpec{Agent: "codex"}, "task")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, job.ID, "job.json")
	h := lockFileExclusively(t, path)
	released := make(chan struct{})
	go func() {
		time.Sleep(60 * time.Millisecond)
		_ = syscall.CloseHandle(h)
		close(released)
	}()
	defer func() { <-released }()
	loaded, err := store.Load(job.ID)
	if err != nil || loaded.ID != job.ID {
		t.Fatalf("load after sharing lock = %#v, %v", loaded, err)
	}
}

func TestAtomicWriteFileRetriesSharingViolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := lockFileExclusively(t, path)
	released := make(chan struct{})
	go func() {
		time.Sleep(60 * time.Millisecond)
		_ = syscall.CloseHandle(h)
		close(released)
	}()
	defer func() { <-released }()
	if err := AtomicWriteFile(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "new" {
		t.Fatalf("content after sharing lock = %q, %v", content, err)
	}
}

func lockFileExclusively(t *testing.T, path string) syscall.Handle {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
