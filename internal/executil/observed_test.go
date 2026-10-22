package executil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLineObserverPreservesLargeFragmentedLines(t *testing.T) {
	message := strings.Repeat("long message ", 20000)
	var raw bytes.Buffer
	var lines []string
	writer := &lineObserver{raw: &raw, terminal: make(chan struct{}), failure: make(chan error, 1), observe: func(line string) (bool, error) { lines = append(lines, line); return line == "terminal", nil }}
	parts := []string{message[:500], message[500:] + "\nter", "minal\nlast"}
	for _, part := range parts {
		if _, err := writer.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if raw.String() != message+"\nterminal\nlast" || len(lines) != 2 || lines[0] != message || lines[1] != "terminal" || string(writer.pending) != "last" {
		t.Fatal("fragmented stream changed or truncated")
	}
	select {
	case <-writer.terminal:
	default:
		t.Fatal("terminal not observed")
	}
}

func TestObservedHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_OBSERVED_HELPER") != "1" {
		return
	}
	fmt.Println("complete")
	for {
		time.Sleep(time.Hour)
	}
}

func TestOSRunnerStopsOwnedCompletedProcess(t *testing.T) {
	t.Setenv("PEERAGENT_OBSERVED_HELPER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := (OSRunner{ObserveStdout: func(line string) (bool, error) { return line == "complete", nil }, CompletionGrace: time.Millisecond}).Run(ctx, os.Args[0], []string{"-test.run=^TestObservedHelper$"}, t.TempDir(), "")
	if err != nil || !res.CompletionStopped || res.Stdout != "complete\n" {
		t.Fatalf("completion: %#v %v", res, err)
	}
}

func TestOSRunnerStopsOwnedProcessOnObserverFailure(t *testing.T) {
	t.Setenv("PEERAGENT_OBSERVED_HELPER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	want := errors.New("cannot persist observation")
	_, err := (OSRunner{ObserveStdout: func(string) (bool, error) { return false, want }}).Run(ctx, os.Args[0], []string{"-test.run=^TestObservedHelper$"}, t.TempDir(), "")
	if !errors.Is(err, want) {
		t.Fatalf("observer failure lost: %v", err)
	}
}
