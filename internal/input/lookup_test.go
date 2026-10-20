package input

import (
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
)

func TestIndexedDirectoriesAndExplicitOverride(t *testing.T) {
	t.Setenv("PEERAGENT_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	const id = "20000101T000000Z-12345678"
	if err := jobs.IndexJob(id, cwd); err != nil {
		t.Fatal(err)
	}
	if err := jobs.IndexSession("codex", "session-lookup", cwd); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--status", "--wait", "--result", "--cancel", "--resume"} {
		value := id
		tail := []string{}
		if flag == "--resume" {
			value = "session-lookup"
			tail = []string{"continue"}
		}
		args := append([]string{flag, value}, tail...)
		req, err := Parse(args, nil, fixedCWD)
		if err != nil || req.CWD != cwd {
			t.Fatalf("%s indexed cwd = %q, %v", flag, req.CWD, err)
		}
		args = append(args, "--cwd", "/explicit")
		req, err = Parse(args, nil, fixedCWD)
		if err != nil || req.CWD != absoluteFixturePath(t, "/explicit") {
			t.Fatalf("%s override = %q, %v", flag, req.CWD, err)
		}
	}
}
