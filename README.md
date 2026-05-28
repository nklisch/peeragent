# Peeragent

Peeragent is a Claude Code and Codex plugin that lets a host assistant
delegate arbitrary task work to a peer local coding agent — Codex, Claude
Code, or Gemini through Google Antigravity.

Use it when you are working in Claude Code or Codex and want one of the
other two agents to take a focused task pass — implementation, research,
review, debugging, refactors, docs, build fixes, anything. The host
assistant keeps the conversation with you; the peer agent inspects or
edits the repo, runs verification when it can, and returns a small result
for the host to summarize.

## What It Can Call

Peeragent wraps local CLIs you already have installed:

- `codex`: OpenAI Codex CLI through `codex exec`
- `gemini`: Gemini through Google Antigravity CLI, `agy --print`
- `claude`: Claude Code CLI through `claude --print`

It does not include accounts, API keys, Codex, Claude Code, or Antigravity.
Install and sign in to the target CLIs you want to use before delegating work.

## Install From The Marketplace

Claude Code:

```sh
claude plugin marketplace add nklisch/peeragent
claude plugin install peeragent@peeragent
```

Codex:

```sh
codex plugin marketplace add nklisch/peeragent
codex plugin add peeragent@peeragent
```

The marketplace installs the plugin source. On first use, `bin/peeragent`
looks for a local compiled binary, then a cached release binary, then downloads
the matching binary from the GitHub release for the plugin version. If no release
asset is available, it falls back to `go run` when Go is installed.

## Using It

The plugin exposes two skills, available in both Claude Code and Codex:

- `/peer`: delegate a focused task pass to a peer local coding agent
- `/peer-review`: iterative cross-model peer review of recent work

Example prompts:

```text
/peer Fix the failing parser test and run the relevant test package.
/peer --agent claude --model opus --effort xhigh Refactor the result formatter and update its tests.
/peer --agent gemini Inspect the CLI docs and patch stale usage text.
/peer-review
```

The host assistant remains responsible for reading the wrapper result and
explaining the outcome to you.

## Agent Equivalence And Defaults

Peeragent does not automatically replace a host assistant's normal sub-agent
pattern. If you want that behavior, add a project instruction to `CLAUDE.md` or
`AGENTS.md` telling the host when to delegate through these skills.

Use this rough equivalence when choosing a target:

| Desired delegated pass | Codex target | Claude target | Gemini target |
| --- | --- | --- | --- |
| Lightweight or fast pass | `--agent codex --effort medium` | `--agent claude --model haiku --effort high` | `--agent gemini --model gemini-3.5` |
| Normal implementation, research, or review pass | `--agent codex` or `--agent codex --effort high` | `--agent claude --model sonnet` or `--agent claude --model sonnet --effort xhigh` | `--agent gemini --model gemini-3.5` |
| Deeper implementation, research, or review pass | `--agent codex --effort xhigh` | `--agent claude --model opus --effort xhigh` | `--agent gemini --model gemini-3.5` |

Gemini through Antigravity is treated as fixed Gemini 3.5 for this wrapper. The
`--model gemini-3.5` spelling is accepted when you want to be explicit, but the
wrapper does not pass a model flag to `agy` because `agy --print` does not
expose a non-interactive model option today.

Claude Code project snippet:

```markdown
## Peer Delegation

When you would normally use an implementation, research, or review sub-agent,
prefer `/peer` for concrete code changes, bug fixes, refactors, tests, docs
updates, build fixes, research passes, and review passes in this repository.

- Use `/peer` with no `--agent` flag for the default Codex pass.
- Use `/peer --agent claude --model sonnet` for a normal Claude pass.
- Use `/peer --agent gemini` for a Gemini 3.5 pass through Antigravity.
- Use `--effort xhigh` when the work is dense or the stakes are high.
- Research-only and review-only delegation are allowed.
- Use `/peer-review` for iterative cross-model review of recent work.
- Do not use peeragent for planning-only orchestration work.
```

