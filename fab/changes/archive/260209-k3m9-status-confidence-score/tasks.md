# Tasks: Show confidence score in fab-status

**Change**: 260209-k3m9-status-confidence-score
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Implementation

- [x] T001 Add confidence field parsing to `fab/.kit/scripts/fab-status.sh` — parse `score`, `certain`, `confident`, `tentative`, and `unresolved` from the `confidence:` block in `.status.yaml`, defaulting to empty when the block is missing
- [x] T002 Add confidence line rendering to `fab/.kit/scripts/fab-status.sh` — render `Confidence: {score}/5.0 ({N} certain, {N} confident, {N} tentative)` after Checklist and before Next, appending `, {N} unresolved` only when unresolved > 0, and showing `Confidence: not yet scored` when the confidence block is missing

## Phase 2: Documentation

- [x] T003 [P] Update `fab/.kit/skills/fab-status.md` — add the Confidence line to the output format description, documenting all three display variants (normal, with unresolved, not yet scored)
- [x] T004 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` — mention the Confidence line in the `/fab-status` section description

---

## Execution Order

- T001 blocks T002 (parsing must exist before rendering)
- T003 and T004 are independent documentation tasks, can run in parallel after T002
