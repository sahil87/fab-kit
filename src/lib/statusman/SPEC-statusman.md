# Stage Manager (statusman)

CLI utility for querying workflow stages and states from the canonical schema (`fab/.kit/schemas/workflow.yaml`). Invoke as a subprocess to replace hardcoded stage/state knowledge with schema-driven queries.

## Sources of Truth

- **Schema**: `fab/.kit/schemas/workflow.yaml` — canonical workflow definition
- **Implementation**: `fab/.kit/scripts/lib/statusman.sh` — main file (distributed with kit)
- **Dev symlink**: `src/lib/statusman/statusman.sh` → `../../../fab/.kit/scripts/lib/statusman.sh`
- **Schema docs**: `fab/docs/fab-workflow/schemas.md` — what the schema defines and design principles

## Usage

```bash
STATUSMAN="path/to/statusman.sh"

# Query subcommands
"$STATUSMAN" all-stages              # List all stage IDs in order
"$STATUSMAN" progress-map .status.yaml  # Extract stage:state pairs
"$STATUSMAN" current-stage .status.yaml # Detect active stage

# Event subcommands
"$STATUSMAN" start 6boq spec fab-continue
"$STATUSMAN" finish 6boq spec fab-continue

# Flags
"$STATUSMAN" --help      # Show usage and subcommand reference
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

### Event Commands

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `start <change> <stage> [driver]` | change ID/path, stage, optional driver | — | 0 ok, 1 invalid |
| `advance <change> <stage> [driver]` | change ID/path, stage, optional driver | — | 0 ok, 1 invalid |
| `finish <change> <stage> [driver]` | change ID/path, stage, optional driver | — | 0 ok, 1 invalid |
| `reset <change> <stage> [driver]` | change ID/path, stage, optional driver | — | 0 ok, 1 invalid |
| `fail <change> <stage> [driver]` | change ID/path, stage, optional driver | — | 0 ok, 1 invalid |

### Write Commands

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
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
# Full test suite (115 tests)
bats src/lib/statusman/test.bats

# Quick smoke test
src/lib/statusman/test-simple.sh
```

## Changelog

### 4.0.0 (2026-02-26)

- Replaced `set-state` and `transition` with 5 event commands: `start`, `advance`, `finish`, `reset`, `fail`
- All event commands accept change identifiers (4-char ID, partial slug, full name) or raw file paths via `resolve_change_arg`
- `finish` atomically activates next pending stage; `reset` cascades downstream stages to pending
- `fail` restricted to review stage only
- Driver parameter optional on all event commands
- Transitions defined by event-keyed format in workflow.yaml (replaces from→to with condition)
- Non-event commands also accept change identifiers via universal resolution
- Test suite expanded from 53 to 115 tests (all event-based)

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

### 2.0.0 (2026-02-28)

- Renamed from `stageman.sh` to `statusman.sh`
- Removed `resolve_change_arg()` — resolution delegated to `resolve.sh`
- Removed `log_command`, `log_confidence`, `log_review` — logging delegated to `logman.sh`
- Added auto-log review outcomes: `finish review` calls `logman.sh review "passed"`, `fail review` calls `logman.sh review "failed" [rework]`
- `fail` subcommand gains optional `[rework]` argument for review stage

### 1.1.0 (2026-02-14)

- Added `.status.yaml` accessor functions: `get_progress_map`, `get_checklist`, `get_confidence`
- Refactored `get_current_stage` to use `get_progress_map` internally

### 1.0.0 (2026-02-12)

- Renamed from `workflow-lib.sh` to `stageman.sh` (later renamed to `statusman.sh` in 2.0.0)
- Reversed directory structure: main file in `fab/.kit/scripts/lib/`, dev symlink in `src/lib/statusman/`
- All state/stage query functions (20+)
- Validation functions (`validate_status_file`, `validate_stage_state`)
- CLI interface (`--help`, `--version`, `--test`)
- Path resolution for both src and symlink locations
