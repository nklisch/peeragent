---
id: terminal-environment-isolation
kind: story
stage: implementing
tags: [security, infra, tests, docs]
parent: null
depends_on: []
release_binding: null
created: 2026-10-22
updated: 2026-10-22
---

# Isolate delegated agents from terminal control

Godot-41 requested the v0.10.1 correction after a delegated Codex used Herdr
prompt input against the launching session. Its command events were confirmed
read-only in the supplied rollout; no terminal or peer session was contacted.

The threat is delegated input injected into the host terminal indistinguishably
from the person. The protection removes the launching terminal's inherited
control addresses from every target agent, with the deliberate, recorded
`--keep-terminal-env` override. It is an environment policy, not a broader
process/desktop sandbox or a general secrets allowlist.

## Implementation

- One shared filter at OSRunner's process boundary applies to all adapters and
  launch modes. It preserves the parent environment, authentication, ordinary
  settings and Windows batch argument variables. Windows uses case-insensitive
  key matching; Unix uses exact names. The complete variable list is documented
  once in `docs/TERMINAL_ENV.md`, including the confirmed Kitty, Alacritty and
  Konsole equivalents.
- The CLI flag flows through normalized requests, adapter options and saved
  execution specs. Async/resume and capacity retries use the same policy.
  Job/result/status metadata records a true override; text results and the
  multi-status note expose it. A fresh resume defaults to stripping unless
  the caller explicitly passes the flag again.
- Independent synthetic fixtures verify the complete list, prefix boundaries,
  preservation of representative authentication and ordinary settings, parent
  environment immutability and the override. Fake executables cover all four
  target adapters with fresh/resumed calls, blocking/async CLI calls and both
  attempts of a capacity retry. A native Windows batch case is included for CI.
  No fake agent prints the real environment or starts an installed agent CLI.
- Prepare v0.10.1 in every versioned manifest and README installation example,
  add CHANGELOG notes and sync both bundled skill copies. No tag/push or installed
  plugin update is authorized; godot-41 owns publication.

## Scope and open work

The existing `gate-security-environment-inheritance` backlog item retains the
broader configuration/secrets question. This patch deliberately preserves
unrelated settings, and records the terminal-control portion there without
closing that broader item. No other in-scope work is deferred.
