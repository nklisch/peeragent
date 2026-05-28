package jobs

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
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
	if job.Status != "running" {
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
	job.Status = "complete"
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "complete" {
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
	raw, err := os.ReadFile(filepath.Join(store.jobDir(job.ID), "pid"))
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
	if err := AtomicWriteFile(filepath.Join(store.jobDir(job.ID), "pid"), []byte("not-a-pid\n"), 0o644); err != nil {
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
	for _, status := range []string{"cancelled", "failed", "complete"} {
		job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
		if err != nil {
			t.Fatal(err)
		}
		job.Status = status
		if err := store.Save(job); err != nil {
			t.Fatal(err)
		}
		job.Status = "running"
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
	matches, err := filepath.Glob(filepath.Join(store.jobDir(job.ID), "*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("unexpected tmp files: %v", matches)
	}
}

func TestAtomicSaveFailureLeavesExistingJobJSON(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("/repo", ExecSpec{Agent: "codex", Access: "default", JSON: true}, "do work")
	if err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(store.jobDir(job.ID), "job.json")
	before, err := os.ReadFile(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := jobPath + ".tmp"
	if err := os.Mkdir(tmpPath, 0o755); err != nil {
		t.Fatal(err)
	}
	job.Status = "complete"
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
