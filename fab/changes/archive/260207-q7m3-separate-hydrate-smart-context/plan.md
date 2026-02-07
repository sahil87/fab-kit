# Plan: Separate Doc Hydration from Init, Add Smart Context Loading, and Index fab/docs

**Change**: 260207-q7m3-separate-hydrate-smart-context
**Created**: 2026-02-07
**Proposal**: `proposal.md`
**Spec**: `spec.md`

## Summary

Extract source hydration from `/fab:init` into a new `/fab:hydrate` skill, enhance `_context.md` so all skills smart-load relevant docs from `fab/docs/`, and ensure hydration and archive always maintain navigable `index.md` files at both the top level and per domain. This is a structural refactor of skill files and documentation — no runtime code changes.

## Goals / Non-Goals

**Goals:**
- Clean separation: `/fab:init` = structural bootstrap only, `/fab:hydrate` = doc ingestion
- Every skill (except init/switch/status/hydrate) reads `fab/docs/index.md` as part of "Always Load"
- Selective domain-level doc loading based on the active change's scope
- Consistent index maintenance in both hydration and archive paths
- Updated fab-spec documentation reflecting the new skill and simplified init

**Non-Goals:**
- Changing the hydration algorithm itself (domain analysis, content merging) — just moving it
- Adding size/depth limits to context loading (deferred per proposal)
- Making `/fab:hydrate` callable without `fab/docs/` existing (it requires init first)
- Changing archive behavior beyond index maintenance

## Technical Context

- **Relevant stack**: Markdown skill files, `_context.md` shared preamble, shell scripts
- **Key dependencies**: `fab-setup.sh` glob pattern (`fab/.kit/skills/fab-*.md`) for auto-discovery
- **Constraints**: Pure prompt play (Constitution I) — all logic in markdown/shell. Idempotent operations (Constitution III). Markdown-only artifacts (Constitution IV).

## Decisions

1. **Extract, don't fork**: Move Phase 2 from `fab-init.md` verbatim into `fab-hydrate.md`, then remove it from init. This avoids divergence.
   - *Why*: Preserves tested hydration logic; single source of truth.
   - *Rejected*: Rewriting hydration from scratch in the new skill — risks introducing bugs and inconsistencies.

2. **`fab/docs/` must exist for hydrate**: `/fab:hydrate` checks for `fab/docs/` and aborts if missing, telling the user to run `/fab:init` first.
   - *Why*: Keeps the dependency clear — init creates structure, hydrate populates it.
   - *Rejected*: Auto-creating `fab/docs/` in hydrate — would blur the separation of concerns.

3. **Smart context loading in `_context.md` applies to all skills on active changes**: Move the "Centralized Doc Lookup" layer from "when writing or validating specs" to "when operating on an active change". Add `fab/docs/index.md` to "Always Load".
   - *Why*: Agents need domain awareness for planning, implementation, and review — not just spec writing.
   - *Rejected*: Per-skill opt-in — too much maintenance overhead and easy to miss.

4. **Index maintenance is embedded in hydrate and archive skill instructions**: Rather than creating a shared "index update" utility, each skill (hydrate, archive) includes inline instructions for updating indexes.
   - *Why*: Constitution I forbids system dependencies. Markdown skill instructions are the right abstraction.
   - *Rejected*: Shell script for index updates — would be brittle (parsing markdown tables in bash) and violate the "prompt play" spirit.

5. **Init with arguments shows redirect message**: When a user runs `/fab:init https://...`, instead of silently ignoring the arguments, init outputs a helpful redirect to `/fab:hydrate`.
   - *Why*: Better UX — users who remember the old interface get guided to the new one.
   - *Rejected*: Silently ignoring arguments — confusing, user thinks hydration happened.

## Risks / Trade-offs

- **Existing documentation drift**: Users reading old README/SKILLS docs may still try `/fab:init` with sources. Mitigated by the redirect message in init and updated docs.
- **Index format consistency**: Two skills (hydrate, archive) independently maintain indexes. If the format instructions diverge over time, indexes could become inconsistent. Mitigated by referencing `TEMPLATES.md` as the canonical format in both skills.
- **Context window pressure**: Loading `fab/docs/index.md` for every skill adds context. For large projects with many domains, this could matter. Acceptable for now — the index is a lightweight table, and selective loading means only relevant domain docs are loaded.

## File Changes

### New Files
- `fab/.kit/skills/fab-hydrate.md`: New skill containing all hydration logic (moved from `fab-init.md` Phase 2), plus index maintenance instructions

### Modified Files
- `fab/.kit/skills/fab-init.md`: Remove Phase 2 (source hydration), remove `[sources...]` arguments, add redirect message when arguments are passed, update output sections
- `fab/.kit/skills/_context.md`: Add `fab/docs/index.md` to "Always Load"; expand "Centralized Doc Lookup" from spec-writing-only to all-skills-on-active-change; add `/fab:hydrate` to exceptions list
- `fab/.kit/skills/fab-archive.md`: Add index maintenance instructions (update domain and top-level indexes after hydration)
- `doc/fab-spec/README.md`: Add `/fab:hydrate` to Quick Reference table; update `/fab:init` row; update "Hydrating Docs" section
- `doc/fab-spec/SKILLS.md`: Add `/fab:hydrate` section; update `/fab:init` section (remove source hydration); update Context Loading Convention section
- `doc/fab-spec/ARCHITECTURE.md`: Update bootstrap sequence to show `/fab:hydrate` as step 4; update re-running init paragraph

### Deleted Files
_(none)_
