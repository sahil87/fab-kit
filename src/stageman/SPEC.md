# Stage Manager (stageman) Specification

Version: 1.0.0

## Purpose

Provide a bash library for querying the canonical workflow schema (`fab/.kit/schemas/workflow.yaml`) with consistent, type-safe accessors.

## Requirements

### Schema Dependency

- MUST read schema from `fab/.kit/schemas/workflow.yaml`
- MUST fail gracefully if schema is not found
- MUST NOT cache schema data (always read fresh)

### Path Resolution

- MUST work when sourced from `src/stageman/stageman.sh`
- MUST work when sourced from `fab/.kit/scripts/stageman.sh` (symlink)
- MUST use `readlink -f` to resolve symlinks
- MUST handle both `BASH_SOURCE[0]` and `$0` for path detection

### Function Contracts

#### State Queries

**`get_all_states`**
- Input: none
- Output: newline-separated list of state IDs
- Exit: 0 always
- Example: `pending\nactive\ndone\nskipped\nfailed`

**`validate_state <state>`**
- Input: state ID string
- Output: none
- Exit: 0 if valid, 1 if invalid
- Side effects: none

**`get_state_symbol <state>`**
- Input: state ID string
- Output: single-character symbol
- Exit: 0 if found, 1 if not found
- Example: `active` → `●`

**`get_state_suffix <state>`**
- Input: state ID string
- Output: display suffix string (may be empty)
- Exit: 0 if state exists, 1 otherwise
- Example: `skipped` → ` (skipped)`

**`is_terminal_state <state>`**
- Input: state ID string
- Output: none
- Exit: 0 if terminal, 1 if not
- Definition: Terminal states (`done`, `skipped`) cannot transition without explicit reset

#### Stage Queries

**`get_all_stages`**
- Input: none
- Output: newline-separated list of stage IDs in workflow order
- Exit: 0 always
- Example: `brief\nspec\ntasks\napply\nreview\narchive`

**`validate_stage <stage>`**
- Input: stage ID string
- Output: none
- Exit: 0 if valid, 1 if invalid

**`get_stage_number <stage>`**
- Input: stage ID string
- Output: 1-indexed position (1-6)
- Exit: 0 if found
- Example: `spec` → `2`

**`get_stage_name <stage>`**
- Input: stage ID string
- Output: human-readable display name
- Exit: 0 if found
- Example: `spec` → `Specification`

**`get_stage_artifact <stage>`**
- Input: stage ID string
- Output: generated filename (empty if stage doesn't generate files)
- Exit: 0 always
- Example: `spec` → `spec.md`, `apply` → ``

**`get_allowed_states <stage>`**
- Input: stage ID string
- Output: newline-separated list of allowed state IDs
- Exit: 0 if stage found
- Example: `review` → `pending\nactive\ndone\nfailed`

**`get_initial_state <stage>`**
- Input: stage ID string
- Output: default state for new changes
- Exit: 0 if found
- Example: `brief` → `active`, `spec` → `pending`

**`is_required_stage <stage>`**
- Input: stage ID string
- Output: none
- Exit: 0 if required, 1 if optional

**`has_auto_checklist <stage>`**
- Input: stage ID string
- Output: none
- Exit: 0 if stage generates checklist, 1 otherwise

#### Progression

**`get_current_stage <status_file>`**
- Input: path to `.status.yaml` file
- Output: stage ID of first active stage, or `archive` if all done
- Exit: 0 always
- Behavior: Reads `progress:` block, finds first `active` state

**`get_next_stage <current_stage>`**
- Input: current stage ID
- Output: next stage ID in sequence
- Exit: 0 if next exists, 1 if current is last stage
- Example: `spec` → `tasks`

#### Validation

**`validate_status_file <status_file>`**
- Input: path to `.status.yaml` file
- Output: error messages to stderr (if invalid)
- Exit: 0 if valid, 1 if invalid
- Checks:
  - All stages have valid states
  - States are in `allowed_states` for that stage
  - Exactly 0-1 active stages
  - No missing progress fields

**`validate_stage_state <stage> <state>`**
- Input: stage ID, state ID
- Output: none
- Exit: 0 if state allowed for stage, 1 otherwise

#### Display

**`format_state <state>`**
- Input: state ID
- Output: formatted string (symbol + suffix)
- Exit: 0 if state exists
- Example: `skipped` → `— (skipped)`

### CLI Interface

When executed directly (not sourced):

**`stageman.sh --help`**
- Display usage, available functions, examples
- Exit: 0

**`stageman.sh --version`**
- Display library version and schema version
- Exit: 0

**`stageman.sh --test`**
- Run self-tests on all functions
- Display test results
- Exit: 0 if all pass, 1 if any fail

**`stageman.sh` (no args)**
- Default to `--test` for backward compatibility
- Exit: as `--test`

### Error Handling

- Schema not found: exit 1 with error to stderr
- Unknown CLI option: exit 1 with error to stderr
- Invalid function input: return empty string or exit 1 (per function spec)
- Validation errors: print to stderr, continue checking, return 1 at end

### Performance

- SHOULD minimize subprocess spawns
- SHOULD use awk over grep+sed chains where possible
- MAY cache within a single script execution (future enhancement)
- MUST NOT cache across script invocations

### Compatibility

- MUST work with bash 4.0+
- MUST work with GNU coreutils (grep, sed, awk)
- MUST NOT depend on external YAML parsers
- SHOULD work on macOS and Linux

## Non-Requirements

- JSON output format (future enhancement)
- Schema validation (schema is trusted)
- Writing to schema (read-only library)
- Colored output (caller's responsibility)

## Testing

### Unit Tests

Test each function independently:
- Valid inputs return expected outputs
- Invalid inputs fail gracefully
- Edge cases handled (empty strings, missing data)

### Integration Tests

Test sourcing from both locations:
- `src/stageman/stageman.sh`
- `fab/.kit/scripts/stageman.sh` (via symlink)

### Validation Tests

Test `validate_status_file` with:
- Valid status files (all checks pass)
- Invalid states
- Multiple active stages
- States not in `allowed_states`
- Missing progress fields

## Migration Path

This library replaces hardcoded stage/state knowledge in:
- `fab/.kit/scripts/fab-status.sh`
- `fab/.kit/scripts/fab-preflight.sh`
- `.claude/skills/*/SKILL.md`

See `fab/.kit/schemas/MIGRATION.md` for refactoring examples.

## Version History

**1.0.0** (2026-02-12)
- Initial release
- All state/stage query functions
- Validation functions
- CLI interface (--help, --version, --test)
- Path resolution for both src and symlink locations
