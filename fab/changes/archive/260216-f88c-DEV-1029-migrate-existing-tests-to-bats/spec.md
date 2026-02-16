# Spec: Migrate Existing Bash Test Suites to bats-core

**Change**: 260216-f88c-DEV-1029-migrate-existing-tests-to-bats
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Adding new test coverage beyond what exists today — this is a mechanical 1:1 migration
- Modifying the scripts under test (`preflight.sh`, `resolve-change.sh`, `stageman.sh`, `calc-score.sh`)
- Changing the `test-simple.sh` smoke test files — they remain as-is

## Test Migration: Hand-Rolled to bats-core

### Requirement: Convert all 4 legacy test suites to bats format

Each legacy `test.sh` in `src/lib/{preflight,resolve-change,stageman,calc-score}/` SHALL be replaced with a `test.bats` file that preserves all existing test coverage using bats-core conventions.

The converted test files SHALL:
- Use `@test "name" { ... }` blocks instead of inline `assert_*` calls
- Use `run <command>` with `$status` and `$output` for capturing exit codes and output
- Use `setup()` and `teardown()` functions for per-test fixture creation/cleanup (temp dirs, stubs)
- Follow the patterns established by the existing bats suites (`changeman/test.bats`, `sync-workspace/test.bats`)

#### Scenario: Preflight test suite migration
- **GIVEN** `src/lib/preflight/test.sh` exists with ~28 assertions
- **WHEN** the migration is applied
- **THEN** `src/lib/preflight/test.bats` exists with equivalent `@test` blocks covering the same scenarios
- **AND** `src/lib/preflight/test.sh` is deleted
- **AND** `bats src/lib/preflight/test.bats` passes with all tests green

#### Scenario: Resolve-change test suite migration
- **GIVEN** `src/lib/resolve-change/test.sh` exists with ~20 assertions
- **WHEN** the migration is applied
- **THEN** `src/lib/resolve-change/test.bats` exists with equivalent `@test` blocks
- **AND** `src/lib/resolve-change/test.sh` is deleted
- **AND** `bats src/lib/resolve-change/test.bats` passes

#### Scenario: Stageman test suite migration
- **GIVEN** `src/lib/stageman/test.sh` exists with ~131 assertions
- **WHEN** the migration is applied
- **THEN** `src/lib/stageman/test.bats` exists with equivalent `@test` blocks
- **AND** `src/lib/stageman/test.sh` is deleted
- **AND** `bats src/lib/stageman/test.bats` passes

#### Scenario: Calc-score test suite migration
- **GIVEN** `src/lib/calc-score/test.sh` exists with ~30 assertions
- **WHEN** the migration is applied
- **THEN** `src/lib/calc-score/test.bats` exists with equivalent `@test` blocks
- **AND** `src/lib/calc-score/test.sh` is deleted
- **AND** `bats src/lib/calc-score/test.bats` passes

### Requirement: Preserve hand-rolled harness patterns as bats idioms

The migration SHALL replace the hand-rolled test harness code with bats equivalents:

| Legacy pattern | bats equivalent |
|----------------|-----------------|
| `assert_equal "expected" "$actual" "name"` | `@test "name" { run ...; [ "$output" = "expected" ] }` or `[[ "$output" == "expected" ]]` |
| `assert_exit_code 0 $? "name"` | `@test "name" { run ...; [ "$status" -eq 0 ] }` |
| `assert_contains "needle" "$haystack" "name"` | `@test "name" { run ...; [[ "$output" == *"needle"* ]] }` |
| `assert_success` / `assert_failure` | `[ "$status" -eq 0 ]` / `[ "$status" -ne 0 ]` |
| `assert_not_contains "needle" "$haystack" "name"` | `@test "name" { run ...; [[ "$output" != *"needle"* ]] }` |
| Color constants + counter variables + summary printer | Removed entirely — bats provides test reporting |
| Inline `set -uo pipefail` | Not needed — bats manages execution |
| `TESTS_RUN`, `TESTS_PASSED`, `TESTS_FAILED` counters | Removed — bats counts tests natively |

