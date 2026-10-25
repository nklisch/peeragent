---
id: resume-codex-jobs-after-an-auth-stop
tags: [codex, recovery]
created: 2026-10-25
updated: 2026-10-25
---

# Resume Codex jobs after an auth stop

A Codex job that stops with `auth_required` (including `401 token_revoked`) ends terminal today
(`internal/jobs/store.go` `StatusAuthRequired`), and the host has to notice and resume it. Nathan (2026-10-25,
relayed verbatim by voxlar-85): "codex is always signed in again when it ends on an auth - tell the others, just retyr
immediately unless you see 3-5 failures in arrow immediatily it's fine".

Outcome: an async Codex job that stops on auth resumes its own session at once under the same job id, the way
`--retry-capacity` resumes after a capacity stop but with no backoff, and gives up after three consecutive auth stops
with the existing `auth_required` result. A resume that makes progress resets the count. Acceptance: a test where the
first attempt ends `token_revoked` and the resume succeeds reports success under one job id with the retry counted; a
test with three consecutive auth stops ends `auth_required`.
