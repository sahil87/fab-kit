# Tasks: Fix Documentation Consistency Drift

**Change**: 260218-5isu-fix-docs-consistency-drift
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Spec File Bulk Corrections

- [x] T001 [P] Replace all `/fab-init` → `/fab-setup` in 5 spec files: `docs/specs/glossary.md`, `docs/specs/architecture.md`, `docs/specs/overview.md`, `docs/specs/user-flow.md`, `docs/specs/templates.md` (excludes `skills.md` — handled by T004)
- [x] T002 [P] Replace "briefs" → "intakes" in `docs/specs/architecture.md` — change `(briefs, specs, tasks)` to `(intakes, specs, tasks)`
- [x] T003 [P] Remove or replace all `_init_scaffold.sh` references in `docs/specs/architecture.md` with `fab-sync.sh` / `scaffold/` directory approach

## Phase 2: Targeted Spec Rewrites

- [x] T004 Rewrite `/fab-init` section in `docs/specs/skills.md` as `/fab-setup` — replace the current `/fab-init` block with documentation of three subcommands (`config`, `constitution`, `migrations`), using `docs/memory/fab-workflow/setup.md` as source of truth
- [x] T005 Remove `/fab-update` node from Mermaid diagram in `docs/specs/user-flow.md` and replace any prose `/fab-update` references with `/fab-setup migrations`
- [x] T006 Fix `.status.yaml` documentation in `docs/specs/templates.md` — replace `archive:` → `hydrate:` in progress map, add missing fields (`change_type`, `confidence` block, `stage_metrics`)

## Phase 3: Memory File Corrections

- [x] T007 [P] Replace `/fab-init` → `/fab-setup` in 4 memory files: `docs/memory/fab-workflow/context-loading.md`, `docs/memory/fab-workflow/hydrate.md`, `docs/memory/fab-workflow/hydrate-specs.md`, `docs/memory/fab-workflow/specs-index.md`
- [x] T008 [P] Replace stale `lib/sync-workspace.sh` references with correct current paths in memory files: `docs/memory/fab-workflow/hydrate.md`, `docs/memory/fab-workflow/model-tiers.md`, `docs/memory/fab-workflow/templates.md` — verify actual content before replacing
- [x] T009 [P] Replace `/fab-update` → `/fab-setup migrations` in `docs/memory/fab-workflow/migrations.md` (preserve intentional deprecated references in `setup.md`)
- [x] T010 [P] Update `docs/memory/fab-workflow/kit-architecture.md` — remove `model-tiers.yaml` from directory tree, add `fab-fff.md` after `fab-ff.md` in skills listing

## Phase 4: Verification

- [x] T011 Run grep verification: confirm zero matches for `/fab-init`, `_init_scaffold.sh`, `/fab-update`, `lib/sync-workspace.sh` across `docs/specs/` and `docs/memory/` (excluding intentional deprecated references in `setup.md`)

---

## Execution Order

- T001, T002, T003 are independent (different content in same/different files) — can run in parallel
- T004, T005, T006 are independent (different files) but follow Phase 1 for `skills.md` clarity
- T007, T008, T009, T010 are independent — can run in parallel
- T011 depends on all previous tasks
