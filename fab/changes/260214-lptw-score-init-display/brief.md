# Brief: Confidence score initial value and display format

**Change**: 260214-lptw-score-init-display
**Created**: 2026-02-14
**Status**: Draft

## Origin

> When starting a change, start with the score 0 instead of 5. Also, whenever displaying the score, make it obvious what a number like 2.4 means. It should be "2.4 of 5.0", so it is obvious that the scoring is out of 5.

## Why

The current initial confidence score of 5.0 is misleading — a brand-new change with zero scored decisions appears to have perfect confidence. Starting at 0.0 better reflects reality: no decisions have been evaluated yet. Additionally, displaying the score as bare "2.4" or "2.4/5.0" doesn't immediately communicate the scale. "2.4 of 5.0" is self-documenting.

## What Changes

- **Modify** `fab/.kit/templates/status.yaml` — change initial `score: 5.0` to `score: 0.0`
- **Modify** `fab/.kit/scripts/_calc-score.sh` — change `prev_score` fallback from `"5.0"` to `"0.0"`
- **Modify** `fab/.kit/skills/fab-status.md` — change confidence display format from `{score}/5.0` to `{score} of 5.0`
- **Modify** `fab/.kit/skills/fab-fff.md` — change gate display from `Confidence is {score} (need >= 3.0)` to use "of 5.0" format
- **Modify** `fab/.kit/skills/_context.md` — update documentation references: template default description, and any score display examples

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Update SRAD confidence scoring initial value documentation
- `fab-workflow/change-lifecycle`: (modify) Update confidence display format in `/fab-status` description
- `fab-workflow/templates`: (modify) Update status.yaml template default for score

## Impact

- `fab/.kit/templates/status.yaml` — template change affects all future changes
- `fab/.kit/scripts/_calc-score.sh` — fallback value change; no formula change
- `fab/.kit/skills/fab-status.md` — display format instruction change
- `fab/.kit/skills/fab-fff.md` — gate message format change
- `fab/.kit/skills/_context.md` — documentation update
- Existing `.status.yaml` files in active changes are unaffected (they already have their own score values)

## Open Questions

None — the changes are explicit and scoped.
