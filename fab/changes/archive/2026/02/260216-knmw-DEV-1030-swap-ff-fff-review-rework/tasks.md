# Tasks: Swap fab-ff and fab-fff Review Failure Behavior

**Change**: 260216-knmw-DEV-1030-swap-ff-fff-review-rework
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill Files

- [x] T001 [P] Update `/fab-ff` review failure behavior in `fab/.kit/skills/fab-ff.md` — replace bail-on-failure in Step 5 (Review) with interactive rework menu (3 options: fix code, revise tasks, revise spec). Update Purpose, description frontmatter, and Error Handling table.
- [x] T002 [P] Update `/fab-fff` review failure behavior in `fab/.kit/skills/fab-fff.md` — replace interactive rework menu in Step 7 (Review) with autonomous rework: agent decision heuristics, 3-cycle retry cap, escalation after 2 consecutive fix-code failures, bail message format. Update Purpose, description frontmatter, and Error Handling table.

## Phase 2: Supporting Documentation

- [x] T003 [P] Update autonomy levels table in `fab/.kit/skills/_context.md` — swap Escape valve entries for `/fab-ff` and `/fab-fff` rows.
- [x] T004 [P] Update `docs/memory/fab-workflow/planning-skills.md` — swap review failure descriptions in `/fab-ff` and `/fab-fff` sections, update "Scope Differentiation" design decision, add changelog entry.
- [x] T005 [P] Update `docs/memory/fab-workflow/execution-skills.md` — update "Pipeline invocation" note in Overview to reflect swapped behavior, add changelog entry.

## Phase 3: Verification

- [x] T006 Check `docs/specs/skills.md` and `docs/specs/user-flow.md` for any references to the old review failure behavior and update if needed.

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003, T004, T005 are independent (parallel), but should reference T001/T002 for consistency
- T006 depends on T001-T005 (final verification pass)
