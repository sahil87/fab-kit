# Proposal: Fix created_by Format to Use GitHub ID

**Change**: 260212-k7p3-fix-created-by-github
**Created**: 2026-02-12
**Status**: Draft

## Origin

> User requested: `/fab-new 0r8e`
>
> **Backlog**: [0r8e]
> **Linear**: DEV-1012
> **Milestone**: M5: Trial Fixes — Correctness & Ergonomics

## Why

The current `.status.yaml` format captures `created_by` using `git config user.name`, which returns email addresses in some configurations (e.g., "user@example.com") and human names in others (e.g., "Sahil Ahuja"). This inconsistency makes it difficult to reliably attribute changes to specific developers, especially in team environments where multiple people may share similar names or use different git configurations.

Using GitHub IDs via `gh api user --jq .login` provides a stable, globally unique identifier that's consistent across all git configurations and directly links to the developer's GitHub profile.

## What Changes

- Update `/fab-new` skill to use `gh api user --jq .login` for `created_by` field instead of `git config user.name`
- Add graceful fallback: if `gh` command fails (not installed, not authenticated, or API error), fall back to `git config user.name` as before
- Only affects new changes created after this fix — existing changes remain unchanged
<!-- assumed: Only future changes affected — updating historical .status.yaml files could be controversial and requires user buy-in -->

## Affected Docs

### New Docs
*None*

### Modified Docs
- `fab-workflow/planning-skills`: Update `/fab-new` implementation details for `created_by` field behavior

### Removed Docs
*None*

## Impact

- **Skills affected**: `/fab-new` (Step 4: Initialize `.status.yaml`)
- **Files modified**: `fab/.kit/skills/fab-new.md` (instruction update for `created_by` field)
- **Backward compatibility**: Full — existing `.status.yaml` files retain their current `created_by` values
- **Dependencies**: Assumes `gh` CLI is available (graceful fallback to git if not)

## Open Questions

*No unresolved blocking questions*

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Use `git config user.name` as fallback field | Follows existing pattern in current implementation |
| 2 | Tentative | Only affect future changes, not historical | Updating historical data requires user buy-in; safer to leave existing changes unchanged |

2 assumptions made (1 confident, 1 tentative). Run /fab-clarify to review.
