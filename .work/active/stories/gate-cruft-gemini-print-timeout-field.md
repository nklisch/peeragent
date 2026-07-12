---
id: gate-cruft-gemini-print-timeout-field
kind: story
stage: implementing
tags: [cleanup]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: cruft
created: 2026-07-12
updated: 2026-07-12
---

# Replace dead Gemini PrintTimeout option with a constant

## Confidence
Medium

## Category
Never-configured option field

## Location
`internal/gemini/exec.go:20`

## Evidence
```go
type Options struct {
    // ...
    PrintTimeout string
}
```

No production or test caller sets `PrintTimeout`; every invocation uses the same 15-minute fallback.

## Removal
Remove the unused option field and replace the fallback branch with a package-level `agyPrintTimeout = "15m"` constant passed unconditionally. Preserve argv behavior and tests.

## Design

- Edit only `internal/gemini/exec.go` and its tests.
- Remove `Options.PrintTimeout`; add one package constant; keep `--print-timeout 15m` in every generated argv.
- Update the argv test to assert the unchanged value and run Gemini plus full Go tests.
- This is behavior-preserving cleanup; no documentation or public contract changes.
