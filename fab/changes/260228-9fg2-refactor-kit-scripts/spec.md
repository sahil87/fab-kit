# Spec: Refactor Kit Scripts Into Atomic Responsibilities

**Change**: 260228-9fg2-refactor-kit-scripts
**Created**: 2026-02-28
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the workflow schema (`workflow.yaml`) or the stage state machine logic — only the script boundary and naming change
- Adding new subcommands or features beyond what's needed for the responsibility split
- Modifying the confidence formula or gate thresholds — only deduplicating the existing formula
- Changing the `.status.yaml` schema or `.history.jsonl` format — only changing which scripts write to them

## Script Architecture: resolve.sh

### Requirement: Pure Change Resolution

`resolve.sh` SHALL be a standalone script at `fab/.kit/scripts/lib/resolve.sh` that converts any change reference to a canonical output format. It SHALL have no side effects — no file writes, no `.status.yaml` modifications, no logging.

#### Scenario: Default output is 4-char change ID
- **GIVEN** a change folder `260228-9fg2-refactor-kit-scripts` exists in `fab/changes/`
- **WHEN** `resolve.sh 9fg2` is called with no output flag
- **THEN** stdout contains `9fg2`

#### Scenario: Folder output via --folder flag
- **GIVEN** a change folder `260228-9fg2-refactor-kit-scripts` exists
- **WHEN** `resolve.sh --folder 9fg2` is called
- **THEN** stdout contains `260228-9fg2-refactor-kit-scripts`

#### Scenario: Directory path output via --dir flag
- **GIVEN** a change folder `260228-9fg2-refactor-kit-scripts` exists
- **WHEN** `resolve.sh --dir 9fg2` is called
- **THEN** stdout contains `fab/changes/260228-9fg2-refactor-kit-scripts/`

#### Scenario: Status file path output via --status flag
- **GIVEN** a change folder `260228-9fg2-refactor-kit-scripts` exists with `.status.yaml`
- **WHEN** `resolve.sh --status 9fg2` is called
- **THEN** stdout contains `fab/changes/260228-9fg2-refactor-kit-scripts/.status.yaml`

### Requirement: Resolution Input Forms

`resolve.sh` SHALL accept the same input forms currently handled by `changeman.sh cmd_resolve()`: 4-char change ID, folder name substring (case-insensitive), full folder name, or no argument (reads `fab/current`). The archive directory SHALL be excluded from resolution.

#### Scenario: Resolve from fab/current when no argument
- **GIVEN** `fab/current` contains `260228-9fg2-refactor-kit-scripts`
- **WHEN** `resolve.sh` is called with no arguments
- **THEN** stdout contains `9fg2`

#### Scenario: Substring matching
- **GIVEN** a change folder `260228-9fg2-refactor-kit-scripts` exists
- **WHEN** `resolve.sh refactor-kit` is called
- **THEN** stdout contains `9fg2`

#### Scenario: Single-change guess fallback
- **GIVEN** `fab/current` does not exist and exactly one change folder exists
- **WHEN** `resolve.sh` is called with no arguments
- **THEN** stdout contains the 4-char ID of the single change
- **AND** stderr contains a notice about single-change guessing

#### Scenario: Ambiguous match error
- **GIVEN** two change folders both contain `refactor` in their names
- **WHEN** `resolve.sh refactor` is called
- **THEN** exit code is 1
- **AND** stderr lists the matching folders

#### Scenario: No match error
- **GIVEN** no change folder matches `nonexistent`
- **WHEN** `resolve.sh nonexistent` is called
- **THEN** exit code is 1
- **AND** stderr contains "No change matches"

### Requirement: Change ID Extraction

`resolve.sh` SHALL extract the 4-char change ID from the folder name by parsing the `YYMMDD-XXXX-slug` naming convention. The ID is the second hyphen-delimited segment.

#### Scenario: ID extraction from folder name
- **GIVEN** a resolved folder name `260228-9fg2-refactor-kit-scripts`
- **WHEN** the ID extraction runs
- **THEN** the result is `9fg2`

## Script Architecture: statusman.sh

### Requirement: Rename from stageman.sh

