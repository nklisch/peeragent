---
id: terminal-environment-isolation
kind: story
stage: review
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


## Verification and handback

Frozen source tested: `98e9a537d718b65bf9991e2e82b4437e648a1655`, using
`go version go1.26.4 linux/amd64`. The final handback commit changes this record
only. All commands passed:

- Focused `go test -json ./internal/executil ./internal/app ./cmd/peeragent -run
  'TestTargetEnvironment|TestOSRunnerTerminalEnvironment|TestEveryTargetFilters|TestTerminalEnvironmentLaunch'
  -count=1`: **4 top-level / 34 total passes**, 3 packages. Covers all four
  targets, fresh/resumed adapter calls, observed/ordinary runners, default/keep,
  blocking/async/resumed CLI calls and both capacity-retry attempts, plus saved
  job spec, launch/result/status JSON and text metadata. Unit checks preserve
  representative authentication variables and Windows private-drive entries.
- `go build ./...` and `go vet ./...`: passed once before the source commit;
  source is unchanged from those checks. No redundant native build/vet replay.
- `go test -json ./...`: **246 top-level / 402 total passes**, 12 tested
  packages, zero failures/skips. The two testsupport packages have no tests.
- `./scripts/validate.sh`: passed once offline in a disposable checkout of the
  same SHA. Includes its required test/build replay, package/shim checks,
  version/docs checks and Linux/macOS amd64/arm64 archives. Existing missing-icon
  warning is nonblocking. The script's local smoke uses only checked-out code;
  no installed plugin or real target agent was invoked.
- `GOOS=<os> GOARCH=<arch> CGO_ENABLED=0 go test -c -o
  <scratch>/<os>-<arch>-tests/ ./...`: passed for Windows amd64 and macOS
  amd64/arm64. Cross-vet passed for Windows amd64 and macOS arm64. Native
  Windows/macOS runtime behavior, including the new Windows batch environment
  regression, remains unverified here.
- `git diff --check` and curated manifest/skill comparisons passed. All six
  versioned manifests and README install examples name v0.10.1. No tag/push,
  installed-plugin modification, committed binary refresh, benchmark or real
  model call.

The combined command/result record is
`/storage/scratch/peeragent/terminal-env/verification.log`; raw focused events
are `focused.jsonl`, full-suite events `full-test.jsonl`, validator detail
`validate.log`, and final exit `verify.exit` (0), all in the same directory.
Checks used `GOCACHE=/storage/build/peeragent/job-controls-3-go-cache` (reusing
this implementer's warm cache) and `GOFLAGS=-p=6`. Keep these logs/cache for
review and any correction round; remove them after the patch lands or is
abandoned. The disposable validation checkout, generated artifacts, cross-test
binaries and one-off edit/verification scripts were removed before handback.

The unrelated configuration/secrets policy remains in the existing backlog
item. The requested terminal-control protection and local v0.10.1 preparation
are complete; godot-41 owns review, tagging and publication.


## v0.10.1 follow-up: wait for async test workers before directory cleanup

Godot-41 added the public Windows CI failures in
`TestStatusShowsLiveProgressInSingleAndMultipleOutput` and
`TestResumeGuardCLIExitAndOverride`: TempDir removal raced workers still holding
`agent.log`. Existing cleanup only requested cancellation. Production Windows
cancel records terminal state without signalling the worker, so terminal state
alone did not establish that files were released.

- A shared test-only cleanup helper now cancels by job id, releases only the
  fake agent's captured test gate, and waits (bounded) for both terminal state
  and the recorded worker to be gone. The fake worker preserves its PID in a
  separate test sidecar because production cancellation removes `pid` before
  exit is guaranteed. The helper never signals a guessed process.
- Every async launch through cliResult registers this cleanup immediately;
  the text-launch test and the custom Unix launch register it explicitly.
  Old cancel-only callbacks are removed. Capturing job location/gate at
  registration preserves environment and TempDir cleanup ordering.
- Fake live/final-hang targets have a cooperative cleanup gate, allowing
  Windows teardown without changing production cancellation or the watchdog
  scenarios before cleanup. Normal completion/cancellation assertions are
  unchanged. `TestAsyncCleanupWaitsForWorkerBeforeTempDirRemoval` checks cleanup
  order, terminal state, worker exit and deletion of agent.log while the test
  directory still exists.
- CHANGELOG v0.10.1 records the Windows test cleanup fix. No production source
  or terminal environment policy changed in this follow-up.


### Async-test cleanup qualification

Frozen test/source SHA: `8b258395888d3de4c3dab0f887721d4e3535a9b5`, with
`go version go1.26.4 linux/amd64`. The final evidence commit changes this record
only. Focused cleanup/live-status/resume-guard/watchdog/backoff tests passed;
the full frozen `go test -json ./...` passed once with **247 top-level / 403
including subtests**, 12 tested packages, zero failures/skips. Two testsupport
packages have no tests. `go build ./...`, `go vet ./...` and `git diff --check`
passed before the implementation commit.

`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o
<scratch>/cleanup-windows-tests/ ./...` passed. Native Windows execution is not
available here; the public Windows runner must confirm the runtime cleanup fix.
No extra validator run was made: this follow-up changes test fixtures/cleanup,
CHANGELOG and this record, and the requested correction checks are the full
suite and Windows compile. The preceding production validation remains recorded
above. No production Go source, manifests, installed plugin, tag or push changed.

Result log: `/storage/scratch/peeragent/terminal-env/cleanup-verification.log`;
raw suite events: `cleanup-full-test.jsonl`; aggregate exit:
`cleanup-verify.exit` (0), all in the same scratch directory. The temporary
Windows test binaries and one-off verification script were removed. Checks
reused `GOCACHE=/storage/build/peeragent/job-controls-3-go-cache` with
`GOFLAGS=-p=6`; retain logs/cache for owner review as above. No in-scope fix is
deferred, and the branch remains ready for godot-41's publication decision.
