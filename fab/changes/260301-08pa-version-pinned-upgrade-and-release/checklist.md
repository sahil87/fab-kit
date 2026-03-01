# Quality Checklist: Version-Pinned Upgrade and Release

**Change**: 260301-08pa-version-pinned-upgrade-and-release
**Generated**: 2026-03-02
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Upgrade tag argument: `fab-upgrade.sh v0.24.0` downloads the specified tagged release
- [ ] CHK-002 Upgrade latest fallback: `fab-upgrade.sh` (no args) downloads latest release (unchanged behavior)
- [ ] CHK-003 Release push to current branch: `fab-release.sh` pushes to `$(git branch --show-current)` instead of hardcoded `main`
- [ ] CHK-004 Release `--no-latest` flag: `fab-release.sh patch --no-latest` passes `--latest=false` to `gh release create`
- [ ] CHK-005 Argument parsing: bump type is required positional, `--no-latest` is optional flag, unknown flags error

## Behavioral Correctness
- [ ] CHK-006 Release on main: behavior is identical to before (push to main, release marked latest)
- [ ] CHK-007 Upgrade "already up to date" with tag: displays `Already on $tag ($version)` instead of generic message

## Scenario Coverage
- [ ] CHK-008 Tag not found: error message includes the tag and `gh release view` hint
- [ ] CHK-009 Backport workflow: release from non-main branch pushes to that branch, tag created, `--latest=false` applied
- [ ] CHK-010 Missing bump type with `--no-latest`: displays usage, does not treat flag as bump type
- [ ] CHK-011 Completion note: `--no-latest` releases show "Note: This release was NOT marked as 'latest'."

## Edge Cases & Error Handling
- [ ] CHK-012 Unknown flag: `fab-release.sh patch --unknown` produces error listing valid options
- [ ] CHK-013 No arguments to release: shows usage (existing behavior preserved)

## Code Quality
- [ ] CHK-014 Pattern consistency: new argument parsing follows existing script patterns (set -euo pipefail, function-based structure)
- [ ] CHK-015 No unnecessary duplication: existing utilities reused where applicable

## Documentation Accuracy
- [ ] CHK-016 README documents both upgrade modes: `fab-upgrade.sh` and `fab-upgrade.sh v0.24.0`

## Cross References
- [ ] CHK-017 Memory file `docs/memory/fab-workflow/distribution.md` updated with version-pinned upgrade and backport release scenarios

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
