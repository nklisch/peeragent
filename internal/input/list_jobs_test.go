package input

import (
	"strings"
	"testing"
)

func TestJobsAndVersionControlParsing(t *testing.T) {
	cwd := t.TempDir()
	for _, args := range [][]string{{"--jobs"}, {"--jobs", "--cwd", cwd, "--all", "--text"}, {"--version"}} {
		req, err := Parse(args, strings.NewReader("must not become task text"), func() (string, error) { return cwd, nil })
		if err != nil || req.TaskText != "" || req.Async || req.Version != (args[0] == "--version") || req.Jobs != (args[0] == "--jobs") {
			t.Fatalf("%v: %#v %v", args, req, err)
		}
		if req.Jobs && req.CWD != cwd {
			t.Fatalf("directory: %#v", req)
		}
		if len(args) > 1 && (!req.All || req.JSON) {
			t.Fatalf("modifiers: %#v", req)
		}
	}
	for _, args := range [][]string{
		{"--jobs", "--bogus"}, {"--version", "--bogus"}, {"--all"}, {"--jobs", "--async"},
		{"--jobs", "--status", "20000101T000000Z-00000000"}, {"--jobs", "task"}, {"--version", "task"},
		{"--jobs", "--version"}, {"--version", "--resume", "session"}, {"--jobs", "--retry-capacity", "1"},
	} {
		if req, err := Parse(args, nil, func() (string, error) { return cwd, nil }); err == nil {
			t.Fatalf("accepted %v: %#v", args, req)
		}
	}
}
