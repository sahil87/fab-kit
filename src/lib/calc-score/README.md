# Confidence Score Calculator (calc-score.sh)

Computes confidence scores from `## Assumptions` tables in `brief.md` and `spec.md`. Supports two scoring modes:
- `legacy` (default): classic integer penalty counts
- `fuzzy`: weighted SRAD composite + fuzzy memberships for effective penalties

The script writes the updated `confidence` block to `.status.yaml` and emits score, scoring metadata, and gate metadata to stdout.

## Sources of Truth

- **Implementation**: `fab/.kit/scripts/lib/calc-score.sh` — main file (distributed with kit)
- **Dev symlink**: `src/lib/calc-score/calc-score.sh` → `../../../fab/.kit/scripts/lib/calc-score.sh`

## Usage

```bash
calc-score.sh <change-dir>
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
scoring:
  mode: legacy
  effective_confident: 2.00
  effective_tentative: 1.00
  weight_confident: 0.30
  weight_tentative: 1.00
  weights_source: baseline
  historical_samples: 18
  sparse_fallback: false
gate:
  change_type: feature
  threshold: 3.0
  passes_fff: true
  change_type_inferred: true
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
  score = max(0.0, 5.0 - weight_confident * effective_confident - weight_tentative * effective_tentative)
```

Default weights are `0.3` and `1.0`. In fuzzy mode, custom calibrated weights are honored only when historical sample count is not sparse.

### Modes

- **legacy**: `effective_confident = confident`, `effective_tentative = tentative`
- **fuzzy**: each assumption can contribute fractional effective penalties via fuzzy memberships

For fuzzy mode, the script reads optional SRAD metadata in assumption row text:
- `S=<0-100>, R=<0-100>, A=<0-100>, D=<0-100>`

If metadata is missing, grade-based defaults are used:
- `Confident -> composite 60`
- `Tentative -> composite 30`
- `Certain -> composite 90` (zero penalty contribution)

Composite formula (confirmed SRAD weighting):

```
composite = 0.2*S + 0.3*R + 0.3*A + 0.2*D
```

### Gate Metadata

The script emits type-aware threshold metadata for `/fab-fff` consumers:

- legacy mode threshold: `3.0`
- fuzzy mode thresholds:
  - `bugfix`: `2.7`
  - `refactor`: `3.0`
  - `feature`: `3.3`
  - `architecture`: `3.6`

Unknown change types default to `feature`.

### Carry-Forward

Implicit Certain counts are preserved from the previous `.status.yaml`. If the previous `certain` count was 5 and 0 Certain grades appear in Assumptions tables, all 5 are carried forward.

### Optional Environment Variables

- `FAB_SCORE_MODE` — `legacy` or `fuzzy`
- `FAB_CHANGE_TYPE` — `bugfix`, `feature`, `refactor`, `architecture`
- `FAB_WEIGHT_CONFIDENT` — non-negative float (fuzzy mode only, non-sparse history)
- `FAB_WEIGHT_TENTATIVE` — non-negative float (fuzzy mode only, non-sparse history)
- `FAB_CALIBRATION_MIN_SAMPLES` — minimum archived sample count to allow calibrated weights (default `20`)

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

### 1.1.0 (2026-02-14)

- Added scoring modes (`legacy`, `fuzzy`)
- Added weighted SRAD composite support (`S/R/A/D = 0.2/0.3/0.3/0.2`)
- Added fuzzy effective penalty metadata output
- Added type-aware gate threshold metadata output with legacy fallback
- Added sparse-history fallback for calibrated weights

### 1.0.0 (2026-02-14)

- Initial dev folder setup
- Symlink to `fab/.kit/scripts/lib/calc-score.sh`
- Smoke test (`test-simple.sh`) and comprehensive test suite (`test.sh`)
