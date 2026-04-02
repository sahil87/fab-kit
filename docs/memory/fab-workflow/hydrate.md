# Hydrate

**Domain**: fab-workflow

## Overview

`/docs-hydrate-memory [sources...|folders...]` is a standalone skill that operates in two modes: **ingest mode** (fetching URLs or reading `.md` files into `docs/memory/`) and **generate mode** (scanning the codebase for undocumented areas and producing structured memory files). Mode is determined automatically by argument type — no flags needed. It requires `docs/memory/` to exist (created by `/fab-setup`). See [hydrate-generate](hydrate-generate.md) for full generate mode requirements.

## Requirements

### Standalone Hydrate Skill

The system provides `/docs-hydrate-memory [sources...|folders...]` as an independent skill containing hydration and generation logic. It is defined in `$(fab kit-path)/skills/docs-hydrate-memory.md` and is auto-discovered by `sync/2-sync-workspace.sh`'s `*.md` glob pattern.

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
- Creates or merges memory files in `docs/memory/{domain}/`
- Creates/updates domain indexes (`docs/memory/{domain}/index.md`)
- Updates top-level index (`docs/memory/index.md`)
- Multiple sources are processed in a single pass; indexes updated once at the end

### Generate Mode Behavior

When arguments route to generate mode (no arguments or folder paths), the skill scans the codebase for undocumented areas, presents an interactive gap report, and generates structured memory files. See [hydrate-generate](hydrate-generate.md) for full requirements.

### Prerequisite

`/docs-hydrate-memory` requires `docs/memory/` to exist. If missing, it aborts with: "docs/memory/ not found. Run /fab-setup first to create the memory directory."

### Idempotent Hydration

Safe to run repeatedly with the same sources:
- New requirements from the source are added
- Existing requirements are updated if source content changed
- Manually-added content in memory files is preserved
- No duplication of requirements on re-hydration

### Index Maintenance

Every hydration operation maintains navigable indexes:
- **Top-level** (`docs/memory/index.md`): `| [domain](domain/index.md) | description | file-list |`
- **Domain-level** (`docs/memory/{domain}/index.md`): `| [file-name](file-name.md) | description | last-updated |`
- All links are relative (not absolute paths)
- Formats follow `docs/specs/templates.md`

## Design Decisions

### Extract Hydration from Init into Standalone Skill
**Decision**: Move Phase 2 verbatim from `fab-init.md` into `fab-hydrate.md`, then remove it from init.
**Why**: Preserves tested hydration logic; single source of truth. Clean separation — init = structure, hydrate = content.
**Rejected**: Rewriting hydration from scratch in the new skill — risks introducing bugs and inconsistencies.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Hydrate Requires docs/memory/ to Exist
**Decision**: `/docs-hydrate-memory` checks for `docs/memory/` and aborts if missing, directing user to run `/fab-setup` first.
**Why**: Keeps the dependency clear — init creates structure, hydrate populates it.
**Rejected**: Auto-creating `docs/memory/` in hydrate — would blur the separation of concerns.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Index Maintenance Embedded in Skill Instructions
**Decision**: Each skill (hydrate, archive) includes inline instructions for updating indexes rather than sharing a utility.
**Why**: Constitution I forbids system dependencies. Markdown skill instructions are the right abstraction level.
**Rejected**: Shell script for index updates — would be brittle (parsing markdown tables in bash) and violate the "prompt play" spirit.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `/fab-init` → `/fab-setup` (3 occurrences) and `lib/sync-workspace.sh` → `sync/2-sync-workspace.sh` in glob pattern reference |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated hydration target paths from `fab/memory/` to `docs/memory/` |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `lib/sync-workspace.sh` in glob pattern reference |
| 260214-r8kv-docs-skills-housekeeping | 2026-02-14 | Renamed skill from `/fab-hydrate` to `/docs-hydrate-memory`; updated skill file path, glob pattern reference, and all cross-references |
| 260208-4wg3-fix-hydrate-links | 2026-02-08 | Fixed stale `doc/fab-spec/TEMPLATES.md` reference in Index Maintenance to `docs/specs/templates.md` |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-k5od-hydrate-generate-mode | 2026-02-07 | Added generate mode — unified argument routing, dual-mode overview, cross-reference to hydrate-generate doc |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Created hydrate doc — extracted `/docs-hydrate-memory` as standalone skill from `/fab-init` Phase 2 |
