# Intake: Fix Documentation Consistency Drift

**Change**: 260218-5isu-fix-docs-consistency-drift
**Created**: 2026-02-18
**Status**: Draft

## Origin

> `/internal-consistency-check` was run against all three sources of truth (specs in `docs/specs/`, memory in `docs/memory/`, implementation in `fab/.kit/` and `src/`). The scan found 12 findings: 6 critical, 6 minor. All stem from recent renames and reorganizations that updated the implementation but left specs and/or memory stale.

## Why

1. **Problem**: Specs and memory contain stale references to renamed skills (`/fab-init`, `/fab-update`), removed files (`model-tiers.yaml`, `_init_scaffold.sh`), old script paths (`lib/sync-workspace.sh`), and an incorrect stage name (`archive` instead of `hydrate`) in the `.status.yaml` template spec. The specs also omit three fields actually present in the status template.
2. **Consequence**: Anyone reading specs will attempt commands that don't exist (`/fab-init`, `/fab-update`), look for files in wrong locations, or generate status files with incorrect stage names. Memory files similarly mislead.
3. **Approach**: Bulk find-and-replace plus targeted rewrites. No implementation changes — this is purely docs/specs/memory alignment.

## What Changes

### Fix: Replace `/fab-init` → `/fab-setup` (~20 locations)

Replace all occurrences across specs and memory:

**Specs** (all instances):
- `docs/specs/glossary.md` — lines 44, 58, 103
- `docs/specs/architecture.md` — lines 20, 303, 371, 374, 399, 406, 413, 415, 431, 434
- `docs/specs/overview.md` — line 105
- `docs/specs/skills.md` — lines 24, 58, 72-104, 123
- `docs/specs/user-flow.md` — line 82
- `docs/specs/templates.md` — line 493

**Memory** (all instances):
- `docs/memory/fab-workflow/context-loading.md` — lines 13, 79
- `docs/memory/fab-workflow/hydrate.md` — lines 7, 45, 72
- `docs/memory/fab-workflow/hydrate-specs.md` — line 29
- `docs/memory/fab-workflow/specs-index.md` — line 29

### Fix: Replace `archive:` → `hydrate:` in template spec

In `docs/specs/templates.md`:
- Line 21: progress map keys list `archive` → change to `hydrate`
- Line 39: template shows `archive: pending` → change to `hydrate: pending`

### Fix: Replace `/fab-update` → `/fab-setup migrations`

- `docs/specs/user-flow.md` line 88: remove `/fab-update` from diagram or replace with `/fab-setup migrations`
- `docs/memory/fab-workflow/migrations.md` line 78: replace `/fab-update` with `/fab-setup migrations`

### Fix: Replace `lib/sync-workspace.sh` → `fab-sync.sh` in memory

- `docs/memory/fab-workflow/hydrate.md` — line 13
- `docs/memory/fab-workflow/model-tiers.md` — lines 42, 78
- `docs/memory/fab-workflow/templates.md` — line 100

### Fix: Replace "briefs" → "intakes" in specs

- `docs/specs/architecture.md` line 358: "change artifacts (briefs, specs, tasks)" → "(intakes, specs, tasks)"

### Add: Rewrite `/fab-init` section in `skills.md` as `/fab-setup`

Replace `docs/specs/skills.md` lines 72-104 with a `/fab-setup` section documenting three subcommands:
- `/fab-setup config [section]` — create/update `fab/config.yaml`
- `/fab-setup constitution` — create/amend `fab/constitution.md`
- `/fab-setup migrations [file]` — run version migrations

Use `docs/memory/fab-workflow/setup.md` as the source of truth for current behavior.

### Add: Missing `.status.yaml` fields to template spec

Add to `docs/specs/templates.md` (lines 27-45) the three missing fields:
- `change_type: feature` (line 4 of implementation template)
- `confidence:` block with `certain`, `confident`, `tentative`, `unresolved`, `score` (lines 17-22)
- `stage_metrics: {}` (line 23)

Use the actual `fab/.kit/templates/status.yaml` as reference.

### Add: `fab-fff.md` to kit architecture listing

- `docs/memory/fab-workflow/kit-architecture.md` lines 19-35: add `fab-fff.md` after `fab-ff.md`

### Remove: `model-tiers.yaml` from kit architecture

- `docs/memory/fab-workflow/kit-architecture.md` line 18: remove `model-tiers.yaml` reference (absorbed into `config.yaml` as of v0.8.0)

### Remove: `_init_scaffold.sh` references from specs

- `docs/specs/architecture.md` lines 37, 96, 371, 405, 413: remove or replace references to `_init_scaffold.sh` with documentation of the `fab/.kit/scaffold/` directory approach

### Remove: `/fab-update` from user-flow diagram

- `docs/specs/user-flow.md` line 88: remove `UPDATE["/fab-update"]` node from the diagram

## Affected Memory

- `fab-workflow/context-loading`: (modify) fix `/fab-init` → `/fab-setup`
- `fab-workflow/hydrate`: (modify) fix `/fab-init` → `/fab-setup`, fix `lib/sync-workspace.sh` → `fab-sync.sh`
- `fab-workflow/hydrate-specs`: (modify) fix `/fab-init` → `/fab-setup`
- `fab-workflow/specs-index`: (modify) fix `/fab-init` → `/fab-setup`
- `fab-workflow/model-tiers`: (modify) fix `lib/sync-workspace.sh` → `fab-sync.sh`
- `fab-workflow/templates`: (modify) fix `lib/sync-workspace.sh` → `fab-sync.sh`
- `fab-workflow/kit-architecture`: (modify) remove `model-tiers.yaml`, add `fab-fff.md`
- `fab-workflow/migrations`: (modify) fix `/fab-update` → `/fab-setup migrations`

## Impact

- **Specs**: 7 files modified (`glossary.md`, `architecture.md`, `overview.md`, `skills.md`, `user-flow.md`, `templates.md`, `packages.md` if applicable)
- **Memory**: 8 files modified (listed above)
- **Implementation**: No changes — implementation is already correct
- **Risk**: Low — all changes are text replacements in documentation, no behavioral impact

## Open Questions

None — all findings are straightforward corrections with clear targets.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No implementation changes needed | Consistency check confirmed implementation is already correct; only docs/specs/memory are stale | S:95 R:95 A:95 D:95 |
| 2 | Certain | Use memory as source of truth for `/fab-setup` rewrite | Constitution mandates memory as authoritative post-implementation source (§II) | S:90 R:90 A:95 D:90 |
| 3 | Confident | Remove `/fab-update` from user-flow diagram entirely | No replacement needed — `/fab-setup migrations` is not a diagram-level flow, it's a maintenance subcommand | S:70 R:85 A:80 D:70 |
| 4 | Confident | Line numbers may have shifted since scan | Consistency check ran at a point-in-time; edits use content matching not line offsets | S:75 R:90 A:85 D:80 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
