---
id: story-fix-gemini-sandbox-false-positive
kind: story
stage: review
tags: [bug]
parent: null
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-31
updated: 2026-05-31
---

# Gemini delegation returns a false auth/timeout failure caused by the default --sandbox flag

## Symptom
Delegating to Gemini through Antigravity (`agy`) fails on every non-full-access
call with output that looks like an auth failure ("auth false positive"), even
though the user is interactively logged in and `agy` is authenticated. Reported
as "gemini peeragent auth false positive / wiring".

## Root cause
Two compounding defects, both in `internal/gemini`:

1. **Wiring (`buildArgs`)** — non-full-access Gemini ran `agy --print --sandbox
   ...`. With `--sandbox`, agy restricts its terminal tool to the workspace; the
   print-mode agent then loops on blocked/non-converging tool calls
   (`PlannerResponse without ModifiedResponse`, `checkpoint model generated tool
   calls`) and never produces a final response, hitting `Print mode: timed out
   after 60 polls` and printing `Error: timed out waiting for response`. agy
   logs confirm authentication succeeds in both sandbox and non-sandbox runs
   (`authenticated via keyring`, `OAuth: authenticated successfully`, `silent
   auth succeeded`) — so this is a sandbox-behavior failure, not auth. Without
   `--sandbox`, agy converges and answers normally.

2. **Exit-code (`ExecWithRunner`)** — `agy --print` exits 0 even when it emits a
   fatal `Error: ...` line and produced no response. peeragent derives status
   from the exit code (`main.go`: `if result.ExitCode != 0 { status = Failed }`),
   so a 0 exit was reported as a false success.

## Fix approach
- Drop `--sandbox` from the Gemini wiring entirely. Antigravity exposes no other
  isolation flag; non-full-access now runs `agy --print --add-dir <cwd>
  --print-timeout <t>` (agy's own request-review permission default applies),
  full-access keeps `--dangerously-skip-permissions`.
- Add `normalizeResult` to `ExecWithRunner`: when agy exits 0 but its final
  output line is an `Error: ...` print-mode failure, remap the exit code to
  non-zero so the wrapper reports an honest failure instead of a false success.

## Regression test
`internal/gemini/exec_test.go`:
- Argv tests assert no `--sandbox` for default / fixed-model / resume paths;
  full-access path unchanged.
- New tests assert that an agy result with `ExitCode: 0` plus a trailing
  `Error: timed out waiting for response` is remapped to a non-zero exit, while
  a normal successful result keeps exit 0.
