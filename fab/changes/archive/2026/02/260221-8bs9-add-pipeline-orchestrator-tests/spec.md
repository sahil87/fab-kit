# Spec: Add Pipeline Orchestrator Tests

**Change**: 260221-8bs9-add-pipeline-orchestrator-tests
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Non-Goals

- Full integration tests for `main()` loops in run.sh/dispatch.sh — deferred to a future change
- Testing tmux pane layout or Claude CLI session behavior — these require real infrastructure
- Testing `batch-fab-pipeline.sh` — simple entry point, low risk

## Test Infrastructure: Source Guards

### Requirement: Source guards for testability

`run.sh` and `dispatch.sh` SHALL have source guards at the bottom so they can be sourced by test files without executing `main()`.

#### Scenario: Source guard prevents main execution when sourced
- **GIVEN** `run.sh` has a source guard `if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then main; fi`
- **WHEN** a BATS test file sources `run.sh`
- **THEN** `main()` SHALL NOT execute
- **AND** all functions (validate_manifest, detect_cycles, is_dispatchable, etc.) SHALL be available in the test's shell

#### Scenario: Direct execution still works
- **GIVEN** `run.sh` has a source guard
- **WHEN** invoked directly via `bash run.sh manifest.yaml`
- **THEN** `main()` SHALL execute normally

## Test Infrastructure: Setup and Fixtures

### Requirement: Test file location and structure

The test file SHALL be located at `src/scripts/pipeline/test.bats`, following the established pattern where `src/` mirrors the `fab/.kit/scripts/` directory structure.

#### Scenario: Test file follows existing patterns
- **GIVEN** the test file at `src/scripts/pipeline/test.bats`
- **WHEN** the test suite runs
- **THEN** it SHALL use the same BATS + bats-assert + bats-support infrastructure as `src/lib/stageman/test.bats`

### Requirement: Test isolation via temp directories

Each test SHALL create an isolated `TEST_DIR` via `mktemp -d` in `setup()` and clean up via `rm -rf` in `teardown()`.

#### Scenario: Tests are isolated
- **GIVEN** `setup()` creates a fresh `TEST_DIR`
- **WHEN** a test creates files or modifies manifests
- **THEN** no artifacts SHALL persist after `teardown()`

### Requirement: External command mocking

External commands (`tmux`, `claude`, `gh`, `wt-create`, `changeman.sh`, `stageman.sh`, `calc-score.sh`) SHALL be stubbed via executables in `$TEST_DIR/bin/` prepended to `$PATH`.

#### Scenario: Stubbed commands intercept real calls
- **GIVEN** a stub `tmux` in `$TEST_DIR/bin/` that records calls
- **WHEN** a function under test calls `tmux`
- **THEN** the stub SHALL execute instead of the real `tmux`
- **AND** the stub's behavior SHALL be controllable per test

### Requirement: YAML manifest fixture helper

A helper function `make_manifest` SHOULD exist to create test manifests from inline YAML, reducing boilerplate.

#### Scenario: Manifest fixture creation
- **GIVEN** the `make_manifest` helper
- **WHEN** called with YAML content
- **THEN** it SHALL write to `$TEST_DIR/manifest.yaml` and echo the path

## run.sh: Manifest Validation

### Requirement: validate_manifest rejects invalid manifests

`validate_manifest()` SHALL return non-zero and print an error to stderr for manifests that violate structural requirements.

#### Scenario: Valid manifest passes
- **GIVEN** a manifest with `base: main` and changes with valid `id` and `depends_on` fields
- **WHEN** `validate_manifest` is called
- **THEN** it SHALL return 0

#### Scenario: Missing base field fails
- **GIVEN** a manifest without a `base` field
- **WHEN** `validate_manifest` is called
- **THEN** it SHALL return 1
- **AND** stderr SHALL contain "missing 'base' field"

#### Scenario: Empty changes array fails
- **GIVEN** a manifest with `changes: []`
- **WHEN** `validate_manifest` is called
- **THEN** it SHALL return 1
- **AND** stderr SHALL contain "no changes"

