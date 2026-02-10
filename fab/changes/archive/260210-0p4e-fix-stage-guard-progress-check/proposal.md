# Proposal: Fix stage guard to check progress value instead of stage name

**Change**: 260210-0p4e-fix-stage-guard-progress-check
**Created**: 2026-02-10
**Status**: Draft

## Why

`fab-continue`'s stage guard checks the `stage` field name to determine whether to proceed, but does not account for the `progress` value of that stage. If a change has `stage: tasks` but `progress.tasks: active` (meaning task generation was interrupted mid-way), the guard sees "tasks" and blocks with "Planning is complete. Run /fab-apply", when it should allow resumption of the interrupted tasks generation. The guard should check the progress value (`done` vs `active`) to distinguish between "completed" and "interrupted" states.

## What Changes

- Update the stage guard logic in `fab-continue.md` to check `progress.{stage}` value, not just the `stage` name
- When `stage: tasks` and `progress.tasks: active`, the guard should allow regeneration/resumption instead of blocking
- Same fix applies to any other stage where the guard could incorrectly block on `active` state (e.g., `stage: specs` with `progress.specs: active`)

## Affected Docs

### New Docs
- None

### Modified Docs
- `fab-workflow/planning-skills`: Update to document the corrected guard behavior

### Removed Docs
- None

## Impact

- `fab/.kit/skills/fab-continue.md` — Step 1 (Determine Current Stage) guard logic updated
- The fix is contained to the guard conditions in the Normal Flow section

## Open Questions

- None — the bug is clearly described and the fix is straightforward (check progress value, not just stage name).

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Apply the fix only to `fab-continue.md` | `fab-ff.md` already uses `progress` map for resumability checks; the bug is specific to `fab-continue`'s guard logic |

1 assumption made (1 confident, 0 tentative).
