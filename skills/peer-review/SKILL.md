---
name: peer-review
description: >
  Run an honest, iterative cross-model peer review on recent work — designs,
  code, plans, refactors, architecture, docs, or any artifact in the repo — by
  delegating review to the OTHER local coding agent through peeragent (Claude
  asks Codex, Codex asks Claude). Then loop: weigh the feedback honestly,
  accept what's valid, push back on what isn't, refine the work, and ask
  again. Run at least 3 substantive passes; continue until only nits remain.
  Use this proactively after any nontrivial design or implementation pass,
  before declaring work done, when you suspect a blind spot, or when the user
  asks for "peer review", "second opinion", "cross-review", "have the other
  agent look at this", "what does Codex/Claude think", "/peer-review", or any
  phrasing that asks for feedback from the alternate agent. Free-form scope —
  invoke with a description of what to review, or with no args to review the
  most recent work in the conversation. Safe to invoke without explicit user
  request when shipping unreviewed feels risky.
allowed-tools: Bash
metadata:
  short-description: Iterative cross-model peer review
---

# Peer Review

A second pair of eyes from a different model, run as a loop. The other agent
reviews; you weigh the feedback honestly; you refine the work; you ask again.
Convergence is the goal — not consensus, not politeness, not a clean signoff
no one earned.

## When To Use

- You just finished a design, refactor, implementation, plan, or doc and
  want a real sanity check before declaring done.
- You suspect a blind spot — the kind of mistake only someone who didn't
  write it would catch.
- The user asked for peer review, a second opinion, cross-review, or
  feedback from "the other agent" in any phrasing.
- Proactively, after substantive work, when shipping unreviewed feels
  risky. You do not need explicit user permission to invoke this skill —
  proposing peer review is part of doing the work well.

Do not use this for trivial changes (a typo fix, a one-line rename). The
loop is overhead; reserve it for work where the loop pays for itself.

## The Other Agent

The peer reviewer is the agent you are not.

- If you are Claude Code, delegate review to Codex with
  `peeragent --agent codex ...`.
- If you are Codex, delegate review to Claude with
  `peeragent --agent claude ...`.
- Gemini is also available (`--agent gemini`) when the user asks for it or
  when the natural pair is unavailable.

The point of the alternate agent is **different blind spots**, not a more
authoritative answer. Their misses are your catches and vice versa.

## The Loop

Run at least 3 full review→refine passes. Continue past 3 only while
substantive issues keep surfacing. Stop when a pass returns mostly nits
(taste, minor wording, micro-style). Cap at 5 — if you are still in
substantive disagreement after 5, the work needs a human, not more loops.

Each pass has five steps. Do all five every time.

### 1. Frame the review request

Tailor the framing to the type of work. Keep it open. Ask for "the good,
the bad, and the ugly" — explicit permission for honest negatives. Do not
tell the reviewer your conclusions; let them form their own.

| Type of work | What to ask the reviewer for |
| --- | --- |
| Code / implementation | Correctness bugs, edge cases, security, reuse opportunities, simplifications. |
| Design / architecture | Coherence, missing concerns, scope errors, alternatives not considered, integration risks. |
| Plan / task breakdown | Completeness, sequencing, risk, hidden dependencies, unrealistic estimates. |
| Refactor | Whether the abstraction earns its weight, dead code, unintended behavior change. |
| Docs / writeup | Accuracy against the code, gaps, audience fit, lying-by-omission. |

The prompt should orient on **what kind of feedback you want** and **what
the artifact is** — not on the conclusions you want validated. Avoid
leading questions ("don't you think X is good?"). The reviewer is in the
same repo, so file paths work directly.

### 2. Delegate

A blocking call. Default model/effort is fine for most passes; bump effort
only when the work is dense or the stakes are high.

```bash
peeragent --agent <other-agent> "Review the following <work type> and give
me the good, the bad, and the ugly. Focus on <task-specific concerns>.
Artifact: <paths or inline description>. Context: <what changed, what
success looks like>."
```

For a deeper review pass:

```bash
peeragent --agent codex --effort xhigh "..."
peeragent --agent claude --model opus --effort xhigh "..."
```

For very large prompts, write to a file and use `--prompt-file <path>`.
Never request `--full-access` for review — review shouldn't need it.

### 3. Read the response honestly

For each point the reviewer raises, ask:

- **Is it correct?** Does the criticism hold up against the actual
  artifact, or did the reviewer misread? Verify before agreeing.