#### Scenario: No hand-rolled harness code remains
- **GIVEN** the migration is complete
- **WHEN** any converted `test.bats` file is inspected
- **THEN** it contains no `assert_equal`, `assert_exit_code`, `assert_contains`, `assert_success`, `assert_failure`, `assert_not_contains` function definitions
- **AND** it contains no `TESTS_RUN`, `TESTS_PASSED`, `TESTS_FAILED` variables
- **AND** it contains no color constant definitions (`RED`, `GREEN`, `NC`)

### Requirement: Follow established bats patterns from existing suites

The converted tests SHALL follow the conventions already set by `changeman/test.bats` and `sync-workspace/test.bats`:

- `SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"` for locating the script under test
- `setup()` creates a `TEST_DIR="$(mktemp -d)"` with required fixture structure
- `teardown()` calls `rm -rf "$TEST_DIR"` for cleanup
- Scripts under test are resolved via `readlink -f` or symlink path
- Stub scripts (e.g., stub `stageman.sh`) are created in `setup()` when the script under test calls external dependencies

#### Scenario: Consistent setup/teardown pattern
- **GIVEN** any converted `test.bats` file
- **WHEN** `bats` runs it
- **THEN** each test starts with a fresh temp directory
- **AND** no test artifacts remain after teardown

### Requirement: Handle resolve-change sourcing pattern

`resolve-change.sh` is a sourced library (not a CLI), so its tests cannot use `run resolve-change.sh`. The bats tests for resolve-change SHALL source the library in `setup()` and call `resolve_change` directly, checking `$RESOLVED_CHANGE_NAME` and exit behavior via subshell `run` patterns.

#### Scenario: Sourced library testing in bats
- **GIVEN** `resolve-change.sh` is a source-only library
- **WHEN** the bats test sources it in setup
- **THEN** tests can call `resolve_change "$fab_root" "$override"` and assert on `$RESOLVED_CHANGE_NAME`
- **AND** exit-code tests use `run bash -c 'source ...; resolve_change ...'` subshell pattern

## Justfile: Remove Legacy Runner Path

### Requirement: Simplify test-bash to bats-only

The `test-bash` recipe in `justfile` SHALL be simplified to invoke only bats. The legacy `test.sh` runner loop SHALL be removed.

#### Scenario: Justfile runs bats only
- **GIVEN** all 4 legacy suites have been migrated to bats
- **WHEN** `just test-bash` is run
- **THEN** only the bats runner loop (`src/lib/*/test.bats`) executes
- **AND** the legacy runner loop (`src/lib/*/test.sh`) is absent from the justfile
- **AND** all 6 suites (changeman, sync-workspace, preflight, resolve-change, stageman, calc-score) are discovered and run

## Deprecated Requirements

### Legacy test.sh runner path in justfile
**Reason**: All suites migrated to bats; dual-runner path no longer needed.
**Migration**: `just test-bash` uses bats-only loop.

### Hand-rolled test harness functions
**Reason**: bats-core provides test organization, assertions, and reporting out of the box.
**Migration**: Replaced by `@test` blocks with `run`/`$status`/`$output`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use bats-core `@test` blocks with `run`/`$status`/`$output` | Standard bats pattern, confirmed by existing changeman/sync-workspace suites | S:95 R:95 A:95 D:95 |
| 2 | Certain | Preserve all existing test coverage 1:1 | Explicit in intake — mechanical migration, no deletions | S:95 R:85 A:90 D:95 |
| 3 | Certain | Follow changeman/sync-workspace bats patterns for setup/teardown | Two existing bats suites establish the project convention | S:90 R:95 A:95 D:95 |
| 4 | Confident | Delete legacy test.sh after migration, keep test-simple.sh | Confirmed from intake #3 — no reason to keep dual formats; smoke tests may still be useful | S:80 R:85 A:80 D:75 |
| 5 | Confident | Use subshell `run bash -c 'source ...; ...'` for resolve-change tests | resolve-change.sh is source-only; subshell pattern is the standard bats approach for sourced libs | S:70 R:90 A:85 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
