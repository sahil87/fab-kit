# Tasks: Streamline Planning Stage Dispatch

**Change**: 260227-ijql-streamline-planning-dispatch
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Template Fix

- [x] T001 Change `intake: active` → `intake: pending` in `fab/.kit/templates/status.yaml`

## Phase 2: Core Skill Changes

- [x] T002 Update `fab/.kit/skills/fab-new.md` — add `stageman.sh advance` call after intake generation so intake ends as `ready`
- [x] T003 Rewrite dispatch table and remove single-dispatch rule in `fab/.kit/skills/fab-continue.md` — consolidated planning dispatch where `ready` stages finish + generate next + advance to `ready` in one invocation

## Phase 3: Spec Updates

- [x] T004 [P] Update `/fab-continue` section and Next Steps table in `docs/specs/skills.md` to reflect consolidated dispatch
- [x] T005 [P] Update `/fab-new` section in `docs/specs/skills.md` to note intake ends as `ready`
- [x] T006 [P] Update flow diagrams in `docs/specs/user-flow.md` — Section 5 annotation or Section 2 clarification if needed (verified: diagrams already correct — `/fab-continue` transitions between stages match; per-stage state machine unchanged)

## Phase 4: Memory File Consistency

- [x] T007 [P] Update `.status.yaml` initial state description in `docs/memory/fab-workflow/templates.md` — `intake: pending` not `intake: active`
- [x] T008 [P] Update dispatch description in `docs/memory/fab-workflow/planning-skills.md` — remove single-dispatch rule references, describe consolidated dispatch
- [x] T009 [P] Update state transition examples in `docs/memory/fab-workflow/change-lifecycle.md` — `ready` as default post-generation state

---

## Execution Order

- T001 is independent (template fix, no dependencies)
- T002 and T003 are the core changes, no dependency between them
- T004-T006 are independent spec updates (all [P])
- T007-T009 are independent memory updates (all [P])
