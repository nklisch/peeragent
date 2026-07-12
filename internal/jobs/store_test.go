package jobs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestCreateAndLoadJob(t *testing.T) {
	store := NewStore(t.TempDir())
	spec := ExecSpec{
		Agent:      "codex",
		Access:     "full-access",
		Profile:    "work",
		Effort:     "high",
		Model:      "model-a",
		Resume:     "session-1",
		JSON:       true,
		FullAccess: true,
	}
	job, err := store.Create("/repo", spec, "do work")
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" {
		t.Fatal("expected id")
	}
	if job.Status != StatusRunning {
		t.Fatalf("Status = %q", job.Status)
	}

	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CWD != "/repo" {
		t.Fatalf("CWD = %q", loaded.CWD)
	}
	if loaded.Spec != spec {
		t.Fatalf("Spec = %#v, want %#v", loaded.Spec, spec)
	}
	if loaded.LogPath == "" || loaded.ResultPath == "" {
		t.Fatalf("expected paths: %#v", loaded)
	}
	if loaded.PromptPath == "" || filepath.Base(loaded.PromptPath) != "prompt.txt" {
		t.Fatalf("PromptPath = %q", loaded.PromptPath)
	}
	prompt, err := store.ReadPrompt(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "do work" {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestSaveJob(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	job.Status = StatusComplete
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != StatusComplete {
		t.Fatalf("Status = %q", loaded.Status)
	}
}

func TestPromptRoundTripPreservesRawBytes(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "")
	if err != nil {
		t.Fatal(err)
	}
	prompt := string([]byte{'a', 0, 'b', '\n', 0xff})
	if err := store.WritePrompt(job.ID, prompt); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadPrompt(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != prompt {
		t.Fatalf("prompt bytes = %v, want %v", []byte(got), []byte(prompt))
	}
}

func TestPIDRoundTrip(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 12345); err != nil {
		t.Fatal(err)
	}
	dir, err := store.jobDir(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "pid"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "12345\n" {
		t.Fatalf("pid file = %q, want newline-terminated decimal", raw)
	}
	got, err := store.ReadPID(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != 12345 {
		t.Fatalf("pid = %d, want 12345", got)
	}
	if err := store.RemovePID(job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID after RemovePID err = %v, want not exist", err)
	}
}

func TestReadPIDRejectsInvalidContent(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.jobDir(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := AtomicWriteFile(filepath.Join(dir, "pid"), []byte("not-a-pid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadPID(job.ID); err == nil {
		t.Fatal("expected invalid pid parse error")
	}
}

func TestApplyDetachAttrsSetsidOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only detach attribute")
	}
	cmd := exec.Command("true")
	ApplyDetachAttrs(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	attr := cmd.SysProcAttr
	if !attr.Setsid {
		t.Fatalf("Setsid = false in %#v", attr)
	}
}

func TestProcessGroupHelpersRejectUnsafePID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only process group validation")
	}
	if err := SignalProcessGroup(1, syscall.SIGTERM); err == nil {
		t.Fatal("expected unsafe pid rejection")
	}
	if ProcessGroupExists(1) {
		t.Fatal("unsafe process group should not be reported as existing")
	}
}

func TestSaveGuardedRefusesTerminalStatusOverwrite(t *testing.T) {
	store := NewStore(t.TempDir())
	for _, status := range []Status{StatusCancelled, StatusFailed, StatusComplete} {
		job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
		if err != nil {
			t.Fatal(err)
		}
		job.Status = status
		if err := store.Save(job); err != nil {
			t.Fatal(err)
		}
		job.Status = StatusRunning
		prior, err := store.SaveGuarded(job)
		if err != nil {
			t.Fatal(err)
		}
		if prior != status {
			t.Fatalf("prior = %q, want %q", prior, status)
		}
		loaded, err := store.Load(job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Status != status {
			t.Fatalf("Status = %q, want %q", loaded.Status, status)
		}
	}
}

func TestWithJobLockSerializesConcurrentAccess(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}

	enteredFirst := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- store.WithJobLock(job.ID, func() error {
			close(enteredFirst)
			<-releaseFirst
			return nil
		})
	}()
	<-enteredFirst

	enteredSecond := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- store.WithJobLock(job.ID, func() error {
			close(enteredSecond)
			return nil
		})
	}()

	select {
	case <-enteredSecond:
		close(releaseFirst)
		<-firstDone
		<-secondDone
		t.Fatal("second lock holder entered while first lock was held")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("second lock holder did not acquire after first released")
	}
}

