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
