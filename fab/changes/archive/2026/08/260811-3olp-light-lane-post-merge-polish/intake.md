# Intake: Light Lane Post-Merge Polish

**Change**: 260811-3olp-light-lane-post-merge-polish
**Created**: 2026-08-12

## Origin

One-shot `/fab-new 3olp` from the backlog. Raw backlog entry:

> [3olp] 2026-08-11: Post-merge polish from 3ol6's passing review (3 should-fix + 2 nice-to-have, all one-liners): (1) _pipeline.md:157 Auto-Rework item 3 — half-line scoping fab-adopt's loop consumption to the FULL branch; (2) stage-models.md:694 — qualify "/fab-continue's ship/review-pr rows mirror fab-fff Steps 4–5" to the full-lane Steps 4–5; (3) user-flow.md:107 — mermaid rework edge label asserts sub-agent rework unconditionally, qualify for inline light-lane rework; (4) OBE — the original target SPEC-_pipeline.md was retired with the mirror tree (260811-rehi); its successor, the `_pipeline` skeleton in docs/specs/skills.md § Partial Flow Skeletons, already has consistent ├─/└─ glyphs in the Step 1 group — nothing left to fix; (5) glossary.md:141 — trim the "Light lane" entry to definition + pointer (owner-or-pointer). All in files already touched by 260811-3ol6; fix on main after merge or fold into the next lane-adjacent change.

