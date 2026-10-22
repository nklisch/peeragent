# Job messaging

The launcher sends a reply with `peeragent tell <job-id> [--cwd DIR] "<text>"`.
The usual job index resolves the directory unless `--cwd` overrides it. A reply
appends `{time, from, text}` to `inbox.jsonl`. `from` uses the caller's Herdr pane,
its peeragent job identity, or `launcher`. A finished job refuses replies and
suggests `--resume <agent-session>`.

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

A resume includes unread entries from previous jobs for the same agent/session,
in time order, and acknowledges the included snapshot after successful worker
startup. Failed startup consumes nothing. Later arrivals remain unread. Capacity
continuations include and acknowledge their own unread inbox before execution.
Replies stay archived in the originating job's conversation after reading.

## Outbound messages

Inside a job, `peeragent say "<text>"` invokes the recorded Herdr executable with
literal arguments `agent prompt <recorded-pane> <labelled-text>`. It always puts
this header before the body:

```text
⟦peer · <agent> job <job-id> · <worktree basename>@<branch> · reply: peeragent tell <job-id> "…"⟧ <text>
```

The branch is captured at launch (`unknown` when unavailable; `HEAD` for a
detached checkout). Whitespace, control characters and header brackets in label
components are replaced with underscores. In the body, `⟦`/`⟧` become `[`/`]`,
and control characters become spaces. A body cannot create another supported
origin header. Quotes and shell syntax remain literal text. Use `--` before
message text beginning with a dash.

A message may contain at most 2,000 Unicode characters before sanitization. A
job can attempt delivery at most three times in a rolling five-minute window.
A Herdr failure still uses a slot because delivery may have occurred before the
error. Concurrent senders reserve their slots before delivery; a sender
interrupted during delivery leaves a `pending` entry that counts until that window expires. Herdr execution
has a ten-second timeout. No recorded pane/binary or a Herdr delivery failure
produces a one-line nonzero refusal naming the handback alternative. Sent and
refused attempts are timestamped in `outbox.jsonl`; refusals include a reason.

At launch, the launcher records present `HERDR_PANE_ID`, `HERDR_SOCKET_PATH`,
`HERDR_WORKSPACE_ID` values and the Herdr binary path (`HERDR_BIN_PATH`, falling back to PATH lookup) in `launcher.json`
(mode 0600). No file is created without a pane. The file is separate from public
job metadata and results. Targets still start without terminal-control variables
unless the launcher deliberately uses `--keep-terminal-env`. This interface is
for trusted local agents running as the same user; private files protect against
other users rather than restrict the account owner.

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
