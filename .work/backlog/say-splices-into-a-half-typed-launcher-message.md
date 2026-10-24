---
id: say-splices-into-a-half-typed-launcher-message
kind: story
tags: [reliability]
parent: null
depends_on: []
created: 2026-10-24
updated: 2026-10-24
---

# A job's `say` splices into the person's half-typed message in the launcher pane

`peeragent say` delivers to the job's recorded launching Herdr pane. When the person is typing a message to the
launcher's agent at that moment, the job's labelled message lands inside their draft. The draft is sent cut off, with
the job's message in the middle or at the end, and the person has to finish their sentence in a second message.

Observed in a Claude Code launcher session on 2026-10-24 (Voxlar's generation owner, about eight Codex jobs running):
- twice in one hour, Nathan's typed message arrived cut mid-word with a `⟦peer · codex job …⟧` message appended after
  it. Examples: "…the fog should feel like a natually lif⟦peer · codex job 20261024T033639Z-4f1b88e3 …⟧ Can the tools
  owner check…", completed by a second message "lifting curtain"; then "…what is this epic⟦peer · codex job
  20261024T033639Z-4f1b88e3 …⟧ Could the tools owner check…", followed by "again".
- An earlier occurrence the same day split a sentence the agent then misread as a separate instruction ("pursue it
  again").

Expected: a job's message never lands inside the person's unsent input. It waits until the input is empty or the
draft is sent, or it arrives through a channel separate from the person's input.

Known versions: peeragent 0.12.3, Codex target, Claude Code launcher on Linux in Herdr. The root cause isn't confirmed:
it could be pane input injection racing the person's keystrokes, or the launcher harness merging queued input. The
person's own words get attributed to a mixed message, so the launcher's agent can misread an instruction.

Workaround: none on the job side beyond sending fewer messages; the person retypes the cut part.

## Delivery design and channel findings (2026-10-24)

The sweep task authorizes replacing unsafe pane input with a separate channel and retaining messages locally when
none exists. The change stays in peeragent. It adds no daemon, background retry, harness patch or terminal probing.

- The released implementation called the recorded Herdr binary with literal `agent prompt <pane> <labelled-text>`
  arguments. That submits terminal input. Herdr 0.9.1's protocol-22 API schema offers `agent.prompt` and
  `pane.send_input`, neither with an empty-draft precondition. Agent status describes activity, not unsent input.
  A read followed by a send would still race typing. `pane.input.set` selects mouse input behavior, not draft text.
- Herdr also offers `notification.show` (`herdr notification show <title> --body <text>`). This is a visual toast,
  separate from input, but it does not feed the launching agent's inbox. It is not treated as message delivery.
- Claude Code 2.1.307 and 2.1.309's installed runtimes implements a local native inbox. It publishes
  `CLAUDE_CODE_MESSAGING_SOCKET`, `CLAUDE_CODE_MESSAGING_TOKEN` and session identity into its shell environment;
  session registry records also contain `messagingSocketPath` and `sessionId`. The runtime's own diagnostic shows
  newline-delimited auth and user frames. Its handler enqueues user content with peer origin, `isMeta: true`,
  `skipSlashCommands: true`, separately from the terminal editor. The optional `session_id` binds a frame to that
  conversation. There is no acknowledgement on this stream: successful submission does not prove acceptance or reading.
  These findings came from the installed runtime and CLI schema/help, without sending to a real session.

Chosen path: capture the Claude socket, child auth token and session id at launch in the existing private
`launcher.json`. Use auth then a labelled user frame, bound to the captured session when known; use the existing
10-second delivery bound. Preserve the fixed `⟦peer · …⟧` origin/reply header and body normalization. A launcher that is itself a peer job instead records its existing mailbox (job id and directory),
and the question is delivered through the same locked inbox as `tell`, retaining the fixed label.
This supports nested jobs of every target type without new mailbox machinery. A Codex launch
marker or any peer job excludes inherited Claude socket addresses. Ambiguous launchers
therefore queue instead of targeting an ancestor. Do not discover recipients by matching worktree names or probing
live sessions. Older Herdr-only launcher files decode with ignored legacy fields and queue without sending input.

When there is no native inbox/token, the platform lacks Unix sockets, or transport fails, retain the message as
`queued` in the existing outbox journal. The command exits successfully with an explicit queued receipt and the
`--result <job>` conversation recovery path. There is no automatic retry: an interrupted/failed write might have
reached the inbox, so review before repeating a question. The job continues independent work and carries unresolved
questions in its handback. Malformed/unreadable mailbox or channel files remain explicit infrastructure errors.
`sent` means submitted to the native inbox, with the lack of acknowledgement stated in both receipt and journal.
The existing lock reserves attempt slots; no second mailbox/history read is added after delivery.

This trades immediate delivery for preserved drafts on unsupported/ambiguous launchers, including current Codex,
Gemini, Pi and Herdr-only standalone launchers. Parent peer jobs use their existing inbox. A safe immediate channel for those launchers would require a separate harness
integration and is deferred. No input injection override is offered: queueing is the authorized degraded path.
Nothing is automatically replayed when an old job is resumed; its existing launcher record governs that job.

Adds: a private `peer_job` launcher mailbox reference, three private launcher fields (`claude_socket`, `claude_token`, `claude_session`), the outbox status `queued`,
and a plain-text submission/queue receipt for `say`. Retires: capture/use of terminal addresses, production invocation of Herdr, the Windows-only
refusal, and tests that require terminal input injection. No new dependency, command, daemon or durable queue file.
Verification uses owned fake Unix sockets and the existing offline CLI/target fixtures; no real session receives text.

## Other sweep items

- Launch and single-job status JSON gain top-level `job_id`; `metadata.job_id` remains canonical and is mirrored at
  the JSON formatting boundary, including older persisted results. Multi-job status already has top-level `job_id`.
  The skill now distinguishes top-level `job_id` from `metadata.cwd`.
- `--wait` in the fetched v0.12.3 source already polls without printing running status. Its only result write is
  after a terminal outcome. The reported two-document output could not be reproduced from this tree; no root cause
  is inferred from the three real job ids. Preserve strict single-document regression coverage and state the
  contract explicitly; do not add a second output format or speculative polling changes.

## Verification and handoff

Measured result: 22 unique top-level Go tests passed, 48 including named subtests, with no failures or skips.
The final messaging run passed 15 top-level tests (37 including subtests). The JSON regression run passed two;
the earlier focused run supplies the remaining unchanged JSON formatting, terminal wait and message-normalization checks.
Logs and a command/result summary are in `/storage/scratch/peeragent/sweep-say-and-output/verification.log`,
with the tested source commit recorded there at handoff. The logs contain only offline fixtures.

Commands:

```sh
go test ./internal/app ./internal/result ./cmd/peeragent -run 'Test(Say|ConcurrentSay|CommandsSnapshotEachMailboxOnce|WaitReturnsTerminalAsyncResult|WaitJSON|LaunchAndStatusJSON|FormatJSON|AbsentLauncher|HeaderSanitizes|MessageSanitizer|TellInbox|MessagingCapacityRetry)' -count=1 -json
go test ./internal/app ./cmd/peeragent -run 'Test(Say|ConcurrentSay|CommandsSnapshotEachMailboxOnce|AbsentLauncher|MessagingCapacityRetry|ConcurrentTell|JobPreambleEnvironment|RetryStartupFailure|CapacityRepliesReach|CarriedReplyIsNot|ResumeAcknowledges|CancelledTurnLeaves|ResumeDirectoryAlias)' -count=1 -json
go test ./cmd/peeragent -run 'Test(WaitJSON|LaunchAndStatusJSON)' -count=1 -json
scripts/build.sh
GOOS=windows GOARCH=amd64 go build -o dist/peeragent-windows.exe ./cmd/peeragent
claude plugin validate --strict plugin
cmp skills/peer/SKILL.md plugin/skills/peer/SKILL.md
git diff --check
```

The exact plugin metadata, skill metadata constraints and documentation-example sections from `scripts/validate.sh`
also passed, extracted without its full-suite or release-artifact steps. Plugin validation passed with its existing
missing-icon warning. Linux execution and Windows compilation were checked; Windows/macOS runtime behavior and
real-session delivery were not exercised. No timing or Steam Deck run was made.

Tests cover captured socket/auth/session frames and label normalization; inherited-address rejection; legacy,
missing and unavailable channels; queue visibility and failed-attempt bounds; concurrent sends; nested launcher
mailboxes (running, finished and resume-only); both job-id fields; and a single terminal JSON document with unread
notices kept on stderr, including an older persisted result without a top-level id. Existing inbox/resume tests
also passed. No redundant mailbox scan or new mailbox owner was added.

Review, landing, full release validation, version bump, packaged binary refresh and release remain with godot-2d.
No host deploy is needed beyond updating the installed peeragent release. Older pane-only jobs queue with the new
binary; start new jobs to capture native launcher channels. Immediate delivery for unsupported standalone harnesses
and the unreproduced original extra-document report remain explicit limits, rather than speculative fixes.
