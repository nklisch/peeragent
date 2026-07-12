package jobs

import "testing"

func TestIsTerminalStatusUsesKnownPersistedStatesOnly(t *testing.T) {
	for _, tt := range []struct {
		name     string
		status   Status
		terminal bool
	}{
		{name: "running", status: StatusRunning},
		{name: "complete", status: StatusComplete, terminal: true},
		{name: "failed", status: StatusFailed, terminal: true},
		{name: "cancelled", status: StatusCancelled, terminal: true},
		{name: "unknown remains non-terminal", status: Status("future-state")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTerminalStatus(tt.status); got != tt.terminal {
				t.Fatalf("IsTerminalStatus(%q) = %v, want %v", tt.status, got, tt.terminal)
			}
		})
	}
}
