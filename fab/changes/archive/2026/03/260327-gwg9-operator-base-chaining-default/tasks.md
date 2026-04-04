# Tasks: Operator Base-Chaining Default

**Change**: 260327-gwg9-operator-base-chaining-default
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Rewrite autopilot steps 5–9 in `fab/.kit/skills/fab-operator7.md` §6 Autopilot to implement stack-then-review default: replace merge/rebase steps (6–9) with record/dispatch-next/report/summary steps. Add note that `--merge-on-complete` reverts to previous behavior.
- [x] T002 Update the queue ordering strategy table in `fab/.kit/skills/fab-operator7.md` §6 Autopilot to note that user-provided ordering now implies implicit `--base` chaining by default (every change after the first gets `depends_on: [<prev-change-id>]`).
- [x] T003 Add `--merge-on-complete` flag documentation to `fab/.kit/skills/fab-operator7.md` §6 Autopilot — describe the flag, its natural language equivalents ("merge as you go"), and that it reverts to the previous merge/rebase behavior.
- [x] T004 Update the autopilot confirmation prompt text in `fab/.kit/skills/fab-operator7.md` §6 Autopilot: default says "Confirm upfront (creates PRs — merge after review).", `--merge-on-complete` says "Confirm upfront (merges PRs on completion)."

## Phase 2: Completion & Merge Behavior

- [x] T005 Add queue completion summary behavior to `fab/.kit/skills/fab-operator7.md` §6 Autopilot — after all changes complete, operator lists all PR links with dependency annotations and merge order suggestion. Include "merge all" conversational command.
- [x] T006 Add ordered merge behavior to `fab/.kit/skills/fab-operator7.md` §6 Autopilot — when user says "merge all", operator merges in dependency order waiting for CI on each. On CI failure, halt and report.

## Phase 3: Edge Cases & Cleanup

- [x] T007 Update failure handling in `fab/.kit/skills/fab-operator7.md` §6 Autopilot — note that "Rebase conflict → skip" does not apply in default stack-then-review mode (no rebase steps); cherry-pick conflict escalation remains unchanged.
- [x] T008 Update the "Working a Change" subsections in `fab/.kit/skills/fab-operator7.md` §6 — change "On completion: merge PR, optionally archive" to "On completion: PR ready, optionally archive" for consistency with the new default (merge is user-initiated, not automatic).

---

## Execution Order

- T001 should be done first as it establishes the new step numbering
- T002, T003, T004 can follow T001 in any order (they reference the updated steps)
- T005, T006 depend on T001 (extend the new flow)
- T007, T008 are independent cleanup tasks
