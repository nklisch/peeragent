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

New jobs record `messaging_enabled` in `job.json`. For older jobs without that
field, a saved messaging preamble or an outbound `say` entry proves messaging
support. Controls report `messaging_enabled: true` and `messaging_source` as
`inferred from prompt` or `inferred from outbox`, and show that source in text
and delivery confirmations. Explicit recorded flags remain authoritative.
Only jobs with neither evidence source get the pre-messaging warning; delivery
still succeeds and replies are carried on `--resume`. `--no-preamble` jobs get
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
Standard prompts tell it to check the inbox after long steps and before handback,
and to use `say` for questions it cannot settle or short status the launcher
requested. `--no-preamble` sends task text without the standard execution or
messaging instructions; the mailbox and unread notices remain available.

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

Inside a job, `peeragent say "<text>"` invokes the recorded Herdr executable with
literal arguments `agent prompt <recorded-pane> <labelled-text>`. It always puts
this header before the body:

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
A Herdr failure still uses a slot because delivery may have occurred before the
error. Concurrent senders reserve their slots before delivery; a sender
interrupted during delivery leaves a `pending` entry that counts until that window expires. Herdr execution
has a ten-second timeout. No recorded pane/binary or a Herdr delivery failure
produces a one-line nonzero refusal naming the handback alternative. Sent and
refused attempts are timestamped in `outbox.jsonl`; refusals include a reason.
Pending reservations and their delivery outcomes share a message ID in this
append-only journal. Readers fold outcomes into one message per ID, keeping
concurrent sends intact without a second mailbox snapshot during delivery.

At launch, the launcher records present `HERDR_PANE_ID`, `HERDR_SOCKET_PATH`,
`HERDR_WORKSPACE_ID` values and the Herdr binary path (`HERDR_BIN_PATH`, falling back to PATH lookup) in `launcher.json`
(mode 0600). No file is created without a pane. The file is separate from public
job metadata and results. Targets still start without terminal-control variables
unless the launcher deliberately uses `--keep-terminal-env`.

The labels and environment strip help well-meaning agents avoid accidental
unlabelled input. A deliberately misbehaving job runs as the same user and can
read the private launcher file, invoke Herdr directly without labels, or change
its own state. The preamble and `say` define the supported path, not an
adversarial boundary. Private files protect against other users; they do not
restrict the account owner.

Windows supports the mailbox and shared status/result path. `say` records a
refusal and reports `no launcher channel on this platform; use your handback
instead`. Linux and macOS use the Herdr transport when a launcher is recorded.

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
