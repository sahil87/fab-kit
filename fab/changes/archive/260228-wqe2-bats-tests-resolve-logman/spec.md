# Spec: Add BATS Tests for resolve.sh and logman.sh

**Change**: 260228-wqe2-bats-tests-resolve-logman
**Created**: 2026-02-28
**Affected memory**: None — test-only change

## resolve.sh Test Suite

### Requirement: Isolated Fixture Structure

Each BATS test SHALL create an isolated temporary directory with the structure `$TEST_DIR/fab/.kit/scripts/lib/` and copy the real `resolve.sh` into it. The `fab/changes/` directory and `fab/current` file SHALL be created relative to `$TEST_DIR/fab/` for each test's needs. `teardown()` SHALL remove `$TEST_DIR`.

#### Scenario: Setup creates isolated environment
- **GIVEN** a BATS test begins execution
- **WHEN** `setup()` runs
- **THEN** a temporary directory exists with `resolve.sh` copied to the correct kit path
- **AND** the script resolves `FAB_ROOT` relative to its own location within `$TEST_DIR`

### Requirement: Output Mode Coverage

The test suite SHALL verify all four output modes (`--id`, `--folder`, `--dir`, `--status`) produce the correct format for a given change.

#### Scenario: --id extracts 4-char ID
- **GIVEN** a change folder `260228-a1b2-test-change` exists
- **WHEN** `resolve.sh --id a1b2` is invoked
- **THEN** stdout is exactly `a1b2`

#### Scenario: --folder returns full folder name
- **GIVEN** a change folder `260228-a1b2-test-change` exists
- **WHEN** `resolve.sh --folder a1b2` is invoked
- **THEN** stdout is exactly `260228-a1b2-test-change`

#### Scenario: --dir returns directory path with trailing slash
- **GIVEN** a change folder `260228-a1b2-test-change` exists
- **WHEN** `resolve.sh --dir a1b2` is invoked
- **THEN** stdout is `fab/changes/260228-a1b2-test-change/`

#### Scenario: --status returns .status.yaml path
- **GIVEN** a change folder `260228-a1b2-test-change` exists
- **WHEN** `resolve.sh --status a1b2` is invoked
- **THEN** stdout is `fab/changes/260228-a1b2-test-change/.status.yaml`

#### Scenario: Default mode is --id
- **GIVEN** a change folder `260228-a1b2-test-change` exists
- **WHEN** `resolve.sh a1b2` is invoked (no flag)
- **THEN** stdout is `a1b2`

### Requirement: Input Form Coverage

The test suite SHALL verify resolution from 4-char IDs, substring matches, full folder names, and `fab/current` fallback.

#### Scenario: Full folder name resolves via exact match
- **GIVEN** a change folder `260228-a1b2-test-change` exists
- **WHEN** `resolve.sh --folder 260228-a1b2-test-change` is invoked
- **THEN** stdout is `260228-a1b2-test-change`

#### Scenario: Substring match resolves uniquely
- **GIVEN** change folders `260228-a1b2-alpha` and `260228-c3d4-beta` exist
- **WHEN** `resolve.sh --folder alpha` is invoked
- **THEN** stdout is `260228-a1b2-alpha`

#### Scenario: Case-insensitive matching
- **GIVEN** a change folder `260228-a1b2-test-change` exists
- **WHEN** `resolve.sh --folder A1B2` is invoked
- **THEN** stdout is `260228-a1b2-test-change`

#### Scenario: No argument reads fab/current
- **GIVEN** `fab/current` contains `260228-a1b2-test-change` and that folder exists
- **WHEN** `resolve.sh --folder` is invoked (no change argument)
- **THEN** stdout is `260228-a1b2-test-change`

#### Scenario: fab/current with trailing whitespace
- **GIVEN** `fab/current` contains `260228-a1b2-test-change\n  `
- **WHEN** `resolve.sh --folder` is invoked
- **THEN** stdout is `260228-a1b2-test-change`

### Requirement: Single-Change Guess Fallback

When `fab/current` is missing or empty and exactly one change (with `.status.yaml`) exists, `resolve.sh` SHALL resolve to that single change and emit a diagnostic on stderr.

