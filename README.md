# Peeragent

Peeragent is a Claude Code and Codex plugin that lets a host assistant
delegate arbitrary task work to a peer local coding agent — Codex, Claude
Code, Gemini through Google Antigravity, or Z.AI GLM 5.2 through Pi.

Use it when you are working in Claude Code or Codex and want another local
agent/model to take a focused task pass — implementation, research,
review, debugging, refactors, docs, build fixes, anything. The host
assistant keeps the conversation with you; the peer agent inspects or
edits the repo, runs verification when it can, and returns a small result
for the host to summarize.

## What It Can Call

Peeragent wraps local CLIs you already have installed:

- `codex`: OpenAI Codex CLI through `codex exec`
- `gemini`: Gemini through Google Antigravity CLI, `agy --print`
- `claude`: Claude Code CLI through `claude --print`
- `zai`: Z.AI GLM 5.2 through Pi, `pi --provider zai --model glm-5.2 -p`

It does not include accounts, API keys, Codex, Claude Code, Antigravity, Pi,
or Z.AI access. Install and authenticate/configure the target CLIs you want to
use before delegating work.

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

Pi:

```sh
pi install git:github.com/nklisch/peeragent@v0.4.2
```

The Pi package loads the same `peer` and `peer-review` skills from `plugin/skills`
so their wrapper resolution lands on the bundled `plugin/bin/peeragent` shim and
committed platform binaries.

On the four supported platforms (linux amd64/arm64, darwin amd64/arm64),
the plugin runs immediately with no download and no Go toolchain required —
prebuilt binaries are committed in `plugin/bin/<goos>-<goarch>/peeragent`.

On any other platform, install from source (requires Go 1.25 or newer):
`go install github.com/nklisch/peeragent/cmd/peeragent@latest` (or pin
`@v<plugin-version>` to match your installed plugin), then set `PEERAGENT_BIN`
to the installed binary (typically `$(go env GOPATH)/bin/peeragent`).
Prebuilt release archives for the four supported platforms are also published at
https://github.com/nklisch/peeragent/releases for manual install. If your
platform is misdetected, set `PEERAGENT_TARGET_OVERRIDE=<goos>-<goarch>` to
select a present binary.

## Using It

The plugin exposes two skills, available in both Claude Code and Codex:

- `/peer`: delegate a focused task pass to a peer local coding agent
- `/peer-review`: iterative cross-model peer review of recent work

Example prompts:

```text
/peer --agent codex --model luna --effort high Fix the failing parser test and run the relevant test package.
/peer --agent claude --model fable --effort xhigh Refactor the result formatter and update its tests.
/peer --agent gemini Inspect the CLI docs and patch stale usage text.
/peer --agent zai --effort xhigh Ask GLM 5.2 to audit the retry edge cases.
/peer-review
```

The host assistant remains responsible for reading the wrapper result and
explaining the outcome to you.

## Bundled MCP Server

Installing and enabling the Claude Code or Codex plugin also enables the local
`peeragent` MCP server. No separate global MCP configuration is required. The
plugin manifest points at a host-specific config so Claude Code resolves
`${CLAUDE_PLUGIN_ROOT}/bin/peeragent` and Codex resolves
`${PLUGIN_ROOT}/bin/peeragent`; the two root variables are intentionally not
interchangeable.

The server exposes four tools:

- `delegate` runs a short task in blocking mode or starts a tracked job with
  `async: true`.
- `job_status` polls a repository-local async job.
- `job_result` retrieves its structured result and target details.
- `job_cancel` marks a job cancelled and terminates its detached process group.

Prefer the async workflow for implementation, research, or review work likely
to exceed the host MCP tool timeout: call `delegate` with `async: true`, poll
with `job_status`, and call `job_result` when complete. `delegate` and
`job_cancel` can write to the checkout and should remain approval-gated in the
host. `job_status` and `job_result` are read-only and are reasonable candidates
for automatic approval. These annotations reinforce host policy; they are not
a security boundary. For Codex, plugin-scoped enablement is visible with
`codex plugin list` after `codex plugin add peeragent@peeragent`; approve
`delegate` and `job_cancel` when the host asks, while allowing read-only status
and result calls according to your normal MCP policy. Claude Code uses the
equivalent plugin lifecycle (`claude plugin list`, `claude plugin enable`) and
its normal tool permission prompts for the same approval distinction.

The server uses the current repository by default. Omit `cwd` unless the user
explicitly requests work in another checkout; setting it is intentional
cross-repository reach and must be repeated for job status, result, or cancel.
Set `full_access: true` only after explicit user approval. It disables the
target's bounded mode and can modify or delete files beyond normal sandbox
limits. MCP provides no review-orchestration tool, HTTP transport, or arbitrary
MCP proxy; `/peer-review` remains host-side orchestration.

### Standalone MCP setup

For an MCP host without the plugin, install an executable first and make
`peeragent` available on `PATH` (or replace it with an absolute path). Then use
the host's stdio MCP configuration, for example:

```json
{
  "mcpServers": {
    "peeragent": {
      "command": "peeragent",
      "args": ["mcp"]
    }
  }
}
```

