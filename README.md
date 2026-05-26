# Alt Subagent

Alt Subagent is a dual Claude Code and Codex plugin that lets one local coding
assistant delegate implementation work to another local coding assistant.

Use it when you are working in Claude Code or Codex and want a second agent to
take an implementation pass. The host assistant keeps the conversation with you;
the target agent edits the repo, runs verification when it can, and returns a
small result for the host to summarize.

## What It Can Call

Alt Subagent wraps local CLIs you already have installed:

- `codex`: OpenAI Codex CLI through `codex exec`
- `gemini`: Gemini through Google Antigravity CLI, `agy --print`
- `claude`: Claude Code CLI through `claude --print`

It does not include accounts, API keys, Codex, Claude Code, or Antigravity.
Install and sign in to the target CLIs you want to use before delegating work.

## Install From The Marketplace

Claude Code:

```sh
claude plugin marketplace add nklisch/alt-subagent
claude plugin install alt-subagent@alt-subagent
```

Codex:

```sh
codex plugin marketplace add nklisch/alt-subagent
codex plugin add alt-subagent@alt-subagent
```

The marketplace installs the plugin source. On first use, `bin/alt-subagent`
looks for a local compiled binary, then a cached release binary, then downloads
the matching binary from the GitHub release for the plugin version. If no release
asset is available, it falls back to `go run` when Go is installed.

## Using It From Claude Code

The Claude Code plugin exposes these skills:

- `/codex-implement`: delegate implementation work to Codex
- `/gemini-implement`: delegate implementation work to Gemini through Antigravity

Example prompts:

```text
/codex-implement Fix the failing parser test and run the relevant test package.
/gemini-implement Refactor the result formatter and update its tests.
```

Claude Code remains responsible for reading the wrapper result and explaining
the outcome to you.

## Using It From Codex

The Codex plugin exposes these skills:

- `claude-implement`: delegate implementation work to Claude Code
- `gemini-implement`: delegate implementation work to Gemini through Antigravity

Example requests:

```text
Use claude-implement to add the missing validation test.
Use gemini-implement to inspect the CLI docs and patch stale usage text.
```

Codex remains responsible for the final response. The delegated agent is only
the implementation worker.

## Direct CLI Usage

Clone and build for local development:

```sh
git clone https://github.com/nklisch/alt-subagent.git
cd alt-subagent
make build
```

Blocking mode is the default:

```sh
bin/alt-subagent --agent codex "Implement the requested change and run relevant tests."
bin/alt-subagent --agent gemini "Implement the requested change and run relevant tests."
bin/alt-subagent --agent claude "Implement the requested change and run relevant tests."
```

Read task text from a file:

```sh
bin/alt-subagent --agent codex --prompt-file task.md
```

Run against another checkout:

```sh
bin/alt-subagent --cwd /path/to/repo --agent claude "Update the CLI help text."
```

Ask for human-readable output:

```sh
bin/alt-subagent --text --agent gemini "Fix the failing parser test."
```

## Effort, Profiles, And Access

Codex and Claude support `--effort`:

```sh
bin/alt-subagent --agent codex --effort medium "Implement the small change."
bin/alt-subagent --agent codex --effort high "Implement the cross-module migration."
bin/alt-subagent --agent claude --effort high "Untangle the failing integration test."
```

Codex also supports profiles:

```sh
bin/alt-subagent --agent codex --profile alt-subagent "Use this Codex profile."
```

Default execution stays inside the current checkout using the bounded mode each
target CLI exposes:

```text
codex exec --cd <repo> --sandbox workspace-write --ask-for-approval on-request ...
agy --print --sandbox --add-dir <repo> ...
claude --print --permission-mode auto --add-dir <repo> ...
```

Use full access only for a trusted repo and an explicit reason:

```sh
bin/alt-subagent --agent claude --full-access "Run the trusted local migration."
```

`--worktree` is reserved for future isolated worktree execution. Today it
returns a clear JSON failure instead of silently changing how work is done.

## Async Jobs

For longer work, start a background job:

```sh
bin/alt-subagent --agent gemini --async "Refactor the result formatter and run tests."
```

Check status:

```sh
bin/alt-subagent --status <job-id>
```

Fetch the result:

```sh
bin/alt-subagent --result <job-id>
```

Cancel the job:

```sh
bin/alt-subagent --cancel <job-id>
```

Async state lives under `.alt-subagent/jobs/` in the target repository. It is
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
plugin/skills/claude-implement/SKILL.md
plugin/skills/codex-implement/SKILL.md
plugin/skills/gemini-implement/SKILL.md
plugin/bin/alt-subagent
```

The root also keeps the same manifests, skills, and shim for local development.
Run `scripts/package-plugin.sh` after changing plugin metadata, skills, or the
shim so `plugin/` stays in sync.

## Releasing

Release artifacts are the compiled binaries used by marketplace installs when Go
is not available locally.

Build release archives locally:

```sh
make release VERSION=0.1.0
```

That writes:

```text
dist/release/alt-subagent_0.1.0_linux_amd64.tar.gz
dist/release/alt-subagent_0.1.0_linux_arm64.tar.gz
dist/release/alt-subagent_0.1.0_darwin_amd64.tar.gz
dist/release/alt-subagent_0.1.0_darwin_arm64.tar.gz
dist/release/checksums.txt
```

Publish a GitHub release from a machine with `gh` authenticated:

```sh
make publish-release VERSION=0.1.0
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
or set `ALT_SUBAGENT_BIN` to an executable `alt-subagent` binary.

If an async lookup fails, make sure the job id came from the same repository and
that `.alt-subagent/jobs/<job-id>/job.json` still exists.
