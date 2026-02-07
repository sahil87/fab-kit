# Proposal: Separate Doc Hydration from Init, Add Smart Context Loading, and Index fab/docs

**Change**: 260207-q7m3-separate-hydrate-smart-context
**Created**: 2026-02-07
**Status**: Draft

## Why

`/fab:init` currently has a dual responsibility: structural bootstrap **and** doc hydration from external sources. These are conceptually different operations — bootstrap runs once per project, hydration runs whenever new sources need ingesting. Coupling them means the user must invoke `/fab:init` (which sounds like a one-time setup command) every time they want to hydrate docs. Separating these concerns makes the interface clearer and enables a second improvement: now that `fab/docs/` is properly populated after hydration, all `/fab:*` skills can smart-load relevant documentation from `fab/docs/` before execution — but only if the docs are properly indexed with navigable `index.md` files at each level. Today `fab/docs/index.md` is a bare skeleton and domain indexes don't exist yet in the template/spec for initial hydration.

## What Changes

- **Extract `/fab:hydrate` as a new skill**: Move the source hydration logic (Phase 2 of `fab-init.md`) into a new `fab-hydrate.md` skill. `/fab:init` becomes purely structural bootstrap — it no longer accepts `[sources...]` arguments.
- **Update `/fab:init`**: Remove Phase 2 (source hydration) and all `[sources...]` argument handling. Init only does structural bootstrap. Update output, error handling, and idempotency docs accordingly.
- **Smart context loading in `_context.md`**: Enhance the "Always Load" and "Centralized Doc Lookup" layers so that every skill (except init/switch/status) reads `fab/docs/index.md` and then selectively loads the relevant domain indexes and doc files based on the change's scope. This replaces the current pattern where only spec-writing skills consult docs.
- **Proper indexing of `fab/docs/`**: Update the hydration logic (now in `fab-hydrate.md`) and the archive hydration logic to always maintain well-structured `index.md` files — both the top-level `fab/docs/index.md` and per-domain `fab/docs/{domain}/index.md` files — with proper cross-references so agents can navigate to relevant sections efficiently.
- **Update fab spec documentation**: Update `doc/fab-spec/` files (README, SKILLS, ARCHITECTURE) to reflect the new `/fab:hydrate` command and the separation of concerns.

## Affected Docs

### New Docs
- `fab-workflow/hydrate`: Documentation for the new `/fab:hydrate` skill — arguments, behavior, hydration rules

### Modified Docs
- `fab-workflow/init`: Remove source hydration responsibility, simplify to structural bootstrap only
- `fab-workflow/context-loading`: Update context loading convention to describe smart doc loading for all skills

### Removed Docs
_(none)_

## Impact

- **Skill files affected**: `fab/.kit/skills/fab-init.md` (remove Phase 2), `fab/.kit/skills/_context.md` (enhance context loading), new `fab/.kit/skills/fab-hydrate.md`
- **Spec docs affected**: `doc/fab-spec/README.md`, `doc/fab-spec/SKILLS.md`, `doc/fab-spec/ARCHITECTURE.md`
- **Symlink setup**: `fab/.kit/scripts/fab-setup.sh` will automatically discover the new `fab-hydrate.md` skill via its glob pattern — no changes needed there
- **Template affected**: `fab/.kit/templates/` — the centralized doc format templates already define `index.md` structure; this change ensures hydration **consistently creates and maintains** those indexes
- **No breaking changes to existing changes**: Active changes continue to work. The separation is purely at the skill interface level.

## Open Questions

- [DEFERRED] Should `/fab:hydrate` also be callable standalone (without prior `/fab:init`) if `fab/docs/` already exists? (Assumed: yes — hydrate only needs `fab/docs/` to exist, which init creates. If someone manually creates `fab/docs/`, hydrate should still work.)
- [DEFERRED] Should the smart context loading have a size/depth limit to avoid loading too many docs into context for large projects? (Assumed: the agent reads indexes first and selectively loads only relevant domain docs, which is sufficient for now.)