Codex's equivalent command is:

```sh
codex mcp add peeragent -- peeragent mcp
```

A standalone server does not download an executable, provide target-agent
credentials, or open a network listener. The installed executable still needs
the local target CLI and authentication described below.

### Plugin reload and troubleshooting

After changing plugin files, restart the host session; Claude Code plugin
changes also require a reload/restart before MCP configuration is rescanned.
For Claude Code, check the package without starting a model session with:

```sh
claude plugin validate --strict plugin
```

For Codex, verify that the installed plugin is enabled and that its bundled
server is listed:

```sh
codex plugin list
codex mcp list
codex mcp get peeragent
```

If the server is listed but tools do not appear, restart the host and confirm
that the plugin's platform binary is executable. Run `peeragent mcp` only with
an MCP client or protocol inspector: stdout is reserved for JSON-RPC frames and
human-readable diagnostics go to stderr. The packaged configs and manifests
are `.mcp.claude.json`, `.mcp.codex.json`, `.claude-plugin/plugin.json`, and
`.codex-plugin/plugin.json`; do not substitute one host's root variable for the
other.

## Agent Equivalence And Defaults

Peeragent does not automatically replace a host assistant's normal sub-agent
pattern. If you want that behavior, add a project instruction to `CLAUDE.md` or
`AGENTS.md` telling the host when to delegate through these skills.

Use GPT-5.6 for Codex work. Luna is the fast default, Sol is the direct jump
for demanding work, and Terra is an optional bridge when Luna is not enough but
Sol is more than the task needs:

| Desired delegated pass | Recommended Codex target | Rough Claude tier |
| --- | --- | --- |
| Fast or routine | `--agent codex --model luna --effort high` | Sonnet-class |
| Lots of routine work | `--agent codex --model luna --effort xhigh` | Strong general-purpose pass |
| Middle bridge | `--agent codex --model terra --effort high` (or `xhigh`) | Between the general and flagship tiers |
| Opus-tier | `--agent codex --model sol --effort low` (or `medium`) | `--agent claude --model opus --effort xhigh` |
| Fable-tier | `--agent codex --model sol --effort high` (or `xhigh`) | `--agent claude --model fable --effort high` (or `xhigh`) |

Most callers can jump directly from Luna to Sol rather than routing through
Terra. Gemini through Antigravity is treated as fixed Gemini 3.5 for this wrapper. The
`--model gemini-3.5` spelling is accepted when you want to be explicit, but the
wrapper does not pass a model flag to `agy` because `agy --print` does not
expose a non-interactive model option today. Z.AI through Pi is fixed to
`glm-5.2`; `--model glm-5.2` is accepted for explicit metadata, and no other
Z.AI models are surfaced by peeragent.

Claude Code project snippet:

```markdown
## Peer Delegation

When you would normally use an implementation, research, or review sub-agent,
prefer `/peer` for concrete code changes, bug fixes, refactors, tests, docs
updates, build fixes, research passes, and review passes in this repository.

- Use `/peer --agent codex --model luna --effort high` for routine work and
  Luna at `xhigh` when there is lots of work. Jump to Sol at `low|medium` for an
  Opus-tier pass or `high|xhigh` for a Fable-tier pass; Terra is an optional bridge.
- Use `/peer --agent claude --model fable` for the strongest Claude pass.
- Use `/peer --agent gemini` for a Gemini 3.5 pass through Antigravity.
- Use `/peer --agent zai` for a Z.AI GLM 5.2 pass through Pi.
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

- Use `/peer --agent claude --model fable --effort xhigh` for the strongest
  Claude pass; Opus, Sonnet, and Haiku remain available for lower tiers.
- Use `/peer --agent gemini` for a Gemini 3.5 pass through Antigravity.
- Use `/peer --agent zai` for a Z.AI GLM 5.2 pass through Pi.
- For Codex, use GPT-5.6 Luna at `high` for routine work or `xhigh` for lots of
  work. Jump directly to Sol at `low|medium` for an Opus-tier pass or
  `high|xhigh` for a Fable-tier pass. Terra is an optional middle bridge.
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
bin/peeragent --agent codex --model luna --effort high "Implement the requested change and run relevant tests."
bin/peeragent --agent gemini "Implement the requested change and run relevant tests."
bin/peeragent --agent claude "Implement the requested change and run relevant tests."
bin/peeragent --agent zai "Implement the requested change and run relevant tests."
```

Read task text from a file:

```sh
bin/peeragent --agent codex --model luna --prompt-file task.md
```

Run against another checkout:

```sh
bin/peeragent --cwd /path/to/repo --agent claude "Update the CLI help text."
```

Ask for human-readable output:

```sh
bin/peeragent --text --agent gemini "Fix the failing parser test."
```

Continue a prior target-agent session:

```sh
bin/peeragent --agent claude --resume <agent-session> "Check whether the revision addressed your prior concerns."
```

When available, JSON output includes `metadata.agent_session`. Use that value
for continuity inside one review loop. Start a fresh call without `--resume`
when you want an independent second opinion.

