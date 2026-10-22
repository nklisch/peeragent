package app

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"unsafe"

	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestExitedUnreapedWorkerDoesNotBlockResume(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cmd := exec.Command(os.Args[0], "-test.run=^TestExitedProcessHelper$")
	cmd.Env = append(os.Environ(), "PEERAGENT_EXIT_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Wait()
	// Confirm exit without collecting it: kill(pid, 0) still sees this zombie.
	var info [16]uint64
	for {
		_, _, errno := syscall.Syscall6(syscall.SYS_WAITID, 1, uintptr(cmd.Process.Pid), uintptr(unsafe.Pointer(&info)), syscall.WEXITED|syscall.WNOWAIT, 0, 0)
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 {
			t.Fatal(errno)
		}
		break
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); err != nil {
		t.Fatalf("premise: unreaped child is absent: %v", err)
	}
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, jobs.ExecSpec{Agent: "codex", Resume: "dead-session"}, "work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{Launcher: &fakeLauncher{}, Executable: func() (string, error) { return "unused", nil }})
	res, err := service.Launch(context.Background(), input.Delegation{CWD: cwd, Agent: "codex", Resume: "dead-session", TaskText: "continue"})
	if err != nil || res.Status != result.StatusRunning {
		t.Fatalf("exited child refused resume: %#v %v", res, err)
	}
}
