# Tasks: Version Staleness Warning

**Change**: 260226-koj1-version-staleness-warning
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `fab/.kit-sync-version` to scaffold gitignore fragment (`fab/.kit/scaffold/fragment-.gitignore`)

## Phase 2: Core Implementation

- [x] T002 Add sync version stamp logic to `fab/.kit/sync/2-sync-workspace.sh` — write `fab/.kit-sync-version` after skill deployment with Created/Updated/OK output
- [x] T003 Add backward-compat migration of `fab/project/VERSION` → `fab/.kit-migration-version` in `fab/.kit/sync/2-sync-workspace.sh` — one-time rename during sync
- [x] T004 Update `fab/.kit/sync/2-sync-workspace.sh` creation logic — replace all `fab/project/VERSION` references with `fab/.kit-migration-version` in the "1b" section
- [x] T005 Add staleness detection to `fab/.kit/scripts/lib/preflight.sh` — compare `fab/.kit/VERSION` vs `fab/.kit-sync-version`, emit stderr warning, non-blocking

## Phase 3: Integration & Edge Cases

- [x] T006 [P] Update `fab/.kit/scripts/fab-upgrade.sh` — replace `fab/project/VERSION` with `fab/.kit-migration-version`
- [x] T007 [P] Update `fab/.kit/skills/fab-setup.md` — replace all `fab/project/VERSION` references with `fab/.kit-migration-version`
- [x] T008 [P] Update `fab/.kit/skills/fab-status.md` — replace `fab/project/VERSION` references with `fab/.kit-migration-version`
- [x] T009 [P] Update `fab/.kit/migrations/0.9.0-to-0.10.0.md` — replace `fab/project/VERSION` references with `fab/.kit-migration-version`
- [x] T010 [P] Update `fab/.kit/migrations/0.10.0-to-0.20.0.md` — replace `fab/project/VERSION` references with `fab/.kit-migration-version`
- [x] T011 Create `fab/.kit/migrations/0.20.0-to-0.21.0.md` — migration file for the VERSION rename

## Phase 4: Polish

- [x] T012 Rename the actual `fab/project/VERSION` → `fab/.kit-migration-version` in this repo (move the file)

---

## Execution Order

- T003 and T004 both modify `2-sync-workspace.sh` — execute sequentially (T003 then T004, or combined)
- T005 depends on T001 (gitignore must be in place conceptually, though not a hard build dependency)
- T006-T011 are independent reference updates, all parallelizable
- T012 should run last (after all references are updated)
