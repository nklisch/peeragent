//go:build unix

package executil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	// The longer terminal grace deliberately lets the inherited-pipe WaitDelay
	// path decide completion first. This is a correctness probe, not a timing
	// assertion: success is the observed event and the completed owned cleanup.
	res, err := (OSRunner{ObserveStdout: func(line string) (bool, error) { return line == "complete", nil }, CompletionGrace: time.Minute}).Run(ctx, os.Args[0], []string{"-test.run=^TestInheritedOutputHelper$"}, t.TempDir(), "")
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
	res, err := (OSRunner{ObserveStdout: func(line string) (bool, error) { return line == "complete", nil }, CompletionGrace: time.Millisecond}).Run(ctx, os.Args[0], []string{"-test.run=^TestObservedHelper$"}, filepath.Dir(os.Args[0]), "")
	if err != nil || !res.CompletionStopped {
		t.Fatalf("owned target: %#v %v", res, err)
	}
	if err := unrelated.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("unrelated process was stopped: %v", err)
	}
}
