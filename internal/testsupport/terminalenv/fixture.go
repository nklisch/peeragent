package terminalenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Independent fixture names pin the complete policy, including future-prefix
// cases. Values are synthetic; fake agents never dump the real environment.
func TerminalEnvNames() []string {
	return []string{"HERDR_PANE_ID", "HERDR_SOCKET_PATH", "HERDR_WORKSPACE_ID", "HERDR_FUTURE", "TMUX", "TMUX_PANE", "ZELLIJ", "ZELLIJ_SESSION_NAME", "ZELLIJ_FUTURE", "KITTY_LISTEN_ON", "KITTY_WINDOW_ID", "WEZTERM_UNIX_SOCKET", "WEZTERM_PANE", "STY", "WINDOWID", "ALACRITTY_SOCKET", "ALACRITTY_WINDOW_ID", "KONSOLE_DBUS_SERVICE", "KONSOLE_DBUS_SESSION", "KONSOLE_DBUS_WINDOW"}
}

func TerminalEnvSnapshot() map[string]string {
	snapshot := map[string]string{}
	for _, name := range append(TerminalEnvNames(), "PEERAGENT_ENV_SENTINEL", "PEERAGENT_JOB_ID", "PEERAGENT_JOB_DIR") {
		if value, exists := os.LookupEnv(name); exists {
			snapshot[name] = value
		}
	}
	return snapshot
}

// A copy of the test executable acts as each target; no installed CLI runs.
func WriteTerminalEnvAgentSnapshot() bool {
	if os.Getenv("PEERAGENT_TERMINAL_ENV_HELPER") != "1" {
		return false
	}
	switch strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") {
	case "codex", "claude", "agy", "pi":
		if err := json.NewEncoder(os.Stdout).Encode(TerminalEnvSnapshot()); err != nil {
			panic(err)
		}
		return true
	}
	return false
}