`stageman.sh` SHALL be renamed to `statusman.sh` at the same path (`fab/.kit/scripts/lib/statusman.sh`). All internal variable names referencing "stageman" (e.g., `STAGEMAN_DIR`) SHALL be updated to use "statusman" equivalents. The `show_help()` output SHALL reflect the new name.

#### Scenario: Script is callable by new name
- **GIVEN** `statusman.sh` exists at `fab/.kit/scripts/lib/statusman.sh`
- **WHEN** `statusman.sh --help` is called
- **THEN** help text references `statusman.sh` throughout
- **AND** `stageman.sh` does not exist at the old path

### Requirement: Remove History Logging Functions

The three history-logging functions (`log_command`, `log_confidence`, `log_review`) and their CLI dispatch cases (`log-command`, `log-confidence`, `log-review`) SHALL be removed from statusman.sh. The "History" section of the help text SHALL be removed.

#### Scenario: History subcommands no longer recognized
- **GIVEN** `statusman.sh` exists
- **WHEN** `statusman.sh log-command 9fg2 "test"` is called
- **THEN** exit code is 1
- **AND** stderr contains "Unknown option"

### Requirement: Remove resolve_change_arg

The `resolve_change_arg()` function SHALL be removed from statusman.sh. All CLI dispatch cases SHALL resolve the `<change>` argument by calling `resolve.sh --status` externally and using the returned `.status.yaml` path.

#### Scenario: statusman uses resolve.sh for change resolution
- **GIVEN** `statusman.sh` and `resolve.sh` both exist
- **WHEN** `statusman.sh progress-map 9fg2` is called
- **THEN** the command succeeds (resolve.sh handles resolution)
- **AND** the internal `resolve_change_arg()` function does not exist in the source

### Requirement: Auto-Log Review Outcomes

`statusman.sh finish <change> review [driver]` SHALL automatically call `logman.sh review <change> "passed"` after successfully transitioning the review stage to `done`.

`statusman.sh fail <change> review [driver] [rework]` SHALL automatically call `logman.sh review <change> "failed" [rework]` after successfully transitioning the review stage to `failed`. The `fail` subcommand SHALL accept an optional `[rework]` argument (4th positional for review stage) that is passed through to logman.

#### Scenario: Finish review auto-logs pass
- **GIVEN** a change with `review: active`
- **WHEN** `statusman.sh finish 9fg2 review fab-continue` is called
- **THEN** `.status.yaml` shows `review: done`
- **AND** `.history.jsonl` contains a `{"event":"review","result":"passed"}` entry

#### Scenario: Fail review auto-logs failure with rework type
- **GIVEN** a change with `review: active`
- **WHEN** `statusman.sh fail 9fg2 review fab-ff fix-code` is called
- **THEN** `.status.yaml` shows `review: failed`
- **AND** `.history.jsonl` contains a `{"event":"review","result":"failed","rework":"fix-code"}` entry

#### Scenario: Non-review finish does not log
- **GIVEN** a change with `spec: active`
- **WHEN** `statusman.sh finish 9fg2 spec fab-continue` is called
- **THEN** `.status.yaml` shows `spec: done`
- **AND** no review entry is added to `.history.jsonl`

### Requirement: Retained Subcommands

All existing subcommands not listed for removal SHALL remain with identical behavior: `all-stages`, `progress-map`, `checklist`, `confidence`, `current-stage`, `display-stage`, `progress-line`, `validate-status-file`, `start`, `advance`, `finish`, `reset`, `fail`, `set-change-type`, `set-checklist`, `set-confidence`, `set-confidence-fuzzy`, `add-issue`, `get-issues`, `add-pr`, `get-prs`.

#### Scenario: Event commands work unchanged
- **GIVEN** a change with `intake: pending`
- **WHEN** `statusman.sh start 9fg2 intake fab-continue` is called
- **THEN** `.status.yaml` shows `intake: active`

## Script Architecture: logman.sh

### Requirement: Standalone History Logger

`logman.sh` SHALL be a standalone CLI script at `fab/.kit/scripts/lib/logman.sh` with three subcommands: `command`, `confidence`, `review`. Each subcommand SHALL resolve `<change>` via `resolve.sh --dir`, append a single JSON line to `{change_dir}/.history.jsonl`, and exit.

