# Job messaging

The launcher sends a reply with `peeragent tell <job-id> [--cwd DIR] "<text>"`.
The usual job index resolves the directory unless `--cwd` overrides it. A reply
appends `{time, from, text}` to `inbox.jsonl`. `from` uses the caller's Herdr pane,
its peeragent job identity, or `launcher`. A finished job refuses replies and
suggests `--resume <agent-session>`.

`tell` confirms persisted delivery, for example:
`delivered to job <id> (<agent>, <status>); unread 3, oldest 2h14m; the job reads its inbox between long steps`.
Delivery does not mean the job has read the reply. The count and oldest unread
age come from the delivery lock; the job may read the reply immediately afterward.
Status and job listings expose `oldest_unread_at` and `oldest_unread_age` alongside
`inbox_unread` (in metadata for single-job results). Text includes the age and
context when replies remain unread. Ages use whole seconds below a minute,
then whole minutes; future timestamps display `0s`. Missing timestamps display
`unknown`; an empty inbox has no oldest-unread fields. Entries reserved for a
running prompt are excluded until their reservation is released.

New jobs record `messaging_enabled` in `job.json`, and launch JSON reports that same saved state.
`tell` to a job without messaging prints `queued for the next --resume (N pending)`
when its agent session is known. Replies stay in the job folder and are prepended to the next
resume prompt, clearly labelled as messages sent while the job ran. If no session is known,
`tell` exits non-zero and says the message was not queued; retry after the session is recorded.

For older jobs without that field, a saved messaging preamble or an outbound `say` entry proves messaging
support. Controls report `messaging_enabled: true` and `messaging_source` as
`inferred from prompt` or `inferred from outbox`, and show that source in text
and delivery confirmations. Explicit recorded flags remain authoritative.
Only jobs with neither evidence source get the pre-messaging warning; replies
are queued for `--resume` when a session is known. `--no-preamble` jobs get
a specific missing-preamble warning. Saving a legacy job's lifecycle preserves
an absent marker so later controls can still infer support from evidence.

Inside a job, `peeragent inbox` emits unread replies as JSONL in arrival order.
It holds the job lock through printing and marking them read. Concurrent tells
stay in the next unread batch if they arrive after that lock was acquired.
Failed output leaves the batch unread. Every peeragent command inside a job
first prints this notice on stderr when replies are pending:

```text
peeragent: N unread message(s): run peeragent inbox
```

Every target receives `PEERAGENT_JOB_ID` and the absolute `PEERAGENT_JOB_DIR`.
Standard prompts tell it to check the inbox after long steps and before handback:

Use `peeragent say` only to ask a question you can't settle yourself, never for status or progress updates.
Your final handback carries the final status, so don't repeat it in a message.
After asking a question, watch your inbox for the answer: run `peeragent inbox` every few minutes, between steps,
until it arrives, and keep doing independent work meanwhile.

`--no-preamble` sends task text without the standard execution or messaging
instructions; the mailbox and unread notices remain available.

A resume records references to previous jobs for the same agent/session. Each
attempt reserves a fresh snapshot of their unread entries before putting them
in its prompt. Directory aliases are deduplicated by recorded job ID. Capacity
continuations also reserve replies from their own inbox.

Replies have three states: unread, reserved by an attempt, and read. Reserved
replies are hidden from inbox output, unread notices and status counts, so the
running target does not receive the prompt's carried reply again through
`peeragent inbox`. Later replies remain independently readable. Reading them
cannot accidentally acknowledge an earlier reserved reply.

A non-capacity execution outcome after target output accepts the attempt and
marks its reserved entries read. Target startup failure, cancellation and
capacity refusal release the reservation, making those replies unread again
for the next attempt or a later resume. A worker process alone is insufficient.
An attempt's terminal outcome is recorded before settling source mailboxes,
so repeated settlement cannot consume a reply twice. Replies stay archived in
the originating job's conversation regardless of their delivery state.

On each mailbox read, reservations belonging to a dead worker without a terminal
attempt outcome are released. A recorded acceptance is committed even if the
worker exited before finishing mailbox settlement. Recovery uses the attempt's
recorded process ID and never signals a process. A crash before an acceptance
record may therefore cause a later attempt to receive the reply again; the
reservation prevents duplicate delivery within the running attempt.

