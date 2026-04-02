# Tasks: Brew Install System Shim

**Change**: 260401-46hw-brew-install-system-shim
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create shim Go module at `src/go/shim/` with `go.mod`, `cmd/main.go` entry point, and basic Cobra root command (`fab --version`, `fab --help`)
- [x] T002 [P] Add `fab_version` field to `fab/project/config.yaml` (set to current VERSION value)
- [x] T003 [P] Create `Formula/fab-kit.rb` Homebrew formula skeleton (installs `fab`, `wt`, `idea` from release artifacts)

## Phase 2: Core Implementation

- [x] T004 Implement shim config resolution in `src/go/shim/internal/config.go` — walk up from CWD to find `fab/project/config.yaml`, read `fab_version`, error handling for missing config or missing field
- [x] T005 Implement shim cache management in `src/go/shim/internal/cache.go` — check `~/.fab-kit/versions/{version}/fab-go` existence, return cache path
- [x] T006 Implement shim download logic in `src/go/shim/internal/download.go` — download platform-specific `kit-{os}-{arch}.tar.gz` from GitHub releases, extract `fab-go` to `~/.fab-kit/versions/{version}/fab-go` and `.kit/` content to `~/.fab-kit/versions/{version}/kit/`
- [x] T007 Implement shim dispatch in `src/go/shim/cmd/main.go` — resolve version, ensure cached, exec `fab-go` with full argument passthrough
- [x] T008 Implement `fab init` subcommand in `src/go/shim/internal/init.go` — resolve latest release, ensure cached, copy `kit/` → repo's `fab/.kit/`, create/update `config.yaml` with `fab_version`, run `fab-sync.sh`
- [x] T009 Implement `fab upgrade` subcommand in `src/go/shim/internal/upgrade.go` — resolve target version (latest or explicit), ensure cached, atomic swap of `fab/.kit/`, update `fab_version` in `config.yaml`, run `fab-sync.sh`, display version change and migration reminder

## Phase 3: Integration & Edge Cases

- [x] T010 Bulk update all skill files: replace `fab/.kit/bin/fab` with `fab` in `fab/.kit/skills/_preamble.md`, `_cli-fab.md`, `_generation.md`, `fab-new.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `fab-clarify.md`, `fab-switch.md`, `fab-status.md`, `fab-archive.md`, `fab-setup.md`, `fab-operator7.md`, `fab-discuss.md`, `fab-help.md`, `fab-proceed.md`, `git-branch.md`, `git-pr.md`, `git-pr-review.md` <!-- clarified: added fab-discuss.md, fab-help.md, fab-proceed.md — verified via grep that 19 skill files reference fab/.kit/bin/fab, not 16 -->
- [x] T011 [P] Update hook scripts: replace `fab/.kit/bin/fab` or `$kit_dir/bin/fab` with `fab` in all `fab/.kit/hooks/on-*.sh` files
- [x] T012 [P] Update sync pipeline: remove `fab/.kit/sync/4-get-fab-binary.sh`, update `fab/.kit/sync/5-sync-hooks.sh` to call `fab hook sync`
- [x] T013 [P] Update `.envrc` scaffold: remove `PATH_add fab/.kit/bin` from `fab/.kit/scaffold/fragment-.envrc`
- [x] T014 [P] Update `fab-doctor.sh`: add check for `fab` system binary on PATH (check 7)
- [x] T015 [P] Remove `fab/.kit/scripts/fab-upgrade.sh`
- [x] T016 Remove binaries from `fab/.kit/bin/`: delete `fab` (shell dispatcher), `fab-go`, `wt`, `idea` — keep `.gitkeep`
- [x] T017 Amend Constitution Principle V in `fab/project/constitution.md` — update portability text to require system shim
- [x] T018 Write unit tests for shim: `src/go/shim/cmd/main_test.go`, `src/go/shim/internal/config_test.go`, `src/go/shim/internal/cache_test.go`, `src/go/shim/internal/download_test.go`, `src/go/shim/internal/init_test.go`
- [x] T019 Create migration file `fab/.kit/migrations/{FROM}-to-{TO}.md` — prerequisite gate (brew install), add `fab_version`, clean `.envrc`, clean `fab/.kit/bin/`
- [x] T020 Update `justfile` — add shim build recipes (`build-shim`, `build-shim-target`), update `build` to include shim, update `build-target`/`build-all` for shim cross-compilation, update `package-kit` to exclude shim from kit archives

## Phase 4: Polish

- [x] T021 Update `README.md` — new install instructions (`brew install fab-kit` → `fab init`), remove curl bootstrap one-liner, update Quick Start

---

## Execution Order

- T004 blocks T007 (dispatch needs config resolution)
- T005 blocks T007 (dispatch needs cache check)
- T006 blocks T007 (dispatch needs download on cache miss)
- T007 blocks T008, T009 (init/upgrade build on dispatch infrastructure)
- T010-T016 are independent of T001-T009 (file changes, no code dependency)
- T018 should run after T004-T009 (tests need implementation)
- T019 is independent (migration file is markdown)