#### Scenario: Missing id field fails
- **GIVEN** a change entry without an `id` field
- **WHEN** `validate_manifest` is called
- **THEN** it SHALL return 1
- **AND** stderr SHALL contain "missing 'id' field"

#### Scenario: Missing depends_on field fails
- **GIVEN** a change entry without a `depends_on` field
- **WHEN** `validate_manifest` is called
- **THEN** it SHALL return 1
- **AND** stderr SHALL contain "missing 'depends_on' field"

#### Scenario: Dangling dependency reference fails
- **GIVEN** a change that depends on an ID not present in the manifest
- **WHEN** `validate_manifest` is called
- **THEN** it SHALL return 1
- **AND** stderr SHALL contain "not in the manifest"

#### Scenario: Multi-dependency marks invalid but passes
- **GIVEN** a change with `depends_on: [a, b]`
- **WHEN** `validate_manifest` is called
- **THEN** it SHALL return 0 (manifest is valid)
- **AND** the change's stage SHALL be set to `invalid` in the manifest

## run.sh: Cycle Detection

### Requirement: detect_cycles identifies circular dependencies

`detect_cycles()` SHALL detect and reject circular dependency chains.

#### Scenario: Linear chain has no cycles
- **GIVEN** a manifest with A → B → C (no cycles)
- **WHEN** `detect_cycles` is called
- **THEN** it SHALL return 0

#### Scenario: Direct cycle detected
- **GIVEN** a manifest with A depends_on B, B depends_on A
- **WHEN** `detect_cycles` is called
- **THEN** it SHALL return 1
- **AND** stderr SHALL contain "Circular dependency"

#### Scenario: Indirect cycle detected
- **GIVEN** a manifest with A → B → C → A
- **WHEN** `detect_cycles` is called
- **THEN** it SHALL return 1

#### Scenario: Independent nodes have no cycles
- **GIVEN** a manifest with multiple changes that have no dependencies
- **WHEN** `detect_cycles` is called
- **THEN** it SHALL return 0

## run.sh: Stage Predicates

### Requirement: is_terminal classifies stages correctly

`is_terminal()` SHALL return 0 for terminal stages (`done`, `failed`, `invalid`) and 1 for all others.

#### Scenario: Terminal stages
- **GIVEN** stage values `done`, `failed`, `invalid`
- **WHEN** `is_terminal` is called for each
- **THEN** it SHALL return 0 for all three

#### Scenario: Non-terminal stages
- **GIVEN** stage values `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate`, and empty string
- **WHEN** `is_terminal` is called for each
- **THEN** it SHALL return 1 for all

### Requirement: is_dispatchable checks self stage and dependency completion

`is_dispatchable()` SHALL return 0 only when the change is non-terminal and all its dependencies are `done`.

#### Scenario: Dispatchable — no deps, non-terminal
- **GIVEN** a change with no dependencies and no stage set
- **WHEN** `is_dispatchable` is called
- **THEN** it SHALL return 0

#### Scenario: Not dispatchable — self is terminal
- **GIVEN** a change with stage `done`
- **WHEN** `is_dispatchable` is called
- **THEN** it SHALL return 1

#### Scenario: Not dispatchable — dependency not done
- **GIVEN** change B depends on A, and A has no stage set
- **WHEN** `is_dispatchable` is called for B
- **THEN** it SHALL return 1

#### Scenario: Dispatchable — dependency done
- **GIVEN** change B depends on A, and A has stage `done`
- **WHEN** `is_dispatchable` is called for B
- **THEN** it SHALL return 0

## run.sh: Dispatch Ordering

### Requirement: find_next_dispatchable selects in list order

`find_next_dispatchable()` SHALL return the first dispatchable change in manifest list order.

#### Scenario: First dispatchable selected
- **GIVEN** changes [A (done), B (dispatchable), C (dispatchable)]
- **WHEN** `find_next_dispatchable` is called
- **THEN** it SHALL return B

