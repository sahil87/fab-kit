# Spec: Add .gitkeep to fab/changes/archive/

**Change**: 260216-pr1u-DEV-1017-add-archive-gitkeep
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## fab-sync: Archive Directory Bootstrap

### Requirement: Archive directory creation

`fab-sync.sh` SHALL create `fab/changes/archive/` during the directory creation phase (Section 1). The archive directory MUST be included in the same `for dir in ...` loop that creates `fab/changes/`, `docs/memory/`, and `docs/specs/`.

#### Scenario: Fresh project with no archive directory

- **GIVEN** a workspace where `fab/changes/archive/` does not exist
- **WHEN** `fab-sync.sh` runs
- **THEN** `fab/changes/archive/` is created
- **AND** the script prints `Created: fab/changes/archive`

#### Scenario: Archive directory already exists

- **GIVEN** a workspace where `fab/changes/archive/` already exists
- **WHEN** `fab-sync.sh` runs
- **THEN** the directory is preserved (no error, no recreation message)

### Requirement: Archive .gitkeep file

`fab-sync.sh` SHALL create `fab/changes/archive/.gitkeep` if it does not exist, using the same conditional touch pattern as `fab/changes/.gitkeep` (lines 81-83 of the current implementation).

#### Scenario: Fresh project with no archive .gitkeep

- **GIVEN** `fab/changes/archive/` exists but contains no `.gitkeep`
- **WHEN** `fab-sync.sh` runs
- **THEN** `fab/changes/archive/.gitkeep` is created

#### Scenario: Archive .gitkeep already exists

- **GIVEN** `fab/changes/archive/.gitkeep` already exists
- **WHEN** `fab-sync.sh` runs
- **THEN** the file is preserved (idempotent)

## fab-sync SPEC: Documentation update

### Requirement: SPEC directory creation section

`src/lib/fab-sync/SPEC-fab-sync.md` Section "1. Directory Creation" SHALL mention `fab/changes/archive/` and its `.gitkeep` alongside the existing `fab/changes/` and `fab/changes/.gitkeep` references.

#### Scenario: Developer reads SPEC

- **GIVEN** a developer reads `SPEC-fab-sync.md`
- **WHEN** they look at the "1. Directory Creation" section
- **THEN** they see `fab/changes/archive/` listed as a created directory
- **AND** they see `fab/changes/archive/.gitkeep` listed as a created file

## fab-sync tests: Archive coverage

### Requirement: Bats test for archive .gitkeep

`src/lib/fab-sync/test.bats` SHALL include a test case verifying `fab/changes/archive/.gitkeep` is created, following the pattern of the existing `"creates fab/changes/.gitkeep"` test (lines 118-122).

#### Scenario: Test suite validates archive bootstrap

- **GIVEN** the bats test suite runs against a clean temp workspace
- **WHEN** `fab-sync.sh` executes
- **THEN** the test asserts `fab/changes/archive/.gitkeep` exists as a regular file

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add archive dir to existing `for` loop | Confirmed from intake #1 — mirrors existing directory creation pattern at line 74 | S:90 R:95 A:95 D:95 |
| 2 | Certain | Use same conditional `.gitkeep` pattern | Confirmed from intake #2 — identical to `fab/changes/.gitkeep` block at lines 81-83 | S:90 R:95 A:95 D:95 |
| 3 | Certain | Update SPEC and add bats test | Confirmed from intake #3 — existing SPEC and test file cover current `.gitkeep`; extend both | S:85 R:95 A:90 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
