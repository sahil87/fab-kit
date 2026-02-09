# Proposal: Show confidence score in fab-status

**Change**: 260209-k3m9-status-confidence-score
**Created**: 2026-02-09
**Status**: Draft

## Why

`/fab-status` is the at-a-glance dashboard for a change, yet it omits the confidence score — a key signal for deciding whether to run `/fab-fff`, `/fab-clarify`, or `/fab-discuss`. The data already lives in `.status.yaml`; the script just never reads or renders it.

## What Changes

- Parse `confidence.score`, `confidence.certain`, `confidence.confident`, `confidence.tentative`, and `confidence.unresolved` from `.status.yaml` in `fab-status.sh`
- Render a new `Confidence:` line between Checklist and Next in the output
- Display format: `Confidence: {score}/5.0 ({N} certain, {N} confident, {N} tentative)` — with `unresolved` count appended only when > 0
- When the confidence block is missing from `.status.yaml`, display: `Confidence: not yet scored`
- Update the skill definition (`fab/.kit/skills/fab-status.md`) output format documentation
- Update workflow docs (`fab/docs/fab-workflow/change-lifecycle.md`) output example if present

## Affected Docs

### New Docs
(none)

### Modified Docs
- `fab-workflow/change-lifecycle`: update the `/fab-status` output example to include the Confidence line

### Removed Docs
(none)

## Impact

- `fab/.kit/scripts/fab-status.sh` — ~10-15 lines added for parsing + rendering
- `fab/.kit/skills/fab-status.md` — output format section updated
- `fab/docs/fab-workflow/change-lifecycle.md` — output example updated
- No behavioral changes to any other skill; purely additive

## Open Questions

(none — all decisions resolved during discussion)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Place confidence line after Checklist, before Next | Logically groups "health of this change" info together |
| 2 | Confident | Show `unresolved` count only when > 0 | Keeps common case clean; surfaces the problem when it matters |
