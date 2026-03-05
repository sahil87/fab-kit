# Proposal: Add fab-fff Full-Pipeline Command with Confidence Gating

**Change**: 260208-k3m7-add-fab-fff
**Created**: 2026-02-08
**Status**: Draft

## Why

The current `fab-ff --auto` mode has a philosophical problem: it guesses everything and marks guesses with `<!-- auto-guess -->` markers, which then need `/fab-clarify` before `/fab-apply` anyway (soft gate). In practice, `--auto` defers human interaction rather than eliminating it. A better model: get confidence high enough first (via `/fab-clarify` if needed), then run the full pipeline autonomously.

## What Changes

- **New `/fab-fff` command**: A thin wrapper that chains `fab-ff` → `fab-apply` → `fab-review` → `fab-archive`. Gated on confidence score >= 3.0. Same exact `fab-ff` behavior internally — no behavioral differences. Bails on review failure.
- **Confidence scoring in `.status.yaml`**: New `confidence` field tracking all four SRAD grades (certain, confident, tentative, unresolved) and a derived score. Starts at 5.0, subtracts penalties: Certain -0, Confident -0.1, Tentative -1, Unresolved → instant 0. Clamped to floor of 0. Computed during `/fab-new` and recomputed on every `/fab-clarify` pass.
- **Remove `fab-ff --auto` mode**: The `--auto` flag and full-auto behavior are removed from `/fab-ff`. Default mode (frontload questions, bail on blockers) remains unchanged. The replacement workflow is: raise confidence via `/fab-clarify`, then `/fab-fff`.
- **Remove `<!-- auto-guess -->` marker system**: Since `fab-fff` only runs at high confidence and `fab-ff --auto` is removed, no skill produces auto-guess markers anymore. The `<!-- assumed: ... -->` (Tentative) markers remain.
- **Remove auto-guess soft gate from `/fab-apply`**: `/fab-apply` no longer needs to scan for `<!-- auto-guess -->` markers since they won't exist.

## Affected Docs

### New Docs
- (none — fab-fff will be documented via updates to existing docs)

### Modified Docs
- `fab-workflow/planning-skills.md`: Add `/fab-fff` documentation, remove `/fab-ff --auto` documentation
- `fab-workflow/change-lifecycle.md`: Add the fab-fff path (proposal → archive in one shot, gated on confidence)
- `fab-workflow/clarify.md`: Remove `<!-- auto-guess -->` references from auto mode, add confidence score recomputation behavior
- `fab-workflow/execution-skills.md`: Remove auto-guess soft gate from `/fab-apply` documentation

### Removed Docs
(none)

## Impact

### Skill files (primary)
- `fab/.kit/skills/fab-fff.md` — **New**: thin wrapper — confidence gate + fab-ff + fab-apply + fab-review + fab-archive
- `fab/.kit/skills/fab-ff.md` — **Modified**: remove `--auto` mode, all auto-guess references, full-auto output examples
- `fab/.kit/skills/fab-new.md` — **Modified**: add confidence score computation at proposal completion (Step 8)
- `fab/.kit/skills/fab-continue.md` — **Modified**: add confidence score recomputation after each artifact generation
- `fab/.kit/skills/fab-clarify.md` — **Modified**: add confidence score recomputation after each session, remove auto-guess scanning references
- `fab/.kit/skills/fab-apply.md` — **Modified**: remove auto-guess soft gate
- `fab/.kit/skills/_context.md` — **Modified**: update SRAD skill autonomy table (remove fab-ff --auto column, add fab-fff column), update artifact markers table (remove auto-guess row), update Next Steps table, add confidence scoring formula

### Skill registration
- `.claude/skills/fab-fff/prompt.md` — **New**: Claude Code skill registration for `/fab-fff`

### Scripts
- `fab/.kit/scripts/fab-preflight.sh` — **Modified**: extract and emit confidence fields from `.status.yaml`

### Templates
- `fab/.kit/templates/status.yaml` — **New**: stamp-out template for `.status.yaml` with confidence block at zero values (instantiation convenience; schema definition lives in `_context.md`)
<!-- clarified: standalone template for instantiation, schema definition in _context.md -->

## Confidence Scoring Design

### `.status.yaml` structure

```yaml
confidence:
  certain: 12
  confident: 3
  tentative: 2
  unresolved: 0
  score: 2.7    # max(0, 5 - 0.1*confident - 1*tentative), or 0 if unresolved > 0
```

### Formula

```
if unresolved > 0:
  score = 0
else:
  score = max(0, 5.0 - 0.1 * confident - 1.0 * tentative)
```

- **Range**: 0.0 to 5.0
- **5.0**: All decisions are Certain — maximum confidence
- **0.0**: Any Unresolved decision, OR 5+ Tentative decisions
- Certain contributes 0 penalty (deterministic, no ambiguity)
- Confident contributes 0.1 penalty (minor — strong signal, one obvious interpretation)
- Tentative contributes 1.0 penalty (meaningful — reasonable guess but multiple valid options)
- Unresolved is a hard zero (cannot run autonomously with unresolved decisions)

### Gate

`/fab-fff` requires `confidence.score >= 3.0`. This allows at most 2 Tentative decisions (with some Confident erosion). If the score is below the threshold:

> `Confidence is {score} (need >= 3.0). Run /fab-clarify to resolve tentative/unresolved decisions, then retry.`

### Lifecycle

- **Computed**: `/fab-new` calculates initial confidence after generating the proposal
- **Recomputed**: `/fab-continue` and `/fab-clarify` recalculate after each invocation (new assumptions from artifact generation update the counts; resolving a Tentative removes its -1 penalty; resolving an Unresolved lifts the hard zero)
- **Not recomputed by**: `/fab-ff` and `/fab-fff` — autonomous skills do not update the score. The gate check uses the score from the last manual step.
- **Consumed**: `/fab-fff` reads the score as a gate check before proceeding
<!-- clarified: confidence recomputed by manual skills only (fab-new, fab-continue, fab-clarify), not autonomous ones (fab-ff, fab-fff) -->

## Clarifications

### Session 2026-02-08

- **Q**: Should confidence fields be defined in a standalone `status.yaml` template, inline in `_context.md`, or both?
  **A**: Both — schema definition in `_context.md` (1 shared doc), standalone `fab/.kit/templates/status.yaml` as stamp-out template for instantiation (1 per change)
- **Q**: Should confidence be recomputed by all planning skills or only manual ones?
  **A**: Manual skills only (`fab-new`, `fab-continue`, `fab-clarify`). Autonomous skills (`fab-ff`, `fab-fff`) do not recompute — the gate check uses the score from the last manual step.
- **Q**: Does the Affected Docs list need a separate entry for `/fab-continue` recomputation in `planning-skills.md`?
  **A**: Accepted recommendation: current list is sufficient — `planning-skills.md` already covers `/fab-continue` changes under its existing entry.

## Open Questions

(none — all design decisions resolved in discussion)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `<!-- assumed: ... -->` markers remain in the system | Only `<!-- auto-guess -->` is removed; Tentative markers are still useful for `/fab-clarify` |
| 2 | Confident | `/fab-fff` is inherently resumable by checking progress map | It chains fab-ff (resumable) → fab-apply (resumable) → fab-review → fab-archive |
| 3 | Confident | `/fab-clarify` auto mode stays (used internally by both `/fab-ff` and `/fab-fff`) | Only the `--auto` flag on `/fab-ff` is removed; clarify's auto mode is a separate internal mechanism |

3 assumptions made (3 confident, 0 tentative).
