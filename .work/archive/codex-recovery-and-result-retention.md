---
id: codex-recovery-and-result-retention
kind: story
stage: done
tags: [infra, tests, docs]
parent: null
depends_on: []
release_binding: v0.12.3
created: 2026-10-24
updated: 2026-10-24
---

# Recover Codex stops and retain results after worktree removal

Implementation: `6ea8944a271c249240010dbd19295fad438c5b1f`, based on v0.12.2
at `8a845a4`. The launching session authorized replacing the initially stale
v0.8.1 branch baseline with origin/main before implementation.

## Outcomes

- Revoked login previously produced `failed`. It now produces `auth_required`,
  preserves the agent session, and offers manual resumption. Async Codex's
  existing `--retry-capacity N` budget permits one immediate continuation using
  the current login. A second auth stop stays terminal even with budget left;
  no session means no retry. The wrapper never edits credentials.
- Rollout-recording loss previously produced generic failure. The precise
  `ERROR codex_core::session: failed to record rollout items: thread ... not found`
  diagnostic now produces `interrupted` only without a structured terminal
  outcome. The conversation remains resumable; no automatic retry is added.
- Inspection of the reported Godot incident revealed a structured cybersecurity
  refusal before its rollout stderr. The launching session confirmed that
  structured outcomes win, so that incident correctly stays `failed` and now
  exposes `failure: policy_refusal` plus the actual refusal reason in its
  summary. Single-job status and job listings expose that classification too.
  Policy refusals are never retried automatically.
- Per-user id lookup from any directory already exists in v0.12.2. Terminal
  results now also have a private JSON snapshot outside the worktree, allowing
  result, wait, single-job status, and cancellation reads after job metadata
  disappears. Explicit directories retain precedence. Corrupt local files
  remain errors rather than being hidden by snapshots.
- The launching session chose 30-day snapshot retention because no prior job
  expiry policy existed. Writing a new snapshot prunes older snapshots, with
  no daemon or extra command. Repository job directories keep their existing
  lifetime. Only result JSON is copied; prompts and raw log files are not.
  Archive failure warns in `metadata.index_warning` while preserving completion.

Inbox replies remain unread across auth/interruption stops, matching capacity
stops, so resumed turns do not silently lose launcher instructions. Existing
cancellation and terminal-winner precedence remain in force.

## Verification log

The implementation source above passed 29 distinct top-level tests across the
focused commands below (subtests excluded from that count). No full suite or
release validation was run, following the implementation brief's cadence.
All commands ran directly as light Go checks, without heavy-job or timing runs.
This file is the consolidated evidence log; generated binaries were removed.

```sh
go test ./internal/codex -run 'Test(CodexRecovery|ReportedCapacity|OutputLimit)' -count=1 -v
# PASS: four top-level tests; JSONL/stream parity, exact diagnostics, precedence,
# prose/tool-output exclusions, and existing capacity/output-limit behavior.

go test ./internal/jobs ./internal/app -run 'Test(ArchivedResults|TerminalResultsSurvive|Archive|JobStatusMapsPersisted|JobStatusMappings)' -count=1 -v
# PASS: one storage and five application top-level tests.
# Covers expiry/pruning/permissions, removed repositories, final message/detail
# preservation, explicit directory precedence, corrupt local state, warnings,
# and lifecycle mappings. Mapping cases were subsequently extended below.

go test ./cmd/peeragent -run 'Test(AuthRecovery|OutputLimitRecoveryBudget|OutputLimitWithoutSession)' -count=1 -v
# PASS: four top-level tests. Shared budget, one auth continuation, no-session
# behavior, unchanged execution settings/session, interruption/refusal stops.

go test ./cmd/peeragent -run '^TestRecoveryCLIAndRemovedWorktreeResult$' -count=1 -v
# PASS: one top-level test with five incident scenarios. Runs a fake Codex
# executable, verifies persisted classifications/resume argv, waits for owned
# workers to exit, removes job state, and reads archived results from elsewhere.
# Initial failure was test cleanup deleting its own worker identity too early;
# nesting the worker-owning subtest before deletion fixed the test. Production
# assertions did not change.

go test ./internal/jobs ./internal/app -run 'Test(IsTerminalStatus|JobStatusMapsPersisted|JobStatusMappings|WorkerExitPreservesTerminalWinner|Finish|Cancel|TurnInbox|PrepareTurnInbox)' -count=1 -v
# PASS: one storage and thirteen application top-level tests. Includes the
# extended auth/interrupted lifecycle mappings, worker-exit precedence,
# cancellation races/idempotence, inbox cancellation and pending-heavy metadata.

go test ./internal/codex -run '^TestCodexRecoveryDiagnosticsAndTerminalPrecedence$' -count=1 -v
# PASS after tightening diagnostic prefix handling.

go test ./internal/jobs ./internal/app -run 'Test(IndexConcurrentWritesAndPermissions|ResumeAcknowledgesOnlyAcceptedTargetTurn)$' -count=1 -v
# PASS: two top-level tests, including auth/interruption inbox reservations.

go test ./internal/app -run '^TestTerminalResultsSurviveRemovedJobDirectory$' -count=1 -v
# PASS after correcting the fixture to use null final_message on resumable stops.

go vet ./internal/codex ./internal/app ./internal/jobs ./internal/result ./internal/executil ./cmd/peeragent
# PASS, no diagnostics.

scripts/build.sh
# PASS, Linux development binary built; --help smoke passed.

GOOS=windows GOARCH=amd64 go build -o /storage/scratch/peeragent/robustness-2026-10-24/peeragent-windows.exe ./cmd/peeragent
# PASS, Windows cross-compilation only. Native Windows/macOS not run.

git diff --check
cmp skills/peer/SKILL.md plugin/skills/peer/SKILL.md
git diff --name-only -- '*.go' | xargs gofmt -l
# PASS; new Go files also formatted explicitly with gofmt.
```

