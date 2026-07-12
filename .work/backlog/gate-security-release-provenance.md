---
id: gate-security-release-provenance
kind: story
tags: [security, infra]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-07-12
---

# Add signatures and provenance to release binaries

## Severity
Low

## Domain
Dependencies / supply chain

## Location
`scripts/release.sh:43`, `scripts/build-committed.sh:22`, `.github/workflows/build-binaries.yml:55`

## Evidence
```sh
sha256sum peeragent_*.tar.gz > checksums.txt
go build -trimpath -buildvcs=false -ldflags="-s -w"
```

Release archives and committed plugin binaries have checksums but no signatures, embedded source revision, or build provenance.

## Remediation direction
Publish signed checksums and provenance and embed a source revision in binaries.
