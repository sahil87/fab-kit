# Quality Checklist: Version Staleness Warning

**Change**: 260226-koj1-version-staleness-warning
**Generated**: 2026-02-26
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Stamp written: `fab-sync.sh` writes `fab/.kit-sync-version` with engine version after skill deployment
- [ ] CHK-002 Staleness detection: `lib/preflight.sh` emits stderr warning when stamp mismatches or is missing
- [ ] CHK-003 Rename complete: All references to `fab/project/VERSION` updated to `fab/.kit-migration-version`
- [ ] CHK-004 Migration file: `0.20.0-to-0.21.0.md` exists with correct Pre-check, Changes, Verification sections
- [ ] CHK-005 Backward compat: `2-sync-workspace.sh` migrates old `fab/project/VERSION` to new location

## Behavioral Correctness
- [ ] CHK-006 Non-blocking: Preflight staleness warning does not change exit code or alter stdout YAML
- [ ] CHK-007 Idempotent stamp: Re-running `fab-sync.sh` on same version produces OK output, not error
- [ ] CHK-008 Gitignored: `fab/.kit-sync-version` listed in scaffold gitignore fragment

## Scenario Coverage
- [ ] CHK-009 Fresh sync: stamp created on first run
- [ ] CHK-010 Stamp behind engine: warning includes both version numbers
- [ ] CHK-011 Stamp missing: generic "may be out of sync" warning
- [ ] CHK-012 Init fails first: no staleness warning when config.yaml missing

## Edge Cases & Error Handling
- [ ] CHK-013 Both old and new VERSION files exist: old file deleted, new file preserved
- [ ] CHK-014 Stamp survives kit replacement: file is outside `.kit/` directory

## Code Quality
- [ ] CHK-015 Pattern consistency: New shell code follows existing `2-sync-workspace.sh` output patterns (Created/Updated/OK)
- [ ] CHK-016 No unnecessary duplication: Version comparison logic is minimal, not duplicating existing patterns

## Documentation Accuracy
- [ ] CHK-017 Migration file references: All inline `fab/project/VERSION` references in migration files updated
- [ ] CHK-018 Skill references: `fab-setup.md` and `fab-status.md` reference new path

## Cross References
- [ ] CHK-019 Scaffold gitignore: entry uses correct path `fab/.kit-sync-version`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
