# Tasks: Relocate Kit to System Cache

**Change**: 260402-gnx5-relocate-kit-to-system-cache
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/kitpath/kitpath.go` — shared `KitDir()` utility that resolves `os.Executable()` → `filepath.EvalSymlinks()` → `filepath.Join(dir, "kit")`. Include `KitDir_test.go`.
- [x] T002 Add `fab kit-path` subcommand to `src/go/fab/cmd/fab/` — reads `fab_version` from config, resolves `~/.fab-kit/versions/{version}/kit/`, prints path to stdout. Include test.
- [x] T003 Hardcode `defaultRepo` constant — add `const defaultRepo = "sahil87/fab-kit"` in `src/go/fab-kit/internal/` (or shared location), remove all `kit.conf` reads from Go code.

## Phase 2: Core Implementation

- [x] T004 [P] Update `src/go/fab/internal/change/change.go` — replace `filepath.Join(fabRoot, ".kit", "templates", "status.yaml")` with `kitpath.KitDir()` resolution.
- [x] T005 [P] Update `src/go/fab/internal/preflight/preflight.go` — `checkSyncStaleness()` reads VERSION from `kitpath.KitDir()` instead of `filepath.Join(fabRoot, ".kit", "VERSION")`.
- [x] T006 [P] Update `src/go/fab/cmd/fab/fabhelp.go` — replace `filepath.Join(fabRoot, ".kit")` with `kitpath.KitDir()` for skill scanning.
- [x] T007 [P] Update `src/go/fab/internal/hooklib/sync.go` — replace `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/` command construction with inline `fab hook <subcommand>` commands. Remove `hooksDir` parameter dependency. Update `DefaultMappings` to map to `fab hook` commands directly.
- [x] T008 [P] Update `src/go/fab-kit/internal/hooksync.go` — same inline hook changes as T007. Remove old-style hook path migration logic.
- [x] T009 Update `src/go/fab-kit/internal/init.go` — remove `fab/.kit/` copy step. Init should set `fab_version`, run scaffold from cache kit, deploy skills via sync. No `fab/.kit/` created in project.
- [x] T010 Update `src/go/fab-kit/internal/upgrade.go` — remove `fab/.kit/` copy step. Upgrade should update `fab_version` in config, re-run sync for skill deployment. No `fab/.kit/` touched.
- [x] T011 [P] Update `src/go/fab-kit/internal/download.go` — update archive prefix handling (strip `kit/` or `.kit/` prefix as appropriate for cache extraction).
- [x] T012 [P] Update `src/go/fab-kit/internal/sync.go` — resolve kit content from cache (version-resolved) instead of `fab/.kit/`. Remove `fab/.kit-migration-version` management if no longer needed.
- [x] T013 Update all affected `*_test.go` files — `src/go/fab/internal/hooklib/sync_test.go`, `src/go/fab-kit/internal/hooksync_test.go`, `src/go/wt/cmd/init_test.go`, `src/go/wt/cmd/create_test.go`. Update hardcoded `fab/.kit/` paths in test fixtures.
- [x] T014 [P] Update user-facing CLI messages — `src/go/fab-kit/cmd/fab-kit/main.go` ("Upgrade fab/.kit/"), `src/go/fab-kit/cmd/fab/main.go` (help text), `src/go/fab-kit/internal/init.go` ("Populating fab/.kit/..."), `src/go/fab-kit/internal/upgrade.go` ("Updating fab/.kit/...").

## Phase 3: Skill & Build Updates

- [x] T015 Move `fab/.kit/` → `src/kit/` in the source repo. Remove `kit.conf` and `hooks/` directory from the moved content.
- [x] T016 [P] Update `justfile` — change all `fab/.kit/` references to `src/kit/`.
- [x] T017 [P] Update `scripts/release.sh` — change `kit_dir` to `src/kit`.
- [x] T018 [P] Update `scripts/install.sh` — update to reference new paths if needed.
- [x] T019 [P] Update `.gitignore` — remove `fab/.kit/bin/*`, `!fab/.kit/bin/.gitkeep` entries.
- [x] T020 [P] Update `.github/copilot-code-review.yml` — change `fab/.kit/**` to `src/kit/**`.
- [x] T021 Update skill files in `src/kit/skills/` — replace `fab/.kit/templates/` references with `$(fab kit-path)/templates/` in `_generation.md`. Remove test-build guard from `_preamble.md`. Update `fab/.kit/skills/` references to deployed `.claude/skills/` paths where appropriate. Update `_cli-fab.md` references.
- [x] T022 Write migration file `src/kit/migrations/{FROM}-to-{TO}.md` — verify cache, inline hooks in settings.local.json, remove `fab/.kit/` from project, clean `.envrc` and `.gitignore`.
- [x] T023 [P] Update `.envrc` in source repo — remove `PATH_add fab/.kit/scripts` line.
- [x] T024 [P] Update `src/kit/scaffold/fragment-.gitignore` — remove `fab/.kit-sync-version` entry if still present.

## Phase 4: Documentation

- [x] T025 [P] Update `docs/memory/fab-workflow/kit-architecture.md` — directory structure, path resolution, hook removal.
- [x] T026 [P] Update `docs/memory/fab-workflow/distribution.md` — init/upgrade no longer copy kit to project, release from src/kit/.
- [x] T027 [P] Update `docs/memory/fab-workflow/setup.md` — init flow changes.
- [x] T028 [P] Update `docs/memory/fab-workflow/preflight.md` — VERSION source change.
- [x] T029 [P] Update `docs/memory/fab-workflow/migrations.md` — new migration, updated version tracking.
- [x] T030 [P] Update `docs/memory/fab-workflow/context-loading.md` — skill/template path references.
- [x] T031 [P] Update `docs/memory/fab-workflow/configuration.md` — remove kit.conf references.
- [x] T032 [P] Update `docs/memory/fab-workflow/execution-skills.md` and `planning-skills.md` — template access pattern.
- [x] T033 [P] Update `docs/specs/architecture.md` — directory structure, kit internals.
- [x] T034 [P] Update `fab/project/constitution.md` — reword Principle V (Portability), remove "cp -r fab/.kit/" language.
- [x] T035 [P] Update `fab/project/context.md` — update distribution description.
- [x] T036 [P] Update `CONTRIBUTING.md` and `README.md` — update `fab/.kit/` references.

---

## Execution Order

- T001 blocks T004-T006 (kitpath utility must exist before consumers)
- T003 blocks T007-T008 (repo constant must exist before removing kit.conf reads)
- T004-T014 (Phase 2) blocks T015 (source move depends on code changes being in place)
- T015 blocks T016-T024 (build/skill updates reference new paths)
- T007-T008 block T022 (migration references inline hooks)
- Phase 3 blocks Phase 4 (docs describe the new state)
