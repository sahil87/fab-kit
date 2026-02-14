# Tasks: Reorganize src/ and kit script internals

**Change**: 260214-q7f2-reorganize-src
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 [P] Create `fab/.kit/scripts/lib/` directory
- [x] T002 [P] Create `src/lib/` directory
- [x] T003 [P] Create `src/scripts/` directory

## Phase 2: Core — Move kit internal scripts to lib/

- [x] T004 Move `fab/.kit/scripts/_calc-score.sh` → `fab/.kit/scripts/lib/calc-score.sh` (git mv, drop prefix)
- [x] T005 Move `fab/.kit/scripts/_preflight.sh` → `fab/.kit/scripts/lib/preflight.sh` (git mv, drop prefix)
- [x] T006 Move `fab/.kit/scripts/_stageman.sh` → `fab/.kit/scripts/lib/stageman.sh` (git mv, drop prefix)
- [x] T007 Move `fab/.kit/scripts/_resolve-change.sh` → `fab/.kit/scripts/lib/resolve-change.sh` (git mv, drop prefix)
- [x] T008 Move `fab/.kit/scripts/_init_scaffold.sh` → `fab/.kit/scripts/lib/init-scaffold.sh` (git mv, drop prefix, hyphenate)
- [x] T009 Move `fab/.kit/scripts/fab-release.sh` → `src/scripts/fab-release.sh` (git mv)

## Phase 3: Core — Update inter-script references

- [x] T010 Update `fab/.kit/scripts/lib/preflight.sh` — source paths for `stageman.sh` and `resolve-change.sh` within `lib/`
- [x] T011 Update `fab/.kit/scripts/lib/calc-score.sh` — source path for `stageman.sh` within `lib/`
- [x] T012 Update `fab/.kit/scripts/fab-upgrade.sh` — call `lib/init-scaffold.sh` instead of `_init_scaffold.sh`
- [x] T013 [P] Update `fab/.kit/scripts/batch-archive-change.sh` — source `lib/resolve-change.sh`
- [x] T014 [P] Update `fab/.kit/scripts/batch-switch-change.sh` — source `lib/resolve-change.sh`

## Phase 4: Core — Move src/ test infra to lib/ and fix symlinks

- [x] T015 Move `src/calc-score/`, `src/preflight/`, `src/resolve-change/`, `src/stageman/` → `src/lib/` (git mv)
- [x] T016 Update symlinks in `src/lib/calc-score/` — remove old `_calc-score.sh`, create `calc-score.sh` → `../../../fab/.kit/scripts/lib/calc-score.sh`
- [x] T017 Update symlinks in `src/lib/preflight/` — remove old `_preflight.sh`, create `preflight.sh` → `../../../fab/.kit/scripts/lib/preflight.sh`
- [x] T018 Update symlinks in `src/lib/resolve-change/` — remove old `_resolve-change.sh`, create `resolve-change.sh` → `../../../fab/.kit/scripts/lib/resolve-change.sh`
- [x] T019 Update symlinks in `src/lib/stageman/` — remove old `_stageman.sh`, create `stageman.sh` → `../../../fab/.kit/scripts/lib/stageman.sh`

## Phase 5: Integration — Update skill references

- [x] T020 Update `fab/.kit/skills/_context.md` — all `_preflight.sh` → `lib/preflight.sh`, `_calc-score.sh` → `lib/calc-score.sh`
- [x] T021 Update `fab/.kit/skills/fab-continue.md` — `_calc-score.sh` → `lib/calc-score.sh`, `_stageman.sh` → `lib/stageman.sh`
- [x] T022 Update `fab/.kit/skills/fab-init.md` — `_init_scaffold.sh` → `lib/init-scaffold.sh`
- [x] T023 [P] Update `fab/.kit/skills/fab-status.md` — `_preflight.sh` → `lib/preflight.sh`
- [x] T024 [P] Update `fab/.kit/skills/fab-archive.md` — `_preflight.sh` → `lib/preflight.sh`
- [x] T025 [P] Update `fab/.kit/skills/fab-clarify.md` — `_calc-score.sh` → `lib/calc-score.sh`
- [x] T026 [P] Update `fab/.kit/skills/fab-ff.md` — `_stageman.sh` → `lib/stageman.sh`
- [x] T027 [P] Update `fab/.kit/skills/fab-fff.md` — `_stageman.sh` → `lib/stageman.sh`
- [x] T028 Update `fab/.kit/skills/_generation.md` — `_stageman.sh` → `lib/stageman.sh`

## Phase 6: Integration — Update build & dev config

- [x] T029 Update `justfile` — test glob from `src/*/test.sh` to `src/lib/*/test.sh`
- [x] T030 Update repo `.envrc` — add `PATH_add src/scripts`

## Phase 7: Verification

- [x] T031 Run `just test` — verify all 4 test suites still pass after all moves and reference updates

---

## Execution Order

- T001-T003 are independent setup (parallel)
- T004-T008 depend on T001 (lib/ dir exists)
- T009 depends on T003 (scripts/ dir exists)
- T010-T014 depend on T004-T008 (scripts moved before updating references)
- T015 depends on T002 (src/lib/ dir exists)
- T016-T019 depend on T004-T008 and T015 (both kit scripts moved AND src dirs moved)
- T020-T028 depend on T004-T008 (kit scripts renamed — skill files reference new names)
- T029-T030 depend on T015 (src dirs moved)
- T031 depends on all prior tasks
