package jobs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadProgressUsesBoundedTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	// No valid JSON exists in the large historical prefix. A partial trailing
	// write must not hide the latest complete record.
	if err := os.WriteFile(path, []byte(strings.Repeat("old data\n", progressTailBytes)), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	want := Progress{At: &now, Event: "item.completed", LastMessage: "Still building"}
	if err := AppendProgress(path, want); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(`{"at":`)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadProgress(path)
	if err != nil || got.At == nil || !got.At.Equal(now) || got.LastMessage != want.LastMessage {
		t.Fatalf("tail: %#v %v", got, err)
	}
}

func TestProcessExists(t *testing.T) {
	if !ProcessExists(os.Getpid()) {
		t.Fatal("current process reported gone")
	}
	if ProcessExists(0) || ProcessExists(-1) {
		t.Fatal("invalid pid reported alive")
	}
}
