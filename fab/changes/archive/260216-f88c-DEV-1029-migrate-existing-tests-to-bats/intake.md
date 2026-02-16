# Intake: Migrate Existing Bash Test Suites to bats-core

**Change**: 260216-f88c-DEV-1029-migrate-existing-tests-to-bats
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Follow-up to DEV-1028 (which introduces bats-core for new tests). Once bats is established as the bash testing standard, migrate the 4 existing hand-rolled test suites to use it — eliminating the dual-format runner path and unifying on a single framework.

## Why

1. **Consistency**: DEV-1028 introduces bats-core for new tests (changeman, sync-workspace). Having two test formats (legacy hand-rolled + bats) adds cognitive overhead and requires the justfile to support both runners.

2. **Maintenance**: The hand-rolled test harness (assert_equal, assert_exit_code, color output, manual counters) is duplicated across 4 test files. bats-core provides all of this out of the box with better reporting.

3. **Simplicity**: After migration, `just test-bash` only needs to invoke bats — no legacy runner path.

## What Changes

### Migrate 4 test suites from hand-rolled to bats-core

| Suite | Current file | Tests (approx) | Target |
|-------|-------------|-----------------|--------|
| preflight | `src/lib/preflight/test.sh` | 28 | `test.bats` |
| resolve-change | `src/lib/resolve-change/test.sh` | 20 | `test.bats` |
| stageman | `src/lib/stageman/test.sh` | 131 | `test.bats` |
| calc-score | `src/lib/calc-score/test.sh` | 30 | `test.bats` |

For each suite:
- Convert `assert_equal "expected" "$actual" "name"` → `@test "name" { ... }` with `run`/`$status`/`$output`
- Remove hand-rolled harness code (color constants, counter variables, summary printer)
- Use bats helpers (`bats-assert`, `bats-support`) where beneficial
- Preserve all existing test coverage — mechanical conversion, no test deletions
- Remove the legacy `test.sh` after conversion (keep `test-simple.sh` as quick smoke tests if still useful)

### Update justfile

- Remove the legacy test.sh runner path from `just test-bash`
- Simplify to bats-only invocation

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update testing section to reflect bats-only setup

## Impact

- **Test files**: 4 test.sh files replaced with test.bats equivalents
- **Build**: justfile simplified (single runner path)
- **Dev workflow**: Developers only need to know bats syntax for writing/reading tests

## Open Questions

- (None — scope is mechanical migration with clear before/after)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use bats-core `@test` blocks with `run`/`$status`/`$output` | Standard bats pattern, no alternatives | S:90 R:95 A:95 D:95 |
| 2 | Certain | Preserve all existing test coverage 1:1 | Explicit in description — mechanical migration | S:95 R:85 A:90 D:95 |
| 3 | Confident | Remove legacy test.sh after migration | No reason to keep dual formats; test-simple.sh may stay as smoke tests | S:75 R:85 A:80 D:75 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
