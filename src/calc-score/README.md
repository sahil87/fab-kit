# Confidence Score Calculator (_calc-score.sh)

Computes confidence scores from `## Assumptions` tables in `brief.md` and `spec.md`. Scans for SRAD grade counts (Certain, Confident, Tentative), applies the confidence formula, writes the updated confidence block to `.status.yaml`, and emits YAML with delta to stdout.

## Sources of Truth

- **Implementation**: `fab/.kit/scripts/_calc-score.sh` — main file (distributed with kit)
- **Dev symlink**: `src/calc-score/_calc-score.sh` → `../../fab/.kit/scripts/_calc-score.sh`

## Usage

```bash
_calc-score.sh <change-dir>
```

Where `<change-dir>` is the path to a change directory (e.g., `fab/changes/260214-mgh5-calc-score-dev-setup`).

The directory MUST contain `spec.md`. `brief.md` is optional — if present, its Assumptions table is also scanned.

## API Reference

| Field | Value |
|-------|-------|
| **Arguments** | `<change-dir>` — path to change directory (required) |
| **Output** | YAML confidence block to stdout (see format below) |
| **Side effects** | Replaces `confidence:` block in `<change-dir>/.status.yaml` |
| **Exit 0** | Success — score computed and written |
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
| No arguments | `Usage: _calc-score.sh <change-dir>` |
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
src/calc-score/test-simple.sh

# Comprehensive suite
src/calc-score/test.sh
```

## Changelog

### 1.0.0 (2026-02-14)

- Initial dev folder setup
- Symlink to `fab/.kit/scripts/_calc-score.sh`
- Smoke test (`test-simple.sh`) and comprehensive test suite (`test.sh`)