#### Scenario: Log a command invocation
- **GIVEN** a change `9fg2` exists
- **WHEN** `logman.sh command 9fg2 "fab-continue" "spec"` is called
- **THEN** `.history.jsonl` has a new line with `{"ts":"...","event":"command","cmd":"fab-continue","args":"spec"}`

#### Scenario: Log a confidence change
- **GIVEN** a change `9fg2` exists
- **WHEN** `logman.sh confidence 9fg2 4.1 "+0.5" "calc-score"` is called
- **THEN** `.history.jsonl` has a new line with `{"ts":"...","event":"confidence","score":4.1,"delta":"+0.5","trigger":"calc-score"}`

#### Scenario: Log a review outcome
- **GIVEN** a change `9fg2` exists
- **WHEN** `logman.sh review 9fg2 "failed" "fix-code"` is called
- **THEN** `.history.jsonl` has a new line with `{"ts":"...","event":"review","result":"failed","rework":"fix-code"}`

#### Scenario: Review log without rework
- **GIVEN** a change `9fg2` exists
- **WHEN** `logman.sh review 9fg2 "passed"` is called
- **THEN** `.history.jsonl` has a new line with `{"ts":"...","event":"review","result":"passed"}` (no `rework` field)

### Requirement: No Side Effects Beyond Append

`logman.sh` SHALL NOT read or write `.status.yaml`. It SHALL NOT read `.history.jsonl`. It SHALL only append to `.history.jsonl`. Each JSON line SHALL include a `ts` field with the current ISO-8601 timestamp.

#### Scenario: Append-only behavior
- **GIVEN** `.history.jsonl` contains 3 existing lines
- **WHEN** `logman.sh command 9fg2 "test"` is called
- **THEN** `.history.jsonl` contains 4 lines
- **AND** the first 3 lines are unchanged

## Script Architecture: preflight.sh

### Requirement: --driver Flag for Auto-Logging

`preflight.sh` SHALL accept an optional `--driver <skill-name>` flag. When present and validation succeeds, preflight SHALL call `logman.sh command <change> <skill-name>` before emitting the YAML output. The `<change>` argument SHALL be the resolved change name (passed as the folder name, which resolve.sh can handle).

#### Scenario: Preflight with --driver logs invocation
- **GIVEN** a valid project and active change `9fg2`
- **WHEN** `preflight.sh --driver fab-continue` is called
- **THEN** `.history.jsonl` contains a `{"event":"command","cmd":"fab-continue"}` entry
- **AND** the YAML output is emitted normally

#### Scenario: Preflight without --driver does not log
- **GIVEN** a valid project and active change `9fg2`
- **WHEN** `preflight.sh` is called with no `--driver` flag
- **THEN** no entry is added to `.history.jsonl`
- **AND** the YAML output is emitted normally

#### Scenario: Preflight with --driver and change override
- **GIVEN** a valid project and a change `9fg2`
- **WHEN** `preflight.sh --driver fab-continue 9fg2` is called
- **THEN** `.history.jsonl` for the `9fg2` change contains the command entry

### Requirement: Update Internal References

`preflight.sh` SHALL reference `statusman.sh` instead of `stageman.sh` for all internal calls (variable name `STATUSMAN` instead of `STAGEMAN`).

#### Scenario: Preflight calls statusman
- **GIVEN** `statusman.sh` exists and `stageman.sh` does not
- **WHEN** `preflight.sh` is called
- **THEN** preflight succeeds (it uses statusman internally)

## Script Architecture: changeman.sh

### Requirement: Delegate Resolution to resolve.sh

`changeman.sh` SHALL delegate all change resolution to `resolve.sh` internally. The `cmd_resolve()` function SHALL be replaced by calls to `resolve.sh`. The `changeman.sh resolve` subcommand MAY be retained as a passthrough to `resolve.sh --folder` for backward compatibility, or removed entirely.

#### Scenario: changeman.sh new uses resolve.sh internally
- **GIVEN** `resolve.sh` exists
- **WHEN** `changeman.sh new --slug test-change` is called
- **THEN** the change is created successfully
- **AND** `changeman.sh` does not contain `cmd_resolve()` function

### Requirement: Update Log Calls

