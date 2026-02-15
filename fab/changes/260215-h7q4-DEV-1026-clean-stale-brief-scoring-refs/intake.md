# Intake: Clean Stale Brief Scoring Refs

**Change**: 260215-h7q4-DEV-1026-clean-stale-brief-scoring-refs
**Linear**: [DEV-1026](https://linear.app/weaver-ai/issue/DEV-1026/clean-stale-brief-scanning-refs-in-calc-score-docs-and-tests)
**Created**: 2026-02-15
**Status**: Draft

## Origin

> User identified stale references to `brief.md` scanning in calc-score documentation and tests. The original design (change 260213-w8p3-extract-fab-score) planned to scan both `brief.md` and `spec.md` Assumptions tables, but the implementation was scoped to spec-only. The code (`calc-score.sh`) is correct — it only scans `spec.md`. However, three documentation/test locations were never cleaned up and still describe the old brief+spec behavior.

## Why

The calc-score README, the SRAD spec, and the calc-score test suite all claim that `calc-score.sh` scans `brief.md` (or `intake.md`) Assumptions in addition to `spec.md`. This is wrong — the script only scans `spec.md`, and the memory file (`change-lifecycle.md:59`) already documents the spec-only behavior as authoritative.

If left unfixed:
- New contributors reading the README will expect brief/intake scanning behavior that doesn't exist
- The srad spec creates a false expectation that intake assumptions affect confidence scoring
- The test suite asserts behavior (brief.md grade counting) that the script intentionally doesn't implement, and would fail if the test fixtures were ever run against the actual script behavior (currently the test passes because the script happens to scan any `.md` file with an Assumptions table via glob, but this is incidental not intentional)

## What Changes

### 1. `src/lib/calc-score/README.md` — Remove brief.md claim

Line 18 currently says:
```
The directory MUST contain `spec.md`. `brief.md` is optional — if present, its Assumptions table is also scanned.
```

Change to:
```
The directory MUST contain `spec.md` with an `## Assumptions` table.
```

Also update line 1's description if it references brief.

### 2. `docs/specs/srad.md` — Fix Confidence Lifecycle table

Line 205 currently says:
```
| Computation | `/fab-continue` (spec stage) | `_calc-score.sh` scans intake + spec, writes to `.status.yaml` |
```

Change to:
```
| Computation | `/fab-continue` (spec stage) | `calc-score.sh` scans spec, writes to `.status.yaml` |
```

(Also fixes the stale `_calc-score.sh` name — the underscore prefix was dropped.)

### 3. `src/lib/calc-score/test.sh` — Remove stale brief.md test

Lines 143-168 test "Combined grades from brief and spec" — creating a `brief.md` with a Tentative assumption and a `spec.md` with a Confident assumption, then asserting both are counted. This test should be removed since the script is spec-only by design.

## Affected Memory

- `fab-workflow/change-lifecycle`: (no change) Already correct — documents spec-only scoring at line 59

## Impact

- `src/lib/calc-score/README.md` — documentation fix
- `docs/specs/srad.md` — spec correction
- `src/lib/calc-score/test.sh` — test removal
- No code changes — `calc-score.sh` is already correct

## Open Questions

(None — the scope is clear from the prior conversation and verified by checking each location.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Script only scans spec.md | Verified: `calc-score.sh` has zero references to "brief"; memory doc confirms spec-only | S:95 R:95 A:95 D:95 |
| 2 | Certain | Scope limited to these 3 files | These are the only stale locations referencing brief-scanning behavior for calc-score specifically | S:90 R:90 A:90 D:90 |
| 3 | Confident | Remove test rather than update it | Test asserts behavior (brief scanning) that was intentionally excluded from the design; no replacement test needed since spec-only scanning is covered by existing tests | S:80 R:85 A:75 D:80 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