The default result is intentionally compact: for Codex JSONL output, peeragent
surfaces only the final completed agent message in `details`. If you need to
inspect the raw target stdout/stderr, open `metadata.log_path`; for continuity,
use `metadata.agent_session` to resume the target session.

## Models, Effort, Profiles, And Access

Codex, Claude, and Z.AI GLM 5.2 support `--effort`. Codex defaults to `high`
and accepts `low`, `medium`, `high`, and `xhigh`. Z.AI defaults to `high` and
accepts `medium`, `high`, and `xhigh`. Claude defaults to `xhigh` and accepts
only `high` or `xhigh` through peeragent:

```sh
bin/peeragent --agent codex --model luna --effort high "Implement the routine change."
bin/peeragent --agent codex --model luna --effort xhigh "Work through the large routine change."
bin/peeragent --agent codex --model sol --effort medium "Run an Opus-tier review."
bin/peeragent --agent codex --model sol --effort xhigh "Run a Fable-tier migration."
bin/peeragent --agent codex --model terra --effort high "Use the optional middle tier."
bin/peeragent --agent claude --model fable --effort xhigh "Untangle the hardest integration test."
bin/peeragent --agent claude --model opus --effort xhigh "Run an Opus pass."
bin/peeragent --agent zai --effort xhigh "Review the cross-module migration for hidden regressions."
```

Codex accepts the short aliases `luna`, `terra`, and `sol` and passes their
canonical `gpt-5.6-*` model IDs to the Codex CLI. Claude supports `--model
fable`, `--model sonnet`, `--model opus`, and `--model haiku`. Gemini
accepts only `--model gemini-3.5`; this records the fixed Gemini target but does
not add an `agy` model flag because `agy --print` does not expose a
non-interactive model option. Use Antigravity's own `/model` flow outside this
wrapper if you want to change its global default. Z.AI accepts only
`--model glm-5.2`; peeragent intentionally does not surface the other Z.AI
models that Pi may list.

```sh
bin/peeragent --agent gemini --model gemini-3.5 "Implement the requested change."
bin/peeragent --agent zai --model glm-5.2 "Implement the requested change."
```

Codex also supports profiles:

```sh
bin/peeragent --agent codex --model luna --profile peeragent "Use this Codex profile."
```

Quick Z.AI GLM 5.2 configuration checks:

```sh
pi --list-models zai | grep -w 'glm-5.2'
pi --provider zai --model glm-5.2 --thinking high --no-session --no-tools -p 'Reply with OK.'
bin/peeragent --agent zai --text 'Reply with OK and do not edit files.'
```

If the Pi smoke test fails, configure Pi with `ZAI_API_KEY` or `/login` for the
ZAI provider, then retry. Peeragent's Z.AI target always uses `glm-5.2`; it does
not expose the other Z.AI models Pi may know about.

Default execution stays inside the current checkout using the bounded mode each
target CLI exposes:

```text
codex exec --json --cd <repo> --sandbox workspace-write ...
agy --print --add-dir <repo> ...
claude --print --output-format json --permission-mode auto --add-dir <repo> ...
pi --provider zai --model glm-5.2 --thinking <effort> --no-session -p ...
```

You can pass `--sandbox` explicitly to select that same default bounded mode
where the target CLI has one. Gemini is the exception: `agy` has no usable
sandbox in print mode, so its bounded default is `agy --print` scoped only by
`--add-dir`. Pi similarly exposes no separate peeragent sandbox flag; run the
Z.AI target only in repositories where you trust a Pi print-mode agent to use
its normal local tools.

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

This repo is shaped as a Claude Code marketplace, a Codex marketplace, and a
Pi package. The root is the development source; `plugin/` is the committed
install package that marketplaces point at.

```text
package.json                            # Pi package manifest, loads ./plugin/skills
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

Marketplace installs use the committed `plugin/bin/<goos>-<goarch>/peeragent`
binaries directly. Release artifacts are those same four platform binaries
published as downloadable archives for manual install.

Build release archives locally:

```sh
make release VERSION=0.4.2
```

That writes:

```text
dist/release/peeragent_0.4.2_linux_amd64.tar.gz
dist/release/peeragent_0.4.2_linux_arm64.tar.gz
dist/release/peeragent_0.4.2_darwin_amd64.tar.gz
dist/release/peeragent_0.4.2_darwin_arm64.tar.gz
dist/release/checksums.txt
```

Publish a GitHub release from a machine with `gh` authenticated:

```sh
make publish-release VERSION=0.4.2
```

The GitHub Actions workflow in `.github/workflows/release.yml` also publishes
these assets whenever a `v*` tag is pushed, or when run manually with a version.

## Development

The supported minimum Go version for source builds and development is Go 1.25.

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
pi --version
```

If peeragent reports it has no prebuilt binary for this platform (exit code 3),
note that the prebuilt platforms are linux/darwin on amd64/arm64. On those,
reinstall the plugin or download the matching archive from
https://github.com/nklisch/peeragent/releases. On any other platform, install
from source (requires Go 1.25 or newer): `go install github.com/nklisch/peeragent/cmd/peeragent@latest`
then set `PEERAGENT_BIN` to the installed binary. For source checkouts,
`make build` also produces a local `dist/peeragent` binary.

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
