package executil

import (
	"runtime"
	"strings"
)

// targetEnvironment removes inherited addresses for the launching terminal, so
// a delegated agent cannot use them to inject input that looks like the user.
// --keep-terminal-env is the deliberate override. Authentication, PATH and
// ordinary shell settings are preserved. See docs/TERMINAL_ENV.md for policy.
func targetEnvironment(env []string, keep bool) []string {
	if keep {
		return env
	}
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, _ := strings.Cut(entry, "=")
		// Windows environment names are case-insensitive; Unix names are exact.
		if runtime.GOOS == "windows" {
			name = strings.ToUpper(name)
		}
		if !terminalControlVariable(name) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func terminalControlVariable(name string) bool {
	if strings.HasPrefix(name, "HERDR_") || strings.HasPrefix(name, "ZELLIJ_") || strings.HasPrefix(name, "KONSOLE_DBUS_") {
		return true
	}
	switch name {
	case "TMUX", "TMUX_PANE", "ZELLIJ", "KITTY_LISTEN_ON", "KITTY_WINDOW_ID",
		"WEZTERM_UNIX_SOCKET", "WEZTERM_PANE", "STY", "WINDOWID",
		"ALACRITTY_SOCKET", "ALACRITTY_WINDOW_ID":
		return true
	}
	return false
}
