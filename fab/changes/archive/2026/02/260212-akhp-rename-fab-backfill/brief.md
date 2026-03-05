# Brief: Rename /fab-backfill to /fab-hydrate-design

**Change**: 260212-akhp-rename-fab-backfill
**Created**: 2026-02-12
**Status**: Draft

## Origin

**Backlog**: [akhp]
User requested: `/fab-new akhp`

> Rename /fab-backfill to /fab-hydrate-design

## Why

The current name `/fab-backfill` doesn't clearly communicate what the command does. The command identifies structural gaps between centralized docs and specs, then proposes additions back to the design layer (specs). The new name `/fab-hydrate-design` makes this purpose explicit — it hydrates the design layer from the documentation layer, similar to how `/fab-hydrate` ingests external sources into docs.

This rename improves semantic consistency across the fab workflow: `/fab-hydrate` brings content in, `/fab-hydrate-design` moves insights back to design specs.

## What Changes

- Rename skill file: `.claude/skills/fab-backfill/fab-backfill.md` → `.claude/skills/fab-hydrate-design/fab-hydrate-design.md`
- Rename skill directory: `.claude/skills/fab-backfill/` → `.claude/skills/fab-hydrate-design/`
- Update all references to `/fab-backfill` across documentation:
  - README.md
  - SKILLS.md
  - fab/docs/ documentation files
  - Any archived change references
- Update skill metadata (command name, description headers)
- Functionality remains unchanged — pure rename

## Affected Docs

### Modified Docs
- `fab-workflow/backfill`: Update to reflect new command name `/fab-hydrate-design`
- Any docs referencing the command in workflows or examples

## Impact

**Files affected:**
- Skill implementation: `.claude/skills/fab-backfill/`
- Documentation: README.md, SKILLS.md, fab/docs/fab-workflow/
- Potentially: archived changes that mention the command

**User impact:**
- Users must learn new command name
- Existing documentation/tutorials referencing `/fab-backfill` will be outdated
- No functional changes — behavior is identical

**Related changes:**
- This aligns with the semantic clarity improvements across the fab workflow
- Part of ongoing efforts to make command names self-documenting

## Open Questions

<!-- No blocking questions — rename intent is clear from backlog and naming pattern -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Certain | Functionality unchanged | Backlog item specifies rename only |
| 2 | Confident | Rename clarifies semantic purpose | New name `/fab-hydrate-design` explicitly indicates hydration to design layer, matching pattern of `/fab-hydrate` for docs |
| 3 | Confident | Standard skill rename procedure | Move directory, update references, update internal metadata |
| 4 | Confident | All documentation references need updating | Ensures consistency across README, SKILLS.md, and centralized docs |

4 assumptions made (1 certain, 3 confident, 0 tentative).
