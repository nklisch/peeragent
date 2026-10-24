---
id: resume-keeps-model-and-effort
kind: story
stage: review
tags: [infra, tests, docs]
parent: null
depends_on: []
release_binding: null
created: 2026-10-24
updated: 2026-10-24
---

# Resume keeps the session's model and effort

Godot-41 requested this fix after voxlar-da reported that resuming a review
without flags silently replaced its xhigh effort and chosen model with defaults.
A resume must use its most recent job's options for each omitted flag; explicit
`--model` and `--effort` independently win. No release or push is authorized.

## Design

Preserve omission through input normalization with two inheritance markers, then
resolve options from the existing session-discovery results while the launch
lock is held. Select the greatest creation time, with job ID breaking ties;
progress updates on an older job do not make it the latest turn. Reuse the same
records for inbox carryover rather than adding another scan or index format.
The saved job spec contains resolved values, so workers and automatic retries
keep them without another lookup. Revalidate an inherited option pair because
Gemini Pro does not accept medium effort.

A missing previous job keeps current defaults for omitted options and records
`resume_warning` on the job and in launch/result metadata. Explicit options
still win. An empty recorded Codex/Claude model means the target CLI's native
default; preserve that absence rather than inventing a model. Existing corrupt
storage errors remain infrastructure errors in session protection. No new
refusal or index migration is introduced.

## Acceptance and verification

Focused checks cover latest-job selection across indexed directories, all four
targets, independent overrides (including values equal to defaults), unknown
sessions, launch and terminal metadata, persisted settings and actual target
arguments through the fake CLI. Run tests for touched packages, vet and one
Windows cross-build directly as light Go work. Native Windows/macOS execution
is unavailable here. The owner reviews and releases this unit.

## Verification log

Tested implementation: `5fc3b4a85b9b3219bedaed3e29d781ef499645dc`.
The subsequent evidence commit changes only this story.

Verified on Linux with `go version go1.26.4 linux/amd64`. Counts below are
measured from Go's JSON test events; top-level counts exclude subtests.

```sh
go test -json ./internal/app ./cmd/peeragent -run 'TestResume(InheritsLatestJobOptions|OverridesAndCombinedValidation|OptionsCLI)$' -count=1
# PASS: 3 top-level tests / 22 including subtests; zero failures or skips.

go test -json ./cmd/peeragent ./internal/app ./internal/input ./internal/jobs ./internal/result -count=1
# PASS: all five touched packages; 240 top-level / 462 including subtests.
# cmd/peeragent: 76 / 156; internal/app: 59 / 135;
# internal/input: 70 / 101; internal/jobs: 30 / 65; internal/result: 5 / 5.
# Zero failures or skips. Other packages' test suites were not rerun.

go vet ./...
# PASS, no diagnostics.

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /storage/scratch/peeragent/resume-keeps-model-and-effort/peeragent-windows.exe ./cmd/peeragent
# PASS, cross-compilation only.

git diff --check
cmp skills/peer/SKILL.md plugin/skills/peer/SKILL.md
gofmt -l cmd/peeragent/main.go cmd/peeragent/resume_options_test.go internal/app/jobs.go internal/app/progress.go internal/app/service.go internal/app/session.go internal/app/resume_options_test.go internal/input/delegation.go internal/input/input.go internal/jobs/store.go internal/result/result.go
# PASS, no whitespace, skill-copy or formatting differences.
```

Three new regression tests cover inheritance across all four targets, partial
and complete overrides, invalid inherited Gemini combinations, native model
defaults, missing-session fallback, saved job specs, launch/status/terminal
metadata and the fake Codex's actual arguments. CLI checks exercise blocking
and async launches and inherit an earlier resumed turn's override.

The implementation reuses session discovery; it adds no directory scan or
index migration. No performance measurement or real target-model call was made.
Native Windows/macOS execution and macOS compilation remain unverified. No
release, push, installed-plugin update, host deploy or committed binary refresh
was performed; distribution belongs to the release owner. Nothing was backlogged
or deferred. This story is the consolidated result log; disposable test events
and the Windows build artifact were removed after recording their results.
