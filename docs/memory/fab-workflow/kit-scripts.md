# Kit Scripts Reference

> Deep reference for agents working on the kit's shell scripts. For calling conventions, see `fab/.kit/skills/_scripts.md`.

## Scripts Overview

| Script | Location | Purpose |
|--------|----------|---------|
| `stageman.sh` | `fab/.kit/scripts/lib/stageman.sh` | Stage management — state transitions, `.status.yaml` accessors, history logging |
| `changeman.sh` | `fab/.kit/scripts/lib/changeman.sh` | Change management — create, rename, resolve, switch, list |
| `calc-score.sh` | `fab/.kit/scripts/lib/calc-score.sh` | Confidence scoring — parse Assumptions tables, compute SRAD scores, gate checks |
| `preflight.sh` | `fab/.kit/scripts/lib/preflight.sh` | Pre-flight validation — project init checks, change resolution, structured YAML output |

## Argument Resolution

### `resolve_change_arg` (stageman.sh)

Central resolution function used by all stageman commands. Accepts:

1. **Existing file path** — if `[ -f "$arg" ]`, returned directly
2. **Change identifier** — if no `/` and no `.yaml` suffix, delegates to `changeman.sh resolve` for fuzzy matching, then appends `/.status.yaml`
3. **Path-like non-file** — if contains `/` or ends with `.yaml` but file doesn't exist, errors with "Status file not found"

Does NOT accept bare directory paths. Internal callers pass folder names (valid change identifiers) or `.status.yaml` paths.

### `changeman.sh resolve`

Fuzzy matching against `fab/changes/` folders (excludes `archive/`):

1. Exact match (case-insensitive) → return folder name
2. Single substring match → return folder name
3. Multiple matches → error with comma-separated list
4. No match → error
5. No argument → read `fab/current`, or single-change guess if `fab/current` is empty

Always returns the full folder name (e.g., `260227-yobi-fix-kit-scripts`).

## Stage State Machine

### States

`pending` → `active` → `ready` → `done`

Branch: `active` → `failed` (review only), `failed` → `active` (via `start`)

Reset: `done`/`ready` → `active` (cascades downstream to `pending`)

### Transitions

| Command | From | To | Side Effects |
|---------|------|----|-------------|
| `start` | pending, failed | active | None |
| `advance` | active | ready | None |
| `finish` | active, ready | done | Auto-activates next pending stage |
| `reset` | done, ready | active | Cascades all downstream stages to pending |
| `fail` | active | failed | Review stage only |

### Auto-Activation Chain

`finish` on any stage sets the next stage to `active`. The full chain: intake → spec → tasks → apply → review → hydrate.

## History Logging

### `.history.jsonl` Format

One JSON object per line, appended to `{change_dir}/.history.jsonl`.

**Command event**: `{"ts":"ISO-8601","event":"command","cmd":"fab-ff","args":"optional"}`

**Confidence event**: `{"ts":"ISO-8601","event":"confidence","score":3.5,"delta":"+0.3","trigger":"calc-score"}`

**Review event**: `{"ts":"ISO-8601","event":"review","result":"passed"}` or `{"ts":"ISO-8601","event":"review","result":"failed","rework":"fix-code"}`

### Internal Function Signatures

All history functions accept a resolved `.status.yaml` path and derive the change directory via `dirname`:

- `log_command(status_file, cmd, args?)` — logs command invocation
- `log_confidence(status_file, score, delta, trigger)` — logs confidence change
- `log_review(status_file, result, rework?)` — logs review outcome

## Confidence Scoring

### Formula

```
if unresolved > 0: score = 0.0
else:
  base = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  cover = min(1.0, total_decisions / expected_min)
  score = base * cover
```

Range: 0.0 to 5.0. `expected_min` varies by `{stage, change_type}`.

### Gate Check

`--check-gate` parses the relevant artifact (intake.md or spec.md) and computes the score inline. Read-only — does not write to `.status.yaml`. Returns YAML with `gate: pass|fail`, `score`, `threshold`, `change_type`, and grade counts.

## Design Rationale

### Unified `<change>` Convention

All stageman commands route through `resolve_change_arg` for consistent argument handling. History commands were migrated from a separate `resolve_change_dir` function (which only handled directory paths) to the unified resolver. This ensures change IDs, folder names, and `.status.yaml` paths all work everywhere.

### Separation of Concerns

- **changeman.sh** owns change identity (folders, naming, pointer)
- **stageman.sh** owns stage state (.status.yaml) and history (.history.jsonl)
- **calc-score.sh** owns confidence scoring (SRAD formula, gate checks)
- **preflight.sh** owns validation (project init, change resolution, structured output)

Each script accepts change identifiers at its boundary and resolves internally.

---

*Last updated: 2026-02-28*
