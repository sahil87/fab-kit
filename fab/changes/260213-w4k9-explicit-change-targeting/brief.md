# Brief: Explicit Change Targeting for Workflow Commands

**Change**: 260213-w4k9-explicit-change-targeting
**Created**: 2026-02-13
**Status**: Draft

## Origin

> the workflow based fab commends (fab-archive, fab-continue etc) should be able to work on changes other the current active change also. For example, I should be able to say "/fab-continue 260213-r3m7-add-conventions-section" to continue another change instead of the current one.

## Why

Currently, all workflow skills (`/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-clarify`, `/fab-status`) resolve the active change exclusively from `fab/current` via the preflight script. To operate on a different change, you must first run `/fab-switch {name}` and then the desired command — a two-step process that adds friction when juggling multiple in-flight changes.

This change enables a one-step workflow: `/fab-continue 260213-r3m7-add-conventions-section` targets the named change directly, without modifying `fab/current`. This is critical for parallel workflows — multiple Claude Code tabs can operate on different changes simultaneously without racing on the pointer file.

## What Changes

- **`fab/.kit/scripts/fab-preflight.sh`**: Accept an optional positional argument `$1` as a change name override. When provided, resolve and validate against that name instead of reading `fab/current`. **Do not modify `fab/current`** — the override is transient for this invocation only, enabling safe parallel operation across multiple tabs.
- **`/fab-continue` skill**: Accept an optional `[change-name]` argument. If provided, pass it to the preflight script. Document the argument in the skill's Arguments section.
- **`/fab-ff` skill**: Same — accept optional `[change-name]` argument, pass to preflight.
- **`/fab-fff` skill**: Same — accept optional `[change-name]` argument, pass to preflight.
- **`/fab-clarify` skill**: Same — accept optional `[change-name]` argument, pass to preflight.
- **`/fab-status` skill**: Accept optional `[change-name]` argument, pass to its status script for displaying a non-active change's status.
- **Partial/substring matching**: Reuse the same flexible matching logic from `/fab-switch` (exact match, partial substring, case-insensitive) so the argument behavior is consistent across all commands.
<!-- assumed: reuse fab-switch matching — consistent UX across all skills that accept change names -->
- **`_context.md`**: Update the "Change Context" section to document the optional override argument in the preflight invocation pattern.

## Affected Docs

### New Docs
- None

### Modified Docs
- `fab-workflow/context-loading`: Preflight now accepts optional change name override argument
- `fab-workflow/planning-skills`: Skills now accept optional change name argument
- `fab-workflow/execution-skills`: Apply/review skills inherit the override via preflight

### Removed Docs
- None

## Impact

- **Preflight script** (`fab/.kit/scripts/fab-preflight.sh`): Core change — add argument parsing and matching logic
- **6 skill files**: Each gains an optional `[change-name]` argument in its SKILL.md
- **Shared context** (`_context.md`): Preflight invocation pattern updated
- **User workflow**: No breaking changes — existing commands with no argument continue to work identically via `fab/current`
- **Parallel safety**: Transient targeting means `fab/current` is never modified by the override — multiple tabs can safely target different changes concurrently

## Open Questions

- None — all decision points resolved via SRAD analysis (see Assumptions below)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Skills affected: fab-continue, fab-ff, fab-fff, fab-clarify, fab-status | These are the workflow skills that use preflight to resolve the active change; deterministic from codebase analysis |
| 2 | Confident | Transient targeting (do NOT update `fab/current`) | Parallel tabs would race on the pointer file; transient resolution keeps each invocation independent and safe |
| 3 | Confident | Reuse fab-switch's partial/substring matching | Consistent UX across all skills that accept change names; the matching logic already exists |
| 4 | Confident | Implement in preflight script (centralized) | All skills go through preflight — adding the override there avoids duplicating matching logic in every skill |

4 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.
