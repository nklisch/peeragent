---
id: peer-say-delivery-refusal-reason
kind: story
tags: [infra]
parent: null
depends_on: []
release_binding: null
created: 2026-10-24
updated: 2026-10-24
---

# Explain why an outbound peer question could not be delivered

## Observed behavior

In an active peer-launched Codex implementation job on Linux, installed
peeragent 0.12.2 accepted `peeragent inbox` reads. One
`peeragent say <question text redacted>` attempt exited 1 and printed:

```text
say: Herdr refused or could not deliver the message; use your handback instead
```

The question was below the documented 2,000-character limit. This was one
attempt, with no retry burst. The parent question remained undelivered. The
fallback was to retain the unresolved question and its evidence for the final
handback, without another terminal, desktop, or keyboard communication path.

## Expected evidence and limits

The supported question channel should deliver the question or distinguish a
refused destination, unavailable destination, and transport failure with a
concrete reason. The observed message combines them, so the caller cannot tell
which condition applies. The failed delivery is confirmed; no root cause or
peeragent defect has been established. It may be an intentional destination
restriction or an availability problem.

Reproduction shape: launch an ordinary asynchronous job with messaging enabled;
from that running job, read its inbox and send one bounded question with
`peeragent say`. No prompt text, parent task details, credentials, or clipboard
contents are retained in this report. No implementation change is authorized
by this observation.
