package executil

import (
	"strings"
	"testing"
)

func TestQuoteBatchArg(t *testing.T) {
	for _, tc := range []struct{ arg, want string }{
		{"", `""`},
		{"plain", `"plain"`},
		{"a b", `"a b"`},
		{`"`, `"\""`},
		{`C:\path\`, `"C:\path\\"`},
		{`\"`, `"\\\""`},
		{`&|()^%!`, `"&|()^%!"`},
		{"café 日本語", `"café 日本語"`},
		{"multi\nline\r\ntext", "\"multi\nline\r\ntext\""},
	} {
		t.Run(tc.arg, func(t *testing.T) {
			got, err := quoteBatchArg(tc.arg)
			if err != nil || got != tc.want {
				t.Fatalf("quoteBatchArg(%q) = %q, %v; want %q", tc.arg, got, err, tc.want)
			}
		})
	}
}

func TestQuoteBatchArgRejectsNUL(t *testing.T) {
	if _, err := quoteBatchArg("before\x00after"); err == nil {
		t.Fatal("accepted NUL")
	}
}

func TestQuoteBatchArgLengthLimit(t *testing.T) {
	if _, err := quoteBatchArg(strings.Repeat("x", 8191)); err == nil {
		t.Fatal("accepted a value that cmd.exe would silently drop")
	}
	if _, err := quoteBatchArg(strings.Repeat("😀", 4095)); err == nil {
		t.Fatal("length limit did not count UTF-16 units and enclosing quotes")
	}
	if _, err := quoteBatchArg(strings.Repeat("x", 8189)); err != nil {
		t.Fatalf("rejected value at cmd.exe's limit: %v", err)
	}
}
