# Quality Checklist: Move 5 Cs and VERSION into fab/project/

**Change**: 260219-wq0e-move-5cs-to-project-folder
**Generated**: 2026-02-19
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Directory structure: `fab/project/` contains all 6 files (config.yaml, constitution.md, context.md, code-quality.md, code-review.md, VERSION)
- [x] CHK-002 No 5 C files remain at `fab/` root (only backlog.md, current, changes/, sync/, .kit/)
- [x] CHK-003 Shell scripts: All variable-based path references updated (preflight.sh, changeman.sh, fab-upgrade.sh, batch-fab-switch-change.sh, 2-sync-workspace.sh)
- [x] CHK-004 Scaffold: 5 Cs moved to `fab/.kit/scaffold/fab/project/` with correct header comments
- [x] CHK-005 Skills: All 9 skill files updated — _context.md, fab-setup.md, _generation.md, fab-continue.md, fab-new.md, fab-status.md, fab-switch.md, fab-help.md, internal-consistency-check.md
- [x] CHK-006 Migration: `fab/.kit/migrations/0.9.0-to-0.10.0.md` exists with pre-check, changes, and verification sections
- [x] CHK-007 **N/A**: `fab/.kit/VERSION` contains `0.10.1` (post-release bump); migration file targets 0.10.0 correctly
- [x] CHK-008 Agent files: `.claude/agents/` files regenerated with new paths

## Behavioral Correctness

- [x] CHK-009 preflight.sh: Validates `fab/project/config.yaml` and `fab/project/constitution.md` (not old paths)
- [x] CHK-010 changeman.sh: Reads git settings from `fab/project/config.yaml`
- [x] CHK-011 2-sync-workspace.sh: Detects existing project via `fab/project/config.yaml`, creates VERSION at `fab/project/VERSION`
- [x] CHK-012 _context.md: Always Load section lists `fab/project/` paths; state derivation checks `fab/project/config.yaml`

## Scenario Coverage

- [x] CHK-013 Clean directory hierarchy: `fab/` root shows .kit/, project/, changes/, sync/, backlog.md, current
- [x] CHK-014 Migration idempotent: Pre-check skips when files already at `fab/project/`
- [x] CHK-015 Scaffold copy-if-absent: Existing `fab/project/config.yaml` not overwritten by sync

## Edge Cases & Error Handling

- [x] CHK-016 Optional files: References to context.md, code-quality.md, code-review.md still marked optional in skills
- [x] CHK-017 fab/current unchanged: Pointer file remains at `fab/current`, not moved to `fab/project/`
- [x] CHK-018 fab-help.sh reads `fab/.kit/VERSION` (engine version) — confirmed no incorrect change to project VERSION path

## Code Quality

- [x] CHK-019 Pattern consistency: Path changes use same variable patterns as surrounding code ($fab_root, $FAB_ROOT, $fab_dir, $FAB_DIR)
- [x] CHK-020 No unnecessary duplication: No new utility functions or abstractions introduced for path resolution
- [x] CHK-021 Readability: Uses `$fab_root/project/config.yaml` (simple concatenation), not complex path-building logic

## Documentation Accuracy

- [x] CHK-022 README.md: Directory structure, quick start instructions, and 5 Cs table all reference `fab/project/` paths
- [x] CHK-023 Memory files: All 12 affected memory files in docs/memory/fab-workflow/ updated
- [x] CHK-024 Spec files: architecture.md, skills.md, glossary.md updated

## Cross References

- [x] CHK-025 No stale references: Grep for `fab/config.yaml` (without project/) across codebase returns 0 matches outside changelog entries
- [x] CHK-026 Test files: src/lib/preflight/test.bats and src/lib/sync-workspace/ tests reference new paths
- [x] CHK-027 Scaffold path references in skills (e.g., fab-setup.md) point to `fab/.kit/scaffold/fab/project/`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
