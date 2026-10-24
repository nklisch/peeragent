---
id: say-refuses-contract-clarification-delivery
kind: story
stage: done
tags: [infra]
parent: null
depends_on: []
release_binding: v0.13.0
created: 2026-10-24
updated: 2026-10-24
---

# A running job cannot deliver a contract clarification to its launcher

On 2026-10-24, a running implementation job invoked `peeragent say <contract clarification>` from its assigned Voxlar worktree. Earlier questions in the same job had delivered and received answers through `peeragent inbox`. This invocation exited 1 with the exact message:

```text
say: Herdr refused or could not deliver the message; use your handback instead
```

Expected: deliver the clarification, or distinguish an unavailable launcher from a rejected destination and give a supported recovery. Observed: no cause or recovery was supplied; the job could not establish whether its required contract question reached the launcher. `peeragent inbox` continued to run successfully.

The active wrapper resolved to the installed peeragent 0.12.2 plugin on Linux, with a Codex implementation job and a Herdr launcher. Reproduction is not yet established beyond this occurrence. A launcher state change or transport issue remains possible; this is not a confirmed product defect. No terminal or desktop delivery was attempted.

Workaround: continue independent authorized work, retain the unresolved clarification, and report it in the final handback. Prompt text and private launcher identifiers are intentionally omitted.

The [safe launcher delivery change](say-splices-into-a-half-typed-launcher-message.md) replaces Herdr prompt input with
native inbox delivery. Transport failures now retain the question as queued and expose the underlying connection or
write error in its receipt and result conversation. This provides a supported recovery; the original Herdr refusal's
cause remains unconfirmed.

A later coordination question on 2026-10-24 from the same assigned Voxlar worktree returned exit 1 and the same exact refusal. `peeragent inbox` still exited successfully immediately afterward. The message body is omitted; the implementation handback carries the overlap instead. No alternate delivery route was attempted.

Another assigned Voxlar implementation job reproduced the same non-delivery on 2026-10-24 while asking how to
dispose of focused-test failures outside its mechanical lint correction. The operation was `peeragent say
<verification-scope clarification>`; the prompt body is omitted. The wrapper printed the same exact refusal above,
and an immediately following `peeragent inbox` succeeded without an answer. The installed wrapper was peeragent
0.12.2. This is another observed non-delivery, with no isolated transport reproduction or established cause. The
job retained the unresolved question for its committed handback and used no alternate delivery route.

## Closure (2026-10-24)

Released in v0.13.0 with the inbox delivery above: a launcher without a native inbox now queues the clarification
with a receipt instead of refusing it.
