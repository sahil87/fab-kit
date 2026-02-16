# Tasks: Rename Scaffold & Add Kit Script Tests

**Change**: 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Rename `fab/.kit/scripts/lib/init-scaffold.sh` → `fab/.kit/scripts/lib/sync-workspace.sh` and update the script's header comment and description line
- [x] T002 Rename `fab/.kit/worktree-init-common/2-rerun-init-scaffold.sh` → `2-rerun-sync-workspace.sh` and update the call inside to reference `sync-workspace.sh`
- [x] T003 Update all references to `init-scaffold.sh` across the codebase (kit scripts, skills, memory files, README.md) — see spec for full list. Verify zero matches for `init-scaffold` outside `fab/changes/archive/`
- [x] T004 [P] Create `src/lib/sync-workspace/` directory with symlink `sync-workspace.sh → ../../../fab/.kit/scripts/lib/sync-workspace.sh`
- [x] T005 [P] Create `src/lib/changeman/` directory with symlink `changeman.sh → ../../../fab/.kit/scripts/lib/changeman.sh`

## Phase 2: Core Implementation

- [x] T006 [P] Write `src/lib/sync-workspace/SPEC-sync-workspace.md` following the SPEC-stageman.md format (Sources of Truth, Usage, API/Behavior Reference, Requirements, Testing)
- [x] T007 [P] Write `src/lib/changeman/SPEC-changeman.md` following the SPEC-stageman.md format
- [x] T008 Write `src/lib/sync-workspace/test.bats` — bats test suite covering: directory creation, VERSION file logic, .envrc symlink, memory/specs index seeding, skill sync (3 platforms), model-tier agent generation, .gitignore management, idempotency
- [x] T009 Write `src/lib/changeman/test.bats` — bats test suite covering: `new` happy path, slug validation, change-id validation, random ID generation, collision detection, `--help`, error cases, `detect_created_by` fallback, stageman integration
- [x] T010 Restructure `justfile` with three recipes: `test-bash` (runs bats `.bats` + legacy `test.sh`), `test-rust` (no-op placeholder), `test` (both + per-suite pass/fail summary with overall verdict)

## Phase 3: Integration & Verification

- [x] T011 Run `just test` and verify all suites pass, summary output is correct

---

## Execution Order

- T001 blocks T002, T003, T004 (rename must happen before references are updated or symlinks created)
- T004 blocks T008 (symlink must exist before test file is written)
- T005 blocks T009 (symlink must exist before test file is written)
- T006, T007 are independent of each other and of T008, T009
- T008, T009, T010 block T011 (all test infrastructure must exist before final verification)
