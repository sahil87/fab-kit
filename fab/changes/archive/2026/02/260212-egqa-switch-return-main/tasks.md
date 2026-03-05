# Tasks: Add fab-switch --blank to deactivate the current change

**Change**: 260212-egqa-switch-return-main
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 [P] Update `fab/.kit/skills/fab-switch.md` — add `--blank` flag to Arguments section, add Deactivation Flow subsection to Behavior section (between Argument Flow and Switch Flow), add deactivation output examples to Output section, add deactivation error cases to Error Handling table, update Key Properties table to note `fab/current` may be deleted
- [x] T002 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` — add `/fab-switch --blank` to the `fab/current` lifecycle bullet list (alongside `/fab-archive`), add `--blank` deactivation behavior to the `/fab-switch` section description

---

## Execution Order

- T001 and T002 are independent (different files) — can execute in parallel
