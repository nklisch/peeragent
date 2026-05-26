# Alt Subagent

Alt Subagent lets one local coding assistant hand an implementation task to
another local coding assistant without leaving the current repository.

Use it when you are working in Claude Code or Codex and want a second agent to
take an implementation pass. The host assistant keeps the conversation with you;
the target agent edits the repo, runs whatever verification it can, and returns a
small JSON result for the host to summarize.

## What It Can Call

Alt Subagent wraps local CLIs you already have installed:

- `codex`: OpenAI Codex CLI through `codex exec`
- `gemini`: Gemini through Google Antigravity CLI, `agy --print`
- `claude`: Claude Code CLI through `claude --print`

It does not include accounts, API keys, or the target agents themselves. Install
and sign in to the CLIs you want to use before delegating work.

## Quick Start

Clone and build the wrapper:

```sh
git clone https://github.com/nklisch/alt-subagent.git
cd alt-subagent
make build
```

That creates `dist/alt-subagent`. The checked-in shim at `bin/alt-subagent`
uses the compiled binary when it exists and falls back to `go run` during local
development.

Try a direct call:

```sh
bin/alt-subagent --agent gemini --text "Inspect the repo and suggest a small cleanup."
```

Use JSON output, the default, when another assistant is reading the result:

```sh
bin/alt-subagent --agent claude "Implement the requested README update and run tests."
```

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

Default execution stays inside the current checkout using the safest bounded
mode each target CLI exposes:

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

## Development

Run the test suite:

```sh
make test
```

Run the full validation script:

```sh
scripts/validate.sh
```

The validation script builds the binary, checks plugin metadata, verifies README
examples, and runs a small shim smoke test.

## Troubleshooting

If delegation fails immediately, confirm the target CLI is installed and signed
in:

```sh
codex --version
agy --version
claude --version
```

If an async lookup fails, make sure the job id came from the same repository and
that `.alt-subagent/jobs/<job-id>/job.json` still exists.

If `bin/alt-subagent` falls back to `go run` and you expected a compiled binary,
run `make build` and confirm `dist/alt-subagent` is executable.
