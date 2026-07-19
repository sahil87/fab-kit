---
type: memory
description: "`lib/preflight.sh` script — validation, accessor-based architecture, structured YAML output, skill integration"
---
# Preflight

**Domain**: pipeline

## Overview

The preflight script (`src/kit/scripts/lib/preflight.sh`) validates the active change's state and outputs structured YAML for agent consumption. It consolidates repeated validation logic from individual skills into a single reusable script. Preflight is purely validation + structured output — it has no logging side-effects.

## Requirements

### Structured YAML Output

`lib/preflight.sh` outputs a YAML document to stdout containing the active change's resolved state. Fields include:

- `name` — the change folder name (resolved via `lib/changeman.sh resolve`)
- `change_dir` — path to `fab/changes/{name}/`, relative to `fab/`
- `stage` — routing stage: what `/fab-continue` will produce next (derived via `get_current_stage` from `lib/statusman.sh`)
- `display_stage` — display stage: "where you are" in the pipeline (derived via `get_display_stage` from `lib/statusman.sh`). Five-tier walk: first `active`, else first `failed` (parked review/review-pr failures surface instead of falling through to the last done stage) (dkn3), else first `ready`, else last `done`/`skipped`, else first `pending` (`intake` if nothing started) — see [change-lifecycle.md](/pipeline/change-lifecycle.md) "Deriving display stage"
- `display_state` — the state of the display stage: `active`, `ready`, `done`, `failed`, `pending`, or `skipped`
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

1. `fab/project/config.yaml` and `fab/project/constitution.md` exist (project initialized)
1b. Sync staleness check (non-blocking) — compares `$(fab kit-path)/VERSION` against `fab_version` in `fab/project/config.yaml` (read via the shared `internal/config` accessor) (ye8r); emits stderr warning if they differ, silently skips if either is unreadable. Does NOT exit or alter stdout
2. Change name resolves (via `lib/changeman.sh resolve` — from `$1` override or `.fab-status.yaml`)
3. Change directory `fab/changes/{name}/` exists
4. `.status.yaml` exists within the change directory
5. `.status.yaml` passes schema validation via `validate_status_file()` from `lib/statusman.sh` (catches invalid states, missing stages, multiple active stages)

Each failure exits with code 1 and prints a diagnostic message to stderr. The staleness check (1b) is the exception — it is advisory only and never blocks execution.

### Accessor-Based Architecture

The script invokes `lib/changeman.sh` and `lib/statusman.sh` via CLI subprocess calls, delegating all resolution and `.status.yaml` parsing to their respective subcommands:

- **Change resolution**: `$CHANGEMAN resolve [override]` handles both default mode (reads `.fab-status.yaml` symlink) and override mode (case-insensitive substring matching against `fab/changes/`). Returns resolved folder name to stdout; errors to stderr.
- **Progress extraction**: `$STATUSMAN progress-map` returns `stage:state` pairs, consumed via `while IFS=: read -r`
- **Stage derivation (routing)**: `$STATUSMAN current-stage` — returns the next stage to work on (three-tier fallback: first active, first pending after last done, review-pr if all done/skipped)
- **Stage derivation (display)**: `$STATUSMAN display-stage` — returns `stage:state` for "where you are" (five-tier walk: first active, else first failed, else first ready, else last done/skipped, else first pending). Used for user-facing display in `/fab-status` and `/fab-switch`
- **Checklist fields**: `$STATUSMAN checklist` returns `generated`, `completed`, `total` with defaults
- **Confidence fields**: `$STATUSMAN confidence` returns `certain`, `confident`, `tentative`, `unresolved`, `score` with defaults
- **Schema validation**: `$STATUSMAN validate-status-file` for structural correctness

No inline `grep | sed` parsing of `.status.yaml` — all field extraction goes through statusman CLI subcommands.

### No External Dependencies

The script uses only POSIX-standard tools (`grep`, `sed`, `tr`, `cat`) and Bash builtins. It invokes `lib/changeman.sh` and `lib/statusman.sh` as CLI subprocesses — both require `yq` v4, but preflight itself has no direct `yq` dependency.

### Idempotent and Read-Only

The script does not modify any files. Safe to run any number of times without side effects.

### Relative Path Resolution

All internal paths resolve relative to the script's own location via `$(dirname "$0")/../../..` (three levels up from `scripts/lib/` to the `fab/` root). Works regardless of the caller's working directory.

### Skill Integration

Skills that perform pre-flight checks (ff, apply, review, archive, continue, clarify) reference `lib/preflight.sh` instead of inline validation. On non-zero exit, the agent stops and surfaces the stderr message. On success, the agent uses the stdout YAML for change context. After preflight, skills log the command invocation via a direct `fab log command "<skill>" "<id>"` call (per `_preamble.md` §2 step 4) — best-effort: the command always exits 0 given valid usage, so no shell guard is needed (260612-ye8r).

Skills exempt from preflight: `setup`, `new`, `switch`, `status`, `discuss`, `help`. Exempt skills call `fab log command` directly in their own skill files for best-effort logging.

## Design Decisions

### CLI Subprocess Over Source Import
**Decision**: `lib/preflight.sh` invokes `lib/statusman.sh` via CLI subprocess calls (`$STATUSMAN progress-map`, `$STATUSMAN checklist`, etc.) instead of sourcing it.
**Why**: Decouples preflight from statusman's internal function signatures. Enables future replacement of `statusman.sh` with a compiled binary (e.g., Rust) without modifying callers. The CLI interface is the stable contract.
**Rejected**: Continuing to source `statusman.sh` — tight coupling to internal function names; not compatible with a binary replacement.
*Updated by*: 260215-lqm5-statusman-cli-only (formerly "Accessor Functions Over Inline Parsing")

### lib/ Subfolder for Internal Scripts
**Decision**: All internal scripts (`preflight.sh`, `statusman.sh`, `changeman.sh`, `calc-score.sh`, `sync-workspace.sh`) live in `src/kit/scripts/lib/` without underscore prefix, replacing the previous `_`-prefixed convention in the parent `scripts/` directory.
**Why**: The `lib/` subfolder provides a clearer structural boundary between internal plumbing and user-facing scripts than naming conventions alone. All internal scripts are co-located, making the dependency graph explicit.
**Rejected**: Retaining underscore prefix — naming conventions are less discoverable than directory structure.
*Introduced by*: 260214-q7f2-reorganize-src

### Change Resolution via changeman CLI
**Decision**: Change name resolution (fuzzy matching against `fab/changes/`) is a `resolve` subcommand of `lib/changeman.sh`, invoked as a CLI subprocess by `lib/preflight.sh`, batch scripts, and `/fab-switch` (via `changeman.sh switch` which calls `resolve` internally).
**Why**: Resolution is a change lifecycle operation — it belongs with other change operations in changeman rather than as a standalone sourced library. The CLI subprocess pattern (`$CHANGEMAN resolve <override>`) is consistent with statusman's interface and enables future Rust rewrite. Error messages remain generic — callers add context-appropriate guidance.
**Rejected**: Keeping as a standalone sourced library (`resolve-change.sh`) — the variable-setting pattern (`RESOLVED_CHANGE_NAME`) was inconsistent with the CLI subprocess convention used by all other lib/ scripts. Consolidating into statusman — change resolution is filesystem/string matching with no stage awareness.
*Updated by*: 260216-oinh-DEV-1045-fold-resolve-into-changeman (formerly "Shared Change Resolution Library" using `resolve-change.sh`)
