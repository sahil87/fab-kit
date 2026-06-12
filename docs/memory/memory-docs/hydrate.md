---
description: "`/docs-hydrate-memory` skill — argument routing, dual-mode (ingest + generate), hydration rules, mechanical index regen via `fab memory-index`"
---
# Hydrate

**Domain**: memory-docs

## Overview

`/docs-hydrate-memory [sources...|folders...]` is a standalone skill that operates in two modes: **ingest mode** (fetching URLs or reading `.md` files into `docs/memory/`) and **generate mode** (scanning the codebase for undocumented areas and producing structured memory files). Mode is determined automatically by argument type — no flags needed. It requires `docs/memory/` to exist (created by `/fab-setup`). See [hydrate-generate](hydrate-generate.md) for full generate mode requirements.

> **Distinct from pipeline hydrate**: This file documents the standalone `/docs-hydrate-memory` skill. The `/fab-continue` pipeline hydrate stage (which advances a change from `review: done` to `hydrate: done` and updates `docs/memory/` from the change's spec/plan) is documented in [execution-skills](../pipeline/execution-skills.md) under "Hydrate Behavior (via `/fab-continue`)". The pipeline hydrate stage reads `## Deletion Candidates` from `plan.md` informationally (per execution-skills Hydrate Behavior Step 3) — see that file for the authoritative behavior.

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
- Creates or merges memory files in `docs/memory/{domain}/`, authoring each file's `description:` frontmatter
- Regenerates all indexes (root + every domain) mechanically via `fab memory-index` — never by hand-editing rows
- Multiple sources are processed in a single pass; `fab memory-index` runs once at the end

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

Every hydration operation regenerates the navigable indexes **mechanically** via `fab memory-index` — the skill never hand-edits index rows:
- **Top-level** (`docs/memory/index.md`): domains-only — `| Domain | Description |`. The legacy inlined per-file "Memory Files" column was dropped (tciy); per-domain descriptions come from each domain `index.md`'s `description:` frontmatter (round-tripped by the generator).
- **Domain-level** (`docs/memory/{domain}/index.md`): file rows — `| File | Description | Last Updated |`. Each row's Description is read from the topic file's `description:` frontmatter; "Last Updated" is git-stamped from ONE batched `git -c core.quotepath=off log --date=short --format=%x00%ad --name-only -- docs/memory` pass (newest-first; the first date seen per path wins, keyed relative to the git top-level — output-equivalent to the old per-file `git log -1 --date=short` defaults, which is retained solely as the fallback when the batched call fails), degrading to `—` for uncommitted files; never hand-stamped (pw3k).
- The command is the single writer of both index levels — output is byte-stable / idempotent, so re-running produces no diff and any post-merge conflict auto-resolves by re-running `fab memory-index`.
- Memory writers MUST author the `description:` frontmatter on every new/modified topic file so the regenerated index has content.
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

### Memory Index Maintenance is a Mechanical `fab memory-index` Call
**Decision**: The hydrate skill regenerates `docs/memory/` indexes by invoking the deterministic `fab memory-index` Go subcommand, not by hand-editing index rows in skill instructions.
**Why**: Hand-maintained per-row index cells (`description` + `Last Updated`) were the dominant merge-conflict and drift source — they get rewritten on nearly every memory edit. A generated, byte-stable index removes the hand-edit entirely, so two branches can never produce conflicting hand-edits to the same row, and any residual textual conflict auto-resolves by re-running the command. The render is a pure function of folder contents + `description:` frontmatter + git dates, mirroring the established `internal/prmeta` Render/Gather pattern.
**Rejected**: Markdown skill instructions for index updates (the prior approach) — they silently drift (the old root roster listed 18 files when 20+ existed; hand-stamped dates were already wrong). A bespoke bash table-parser was also rejected earlier as brittle; the deterministic Go helper is admitted by the constitution (cf. `prmeta`/`impact`/`score`) and is fully unit-testable.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context (original inline-instruction design); *Superseded by*: 260607-tciy-memory-tree-shape-rebalance (mechanical `fab memory-index`)

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260612-pw3k-operator-pane-perf-error-surfacing | 2026-06-12 | Index Maintenance "Last Updated" date-sourcing corrected to the shipped mechanism (binary-review B5, F34): `fab memory-index` now sources dates from ONE batched `git -c core.quotepath=off log --date=short --format=%x00%ad --name-only -- docs/memory` pass (newest-first, first date per path wins, keyed relative to the git top-level) instead of one `git log -1` spawn per memory file; the per-file call is kept solely as fallback when the batched call fails (a per-path miss = uncommitted → `—`, as before). Rendered index output is byte-identical. Mechanism description only — no skill-behavior change. |
| 260607-tciy-memory-tree-shape-rebalance | 2026-06-07 | Index Maintenance rewired to a mechanical `fab memory-index` call — the skill no longer hand-edits index rows. Ingest-mode behavior bullets updated (author `description:` frontmatter on files; run `fab memory-index` once at end). Index Maintenance requirement: root index is now **domains-only** (`\| Domain \| Description \|`; the inlined per-file "Memory Files" / `file-list` column is dropped), domain rows are `\| File \| Description \| Last Updated \|` with descriptions from each file's `description:` frontmatter and git-stamped "Last Updated". Renamed the "Index Maintenance Embedded in Skill Instructions" design decision → "Memory Index Maintenance is a Mechanical `fab memory-index` Call" (superseded — the hand-maintained rows were the merge-conflict + drift source). |
| 260507-ogf2-restrain-ai-code-bloat | 2026-05-07 | Added Overview disambiguation: this file documents the standalone `/docs-hydrate-memory` skill; the `/fab-continue` pipeline hydrate stage (now reads `## Deletion Candidates` from `plan.md` informationally as Step 3) is documented in [execution-skills](../pipeline/execution-skills.md). No changes to `/docs-hydrate-memory` behavior. |
| 260423-qszh-merge-tasks-checklist | 2026-05-06 | Reviewed for `tasks.md`/`checklist.md` references in light of the apply-stage artifact merge into `plan.md`. No live references found — this file documents the standalone `/docs-hydrate-memory` skill (URL/folder ingest + generate from codebase), not the `/fab-continue` pipeline-stage hydrate behavior, and it never named those legacy artifacts. No changes required. |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `/fab-init` → `/fab-setup` (3 occurrences) and `lib/sync-workspace.sh` → `sync/2-sync-workspace.sh` in glob pattern reference |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated hydration target paths from `fab/memory/` to `docs/memory/` |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `lib/sync-workspace.sh` in glob pattern reference |
| 260214-r8kv-docs-skills-housekeeping | 2026-02-14 | Renamed skill from `/fab-hydrate` to `/docs-hydrate-memory`; updated skill file path, glob pattern reference, and all cross-references |
| 260208-4wg3-fix-hydrate-links | 2026-02-08 | Fixed stale `doc/fab-spec/TEMPLATES.md` reference in Index Maintenance to `docs/specs/templates.md` |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-k5od-hydrate-generate-mode | 2026-02-07 | Added generate mode — unified argument routing, dual-mode overview, cross-reference to hydrate-generate doc |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Created hydrate doc — extracted `/docs-hydrate-memory` as standalone skill from `/fab-init` Phase 2 |
