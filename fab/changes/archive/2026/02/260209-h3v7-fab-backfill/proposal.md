# Proposal: Add `/fab-backfill` Command

**Change**: 260209-h3v7-fab-backfill
**Created**: 2026-02-09
**Status**: Draft

## Why

The data flow in fab is one-directional: specs inform changes, changes hydrate into docs. As docs evolve through successive `/fab-archive` runs, they accumulate coverage that specs never had — new skills, new concepts, new behavioral rules. There's no mechanism to detect these structural gaps and suggest concise additions back to specs. Specs risk becoming stale outlines while docs silently outgrow them.

## What Changes

- **New skill: `/fab-backfill`** — reads all doc domains and all spec files, cross-references at the section level to find structural gaps (topics docs cover that specs don't mention at all), and presents the top 3 gaps ranked by impact
- **Interactive propose-then-apply flow** — for each gap, shows the exact markdown snippet that would be inserted into the target spec file. User confirms, rejects, or skips each one individually. Only confirmed additions are written.
- **New skill file**: `fab/.kit/skills/fab-backfill.md` — skill definition
- **New Claude Code skill symlink**: `.claude/skills/fab-backfill.md` → `fab/.kit/skills/fab-backfill.md`

## Affected Docs

### New Docs
- `fab-workflow/backfill`: Documentation for the `/fab-backfill` skill — gap detection logic, confirmation flow, spec size guardrails

### Modified Docs
- `fab-workflow/specs-index`: Update "Human-Curated Ownership" section to reference `/fab-backfill` as the assisted (not automated) reverse-hydration mechanism, replacing the "future consideration" note

### Removed Docs
(none)

## Impact

- **`fab/.kit/skills/`** — new skill file
- **`.claude/skills/`** — new symlink
- **`fab/specs/`** — target of proposed insertions (only modified with user confirmation)
- **`fab/docs/fab-workflow/specs-index.md`** — minor update to reflect that reverse-hydration now exists
- **No changes to existing skills** — `/fab-backfill` is additive, doesn't modify the behavior of any existing command

## Open Questions

(none — all key decisions resolved during discussion)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Utility command pattern (no `fab/current`, no git branch) | Follows `/fab-status` precedent — read-only analysis + optional writes to specs |
| 2 | Confident | Reads all doc and spec files for cross-referencing | Necessary for section-level gap detection; file count is small enough to load fully |

2 assumptions made (2 confident, 0 tentative).
