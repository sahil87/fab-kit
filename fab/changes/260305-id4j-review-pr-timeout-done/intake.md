# Intake: Review-PR Timeout Treated as Done

**Change**: 260305-id4j-review-pr-timeout-done
**Created**: 2026-03-05
**Status**: Draft

## Origin

> "for the git-pr-review stage, if there's no human review, and no copilot review comes in time, the final state should be done, not failed"

One-shot directive. The user wants the `review-pr` stage to end gracefully when no review materializes, rather than marking the pipeline as failed.

## Why

Currently, `/git-pr-review` treats a Copilot timeout (Phase 3: 12 polls over 6 minutes with no response) as a failure, calling `statusman.sh fail <change> review-pr`. This marks the `review-pr` stage as `failed` in `.status.yaml`, which:

1. **Blocks archiving** — `/fab-archive` requires all stages to be resolved; a `failed` review-pr leaves the change in limbo.
2. **Misrepresents the outcome** — the absence of a review is not a defect or error. The code was already reviewed internally (the `review` stage passed). External PR review is best-effort.
3. **Requires manual intervention** — the user must re-run `/git-pr-review` or manually fix the status to proceed.

The "no reviews and Copilot unavailable" case (Phase 2 failure) already correctly calls `finish` (done). The Copilot timeout case should follow the same logic — it's a successful no-op, not a failure.

## What Changes

### `/git-pr-review` Skill (`fab/.kit/skills/git-pr-review.md`)

**Step 2, Phase 3 — Copilot timeout handling**:

Currently, polling uses max 12 attempts at 30s intervals (6 minutes total). Change to **16 attempts** (8 minutes total). The timeout message updates accordingly:

- Current: `Copilot review did not arrive within 6 minutes.`
- New: `Copilot review did not arrive within 8 minutes.`

**Step 6 — Stage update routing**:

Currently has three cases:
1. On success → `finish`
2. On failure (Copilot timeout, no PR found, processing error) → `fail`
3. On no reviews and Copilot unavailable → `finish`

Change to:
1. On success (comments processed and pushed, or no actionable comments) → `finish`
2. On failure (no PR found, processing error) → `fail`
3. On no reviews (Copilot unavailable OR Copilot timeout OR no reviews at all) → `finish`

The key change: **Copilot timeout moves from case 2 to case 3**. The rationale is that the absence of external review feedback is a graceful completion, not a pipeline failure.

### Memory file (`docs/memory/fab-workflow/execution-skills.md`)

Update the PR review handling section to reflect that Copilot timeout results in `finish` (done) rather than `fail` (failed).

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update PR review handling to reflect Copilot timeout as graceful done

## Impact

- **`fab/.kit/skills/git-pr-review.md`** — Step 6 routing logic, removing Copilot timeout from the failure case
- **`docs/memory/fab-workflow/execution-skills.md`** — PR review handling description
- **Pipeline behavior** — changes with no external review will complete cleanly through to archive-ready state
- No script changes needed — `statusman.sh finish` and `fail` are already correct; only the skill's routing decision changes

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only the skill markdown and memory file need changes, no script modifications | The statusman commands (`finish`, `fail`) are already correct — the change is purely about which one gets called when and the poll count is just a constant in the skill prose | S:90 R:95 A:95 D:95 |
| 2 | Certain | "No PR found" and "processing error" remain as failure cases | These represent actual errors (missing PR, API failures) — distinct from "no review arrived" | S:85 R:90 A:90 D:90 |
| 3 | Certain | Copilot timeout follows the same pattern as "Copilot unavailable" | Both represent "no external review feedback available" — same logical outcome | S:90 R:95 A:85 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
