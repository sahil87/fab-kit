# Preflight

**Domain**: fab-workflow

## Overview

The preflight script (`fab/.kit/scripts/fab-preflight.sh`) validates the active change's state and outputs structured YAML for agent consumption. It consolidates repeated validation logic from individual skills into a single reusable script.

## Requirements

### Structured YAML Output

`fab-preflight.sh` outputs a YAML document to stdout containing the active change's resolved state. Fields include:

- `name` — the change folder name (resolved via `_resolve-change.sh`)
- `change_dir` — path to `fab/changes/{name}/`, relative to `fab/`
- `stage` — current stage (derived via `get_current_stage` from `_stageman.sh`)
- `progress` — full progress map (all 6 stages with their status, via `get_progress_map`)
- `checklist.generated` — boolean (via `get_checklist`)
- `checklist.completed` — integer
- `checklist.total` — integer
- `confidence.certain` — integer (via `get_confidence`)
- `confidence.confident` — integer
- `confidence.tentative` — integer
- `confidence.unresolved` — integer
- `confidence.score` — float

Agents consume this output by running the script via Bash and parsing the stdout YAML directly.

### Validation Checks

The script validates in this order, stopping at the first failure:

1. `fab/config.yaml` and `fab/constitution.md` exist (project initialized)
2. Change name resolves (via `_resolve-change.sh` — from `$1` override or `fab/current`)
3. Change directory `fab/changes/{name}/` exists
4. `.status.yaml` exists within the change directory
5. `.status.yaml` passes schema validation via `validate_status_file()` from `_stageman.sh` (catches invalid states, missing stages, multiple active stages)

Each failure exits with code 1 and prints a diagnostic message to stderr.

### Accessor-Based Architecture

The script sources `_stageman.sh` and `_resolve-change.sh`, delegating all `.status.yaml` parsing to stageman accessor functions:

- **Change resolution**: `resolve_change` from `_resolve-change.sh` handles both default mode (reads `fab/current`) and override mode (fuzzy matching against `fab/changes/`)
- **Progress extraction**: `get_progress_map` returns `stage:state` pairs, consumed via `while IFS=: read -r`
- **Stage derivation**: `get_current_stage` (which itself uses `get_progress_map` internally)
- **Checklist fields**: `get_checklist` returns `generated`, `completed`, `total` with defaults
- **Confidence fields**: `get_confidence` returns `certain`, `confident`, `tentative`, `unresolved`, `score` with defaults
- **Schema validation**: `validate_status_file` for structural correctness

No inline `grep | sed` parsing of `.status.yaml` — all field extraction goes through stageman accessors.

### No External Dependencies

The script uses only POSIX-standard tools (`grep`, `sed`, `tr`, `cat`), Bash builtins, `_stageman.sh`, and `_resolve-change.sh` (which themselves use only POSIX tools). No `yq`, `jq`, Python, or other non-standard tools required.

### Idempotent and Read-Only

The script does not modify any files. Safe to run any number of times without side effects.

### Relative Path Resolution

All internal paths resolve relative to the script's own location via `$(dirname "$0")/../..`. Works regardless of the caller's working directory.

### Skill Integration

Skills that perform pre-flight checks (ff, apply, review, archive, continue, clarify) reference `fab-preflight.sh` instead of inline validation. On non-zero exit, the agent stops and surfaces the stderr message. On success, the agent uses the stdout YAML for change context.

Skills exempt from preflight: `init`, `switch`, `status`, `hydrate`, `help`, `new`.

## Design Decisions

### Accessor Functions Over Inline Parsing
**Decision**: `fab-preflight.sh` delegates all `.status.yaml` field extraction to `_stageman.sh` accessor functions (`get_progress_map`, `get_checklist`, `get_confidence`, `get_current_stage`) instead of inline `grep | sed`.
**Why**: Eliminates duplicated parsing logic across scripts. Stageman is the single owner of `.status.yaml` read semantics — field defaults, missing-block handling, and format normalization live in one place.
**Rejected**: Keeping inline extraction — maintained two copies of parsing logic (preflight + status) that could drift.

### Shared Change Resolution Library
**Decision**: Change name resolution (fuzzy matching against `fab/changes/`) extracted to `_resolve-change.sh`, sourced by both `fab-preflight.sh` and `fab-status.sh`.
**Why**: Both scripts had ~65 identical lines of resolution logic. The library uses a variable-setting pattern (`RESOLVED_CHANGE_NAME`) for clean exit code handling, and keeps error messages generic so callers add their own context.
**Rejected**: Consolidating into `_stageman.sh` — change resolution is pure filesystem/string matching with no stage awareness; mixing concerns would violate stageman's schema-query focus.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260213-puow-consolidate-status-reads | 2026-02-14 | Replaced inline `grep \| sed` parsing with stageman accessor calls (`get_progress_map`, `get_checklist`, `get_confidence`); delegated change resolution to `_resolve-change.sh`; added confidence fields to output; renamed `stageman.sh` → `_stageman.sh` |
| 260212-4tw0-migrate-scripts-stageman | 2026-02-12 | Migrated to source stageman.sh: dynamic stage iteration, schema validation via validate_status_file |
| 260212-v5p2-simplify-stages-entry-paths | 2026-02-12 | Updated from 6 to 5 stages, documented stage derivation from active entry |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated progress map to 6 stages |
| 260207-5mjv-preflight-grep-scripts | 2026-02-07 | Created preflight doc — script purpose, output format, validation order, skill integration |