Codex project snippet:

```markdown
## Peer Delegation

When you would normally use an implementation, research, or review sub-agent,
prefer `/peer` for concrete code changes, bug fixes, refactors, tests, docs
updates, build fixes, research passes, and review passes in this repository.

- Use `/peer --agent claude` (default `--model sonnet`, default `--effort xhigh`)
  for a normal Claude pass; `--model opus --effort xhigh` for the deeper Claude
  pass; `--model haiku --effort high` for lightweight Claude work.
- Use `/peer --agent gemini` for a Gemini 3.5 pass through Antigravity.
- Use `/peer` with no `--agent` flag for a Codex pass at default high effort;
  `--effort xhigh` for the deeper Codex pass.
- Use `/peer-review` for iterative cross-model review of recent work.
- Do not use peeragent for planning-only orchestration work.
```

## Direct CLI Usage

Clone and build for local development:

```sh
git clone https://github.com/nklisch/peeragent.git
cd peeragent
make build
```

Blocking mode is the default:

```sh
bin/peeragent --agent codex "Implement the requested change and run relevant tests."
bin/peeragent --agent gemini "Implement the requested change and run relevant tests."
bin/peeragent --agent claude "Implement the requested change and run relevant tests."
```

Read task text from a file:

```sh
bin/peeragent --agent codex --prompt-file task.md
```

Run against another checkout:

```sh
bin/peeragent --cwd /path/to/repo --agent claude "Update the CLI help text."
```

Ask for human-readable output:

```sh
bin/peeragent --text --agent gemini "Fix the failing parser test."
```

## Models, Effort, Profiles, And Access

Codex and Claude support `--effort`. Codex defaults to `high`; use `medium`
only for lightweight Codex work and `xhigh` for deeper Codex passes. Claude
defaults to `xhigh` and accepts only `high` or `xhigh`:

```sh
bin/peeragent --agent codex "Implement the routine change."
bin/peeragent --agent codex --effort medium "Make the localized docs update."
bin/peeragent --agent codex --effort xhigh "Review the cross-module migration for hidden regressions."
bin/peeragent --agent claude --model sonnet "Implement the small change."
bin/peeragent --agent claude --model opus --effort xhigh "Untangle the failing integration test."
bin/peeragent --agent claude --model haiku --effort high "Make the localized docs update."
```

Claude supports `--model sonnet`, `--model opus`, and `--model haiku`. Gemini
accepts only `--model gemini-3.5`; this records the fixed Gemini target but does
not add an `agy` model flag because `agy --print` does not expose a
non-interactive model option. Use Antigravity's own `/model` flow outside this
wrapper if you want to change its global default.

```sh
bin/peeragent --agent gemini --model gemini-3.5 "Implement the requested change."
```

Codex also supports profiles:

```sh
bin/peeragent --agent codex --profile peeragent "Use this Codex profile."
```

Default execution stays inside the current checkout using the bounded mode each
target CLI exposes:

```text
codex exec --cd <repo> --sandbox workspace-write ...
agy --print --sandbox --add-dir <repo> ...
claude --print --permission-mode auto --add-dir <repo> ...
```

Use full access only for a trusted repo and an explicit reason:

```sh
bin/peeragent --agent claude --full-access "Run the trusted local migration."
```

`--worktree` is reserved for future isolated worktree execution. Today it
returns a clear JSON failure instead of silently changing how work is done.

## Async Jobs

For longer work, start a background job:

```sh
bin/peeragent --agent gemini --async "Refactor the result formatter and run tests."
```

Check status:

```sh
bin/peeragent --status <job-id>
```

Fetch the result:

```sh
bin/peeragent --result <job-id>
```

Cancel the job:

```sh
bin/peeragent --cancel <job-id>
```

Async state lives under `.peeragent/jobs/` in the target repository. It is
local runtime state and ignored by git.

