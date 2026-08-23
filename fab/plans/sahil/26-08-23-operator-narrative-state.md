# Operator narrative state — no owned surface for cross-cutting coordination notes

> Backlog detail doc — written 2026-08-23 by the run-kit operator itself, surfacing a
> real friction it hit repeatedly this session. Not a decided plan — flagged for
> brainstorming. Verify file/line anchors before implementing.

## Problem

The operator's state file (`fab operator state`, server-keyed, `_cli-fab.md` § fab
operator) has an official, narrow schema: `monitored` (per-pane pipeline tracking),
`branch_map` (change → {branch, repo}), `autopilot`, `watches`, `tick_count`. Every
field in that schema is written through a dedicated `fab operator` subcommand
(`enroll`/`update`/`remove`, `autopilot start/advance/...`, `watch add/...`) — the
skill is explicit that **the operator never hand-writes this file**.

That schema has no home for the kind of state the operator accumulates constantly in
real multi-hour, multi-repo, multi-agent sessions:

- **Cross-change dependency waits** that outlive a single tick or a single `/clear`:
  "spawn X only once Y merges to main," "Z is stacked on W, don't ship independently."
  `depends_on` on a monitored entry covers same-repo cherry-pick/ordering *while the
  change is still monitored* — but a dependency that spans a `/clear`, a removal from
  `monitored` (terminal state reached), and a *later* spawn has nowhere to live.
- **Multi-phase plan progress** (e.g. a 5-phase plan where each phase is a separate
  fab change): which phases are done, which are merged vs. just shipped-as-PR, which
  are intentionally held back and why, what the next phase's spawn should say.
- **Cross-session coordination** with peer operator/agent sessions working the same
  repo concurrently: scoping agreements ("you take shortcuts-panel, I stay out of
  it"), promises to report back, discovered file-level overlaps.
- **Corrections to earlier operator conclusions** — e.g. discovering that a
  Copilot-review "silent drop" the operator reported to a peer was actually just
  *delayed*, which should inform how the operator (and its peers) treat the *next*
  apparent drop, potentially hours or a session-restart later.
- **Process lessons the operator draws about its own behavior mid-session** — e.g.
  "when holding on a PR-merge gate, actively re-check every tick" — that should
  survive a restart, not just live in the current conversation's context.

None of this fits `monitored` (pane-scoped, cleared on terminal state) or
`branch_map` (a flat id→branch/repo map with no room for prose). Lacking an owned
surface, the operator this session began hand-writing a `plan_queue:` top-level key
directly into the state file via `yq` — free-form `note:` strings keyed by an
ad-hoc slug. It works (the file persists, a restarted operator would read it back),
but it is a doctrine violation: no schema, no validation, no pruning, no distinction
between "this is done, keep for history" and "this is live, check it," and it grew
to ~10 entries of unstructured prose over one session with no way to tell which are
stale.

## What "done" would look like

A restarted operator (after `/clear`, a crash, or a fresh `fab operator` launch on
the same server) should be able to answer, from the state file alone and without
re-deriving from conversation history:

1. What multi-phase plans are in flight, which phases are done/merged/held, and why.
2. What cross-repo or cross-session coordination commitments are outstanding (a
   promise to report back, a scoping agreement with a peer).
3. What corrections/lessons apply to *how* it should behave going forward (e.g. "this
   class of Copilot-drop needs a re-check before being treated as final").

## Open questions for brainstorming

- **Shape**: a flat `notes: [{id, created_at, kind, text, resolved: bool}]` list?
  Structured by kind (`dependency_wait`, `phase_plan`, `coordination`, `correction`,
  `lesson`) with kind-specific fields? Or deliberately unstructured prose (current
  behavior) but through an owned verb (`fab operator note add/resolve/list`) instead
  of raw `yq`?
- **Lifecycle**: does a note ever auto-expire, or is pruning always explicit
  (`fab operator note resolve <id>`)? The `watches.known` precedent caps at 200
  entries, oldest-pruned — is an unbounded prose log the wrong shape entirely, or is
  bounded-with-explicit-resolve better for this use case (notes are decisions, not a
  dedupe cache)?
- **Granularity vs. `monitored`**: should `depends_on` on `monitored` entries be
  taught to survive removal (i.e., become a `branch_map`-like persistent record
  instead of vanishing when the entry is removed at terminal state), rather than
  inventing a wholly separate notes surface? That would fix the "dependency wait
  outlives monitoring" case specifically without solving the plan-progress or
  coordination cases.
- **Who queries it**: is this purely for the operator's own restart-recovery, or
  should `fab operator state` output surface it prominently (a `notes:` block
  printed at the top), so a human skimming the state file sees it without knowing to
  look?
- **Overlap with memory**: some of what ended up here this session (the Copilot-drop
  correction, the "re-check merge gates every tick" lesson) reads more like a
  `docs/memory` entry than operator-session state — is there a cleaner line between
  "state a live operator needs" and "a lesson that belongs in memory for future
  sessions regardless of operator restart"?

## Non-goals (tentative — for discussion, not settled)

- Not a replacement for `monitored`/`branch_map` — this is explicitly for state that
  doesn't fit the pane-scoped or flat-map shapes those already serve well.
- Not intended to become a general-purpose scratchpad — if every operator session
  reflexively dumps prose here, it degrades to exactly the unstructured mess it's
  meant to fix. Whatever shape comes out of this brainstorm should make "do I even
  need to write a note here" a real question, not a reflex.
