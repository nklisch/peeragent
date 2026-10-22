package input

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestMultipleJobReferences(t *testing.T) {
	const other = "20000101T000000Z-12345678"
	dir := t.TempDir()
	for _, flag := range []string{"--status", "--wait-any"} {
		req, err := Parse([]string{flag, validJobID, other + "@" + dir, "--cwd", dir}, nil, fixedCWD)
		if err != nil {
			t.Fatal(err)
		}
		refs := req.StatusJobs
		if flag == "--wait-any" {
			refs = req.WaitAnyJobs
		}
		want := []JobReference{{JobID: validJobID, CWD: dir}, {JobID: other, CWD: dir}}
		if !reflect.DeepEqual(refs, want) {
			t.Fatalf("%s references = %#v", flag, refs)
		}
	}
	req, err := Parse([]string{"--wait-any", validJobID + "@relative@directory", other}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	absolute, _ := filepath.Abs("relative@directory")
	if req.WaitAnyJobs[0].CWD != absolute || req.WaitAnyJobs[1].CWD != "" {
		t.Fatalf("relative and indexed references: %#v", req.WaitAnyJobs)
	}
}

func TestWaitAnyRejectsMalformedAndCombinedControls(t *testing.T) {
	for _, args := range [][]string{
		{"--wait-any"}, {"--wait-any", ""}, {"--wait-any", "bad-id"},
		{"--wait-any", validJobID + "@"}, {"--wait-any", validJobID, "--status", validJobID},
		{"--wait-any", validJobID, "--wait", validJobID}, {"--wait-any", validJobID, "--async"},
		{"--wait-any", validJobID, "--retry-capacity", "1"},
	} {
		if _, err := Parse(args, nil, fixedCWD); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}
