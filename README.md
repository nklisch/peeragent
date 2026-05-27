# Alt Subagent

Alt Subagent is a dual Claude Code and Codex plugin that lets one local coding
assistant delegate implementation, research, or review work to another local
coding assistant.

Use it when you are working in Claude Code or Codex and want a second agent to
take a focused task pass. The host assistant keeps the conversation with you;
the target agent inspects or edits the repo, runs verification when it can, and
returns a small result for the host to summarize.

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

- `/codex-implement`: delegate implementation, research, or review work to Codex
- `/gemini-implement`: delegate implementation, research, or review work to Gemini through Antigravity

Example prompts:

```text
/codex-implement Fix the failing parser test and run the relevant test package.
/gemini-implement Refactor the result formatter and update its tests.
```

Claude Code remains responsible for reading the wrapper result and explaining
the outcome to you.

## Using It From Codex

The Codex plugin exposes these skills:

- `claude-implement`: delegate implementation, research, or review work to Claude Code
- `gemini-implement`: delegate implementation, research, or review work to Gemini through Antigravity

Example requests:

```text
Use claude-implement to add the missing validation test.
Use gemini-implement to inspect the CLI docs and patch stale usage text.
```

Codex remains responsible for the final response. The delegated agent is only
the focused worker for that task.

## Agent Equivalence And Harness Overrides

Alt Subagent does not automatically replace a host assistant's normal sub-agent
pattern. If you want that behavior, add a project instruction to `CLAUDE.md` or
`AGENTS.md` telling the host when to delegate through these skills.

Use this rough equivalence when choosing a target:

| Desired delegated pass | Codex target | Claude target | Gemini target |
| --- | --- | --- | --- |
| Lightweight or fast pass | `--agent codex --effort medium` | `--agent claude --model haiku --effort high` | `--agent gemini --model gemini-3.5` |
| Normal implementation, research, or review sub-agent | `--agent codex` or `--agent codex --effort high` | `--agent claude --model sonnet` or `--agent claude --model sonnet --effort xhigh` | `--agent gemini --model gemini-3.5` |
| Deeper implementation, research, or review pass | `--agent codex --effort xhigh` | `--agent claude --model opus --effort xhigh` | `--agent gemini --model gemini-3.5` |

Gemini through Antigravity is treated as fixed Gemini 3.5 for this wrapper. The
`--model gemini-3.5` spelling is accepted when you want to be explicit, but the
wrapper does not pass a model flag to `agy` because `agy --print` does not
expose a non-interactive model option today.

Claude Code project snippet:

```markdown
## Alt Subagent Delegation

When you would normally use an implementation, research, or review sub-agent,
prefer Alt Subagent for concrete code changes, bug fixes, refactors, tests,
docs updates, build fixes, research passes, and review passes in this
repository.

- Use `/codex-implement` for a Codex task pass.
- Use `/gemini-implement` for a Gemini 3.5 task pass through
  Antigravity; `--model gemini-3.5` is the only accepted Gemini model value.
- Use the default high effort for routine Codex work.
- Ask the wrapper for `--effort xhigh` when Codex should take the deeper pass.
- Research-only and review-only delegation are allowed when a second-agent pass
  is useful.
- Do not use Alt Subagent for planning-only orchestration work.
```

Codex project snippet:

```markdown
## Alt Subagent Delegation

When you would normally use an implementation, research, or review sub-agent,
prefer Alt Subagent for concrete code changes, bug fixes, refactors, tests,
docs updates, build fixes, research passes, and review passes in this
repository.

- Use `claude-implement` for a Claude Code task pass.
- Use `--model sonnet` for normal Claude work, `--model opus --effort xhigh`
  for the deeper Claude pass, and `--model haiku --effort high` for lightweight
  Claude work.
- Use `gemini-implement` for a Gemini 3.5 task pass through
  Antigravity; `--model gemini-3.5` is the only accepted Gemini model value.
- Use default high effort for routine Codex work and default xhigh effort for
  routine Claude work.
- Ask the wrapper for `--effort xhigh` when Codex should take the deeper pass;
  use `--model opus --effort xhigh` for the deeper Claude pass.
- Research-only and review-only delegation are allowed when a second-agent pass
  is useful.
- Do not use Alt Subagent for planning-only orchestration work.
```

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

## Models, Effort, Profiles, And Access

Codex and Claude support `--effort`. Codex defaults to `high`; use `medium`
only for lightweight Codex work and `xhigh` for deeper Codex passes. Claude
defaults to `xhigh` and accepts only `high` or `xhigh`:

```sh
bin/alt-subagent --agent codex "Implement the routine change."
bin/alt-subagent --agent codex --effort medium "Make the localized docs update."
bin/alt-subagent --agent codex --effort xhigh "Review the cross-module migration for hidden regressions."
bin/alt-subagent --agent claude --model sonnet "Implement the small change."
bin/alt-subagent --agent claude --model opus --effort xhigh "Untangle the failing integration test."
bin/alt-subagent --agent claude --model haiku --effort high "Make the localized docs update."
```

Claude supports `--model sonnet`, `--model opus`, and `--model haiku`. Gemini
accepts only `--model gemini-3.5`; this records the fixed Gemini target but does
not add an `agy` model flag because `agy --print` does not expose a
non-interactive model option. Use Antigravity's own `/model` flow outside this
wrapper if you want to change its global default.

```sh
bin/alt-subagent --agent gemini --model gemini-3.5 "Implement the requested change."
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
make release VERSION=0.1.4
```

That writes:

```text
dist/release/alt-subagent_0.1.4_linux_amd64.tar.gz
dist/release/alt-subagent_0.1.4_linux_arm64.tar.gz
dist/release/alt-subagent_0.1.4_darwin_amd64.tar.gz
dist/release/alt-subagent_0.1.4_darwin_arm64.tar.gz
dist/release/checksums.txt
```

Publish a GitHub release from a machine with `gh` authenticated:

```sh
make publish-release VERSION=0.1.4
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
