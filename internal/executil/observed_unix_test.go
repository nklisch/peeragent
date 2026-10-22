//go:build unix

package executil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestInheritedOutputHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_INHERITED_HELPER") != "1" {
		return
	}
	// This owned descendant deliberately keeps the output descriptors open
	// after its direct parent exits. Its lifetime is the runner's process group.
	cmd := exec.Command(os.Args[0], "-test.run=^TestObservedHelper$")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	fmt.Println("complete")
	os.Exit(0)
}

func TestOSRunnerCompletesWithInheritedOutputPipes(t *testing.T) {
	t.Setenv("PEERAGENT_INHERITED_HELPER", "1")
	t.Setenv("PEERAGENT_OBSERVED_HELPER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// The leader exits while its descendant retains output. Completion must
	// stop that owned descendant before the leader is reaped.
	res, err := (OSRunner{ObserveStdout: func(line string) (CompletionObservation, error) {
		if line == "complete" {
			return CompletionPending, nil
		}
		return CompletionUnchanged, nil
	}, CompletionGrace: time.Millisecond}).Run(ctx, os.Args[0], []string{"-test.run=^TestInheritedOutputHelper$"}, t.TempDir(), "")
	if err != nil || !res.CompletionStopped {
		t.Fatalf("inherited output: %#v %v", res, err)
	}
}

func TestCompletionDoesNotStopUnrelatedProcess(t *testing.T) {
	t.Setenv("PEERAGENT_OBSERVED_HELPER", "1")
	unrelated := exec.Command(os.Args[0], "-test.run=^TestObservedHelper$")
	if err := unrelated.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unrelated.Process.Kill(); _ = unrelated.Wait() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := (OSRunner{ObserveStdout: func(line string) (CompletionObservation, error) {
		if line == "complete" {
			return CompletionPending, nil
		}
		return CompletionUnchanged, nil
	}, CompletionGrace: time.Millisecond}).Run(ctx, os.Args[0], []string{"-test.run=^TestObservedHelper$"}, filepath.Dir(os.Args[0]), "")
	if err != nil || !res.CompletionStopped {
		t.Fatalf("owned target: %#v %v", res, err)
	}
	if err := unrelated.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("unrelated process was stopped: %v", err)
	}
}

func TestFiniteOutputWorkerHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_FINITE_WORKER") != "1" {
		return
	}
	fmt.Println("worker started")
	// Deliberately outlive the old unconditional pipe timeout. Success depends
	// on collecting the full eventual output, not on an elapsed-time assertion.
	time.Sleep(completionGrace + time.Second)
	fmt.Print(strings.Repeat("worker final output — ", 12000))
	os.Exit(0)
}

func TestFiniteLauncherHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_FINITE_WORKER") != "1" {
		return
	}
	worker := exec.Command(os.Args[0], "-test.run=^TestFiniteOutputWorkerHelper$")
	worker.Stdout, worker.Stderr = os.Stdout, os.Stderr
	if err := worker.Start(); err != nil {
		panic(err)
	}
	os.Exit(0)
}

func TestLauncherExitDoesNotTimeOutActiveWorkerPipes(t *testing.T) {
	t.Setenv("PEERAGENT_FINITE_WORKER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := (OSRunner{ObserveStdout: func(string) (CompletionObservation, error) { return CompletionUnchanged, nil }}).Run(ctx, os.Args[0], []string{"-test.run=^TestFiniteLauncherHelper$"}, t.TempDir(), "")
	want := "worker started\n" + strings.Repeat("worker final output — ", 12000)
	if err != nil || res.ExitCode != 0 || res.CompletionStopped || res.Stdout != want {
		t.Fatalf("active worker drain: exit=%d stopped=%t bytes=%d err=%v", res.ExitCode, res.CompletionStopped, len(res.Stdout), err)
	}
}
