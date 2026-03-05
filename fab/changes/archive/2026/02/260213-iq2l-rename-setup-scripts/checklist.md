# Quality Checklist: Rename and Reorganize Bootstrap Scripts

**Change**: 260213-iq2l-rename-setup-scripts
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 fab-setup.sh renamed: `fab/.kit/scripts/_fab-scaffold.sh` exists, `fab/.kit/scripts/fab-setup.sh` does not
- [x] CHK-002 fab-update.sh renamed: `fab/.kit/scripts/fab-upgrade.sh` exists, `fab/.kit/scripts/fab-update.sh` does not
- [x] CHK-003 worktree script renamed: `fab/.kit/worktree-init-common/2-rerun-fab-scaffold.sh` exists, old file does not
- [x] CHK-004 fab-init.md updated: no references to `fab-setup.sh` remain in `fab/.kit/skills/fab-init.md`
- [x] CHK-005 model-tiers.yaml updated: no references to `fab-setup.sh` remain in `fab/.kit/model-tiers.yaml`
- [x] CHK-006 README.md updated: no references to `fab-setup.sh` or `fab-update.sh` remain in `README.md`
- [x] CHK-007 All centralized docs updated: grep for `fab-setup.sh` and `fab-update.sh` in `fab/docs/` returns 0 matches (excluding archive)
- [x] CHK-008 Design docs updated: grep for `fab-setup.sh` in `fab/design/` returns 0 matches

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-009 _fab-scaffold.sh executes correctly: running `bash fab/.kit/scripts/_fab-scaffold.sh` produces same behavior as old `fab-setup.sh`
- [x] CHK-010 fab-upgrade.sh internal call correct: `fab-upgrade.sh` references `_fab-scaffold.sh` (not `fab-setup.sh`) in its re-run step
- [x] CHK-011 worktree script calls correct target: `2-rerun-fab-scaffold.sh` calls `_fab-scaffold.sh`

## Removal Verification
<!-- Every deprecated requirement is actually gone -->
- [x] CHK-012 No stale `fab-setup.sh` references: grep entire repo (excluding `fab/changes/archive/`) for `fab-setup\.sh` returns 0 matches
- [x] CHK-013 No stale `fab-update.sh` references: grep entire repo (excluding `fab/changes/archive/`) for `fab-update\.sh` returns 0 matches

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-014 Script content preserved: `_fab-scaffold.sh` functional content identical to original (only self-referencing comments changed)
- [x] CHK-015 fab-upgrade.sh content updated: internal call path now references `_fab-scaffold.sh`

## Documentation Accuracy
<!-- Extra category from config.yaml -->
- [x] CHK-016 kit-architecture.md directory listing matches actual `fab/.kit/scripts/` contents
- [x] CHK-017 distribution.md bootstrap and update instructions reference correct script names
- [x] CHK-018 init.md delegation pattern references correct script name

## Cross References
<!-- Extra category from config.yaml -->
- [x] CHK-019 Archives NOT modified: no files under `fab/changes/archive/` were changed
- [x] CHK-020 Other active changes NOT modified: no files under other `fab/changes/260213-*` directories were changed

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
