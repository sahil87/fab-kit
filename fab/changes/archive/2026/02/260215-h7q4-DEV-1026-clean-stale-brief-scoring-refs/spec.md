# Spec: Clean Stale Brief Scoring Refs

**Change**: 260215-h7q4-DEV-1026-clean-stale-brief-scoring-refs
**Created**: 2026-02-15
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md` (no change — already correct)

## Non-Goals

- Modifying `calc-score.sh` behavior — the script is already correct (spec-only scanning)
- Adding intake/brief scanning support — the spec-only design is intentional

## Confidence Scoring: Documentation Accuracy

### Requirement: README SHALL describe spec-only scanning

`src/lib/calc-score/README.md` SHALL state that the change directory MUST contain `spec.md` with an `## Assumptions` table. The README SHALL NOT reference `brief.md` as an optional scanning source.

#### Scenario: Developer reads README for calc-score usage

- **GIVEN** a developer opens `src/lib/calc-score/README.md`
- **WHEN** they read the usage requirements
- **THEN** the description states the directory MUST contain `spec.md` with an `## Assumptions` table
- **AND** there is no mention of `brief.md` being scanned

### Requirement: SRAD spec SHALL describe spec-only confidence computation

The Confidence Lifecycle table in `docs/specs/srad.md` SHALL state that `calc-score.sh` scans `spec` only (not `intake + spec`). The script name SHALL be `calc-score.sh` (no underscore prefix).

#### Scenario: Contributor reads SRAD spec for confidence lifecycle

- **GIVEN** a contributor opens `docs/specs/srad.md`
- **WHEN** they read the Confidence Lifecycle table's Computation row
- **THEN** the Action column says `calc-score.sh` scans spec, writes to `.status.yaml`
- **AND** the script name does not have an underscore prefix

### Requirement: Test suite SHALL NOT assert brief.md scanning

`src/lib/calc-score/test.sh` SHALL NOT contain tests that assert combined `brief.md` + `spec.md` grade counting. The "Combined grades from brief and spec" test (lines 143–168) SHALL be removed.

#### Scenario: Test suite runs without brief.md assertions

- **GIVEN** a developer runs `src/lib/calc-score/test.sh`
- **WHEN** the test suite executes all test cases
- **THEN** no test creates a `brief.md` fixture
- **AND** no test asserts grade counting from `brief.md`
- **AND** all remaining tests pass

## Deprecated Requirements

### Combined brief+spec scanning documentation

**Reason**: The original design (change 260213-w8p3-extract-fab-score) planned brief+spec scanning but implementation was scoped to spec-only. Documentation and tests were never updated. The code is correct; only the stale references need removal.
**Migration**: N/A — the code already implements the correct spec-only behavior.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Script scans spec.md only | Confirmed from intake #1 — verified in calc-score.sh source and change-lifecycle.md:59 | S:95 R:95 A:95 D:95 |
| 2 | Certain | Scope limited to 3 files | Confirmed from intake #2 — README, srad spec, and test.sh are the only stale locations | S:90 R:90 A:90 D:90 |
| 3 | Confident | Remove combined-grades test rather than rewrite | Confirmed from intake #3 — test asserts intentionally excluded behavior; spec-only scanning is covered by existing tests | S:80 R:85 A:75 D:80 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
