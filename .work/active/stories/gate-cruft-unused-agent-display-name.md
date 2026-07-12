---
id: gate-cruft-unused-agent-display-name
kind: story
stage: review
tags: [cleanup]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: cruft
created: 2026-07-12
updated: 2026-07-12
---

# Remove unused CLI agentDisplayName helper

## Confidence
High

## Category
Dead function

## Location
`cmd/peeragent/main.go:365`

## Evidence
```go
func agentDisplayName(req input.Request) string {
    switch agentID(req) {
    // ...
    }
}
```

The helper has zero call sites after result construction moved into `internal/app`; the live application helper is used instead.

## Removal
Delete the unused function. Keep `agentID`, which remains live elsewhere. Run CLI tests and build.

## Implementation Notes
- Removed only the unused `agentDisplayName` helper from `cmd/peeragent/main.go`.
- Kept `agentID` and all surrounding request/result behavior unchanged.

## Verification
- `go test ./cmd/peeragent`
- `go build ./cmd/peeragent`
