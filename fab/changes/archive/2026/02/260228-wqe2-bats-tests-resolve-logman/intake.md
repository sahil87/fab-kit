# Intake: Add BATS Tests for resolve.sh and logman.sh

**Change**: 260228-wqe2-bats-tests-resolve-logman
**Created**: 2026-02-28
**Status**: Draft

## Origin

> Add BATS test suites for resolve.sh and logman.sh — src/lib/resolve/ and src/lib/logman/ currently only have test-simple.sh but no test.bats, unlike all other test directories.

Identified during the gap check of `260228-9fg2-refactor-kit-scripts`. All other test directories (`statusman`, `changeman`, `calc-score`, `preflight`) have both `test-simple.sh` (quick smoke test) and `test.bats` (comprehensive BATS suite). The two new directories created during the refactor only got `test-simple.sh`.

## Why

Convention consistency. Every kit script test directory follows the same two-file pattern: `test-simple.sh` for quick smoke tests and `test.bats` for comprehensive BATS-format test suites. `src/lib/resolve/` and `src/lib/logman/` break this convention, making them look incomplete. BATS tests also provide better structured output, setup/teardown isolation, and integration with CI tooling.

## What Changes

### 1. Create `src/lib/resolve/test.bats`

BATS test suite for `resolve.sh` covering:
- All four output modes (`--id`, `--folder`, `--dir`, `--status`)
- Input forms: 4-char ID, substring, full folder name, no argument (fab/current)
- Single-change guess fallback
- Error cases: no match, multiple matches, no changes directory, no fab/current with multiple changes
- Edge cases: archive directory excluded from resolution

Use the existing `src/lib/resolve/test-simple.sh` as reference for test fixture setup patterns.

### 2. Create `src/lib/logman/test.bats`

BATS test suite for `logman.sh` covering:
- All three subcommands: `command`, `confidence`, `review`
- JSON structure validation (required fields: `ts`, `event`, plus subcommand-specific fields)
- Append-only behavior (existing lines preserved)
- Optional fields: `args` for command, `rework` for review
- `.history.jsonl` creation when file doesn't exist
- Change resolution via resolve.sh integration

Use the existing `src/lib/logman/test-simple.sh` as reference.

## Affected Memory

None — test-only change, no behavioral or documentation impact.

## Impact

- `src/lib/resolve/test.bats` — new file
- `src/lib/logman/test.bats` — new file

## Open Questions

None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Follow existing BATS test conventions from sibling directories | Codebase has 4 existing test.bats files with consistent patterns — setup/teardown, fixture creation, assertion style | S:90 R:90 A:95 D:95 |
| 2 | Certain | Test scope matches test-simple.sh coverage plus edge cases | test-simple.sh covers the happy paths; BATS adds error cases, edge cases, and structured assertions | S:85 R:90 A:90 D:95 |

2 assumptions (2 certain, 0 confident, 0 tentative, 0 unresolved).
