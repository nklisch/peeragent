package executil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOSRunnerNativeStdin(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PEERAGENT_STDIN_HELPER", "1")
	input := strings.Repeat("quotes: \" & | % !\r\nUnicode: café 日本語\n", 2000)
	for _, observed := range []bool{false, true} {
		name := "ordinary"
		runner := OSRunner{}
		if observed {
			name = "observed"
			runner.ObserveStdout = func(string) (CompletionObservation, error) { return CompletionUnchanged, nil }
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			got, err := runner.Run(ctx, self, []string{"-test.run=^TestStdinHelper$"}, t.TempDir(), input)
			if err != nil || got.ExitCode != 0 {
				t.Fatalf("Run = %+v, %v", got, err)
			}
			if got.Stdout != input {
				t.Fatalf("stdin changed: got %d bytes, want %d", len(got.Stdout), len(input))
			}
		})
	}

}

func TestStdinHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_STDIN_HELPER") != "1" {
		return
	}
	if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestOSRunnerNativeArguments(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PEERAGENT_ARG_HELPER", "1")
	args := []string{"", "task with spaces", `say "OK" & | ( ) ^ % !`, `C:\path with spaces\`, "multi\nline\r\ntext", "unicode: café 日本語"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := (OSRunner{}).Run(ctx, self, append([]string{"-test.run=^TestBatchArgHelper$", "--"}, args...), t.TempDir(), "")
	if err != nil || got.ExitCode != 0 {
		t.Fatalf("Run = %+v, %v", got, err)
	}
	var forwarded []string
	if err := json.Unmarshal([]byte(got.Stdout), &forwarded); err != nil || !reflect.DeepEqual(forwarded, args) {
		t.Fatalf("argv = %q, want %q (%v)", forwarded, args, err)
	}
}

func TestBatchArgHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_ARG_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			encoded, err := json.Marshal(os.Args[i+1:])
			if err != nil {
				t.Fatal(err)
			}
			fmt.Print(string(encoded))
			os.Exit(0)
		}
	}
	t.Fatal("missing argument marker")
}

func TestEarlyStdinCloseHelper(t *testing.T) {
	if os.Getenv("PEERAGENT_EARLY_STDIN_CLOSE") != "1" {
		return
	}
	_ = os.Stdin.Close()
	os.Exit(0)
}

func TestOSRunnerObservedEarlyStdinClose(t *testing.T) {
	t.Setenv("PEERAGENT_EARLY_STDIN_CLOSE", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := (OSRunner{ObserveStdout: func(string) (CompletionObservation, error) { return CompletionUnchanged, nil }}).Run(ctx, os.Args[0], []string{"-test.run=^TestEarlyStdinCloseHelper$"}, t.TempDir(), strings.Repeat("prompt", 100000))
	if err != nil || got.ExitCode != 0 {
		t.Fatalf("early stdin close changed target success: exit=%d err=%v", got.ExitCode, err)
	}
}
