# Quality Checklist: Consolidate Status Field Naming

**Change**: 260227-gasp-consolidate-status-field-naming
**Generated**: 2026-02-27
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 issues field: `fab/.kit/templates/status.yaml` contains `issues: []` and no `issue_id`
- [ ] CHK-002 prs field: `fab/.kit/templates/status.yaml` contains `prs: []` and no `shipped`
- [ ] CHK-003 workflow schema: `fab/.kit/schemas/workflow.yaml` has `issues:` and `prs:` sections, no `shipped:` section
- [ ] CHK-004 add_issue function: `stageman.sh add-issue` appends to `issues[]` with dedup
- [ ] CHK-005 get_issues function: `stageman.sh get-issues` emits one ID per line, empty for `[]`
- [ ] CHK-006 add_pr function: `stageman.sh add-pr` appends to `prs[]` with dedup
- [ ] CHK-007 get_prs function: `stageman.sh get-prs` emits one URL per line, empty for `[]`
- [ ] CHK-008 fab-new: uses `stageman.sh add-issue`, no raw `yq` for issue_id
- [ ] CHK-009 git-pr issues: reads via `stageman.sh get-issues`, joins with space in PR title
- [ ] CHK-010 git-pr add-pr: uses `stageman.sh add-pr` instead of `stageman.sh ship`
- [ ] CHK-011 sentinel rename: `.shipped` → `.pr-done` in git-pr, pipeline/run.sh, fab-archive

## Behavioral Correctness
- [ ] CHK-012 PR title with multiple issues: `feat: DEV-123 DEV-456 {title}` format
- [ ] CHK-013 PR title with no issues: `{type}: {title}` format (no empty prefix)
- [ ] CHK-014 Deduplication: adding same issue/PR twice does not create duplicate entries

## Removal Verification
- [ ] CHK-015 ship_url removed: no `ship_url()` function in stageman.sh
- [ ] CHK-016 is_shipped removed: no `is_shipped()` function in stageman.sh
- [ ] CHK-017 Old CLI routes removed: `ship` and `is-shipped` case branches gone from stageman.sh
- [ ] CHK-018 Old help entries removed: no `ship`/`is-shipped` in stageman help text

## Scenario Coverage
- [ ] CHK-019 New change from Linear ticket: `issues: ["DEV-988"]` via stageman
- [ ] CHK-020 Pipeline runner detects .pr-done: sentinel check updated in run.sh
- [ ] CHK-021 Archive cleans .pr-done: fab-archive Step 1 references .pr-done

## Edge Cases & Error Handling
- [ ] CHK-022 Missing status file: add_issue/add_pr/get_issues/get_prs return error for missing file
- [ ] CHK-023 Empty array: get_issues/get_prs produce empty output for `[]`

## Code Quality
- [ ] CHK-024 Pattern consistency: new functions follow atomic write pattern (tmpfile → mv) like existing stageman functions
- [ ] CHK-025 No unnecessary duplication: add_issue/add_pr share the same structural pattern as ship_url did

## Documentation Accuracy
- [ ] CHK-026 naming.md: PR title pattern uses `{issues}` not `{issue_id}`
- [ ] CHK-027 naming.md: backlog pattern updated for multiple issue IDs

## Cross References
- [ ] CHK-028 .gitignore: `fab/changes/**/.pr-done` pattern (repo root)
- [ ] CHK-029 scaffold fragment: `fab/changes/**/.pr-done` pattern
- [ ] CHK-030 Migration: handles scalar→array, null→empty, sentinel rename

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
