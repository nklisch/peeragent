---
id: completion-notification-or-detached-wait-exit-file
kind: story
tags: [infra]
created: 2026-10-29
updated: 2026-10-29
---

# Notify a launcher when a detached job completes

The 2026-10-29 friction sweep reports from voxlar-8e and godot-34 request a
completion notification or a detached `--wait --exit-file` mode. An attached
waiter can disappear across a session stop or machine restart, leaving the
launcher without a completion signal even though the job metadata survives.

Choose a supported completion mechanism and define restart, terminal failure,
and output-file behavior before implementation. Keep one terminal result and
preserve existing stdout JSON contracts. This is outside the friction batch's
stale-worker recovery and mailbox-history work; no notification or exit-file
mode was added there. A waiter unable to find peeragent is separately resolved
by the `~/.local/bin/peeragent` shim.
