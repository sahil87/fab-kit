# Tasks: Replace .envrc Symlink with Line-Ensuring Sync

**Change**: 260216-ymvx-DEV-1043-envrc-line-sync
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Replace section 2 (.envrc) in `fab/.kit/scripts/fab-sync.sh` (lines 101-119): remove symlink logic, add symlink-to-file migration, add line-ensuring logic modeled on section 7 (.gitignore, lines 372-401). Read lines from `fab/.kit/scaffold/envrc`, skip comments/empty lines, append missing lines to `.envrc`. Guard the entire section on `scaffold/envrc` existence.

---

## Execution Order

- T001 is the only task — no dependencies.