## Outbound messages

Inside a job, `peeragent say "<text>"` submits to the recorded Claude Code native
inbox, or the existing mailbox when its launcher is itself a peer job. It never writes terminal input, so it cannot splice text into an unsent
draft. It always puts this header before the body:

```text
⟦peer · <agent> job <job-id> · <worktree basename>@<branch> · reply: peeragent tell <job-id> "…"⟧ <text>
```

The branch is captured at launch (`unknown` when unavailable; `HEAD` for a
detached checkout). Line-breaking whitespace, control characters and header
brackets in label components are replaced with underscores. Explicit direction
formatting is removed from labels and bodies (U+200E/U+200F, U+202A–U+202E,
U+2066–U+2069). In the body, `⟦⟧`, `〚〛` and `⟬⟭` become `[]`; Unicode vertical
space, including U+2028/U+2029, and control characters become spaces. These are
specific visual normalizations, not general Unicode spoof prevention. Quotes
and shell syntax remain literal text. Use `--` before message text beginning
with a dash.

A message may contain at most 2,000 Unicode characters before sanitization. A
job can attempt delivery at most three times in a rolling five-minute window.
A native socket failure still uses a slot because delivery may have occurred before
the error. Concurrent senders reserve their slots before delivery; an interrupted
sender leaves a `pending` entry until that window expires. Delivery has a
ten-second bound. Validation failures still return a nonzero refusal.

Without a safe native channel, or when transport fails, the message is retained
as `queued` in `outbox.jsonl`. The successful command's receipt explicitly says
queued and names `--result <job-id>` as the recovery path: its conversation
contains the question and reason. There is no automatic retry, since a failed
write can have reached the inbox. Continue independent work and carry unanswered
questions in the handback. Successful transport records `sent` and prints
“submitted”; the native protocol does not acknowledge acceptance or reading.
Sent, queued and refused entries are timestamped. Pending reservations and their
outcomes share an ID; readers fold that journal without rescanning after delivery.

At launch, an identifiable Claude Code launcher records
`CLAUDE_CODE_MESSAGING_SOCKET`, `CLAUDE_CODE_MESSAGING_TOKEN` and
`CLAUDE_CODE_SESSION_ID` as private `claude_socket`, `claude_token` and
`claude_session` fields in `launcher.json` (mode 0600). Delivery sends one JSON
auth line and one JSON user line; optional `session_id` binds the latter to the
captured conversation. The runtime enqueues it separately from its terminal
editor. A launcher that is itself a peer job instead records its job id and directory
under `peer_job` and receives the labelled message through its existing `inbox`.
It reads the question between long steps; if messaging was disabled, the receipt
states that it is queued for that launcher's next `--resume`. Finished or missing
parent jobs leave the question queued locally. Native addresses inherited by a
peer job are never used. A standalone Codex marker excludes an inherited Claude
address; such a launcher queues. No recipient is guessed from a pane or worktree name.

Legacy Herdr fields are ignored when decoding older files; new launches do not
record terminal addresses. Herdr has a
visual notification channel; it does not feed the agent's inbox. Neither activity
status nor a terminal snapshot provides an atomic empty-draft submission, so
there is no terminal-input fallback. Existing jobs with only Herdr records queue.
Windows and any other platform without this Unix socket transport queue when
there is no parent peer mailbox;
`tell`, `inbox` and result conversations continue to work.

The preamble and labels define the supported path, not an adversarial boundary.
Jobs run as the same user and can read the private launcher file or contact local
sockets directly. Private files protect against other users, not the account owner.

## Visibility

Single-job status metadata and multi-job status rows show `inbox_unread`,
`outbox_sent`, and optional `last_message_at`. The last time includes replies and
sent/refused outbound attempts. Text tables expose the same values. `--result`
adds an ordered `conversation` after `details`, with `direction`, `time`, `from`,
`text` and outbound `status`/`reason`. Text output puts it after the agent message.
Reading status or results does not mark replies read.

Each command snapshots a job's mailbox once and shares that snapshot with its
unread notice and output. Mutating inbox/say commands take that snapshot under
the operation's lock. Counts are computed from history; no incremental counter
state is maintained.
