# Preflight

Pre-execution validator for fab workflow commands. Checks the health of the current change and emits structured YAML metadata for downstream consumers.

## Sources of Truth

- **Implementation**: `fab/.kit/scripts/_preflight.sh` — main file (distributed with kit)
- **Dev symlink**: `src/preflight/_preflight.sh` → `../../fab/.kit/scripts/_preflight.sh`
- **Dependencies**: `fab/.kit/scripts/stageman.sh` (sourced for schema-driven validation)
- **Schema**: `fab/.kit/schemas/workflow.yaml` — canonical workflow definition

## Usage

```bash
# Check the current active change (reads fab/current)
fab/.kit/scripts/_preflight.sh

# Check a specific change by name (fuzzy, case-insensitive)
fab/.kit/scripts/_preflight.sh my-feature
```

## Validation Pipeline

| Step | Check | Failure |
|------|-------|---------|
| 1 | `config.yaml` and `constitution.md` exist | "fab/ is not initialized" |
| 2 | Change name resolves (from `$1` or `fab/current`) | "No active change" / "No change matches" |
| 3 | Change directory exists under `fab/changes/` | "Change directory not found" |
| 4 | `.status.yaml` exists | "corrupted — .status.yaml not found" |
| 5 | `.status.yaml` passes schema validation (via stageman) | "Status file validation failed" |

## Change Name Resolution

When `$1` is provided (override mode):

1. Collects non-archive folders from `fab/changes/`
2. Case-insensitive exact match → use it
3. Single partial match → use it
4. Multiple partial matches → error with list
5. No matches → error

When no `$1` (default mode):

1. Reads `fab/current`, strips whitespace
2. Uses as change name directly

## Output Format

On success, emits structured YAML to stdout:

```yaml
name: my-feature
change_dir: changes/my-feature
stage: spec
progress:
  brief: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  archive: pending
checklist:
  generated: false
  completed: 0
  total: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 5.0
```

| Field | Source | Default |
|-------|--------|---------|
| `name` | Resolved change name | — |
| `change_dir` | Relative path under `fab/` | — |
| `stage` | First `active` stage, or `archive` if all done | `archive` |
| `progress.*` | Per-stage state from `.status.yaml` | `pending` |
| `checklist.*` | From `.status.yaml` checklist block | `false`/`0` |
| `confidence.*` | From `.status.yaml` confidence block | `0`/`5.0` |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All validations passed, YAML emitted to stdout |
| 1 | Validation failure, error message on stderr |

## Requirements

- Bash 4.0+
- GNU coreutils (grep, sed, awk)
- `_stageman.sh` accessible at `$(dirname "$0")/_stageman.sh`
- `_resolve-change.sh` accessible at `$(dirname "$0")/_resolve-change.sh`

## Testing

```bash
# Quick smoke test
src/preflight/test-simple.sh

# Full test suite
src/preflight/test.sh
```
