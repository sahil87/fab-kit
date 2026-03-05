# Tasks: Swap batch-fab-archive-change defaults

**Change**: 260305-obib-swap-batch-archive-defaults
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Change the no-argument fallback from `--list` to `--all` in `fab/.kit/scripts/batch-fab-archive-change.sh` (line 83: `set -- --list` → `set -- --all`)
- [x] T002 Update the `usage()` function in `fab/.kit/scripts/batch-fab-archive-change.sh` to document that no-argument behavior archives all eligible changes, and `--list` is available for preview

---

## Execution Order

- T001 and T002 are independent (different functions in the same file)
