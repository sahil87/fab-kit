# Stage Manager (stageman)

CLI utility for querying workflow stages and states from the canonical schema (`fab/.kit/schemas/workflow.yaml`). Invoke as a subprocess to replace hardcoded stage/state knowledge with schema-driven queries.

## Sources of Truth

- **Schema**: `fab/.kit/schemas/workflow.yaml` — canonical workflow definition
- **Implementation**: `fab/.kit/scripts/lib/stageman.sh` — main file (distributed with kit)
- **Dev symlink**: `src/lib/stageman/stageman.sh` → `../../../fab/.kit/scripts/lib/stageman.sh`
- **Schema docs**: `fab/docs/fab-workflow/schemas.md` — what the schema defines and design principles

## Usage

```bash
STAGEMAN="path/to/stageman.sh"

# Query subcommands
"$STAGEMAN" all-stages              # List all stage IDs in order
"$STAGEMAN" stage-number spec       # Get 1-indexed position (2)
"$STAGEMAN" state-symbol active     # Get display symbol (●)
"$STAGEMAN" progress-map .status.yaml  # Extract stage:state pairs

# Write subcommands
"$STAGEMAN" set-state .status.yaml spec done fab-continue
"$STAGEMAN" transition .status.yaml spec tasks fab-continue

# Flags
"$STAGEMAN" --help      # Show usage and subcommand reference
"$STAGEMAN" --version   # Show version
"$STAGEMAN" --test      # Run self-tests
```

## API Reference

### State Queries

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `all-states` | — | newline-separated state IDs | 0 |
| `validate-state <state>` | state ID | — | 0 valid, 1 invalid |
| `state-symbol <state>` | state ID | single symbol char | 0 found, 1 not |
| `state-suffix <state>` | state ID | display suffix (may be empty) | 0 |
| `is-terminal <state>` | state ID | — | 0 terminal, 1 not |

### Stage Queries

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `all-stages` | — | newline-separated stage IDs in order | 0 |
| `validate-stage <stage>` | stage ID | — | 0 valid, 1 invalid |
| `stage-number <stage>` | stage ID | 1-indexed position | 0 |
| `stage-name <stage>` | stage ID | human-readable name | 0 |
| `stage-artifact <stage>` | stage ID | filename or empty | 0 |
| `allowed-states <stage>` | stage ID | newline-separated states | 0 |
| `initial-state <stage>` | stage ID | default state | 0 |
| `is-required <stage>` | stage ID | — | 0 required, 1 optional |
| `has-auto-checklist <stage>` | stage ID | — | 0 yes, 1 no |
| `validate-stage-state <stage> <state>` | stage + state IDs | — | 0 allowed, 1 not |

### .status.yaml Accessors

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `progress-map <file>` | .status.yaml path | `stage:state` pairs, one per line | 0 |
| `checklist <file>` | .status.yaml path | `generated:{val}`, `completed:{val}`, `total:{val}` | 0 |
| `confidence <file>` | .status.yaml path | `certain:{val}`, `confident:{val}`, `tentative:{val}`, `unresolved:{val}`, `score:{val}` | 0 |

### Stage Metrics

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `stage-metrics <file> [stage]` | .status.yaml path, optional stage | JSON metrics (all or one stage) | 0 |
| `set-stage-metric <file> <stage> <field> <value>` | path, stage, field, value | — | 0 |

### Progression

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `current-stage <file>` | .status.yaml path | active stage ID | 0 |
| `next-stage <stage>` | current stage ID | next stage ID | 0 found, 1 at end |

### Validation

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `validate-status-file <file>` | .status.yaml path | errors to stderr | 0 valid, 1 invalid |

### Display

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `format-state <state>` | state ID | symbol + suffix | 0 |

### Write Commands

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `set-state <file> <stage> <state> [driver]` | path, stage, state, optional driver | — | 0 |
| `transition <file> <from> <to> [driver]` | path, stages, optional driver | — | 0 |
| `set-checklist <file> <field> <value>` | path, field name, value | — | 0 |
| `set-confidence <file> <c> <cf> <t> <u> <score>` | path, grade counts, score | — | 0 |
| `set-confidence-fuzzy <file> <c> <cf> <t> <u> <score> <s> <r> <a> <d>` | path, grade counts, score, dimension means | — | 0 |

### History

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `log-command <change_dir> <cmd> [args]` | change dir, command name | — | 0 |
| `log-confidence <change_dir> <score> <delta> <trigger>` | change dir, score fields | — | 0 |
| `log-review <change_dir> <result> [rework]` | change dir, result | — | 0 |

## Requirements

- Bash 4.0+
- `yq` v4 (YAML processing)
- GNU coreutils (grep, sed, awk)
- Works on macOS and Linux

## Testing

```bash
# Full test suite (131 tests)
src/lib/stageman/test.sh

# Quick smoke test
src/lib/stageman/test-simple.sh

# Self-test from main file
fab/.kit/scripts/lib/stageman.sh --test
```

## Changelog

### 2.0.0 (2026-02-15)

- Migrated to CLI-only interface — all consumers use subprocess invocations
- Added ~25 CLI subcommands mapping to all internal functions
- Removed dual-mode (source + CLI) scaffolding
- Added `set-confidence-fuzzy` subcommand for SRAD dimension scores
- Fixed `((var++))` arithmetic incompatibility with `set -e` in subprocess mode
- Migrated all callers: `preflight.sh`, `calc-score.sh`
- Migrated test suites to CLI pattern (contract tests for future Rust rewrite)

### 1.1.0 (2026-02-14)

- Added `.status.yaml` accessor functions: `get_progress_map`, `get_checklist`, `get_confidence`
- Refactored `get_current_stage` to use `get_progress_map` internally

### 1.0.0 (2026-02-12)

- Renamed from `workflow-lib.sh` to `stageman.sh`
- Reversed directory structure: main file in `fab/.kit/scripts/lib/`, dev symlink in `src/lib/stageman/`
- All state/stage query functions (20+)
- Validation functions (`validate_status_file`, `validate_stage_state`)
- CLI interface (`--help`, `--version`, `--test`)
- Path resolution for both src and symlink locations
