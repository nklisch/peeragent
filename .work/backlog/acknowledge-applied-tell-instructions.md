---
id: acknowledge-applied-tell-instructions
kind: story
tags: [messaging]
created: 2026-10-26
updated: 2026-10-26
---

# Acknowledge which told instructions a job applied

voxlar-a6 (2026-10-26): `peeragent tell` to a running Codex job was delivered ("unread 1"), but the added scope (an
extra `fail_start` fix) was silently not acted on, and the hand-back didn't mention it (job
`20261026T122340Z-350f05a2`). There is no acknowledgement that a told instruction was read and applied, or set aside.
Workaround: brief additions as their own round, or check the hand-back for each told item.

Expected: a told message gets an explicit acknowledgement in the job's result, listing each told instruction as
applied, declined (with a reason) or unread, so the host can see what was dropped.
