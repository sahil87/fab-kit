# Brief: Fix reset flow to stop at target stage

**Change**: 260213-wo9v-fix-reset-auto-advance
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Fix fab-continue reset flow to stop at the target stage instead of auto-advancing. When running "fab-continue spec", the reset should regenerate spec.md, set spec to done, but NOT set the next stage (tasks) to active. This avoids the state machine problem where tasks becomes active but tasks.md was invalidated. The downstream stages stay pending — user runs /fab-continue again to advance. Also need to handle the "no active stage" gap: when all completed stages are done and next is pending, preflight/status should report the next pending stage as the suggested next action rather than requiring an active stage.

## Why

The reset flow in `fab-continue` auto-advances past the regenerated stage, leaving the next stage as `active` even though its artifact was invalidated. Example: `fab-continue spec` regenerates spec.md, sets `spec: done`, then sets `tasks: active` — but `tasks.md` was invalidated by the spec reset. The user is now stranded at a stage with no valid artifact.

The real-world trigger: a user at the `apply` stage notices a low confidence score (2.4) and wants to reset to spec to regenerate it. After `fab-continue spec`, they end up at `tasks: active` with a stale `tasks.md` and no clean way to proceed.

## What Changes

- **Reset flow in fab-continue**: After regenerating the target planning artifact and marking it `done`, STOP. Do not set the next stage to `active`. Downstream stages remain `pending`.
- **Preflight fallback logic**: When no stage has `active` state and not all stages are `done`, report the first `pending` stage (after the last `done` stage) instead of falling back to `archive`.
- **Status display**: Same fallback fix — show the correct next pending stage when no stage is active.
- **Downstream artifact preservation**: Existing files (e.g., `tasks.md`) are NOT deleted on reset — only their progress state changes to `pending`. This is already the documented behavior; no change needed here.

## Affected Docs

- `fab-workflow/change-lifecycle`: (modify) Document the reset flow behavior — "stops at target, no auto-advance"

## Impact

- `fab/.kit/skills/fab-continue.md` — Reset Flow section (Steps 4-6)
- `fab/.kit/scripts/fab-preflight.sh` — Stage fallback logic (lines 114-125)
- `fab/.kit/scripts/fab-status.sh` — Stage fallback logic (lines 125-135), next command suggestion
- `fab/.kit/schemas/workflow.yaml` — May need to document the "done + pending = next target" state as valid

## Open Questions

None — approach confirmed interactively.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Preflight reports first pending stage when no active stage exists | Natural extension of the "first active" rule — when there's no active, the first pending after done stages is the logical next target. Alternatives (new field, error state) add complexity without benefit. |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.
