# Spec: Migrate Stageman to CLI-Only Interface

**Change**: 260215-lqm5-stageman-cli-only
**Created**: 2026-02-15
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/preflight.md`, `docs/memory/fab-workflow/schemas.md`

## Non-Goals

- Rust rewrite — this change establishes the CLI-only contract; the rewrite is a future change
- New functionality — no new query/write capabilities beyond exposing existing functions as CLI subcommands
- Output format changes — multi-value accessors retain existing `key:value` line-oriented format

## Stageman CLI: New Subcommands

### Requirement: All read/query functions SHALL have CLI subcommands

`stageman.sh` SHALL expose every function currently available only via `source` as a CLI subcommand in the `case` dispatch block. The subcommand names SHALL use kebab-case, matching existing CLI conventions (`set-state`, `set-checklist`, etc.).

The complete subcommand mapping:

**State queries:**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `all-states` | `get_all_states` | — | newline-separated state IDs |
| `validate-state` | `validate_state` | `<state>` | exit 0 if valid, exit 1 if invalid |
| `state-symbol` | `get_state_symbol` | `<state>` | single symbol character |
| `state-suffix` | `get_state_suffix` | `<state>` | display suffix (may be empty) |
| `is-terminal` | `is_terminal_state` | `<state>` | exit 0 if terminal, exit 1 if not |

**Stage queries:**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `all-stages` | `get_all_stages` | — | newline-separated stage IDs |
| `validate-stage` | `validate_stage` | `<stage>` | exit 0 if valid, exit 1 if invalid |
| `stage-number` | `get_stage_number` | `<stage>` | 1-indexed position |
| `stage-name` | `get_stage_name` | `<stage>` | human-readable name |
| `stage-artifact` | `get_stage_artifact` | `<stage>` | filename or empty |
| `allowed-states` | `get_allowed_states` | `<stage>` | newline-separated states |
| `initial-state` | `get_initial_state` | `<stage>` | default state |
| `is-required` | `is_required_stage` | `<stage>` | exit 0 if required, exit 1 if optional |
| `has-auto-checklist` | `has_auto_checklist` | `<stage>` | exit 0 if yes, exit 1 if no |
| `validate-stage-state` | `validate_stage_state` | `<stage> <state>` | exit 0 if allowed, exit 1 if not |

**.status.yaml accessors:**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `progress-map` | `get_progress_map` | `<file>` | `stage:state` pairs, one per line |
| `checklist` | `get_checklist` | `<file>` | `key:value` lines (generated, completed, total) |
| `confidence` | `get_confidence` | `<file>` | `key:value` lines (certain, confident, tentative, unresolved, score) |

**Stage metrics:**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `stage-metrics` | `get_stage_metrics` | `<file> [stage]` | all: `stage:{json}` per line; single: `field:value` per line |
| `set-stage-metric` | `set_stage_metric` | `<file> <stage> <field> <value>` | — (writes to file) |

**Progression:**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `current-stage` | `get_current_stage` | `<file>` | active stage ID |
| `next-stage` | `get_next_stage` | `<stage>` | next stage ID; exit 1 if at end |

**Validation:**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `validate-status-file` | `validate_status_file` | `<file>` | errors to stderr; exit 0 valid, exit 1 invalid |

**Display:**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `format-state` | `format_state` | `<state>` | symbol + suffix |

**Write (missing from CLI):**
| Subcommand | Function | Args | Output |
|------------|----------|------|--------|
| `set-confidence-fuzzy` | `set_confidence_block_fuzzy` | `<file> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d>` | — (writes to file) |

#### Scenario: Schema query via CLI
- **GIVEN** `stageman.sh` is invoked as a subprocess
- **WHEN** the subcommand is `all-stages`
- **THEN** stdout contains all 6 stage IDs, one per line, in pipeline order
- **AND** exit code is 0

#### Scenario: Boolean query returns exit code only
- **GIVEN** `stageman.sh` is invoked with `validate-state done`
- **WHEN** the state is valid
- **THEN** exit code is 0 and stdout is empty
- **AND** when invoked with `validate-state bogus`, exit code is 1

#### Scenario: Accessor query via CLI
- **GIVEN** a valid `.status.yaml` file
- **WHEN** `stageman.sh progress-map <file>` is invoked
- **THEN** stdout contains `stage:state` pairs in the same format as the existing `get_progress_map` function

#### Scenario: Write subcommand via CLI
- **GIVEN** a valid `.status.yaml` file
- **WHEN** `stageman.sh set-confidence-fuzzy <file> 3 2 1 0 4.1 85.0 90.0 80.0 75.0` is invoked
- **THEN** the confidence block is updated with counts, score, fuzzy flag, and dimension means

#### Scenario: Argument validation for CLI subcommands
- **GIVEN** `stageman.sh` is invoked with a subcommand
- **WHEN** the argument count is wrong
- **THEN** a usage message is printed to stderr and exit code is 1

### Requirement: Help text SHALL reflect CLI-only interface

The `--help` output SHALL list all subcommands (read, write, and history) with usage signatures. The "As library" section and `source` examples SHALL be removed. The "AVAILABLE FUNCTIONS" section SHALL be replaced with a "SUBCOMMANDS" section organized by category.

#### Scenario: Help shows complete subcommand reference
- **GIVEN** `stageman.sh --help` is invoked
- **WHEN** the help text is displayed
- **THEN** all ~35 subcommands are listed (25 new read + 1 new write + existing 9 write/history)
- **AND** no `source` usage examples are present

## Stageman CLI: Dual-Mode Removal

### Requirement: The `BASH_SOURCE` guard SHALL be removed

The `if [ "${BASH_SOURCE[0]}" = "${0}" ]; then` guard at line 950 and its closing `fi` SHALL be removed. All code in the file SHALL execute unconditionally when the script is invoked.

#### Scenario: Direct invocation works without guard
- **GIVEN** `stageman.sh` is invoked directly (not sourced)
- **WHEN** any subcommand is passed
- **THEN** the subcommand executes normally without requiring the `BASH_SOURCE` check

### Requirement: Error handling SHALL use `exit` only

All `return 1 2>/dev/null || exit 1` patterns SHALL be replaced with `exit 1`. Since the file is no longer sourced, `return` is unnecessary and misleading.

#### Scenario: Error exits cleanly
- **GIVEN** `stageman.sh` is invoked with an invalid subcommand
- **WHEN** the script encounters an error
- **THEN** it exits with code 1 (no `return` fallback)

### Requirement: Source-oriented comments SHALL be removed

Comments referencing `source` usage patterns (e.g., "Usage as library:", "source stageman.sh") SHALL be removed from the file header and inline comments.

#### Scenario: No source references in comments
- **GIVEN** the migrated `stageman.sh`
- **WHEN** the file is inspected
- **THEN** no comments reference sourcing, importing, or library usage patterns

## Preflight Migration

### Requirement: `preflight.sh` SHALL invoke stageman via CLI subprocess calls

`preflight.sh` SHALL replace `source "$scripts_dir/lib/stageman.sh"` with subprocess invocations using `STAGEMAN="$scripts_dir/lib/stageman.sh"`. Each of the 6 current accessor calls SHALL become a `$STAGEMAN <subcommand>` call.

The migration mapping:
| Current (source pattern) | New (CLI pattern) |
|--------------------------|-------------------|
| `source "$scripts_dir/lib/stageman.sh"` | `STAGEMAN="$scripts_dir/lib/stageman.sh"` |
| `validate_status_file "$status_file"` | `$STAGEMAN validate-status-file "$status_file"` |
| `get_progress_map "$status_file"` | `$STAGEMAN progress-map "$status_file"` |
| `get_current_stage "$status_file"` | `$STAGEMAN current-stage "$status_file"` |
| `get_checklist "$status_file"` | `$STAGEMAN checklist "$status_file"` |
| `get_confidence "$status_file"` | `$STAGEMAN confidence "$status_file"` |
| `get_all_stages` | `$STAGEMAN all-stages` |

#### Scenario: Preflight produces identical YAML output
- **GIVEN** a valid change with `.status.yaml`
- **WHEN** `preflight.sh` is invoked
- **THEN** the stdout YAML output is byte-identical to the pre-migration output for the same `.status.yaml`

#### Scenario: Preflight error handling preserved
- **GIVEN** `preflight.sh` is invoked
- **WHEN** `validate-status-file` exits non-zero (CLI subprocess)
- **THEN** `preflight.sh` exits with code 1 and prints a diagnostic to stderr

### Requirement: `preflight.sh` SHALL stop sourcing `resolve-change.sh` via stageman dependency

Currently `preflight.sh` sources both `stageman.sh` and `resolve-change.sh`. After migration, `preflight.sh` SHALL still source `resolve-change.sh` directly (it provides `resolve_change` which sets `RESOLVED_CHANGE_NAME` — this is a variable-setting pattern incompatible with subprocess invocation). Only the `stageman.sh` source line is removed.

#### Scenario: Change resolution still works
- **GIVEN** `preflight.sh` is invoked with a change name override
- **WHEN** `resolve_change` is called
- **THEN** `RESOLVED_CHANGE_NAME` is set correctly (source pattern preserved for `resolve-change.sh`)

## Calc-Score Migration

### Requirement: `calc-score.sh` SHALL invoke stageman via CLI subprocess calls

`calc-score.sh` SHALL replace `source "$(dirname "$(readlink -f "$0")")/stageman.sh"` with a `STAGEMAN` variable and subprocess calls.

The migration mapping:
| Current (source pattern) | New (CLI pattern) |
|--------------------------|-------------------|
| `source "$(dirname "$(readlink -f "$0")")/stageman.sh"` | `STAGEMAN="$(dirname "$(readlink -f "$0")")/stageman.sh"` |
| `get_confidence "$status_file"` | `$STAGEMAN confidence "$status_file"` |
| `set_confidence_block "$status_file" ...` | `$STAGEMAN set-confidence "$status_file" ...` |
| `set_confidence_block_fuzzy "$status_file" ...` | `$STAGEMAN set-confidence-fuzzy "$status_file" ...` |
| `log_confidence "$change_dir" ...` | `$STAGEMAN log-confidence "$change_dir" ...` |

#### Scenario: Score computation produces identical results
- **GIVEN** a `spec.md` with an Assumptions table
- **WHEN** `calc-score.sh <change-dir>` is invoked
- **THEN** the confidence score, grade counts, and dimension means are identical to the pre-migration output

#### Scenario: Gate check still works
- **GIVEN** a `.status.yaml` with a confidence score
- **WHEN** `calc-score.sh --check-gate <change-dir>` is invoked
- **THEN** the gate result (pass/fail) is identical to the pre-migration output

## Test Suite Migration

### Requirement: Test suites SHALL use CLI invocation pattern

`src/lib/stageman/test.sh` and `src/lib/stageman/test-simple.sh` SHALL be rewritten to invoke `stageman.sh` as a subprocess instead of sourcing it. The `STAGEMAN` variable pattern SHALL be used consistently.

#### Scenario: Comprehensive test suite uses CLI pattern
- **GIVEN** `test.sh` is executed
- **WHEN** all assertions run
- **THEN** every test calls `$STAGEMAN <subcommand>` (no function calls)
- **AND** boolean tests check exit codes from subprocess invocation
- **AND** output tests capture stdout from subprocess invocation
- **AND** all existing test assertions are preserved (no coverage loss)

#### Scenario: Simple test suite uses CLI pattern
- **GIVEN** `test-simple.sh` is executed
- **WHEN** all assertions run
- **THEN** every test calls `$STAGEMAN <subcommand>` (no `source` line)

### Requirement: Test suites SHALL serve as contract tests for Rust rewrite

The migrated test suites define the exact CLI interface that any future implementation (Rust or otherwise) MUST satisfy. Tests SHALL validate:
- Subcommand argument counts and validation
- Exit codes for boolean queries
- stdout output format for data queries
- Write side-effects (`.status.yaml` mutations)
- Error messages to stderr

#### Scenario: Contract test completeness
- **GIVEN** the migrated `test.sh`
- **WHEN** a new binary is substituted for `stageman.sh`
- **THEN** all tests pass if and only if the binary implements the same CLI interface

## README Update

### Requirement: `src/lib/stageman/README.md` SHALL document CLI-only interface

The README SHALL remove the "As Library" usage section and replace it with CLI-only usage examples. The API reference table SHALL show subcommand names instead of function names.

#### Scenario: README reflects CLI-only usage
- **GIVEN** the updated README
- **WHEN** a developer reads it
- **THEN** all examples use `stageman.sh <subcommand>` syntax
- **AND** no `source` references appear

## Design Decisions

1. **`resolve-change.sh` remains source-only**: `resolve_change` communicates via shell variable (`RESOLVED_CHANGE_NAME`), which cannot work across process boundaries. Converting it to CLI would require stdout parsing and would change the error handling contract. Since only `preflight.sh` sources it, the migration cost/benefit is poor. It can be addressed independently if/when the Rust rewrite proceeds.
   - *Rejected*: Converting to CLI with stdout output — breaks the clean exit-code + variable-setting pattern; adds parsing complexity for no current benefit.

2. **`STAGEMAN` variable for path resolution**: Both `preflight.sh` and `calc-score.sh` define a `STAGEMAN` variable pointing to the stageman script path, then use `$STAGEMAN <subcommand>` throughout. This is more readable than repeating the full path and matches the pattern the brief established.
   - *Rejected*: Inline path on each call — verbose, error-prone, harder to update.

3. **Functions remain in file alongside CLI dispatch**: The function definitions (`get_all_stages`, `set_stage_state`, etc.) stay in the same file. The CLI dispatch block calls them. This keeps stageman.sh as a single self-contained script — no auxiliary files needed.
   - *Rejected*: Splitting functions into a separate library file — adds complexity, breaks the single-file portability, and the Rust rewrite will replace the entire file anyway.

## Deprecated Requirements

### Source/Import Interface

**Reason**: Replaced by CLI-only interface. All callers (preflight.sh, calc-score.sh, test suites) are migrated to subprocess invocation.
**Migration**: Use `$STAGEMAN <subcommand>` instead of `source stageman.sh` + function calls.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skills need no changes — they already use CLI interface | Confirmed from brief #1: all skill .md files invoke `lib/stageman.sh <subcommand>`, none source it | S:95 R:95 A:95 D:95 |
| 2 | Certain | CLI subcommand names use kebab-case | Confirmed from brief #2: consistent with existing CLI commands; standard convention | S:90 R:90 A:95 D:90 |
| 3 | Confident | Incremental 3-phase approach: add CLI → migrate callers → remove dual-mode | Confirmed from brief #3: Phase 1 purely additive (zero regression risk) | S:80 R:90 A:85 D:75 |
| 4 | Confident | Test suites become CLI-only contract tests | Confirmed from brief #4: CLI tests define the interface a Rust binary must satisfy | S:85 R:80 A:80 D:85 |
| 5 | Confident | Output format for multi-value queries uses existing `key:value` line convention | Confirmed from brief #5: matches existing accessor output; callers already parse this | S:85 R:85 A:90 D:80 |
| 6 | Certain | `set-confidence-fuzzy` is the only write function missing from CLI | Confirmed from brief #6: audited dispatch block against function list | S:95 R:90 A:95 D:95 |
| 7 | Certain | Boolean functions use exit codes (0=true, 1=false) with no stdout | Standard shell convention; existing functions already use this pattern (`validate_state`, `is_terminal_state`, etc.) | S:90 R:95 A:95 D:90 |
| 8 | Confident | `resolve-change.sh` remains source-only (not migrated to CLI) | Variable-setting pattern (`RESOLVED_CHANGE_NAME`) incompatible with subprocess; only 1 caller; Rust rewrite will handle differently | S:80 R:85 A:80 D:75 |
| 9 | Certain | Functions remain in the same file — no library split | Single-file portability is a core property; Rust rewrite replaces the whole file | S:90 R:90 A:90 D:90 |
| 10 | Confident | `--test` self-test mode removed or converted to use CLI dispatch | Self-test calls functions directly; after migration it should exercise CLI subcommands instead. Alternatively, remove it since `src/lib/stageman/test.sh` is the comprehensive suite | S:75 R:85 A:80 D:70 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
