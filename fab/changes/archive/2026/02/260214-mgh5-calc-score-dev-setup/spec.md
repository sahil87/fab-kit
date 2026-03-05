# Spec: Add dev folder and tests for _calc-score.sh

**Change**: 260214-mgh5-calc-score-dev-setup
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/kit-architecture.md`

## Dev Folder: Directory Structure

### Requirement: calc-score dev folder layout

`src/calc-score/` SHALL contain the same four files as the existing `src/stageman/`, `src/resolve-change/`, and `src/preflight/` folders:

```
src/calc-score/
├── _calc-score.sh     # symlink → ../../fab/.kit/scripts/_calc-score.sh
├── README.md          # API docs, usage, testing, changelog
├── test-simple.sh     # quick smoke test (executable)
└── test.sh            # comprehensive test suite (executable)
```

#### Scenario: Directory exists with all files

- **GIVEN** the change has been applied
- **WHEN** listing `src/calc-score/`
- **THEN** exactly four files are present: `_calc-score.sh`, `README.md`, `test-simple.sh`, `test.sh`
- **AND** `_calc-score.sh` is a symlink pointing to `../../fab/.kit/scripts/_calc-score.sh`
- **AND** `test-simple.sh` and `test.sh` are executable

### Requirement: Symlink resolves correctly

The `_calc-score.sh` symlink MUST resolve to the actual script at `fab/.kit/scripts/_calc-score.sh` when invoked from `src/calc-score/`.

#### Scenario: Symlink target is valid

- **GIVEN** the symlink `src/calc-score/_calc-score.sh` exists
- **WHEN** `readlink src/calc-score/_calc-score.sh` is run
- **THEN** the output is `../../fab/.kit/scripts/_calc-score.sh`
- **AND** the resolved file exists and is executable

## Dev Folder: README

### Requirement: README follows established pattern

`src/calc-score/README.md` SHALL follow the same structure as `src/stageman/README.md` and `src/resolve-change/README.md`:

1. **Title and description** — what the script does
2. **Sources of Truth** — implementation path, dev symlink path
3. **Usage** — how to invoke (command-line interface)
4. **API Reference** — arguments, output format, exit codes, side effects
5. **Requirements** — Bash version, dependencies
6. **Testing** — commands to run smoke test and comprehensive suite
7. **Changelog** — versioned entries

#### Scenario: README documents the script API

- **GIVEN** `src/calc-score/README.md` exists
- **WHEN** a developer reads it
- **THEN** the README documents: usage (`_calc-score.sh <change-dir>`), output format (YAML with confidence block + delta), side effects (writes to `.status.yaml`), exit codes (0 success, 1 error), and error conditions

## Dev Folder: Smoke Test

### Requirement: test-simple.sh provides quick validation

`test-simple.sh` SHALL be a quick smoke test that validates the script loads and runs against a minimal fixture. It MUST complete in under 2 seconds and exit 0 on success, non-zero on failure.

#### Scenario: Smoke test passes with valid input

- **GIVEN** a temp directory with a minimal `.status.yaml` and a `spec.md` containing an `## Assumptions` table
- **WHEN** `src/calc-score/test-simple.sh` is run
- **THEN** exit code is 0
- **AND** output includes pass indicators (e.g., checkmarks)

#### Scenario: Smoke test is self-contained

- **GIVEN** no prior test state
- **WHEN** `test-simple.sh` runs
- **THEN** it creates its own temp fixtures and cleans them up on exit

## Dev Folder: Comprehensive Test Suite

### Requirement: test.sh covers core functionality

`test.sh` SHALL exercise the following areas of `_calc-score.sh`:

1. **Grade counting** — parsing `## Assumptions` tables from `brief.md` and `spec.md`
2. **Score formula** — correct computation of `max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)`
3. **Carry-forward** — implicit Certain counts preserved from previous `.status.yaml`
4. **Status update** — confidence block written correctly to `.status.yaml`
5. **Delta computation** — `+X.X` / `-X.X` format in stdout output
6. **Error cases** — missing change-dir, missing spec.md

#### Scenario: Grade counting from Assumptions tables

- **GIVEN** a `spec.md` with an `## Assumptions` table containing 2 Confident and 1 Tentative rows
- **WHEN** `_calc-score.sh` is invoked on the change directory
- **THEN** stdout YAML shows `confident: 2` and `tentative: 1`

#### Scenario: Score formula computation

- **GIVEN** 2 Confident and 1 Tentative assumptions
- **WHEN** `_calc-score.sh` computes the score
- **THEN** score is `3.4` (5.0 - 0.3×2 - 1.0×1)

#### Scenario: Carry-forward of implicit Certain counts

- **GIVEN** `.status.yaml` has `certain: 5` and `spec.md` has 0 Certain grades in the table
- **WHEN** `_calc-score.sh` runs
- **THEN** stdout YAML shows `certain: 5` (all 5 carried forward as implicit)

#### Scenario: Status.yaml confidence block update

- **GIVEN** a `.status.yaml` with an existing confidence block
- **WHEN** `_calc-score.sh` completes
- **THEN** the confidence block in `.status.yaml` is replaced with the new values
- **AND** other `.status.yaml` fields (name, progress, checklist) are preserved

#### Scenario: Delta computation

- **GIVEN** `.status.yaml` has `score: 5.0`
- **WHEN** `_calc-score.sh` computes score `3.4`
- **THEN** stdout includes `delta: -1.6`

#### Scenario: Combined grades from brief and spec

- **GIVEN** `brief.md` has 1 Tentative assumption and `spec.md` has 1 Confident assumption
- **WHEN** `_calc-score.sh` runs
- **THEN** both are counted: `confident: 1`, `tentative: 1`, `score: 3.7`

#### Scenario: Missing change directory

- **GIVEN** a non-existent directory path
- **WHEN** `_calc-score.sh <bad-path>` is run
- **THEN** exit code is 1
- **AND** stderr contains `"Change directory not found"`

#### Scenario: Missing spec.md

- **GIVEN** a valid change directory with no `spec.md`
- **WHEN** `_calc-score.sh` is run
- **THEN** exit code is 1
- **AND** stderr contains `"spec.md required for scoring"`

#### Scenario: No arguments

- **GIVEN** no arguments
- **WHEN** `_calc-score.sh` is run
- **THEN** exit code is 1
- **AND** stderr contains usage message

### Requirement: test.sh follows project test conventions

Each test case SHALL print a pass/fail indicator and the test SHALL exit non-zero on first failure. Tests MUST create their own temp fixtures and clean up via `trap`.

#### Scenario: Test cleanup

- **GIVEN** `test.sh` creates temp directories for fixtures
- **WHEN** the test suite finishes (success or failure)
- **THEN** all temp directories are removed

## Memory Hydration: Kit Architecture

### Requirement: Update kit-architecture memory

`fab/memory/fab-workflow/kit-architecture.md` SHALL be updated to include `src/calc-score/` in any inventory or listing of internal script dev folders.

#### Scenario: kit-architecture reflects calc-score dev setup

- **GIVEN** the hydrate stage has completed
- **WHEN** reading `fab/memory/fab-workflow/kit-architecture.md`
- **THEN** `_calc-score.sh` references include its `src/calc-score/` dev folder existence
