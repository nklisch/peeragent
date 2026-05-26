# Codex Implement

Codex Implement is a Claude Code plugin that lets Claude delegate implementation work to OpenAI Codex CLI through a small local wrapper.

Claude stays responsible for the conversation. Codex gets an implementation task, works in the same repository by default, and returns a compact JSON result that Claude can read.

## Prerequisites

- Claude Code with plugin and skill support.
- Codex CLI installed and authenticated locally.
- Go installed for local builds.
- A repository open in Claude Code.

The plugin uses your local Codex CLI and authentication. It does not bundle Codex or manage a separate account.

## Build

Build the wrapper binary:

```sh
make build
```

This writes:

```text
dist/codex-implement
```

The executable shim at `bin/codex-implement` prefers that compiled binary. If the binary is missing, the shim falls back to `go run` for development.

You can also run the build script directly:

```sh
scripts/build.sh
```

Run tests with:

```sh
make test
```

## Plugin Layout

The distributable plugin files are:

```text
.claude-plugin/plugin.json
skills/codex-implement/SKILL.md
bin/codex-implement
dist/codex-implement
```

Use Claude Code's local plugin installation flow for this plugin directory. After installation, Claude should see the `codex-implement` skill.

## Usage

Blocking mode is the default:

```sh
bin/codex-implement "Implement the requested change and run relevant tests."
```

The wrapper emits JSON by default. Use text output for manual terminal use:

```sh
bin/codex-implement --text "Fix the failing parser test."
```

For larger prompts:

```sh
bin/codex-implement --prompt-file task.md
```

Run from another directory:

```sh
bin/codex-implement --cwd /path/to/repo "Update the CLI help text."
```

## Reasoning Effort

The wrapper exposes two effort levels:

```sh
bin/codex-implement --effort medium "Implement the small change."
bin/codex-implement --effort high "Implement the cross-module migration."
```

`medium` is the default and is the standard setting for implementation work. Use `high` for harder or more complex tasks. Lower and extra-high effort modes are intentionally not exposed.

## Async Jobs

Async mode starts a tracked local job and returns a job id:

```sh
bin/codex-implement --async "Refactor the result formatter and run tests."
```

Check status:

```sh
bin/codex-implement --status <job-id>
```

Fetch the final result:

```sh
bin/codex-implement --result <job-id>
```

Cancel a running job:

```sh
bin/codex-implement --cancel <job-id>
```

Async state is stored under `.codex-implement/jobs/` in the target repository. That directory is local runtime state and is ignored by git.

## Safety And Access

By default, Codex runs in the current checkout with classifier-compatible approval review:

```text
codex exec --cd <repo> --sandbox workspace-write --ask-for-approval on-request ...
```

Use full access only when the user explicitly wants Codex to bypass normal approval and sandbox boundaries:

```sh
bin/codex-implement --full-access "Run the trusted local migration."
```

`--worktree` is recognized but not implemented yet. It returns a JSON failure instead of silently changing the execution model.

## Troubleshooting

If Codex CLI is missing or unauthenticated, install or authenticate Codex CLI first, then retry the wrapper.

If an async lookup fails, the wrapper returns JSON with `exit_code: 4` and the requested `job_id`. Confirm the job id came from the same repository and that `.codex-implement/jobs/<job-id>/job.json` still exists.

If the shim falls back to `go run` and you expected a compiled binary, run `make build` and confirm `dist/codex-implement` is executable.
