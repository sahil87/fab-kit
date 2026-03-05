# Tasks: Move 5 Cs and VERSION into fab/project/

**Change**: 260219-wq0e-move-5cs-to-project-folder
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/project/` directory, move 6 files (`config.yaml`, `constitution.md`, `context.md`, `code-quality.md`, `code-review.md`, `VERSION`) from `fab/` to `fab/project/`, and bump `fab/.kit/VERSION` to `0.10.0`

## Phase 2: Shell Script Updates

- [x] T002 Update `fab/.kit/scripts/lib/preflight.sh` — change `$fab_root/config.yaml` and `$fab_root/constitution.md` to `$fab_root/project/config.yaml` and `$fab_root/project/constitution.md`
- [x] T003 Update `fab/.kit/scripts/lib/changeman.sh` — change `$FAB_ROOT/config.yaml` to `$FAB_ROOT/project/config.yaml` in `switch` subcommand
- [x] T004 [P] Update `fab/.kit/scripts/fab-upgrade.sh` — change `$fab_dir/VERSION` to `$fab_dir/project/VERSION` (2 occurrences: existence check and cat)
- [x] T005 [P] Update `fab/.kit/scripts/batch-fab-switch-change.sh` — change `${FAB_DIR}/config.yaml` to `${FAB_DIR}/project/config.yaml`
- [x] T006 Update `fab/.kit/sync/2-sync-workspace.sh` — change `$fab_dir/VERSION` to `$fab_dir/project/VERSION` (section 1b: existence check, create, preserve), change `$fab_dir/config.yaml` to `$fab_dir/project/config.yaml` (section 1b existing-project detection and section 3+4 model_tiers)

## Phase 3: Scaffold & Skills

- [x] T007 Move scaffold files: `fab/.kit/scaffold/fab/{config.yaml,constitution.md,context.md,code-quality.md,code-review.md}` → `fab/.kit/scaffold/fab/project/` and update the header comment in `scaffold/fab/project/config.yaml` from `# fab/config.yaml` to `# fab/project/config.yaml`
- [x] T008 Update `fab/.kit/skills/_context.md` — change all 5 C paths in "Always Load" section and state derivation `(none)` rule from `fab/{file}` to `fab/project/{file}` (8 occurrences)
- [x] T009 Update `fab/.kit/skills/fab-setup.md` — change all path references from `fab/{file}` to `fab/project/{file}` and from `fab/.kit/scaffold/fab/{file}` to `fab/.kit/scaffold/fab/project/{file}` (70 occurrences)
- [x] T010 [P] Update remaining skill files with path references: `_generation.md` (2), `fab-continue.md` (5), `fab-status.md` (3), `fab-new.md` (1), `fab-switch.md` (2), `fab-help.md` (1), `internal-consistency-check.md` (3)

## Phase 4: Documentation & Integration

- [x] T011 Create `fab/.kit/migrations/0.9.0-to-0.10.0.md` — migration instructions for moving 6 files from `fab/` to `fab/project/`, following the pattern in existing migration files
- [x] T012 [P] Update `README.md` — change all 5 C and VERSION path references to `fab/project/` paths and update directory structure diagrams (8 occurrences)
- [x] T013 [P] Update `docs/memory/fab-workflow/` files — update path references in: `configuration.md` (8), `context-loading.md` (7), `kit-architecture.md` (7), `setup.md` (5), `distribution.md` (4), `migrations.md` (11), `planning-skills.md` (2), `model-tiers.md` (2), `change-lifecycle.md` (2), `index.md` (1), `preflight.md` (1), `schemas.md` (1)
- [x] T014 [P] Update `docs/specs/` files — update path references in: `skills.md` (10), `architecture.md` (6), `glossary.md` (2)
- [x] T015 Update test files — `src/lib/preflight/test.bats` (6 occurrences: config.yaml and constitution.md fixture paths), `src/lib/sync-workspace/test.bats` (2 occurrences), `src/lib/sync-workspace/SPEC-sync-workspace.md` (2 occurrences)
- [x] T016 Regenerate agent files by running `fab/.kit/scripts/fab-sync.sh`, then verify no stale `fab/config.yaml` (without `project/`) references remain outside of changelog entries and generated artifacts

---

## Execution Order

- T001 blocks all subsequent tasks (files must be moved first)
- T002, T003, T004, T005 are independent of each other
- T006 blocks T016 (sync script must be updated before running it)
- T007 blocks T016 (scaffold must be moved before sync regenerates)
- T008, T009, T010 are independent of each other but block T016
- T011–T015 are independent of each other and of Phase 2–3
- T016 depends on T006, T007, T008, T009, T010 (all script/skill updates complete)
