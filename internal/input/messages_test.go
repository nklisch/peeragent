package input

import (
	"os"
	"path/filepath"
	"strings"
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

func TestMessagingPromptFileMatchesLaunchJoining(t *testing.T) {
	root := t.TempDir()
	file := writeTempPrompt(t, " \nfrom file\nwith another line\n ")
	const id = "20261022T000000Z-12345678"
	for _, command := range []string{"tell", "say"} {
		for _, prefix := range [][]string{nil, {"  positional", "text  "}} {
			args := []string{command}
			if command == "tell" {
				args = append(args, id, "--cwd", root)
			}
			args = append(args, prefix...)
			args = append(args, "--prompt-file", file)
			got, err := Parse(args, nil, nil)
			launch, launchErr := Parse(append(append([]string{}, prefix...), "--prompt-file", file), nil, fixedCWD)
			if err != nil || launchErr != nil || got.TaskText != launch.TaskText {
				t.Fatalf("%v: got %q (%v), launch %q (%v)", args, got.TaskText, err, launch.TaskText, launchErr)
			}
		}
	}
	for _, args := range [][]string{
		{"say", "--prompt-file"}, {"say", "--prompt-file", ""}, {"tell", id, "--prompt-file", "--cwd"},
		{"inbox", "--prompt-file", file}, {"say", "--prompt-file", filepath.Join(root, "missing")},
	} {
		if _, err := Parse(args, nil, nil); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := Parse([]string{"say", "--prompt-file", root}, nil, nil); err == nil || !strings.Contains(err.Error(), "read prompt file:") {
		t.Fatalf("directory read: %v", err)
	}
	if err := os.WriteFile(file, []byte(" \n\t"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse([]string{"tell", id, "--cwd", root, "--prompt-file", file}, nil, nil); err == nil {
		t.Fatal("tell accepted an empty message file")
	}
	got, err := Parse([]string{"say", "--", "--prompt-file", file}, nil, nil)
	if err != nil || got.TaskText != "--prompt-file "+file {
		t.Fatalf("literal flag: %#v %v", got, err)
	}
}
