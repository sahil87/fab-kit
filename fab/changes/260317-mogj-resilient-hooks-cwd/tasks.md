# Tasks: Resilient Hooks CWD

**Change**: 260317-mogj-resilient-hooks-cwd
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Update `fab/.kit/hooks/on-session-start.sh` — replace `dirname "$0"` resolution with `git rev-parse --show-toplevel` pattern
- [x] T002 [P] Update `fab/.kit/hooks/on-stop.sh` — same pattern
- [x] T003 [P] Update `fab/.kit/hooks/on-user-prompt.sh` — same pattern
- [x] T004 [P] Update `fab/.kit/hooks/on-artifact-write.sh` — same pattern

## Phase 2: Verification

- [x] T005 Verify all four scripts are consistent and functional — run each hook script from a subdirectory to confirm resolution works

---

## Execution Order

- T001–T004 are independent and parallelizable
- T005 depends on T001–T004
