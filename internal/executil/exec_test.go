package executil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestOSRunnerNativeArguments(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PEERAGENT_ARG_HELPER", "1")
	args := []string{"", "task with spaces", `say "OK" & | ( ) ^ % !`, `C:\path with spaces\`, "multi\nline\r\ntext", "unicode: café 日本語"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := (OSRunner{}).Run(ctx, self, append([]string{"-test.run=^TestBatchArgHelper$", "--"}, args...), t.TempDir())
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
