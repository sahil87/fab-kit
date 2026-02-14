# Quality Checklist: Kit Version Migrations

**Change**: 260213-k7m2-kit-version-migrations
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Engine Version File: `fab/.kit/VERSION` continues to work as before — unchanged by this change
- [x] CHK-002 Local Project Version File: `fab/VERSION` is created by `_init_scaffold.sh` with correct new/existing project logic
- [x] CHK-003 Migration Directory: `fab/.kit/migrations/` exists and ships with the kit
- [x] CHK-004 Migration File Naming: skill accepts `{FROM}-to-{TO}.md` with full semver and applies range-based matching (`FROM <= version < TO`)
- [x] CHK-005 Migration File Structure: `/fab-update` skill documents and processes Pre-check → Changes → Verification sections
- [x] CHK-006 `/fab-update` Skill: skill file exists at `fab/.kit/skills/fab-update.md` with correct frontmatter
- [x] CHK-007 Range-Based Discovery: `/fab-update` implements the discovery algorithm (scan, sort, match, chain, gap-skip, finalize to engine version)
- [x] CHK-008 Non-Overlapping Validation: `/fab-update` validates no overlapping ranges before applying
- [x] CHK-009 `/fab-init` VERSION Creation: init skill documents `fab/VERSION` creation in bootstrap steps
- [x] CHK-010 `fab-upgrade.sh` Drift Reminder: script prints drift reminder when `fab/VERSION` < engine after upgrade
- [x] CHK-011 `fab-release.sh` Chain Validation: script warns when no migration targets the new version and when overlapping ranges detected
- [x] CHK-012 Status Drift Display: `/fab-status` shows version drift warning when `fab/VERSION` < `fab/.kit/VERSION`

## Behavioral Correctness

- [x] CHK-013 Scaffold new project: when no `fab/config.yaml` exists, `fab/VERSION` gets engine version value
- [x] CHK-014 Scaffold existing project: when `fab/config.yaml` exists but no `fab/VERSION`, `fab/VERSION` gets base version `0.1.0`
- [x] CHK-015 Scaffold idempotent: when `fab/VERSION` already exists, `_init_scaffold.sh` does not overwrite it
- [x] CHK-016 Upgrade drift check: `fab-upgrade.sh` handles missing `fab/VERSION` gracefully (guidance message, not crash)

## Scenario Coverage

- [x] CHK-017 Versions equal: `/fab-update` reports "already up to date" when versions match
- [x] CHK-018 Single migration match: migration applies when current version falls within FROM-TO range
- [x] CHK-019 Wide range: migration applies when current version is between FROM and TO (e.g., 3.4.0 in 2.1.0-to-4.3.0)
- [x] CHK-020 Chained migrations: multiple migrations execute in sequence, each updating fab/VERSION
- [x] CHK-021 Gap skip: when no migration covers current version, algorithm skips to next available FROM
- [x] CHK-022 No migrations exist: fab/VERSION updated to engine version with appropriate message
- [x] CHK-023 Version ahead: warning message when local > engine
- [x] CHK-024 Overlapping ranges: error reported, no migrations applied
- [x] CHK-025 Migration failure: stops at failed step, fab/VERSION reflects last successful migration

## Edge Cases & Error Handling

- [x] CHK-026 `fab/VERSION` missing for `/fab-update`: clear error directing to `/fab-init`
- [x] CHK-027 `fab/.kit/VERSION` missing for `/fab-update`: clear error about corrupted kit
- [x] CHK-028 Pre-check failure in migration: stops before applying changes, VERSION not updated
- [x] CHK-029 Mid-sequence failure: partial progress preserved, resume guidance provided

## Documentation Accuracy

- [x] CHK-030 **N/A**: Memory hydration happens in the hydrate stage, not during apply/review
- [x] CHK-031 **N/A**: Memory hydration happens in the hydrate stage, not during apply/review
- [x] CHK-032 **N/A**: Memory hydration happens in the hydrate stage, not during apply/review

## Cross References

- [x] CHK-033 `fab-help.sh` includes `/fab-update` in skill listing
- [x] CHK-034 Symlink auto-discovery: `_init_scaffold.sh` glob pattern will pick up `fab-update.md` automatically (no explicit changes needed)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

## Review Notes

- Fixed: `_init_scaffold.sh` was silent when `fab/VERSION` already exists — added `fab/VERSION: OK (...)` output to match spec scenario for re-init
- Spot-checked key spec scenarios (GIVEN/WHEN/THEN) against implementation — all match
- `fab-update.md` is a skill (LLM instruction file), not executable code — scenario coverage is verified against the documented algorithm, which the LLM will interpret at runtime
- `fab-release.sh` overlap detection uses `sort -V` for proper semver comparison
