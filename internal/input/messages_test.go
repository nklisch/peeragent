package input

import (
	"path/filepath"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
)

func TestMessagingCommandsUseIndexAndExplicitDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PEERAGENT_STATE_DIR", filepath.Join(root, "index"))
	id := "20261022T000000Z-12345678"
	if err := jobs.IndexJob(id, root); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"tell", id, "answer"}, {"tell", id, "--cwd", root, "answer"}, {"tell", id, "answer", "--cwd", root}} {
		req, err := Parse(args, nil, func() (string, error) { t.Fatal("index should resolve"); return "", nil })
		if err != nil || req.MessageCommand != "tell" || req.TaskText != "answer" || req.CWD != root {
			t.Fatalf("%v: %#v %v", args, req, err)
		}
	}
	for _, args := range [][]string{{"say", "--", "--literal"}, {"inbox"}, {"say", ""}} {
		req, err := Parse(args, nil, nil)
		if err != nil || req.MessageCommand != args[0] {
			t.Fatalf("%v %#v %v", args, req, err)
		}
	}
	for _, args := range [][]string{{"tell"}, {"tell", "bad", "answer"}, {"tell", id}, {"tell", id, "--cwd"}, {"inbox", "extra"}, {"say", "--bogus"}} {
		if _, err := Parse(args, nil, nil); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	req, err := Parse([]string{"--no-preamble", "task"}, nil, func() (string, error) { return root, nil })
	if err != nil || !req.NoPreamble {
		t.Fatalf("no-preamble %#v %v", req, err)
	}
}