`changeman.sh` SHALL call `logman.sh command` instead of `stageman.sh log-command` for recording `new` and `rename` operations. The internal `STAGEMAN` variable SHALL be replaced with `STATUSMAN` referencing the renamed script.

#### Scenario: changeman new logs via logman
- **GIVEN** a new change is created
- **WHEN** `changeman.sh new --slug test --log-args "Test"` completes
- **THEN** `.history.jsonl` contains a command entry logged by logman (not stageman)

## Script Architecture: calc-score.sh

### Requirement: DRY Internal Helpers

`calc-score.sh` SHALL extract two internal helper functions:

1. **`count_grades <file>`** — parse the `## Assumptions` table from a markdown file. Output grade counts and dimension score sums in a parseable format.
2. **`compute_score <confident> <tentative> <total> <expected_min>`** — compute the confidence score using the formula: `base = max(0.0, 5.0 - 0.3*confident - 1.0*tentative); cover = min(1.0, total/expected_min); score = base * cover` (or `0.0` if unresolved > 0). Return the score on stdout.

The grade counting loop and score formula SHALL each appear exactly once in the script.

#### Scenario: Gate check uses shared helpers
- **GIVEN** `calc-score.sh` with `count_grades` and `compute_score` functions
- **WHEN** `calc-score.sh --check-gate --stage intake <change>` is called
- **THEN** the result is identical to the previous behavior
- **AND** the score formula appears only once in the source

#### Scenario: Normal scoring uses shared helpers
- **GIVEN** a change with `spec.md` containing an Assumptions table
- **WHEN** `calc-score.sh <change>` is called
- **THEN** the confidence block is written to `.status.yaml` via statusman
- **AND** a confidence event is logged via logman

### Requirement: Change Input via resolve.sh

`calc-score.sh` SHALL accept `<change>` (any form supported by resolve.sh) instead of `<change-dir>`. It SHALL resolve the change directory internally via `resolve.sh --dir`.

#### Scenario: calc-score accepts change ID
- **GIVEN** a change `9fg2` exists with `spec.md`
- **WHEN** `calc-score.sh 9fg2` is called
- **THEN** the score is computed correctly (resolve.sh converts `9fg2` → directory path)

### Requirement: Update Internal References

`calc-score.sh` SHALL reference `statusman.sh` instead of `stageman.sh` for `.status.yaml` writes, and `logman.sh` instead of `stageman.sh` for confidence logging.

#### Scenario: calc-score calls statusman and logman
- **GIVEN** `statusman.sh` and `logman.sh` exist
- **WHEN** `calc-score.sh 9fg2` completes
- **THEN** `.status.yaml` confidence block is updated (via statusman)
- **AND** `.history.jsonl` has a confidence entry (via logman)

## Test Suite

### Requirement: Rename stageman Test Directory

`src/lib/stageman/` SHALL be renamed to `src/lib/statusman/`. `SPEC-stageman.md` SHALL be renamed to `SPEC-statusman.md`. All tests referencing `stageman.sh` SHALL be updated to reference `statusman.sh`. Tests for removed subcommands (`log-command`, `log-confidence`, `log-review`) SHALL be removed.

#### Scenario: statusman tests pass
- **GIVEN** `src/lib/statusman/test.bats` references `statusman.sh`
- **WHEN** the test suite runs
- **THEN** all statusman tests pass
- **AND** no test references `stageman.sh`

### Requirement: New resolve.sh Tests

`src/lib/resolve/` SHALL contain tests for: all four output modes (`--id`, `--folder`, `--dir`, `--status`), substring matching, exact matching, `fab/current` fallback, single-change guessing, error cases (no match, multiple matches, no changes directory).

#### Scenario: resolve tests cover all output modes
- **GIVEN** `src/lib/resolve/test.bats` exists
- **WHEN** the test suite runs
- **THEN** tests for `--id`, `--folder`, `--dir`, `--status` all pass

### Requirement: New logman.sh Tests

`src/lib/logman/` SHALL contain tests for each subcommand (`command`, `confidence`, `review`), verifying JSON structure in `.history.jsonl`, append-only behavior, and change resolution via resolve.sh.

#### Scenario: logman tests verify JSON structure
- **GIVEN** `src/lib/logman/test.bats` exists
- **WHEN** the test suite runs
- **THEN** tests verify each subcommand produces valid JSON with required fields

