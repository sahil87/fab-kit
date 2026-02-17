# Quality Checklist: Consolidate worktree-init into fab-sync

**Change**: 260218-bx4d-consolidate-worktree-init-into-sync
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Thin orchestrator: `fab-sync.sh` iterates `fab/.kit/sync/*.sh` then `fab/sync/*.sh` in sorted order
- [x] CHK-002 Workspace sync logic: `2-sync-workspace.sh` contains full sync logic with correct path resolution
- [x] CHK-003 Direnv script: `fab/.kit/sync/1-direnv.sh` runs `direnv allow`
- [x] CHK-004 Envrc scaffold: `WORKTREE_INIT_SCRIPT` points to `fab/.kit/scripts/fab-sync.sh`
- [x] CHK-005 Sync README scaffold: `fab/.kit/scaffold/sync-readme.md` exists with naming guidance
- [x] CHK-006 README scaffolding: `2-sync-workspace.sh` creates `fab/sync/README.md` from scaffold if missing

## Behavioral Correctness
- [x] CHK-007 Path resolution: `2-sync-workspace.sh` derives `kit_dir` correctly from `fab/.kit/sync/` location
- [x] CHK-008 Orchestrator halts on failure: `set -euo pipefail` stops execution on non-zero exit
- [x] CHK-009 Missing project sync dir: no error when `fab/sync/` doesn't exist

## Removal Verification
- [x] CHK-010 `fab/.kit/worktree-init.sh` deleted
- [x] CHK-011 `fab/.kit/worktree-init-common/` deleted
- [x] CHK-012 `fab/worktree-init/` deleted (replaced by `fab/sync/`)
- [x] CHK-013 `fab/worktree-init/1-claude-settings.sh` and `assets/` deleted
- [x] CHK-014 No references to `worktree-init` remain in `README.md`

## Scenario Coverage
- [x] CHK-015 Normal execution: kit-level scripts run before project-level scripts
- [x] CHK-016 Idempotent: re-running `2-sync-workspace.sh` reports OK for configured items
- [x] CHK-017 New worktree env: `$WORKTREE_INIT_SCRIPT` from scaffold points to correct path

## Edge Cases & Error Handling
- [x] CHK-018 Script failure propagation: orchestrator stops on first non-zero exit

## Code Quality
- [x] CHK-019 Pattern consistency: new orchestrator follows existing shell script style (set -euo pipefail, path derivation pattern)
- [x] CHK-020 No unnecessary duplication: sync logic exists only in `2-sync-workspace.sh`, not duplicated in orchestrator

## Documentation Accuracy
- [x] CHK-021 README.md: all bootstrap references point to `fab-sync.sh` as single entry point
- [x] CHK-022 Scaffold sync-readme.md: accurately describes naming convention and execution order

## Cross References
- [x] CHK-023 `fab/.kit/scaffold/envrc` is consistent with actual script location

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
