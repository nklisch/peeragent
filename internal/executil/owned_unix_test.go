//go:build linux || darwin

package executil

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func TestOwnedGroupNeverSignalsReusedNumber(t *testing.T) {
	// Simulate an exited but unreaped leader, followed by numeric reuse at reap.
	// No real unrelated process is created or signalled by this test.
	const number = 12345
	reused, unrelatedAlive, signals := false, true, 0
	group := &ownedGroup{pid: number, signal: func(pid int, sig syscall.Signal) error {
		if pid != -number || sig != syscall.SIGKILL {
			t.Fatal("wrong group signal")
		}
		signals++
		if reused {
			unrelatedAlive = false
		}
		return nil
	}}
	if err := group.stop(); err != nil {
		t.Fatal(err)
	}
	want := errors.New("leader exit status")
	if err := group.reap(func() error {
		reused = true
		// A cancellation may arrive while Wait is releasing the original leader.
		if err := group.stop(); err != nil {
			t.Fatal(err)
		}
		return want
	}); !errors.Is(err, want) {
		t.Fatalf("reap status: %v", err)
	}
	if err := group.stop(); err != nil {
		t.Fatal(err)
	}
	if signals != 1 || !unrelatedAlive {
		t.Fatal("cleanup signalled the reused group number")
	}
}

func TestExitObservationLeavesLeaderWaitable(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	prepareOwnedCommand(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	owned, err := ownProcess(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	defer owned.release()
	if err := owned.waitExit(); err != nil {
		_ = owned.stop()
		_ = owned.reap()
		t.Fatal(err)
	}
	// A real second wait must still collect this exact child's exit. If the
	// observation reaped it, this fails with 'no child processes'.
	if err := owned.reap(); err != nil {
		t.Fatalf("exit observation reaped the leader: %v", err)
	}
}