### Requirement: Update Existing Tests

- `src/lib/changeman/test.bats` — update references from `stageman` to `statusman`, remove or redirect tests that previously tested resolve behavior
- `src/lib/calc-score/test.bats` — update for `<change>` input convention (was `<change-dir>`), verify DRY'd helpers
- `src/lib/preflight/test.bats` — add tests for `--driver` flag and its auto-logging behavior

#### Scenario: All existing test suites pass after refactor
- **GIVEN** all test files are updated
- **WHEN** the full test suite runs
- **THEN** all tests pass with no references to `stageman.sh`

## Reference Updates

### Requirement: Update Skill Files

All skill files that reference `stageman.sh` SHALL be updated to reference `statusman.sh`. Manual `log-command` invocation lines SHALL be removed (replaced by `preflight.sh --driver`). Manual `log-review` invocation lines SHALL be removed (replaced by statusman auto-logging).

Affected skill files: `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `fab-clarify.md`, `fab-new.md`, `fab-status.md`, `fab-archive.md`, `git-pr.md`, `_preamble.md`, `_scripts.md`, `_generation.md`.

#### Scenario: No skill file references stageman
- **GIVEN** all skill files are updated
- **WHEN** searching for "stageman" across `fab/.kit/skills/`
- **THEN** zero matches found

#### Scenario: No skill file contains manual log-command
- **GIVEN** all skill files are updated
- **WHEN** searching for "log-command" across `fab/.kit/skills/`
- **THEN** zero matches found (preflight --driver handles this)

#### Scenario: No skill file contains manual log-review
- **GIVEN** all skill files are updated
- **WHEN** searching for "log-review" across `fab/.kit/skills/`
- **THEN** zero matches found (statusman auto-logging handles this)

### Requirement: Rewrite _scripts.md

`_scripts.md` SHALL be rewritten to document the 5-script architecture: `resolve.sh`, `changeman.sh`, `statusman.sh`, `logman.sh`, `calc-score.sh`, `preflight.sh`. The old `<change>` argument convention table SHALL be replaced with documentation of resolve.sh's role as the canonical resolver. The internal call graph SHALL be documented.

#### Scenario: _scripts.md documents all five scripts
- **GIVEN** `_scripts.md` is rewritten
- **WHEN** an agent reads it
- **THEN** it finds sections for resolve.sh, changeman.sh, statusman.sh, logman.sh, calc-score.sh, and preflight.sh
- **AND** the internal call graph is documented

### Requirement: Update Memory Files

`docs/memory/fab-workflow/kit-architecture.md` SHALL be updated to reflect the new script structure, rename, and call graph. `docs/memory/fab-workflow/execution-skills.md` SHALL be updated to reference `statusman.sh` and document the auto-logging behavior. All other memory files referencing `stageman` SHALL be updated.

#### Scenario: No memory file references stageman
- **GIVEN** all memory files are updated
- **WHEN** searching for "stageman" across `docs/memory/`
- **THEN** zero matches found

## Deprecated Requirements

### resolve_change_arg() in stageman.sh
**Reason**: Replaced by standalone `resolve.sh`. Stageman no longer resolves change arguments itself — it calls resolve.sh externally.
**Migration**: All callers that previously depended on stageman's internal resolution now use resolve.sh.

### log_command, log_confidence, log_review in stageman.sh
**Reason**: History logging is not stage management. Moved to `logman.sh`.
**Migration**: Direct callers switch to `logman.sh` subcommands. Skill files no longer call these directly — auto-logging via preflight and statusman handles it.

### Manual log-command in skill files
**Reason**: Replaced by `preflight.sh --driver <skill-name>` which auto-logs after successful validation.
**Migration**: Remove the manual `log-command` step from every skill's pre-flight section.

### Manual log-review in skill files
**Reason**: Replaced by auto-logging in `statusman.sh finish/fail review`.
**Migration**: Remove the manual `log-review` step from review verdict sections in `fab-continue.md`, `fab-ff.md`, `fab-fff.md`.

## Design Decisions

### 1. **resolve.sh as standalone script, not changeman subcommand**
- *Why*: resolve.sh is the universal dependency (~60 lines). Every other script calls it first. Embedding it in changeman forces every script to load 400+ lines of lifecycle logic for a 50-line substring matcher.
- *Rejected*: `changeman.sh resolve` as the canonical entry point — too much overhead for the most frequently called operation.

### 2. **logman.sh as CLI script, not sourced library**
- *Why*: Uniform `script subcommand args` calling convention across all kit scripts. Testable in isolation without sourcing setup. Same invocation pattern whether called from bash scripts or skill instructions.
- *Rejected*: Sourced library (`source logman.sh; log_command ...`) — different invocation pattern, harder to test in isolation, callers need to know the source path.

### 3. **Metadata writers stay in statusman (not extracted)**
- *Why*: `set_change_type`, `set_checklist`, `set_confidence_block`, `add_issue`, `add_pr` all share the same atomic-write pattern (tmpfile → yq → mv) and yq dependency. Splitting them into a separate script would duplicate this infrastructure for no practical gain.
- *Rejected*: Separate `metadataman.sh` — would create two scripts with identical write patterns, both operating on `.status.yaml`.

### 4. **Auto-logging over explicit logging**
- *Why*: `log-command` is called by every skill as boilerplate. `log-review` is always paired with `finish/fail review`. Making these implicit (via preflight --driver and statusman auto-log) eliminates ~15 manual logging lines across skill files and makes it impossible to forget logging.
- *Rejected*: Keep explicit logging — error-prone (skills can forget), boilerplate (every skill repeats the same line), violates DRY.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five-script architecture: resolve.sh, changeman.sh, statusman.sh, logman.sh, calc-score.sh | Confirmed from intake #1 — user proposed and refined across multiple exchanges | S:95 R:80 A:95 D:95 |
| 2 | Certain | Rename stageman.sh → statusman.sh | Confirmed from intake #2 — broader name for broader responsibility | S:90 R:75 A:90 D:95 |
| 3 | Certain | resolve.sh is standalone, not a changeman subcommand | Confirmed from intake #3 — universal dependency, ~60 lines | S:90 R:85 A:90 D:90 |
| 4 | Certain | logman.sh is a separate CLI script | Confirmed from intake #4 — user explicitly chose CLI for uniform calling | S:95 R:85 A:90 D:95 |
| 5 | Certain | Skills never call logman directly | Confirmed from intake #5 — auto-triggered via preflight, statusman, calc-score | S:95 R:80 A:90 D:90 |
| 6 | Certain | preflight.sh gains --driver flag | Confirmed from intake #6 — natural hook point for auto-logging | S:90 R:80 A:90 D:90 |
| 7 | Certain | statusman finish/fail review auto-logs | Confirmed from intake #7 — always paired with log-review today | S:90 R:80 A:85 D:90 |
| 8 | Certain | resolve.sh default output is 4-char ID with flags | Confirmed from intake #8 — --id (default), --folder, --dir, --status | S:90 R:85 A:90 D:90 |
| 9 | Certain | DRY calc-score.sh with count_grades() and compute_score() | Confirmed from intake #9 — 3x duplication eliminated | S:85 R:85 A:90 D:95 |
| 10 | Certain | calc-score.sh accepts <change> via resolve.sh | Upgraded from intake #10 (was Confident) — straightforward corollary of the resolution design, verified against source | S:85 R:80 A:90 D:90 |
| 11 | Confident | Metadata writers stay in statusman | Confirmed from intake #11 — shared atomic-write pattern, splitting duplicates infrastructure | S:80 R:80 A:80 D:85 |
| 12 | Confident | fail review gains optional [rework] argument | Confirmed from intake #12 — log_review already accepts rework, statusman passes through | S:75 R:85 A:80 D:85 |
| 13 | Confident | changeman.sh resolve subcommand retained as passthrough | New — backward compat for any external callers. Thin wrapper around resolve.sh --folder. Low cost, avoids breaking existing scripts | S:70 R:90 A:75 D:80 |
| 14 | Confident | resolve.sh uses repo-root-relative paths for --dir and --status output | New — matches existing convention in preflight.sh output (e.g., `change_dir: fab/changes/...`). Absolute paths would break portability | S:75 R:85 A:85 D:85 |

14 assumptions (10 certain, 4 confident, 0 tentative, 0 unresolved).
