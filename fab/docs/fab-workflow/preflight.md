# Preflight

**Domain**: fab-workflow

## Overview

The preflight script (`fab/.kit/scripts/fab-preflight.sh`) validates the active change's state and outputs structured YAML for agent consumption. It consolidates repeated validation logic from individual skills into a single reusable script.

## Requirements

### Structured YAML Output

`fab-preflight.sh` outputs a YAML document to stdout containing the active change's resolved state. Fields include:

- `name` — the change folder name (from `fab/current`)
- `change_dir` — path to `fab/changes/{name}/`, relative to `fab/`
- `stage` — current stage (derived from the `active` entry in the progress map)
- `progress` — full progress map (all 5 stages with their status)
- `checklist.generated` — boolean
- `checklist.completed` — integer
- `checklist.total` — integer

Agents consume this output by running the script via Bash and parsing the stdout YAML directly.

### Validation Checks

The script validates in this order, stopping at the first failure:

1. `fab/config.yaml` and `fab/constitution.md` exist (project initialized)
2. `fab/current` exists and is non-empty (active change set)
3. Change directory `fab/changes/{name}/` exists
4. `.status.yaml` exists within the change directory

Each failure exits with code 1 and prints a diagnostic message to stderr.

### No External Dependencies

The script uses only POSIX-standard tools (`grep`, `sed`, `tr`, `cat`) and Bash builtins. No `yq`, `jq`, Python, or other non-standard tools required.

### Idempotent and Read-Only

The script does not modify any files. Safe to run any number of times without side effects.

### Relative Path Resolution

All internal paths resolve relative to the script's own location via `$(dirname "$0")/../..`. Works regardless of the caller's working directory.

### Skill Integration

Skills that perform pre-flight checks (ff, apply, review, archive, continue, clarify) reference `fab-preflight.sh` instead of inline validation. On non-zero exit, the agent stops and surfaces the stderr message. On success, the agent uses the stdout YAML for change context.

Skills exempt from preflight: `init`, `switch`, `status`, `hydrate`, `help`, `new`.

## Design Decisions

<!-- No plan was generated for this change — no design decisions to extract. -->

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260212-v5p2-simplify-stages-entry-paths | 2026-02-12 | Updated from 6 to 5 stages, documented stage derivation from active entry |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated progress map to 6 stages |
| 260207-5mjv-preflight-grep-scripts | 2026-02-07 | Created preflight doc — script purpose, output format, validation order, skill integration |
