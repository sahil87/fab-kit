# Spec: Consolidate Script Signatures

**Change**: 260216-gcw7-DEV-1041-consolidate-script-signatures
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/schemas.md`

## Non-Goals

- **Batch script unification** — The three batch scripts (`batch-fab-archive-change.sh`, `batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh`) share ~30 lines of dispatch pattern (`--list`/`--all`/positional) but differ meaningfully in data sources (`.status.yaml` vs. `backlog.md` vs. change folders), execution model (single Claude session vs. tmux tabs), and filtering logic. The shared code is too small to justify extraction. Leave as-is.
- **changeman.sh style convergence** — `changeman.sh` uses `--flag <value>` style; `stageman.sh` uses positional subcommands. The different styles are justified: changeman has 2 subcommands with optional named parameters, while stageman has many subcommands with fixed positional args. No style change.
- **Functional changes to remaining subcommands** — This change only removes unused surface area. No behavioral changes to retained subcommands, no new subcommands, no modified output formats.

## stageman.sh: Remove Unused CLI Surface Area

### Requirement: Unused functions SHALL be removed entirely

Functions in `stageman.sh` that are called by no external script, no skill, and no other internal function SHALL be deleted (both function definition and CLI dispatch entry). This reduces the codebase size and eliminates untested-in-production code paths.

**Functions to remove (12):**

| Function | Reason |
|----------|--------|
| `is_terminal_state()` | No internal or external callers |
| `get_stage_number()` | No internal or external callers |
| `get_stage_name()` | No internal or external callers |
| `get_stage_artifact()` | No internal or external callers |
| `get_initial_state()` | No internal or external callers |
| `is_required_stage()` | No internal or external callers |
| `has_auto_checklist()` | No internal or external callers |
| `get_state_symbol()` | Only caller is `format_state()`, also being removed |
| `get_state_suffix()` | Only caller is `format_state()`, also being removed |
| `format_state()` | No external callers; skills format state display directly |
| `get_stage_metrics()` | No callers; `_apply_metrics_side_effect` uses direct yq |
| `set_stage_metric()` | No callers; `_apply_metrics_side_effect` uses direct yq |

#### Scenario: Dead function removed entirely
- **GIVEN** `is_terminal_state()` exists in `stageman.sh` with a corresponding `is-terminal` CLI dispatch entry
- **WHEN** the consolidation is applied
- **THEN** both the function definition and the `is-terminal)` case arm are deleted
- **AND** the `--help` text no longer lists `is-terminal`

#### Scenario: All 12 functions removed
- **GIVEN** the 12 functions listed above exist in `stageman.sh`
- **WHEN** the consolidation is applied
- **THEN** none of the 12 functions exist in the file
- **AND** none of their CLI dispatch entries exist in the case block

### Requirement: Internal-only CLI dispatch entries SHALL be removed

Functions that are called only by other functions within `stageman.sh` (never via CLI by external scripts or skills) SHALL have their CLI `case` arms removed. The function definitions SHALL be retained since they are part of the internal API.

**CLI dispatch entries to remove (function retained):**

| CLI Entry | Used by (internally) |
|-----------|---------------------|
| `all-states` | `validate_state()` |
| `validate-state` | `validate_status_file()` |
| `validate-stage` | `set_stage_state()`, `transition_stages()` |
| `allowed-states` | `validate_stage_state()` |
| `validate-stage-state` | `set_stage_state()`, `transition_stages()` |
| `next-stage` | `transition_stages()` |

#### Scenario: Internal-only dispatch removed, function kept
- **GIVEN** `get_all_states()` is called by `validate_state()` internally
- **AND** no external script or skill calls `stageman.sh all-states`
- **WHEN** the consolidation is applied
- **THEN** the `all-states)` case arm is deleted from the CLI dispatch
- **AND** the `get_all_states()` function definition is preserved
- **AND** `validate_state()` continues to call `get_all_states()` as a bash function

### Requirement: Self-test mode SHALL be removed

The `--test` flag, `--version` flag, and empty-argument default (which invoke `run_tests()` and `show_version()` respectively) SHALL be removed. The `run_tests()` and `show_version()` functions SHALL be deleted. The comprehensive bats test suite (`src/lib/stageman/test.bats`, 75 tests) provides full coverage; the inline self-test is redundant. The version string in `show_version()` was never maintained (shows "1.0.0" since initial release).

The empty-argument case (`""`) SHALL display the help text instead of running self-tests.

#### Scenario: Self-test and version removed
- **GIVEN** `stageman.sh --test` currently runs `run_tests()` and `--version` runs `show_version()`
- **WHEN** the consolidation is applied
- **THEN** `stageman.sh --test` outputs "Unknown option: --test" to stderr and exits 1
- **AND** `stageman.sh --version` outputs "Unknown option: --version" to stderr and exits 1
- **AND** `stageman.sh` (no args) outputs the help text

### Requirement: Retained CLI subcommands SHALL be exactly the externally-used set

After consolidation, the CLI dispatch SHALL contain exactly these 14 subcommands plus 1 meta entry:

**Subcommands (14):**

| Category | Subcommand | External Callers |
|----------|-----------|-----------------|
| Stage query | `all-stages` | `preflight.sh` |
| Accessors | `progress-map` | `preflight.sh` |
| Accessors | `checklist` | `preflight.sh` |
| Accessors | `confidence` | `preflight.sh`, `calc-score.sh` |
| Progression | `current-stage` | `preflight.sh` |
| Validation | `validate-status-file` | `preflight.sh` |
| Write | `set-state` | `changeman.sh`, skills |
| Write | `transition` | skills |
| Write | `set-checklist` | skills, `_generation.md` |
| Write | `set-confidence` | `calc-score.sh` |
| Write | `set-confidence-fuzzy` | `calc-score.sh` |
| History | `log-command` | `changeman.sh`, skills |
| History | `log-confidence` | `calc-score.sh` |
| History | `log-review` | skills |

**Meta entries (1):** `--help`

**Default (no args):** display help text

#### Scenario: Retained subcommand works unchanged
- **GIVEN** the consolidation is applied
- **WHEN** `stageman.sh transition .status.yaml intake spec fab-continue` is called
- **THEN** the behavior is identical to pre-consolidation (atomic two-write, metrics side-effects, last_updated refresh)

#### Scenario: Removed subcommand returns error
- **GIVEN** the consolidation is applied
- **WHEN** `stageman.sh stage-number spec` is called
- **THEN** stderr outputs "Unknown option: stage-number"
- **AND** exit code is 1

### Requirement: Help text SHALL reflect the reduced subcommand set

The `show_help()` function SHALL list only the 14 retained subcommands organized by category. Removed subcommands SHALL NOT appear. Category sections with no remaining subcommands SHALL be omitted entirely.

**Categories to remove entirely:**
- "State queries" (all 5 subcommands removed)
- "Display" (format-state removed)
- "Stage metrics" (both subcommands removed)

**Categories to reduce:**
- "Stage queries" → only `all-stages` remains
- "Progression" → only `current-stage` remains

#### Scenario: Help text matches retained set
- **GIVEN** the consolidation is applied
- **WHEN** `stageman.sh --help` is run
- **THEN** the output lists exactly 14 subcommands
- **AND** no removed subcommand name appears in the output

## stageman.sh: Internal Structure

### Requirement: `_apply_metrics_side_effect` SHALL be preserved

The `_apply_metrics_side_effect()` internal helper SHALL remain unchanged. It is called by `set_stage_state()` and `transition_stages()` and performs direct yq writes for stage metrics. It does NOT depend on the removed `get_stage_metrics()` or `set_stage_metric()` functions.

#### Scenario: Metrics side-effects work after function removal
- **GIVEN** `get_stage_metrics()` and `set_stage_metric()` have been removed
- **WHEN** `stageman.sh set-state .status.yaml spec active fab-continue` is called
- **THEN** `_apply_metrics_side_effect` writes `started_at`, `driver`, and `iterations` to `stage_metrics.spec`

## SPEC-stageman.md: Update API Reference

### Requirement: SPEC-stageman.md SHALL reflect the reduced CLI

`src/lib/stageman/SPEC-stageman.md` SHALL be updated to:

1. Remove rows for all 18 removed subcommands from the API Reference tables
2. Remove empty table sections (State Queries, Display, Stage Metrics)
3. Reduce Stage Queries section to only `all-stages`
4. Reduce Progression section to only `current-stage`
5. Update the Usage code block to remove examples of deleted subcommands (e.g., `stage-number spec`, `state-symbol active`)
6. Remove "Self-test" from Testing section (`stageman.sh --test` no longer exists) and `--version` from Usage section
7. Add a changelog entry for version 3.0.0 documenting the consolidation

#### Scenario: SPEC tables match implementation
- **GIVEN** the SPEC is updated
- **WHEN** comparing SPEC-stageman.md API Reference to stageman.sh CLI dispatch
- **THEN** every subcommand in the SPEC has a corresponding dispatch entry
- **AND** every dispatch entry has a corresponding SPEC row
- **AND** no removed subcommand appears in the SPEC

## test.bats: Update Test Suite

### Requirement: Tests for removed subcommands SHALL be deleted

`src/lib/stageman/test.bats` SHALL have all `@test` blocks removed that exercise deleted subcommands or deleted functions. Tests for retained subcommands SHALL remain unchanged.

**Tests to remove** (by subcommand exercised):
- All `is-terminal` tests
- All `stage-number` tests
- All `stage-name` tests
- All `stage-artifact` tests
- All `initial-state` tests
- All `is-required` tests
- All `has-auto-checklist` tests
- All `state-symbol` tests
- All `state-suffix` tests
- All `format-state` tests
- All `stage-metrics` tests (read)
- All `set-stage-metric` tests (write)
- All `all-states` CLI invocation tests (function is kept but CLI dispatch removed)
- All `validate-state` CLI invocation tests
- All `validate-stage` CLI invocation tests
- All `allowed-states` CLI invocation tests
- All `validate-stage-state` CLI invocation tests
- All `next-stage` CLI invocation tests
- All `--test` / self-test tests (if any)

**Tests to retain:**
- All tests for the 14 retained subcommands (`all-stages`, `progress-map`, `checklist`, `confidence`, `current-stage`, `validate-status-file`, `set-state`, `transition`, `set-checklist`, `set-confidence`, `set-confidence-fuzzy`, `log-command`, `log-confidence`, `log-review`)
- All tests for `--help`
- All tests for error handling (unknown subcommand, missing arguments)

#### Scenario: Retained tests pass
- **GIVEN** tests for removed subcommands have been deleted
- **WHEN** `bats src/lib/stageman/test.bats` is run
- **THEN** all remaining tests pass (exit 0)

#### Scenario: Test count is reduced
- **GIVEN** the current test suite has 75 tests
- **WHEN** tests for removed subcommands are deleted
- **THEN** the test count is reduced (exact count depends on how many tests target removed subcommands)

### Requirement: `test-simple.sh` SHALL be updated

`src/lib/stageman/test-simple.sh` SHALL have assertions for removed subcommands removed. Assertions for retained subcommands (e.g., `all-stages`, `progress-map`, `checklist`, `confidence`) SHALL remain.

#### Scenario: test-simple.sh passes
- **GIVEN** removed subcommand assertions have been deleted from `test-simple.sh`
- **WHEN** `bash src/lib/stageman/test-simple.sh` is run
- **THEN** the script exits 0 with "All tests passed"

## Memory Files: Update Descriptions

### Requirement: kit-architecture.md SHALL reflect the reduced CLI

`docs/memory/fab-workflow/kit-architecture.md` Shell Scripts section for `lib/stageman.sh` SHALL be updated to:

1. Change "~35 CLI subcommands" to the actual retained count (14)
2. Replace the 6-category bullet list with a simplified 4-category list reflecting only retained subcommands:
   - **Schema/accessor subcommands**: `all-stages`, `progress-map`, `checklist`, `confidence`, `current-stage`, `validate-status-file`
   - **Write subcommands**: `set-state`, `transition`, `set-checklist`, `set-confidence`, `set-confidence-fuzzy`
   - **History subcommands**: `log-command`, `log-confidence`, `log-review`
3. Note that internal helper functions (validation, state queries) are retained as implementation details but not exposed via CLI
4. Update the test count reference if it changes

#### Scenario: Memory describes actual CLI
- **GIVEN** the memory file is updated
- **WHEN** comparing the subcommand listing to `stageman.sh --help`
- **THEN** the categories and subcommand names match

### Requirement: schemas.md MAY remain unchanged

`docs/memory/fab-workflow/schemas.md` references stageman indirectly and does not list specific subcommands. No changes required unless the overall schema interaction pattern changes (it does not in this change).

## Deprecated Requirements

### Removed: Schema Query CLI Subcommands

**Subcommands**: `all-states`, `validate-state`, `state-symbol`, `state-suffix`, `is-terminal`, `validate-stage`, `stage-number`, `stage-name`, `stage-artifact`, `allowed-states`, `initial-state`, `is-required`, `has-auto-checklist`, `validate-stage-state`, `next-stage`, `format-state`, `stage-metrics`, `set-stage-metric`

**Reason**: These subcommands have no external callers (scripts or skills). Functions used internally are preserved; functions with no callers at all are deleted. The CLI surface area is reduced from ~35 to 14 subcommands.

**Migration**: N/A — no external consumers exist. If a future script or skill needs schema query access, the function can be re-added with a CLI dispatch entry (~5 lines each). The bats test suite provides a template for the test structure.

### Removed: Self-Test Mode and Version Flag

**Subcommands**: `--test`, `--version`, `""` (empty arg default to tests)

**Reason**: The comprehensive bats test suite (`src/lib/stageman/test.bats`) renders the inline `run_tests()` function redundant. The `--version` flag printed a stale "1.0.0" string that was never maintained. Both are dead features.

**Migration**: Run `bats src/lib/stageman/test.bats` for testing. Version tracking lives in `fab/.kit/VERSION` and `SPEC-stageman.md` changelog.

## Design Decisions

1. **Remove functions vs. remove only CLI dispatch**
   - *Chosen*: Remove both function AND dispatch for functions with zero callers (internal or external). Remove only dispatch for functions with internal-only callers.
   - *Why*: Functions with zero callers are dead code — they increase file size and cognitive load without providing value. Functions with internal callers are implementation details that support the public API.
   - *Rejected*: Keep all functions, remove only CLI dispatch — preserves dead code that will never be tested in production.

2. **Aggressive vs. conservative removal**
   - *Chosen*: Remove all 18 unused CLI entries (12 dead functions + 6 internal-only dispatch entries). Keep exactly the externally-used set.
   - *Why*: Every retained subcommand has a verified external caller. Removed subcommands are trivial to re-add (5-line case arm + function). The bats test suite documents the original contract for reference.
   - *Rejected*: Keep "potentially useful" subcommands like `format-state` and `stage-number` — violates the change's goal of reducing maintenance surface area. No current caller = no current value.

3. **Batch scripts: no consolidation**
   - *Chosen*: Leave the three batch scripts as separate files with no shared library extraction.
   - *Why*: Shared dispatch pattern is ~30 lines. Scripts differ in data source (`backlog.md` vs. `.status.yaml` vs. change folders), execution model (single Claude session vs. tmux tabs), and filtering logic. Extracting a library adds a file and indirection for minimal deduplication.
   - *Rejected*: Single `batch-fab.sh` with verb dispatch — would mix unrelated logic and make each verb harder to understand in isolation.

4. **changeman.sh interface: no change**
   - *Chosen*: Preserve the `--flag <value>` interface for changeman.sh.
   - *Why*: changeman has 2 subcommands with optional named parameters (where ordering doesn't matter and names aid readability). stageman has 14 subcommands with fixed positional args (where brevity and tab-completion matter). The styles serve different ergonomic goals.
   - *Rejected*: Convert changeman to positional args — would lose the readability of `--slug`, `--folder`, `--change-id` for a script where you need to distinguish multiple optional string arguments.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All scripts in scope are internal to `.kit/` | Confirmed from intake #1: Constitution V (Portability) — `.kit/` is self-contained, no external consumers | S:90 R:90 A:95 D:90 |
| 2 | Certain | Tests serve as the contract for subcommand existence | Confirmed from intake #2: Memory states "75 tests serve as contract test for future reimplementation" | S:85 R:85 A:90 D:90 |
| 3 | Certain | stageman.sh is the primary consolidation target | Confirmed from intake #3: audit verified 18 of 32 subcommands have no external callers | S:90 R:80 A:90 D:85 |
| 4 | Certain | Batch scripts share insufficient logic to warrant extraction | Audit shows ~30 lines of shared dispatch pattern vs. ~100 lines of unique logic per script | S:85 R:85 A:90 D:85 |
| 5 | Confident | changeman flag style vs. stageman positional style should remain different | Confirmed from intake #5: 2 subcommands with optional named params vs. 14 with positional — different ergonomic needs | S:70 R:80 A:75 D:70 |
| 6 | Confident | Dead code removal is safe without deprecation period | Confirmed from intake #6: all callers verified via grep; no external consumers beyond `.kit/` and skill files | S:75 R:70 A:80 D:70 |
| 7 | Confident | Internal-only functions should lose CLI dispatch but retain function definitions | Internal functions are used by write/validation functions that ARE externally called; removing the function would break the public API | S:75 R:70 A:80 D:75 |
| 8 | Confident | SPEC-stageman.md and test.bats updates are part of this change | User explicitly requested; tests are the contract, SPEC is the API reference — both must match implementation | S:90 R:80 A:85 D:85 |
| 9 | Confident | `_apply_metrics_side_effect` remains correct after removing `get_stage_metrics` and `set_stage_metric` | Verified via source inspection (lines 297-322): function uses direct yq writes, no dependency on removed functions | S:80 R:70 A:85 D:75 |
<!-- clarified: _apply_metrics_side_effect independence — upgraded from Tentative to Confident after source verification -->

9 assumptions (4 certain, 5 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-16

1. **Q**: Upgrade assumption #9 (`_apply_metrics_side_effect` independence) from Tentative to Confident?
   **A**: Yes — source inspection confirms direct yq writes with no dependency on removed functions. Upgraded.

2. **Q**: Should `show_version()` be updated from stale "1.0.0" to "3.0.0"?
   **A**: Remove `--version` entirely — the version string was never maintained, and version tracking exists elsewhere (`fab/.kit/VERSION`, `SPEC-stageman.md` changelog). Also removes `show_version()` function.
