# Quality Checklist: Relocate Kit to System Cache

**Change**: 260402-gnx5-relocate-kit-to-system-cache
**Generated**: 2026-04-02
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 kitpath.KitDir(): Resolves exe-sibling kit/ directory correctly, handles symlinks
- [x] CHK-002 fab kit-path: Outputs correct absolute path to cache kit directory
- [x] CHK-003 change.New: Reads status.yaml template from cache kit (not fab/.kit/)
- [x] CHK-004 preflight staleness: Reads VERSION from cache kit, compares with config fab_version
- [x] CHK-005 fabhelp: Scans skills from cache kit directory
- [x] CHK-006 hooklib.Sync: Registers inline `fab hook <subcommand>` commands (not shell script paths)
- [x] CHK-007 fab init: Creates project without fab/.kit/ directory, sets fab_version, deploys skills
- [x] CHK-008 fab upgrade: Updates fab_version, re-syncs skills, does not touch fab/.kit/
- [x] CHK-009 defaultRepo constant: Hardcoded, no kit.conf reads anywhere in codebase

## Behavioral Correctness
- [x] CHK-010 Old hook commands migrated: fab hook sync replaces bash-script commands with inline fab hook commands in existing settings.local.json
- [x] CHK-011 Preflight works without fab/.kit/: No references to fab/.kit/ paths in preflight code
- [x] CHK-012 Skills read templates via fab kit-path: _generation.md references $(fab kit-path)/templates/ not fab/.kit/templates/

## Removal Verification
- [x] CHK-013 Hook scripts removed: src/kit/ does not contain hooks/ directory
- [x] CHK-014 kit.conf removed: src/kit/ does not contain kit.conf; no Go code reads kit.conf
- [x] CHK-015 fab/.kit/ not created: fab init and fab upgrade do not create fab/.kit/ in user projects
- [x] CHK-016 build-type guard removed: _preamble.md has no Test-Build Guard section

## Scenario Coverage
- [x] CHK-017 fab-go resolves kit from cache: kitpath.KitDir() returns correct path when binary is at ~/.fab-kit/versions/{v}/fab-go
- [x] CHK-018 fab kit-path outputs path: Command exits 0 with correct path when cache exists
- [x] CHK-019 Fresh project hook sync: New settings.local.json has inline hook commands
- [x] CHK-020 Migration on existing project: fab/.kit/ removed, hooks inlined, envrc/gitignore cleaned
- [x] CHK-021 Init creates project without fab/.kit/: Scaffold runs, skills deployed, no fab/.kit/

## Edge Cases & Error Handling
- [x] CHK-022 kit-path without cache: Command exits non-zero with helpful error
- [x] CHK-023 kitpath.KitDir() without kit sibling: Returns error, not panic
- [x] CHK-024 Migration without cache: Stops at prerequisite with guidance message

## Code Quality
- [x] CHK-025 Pattern consistency: kitpath utility follows existing internal package patterns (resolve, status, etc.)
- [x] CHK-026 No unnecessary duplication: Existing utilities reused (resolve.FabRoot(), config reading)

## Documentation Accuracy
- [x] CHK-027 kit-architecture.md: Updated directory structure, path resolution, no hooks
- [x] CHK-028 distribution.md: Updated init/upgrade flow (no kit copy to project)
- [ ] CHK-029 constitution.md: Principle V reworded (no "cp -r fab/.kit/" language) — **PARTIAL**: Principle V is correct, but Additional Constraints section at lines 31-32 still references `fab/.kit/skills/`
- [x] CHK-030 Spec files: architecture.md updated with new directory structure

## Cross References
- [ ] CHK-031 No stale fab/.kit/ references in Go code: grep confirms zero matches in src/go/ — **PARTIAL**: code paths are correct, but stale references remain in comments (upgrade.go line 10, crud.go line 124) and test fixture strings (expected for migration testing)
- [x] CHK-032 No stale fab/.kit/ references in skill files: grep confirms zero matches in src/kit/skills/ (except fab/.kit-migration-version references which are a different file, not fab/.kit/)
- [x] CHK-033 Build scripts reference src/kit/: justfile, release.sh, copilot-code-review.yml all updated

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
