package input

import (
	"os"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	req, err := Parse([]string{"implement", "the", "thing"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "implement the thing" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
	if req.CWD != "/repo" {
		t.Fatalf("CWD = %q", req.CWD)
	}
}

func TestParsePromptFile(t *testing.T) {
	file := writeTempPrompt(t, "from file\n")
	req, err := Parse([]string{"--prompt-file", file}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "from file" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
}

func TestParseStdin(t *testing.T) {
	req, err := Parse(nil, strings.NewReader("from stdin\n"), fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskText != "from stdin" {
		t.Fatalf("TaskText = %q", req.TaskText)
	}
}

func TestParseCombinesInputs(t *testing.T) {
	file := writeTempPrompt(t, "from file")
	req, err := Parse([]string{"prefix", "--prompt-file", file}, strings.NewReader("from stdin"), fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	want := "prefix\n\nfrom file\n\nfrom stdin"
	if req.TaskText != want {
		t.Fatalf("TaskText = %q, want %q", req.TaskText, want)
	}
}

func TestParseCWDOverride(t *testing.T) {
	req, err := Parse([]string{"--cwd", "/other", "task"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.CWD != "/other" {
		t.Fatalf("CWD = %q", req.CWD)
	}
}

func TestParseRequiresTaskText(t *testing.T) {
	_, err := Parse(nil, nil, fixedCWD)
	if err == nil {
		t.Fatal("expected error")
	}
}

func fixedCWD() (string, error) {
	return "/repo", nil
}

func writeTempPrompt(t *testing.T, content string) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "prompt-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}