#### Scenario: Single change guessed when fab/current missing
- **GIVEN** `fab/current` does not exist and exactly one change folder with `.status.yaml` exists
- **WHEN** `resolve.sh` is invoked with no arguments
- **THEN** stdout is the single change's folder name
- **AND** stderr contains "resolved from single active change"

#### Scenario: Guess requires .status.yaml
- **GIVEN** `fab/current` does not exist and one change folder exists WITHOUT `.status.yaml`
- **WHEN** `resolve.sh` is invoked with no arguments
- **THEN** exit code is 1
- **AND** stderr contains "No active change"

### Requirement: Error Case Coverage

The test suite SHALL verify correct error behavior for no-match, multiple-match, missing directory, and missing `fab/current` scenarios.

#### Scenario: No match returns error
- **GIVEN** change folders exist
- **WHEN** `resolve.sh nonexistent` is invoked
- **THEN** exit code is 1
- **AND** stderr contains `No change matches "nonexistent"`

#### Scenario: Multiple matches returns error
- **GIVEN** folders `260228-a1b2-test-alpha` and `260228-c3d4-test-beta` exist
- **WHEN** `resolve.sh test` is invoked (matches both)
- **THEN** exit code is 1
- **AND** stderr contains "Multiple changes match"

#### Scenario: Missing fab/changes/ returns error
- **GIVEN** `fab/changes/` directory does not exist
- **WHEN** `resolve.sh something` is invoked
- **THEN** exit code is 1
- **AND** stderr contains "fab/changes/ not found"

#### Scenario: No fab/current with multiple changes returns error
- **GIVEN** `fab/current` does not exist and multiple change folders exist
- **WHEN** `resolve.sh` is invoked (no argument)
- **THEN** exit code is 1
- **AND** stderr contains "No active change"

### Requirement: Archive Exclusion

The archive directory SHALL be excluded from change resolution.

#### Scenario: archive directory excluded
- **GIVEN** only an `archive/` directory exists under `fab/changes/`
- **WHEN** `resolve.sh archive` is invoked
- **THEN** exit code is 1

### Requirement: Help Flag

#### Scenario: --help prints usage
- **GIVEN** `resolve.sh` is invocable
- **WHEN** `resolve.sh --help` is invoked
- **THEN** exit code is 0
- **AND** stdout contains "USAGE"

## logman.sh Test Suite

### Requirement: Isolated Fixture Structure

Each BATS test SHALL create an isolated temporary directory with both `logman.sh` and `resolve.sh` copied into `$TEST_DIR/fab/.kit/scripts/lib/`. Change directories with `.status.yaml` SHALL be created under `$TEST_DIR/fab/changes/`. `teardown()` SHALL remove `$TEST_DIR`.

#### Scenario: Setup creates isolated environment with both scripts
- **GIVEN** a BATS test begins execution
- **WHEN** `setup()` runs
- **THEN** both `logman.sh` and `resolve.sh` exist in the kit scripts path
- **AND** a change directory with `.status.yaml` is pre-created for test use

### Requirement: Command Subcommand

The `command` subcommand SHALL append one JSON line with fields `ts`, `event: "command"`, `cmd`, and optional `args`.

#### Scenario: command appends one line
- **GIVEN** a `.history.jsonl` file with N existing lines
- **WHEN** `logman.sh command <change> "test-cmd" "test-args"` is invoked
- **THEN** `.history.jsonl` has N+1 lines

#### Scenario: command JSON has required fields
- **GIVEN** a change directory exists
- **WHEN** `logman.sh command <change> "my-skill"` is invoked
- **THEN** the appended JSON line contains `"event":"command"` and `"cmd":"my-skill"`
- **AND** the line contains a `"ts"` field

#### Scenario: command with args includes args field
- **GIVEN** a change directory exists
- **WHEN** `logman.sh command <change> "my-skill" "spec"` is invoked
- **THEN** the appended JSON line contains `"args":"spec"`

#### Scenario: command without args omits args field
- **GIVEN** a change directory exists
- **WHEN** `logman.sh command <change> "my-skill"` is invoked (no 4th argument)
- **THEN** the appended JSON line does NOT contain `"args"`

### Requirement: Confidence Subcommand

The `confidence` subcommand SHALL append one JSON line with fields `ts`, `event: "confidence"`, `score` (numeric), `delta`, and `trigger`.

