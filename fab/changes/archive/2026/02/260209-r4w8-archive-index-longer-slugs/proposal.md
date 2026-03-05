# Proposal: Add archive index and allow longer folder slugs

**Change**: 260209-r4w8-archive-index-longer-slugs
**Created**: 2026-02-09
**Status**: Draft

## Why

As archived changes accumulate, there's no way to search or browse what was done without opening each folder individually. An auto-maintained index in the archive folder makes past changes discoverable. Additionally, the current 2-4 word slug limit produces cryptic folder names that don't convey enough about the change's purpose — expanding the range improves self-documentation across the archive, git branches, and file listings.

## What Changes

- **New: archive `index.md`** — `/fab-archive` generates and maintains `fab/changes/archive/index.md`, a bullet list mapping each archived change folder name to a 1-2 sentence description pulled from the proposal's Why section
- **Modified: `/fab-archive` behavior** — after moving the change folder to archive, append an entry to `index.md` (or create it with a full backfill of existing entries if it doesn't exist yet)
- **Modified: `/fab-new` slug generation** — expand the slug word count constraint from 2-4 words to 2-6 words
- **Modified: `/fab-discuss` slug generation** — same 2-6 word constraint (uses the same naming rules as `/fab-new`)

## Affected Docs

### New Docs

_None_ — `index.md` lives in the archive folder, not in `fab/docs/`.

### Modified Docs

- `fab-workflow/execution-skills`: `/fab-archive` gains a new step (index maintenance)
- `fab-workflow/change-lifecycle`: archive step now includes index update
- `fab-workflow/planning-skills`: `/fab-new` slug constraints updated to 2-6 words
- `fab-workflow/configuration`: naming convention slug length updated

### Removed Docs

_None._

## Impact

- **Skills affected**: `fab-archive` (new index step), `fab-new` (slug constraint), `fab-discuss` (slug constraint)
- **Templates**: None — the archive index is generated dynamically, not from a template
- **Existing archives**: backfilled on first `/fab-archive` run that creates `index.md`
- **Git branches**: longer slugs mean longer branch names, but 2-6 words keeps them reasonable

## Open Questions

_None — all decisions resolved during discussion._
