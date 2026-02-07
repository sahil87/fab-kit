# Hydrate

**Domain**: fab-workflow

## Overview

`/fab:hydrate [sources...]` is a standalone skill that ingests external documentation into `fab/docs/`. It handles fetching content from URLs or reading local files, analyzing and mapping content to domains, creating or merging doc files, and maintaining both top-level and domain-level indexes. It requires `fab/docs/` to exist (created by `/fab:init`).

## Requirements

### Standalone Hydrate Skill

The system provides `/fab:hydrate [sources...]` as an independent skill containing all source hydration logic. It is defined in `fab/.kit/skills/fab-hydrate.md` and is auto-discovered by `fab-setup.sh`'s `fab-*.md` glob pattern.

- Accepts one or more sources (URLs, local paths)
- Fetches/reads each source independently
- Analyzes content and maps to domains
- Creates or merges doc files in `fab/docs/{domain}/`
- Creates/updates domain indexes (`fab/docs/{domain}/index.md`)
- Updates top-level index (`fab/docs/index.md`)
- Multiple sources are processed in a single pass; indexes updated once at the end

### Prerequisite

`/fab:hydrate` requires `fab/docs/` to exist. If missing, it aborts with: "fab/docs/ not found. Run /fab:init first to create the docs directory."

### Idempotent Hydration

Safe to run repeatedly with the same sources:
- New requirements from the source are added
- Existing requirements are updated if source content changed
- Manually-added content in docs is preserved
- No duplication of requirements on re-hydration

### Index Maintenance

Every hydration operation maintains navigable indexes:
- **Top-level** (`fab/docs/index.md`): `| [domain](domain/index.md) | description | doc-list |`
- **Domain-level** (`fab/docs/{domain}/index.md`): `| [doc-name](doc-name.md) | description | last-updated |`
- All links are relative (not absolute paths)
- Formats follow `doc/fab-spec/TEMPLATES.md`

## Design Decisions

### Extract Hydration from Init into Standalone Skill
**Decision**: Move Phase 2 verbatim from `fab-init.md` into `fab-hydrate.md`, then remove it from init.
**Why**: Preserves tested hydration logic; single source of truth. Clean separation — init = structure, hydrate = content.
**Rejected**: Rewriting hydration from scratch in the new skill — risks introducing bugs and inconsistencies.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Hydrate Requires fab/docs/ to Exist
**Decision**: `/fab:hydrate` checks for `fab/docs/` and aborts if missing, directing user to run `/fab:init` first.
**Why**: Keeps the dependency clear — init creates structure, hydrate populates it.
**Rejected**: Auto-creating `fab/docs/` in hydrate — would blur the separation of concerns.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Index Maintenance Embedded in Skill Instructions
**Decision**: Each skill (hydrate, archive) includes inline instructions for updating indexes rather than sharing a utility.
**Why**: Constitution I forbids system dependencies. Markdown skill instructions are the right abstraction level.
**Rejected**: Shell script for index updates — would be brittle (parsing markdown tables in bash) and violate the "prompt play" spirit.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Created hydrate doc — extracted `/fab:hydrate` as standalone skill from `/fab:init` Phase 2 |
