# Hydrate

**Domain**: fab-workflow

## Overview

`/fab:hydrate [sources...|folders...]` is a standalone skill that operates in two modes: **ingest mode** (fetching URLs or reading `.md` files into `fab/docs/`) and **generate mode** (scanning the codebase for undocumented areas and producing structured docs). Mode is determined automatically by argument type — no flags needed. It requires `fab/docs/` to exist (created by `/fab:init`). See [hydrate-generate](hydrate-generate.md) for full generate mode requirements.

## Requirements

### Standalone Hydrate Skill

The system provides `/fab:hydrate [sources...|folders...]` as an independent skill containing hydration and generation logic. It is defined in `fab/.kit/skills/fab-hydrate.md` and is auto-discovered by `fab-setup.sh`'s `fab-*.md` glob pattern.

### Argument-Driven Mode Selection

The skill determines its operating mode from argument type:

| Argument type | Detection rule | Mode |
|---|---|---|
| No arguments | Argument list is empty | Generate (scan from project root) |
| URL | Matches `notion.so`, `notion.site`, `linear.app`, or `http(s)://` | Ingest |
| Markdown file | Path ends with `.md` | Ingest |
| Folder | Path resolves to an existing directory | Generate |

Mixed-mode invocations (e.g., a URL and a folder) SHALL be rejected with an error.

### Ingest Mode Behavior

When arguments route to ingest mode:

- Fetches/reads each source independently
- Analyzes content and maps to domains
- Creates or merges doc files in `fab/docs/{domain}/`
- Creates/updates domain indexes (`fab/docs/{domain}/index.md`)
- Updates top-level index (`fab/docs/index.md`)
- Multiple sources are processed in a single pass; indexes updated once at the end

### Generate Mode Behavior

When arguments route to generate mode (no arguments or folder paths), the skill scans the codebase for undocumented areas, presents an interactive gap report, and generates structured docs. See [hydrate-generate](hydrate-generate.md) for full requirements.

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
| 260207-k5od-hydrate-generate-mode | 2026-02-07 | Added generate mode — unified argument routing, dual-mode overview, cross-reference to hydrate-generate doc |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Created hydrate doc — extracted `/fab:hydrate` as standalone skill from `/fab:init` Phase 2 |
