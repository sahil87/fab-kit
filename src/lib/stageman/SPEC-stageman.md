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
"$STAGEMAN" progress-map .status.yaml  # Extract stage:state pairs
"$STAGEMAN" current-stage .status.yaml # Detect active stage

# Write subcommands
"$STAGEMAN" set-state .status.yaml spec done fab-continue
"$STAGEMAN" transition .status.yaml spec tasks fab-continue

# Flags
"$STAGEMAN" --help      # Show usage and subcommand reference
```

## API Reference

### Stage Queries

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `all-stages` | — | newline-separated stage IDs in order | 0 |

### .status.yaml Accessors

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `progress-map <file>` | .status.yaml path | `stage:state` pairs, one per line | 0 |
| `checklist <file>` | .status.yaml path | `generated:{val}`, `completed:{val}`, `total:{val}` | 0 |
| `confidence <file>` | .status.yaml path | `certain:{val}`, `confident:{val}`, `tentative:{val}`, `unresolved:{val}`, `score:{val}` | 0 |

### Progression

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `current-stage <file>` | .status.yaml path | active stage ID | 0 |

### Validation

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `validate-status-file <file>` | .status.yaml path | errors to stderr | 0 valid, 1 invalid |

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
# Full test suite (53 tests)
bats src/lib/stageman/test.bats

# Quick smoke test
src/lib/stageman/test-simple.sh
```

## Changelog

### 3.0.0 (2026-02-16)

- Consolidated CLI surface area: removed 18 unused subcommands (12 dead functions + 6 internal-only dispatch entries)
- Reduced from ~35 to 14 externally-used subcommands + `--help`
- Removed `--test`, `--version` flags and `run_tests()`, `show_version()` functions
- Empty-arg default now shows help instead of running self-tests
- Internal helper functions (`validate_state`, `validate_stage`, `get_allowed_states`, `validate_stage_state`, `get_next_stage`, `get_all_states`) retained for use by write/validation functions but no longer exposed via CLI
- Updated help text to reflect reduced subcommand set
- Test suite reduced from 75 to 53 tests (removed tests for deleted subcommands)
- Fixed pre-existing `brief` → `intake` stage name bug in `test-simple.sh` fixture

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
