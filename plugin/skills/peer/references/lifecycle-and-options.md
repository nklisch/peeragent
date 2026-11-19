# Peeragent Lifecycle, Messaging, and Options Reference

Detailed behavioral contracts referenced from [SKILL.md](../SKILL.md).

## Cache Wrapper Forwarding

Claude Code cache wrappers follow the active install in its
`installed_plugins.json` record, including when an existing session keeps an
older cache path on `PATH`. The wrapper needs Python 3 or `jq` to read the
record and otherwise keeps its current install. Explicit `PEERAGENT_BIN` and
source checkout paths retain precedence. Codex and Antigravity have no
equivalent versioned cache-forwarding record, so their wrapper selection uses
the installed plugin directory directly.

## Z.AI Configuration Smoke Test

Quickly verify whether the Z.AI target is configured before delegating:

```bash
pi --list-models zai | grep -w 'glm-5.2'
pi --provider zai --model glm-5.2 --thinking high --no-session --no-tools -p 'Reply with OK.'
<resolved-peeragent-bin> --agent zai --text 'Reply with OK and do not edit files.'
```

If the Pi smoke test fails, configure Pi with `ZAI_API_KEY` or `/login` for the
ZAI provider and retry.

## Messaging Contract Details

- Both `tell` and `say` accept `--prompt-file <path>` alone or with positional
  text: trim each source and join positional text first, then file content,
  separated by a blank line. File paths resolve relative to the caller,
  independently of `tell --cwd`. Messaging commands do not read stdin.
- Outbound `say` messages carry a mandatory origin and reply header. Without a
  safe native channel (including Herdr-only or unsupported launchers), the
  message is retained as `queued` in the job's `--result` conversation with no
  automatic retry. Native submission has no acceptance/read acknowledgement.
- `--status` shows mailbox counts and unread ages; `--result` includes the
  ordered conversation after the agent message. Resumes carry unread replies
  into their prompt. `--no-preamble` suppresses standard execution/messaging
  instructions and sends task text only.
- Replies carried in a prompt are reserved by that attempt and hidden from its
  inbox, unread notice, and status count while it runs. Target output and a
  non-capacity outcome mark them read. Startup failure, cancellation, and
  capacity refusal release them to unread; a dead worker's unresolved
  reservation is released on the next mailbox read.
- `tell` confirms persisted delivery (not that the job has read it), e.g.:
  `delivered to job <id> (<agent>, <status>); unread 3, oldest 2h14m; the job reads its inbox between long steps`.
- Status and job listings expose `oldest_unread_at` and `oldest_unread_age`
  alongside `inbox_unread`. Ages use whole seconds below a minute, then whole
  minutes; future timestamps display `0s`; missing timestamps display
  `unknown`; an empty inbox omits oldest-unread fields.
- New jobs record `messaging_enabled` in `job.json`. For older jobs without
  that field, a saved messaging preamble or outbound `say` entry infers support
  (`messaging_source`: `inferred from prompt` or `inferred from outbox`). Only
  jobs with neither evidence source receive the pre-messaging warning;
  `--no-preamble` jobs receive a specific missing-preamble warning.
- Labels and environment stripping help well-meaning agents avoid accidental
  unlabelled input, not act as an adversarial boundary.

## Async Recovery and Completion Details

- Under `--async --agent codex --retry-capacity N`, a Codex `max_output_tokens`
  (`output_limit`) or login (`auth_required`) stop consumes one of the shared N
  additional attempts and resumes immediately once (at most one automatic
  output-limit continuation and one automatic login continuation per job).
  `--capacity-backoff` applies only to capacity stops. `metadata.capacity_retries`
  counts all additional attempts.
- A resume refuses with exit `2` while another job for that agent/session has a
  live recorded process; use `--allow-concurrent-session` only for intentional
  overlap.
- For streaming targets (Codex and Gemini/`agy`), session IDs are recorded and
  indexed as soon as the stream initializes, including while running and after
  cancellation. A terminal event or explicitly final message starts a
  five-second grace period (`final message received, process still alive` in
  status); after the grace period peeragent ends only its own target process
  tree and uses the terminal event's outcome.
- `--result` preserves completed messages without truncation, or provides
  `metadata.message_path` for a message from an earlier attempt.

## Job Storage and 30-Day Compact Index

- Workspace-write Codex jobs use `.peeragent/jobs/` at the worktree root so the
  target can update its inbox. Other Git jobs use `<git-dir>/peeragent/jobs/`
  (separate per linked worktree). Outside Git, jobs use `.peeragent/jobs/` in
  the target directory. Controls, messaging, listings, resumes, and indexed
  lookups find both locations and leave records in their original folders.
- Compact job metadata is retained in the per-user index for 30 days (up to
  10,000 entries, summaries capped at 600 Unicode characters). Removed-worktree
  jobs stay discoverable through `--jobs --all` and return compact fallbacks on
  `--status` and `--result` with `metadata.indexed_only: true` and
  `metadata.index_warning` (no final message, logs, mailbox, or details are
  reconstructed). `--wait` returns last-known indexed history immediately.
- Resume retains the saved model/effort from the most recent peeragent job for
  that session; supply `--cwd <replacement-worktree>` when the original
  directory was removed. If no prior job record is found, omitted options fall
  back to current defaults with `metadata.resume_warning`.
- Full terminal result JSON keeps its separate 30-day archive and takes
  precedence over the compact fallback.