These are the deferred findings from change 260811-3ol6 (Light Lane — PR #587, now merged) — its passing review surfaced them as should-fix/nice-to-have one-liners too small to block the merge. The backlog's precondition ("fix on main after merge") holds: #587's squash commit is in this branch's history. Gap analysis re-verified each item against the current tree: items 1, 2, 3, and 5 are still open; item 4 is OBE per the backlog entry itself and is excluded from scope.

## Why

1. **Pain point**: Three prose sites written before (or alongside) the light lane still describe rework and ship/review-pr dispatch as unconditional sub-agent behavior, which is now only true of the **full lane** — the light lane runs rework (and, for `/fab-fff`, ship/review-pr) inline in the orchestrator's context. A reader following `docs/specs/user-flow.md` or `docs/specs/stage-models.md` gets a claim the shipped behavior contradicts in the light lane. Separately, `_pipeline.md`'s Auto-Rework item 3 gained FULL/LIGHT branches without saying which branch `/fab-adopt` (a partial consumer of the loop, per `_preamble.md` § Worker Continuation) consumes; and the glossary's "Light lane" entry restates lane mechanics that `_pipeline.md` § Light Lane owns, violating the owner-or-pointer convention.
2. **Consequence if unfixed**: spec/skill prose drifts from shipped behavior — exactly the class of stale-claim debt the FKF present-truth passes exist to burn down; the glossary duplication invites divergence the next time lane mechanics change.
3. **Why this approach**: these were reviewer-confirmed one-liners from 3ol6's own review, deliberately deferred to post-merge. A single small docs change that lands all four is cheaper than folding them into an unrelated lane-adjacent change and keeps attribution clean.

## What Changes

Four prose-only edits, no behavior change. Backlog line numbers have drifted slightly; anchors below are by content (verified in the current tree).

### 1. `src/kit/skills/_pipeline.md` — Auto-Rework item 3: scope `/fab-adopt`'s loop consumption to the FULL branch

Item 3 of the per-cycle choreography ("Re-dispatch apply (resume-first)") now forks **FULL lane** / **LIGHT lane**. `_preamble.md` § Worker Continuation names `/fab-adopt` as "a partial consumer of that loop", but item 3 never says which branch adoption uses. Add a half-line to item 3 scoping `/fab-adopt`'s loop consumption to the **FULL** branch (adoption has no lane fork — it always reworks via the dispatched full-lane path). Edit the canonical source at `src/kit/skills/_pipeline.md` (never the deployed `.claude/skills/` copy).

### 2. `docs/specs/stage-models.md` — qualify the "mirrors fab-fff Steps 4–5" claim to the full lane

In § Skill wiring (currently ~line 691–695), the `/fab-continue` ship/review-pr bullet says the rows resolve roles "**mirroring `/fab-fff` Steps 4–5 exactly**". In the light lane, `/fab-fff` runs ship/review-pr inline with no `fab resolve-agent` — so the mirror claim is only true of the **full-lane** Steps 4–5. Qualify the phrase accordingly (e.g., "mirroring `/fab-fff`'s full-lane Steps 4–5 exactly").

### 3. `docs/specs/user-flow.md` — qualify the mermaid rework edge label

The state-diagram edge (currently line 107) reads:

```
review --> apply: auto-rework (sub-agent, fab-ff/fab-fff)
```

It asserts sub-agent rework unconditionally; light-lane rework is inline. Reword the label to be lane-accurate while staying terse enough for a mermaid edge — e.g. `auto-rework (fab-ff/fab-fff; sub-agent full lane, inline light lane)` or drop the parenthetical mechanism qualifier entirely. Exact label wording decided at apply.

### 4. `docs/specs/glossary.md` — trim the "Light lane" entry to definition + pointer

The "Light lane" glossary entry (currently ~line 141) is a full paragraph restating lane mechanics (task-count fork values, flag names, dispatch/resolve-agent behavior, rework budget, promotion valve, full-lane contrast). Per the owner-or-pointer convention, trim it to a one-to-two-sentence definition (the inline execution lane of `/fab-ff`/`/fab-fff`, chosen once at apply entry by plan task count) plus the pointer it already carries ("Lane mechanics are owned by `_pipeline.md` § Light Lane"). The **full lane** contrast mention stays as part of the definition.

### Excluded: backlog item 4 (OBE)

The original target `SPEC-_pipeline.md` was retired with the mirror tree (260811-rehi); its successor skeleton in `docs/specs/skills.md` § Partial Flow Skeletons already has consistent glyphs. Nothing to do.

## Affected Memory

None — all four edits are prose clarifications and a duplication trim; no spec-level behavior changes. `docs/memory/pipeline/execution-skills.md` already states the light-lane inline rework, the full-lane-only worker continuation, and `/fab-adopt`'s partial loop consumption correctly.

## Impact

- `src/kit/skills/_pipeline.md` — one half-line addition (kit skill source; redeploys via `fab sync`)
- `docs/specs/stage-models.md` — one phrase qualification
- `docs/specs/user-flow.md` — one mermaid edge label reword
- `docs/specs/glossary.md` — one entry trimmed (net deletion)
- No Go code, no tests, no templates, no memory files. No behavior change anywhere — reviewer scope is prose accuracy and the owner-or-pointer convention.

## Open Questions

None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Backlog item 4 (SPEC-_pipeline.md glyphs) is excluded — OBE | The backlog entry itself records the target was retired (260811-rehi) and the successor skeleton is already consistent; re-verified nothing to fix | S:95 R:95 A:100 D:95 |
| 2 | Certain | Anchor edits by content, not the backlog's line numbers | Lines drifted since 2026-08-11; all four target passages located and verified in the current tree during gap analysis | S:90 R:95 A:95 D:90 |
| 3 | Confident | Exact qualifier/label wording is crafted at apply (per-item intent is fixed by the backlog) | One-liner docs edits, trivially reversible; the backlog states each item's intent precisely but not final phrasing — mermaid label length needs judgment | S:75 R:90 A:80 D:70 |
| 4 | Confident | `/fab-adopt` consumes the FULL branch of item 3 | `_preamble.md` § Worker Continuation lists /fab-adopt as a partial consumer of the full-lane continuation apparatus; adoption has no lane fork | S:80 R:85 A:85 D:80 |
| 5 | Confident | No memory updates needed (Affected Memory: none) | Prose-only clarifications; pipeline/execution-skills.md already documents the corrected behavior — template rule: implementation-only changes don't need memory updates | S:70 R:85 A:85 D:75 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).
