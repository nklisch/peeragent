# Changelog

## v0.6.0

### Features
- Default Gemini delegation to Gemini 3.7 Flash at high effort, with selectable Flash/Pro families and native Antigravity model and effort flags.
- Capture Antigravity structured output and conversation IDs for reliable results and resume support.
- Run Gemini autonomously with edit auto-approval, tool auto-approval, and terminal sandboxing; `--full-access` additionally removes terminal isolation.
- Add `--wait <job-id>` so native host background monitors can stay attached to an async peeragent job and receive its terminal result.

### Removed
- Remove the bundled MCP adapter and plugin registration. Detached MCP work has no portable completion wake-up, while blocking MCP calls compete with host timeouts; the plugin now uses the bundled CLI completion boundary.
- Remove the opinionated `peer-review` orchestration skill. The general `peer` skill remains the single way to give hosts access to other local agents.

## v0.5.1

### Fixes
- Repair Codex plugin MCP packaging so the bundled server uses the standard `.mcp.json` contract and starts from the plugin root.
- Validate the packaged Codex MCP configuration with the real plugin-relative working directory.

## v0.5.0

### Features
- Add a stdio MCP server with typed `delegate`, `job_status`, `job_result`, and `job_cancel` tools backed by shared application services.
- Bundle MCP server configuration and guidance for Claude Code and Codex plugins.
- Add GPT-5.6 Luna, Terra, and Sol model aliases for Codex and Fable model support for Claude.
- Ship the first-class Pi package alongside Claude Code and Codex marketplace packaging.
- Harden asynchronous jobs with durable prompt/PID state, guarded terminal transitions, process-group cancellation, and robust child cleanup.

### Security
- Reject malformed and path-traversing async job IDs at input and storage boundaries.

### Fixes
- Prevent inherited Go test-helper state from launching peeragent recursively.
- Avoid Gemini sandbox false-positive authentication and timeout failures.
- Preserve cleanup when detached-process PID persistence or process release fails.

### Refactor
- Centralize agent identities, aliases, prompt names, and display names in one typed registry.
- Centralize persisted lifecycle statuses and terminal-state membership.
- Share target-adapter runner test support without weakening target-specific assertions.

### Documentation
- Document MCP setup, Pi packaging, supported models, async job controls, and the current repository layout.