func TestWithJobLockRemovesLockFile(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.jobDir(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, "lock")

	called := false
	if err := store.WithJobLock(job.ID, func() error {
		called = true
		if _, err := os.Stat(lockPath); err != nil {
			t.Fatalf("lock file while held: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("lock callback was not called")
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock file after release err = %v, want not exist", err)
	}
}

func TestAtomicWritesLeaveNoTmpFilesAfterSuccess(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	if err := store.WritePrompt(job.ID, "next"); err != nil {
		t.Fatal(err)
	}
	dir, err := store.jobDir(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("unexpected tmp files: %v", matches)
	}
}

func TestValidateIDAcceptsGeneratedIDsAndRejectsMalformedIDs(t *testing.T) {
	generated, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateID(generated); err != nil {
		t.Fatalf("generated id %q rejected: %v", generated, err)
	}

	for _, id := range []string{
		"",
		"../escape",
		"..",
		".",
		"/absolute/path",
		`a\b`,
		"a/b",
		"20260712T123456Z-deadbeef/child",
		"20260712T123456Z-deadbeeg",
		"20260712T123456Z-DEADBEEF",
		"20260712T12345Z-deadbeef",
		"20260712T123456Z-deadbee",
		"20260712T123456-deadbeef",
		"20260712T123456Z-deadbeefx",
		"20261340T123456Z-deadbeef",
		"20260230T123456Z-deadbeef",
		"20260712T123456Z-😀😀😀😀",
		"20260712T123456Z-deadbeef\x00",
	} {
		if err := ValidateID(id); !errors.Is(err, ErrInvalidID) {
			t.Errorf("ValidateID(%q) = %v, want ErrInvalidID", id, err)
		}
	}
}

func TestStoreRejectsInvalidJobIDsBeforeFilesystemAccess(t *testing.T) {
	root := filepath.Join(t.TempDir(), "jobs")
	store := Store{Root: root}
	invalidIDs := []string{"../escape", "..", ".", "/absolute/path", "a/b", `a\b`, "not-a-job", "20260712T123456Z-deadbeef\x00"}
	for _, id := range invalidIDs {
		t.Run("id", func(t *testing.T) {
			if _, err := store.jobDir(id); !errors.Is(err, ErrInvalidID) {
				t.Fatalf("jobDir(%q) = %v, want ErrInvalidID", id, err)
			}
			operations := []func() error{
				func() error { _, err := store.Load(id); return err },
				func() error { return store.Save(Job{ID: id}) },
				func() error { _, err := store.SaveGuarded(Job{ID: id}); return err },
				func() error { return store.WritePrompt(id, "prompt") },
				func() error { _, err := store.ReadPrompt(id); return err },
				func() error { return store.WritePID(id, 123) },
				func() error { _, err := store.ReadPID(id); return err },
				func() error { return store.RemovePID(id) },
				func() error {
					return store.WithJobLock(id, func() error { t.Fatal("invalid id reached lock callback"); return nil })
				},
			}
			for _, operation := range operations {
				if err := operation(); !errors.Is(err, ErrInvalidID) {
					t.Errorf("operation error = %v, want ErrInvalidID", err)
				}
			}
			if _, err := os.Stat(root); !os.IsNotExist(err) {
				t.Fatalf("invalid id created or probed store root: stat error = %v", err)
			}
		})
	}
}

func TestAtomicSaveFailureLeavesExistingJobJSON(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.jobDir(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(dir, "job.json")
	before, err := os.ReadFile(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := jobPath + ".tmp"
	if err := os.Mkdir(tmpPath, 0o755); err != nil {
		t.Fatal(err)
	}
	job.Status = StatusComplete
	if err := store.Save(job); err == nil {
		t.Fatal("expected save failure")
	}
	after, err := os.ReadFile(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("job.json changed after failed save:\n%s", after)
	}
}
