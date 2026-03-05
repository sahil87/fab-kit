# Proposal: Fix broken template links in fab-hydrate

**Change**: 260208-4wg3-fix-hydrate-links
**Created**: 2026-02-08
**Status**: Draft

## Why

Commit 9329bd5 moved fab spec files from `doc/fab-spec/` to `fab/specs/`, but three template reference links in `fab/.kit/skills/fab-hydrate.md` were not updated. These broken links point users to a non-existent `doc/fab-spec/TEMPLATES.md` path when they need to understand domain index, individual doc, and top-level index formats during hydration.

## What Changes

- Update 3 broken relative links in `fab/.kit/skills/fab-hydrate.md`:
  - **Line 97**: Domain Index format link → `../../specs/templates.md#domain-index-fabdocsdomainindexmd`
  - **Line 104**: Centralized Doc format link → `../../specs/templates.md#individual-doc-fabdomainnamemd`
  - **Line 124**: Top-Level Index format link → `../../specs/templates.md#top-level-index-fabdocsindexmd`
- All three links change from `../../doc/fab-spec/TEMPLATES.md#...` to `../../specs/templates.md#...`

## Affected Docs

### New Docs

(none)

### Modified Docs

(none — this change modifies a kit skill file, not a centralized doc)

### Removed Docs

(none)

## Impact

- **Affected file**: `fab/.kit/skills/fab-hydrate.md` (lines 97, 104, 124)
- **Scope**: Minimal — 3 link path replacements in a single file
- **Risk**: Very low — only changes markdown link targets, no behavioral change to the skill logic
- **Note**: The exploration found ~18 additional files with old `doc/fab-spec/` references (in `fab/docs/fab-workflow/`, archived changes, and specs). Those are out of scope for this change — they could be addressed in a follow-up.
<!-- assumed: Scoping to fab-hydrate.md only — the backlog item [v7qm] specifically calls out lines 97, 104, 124 in fab-hydrate; broader cleanup is a separate concern -->

## Open Questions

(none — the fix is fully deterministic from the commit history and file locations)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Scope limited to fab-hydrate.md only | Backlog item [v7qm] specifically targets lines 97, 104, 124; broader `doc/fab-spec/` cleanup is a separate change |
| 2 | Confident | Anchor slugs unchanged after file move | The heading text in `fab/specs/templates.md` matches the original anchors |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
