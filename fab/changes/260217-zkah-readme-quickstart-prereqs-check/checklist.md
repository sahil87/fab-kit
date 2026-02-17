# Quality Checklist: README Quick Start Restructure + fab-sync Prerequisites Check

**Change**: 260217-zkah-readme-quickstart-prereqs-check
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Collapse Initialize into Install: Initialize content appears as h4 sub-section under "1. Install"
- [ ] CHK-002 Collapse Initialize into Install: "From a local clone" appears as h4 sub-section under "1. Install"
- [ ] CHK-003 Collapse Initialize into Install: "Updating from a previous version" appears as h4 sub-section under "1. Install"
- [ ] CHK-004 Update TOC: Contents line has no standalone "Updating" entry
- [ ] CHK-005 Heading hierarchy: Install sub-sections use `####` (h4)
- [ ] CHK-006 New prerequisites sync step: `fab/.kit/sync/1-prerequisites.sh` exists and checks yq, jq, gh, direnv, bats
- [ ] CHK-007 Existing sync steps renumber: `1-direnv.sh` → `2-direnv.sh`, `2-sync-workspace.sh` → `3-sync-workspace.sh`
- [ ] CHK-008 Prerequisites check is fatal: missing tool causes exit 1
- [ ] CHK-009 Prerequisites uses `command -v`: tool detection via `command -v`, not `which` or version flags

## Behavioral Correctness

- [ ] CHK-010 Step numbering: Quick Start steps are 1 (Install), 2 (Your first change), 3 (Going parallel)
- [ ] CHK-011 Standalone Updating removed: no `## Updating` section outside Quick Start

## Scenario Coverage

- [ ] CHK-012 All prerequisites present: sync pipeline completes successfully when all tools installed
- [ ] CHK-013 Missing prerequisites: script exits 1 and names missing tool(s) with pointer to README
- [ ] CHK-014 Multiple tools missing: error lists all missing tools, not just the first
- [ ] CHK-015 Sync execution order: steps run 1-prerequisites → 2-direnv → 3-sync-workspace

## Edge Cases & Error Handling

- [ ] CHK-016 Fatal exit prevents downstream: when prerequisites fail, no subsequent sync steps execute

## Code Quality

- [ ] CHK-017 Pattern consistency: `1-prerequisites.sh` follows shell conventions of existing sync scripts (shebang, set flags, error output style)
- [ ] CHK-018 No unnecessary duplication: reuses existing patterns rather than inventing new ones

## Documentation Accuracy

- [ ] CHK-019 README content preserved: all original Install, Initialize, and Updating content is present after restructure (no accidental deletions)
- [ ] CHK-020 Anchor links work: TOC anchors point to valid headings

## Cross References

- [ ] CHK-021 No stale references to old sync filenames in other files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
