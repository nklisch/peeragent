package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandLikeTasksRequireExplicitInput(t *testing.T) {
	const id = "20261027T161759Z-654b65f7"
	cwd := t.TempDir()
	getwd := func() (string, error) { return cwd, nil }
	for _, args := range [][]string{{"outbox", id}, {"outbox", id, "--", "extra"}, {"outbox " + id}, {id}, {"--cwd", cwd, "status", id}, {"foo-bar", id, "extra"}} {
		if _, err := Parse(args, nil, getwd); err == nil || !strings.Contains(err.Error(), "unknown command") || !strings.Contains(err.Error(), "messages") || !strings.Contains(err.Error(), "use --") {
			t.Fatalf("%q: %v", args, err)
		}
	}
	if _, err := Parse(nil, strings.NewReader("outbox "+id), getwd); err == nil || !strings.Contains(err.Error(), "unknown command outbox") {
		t.Fatalf("stdin typo: %v", err)
	}
	file := filepath.Join(cwd, "prompt")
	if err := os.WriteFile(file, []byte("outbox "+id), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"--", "outbox", id}, {"--", id}, {"--prompt-file", file}, {"Review", id}, {"review the job", id}, {"outbox", "ordinary words"}} {
		if _, err := Parse(args, nil, getwd); err != nil {
			t.Fatalf("legitimate %q: %v", args, err)
		}
	}
}

func TestMessagesParsing(t *testing.T) {
	for _, format := range []string{"--json", "--text"} {
		req, err := Parse([]string{"messages", "20261027T161759Z-654b65f7", "--cwd", t.TempDir(), format}, nil, nil)
		if err != nil || req.MessageCommand != "messages" || req.JSON != (format == "--json") {
			t.Fatalf("%#v %v", req, err)
		}
	}
	for _, args := range [][]string{{"messages"}, {"messages", "20261027T161759Z-654b65f7", "unexpected"}, {"messages", "20261027T161759Z-654b65f7", "--prompt-file", "file"}} {
		if _, err := Parse(args, nil, func() (string, error) { return t.TempDir(), nil }); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}
