# Intake: Status Indicative Confidence

**Change**: 260227-czs0-status-indicative-confidence
**Created**: 2026-02-27
**Status**: Draft

## Origin

> fab-status shows indicative confidence score when in intake stage — computes on the fly via calc-score.sh --check-gate, and shows persisted actual score when in spec stage or later. Also adds assumption counts to calc-score.sh --check-gate output for intake.

Initiated from a `/fab-discuss` session. The discussion explored whether both scores should always be persisted and shown, but concluded that the on-the-fly approach for indicative scores is cleaner — intake assumptions go stale after spec generation, so persisting them adds complexity with no benefit.

## Why

1. **Problem**: When a change is at the intake stage, `/fab-status` shows `Confidence: not yet scored` — the user has no visibility into whether their intake is strong enough to pass the `/fab-ff` intake gate (>= 3.0). They have to run `/fab-ff` and watch it bail, or mentally estimate from the assumptions table.

2. **Consequence**: Without the indicative score, users waste a `/fab-ff` invocation to discover their intake needs `/fab-clarify` first. This is friction in the workflow loop.

3. **Approach**: Compute the indicative score on the fly in `/fab-status` (using `calc-score.sh --check-gate --stage intake`, which is already read-only). For spec stage and later, read the persisted actual score from `.status.yaml` as before. This requires no new persistence, no changes to `/fab-new`, and stays consistent with the existing architecture where indicative scores are ephemeral and actual scores are persisted.

## What Changes

### 1. `calc-score.sh` — add counts to `--check-gate` output

The `--check-gate` mode for intake already computes `local_certain`, `local_confident`, `local_tentative`, `local_unresolved` internally but doesn't emit them. Add these four fields to the YAML output:

```yaml
gate: pass
score: 4.2
threshold: 3.0
change_type: feat
certain: 3
confident: 1
tentative: 0
unresolved: 0
```

This applies to the intake branch of `--check-gate` only. The spec branch reads from `.status.yaml` where counts are already persisted.

File: `fab/.kit/scripts/lib/calc-score.sh` (lines ~182-191)

### 2. `fab-status` skill — stage-aware confidence display

Replace the single confidence display bullet with three cases:

- **Intake stage**: Run `calc-score.sh --check-gate --stage intake <change-dir>` and display: `Indicative confidence: {score} (fab-ff gate: {threshold}) — {total} assumptions ({N} certain, {N} confident, {N} tentative)`. Appends `, {N} unresolved` only when unresolved > 0.
- **Spec stage or later**: Read persisted confidence from `.status.yaml` and display: `Confidence: {score} of 5.0 ({N} certain, {N} confident, {N} tentative)`. Appends `, {N} unresolved` only when unresolved > 0.
- **No data and not intake**: Shows `Confidence: not yet scored`.

Files: `fab/.kit/skills/fab-status.md` and `.claude/skills/fab-status/SKILL.md`

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update fab-status behavior to document the stage-aware confidence display

## Impact

- `fab/.kit/scripts/lib/calc-score.sh` — output format change for `--check-gate` mode (additive, no breaking change)
- `fab/.kit/skills/fab-status.md` — skill behavior update
- `.claude/skills/fab-status/SKILL.md` — mirrored skill wrapper update
- No impact on `/fab-ff`, `/fab-continue`, or `calc-score.sh` normal mode — they don't parse the `--check-gate` output for counts

## Open Questions

None — all questions were resolved during the `/fab-discuss` session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Indicative scores are ephemeral, not persisted | Discussed — user confirmed: "indicative score aren't persisted. Actual scores are." | S:95 R:90 A:95 D:95 |
| 2 | Certain | calc-score.sh --check-gate is the right mechanism for on-the-fly computation | Discussed — it's already read-only (no .status.yaml writes in --check-gate mode) | S:90 R:95 A:90 D:90 |
| 3 | Certain | Display format includes gate threshold for indicative | Discussed — user approved format showing "(fab-ff gate: {threshold})" | S:90 R:90 A:85 D:90 |
| 4 | Certain | Adding counts to --check-gate output is additive/non-breaking | Existing consumers (fab-ff) parse score/gate/threshold; extra fields are ignored by YAML parsers | S:85 R:95 A:90 D:95 |
| 5 | Confident | Spec branch of --check-gate doesn't need count additions | Spec counts are already in .status.yaml; fab-status reads them directly | S:80 R:90 A:85 D:85 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
