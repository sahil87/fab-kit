# Confidence Score Calculator (calc-score.sh)

Computes confidence scores from the `## Assumptions` table in `spec.md`. Scans for SRAD grade counts (Certain, Confident, Tentative), applies the confidence formula, writes the updated confidence block to `.status.yaml`, and emits YAML with delta to stdout.

## Sources of Truth

- **Implementation**: `fab/.kit/scripts/lib/calc-score.sh` — main file (distributed with kit)
- **Dev symlink**: `src/lib/calc-score/calc-score.sh` → `../../../fab/.kit/scripts/lib/calc-score.sh`

## Usage

```bash
calc-score.sh [--stage <stage>] <change>
calc-score.sh --check-gate [--stage <stage>] <change>
```

Where `<change>` is a change identifier (4-char ID, folder name, or substring). With `--stage intake`, parses `intake.md` and persists with `indicative: true`. Without `--stage`, parses `spec.md` and clears the `indicative` flag. `--check-gate` mode is read-only (no `.status.yaml` writes).

## API Reference

| Field | Value |
|-------|-------|
| **Arguments** | `[--stage <stage>] <change>` — optional stage, change identifier (required) |
| **Output** | YAML confidence block to stdout (see format below) |
| **Side effects** | Normal mode: replaces `confidence:` block in `.status.yaml` (with `indicative: true` for intake, cleared for spec). Gate mode: read-only |
| **Exit 0** | Success — score computed and written (or gate result emitted) |
| **Exit 1** | Error — message to stderr |

### Output Format

```yaml
confidence:
  certain: 5
  confident: 2
  tentative: 1
  unresolved: 0
  score: 3.4
  delta: -1.6
```

### Error Conditions

| Condition | stderr message |
|-----------|---------------|
| No arguments | `Usage: calc-score.sh <change-dir>` |
| Directory not found | `Change directory not found: <path>` |
| No `spec.md` | `spec.md required for scoring` |

### Score Formula

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
```

### Carry-Forward

Implicit Certain counts are preserved from the previous `.status.yaml`. If the previous `certain` count was 5 and 0 Certain grades appear in Assumptions tables, all 5 are carried forward.

## Requirements

- Bash 4.0+
- GNU coreutils (grep, sed, awk)
- No external YAML parsers required

## Testing

```bash
# Quick smoke test
src/lib/calc-score/test-simple.sh

# Comprehensive suite
src/lib/calc-score/test.sh
```

## Changelog

### 1.0.0 (2026-02-14)

- Initial dev folder setup
- Symlink to `fab/.kit/scripts/lib/calc-score.sh`
- Smoke test (`test-simple.sh`) and comprehensive test suite (`test.sh`)