#### Scenario: No dispatchable — all terminal
- **GIVEN** all changes have stage `done`
- **WHEN** `find_next_dispatchable` is called
- **THEN** it SHALL return 1

#### Scenario: No dispatchable — deps not met
- **GIVEN** change B depends on A, A has no stage
- **WHEN** `find_next_dispatchable` is called
- **THEN** it SHALL return 1 (if B is the only non-terminal change)

## run.sh: Branch Resolution

### Requirement: get_parent_branch resolves correctly

`get_parent_branch()` SHALL return the manifest's `base` for root nodes and the parent's branch name (with prefix) for dependent nodes.

#### Scenario: Root node returns base
- **GIVEN** a change with empty `depends_on` and manifest base `main`
- **WHEN** `get_parent_branch` is called
- **THEN** it SHALL return `main`

#### Scenario: Dependent node returns parent branch
- **GIVEN** change B depends on A, and branch prefix is empty
- **WHEN** `get_parent_branch` is called for B
- **THEN** it SHALL return A's resolved branch name

## dispatch.sh: Artifact Provisioning

### Requirement: provision_artifacts always syncs from source

`provision_artifacts()` SHALL copy artifacts from source to worktree on every call, even when the target directory already exists.

#### Scenario: First provision creates target
- **GIVEN** source dir exists with intake.md and spec.md, target dir does not exist
- **WHEN** `provision_artifacts` is called
- **THEN** target dir SHALL be created with all source files

#### Scenario: Re-provision updates stale target
- **GIVEN** target dir exists with only intake.md (stale), source has intake.md + spec.md
- **WHEN** `provision_artifacts` is called
- **THEN** target dir SHALL contain both intake.md and spec.md

#### Scenario: Missing source fails
- **GIVEN** source dir does not exist
- **WHEN** `provision_artifacts` is called
- **THEN** it SHALL return 1
- **AND** stderr SHALL contain "source change folder not found"

## dispatch.sh: Prerequisite Validation

### Requirement: validate_prerequisites checks required files

`validate_prerequisites()` SHALL verify intake.md and spec.md exist and confidence gate passes.

#### Scenario: Missing intake.md fails
- **GIVEN** a change dir without intake.md
- **WHEN** `validate_prerequisites` is called
- **THEN** it SHALL return 2
- **AND** the change stage SHALL be set to `invalid`

#### Scenario: Missing spec.md fails
- **GIVEN** a change dir with intake.md but without spec.md
- **WHEN** `validate_prerequisites` is called
- **THEN** it SHALL return 2
- **AND** the change stage SHALL be set to `invalid`

#### Scenario: Both files present with passing gate succeeds
- **GIVEN** a change dir with intake.md and spec.md, and calc-score returns gate: pass
- **WHEN** `validate_prerequisites` is called
- **THEN** it SHALL return 0

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Test location `src/scripts/pipeline/test.bats` | User explicitly specified; mirrors `src/lib/*` structure | S:95 R:95 A:95 D:95 |
| 2 | Certain | BATS with bats-assert/bats-support | All existing suites use this; infrastructure in place at `src/packages/tests/libs/bats/` | S:90 R:95 A:95 D:95 |
| 3 | Certain | Mock external commands via PATH stubs | Confirmed pattern from `src/lib/changeman/test.bats` lines 54-60 | S:95 R:90 A:95 D:90 |
| 4 | Certain | Add source guards to run.sh and dispatch.sh | Both scripts currently end with bare `main` — source guard is required for testability | S:95 R:90 A:90 D:95 |
| 5 | Confident | Defer poll_change tests to a follow-up | poll_change requires complex mocking (sleep loops, tmux, stageman polling); pure-logic functions provide more value per effort | S:80 R:85 A:75 D:75 |
| 6 | Confident | Single test.bats file for both run.sh and dispatch.sh | Both are small, tightly related scripts; separate files would add overhead without benefit | S:75 R:90 A:80 D:80 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
