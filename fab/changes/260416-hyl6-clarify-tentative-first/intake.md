# Intake: Clarify Tentative First

**Change**: 260416-hyl6-clarify-tentative-first
**Created**: 2026-04-16
**Status**: Draft

## Origin

> Fab clarify should focus on tentative questions first. Right now it always starts from confident assumptions via bulk confirm. Reorder so tentative assumptions are addressed before confident ones.

One-shot description. The user identifies that the current `fab-clarify` suggest mode flow prioritizes confident assumptions (via bulk confirm at Step 1.5) before addressing tentative assumptions (surfaced at Step 2 via taxonomy scan and `<!-- assumed: -->` markers). The desired behavior is to reverse this order so tentative — the higher-risk, less-certain assumptions — are resolved first.

## Why

Tentative assumptions represent decisions with lower confidence than Confident ones — they are "reasonable guesses with multiple valid options." Addressing them first gives the user more agency over the riskiest decisions before being asked to rubber-stamp the safer ones. The current order (bulk confirm Confident first, then taxonomy scan for Tentative) front-loads the easy confirmations and delays the harder, more impactful questions.

If left as-is, users spend their initial attention on confirming assumptions that are already fairly safe, and may reach the tentative questions with less focus or hit the 5-question cap before getting to them.

## What Changes

### Reorder Suggest Mode Steps in `fab-clarify`

The current flow is:

1. Step 1: Read Target Artifact
2. Step 1.5: Bulk Confirm (Confident Assumptions)
3. Step 2: Taxonomy Scan (catches Tentative via `<!-- assumed: -->` markers)
4. Steps 3–8: Questions, updates, audit trail, summary

The new flow should be:

1. Step 1: Read Target Artifact
2. Step 1.5: Tentative Assumption Resolution (from taxonomy scan's `<!-- assumed: -->` markers)
3. Step 2: Bulk Confirm (Confident Assumptions)
4. Steps 3–8: Remaining taxonomy questions, updates, audit trail, summary

Specifically:

- The taxonomy scan (current Step 2) should run **before** bulk confirm, and tentative `<!-- assumed: -->` markers should be presented to the user first
- Bulk confirm (current Step 1.5) should move to **after** tentative resolution
- The taxonomy scan note ("If Step 1.5 triggered, this scan runs on the already-updated artifact") reverses: bulk confirm now runs on the artifact already updated by tentative resolution

### Update `_preamble.md` Bulk Confirm Reference

The `_preamble.md` Bulk Confirm section states: "This flow runs as Step 1.5 in Suggest Mode, before the standard taxonomy scan (Step 2)." This needs to reflect the new ordering where bulk confirm runs after tentative resolution.

### Update Memory File

`docs/memory/fab-workflow/clarify.md` documents the current ordering in the "Bulk Confirm" subsection: "suggest mode SHALL offer a bulk confirm flow before the taxonomy scan (Step 1.5)." This needs to reflect the new order.

## Affected Memory

- `fab-workflow/clarify`: (modify) Reorder suggest mode flow — tentative resolution before bulk confirm

## Impact

- `src/kit/skills/fab-clarify.md` — primary change: reorder Steps 1.5 and 2
- `src/kit/skills/_preamble.md` — update Bulk Confirm description to reflect new ordering
- `docs/memory/fab-workflow/clarify.md` — update to match new behavior
- `docs/specs/skills.md` — may need update if it documents the step ordering

## Open Questions

- None — the change is well-scoped: reorder two existing steps in the suggest mode flow.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The taxonomy scan and tentative resolution happen as a single step before bulk confirm | Directly stated in the user's request — "focus on tentative questions first" | S:95 R:85 A:90 D:90 |
| 2 | Certain | Step renumbering: tentative resolution becomes Step 1.5, bulk confirm becomes Step 2 | Logical consequence of the reorder — steps need consistent numbering | S:90 R:90 A:95 D:95 |
| 3 | Certain | The 5-question cap remains unchanged | User only asked to reorder, not to change question limits | S:85 R:90 A:95 D:95 |
| 4 | Certain | Auto mode is unaffected (already skips bulk confirm) | User's request is about suggest mode ordering only; auto mode has no bulk confirm | S:90 R:95 A:95 D:95 |
| 5 | Confident | Bulk confirm's note about scanning the "already-updated artifact" flips to describe bulk confirm running on the artifact updated by tentative resolution | Logical consequence of the reorder — the context note needs to reference the new preceding step | S:80 R:85 A:85 D:75 |
| 6 | Confident | The taxonomy scan runs in full before any questions are presented (scan first, then present tentative markers, then bulk confirm) | The current flow scans then asks; reordering preserves this scan-then-ask pattern | S:70 R:80 A:80 D:70 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
