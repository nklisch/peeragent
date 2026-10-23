package result

import (
	"strings"
	"testing"
)

func TestCapacitySummaryUsesStatusAndRetryHint(t *testing.T) {
	for _, tc := range []struct{ session, want string }{
		{"", "stopped: capacity_exhausted; use --retry-capacity N to retry automatically"},
		{"session", "stopped: capacity_exhausted; resumable with --resume session; use --retry-capacity N to retry automatically"},
	} {
		summary := CapacitySummary(tc.session)
		if summary != tc.want {
			t.Fatalf("summary %q, want %q", summary, tc.want)
		}
		res := Result{Status: StatusCapacity, Summary: summary}
		data, err := FormatJSON(res)
		if err != nil || !strings.Contains(string(data), `"status":"capacity_exhausted"`) {
			t.Fatalf("JSON %s %v", data, err)
		}
		if text := FormatText(res); !strings.Contains(text, "Summary:\n"+tc.want) {
			t.Fatal(text)
		}
	}
}