- **Does it improve the work?** Some valid observations don't justify a
  change — alternate-but-equal designs, taste preferences.
- **What weight does it carry?** Real bug > design risk > clarity issue >
  taste nit. Triage on weight, not volume.
- **Does the reviewer have a blind spot here?** Sometimes the reviewer
  misses context you have. Trust feedback that names a concrete problem;
  weigh skeptically feedback that's just "I'd do it differently".

You are not obligated to accept the reviewer's framing. You are obligated
to consider it.

### 4. Apply the accepted changes

Make the real edits in the working tree. The work is what changes; the
review is the signal. Nodding at feedback without applying it is the most
common failure mode of this loop — guard against it.

If you reject a point, note why — to yourself, in a running summary you'll
share with the user at the end.

### 5. Decide on the next pass

Look at what's left:

- Substantive issues outstanding → loop again.
- Reviewer caught new high-level issues this pass → keep looping; you
  haven't reached a stable design yet.
- Mostly nits and taste → after at least 3 passes, you're done.
- Reviewer is repeating themselves or contradicting their last pass →
  stop; that's noise.

Three is the minimum because pass 1 catches the obvious, pass 2 catches
what you fixed badly, pass 3 confirms convergence.

## How To Weigh Feedback

A healthy loop rejects roughly as many points as it accepts. Accepting
everything is sycophancy; rejecting everything is stubbornness. Both fail.

**Accept readily:**

- Concrete bugs the reviewer can point at.
- Edge cases you genuinely missed.
- Naming, structure, or factoring issues you can confirm by re-reading.
- Reuse opportunities — duplicate logic that should be shared.
- Contracts or constraints the work violates.

**Push back when:**

- The reviewer asserts a "better" pattern without showing why the current
  one is worse.
- The suggestion would expand scope past what was asked.
- The suggestion contradicts an explicit user constraint or earlier
  decision.
- It's a taste preference dressed as a correctness claim.
- The reviewer is hallucinating about code or APIs that don't exist that
  way — verify before applying.

When you push back, do it in your own notes. Do **not** argue with the
reviewer through another delegation — that wastes budget and doesn't
change their position. Just don't apply the change.

## Focus On The High-Level

The loop is for substantive concerns. Don't burn passes on nits.

- High-level (loop on these): correctness, design coherence, missing
  cases, integration risk, security, reuse, scope, contract violations.
- Nits (acknowledge and move on): variable naming preferences,
  whitespace, minor wording, single-line micro-optimizations, ordering of
  unrelated sections.

If a pass surfaces only nits, that's the signal you've converged. Apply
the worthwhile ones, drop the rest, stop the loop.

## Reporting Back To The User

After the loop ends, summarize:

- How many passes ran and why you stopped (converged on nits / hit cap /
  ran into noise).
- Substantive issues caught and fixed — the value the loop produced.
- Points raised and rejected, one line of reasoning each, so the user can
  override your judgment if they disagree.
- Final state of the work.

Keep it tight. The user doesn't need the full transcript of every pass —
they need to know what got better, what you decided against, and where
things landed.

## Result Handling

The wrapper returns JSON by default. Per delegation:

- `status: success` — read the reviewer's response, run it through the
  honest-evaluation step, apply what you accept.
- `status: blocked` — note the blocker, evaluate the work yourself for
  that pass, or surface to the user if it's structural.
- `status: failed` — surface the failure; one retry is fine; don't loop
  on a broken wrapper.
- `status: running` (if `--async` was used) — record the job id and check
  back; usually peer review wants the blocking default.

Do not claim the reviewer signed off unless a pass actually returned
without substantive findings.

## Guardrails

- **Apply what you accept.** A loop that produces accepted feedback but
  no edits is a wasted loop.
- **Don't loop on nothing.** Pass returns only nits and you've done at
  least 3 — stop. Don't pad the count.
- **Don't argue with the reviewer.** Take the input, judge it, move on.
  The reviewer is a worker, not a debate partner.
- **You own the judgment.** Don't ask the reviewer "is your last feedback
  right?" — they'll either confirm everything or contradict everything.
  The host weighs; the reviewer surfaces.
- **Be honest in the summary.** If you rejected most of the feedback,
  say so. Don't pretend the reviewer agreed with you.
- **Trivial work doesn't need this skill.** Use judgment — a one-line fix
  doesn't earn three peer-review passes.
- **No `--full-access` for review.** Reviews shouldn't need it.
