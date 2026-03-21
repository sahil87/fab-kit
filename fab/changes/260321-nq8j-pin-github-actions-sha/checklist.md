# Quality Checklist: Pin GitHub Actions to commit SHAs for supply chain safety

**Change**: 260321-nq8j-pin-github-actions-sha
**Generated**: 2026-03-21
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 SHA pinning: All 3 external actions in `release.yml` use 40-char commit SHAs
- [x] CHK-002 Tag comments: Each SHA-pinned line has a trailing `# vN` comment with the original tag

## Behavioral Correctness

- [x] CHK-003 No functional change: Only `uses:` reference format changed — no workflow logic, permissions, or configuration modified

## Scenario Coverage

- [x] CHK-004 All external actions pinned: `actions/checkout`, `actions/setup-go`, `extractions/setup-just` all use SHAs
- [x] CHK-005 Internal actions unchanged: No `wvrdz/*` references in this workflow (N/A for fab-kit)

## Edge Cases & Error Handling

- [x] CHK-006 **N/A**: No edge cases — purely declarative YAML reference change

## Code Quality

- [x] CHK-007 Pattern consistency: SHA + comment format matches org convention (`{sha} # {tag}`)
- [x] CHK-008 No unnecessary duplication: Single file, 3 line changes

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
