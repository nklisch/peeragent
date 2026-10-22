package executil

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type stdinHolderState struct {
	Leader   int
	Worker   int
	Capacity int
}

func TestStdinHolderLauncherHelper(t *testing.T) {
	root := os.Getenv("PEERAGENT_STDIN_HOLDER")
	if root == "" {
		return
	}
	child := exec.Command(os.Args[0], "-test.run=^TestStdinHolderWorkerHelper$")
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		panic(err)
	}
	state, _ := json.Marshal(stdinHolderState{Leader: os.Getpid(), Worker: child.Process.Pid})
	if err := os.WriteFile(filepath.Join(root, "launcher.json"), state, 0o600); err != nil {
		panic(err)
	}
	os.Exit(0)
}

func TestStdinHolderWorkerHelper(t *testing.T) {
	root := os.Getenv("PEERAGENT_STDIN_HOLDER")
	if root == "" {
		return
	}
	_ = os.Stdout.Close()
	_ = os.Stderr.Close()
	capacity, _, errno := syscall.Syscall(syscall.SYS_FCNTL, os.Stdin.Fd(), syscall.F_GETPIPE_SZ, 0)
	if errno != 0 {
		panic(errno)
	}
	state, _ := json.Marshal(stdinHolderState{Worker: os.Getpid(), Capacity: int(capacity)})
	if err := os.WriteFile(filepath.Join(root, "worker.json"), state, 0o600); err != nil {
		panic(err)
	}
	// Retain stdin without consuming it. This release file is only a fallback
	// for test cleanup: the assertion requires group cancellation to stop us.
	for {
		if _, err := os.Stat(filepath.Join(root, "release")); err == nil {
			os.Exit(0)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCancelBlockedStdinAfterLauncherExit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PEERAGENT_STDIN_HOLDER", root)
	prompt := strings.Repeat("p", 8<<20)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	type outcome struct {
		result Result
		err    error
	}
	done := make(chan outcome, 1)
	outputsDrained := make(chan struct{})
	resume := make(chan struct{})
	releaseRunner := sync.OnceFunc(func() { close(resume) })
	finished := false
	t.Cleanup(func() {
		// Only files and processes created by this fixture are affected. A failing
		// old runner can finish its blocked copy once the worker releases stdin.
		_ = os.WriteFile(filepath.Join(root, "release"), nil, 0o600)
		cancel()
		releaseRunner()
		if !finished {
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Error("stdin fixture cleanup did not return")
			}
		}
	})
	go func() {
		res, err := (OSRunner{
			ObserveStdout:    func(string) (CompletionObservation, error) { return CompletionUnchanged, nil },
			afterOutputDrain: func() { close(outputsDrained); <-resume },
		}).Run(ctx, os.Args[0], []string{"-test.run=^TestStdinHolderLauncherHelper$"}, root, prompt)
		done <- outcome{res, err}
	}()
	readState := func(name string) (stdinHolderState, bool) {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		var state stdinHolderState
		ready := err == nil && json.Unmarshal(content, &state) == nil && state.Worker > 0
		return state, ready
	}
	var launcher, worker stdinHolderState
	for {
		var launcherReady, workerReady bool
		launcher, launcherReady = readState("launcher.json")
		worker, workerReady = readState("worker.json")
		if launcherReady && workerReady {
			break
		}
		select {
		case early := <-done:
			finished = true
			t.Fatalf("runner returned before blocked-input readiness: %+v", early)
		case <-ctx.Done():
			t.Fatal("stdin holder did not become ready")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if launcher.Worker != worker.Worker || launcher.Leader <= 0 || worker.Capacity <= 0 || len(prompt) <= worker.Capacity {
		t.Fatalf("blocked-input premise missing: prompt=%d launcher=%+v worker=%+v", len(prompt), launcher, worker)
	}
	t.Logf("measured prompt bytes=%d, stdin pipe capacity=%d", len(prompt), worker.Capacity)
	// Pause the actual runner after it consumed leader exit and both output
	// EOFs, before deciding whether to reap. Cancellation cannot race ahead of
	// the vulnerable decision and accidentally let the old implementation pass.
	select {
	case <-outputsDrained:
	case early := <-done:
		finished = true
		t.Fatalf("runner returned before output drain: %+v", early)
	case <-ctx.Done():
		t.Fatal("runner did not observe output drain and leader exit")
	}
	// The leader must remain waitable while input is pending.
	if err := waitWithoutReaping(launcher.Leader); err != nil {
		t.Fatalf("leader ownership lost before input completion: %v", err)
	}
	cancel()
	releaseRunner()
	select {
	case got := <-done:
		finished = true
		if !errors.Is(got.err, context.Canceled) || got.result.CompletionStopped {
			t.Fatalf("cancel outcome: %+v", got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("blocked stdin prevented cancellation from returning")
	}
	// Wait collected the exact leader, and the group signal stopped its worker.
	if _, err := syscall.Wait4(launcher.Leader, nil, syscall.WNOHANG, nil); err != syscall.ECHILD {
		t.Fatalf("leader was not reaped: %v", err)
	}
	deadline := time.After(10 * time.Second)
	for {
		stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(worker.Worker), "stat"))
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		fields := strings.Fields(string(stat)[strings.LastIndexByte(string(stat), ')')+1:])
		if len(fields) > 0 && (fields[0] == "Z" || fields[0] == "X") {
			break
		}
		select {
		case <-deadline:
			t.Fatal("stdin holder survived owned-tree cancellation")
		case <-time.After(10 * time.Millisecond):
		}
	}
}
