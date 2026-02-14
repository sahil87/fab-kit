# Tasks: Kit Version Migrations

**Change**: 260213-k7m2-kit-version-migrations
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/migrations/` directory with `.gitkeep` to ensure it ships in `kit.tar.gz` releases
- [x] T002 Add `fab/VERSION` creation logic to `fab/.kit/scripts/_init_scaffold.sh` — check for `fab/config.yaml` to distinguish new vs existing projects: new project → copy `fab/.kit/VERSION`; existing project → write `0.1.0`; existing `fab/VERSION` → skip

## Phase 2: Core Implementation

- [x] T003 Create `/fab-update` skill file at `fab/.kit/skills/fab-update.md` — migration runner skill with: pre-flight checks (both VERSION files), range-based migration discovery algorithm (`FROM <= version < TO`), overlap validation, sequential execution with progress output, failure handling, gap skip logging, final `fab/VERSION` set to engine version
- [x] T004 [P] Update `fab/.kit/scripts/fab-upgrade.sh` — after successful upgrade, read `fab/VERSION` (if exists), compare to new `fab/.kit/VERSION`, print drift reminder if behind, print init guidance if missing
- [x] T005 [P] Update `fab/.kit/scripts/fab-release.sh` — after VERSION bump and before packaging, check `fab/.kit/migrations/` for: (a) a file with TO matching the new version (warn if missing), (b) overlapping ranges (warn if detected)

## Phase 3: Integration

- [x] T006 Update `fab/.kit/skills/fab-init.md` — add `fab/VERSION` to bootstrap steps (between step 1e and 1f), document in bootstrap output and re-run output
- [x] T007 [P] Update `fab/.kit/skills/fab-status.md` — add version drift warning when `fab/VERSION` < `fab/.kit/VERSION` to the status display behavior
- [x] T008 [P] Update `fab/.kit/scripts/fab-help.sh` — add `/fab-update` to the Maintenance section of the COMMANDS listing

---

## Execution Order

- T001 is independent setup (no dependencies)
- T002 is independent setup (no dependencies)
- T003 depends on T001 (migrations directory must exist for the skill to reference it)
- T004, T005 are independent script changes (can run alongside T003)
- T006 depends on T002 (init skill needs to reference VERSION creation in scaffold)
- T007, T008 are independent integration tasks
