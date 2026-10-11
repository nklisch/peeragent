package executil

import "testing"

func TestQuoteBatchArg(t *testing.T) {
	for _, tc := range []struct{ arg, want string }{
		{"", `^^^"^^^"`},
		{"plain", `^^^"plain^^^"`},
		{"a b", `^^^"a^^^ b^^^"`},
		{`"`, `^^^"\^^^"^^^"`},
		{`C:\path\`, `^^^"C:\path\\^^^"`},
		{`\"`, `^^^"\\\^^^"^^^"`},
		{`&|()^%!`, `^^^"^^^&^^^|^^^(^^^)^^^^^^^%^^^!^^^"`},
		{"café 日本語", `^^^"café^^^ 日本語^^^"`},
	} {
		t.Run(tc.arg, func(t *testing.T) {
			got, err := quoteBatchArg(tc.arg)
			if err != nil || got != tc.want {
				t.Fatalf("quoteBatchArg(%q) = %q, %v; want %q", tc.arg, got, err, tc.want)
			}
		})
	}
}

func TestQuoteBatchArgRejectsCommandBoundaries(t *testing.T) {
	for _, arg := range []string{"before\x00after", "before\rafter", "before\nafter", "before\r\nafter"} {
		if _, err := quoteBatchArg(arg); err == nil {
			t.Fatalf("accepted %q", arg)
		}
	}
}
