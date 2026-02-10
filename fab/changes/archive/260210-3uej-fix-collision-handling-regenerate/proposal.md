# Proposal: Fix collision handling in fab-new to regenerate 4-char component

**Change**: 260210-3uej-fix-collision-handling-regenerate
**Created**: 2026-02-10
**Status**: Draft

## Why

`fab-new.md` Step 2 (Create Change Directory) says: "If a change folder with the same name already exists, append an additional random character to `{XXXX}` and retry." This produces a 5-character random component (`XXXXX`) which breaks the `{YYMMDD}-{XXXX}-{slug}` naming format defined in Step 1 and `fab/config.yaml`. The correct behavior is to regenerate the entire 4-character `{XXXX}` component (new random value, same length) rather than appending to it.

## What Changes

- Update the collision handling text in `fab-new.md` Step 2 from "append an additional random character" to "regenerate the 4-character random component (`{XXXX}`) and retry"
- This preserves the `{YYMMDD}-{XXXX}-{slug}` format invariant

## Affected Docs

### New Docs
- None

### Modified Docs
- `fab-workflow/change-lifecycle`: Update if it documents collision handling behavior

### Removed Docs
- None

## Impact

- `fab/.kit/skills/fab-new.md` — one sentence change in Step 2
- The corresponding SKILL.md loaded by the agent framework also needs the same fix (if it exists as a separate file)

## Open Questions

- None — this is a one-line wording fix with a clear correct answer.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Fix both `fab-new.md` and the SKILL.md template if both exist | The SKILL.md in `.agents/skills/fab-new/` is the agent-facing version; both must be consistent |

1 assumption made (1 confident, 0 tentative).
