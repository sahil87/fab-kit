# Proposal: Auto-activate after /fab-discuss when no current change

**Change**: 260210-zr1f-discuss-auto-activate-when-no-current
**Created**: 2026-02-10
**Status**: Draft

## Why

`/fab-discuss` creates a change folder but never writes to `fab/current`, by design — to avoid disrupting the user's active work. But when `fab/current` is empty (no active change), there is nothing to disrupt. The user must still manually `/fab-switch` before proceeding, adding friction with no safety benefit. This is the only entry path where a change exists but isn't immediately usable.

## What Changes

- After `/fab-discuss` (new change mode) generates the proposal and displays the summary, check whether `fab/current` exists and is non-empty
- If `fab/current` is empty or missing: prompt the user — "No active change — set {name} as active?"
- If the user accepts: call `/fab-switch` internally (same pattern as `/fab-new`) to write `fab/current` AND handle git branch integration
- If the user declines: leave `fab/current` untouched; show the usual "Next: /fab-switch {name}" guidance
- Refine mode is unaffected (it already has an active change)
- The "Next:" output adjusts based on whether activation happened — if activated, skip the `/fab-switch` suggestion

## Affected Docs

### New Docs

(none)

### Modified Docs

- `fab-workflow/change-lifecycle.md`: Update the `fab/current` lifecycle section — `/fab-discuss` conditionally writes via internal `/fab-switch` when no active change exists
- `fab-workflow/planning-skills.md`: Update `/fab-discuss` output section and behavioral description to reflect the conditional offer

### Removed Docs

(none)

## Impact

- **Skill file**: `.claude/skills/fab-discuss` — add conditional logic after Step 7 (summary display), before "Next:" output
- **Key Properties table**: "Switches active change?" changes from "No" to "Conditionally — offers when `fab/current` is empty"
- **Key Differences table**: Update the `/fab-discuss` row to reflect conditional activation
- **Centralized docs**: `change-lifecycle.md` and `planning-skills.md` need changelog entries

## Open Questions

(none — all resolved during discussion)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Prompt wording: "No active change — set {name} as active?" | Clear, concise, matches existing fab prompt style |
| 2 | Confident | Refine mode unaffected | Refine mode requires an active change by definition — the condition can never trigger |
| 3 | Confident | If `/fab-switch` fails (e.g. git error), report error and suggest manual `/fab-switch` | Proposal is already saved; activation failure is non-fatal |
