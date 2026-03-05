# Intake: Redefine fab-ff and fab-fff Scope

**Change**: 260215-237b-DEV-1027-redefine-ff-fff-scope
**Created**: 2026-02-15
**Status**: Draft

## Origin

> Redefine the meaning of `/fab-ff` and `/fab-fff`. Currently both commands cover the same pipeline scope (intake → hydrate) and differ only in interactivity and confidence gating. The new model differentiates them by scope and gating:
>
> - `/fab-fff` becomes the full pipeline command (spec → hydrate) with no confidence gate, starting right after intake is created. Interactive on review failure.
> - `/fab-ff` becomes the fast-forward-from-spec command (tasks → hydrate), gated on confidence score > 3. Bails immediately on review failure.
>
> One-shot mode. Key decisions reached through discussion — the redefinition resolves the chicken-and-egg problem where `/fab-fff` required a confidence score that only exists after the spec stage has run.

## Why

1. **Chicken-and-egg problem**: The current `/fab-fff` gates on confidence >= 3.0, but confidence scores are only computed during the spec stage. Users must run `/fab-continue` through spec first (or `/fab-clarify`) before `/fab-fff` will even start — defeating the "run everything from scratch" promise.

2. **No scope differentiation**: Both `/fab-ff` and `/fab-fff` cover the same stages (spec → hydrate). The only differences are interactivity and the confidence gate. Users have no way to say "I've done the spec work, just finish the rest" in a single command.

3. **Gating misalignment**: The confidence gate makes more practical sense on a command that starts *after* spec (where the score naturally exists) rather than on a command that generates the spec itself.

## What Changes

### `/fab-fff` — Full pipeline, no gate

- **Minimum prerequisite**: intake exists (spec pending or later)
- **Scope**: current stage → hydrate (skips whatever is already `done`)
- **Callable from**: Any stage at or after intake — picks up from the current stage and runs forward
- **Gate**: None (remove the current confidence gate)
- **Auto-clarify**: Yes, interleaved between planning stages (same as current `/fab-ff`)
- **On review failure**: Interactive rework menu — fix code, revise tasks, revise spec (inherits current `/fab-ff` behavior)
- **Frontloaded questions**: Yes, single batch before generation (same as current `/fab-ff`)
- **Resumable**: Yes, skips stages already `done`

### `/fab-ff` — Fast-forward from spec, gated

- **Minimum prerequisite**: spec `active` or later (spec.md and score already exist)
- **Scope**: current stage → hydrate (skips whatever is already `done`)
- **Callable from**: Any stage at or after spec — picks up from the current stage and runs forward
- **Gate**: Confidence score > 3 (moved from current `/fab-fff`)
- **Auto-clarify**: Minimal — only between tasks generation if tasks aren't done yet
- **On review failure**: Bail immediately with actionable message (inherits current `/fab-fff` behavior)
- **No frontloaded questions**: Spec is already done; tasks generation proceeds directly
- **Resumable**: Yes, skips stages already `done`

### Memory and cross-references

- Update `planning-skills.md`: rewrite `/fab-ff` and `/fab-fff` sections
- Update `execution-skills.md`: pipeline invocation note
- Update `change-lifecycle.md`: full pipeline path reference
- Update `_context.md`: Next Steps table, Skill-Specific Autonomy Levels table, Confidence Scoring gate threshold reference

### Skill files

- Rewrite `fab/.kit/skills/fab-ff.md` with new scope, gate, and bail-on-failure behavior
- Rewrite `fab/.kit/skills/fab-fff.md` with new scope, no gate, and interactive rework menu

## Affected Memory

- `fab-workflow/planning-skills`: (modify) rewrite `/fab-ff` and `/fab-fff` sections, update design decisions
- `fab-workflow/execution-skills`: (modify) update pipeline invocation note
- `fab-workflow/change-lifecycle`: (modify) update full pipeline path description

## Impact

- Skill files: `fab/.kit/skills/fab-ff.md`, `fab/.kit/skills/fab-fff.md`
- Context file: `fab/.kit/skills/_context.md` (Next Steps, Autonomy Levels, Confidence Scoring sections)
- Memory files: `planning-skills.md`, `execution-skills.md`, `change-lifecycle.md`
- No script changes needed — `calc-score.sh --check-gate` is still used, just by a different skill

## Open Questions

(None — design decisions resolved in conversation.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-fff inherits current fab-ff's interactive rework menu | Explicitly decided in conversation — longer pipeline needs course correction ability | S:95 R:70 A:90 D:95 |
| 2 | Certain | fab-ff inherits current fab-fff's bail-on-failure behavior | Explicitly decided — confidence was high, failure is unexpected | S:95 R:70 A:90 D:95 |
| 3 | Certain | fab-ff starts when spec is active (score exists at that point) | Clarified by user — .status.yaml has scores when spec stage is active | S:95 R:80 A:95 D:95 |
| 4 | Certain | Confidence gate threshold stays > 3 (moved to fab-ff) | User specified "greater than 3" | S:95 R:85 A:90 D:95 |
| 5 | Certain | No script changes needed — calc-score.sh reused as-is | Gate logic is identical, just invoked by a different skill file | S:90 R:90 A:95 D:95 |
| 6 | Confident | fab-ff does minimal auto-clarify (only if tasks not yet done) | Reasonable inference — spec is already scored and solid, less need for clarification | S:70 R:80 A:75 D:70 |
| 7 | Confident | Dynamic gate thresholds (bugfix=2.0, feature=3.0, arch=4.0) move to fab-ff | Current fab-fff uses dynamic thresholds via calc-score.sh --check-gate; same mechanism transfers | S:75 R:75 A:80 D:70 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
