# Spec: Batch Pipeline — Single Change Support & Default Base Branch

**Change**: 260223-xiuk-batch-pipeline-single-change-and-base-branch
**Created**: 2026-02-23
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Pipeline: Series Script Single-Change Support

### Requirement: Single-Change Invocation

`batch-pipeline-series.sh` SHALL accept a single change argument. The minimum required argument count MUST be 1 (not 2). A single-change invocation generates a manifest with one entry that has `depends_on: []` and delegates to `pipeline/run.sh` identically to a multi-change invocation.

#### Scenario: Single change via series script
- **GIVEN** a valid change folder exists under `fab/changes/`
- **WHEN** the user runs `batch-pipeline-series <change-id>`
- **THEN** a temporary manifest is generated with `base` set to the current branch, one entry with `depends_on: []`, and the orchestrator is invoked

#### Scenario: No arguments
- **GIVEN** the user runs `batch-pipeline-series` with no arguments
- **WHEN** argument parsing completes
- **THEN** the script exits with code 1 and prints the usage message

### Requirement: Updated Usage Text

The usage text and examples SHALL reflect single-change support:
- The arguments line SHALL read `<change> [<change>...]`
- The description SHALL say "Run changes in a simple sequential chain" (no minimum count stated)
- Examples SHALL include a single-change invocation

#### Scenario: Help output shows single-change usage
- **GIVEN** the user runs `batch-pipeline-series --help`
- **WHEN** usage text is displayed
- **THEN** the arguments line shows `<change> [<change>...]`
- **AND** at least one example demonstrates single-change invocation

## Pipeline: Default Base Branch Resolution

### Requirement: Optional Base Field in Manifests

`run.sh`'s `validate_manifest()` function SHALL treat the `base` field as optional. When `base` is missing, empty, or `null`, it MUST resolve to the current git branch via `git branch --show-current`, with `main` as the last-resort fallback if the git command fails (e.g., detached HEAD, not in a git repo).

#### Scenario: Manifest with explicit base passes validation
- **GIVEN** a manifest with `base: "develop"`
- **WHEN** `validate_manifest()` runs
- **THEN** validation passes and the `base` field is unchanged

#### Scenario: Manifest with missing base resolves to current branch
- **GIVEN** a manifest without a `base` field
- **AND** the current git branch is `feat/my-feature`
- **WHEN** `validate_manifest()` runs
- **THEN** validation passes
- **AND** the manifest's `base` field is written as `feat/my-feature`

#### Scenario: Manifest with empty base resolves to current branch
- **GIVEN** a manifest with `base: ""`
- **AND** the current git branch is `main`
- **WHEN** `validate_manifest()` runs
- **THEN** validation passes
- **AND** the manifest's `base` field is written as `main`

#### Scenario: Detached HEAD falls back to main
- **GIVEN** a manifest without a `base` field
- **AND** git is in detached HEAD state (`git branch --show-current` returns empty)
- **WHEN** `validate_manifest()` runs
- **THEN** validation passes
- **AND** the manifest's `base` field is written as `main`

### Requirement: Write-Back for Downstream Consistency

When `validate_manifest()` resolves a default base branch, it MUST write the resolved value back to the manifest file via `yq -i`. This ensures `get_parent_branch()` reads a consistent value without needing its own fallback logic.

#### Scenario: get_parent_branch reads resolved base
- **GIVEN** a manifest was validated with a missing `base` field
- **AND** the resolved base was written back as `feat/my-feature`
- **WHEN** `get_parent_branch()` is called for a root node
- **THEN** it returns `feat/my-feature`

## Pipeline: Example Manifest Documentation

### Requirement: Document Base as Optional

`fab/pipelines/example.yaml` SHALL document the `base` field as optional, with a comment explaining the default behavior (current branch, `main` fallback).

#### Scenario: Example manifest shows optional base
- **GIVEN** a user reads `fab/pipelines/example.yaml`
- **WHEN** they look at the `base` field documentation
- **THEN** they see that `base` is optional and defaults to the current branch

## Pipeline: Test Coverage

### Requirement: Test Default Base Resolution

The BATS test suite SHALL include tests for the new `validate_manifest()` behavior:

1. Missing `base` field — validation passes and resolved base is written back
2. Empty `base` field — validation passes and resolved base is written back

The existing test "validate_manifest: missing base field fails" SHALL be replaced with a test that verifies the new default behavior.

#### Scenario: Test validates missing base resolves successfully
- **GIVEN** the BATS test creates a manifest without a `base` field
- **AND** a `git` stub that returns a known branch name
- **WHEN** `validate_manifest` is called
- **THEN** it returns exit code 0
- **AND** the manifest file contains the resolved `base` value

## Deprecated Requirements

### Requirement: Minimum Two Changes in Series Script
**Reason**: Replaced by single-change support. The `>= 2` guard was artificial — the downstream pipeline already handles single-entry manifests.
**Migration**: The guard is changed to `>= 1`.

### Requirement: Mandatory Base Field in Manifests
**Reason**: Replaced by optional base with current-branch default. Erroring on missing `base` forced unnecessary ceremony for hand-written manifests.
**Migration**: Missing/empty `base` resolves to current branch automatically.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop minimum from 2 to 1 in series script | Confirmed from intake #1 — user explicitly requested; downstream pipeline already supports single-entry manifests | S:95 R:95 A:95 D:95 |
| 2 | Certain | Default base to current branch, fallback to main | Confirmed from intake #2 — user explicitly requested; matches existing `batch-pipeline-series.sh` behavior on line 84-86 | S:90 R:90 A:90 D:90 |
| 3 | Confident | Write resolved base back to manifest via yq -i | Confirmed from intake #3 — avoids duplicating fallback logic in `get_parent_branch()`; single source of truth in manifest after resolution | S:70 R:85 A:80 D:75 |
| 4 | Certain | No changes to dispatch.sh or batch-fab-switch-change.sh | Confirmed from intake #4 — branch naming verified consistent during discussion | S:95 R:95 A:95 D:95 |
| 5 | Confident | Detached HEAD fallback to "main" | Confirmed from intake #5 — standard convention; `git branch --show-current` returns empty on detached HEAD | S:60 R:85 A:80 D:80 |
| 6 | Certain | Replace existing "missing base fails" test with "missing base resolves" test | Existing test asserts the old error behavior — must be updated to match new requirement | S:90 R:95 A:90 D:95 |
| 7 | Confident | Stub git in BATS tests for base resolution | Tests need deterministic branch output; git stub in `$TEST_DIR/bin/` follows established test pattern | S:75 R:90 A:85 D:80 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
