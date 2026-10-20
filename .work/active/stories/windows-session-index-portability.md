---
id: windows-session-index-portability
kind: story
stage: review
tags: [infra, tests]
parent: null
depends_on: []
release_binding: null
created: 2026-10-19
updated: 2026-10-19
---

# Keep session directory lookup portable on Windows

## Evidence and scope

The Test workflow run `39346293889`, for released source `08c031f`, failed on
Windows only. Five input tests compared normalized absolute paths against Unix
fixture spelling (`/repo`, `/other`, `/explicit`). On the Windows runner those
paths resolve to its `D:` drive. Absolute normalization is intended and already
documented by `docs/CONTRACT.md`; retain it and assert the platform's exact
`filepath.Abs` result without relaxing directory precedence or normalization.
The indexed-directory override test lives in `internal/input/lookup_test.go`.

The same run also reported five `Access is denied` errors while concurrent
session-index writers replaced the same destination. Unique temporary files
avoid partial content, but Windows can briefly deny replacement while a handle
is open. Add Windows-only retries for access-denied and sharing-violation errors,
bounded to ten attempts and 450ms of requested delay. Keep Unix rename behavior
unchanged. Permanent and unrelated errors still reach the existing warning path.
No index contents, permissions, or public flags change. Existing Unix permission
assertions remain guarded from Windows execution.

This is a follow-up for the planned patch release. Native Windows execution
belongs to CI after review; local verification compiles its tests and checks
Linux/Windows/macOS with vet. Do not push, tag, bump versions or refresh binaries.

## Implementation and verification

Five tests now compare against `filepath.Abs` of the fixture path, preserving
their exact directory assertions, canonical option checks, and explicit override
precedence. Production directory normalization is unchanged. Index replacement
uses a Windows helper that retries wrapped access/sharing errors and otherwise
returns the original error. The Unix helper delegates directly to `os.Rename`.
Windows-specific tests cover transient recovery, exhaustion, unchanged rename
paths, and immediate return for unrelated errors, without actual sleeps.

Verified on Linux with Go 1.26.4:

- Focused input normalization, lookup/override and concurrent index tests: passed.
- `make test` and `go vet ./...`: passed once on the frozen code.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...`: passed.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c` for every package: passed.
- `GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go vet ./...`: passed.
- Formatting and `git diff --check`: passed.

Reused the unit's copied warm cache, including existing Windows and Darwin
dependencies. An unnecessary queued heavy waiter was stopped before it started
after confirming those cached targets; only that positively owned waiter was
stopped. Warm cross checks then completed as light work. No verification process
remains active. Native Windows and macOS execution was not available locally;
Windows contention and its new unit tests require the normal CI runtime check.
No installed peeragent was run, and no release operation was performed.
