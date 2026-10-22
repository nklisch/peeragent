# Target terminal environment

Every target agent starts without inherited terminal-control addresses by
default. This prevents a delegated agent from using those inherited values to
inject input into the launching terminal that appears to come from the person.
The shared process runner applies the same policy to Codex, Claude, Gemini and
GLM, including blocking, async and resumed launches and capacity retries.

Use `--keep-terminal-env` only when the target deliberately needs the launching
terminal's control environment. The override is recorded as
`spec.keep_terminal_env: true` in job.json and
`metadata.keep_terminal_env: true` in launch/results and single-job status.
Multi-job status includes `keep_terminal_env: true`; its text note reports
“terminal environment retained”. Text results also show the override.
A new resume strips terminal-control variables unless the flag is passed again;
capacity retries retain the original job's setting.

This is the complete list removed by the default policy. Prefixes include any
future variable with that prefix; exact names do not match unrelated prefixes.
Names are case-sensitive on Unix and case-insensitive on Windows. The shared
filter also runs on Windows, even when none of these variables is present.

| Terminal/control surface | Removed variables |
| --- | --- |
| Herdr | `HERDR_*` |
| tmux | `TMUX`, `TMUX_PANE` |
| Zellij | `ZELLIJ`, `ZELLIJ_*` |
| Kitty | `KITTY_LISTEN_ON`, `KITTY_WINDOW_ID` |
| WezTerm | `WEZTERM_UNIX_SOCKET`, `WEZTERM_PANE` |
| GNU Screen | `STY` |
| Terminal window identity | `WINDOWID` |
| Alacritty | `ALACRITTY_SOCKET`, `ALACRITTY_WINDOW_ID` |
| Konsole | `KONSOLE_DBUS_*` |

The additional Kitty window identity is documented in [Kitty remote control](https://sw.kovidgoyal.net/kitty/remote-control/).
Alacritty's [IPC documentation](https://alacritty.org/cmd-alacritty-msg.html)
describes socket control and the inherited window identity. [Konsole scripting](https://docs.kde.org/stable_kf6/en/konsole/konsole/scripting.html)
documents its inherited service/session addresses.

Authentication, PATH, ordinary shell settings and terminal presentation
settings remain available. Filtering does not mutate the launching process's
environment, and values are never written to job metadata. It removes inherited
control addresses; it does not sandbox access to the desktop, sockets or other
processes. The async wrapper remains local orchestration; filtering occurs when
it starts each target process. Windows batch shims retain their private argument
variables while the terminal-control entries are removed.