#### Scenario: confidence produces valid JSON
- **GIVEN** a change directory exists
- **WHEN** `logman.sh confidence <change> 3.8 "+0.5" "calc-score"` is invoked
- **THEN** the appended line contains `"event":"confidence"`, `"score":3.8`, `"delta":"+0.5"`, `"trigger":"calc-score"`

### Requirement: Review Subcommand

The `review` subcommand SHALL append one JSON line with fields `ts`, `event: "review"`, `result`, and optional `rework`.

#### Scenario: review passed without rework
- **GIVEN** a change directory exists
- **WHEN** `logman.sh review <change> "passed"` is invoked
- **THEN** the appended line contains `"event":"review"` and `"result":"passed"`
- **AND** the line does NOT contain `"rework"`

#### Scenario: review failed with rework type
- **GIVEN** a change directory exists
- **WHEN** `logman.sh review <change> "failed" "fix-code"` is invoked
- **THEN** the appended line contains `"event":"review"`, `"result":"failed"`, and `"rework":"fix-code"`

### Requirement: Append-Only Behavior

`logman.sh` SHALL never overwrite or modify existing lines in `.history.jsonl`. New entries are appended.

#### Scenario: existing lines preserved
- **GIVEN** `.history.jsonl` contains 3 lines of pre-existing content
- **WHEN** `logman.sh command <change> "test"` is invoked
- **THEN** the original 3 lines are unchanged
- **AND** the file now has 4 lines total

### Requirement: File Creation

When `.history.jsonl` does not exist, `logman.sh` SHALL create it with the first log entry.

#### Scenario: creates .history.jsonl when absent
- **GIVEN** a change directory exists but `.history.jsonl` does not
- **WHEN** `logman.sh command <change> "first"` is invoked
- **THEN** `.history.jsonl` is created
- **AND** it contains exactly 1 line

### Requirement: Error Cases

#### Scenario: no subcommand returns error
- **GIVEN** `logman.sh` is invocable
- **WHEN** invoked with no arguments
- **THEN** exit code is 1
- **AND** stderr contains "No subcommand"

#### Scenario: unknown subcommand returns error
- **GIVEN** `logman.sh` is invocable
- **WHEN** `logman.sh badcmd <change>` is invoked
- **THEN** exit code is 1
- **AND** stderr contains "Unknown subcommand"

#### Scenario: command with wrong argument count returns error
- **GIVEN** `logman.sh` is invocable
- **WHEN** `logman.sh command <change>` is invoked (missing cmd argument)
- **THEN** exit code is 1

#### Scenario: --help prints usage
- **GIVEN** `logman.sh` is invocable
- **WHEN** `logman.sh --help` is invoked
- **THEN** exit code is 0
- **AND** stdout contains "USAGE"

### Requirement: Change Resolution Integration

`logman.sh` resolves the `<change>` argument via `resolve.sh --dir` internally. Tests SHALL verify this integration works with different input forms.

#### Scenario: 4-char ID resolves correctly
- **GIVEN** a change folder `260228-a1b2-test-change` exists with `.status.yaml`
- **WHEN** `logman.sh command a1b2 "test"` is invoked
- **THEN** the log line is written to `fab/changes/260228-a1b2-test-change/.history.jsonl`

#### Scenario: unresolvable change returns error
- **GIVEN** no change folders exist
- **WHEN** `logman.sh command nonexistent "test"` is invoked
- **THEN** exit code is 1

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Follow changeman/test.bats fixture pattern — isolated TEST_DIR with scripts copied in | Confirmed from intake #1. Four existing BATS suites all use identical setup/teardown with mktemp isolation | S:90 R:90 A:95 D:95 |
| 2 | Certain | Test resolve.sh directly (not via changeman passthrough) | resolve.sh is a standalone script with its own CLI; changeman's resolve tests cover the passthrough, not the full resolve API | S:90 R:95 A:90 D:95 |
| 3 | Certain | Copy both logman.sh and resolve.sh into logman fixtures | logman.sh calls resolve.sh via relative path from LIB_DIR; real script is needed (not a stub) since resolve logic is simple and side-effect-free | S:85 R:90 A:90 D:90 |
| 4 | Certain | Test files located at src/lib/resolve/test.bats and src/lib/logman/test.bats | Confirmed from intake #2. Convention: test directory under src/lib/{script-name}/ with test.bats alongside test-simple.sh | S:95 R:95 A:95 D:95 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
