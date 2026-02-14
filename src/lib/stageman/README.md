# Stage Manager (stageman)

Bash utility for querying workflow stages and states from the canonical schema (`fab/.kit/schemas/workflow.yaml`). Source it in scripts to replace hardcoded stage/state knowledge with schema-driven queries.

## Sources of Truth

- **Schema**: `fab/.kit/schemas/workflow.yaml` — canonical workflow definition
- **Implementation**: `fab/.kit/scripts/lib/stageman.sh` — main file (distributed with kit)
- **Dev symlink**: `src/lib/stageman/stageman.sh` → `../../../fab/.kit/scripts/lib/stageman.sh`
- **Schema docs**: `docs/memory/fab-workflow/schemas.md` — what the schema defines and design principles

## Usage

### As Library

```bash
source "$(dirname "$0")/stageman.sh"

get_all_stages              # List all stage IDs in order
get_stage_number "spec"     # Get 1-indexed position (2)
get_state_symbol "active"   # Get display symbol (●)
validate_status_file path   # Validate .status.yaml against schema
```

### As Command

```bash
stageman.sh --help      # Show usage and function reference
stageman.sh --version   # Show library and schema version
stageman.sh --test      # Run self-tests
```

## API Reference

### State Queries

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `get_all_states` | — | newline-separated state IDs | 0 |
| `validate_state <state>` | state ID | — | 0 valid, 1 invalid |
| `get_state_symbol <state>` | state ID | single symbol char | 0 found, 1 not |
| `get_state_suffix <state>` | state ID | display suffix (may be empty) | 0 |
| `is_terminal_state <state>` | state ID | — | 0 terminal, 1 not |

### Stage Queries

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `get_all_stages` | — | newline-separated stage IDs in order | 0 |
| `validate_stage <stage>` | stage ID | — | 0 valid, 1 invalid |
| `get_stage_number <stage>` | stage ID | 1-indexed position | 0 |
| `get_stage_name <stage>` | stage ID | human-readable name | 0 |
| `get_stage_artifact <stage>` | stage ID | filename or empty | 0 |
| `get_allowed_states <stage>` | stage ID | newline-separated states | 0 |
| `get_initial_state <stage>` | stage ID | default state | 0 |
| `is_required_stage <stage>` | stage ID | — | 0 required, 1 optional |
| `has_auto_checklist <stage>` | stage ID | — | 0 yes, 1 no |

### .status.yaml Accessors

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `get_progress_map <file>` | .status.yaml path | `stage:state` pairs, one per line | 0 |
| `get_checklist <file>` | .status.yaml path | `generated:{val}`, `completed:{val}`, `total:{val}` | 0 |
| `get_confidence <file>` | .status.yaml path | `certain:{val}`, `confident:{val}`, `tentative:{val}`, `unresolved:{val}`, `score:{val}` | 0 |
| `get_stage_metrics <file> [stage]` | .status.yaml path (+ optional stage) | stage metrics as key:value lines | 0 |

### Progression

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `get_current_stage <file>` | .status.yaml path | active stage ID | 0 |
| `get_next_stage <stage>` | current stage ID | next stage ID | 0 found, 1 at end |

### Validation

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `validate_status_file <file>` | .status.yaml path | errors to stderr | 0 valid, 1 invalid |
| `validate_stage_state <stage> <state>` | stage + state IDs | — | 0 allowed, 1 not |

### Display

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `format_state <state>` | state ID | symbol + suffix | 0 |

### Write + History

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `set_stage_state <file> <stage> <state> [driver]` | status path + state update | in-place status update | 0 success, 1 validation error |
| `transition_stages <file> <from> <to> [driver]` | forward transition | in-place status update | 0/1 |
| `set_checklist_field <file> <field> <value>` | checklist mutation | in-place status update | 0/1 |
| `set_confidence_block <file> <c> <cf> <t> <u> <s>` | confidence replacement | in-place status update | 0/1 |
| `set_stage_metric <file> <stage> <field> <value>` | stage metric mutation | in-place status update | 0/1 |
| `log_command <file> <cmd> [args] [outcome]` | command metadata | append JSON event | 0/1 |
| `log_confidence <file> <score> <delta> <trigger>` | confidence metadata | append JSON event | 0/1 |
| `log_review <file> <passed\|failed> [rework]` | review metadata | append JSON event | 0/1 |

## CLI Interface

| Command | Description |
|---------|-------------|
| `--help` | Display usage, available functions, examples |
| `--version` | Display library and schema version |
| `--test` | Run self-tests on all functions |
| *(no args)* | Default to `--test` |

## Requirements

- Bash 4.0+
- GNU coreutils (grep, sed, awk)
- Mike Farah `yq` v4.x (status read/write and validation operations)
- Works on macOS and Linux

## Testing

```bash
# Quick smoke test
src/lib/stageman/test-simple.sh

# Self-test from main file
fab/.kit/scripts/lib/stageman.sh --test
```

## Changelog

### 2.0.0 (2026-02-14)

- Migrated status accessors, validation, and write functions from line parsing to `yq` v4
- Added stage metrics API: `get_stage_metrics`, `set_stage_metric`
- Added history event API: `log_command`, `log_confidence`, `log_review`
- Added automatic `stage_metrics` side-effects on stage activation/completion
- Added strict runtime detection for Mike Farah `yq` v4 with install guidance

### 1.1.0 (2026-02-14)

- Renamed from `stageman.sh` to `_stageman.sh` (underscore prefix for internal libraries), later renamed back to `stageman.sh`
- Added `.status.yaml` accessor functions: `get_progress_map`, `get_checklist`, `get_confidence`
- Refactored `get_current_stage` to use `get_progress_map` internally

### 1.0.0 (2026-02-12)

- Renamed from `workflow-lib.sh` to `stageman.sh`
- Reversed directory structure: main file in `fab/.kit/scripts/lib/`, dev symlink in `src/lib/stageman/`
- All state/stage query functions (20+)
- Validation functions (`validate_status_file`, `validate_stage_state`)
- CLI interface (`--help`, `--version`, `--test`)
- Path resolution for both src and symlink locations
