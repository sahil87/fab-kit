# Proposal: Add --no-switch Flag to fab-new

**Change**: 260212-r7k3-add-no-switch-flag
**Created**: 2026-02-12
**Status**: Draft

## Why

`/fab-new` always activates the newly created change via `/fab-switch` (Step 8). There's no way to create a change for later without disrupting the current active context. This is friction when batching multiple change captures or when you want to stay focused on current work.

## What Changes

- Add an optional `--no-switch` flag to `/fab-new` that skips Step 8 (the internal `/fab-switch` invocation)
- When `--no-switch` is used: the change folder, `.status.yaml`, and `brief.md` are created as normal, but `fab/current` is NOT written and no branch is created/checked out
- Output changes to show a different `Next:` line when `--no-switch` is used, following the existing pattern from `/fab-discuss` (not activated): `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`

## Affected Docs

### New Docs

(none)

### Modified Docs
- `fab-workflow/planning-skills`: Update fab-new documentation to reflect the new `--no-switch` flag

### Removed Docs

(none)

## Impact

- **Skill file**: `fab/.kit/skills/fab-new.md` — add `--no-switch` to Arguments section, add conditional logic to Step 8, update Output section with no-switch example
- **Context file**: `fab/.kit/skills/_context.md` — Next Steps table already has a "not activated" pattern (from fab-discuss); no change needed there
- **No code changes** — this is a prompt-only change (Pure Prompt Play principle)

## Open Questions

(none — all decisions resolved from context)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Scope to fab-new only (not fab-discuss) | fab-discuss already has its own "not activated" flow; user specifically requested fab-new |

1 assumption made (1 confident, 0 tentative).
