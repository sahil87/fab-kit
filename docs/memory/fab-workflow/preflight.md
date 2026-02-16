# Preflight

**Domain**: fab-workflow

## Overview

The preflight script (`fab/.kit/scripts/lib/preflight.sh`) validates the active change's state and outputs structured YAML for agent consumption. It consolidates repeated validation logic from individual skills into a single reusable script.

## Requirements

### Structured YAML Output

`lib/preflight.sh` outputs a YAML document to stdout containing the active change's resolved state. Fields include:

- `name` — the change folder name (resolved via `lib/resolve-change.sh`)
- `change_dir` — path to `fab/changes/{name}/`, relative to `fab/`
- `stage` — current stage (derived via `get_current_stage` from `lib/stageman.sh`)
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
2. Change name resolves (via `lib/resolve-change.sh` — from `$1` override or `fab/current`)
3. Change directory `fab/changes/{name}/` exists
4. `.status.yaml` exists within the change directory
5. `.status.yaml` passes schema validation via `validate_status_file()` from `lib/stageman.sh` (catches invalid states, missing stages, multiple active stages)

Each failure exits with code 1 and prints a diagnostic message to stderr.

### Accessor-Based Architecture

The script sources `lib/resolve-change.sh` (variable-setting pattern) and invokes `lib/stageman.sh` via CLI subprocess calls (`$STAGEMAN <subcommand>`), delegating all `.status.yaml` parsing to stageman subcommands:

- **Change resolution**: `resolve_change` from `lib/resolve-change.sh` (still sourced — uses variable-setting pattern incompatible with subprocess invocation) handles both default mode (reads `fab/current`) and override mode (fuzzy matching against `fab/changes/`)
- **Progress extraction**: `$STAGEMAN progress-map` returns `stage:state` pairs, consumed via `while IFS=: read -r`
- **Stage derivation**: `$STAGEMAN current-stage` (which itself uses `progress-map` internally)
- **Checklist fields**: `$STAGEMAN checklist` returns `generated`, `completed`, `total` with defaults
- **Confidence fields**: `$STAGEMAN confidence` returns `certain`, `confident`, `tentative`, `unresolved`, `score` with defaults
- **Schema validation**: `$STAGEMAN validate-status-file` for structural correctness

No inline `grep | sed` parsing of `.status.yaml` — all field extraction goes through stageman CLI subcommands.

### No External Dependencies

The script uses only POSIX-standard tools (`grep`, `sed`, `tr`, `cat`), Bash builtins, and `lib/resolve-change.sh` (sourced). It invokes `lib/stageman.sh` as a CLI subprocess — stageman requires `yq` v4, but preflight itself has no direct `yq` dependency.

### Idempotent and Read-Only

The script does not modify any files. Safe to run any number of times without side effects.

### Relative Path Resolution

All internal paths resolve relative to the script's own location via `$(dirname "$0")/../../..` (three levels up from `scripts/lib/` to the `fab/` root). Works regardless of the caller's working directory.

### Skill Integration

Skills that perform pre-flight checks (ff, apply, review, archive, continue, clarify) reference `lib/preflight.sh` instead of inline validation. On non-zero exit, the agent stops and surfaces the stderr message. On success, the agent uses the stdout YAML for change context.

Skills exempt from preflight: `init`, `switch`, `status`, `hydrate`, `help`, `new`.

## Design Decisions

### CLI Subprocess Over Source Import
**Decision**: `lib/preflight.sh` invokes `lib/stageman.sh` via CLI subprocess calls (`$STAGEMAN progress-map`, `$STAGEMAN checklist`, etc.) instead of sourcing it.
**Why**: Decouples preflight from stageman's internal function signatures. Enables future replacement of `stageman.sh` with a compiled binary (e.g., Rust) without modifying callers. The CLI interface is the stable contract.
**Rejected**: Continuing to source `stageman.sh` — tight coupling to internal function names; not compatible with a binary replacement.
*Updated by*: 260215-lqm5-stageman-cli-only (previously "Accessor Functions Over Inline Parsing")

### lib/ Subfolder for Internal Scripts
**Decision**: All internal scripts (`preflight.sh`, `stageman.sh`, `resolve-change.sh`, `calc-score.sh`, `sync-workspace.sh`) live in `fab/.kit/scripts/lib/` without underscore prefix, replacing the previous `_`-prefixed convention in the parent `scripts/` directory.
**Why**: The `lib/` subfolder provides a clearer structural boundary between internal plumbing and user-facing scripts than naming conventions alone. All internal scripts are co-located, making the dependency graph explicit.
**Rejected**: Retaining underscore prefix — naming conventions are less discoverable than directory structure.
*Introduced by*: 260214-q7f2-reorganize-src

### Shared Change Resolution Library
**Decision**: Change name resolution (fuzzy matching against `fab/changes/`) extracted to `lib/resolve-change.sh`, sourced by `lib/preflight.sh`, `fab-status.sh`, and the `/fab-switch` skill (via Bash invocation in the Argument Flow).
**Why**: Multiple callers had identical resolution logic. The library uses a variable-setting pattern (`RESOLVED_CHANGE_NAME`) for clean exit code handling, and keeps error messages generic so callers add their own context. `/fab-switch` delegates to the shell script rather than performing in-prompt string matching, making resolution deterministic regardless of model tier.
**Rejected**: Consolidating into `lib/stageman.sh` — change resolution is pure filesystem/string matching with no stage awareness; mixing concerns would violate stageman's schema-query focus.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260216-jmy4-DEV-1044-switch-shell-name-resolution | 2026-02-16 | Updated Shared Change Resolution Library decision: `/fab-switch` now sources `resolve-change.sh` for name resolution in its Argument Flow (previously only `preflight.sh` and `fab-status.sh` sourced it) |
| 260215-lqm5-stageman-cli-only | 2026-02-15 | Migrated from `source stageman.sh` to `$STAGEMAN <subcommand>` CLI subprocess calls; `resolve-change.sh` remains sourced (variable-setting pattern); updated design decision from "Accessor Functions" to "CLI Subprocess" |
| 260214-q7f2-reorganize-src | 2026-02-14 | Moved from `_preflight.sh` to `lib/preflight.sh`; updated all internal references from `_stageman.sh`/`_resolve-change.sh` to `lib/stageman.sh`/`lib/resolve-change.sh`; updated path resolution from `../../` to `../../..`; added lib/ subfolder design decision |
| 260213-puow-consolidate-status-reads | 2026-02-14 | Replaced inline `grep \| sed` parsing with stageman accessor calls (`get_progress_map`, `get_checklist`, `get_confidence`); delegated change resolution to `_resolve-change.sh`; added confidence fields to output; renamed `stageman.sh` → `_stageman.sh` |
| 260212-4tw0-migrate-scripts-stageman | 2026-02-12 | Migrated to source stageman.sh: dynamic stage iteration, schema validation via validate_status_file |
| 260212-v5p2-simplify-stages-entry-paths | 2026-02-12 | Updated from 6 to 5 stages, documented stage derivation from active entry |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated progress map to 6 stages |
| 260207-5mjv-preflight-grep-scripts | 2026-02-07 | Created preflight doc — script purpose, output format, validation order, skill integration |
