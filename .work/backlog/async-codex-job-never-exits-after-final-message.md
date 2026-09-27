---
id: async-codex-job-never-exits-after-final-message
kind: story
tags: [reliability]
parent: null
depends_on: []
created: 2026-09-26
updated: 2026-09-27
---

# An async Codex job can stay "running" for hours after the agent finished

Observed 2026-09-26 while Orogen's client-refinement run used peeragent 0.7.0 with Codex CLI 0.157.0 (`--async --agent codex --model sol --effort high`, job `20260926T223810Z-fa046c5c`, worktree `/storage/orogen-worktrees/ux-uf2`).

- **The agent finished early.** The Codex session log (`~/.codex/sessions/2026/09/26/rollout-2026-09-26T16-38-10-01a0dfde-…jsonl`) ends with the agent's complete final report and a `task_complete` event, about 20 minutes into the run. The work was committed (`9a319db9`), and the worktree was clean.
- **peeragent never saw the end.** `--status` kept reporting `running` for about 3 hours and 40 minutes. The `peeragent --job-run` process (and its `codex` child, with no children of its own) stayed alive, and the host's `--wait` never returned.
- **Recovery:** read the final message from the Codex session log, then run `--cancel`, which ended the process cleanly.

**Second occurrence (2026-09-27).** Job `20260927T041659Z-65c7a364` (worktree `/storage/orogen-worktrees/ux-ut4d`, same peeragent and Codex versions): the session log's `task_complete` arrived at 05:39Z, and the job still reported `running` 38 minutes later. The same recovery (read the log, then `--cancel`) worked.

**Hypothesis, not verified:** Codex `exec` doesn't exit after its final `task_complete` in some condition, for example a lingering internal auto-review turn (the log's last entries were an approvals-reviewer "allow" decision). peeragent's job runner waits on process exit, not on the final event.

**What would help:**
- treat a terminal `task_complete` or final agent message in the stream as completion, with a bounded grace period before terminating the child; or
- surface "final message received, process still alive" in `--status`, so the host can collect the result.

**Impact:** a host orchestrating several parallel peers can wait indefinitely on a finished unit. Other async Codex jobs in the same run exited normally, so this is intermittent.