## Repository Shape

This repo is shaped as both a Claude Code marketplace and a Codex marketplace.
The root is the development source; `plugin/` is the committed install package
that marketplaces point at.

```text
.claude-plugin/marketplace.json        # Claude marketplace entry, source ./plugin
.agents/plugins/marketplace.json       # Codex marketplace entry, source ./plugin
plugin/.claude-plugin/plugin.json      # Claude plugin manifest
plugin/.codex-plugin/plugin.json       # Codex plugin manifest
plugin/skills/peer/SKILL.md
plugin/skills/peer-review/SKILL.md
plugin/bin/peeragent
```

The root also keeps the same manifests, skills, and shim for local development.
Run `scripts/package-plugin.sh` after changing plugin metadata, skills, or the
shim so `plugin/` stays in sync.

## Releasing

Release artifacts are the compiled binaries used by marketplace installs when Go
is not available locally.

Build release archives locally:

```sh
make release VERSION=0.2.1
```

That writes:

```text
dist/release/peeragent_0.2.1_linux_amd64.tar.gz
dist/release/peeragent_0.2.1_linux_arm64.tar.gz
dist/release/peeragent_0.2.1_darwin_amd64.tar.gz
dist/release/peeragent_0.2.1_darwin_arm64.tar.gz
dist/release/checksums.txt
```

Publish a GitHub release from a machine with `gh` authenticated:

```sh
make publish-release VERSION=0.2.1
```

The GitHub Actions workflow in `.github/workflows/release.yml` also publishes
these assets whenever a `v*` tag is pushed, or when run manually with a version.

## Development

Run the test suite:

```sh
make test
```

Run the full validation script:

```sh
scripts/validate.sh
```

The validation script runs tests, builds the binary, builds release archives,
checks plugin and marketplace metadata, verifies README examples, and runs a
small shim smoke test.

## Troubleshooting

If delegation fails immediately, confirm the target CLI is installed and signed
in:

```sh
codex --version
agy --version
claude --version
```

If the wrapper cannot download a release binary, install Go and run `make build`,
or set `PEERAGENT_BIN` to an executable `peeragent` binary.

If an async lookup fails, make sure the job id came from the same repository and
that `.peeragent/jobs/<job-id>/job.json` still exists.

### Gemini auth times out every call

Antigravity (`agy`) stores its OAuth token in the system keyring (Keychain on
macOS, Credential Manager on Windows, libsecret on Linux). On headless Linux
without a desktop session, no Secret Service is running by default, so `agy`
falls through to a one-time browser flow that times out in non-interactive
mode and never persists a token. Symptoms: the wrapper returns immediately
with output containing `Authentication required` and `authentication timed
out` even after a successful interactive login.

Fix on Linux:

1. Install and start a libsecret provider:

   ```sh
   sudo dnf install gnome-keyring         # or: sudo apt install gnome-keyring
   eval "$(gnome-keyring-daemon --start --components=secrets,ssh)"
   ```

2. Use an empty-password default keyring so it auto-unlocks without prompts
   (Seahorse: delete the default keyring and create a new one with no
   password, then mark it default).

3. Run `agy` once interactively, complete the browser login, type a prompt
   to confirm, and exit cleanly. The token lands in libsecret under
   `service=gemini` and persists across runs.

4. Keep the daemon alive across logins by adding to your shell rc:

   ```bash
   if ! pgrep -f 'gnome-keyring-daemon.*--components=secrets' >/dev/null 2>&1; then
     eval "$(gnome-keyring-daemon --start --components=secrets,ssh 2>/dev/null)"
   fi
   ```

If the keyring is impractical (CI, locked-down hosts), set
`ANTIGRAVITY_API_KEY` from <https://aistudio.google.com/apikey> instead —
that bypasses libsecret entirely but routes through AI Studio's quota
rather than your Antigravity subscription.
