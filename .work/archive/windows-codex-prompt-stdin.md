---
id: windows-codex-prompt-stdin
kind: story
stage: done
tags: [infra, tests]
parent: null
depends_on: []
release_binding: null
created: 2026-10-17
updated: 2026-10-17
---

# Large Codex briefs fail through Windows npm shims

A task larger than 8191 characters failed immediately when Codex resolved to
an npm `.cmd` shim. `--prompt-file` read the brief successfully, but the Codex
adapter then passed the entire constructed prompt as an argument, triggering
the shared batch runner's size guard. Reproduced for new and resumed sessions
in the Windows CLI regression test before changing production code.

The shared runner now accepts explicit stdin text. Codex sends its prompt
through stdin and uses `-` for the prompt argument in both permission modes,
including resume. Other adapters retain their existing argument transport.

Validation: `go test ./...` passes on Windows. The CLI tests cover positional
tasks and file-based briefs exceeding 10 KB and 100 KB, both permission
modes, new and resumed sessions, and exact preservation of quotes, shell
characters, Unicode, and mixed line endings through an npm-style shim in a
path containing spaces and parentheses. Native stdin transport is also tested.
Installed Codex help confirms stdin support for both command forms. No model
request is needed for these transport checks.