## Release handoff

No version bump, committed-platform-binary refresh, tag, release, or push was
performed. Review and land this source first. The release owner should refresh
committed binaries using the existing build-binaries workflow or
`scripts/build-committed.sh`, then follow `scripts/bump.sh <next-version>` for
metadata/package synchronization, its one full validation run, commit, tag and
push. Publish release assets through the existing release workflow or
`make publish-release VERSION=<next-version>` as appropriate.

Limits: jobs completed before snapshots were introduced cannot be recovered
after their original files are deleted. Snapshot copies expire after 30 days;
raw logs still need their original directory. Removed jobs are not added to
`--jobs --all` or multi-job status/wait-any listings. No host deployment is
needed beyond releasing/updating the plugin. No additional work was backlogged.

## Review fix round

Source: `fbae004610b4319f0e65b4e3f10ed970c8ab7be4`.

The launching session relayed updated direction from Nathan: retain complete
archived results and the existing live summaries. The review's proposed payload
projection and no-message-summary change were dropped before verification and
commit. Archive contents and retention remain as implemented above.

The two retained findings are fixed:

- Archived-directory lookup cleans paths, then compares them case-insensitively
  on Windows and default macOS filesystems, and exactly on Linux.
  `TestArchivedDirectoryLookupUsesPlatformCaseRules` checks exact, differently
  cased and unrelated directory names against the current platform's rule.
- Archive loading reports a separate found flag. A successful `--status` lookup
  sets its command exit metadata to zero even when the target exited with four;
  genuinely missing jobs still exit four, and result reads retain the target
  exit code. `TestArchivedStatusSeparatesTargetExitCodeFromLookup` exercises
  both CLI exits and checks that result retrieval preserves the target code.

Verification ran once on this source:

```sh
go test ./internal/app ./cmd/peeragent -count=1 -json
# PASS: internal/app 56 and cmd/peeragent 73 top-level tests, 129 passed total.
# Includes both new regressions and all existing tests in the touched packages.

go vet ./internal/app ./cmd/peeragent
# PASS, no diagnostics.

git diff --check
git diff --name-only -- '*.go' | xargs gofmt -l
# PASS, no whitespace or formatting findings.
```

This Linux run did not exercise native Windows/macOS filesystem behavior.
No full repository suite, release, push, host deployment, scratch folder or
persistent build output was needed for this fix round. No blockers remain.

## Closure (2026-10-24)

Released in v0.12.3. Sol xhigh review findings: archive payload projection declined by the owner (credential or
prompt exposure in our own tools' archives is never an issue); Windows path case and archived status exit fixed in
fbae004. Deferred, rare configuration: archived-path matching also ignores case on case-sensitive macOS volumes,
which can break explicit-directory precedence there.
