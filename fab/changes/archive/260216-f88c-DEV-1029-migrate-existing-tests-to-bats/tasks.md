# Tasks: Migrate Existing Bash Test Suites to bats-core

**Change**: 260216-f88c-DEV-1029-migrate-existing-tests-to-bats
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Migrations

- [x] T001 [P] Migrate `src/lib/preflight/test.sh` → `src/lib/preflight/test.bats` — convert ~28 assertions to `@test` blocks with `setup()`/`teardown()`, following changeman/sync-workspace patterns
- [x] T002 [P] Migrate `src/lib/resolve-change/test.sh` → `src/lib/resolve-change/test.bats` — convert ~20 assertions, use subshell `run bash -c 'source ...; ...'` pattern for the sourced library
- [x] T003 [P] Migrate `src/lib/stageman/test.sh` → `src/lib/stageman/test.bats` — convert ~131 assertions to `@test` blocks (largest suite)
- [x] T004 [P] Migrate `src/lib/calc-score/test.sh` → `src/lib/calc-score/test.bats` — convert ~30 assertions to `@test` blocks

## Phase 2: Cleanup

- [x] T005 Delete all 4 legacy `test.sh` files: `src/lib/{preflight,resolve-change,stageman,calc-score}/test.sh`
- [x] T006 Simplify `justfile` `test-bash` recipe — remove the legacy `test.sh` runner loop, keep bats-only loop

---

## Execution Order

- T001–T004 are independent and parallelizable
- T005 depends on T001–T004 (all migrations must pass before deleting originals)
- T006 depends on T005 (delete before simplifying runner)
