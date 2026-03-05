# Quality Checklist: Migrate Existing Bash Test Suites to bats-core

**Change**: 260216-f88c-DEV-1029-migrate-existing-tests-to-bats
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 All 4 legacy suites converted: `test.bats` exists in preflight, resolve-change, stageman, calc-score
- [x] CHK-002 Justfile simplified: `test-bash` recipe contains only bats runner loop

## Behavioral Correctness
- [x] CHK-003 Preflight tests pass: `bats src/lib/preflight/test.bats` exits 0 with all tests green (30/30)
- [x] CHK-004 Resolve-change tests pass: `bats src/lib/resolve-change/test.bats` exits 0 (13/13)
- [x] CHK-005 Stageman tests pass: `bats src/lib/stageman/test.bats` exits 0 (75/75)
- [x] CHK-006 Calc-score tests pass: `bats src/lib/calc-score/test.bats` exits 0 (34/34)

## Removal Verification
- [x] CHK-007 Legacy test.sh deleted: no `test.sh` in {preflight,resolve-change,stageman,calc-score}
- [x] CHK-008 Legacy runner removed: justfile contains no `test.sh` runner loop
- [x] CHK-009 No hand-rolled harness: no `assert_equal`/`assert_exit_code`/`assert_contains` function defs, no `TESTS_RUN`/`TESTS_PASSED`/`TESTS_FAILED` vars, no color constants in any `test.bats`

## Scenario Coverage
- [x] CHK-010 Preflight coverage preserved: 30 @test blocks >= ~28 legacy assertions
- [x] CHK-011 Resolve-change coverage preserved: 13 @test blocks cover all legacy scenarios (legacy ~20 counted multi-assertion sequences)
- [x] CHK-012 Stageman coverage preserved: 75 @test blocks cover all legacy scenarios (legacy ~131 counted individual assertions, many grouped per scenario)
- [x] CHK-013 Calc-score coverage preserved: 34 @test blocks >= ~30 legacy assertions. Also fixed 2 legacy bugs: wrong carry-forward assumptions and incorrect fuzzy column order
- [x] CHK-014 Full suite runs: `just test-bash` discovers and runs all 6 bats suites (152 total tests, 6/6 pass)

## Code Quality
- [x] CHK-015 Pattern consistency: all 4 new bats files use BATS_TEST_FILENAME, setup/teardown, mktemp -d, readlink -f conventions
- [x] CHK-016 No unnecessary duplication: each suite has focused helpers (make_status, make_write_fixture, create_change, etc.) appropriate to its scope

## Documentation Accuracy
- [x] CHK-017 Kit-architecture memory: testing section reflects bats-only setup (updated stageman test reference: test.sh→test.bats, 131→75)

## Cross References
- [x] CHK-018 Test counts in memory match actual test files (stageman: 75, calc-score: 34, preflight: 30, resolve-change: 13)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
