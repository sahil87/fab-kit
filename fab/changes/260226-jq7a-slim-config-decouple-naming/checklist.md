# Quality Checklist: Slim Config & Decouple Naming

**Change**: 260226-jq7a-slim-config-decouple-naming
**Generated**: 2026-02-26
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Remove git section: `config.yaml` and scaffold contain no `git:` key
- [ ] CHK-002 Remove naming section: `config.yaml` and scaffold contain no `naming:` key
- [ ] CHK-003 Rename rules → stage_directives: both config files use `stage_directives` with all 6 stage placeholders
- [ ] CHK-004 issue_id in status.yaml template: `fab/.kit/templates/status.yaml` contains `issue_id: null`
- [ ] CHK-005 fab-new writes issue_id: Linear ticket creates change with `issue_id` in status.yaml, not in slug
- [ ] CHK-006 git-pr reads issue_id: PR title includes issue ID when `issue_id` is non-null
- [ ] CHK-007 git-branch no gate: `/git-branch` has no `git.enabled` check
- [ ] CHK-008 git-branch no prefix: branch name equals change name directly
- [ ] CHK-009 fab-switch always shows tip: no `git.enabled` conditional on the hint line
- [ ] CHK-010 fab-status always shows branch: no `git.enabled` conditional on branch display
- [ ] CHK-011 Config verbosity: `config.yaml` is under 40 lines total
- [ ] CHK-012 naming.md spec: `docs/specs/naming.md` covers all 5 naming conventions
- [ ] CHK-013 specs index: `docs/specs/index.md` has naming.md row
- [ ] CHK-014 Migration file: `fab/.kit/migrations/0.10.0-to-0.20.0.md` exists with correct steps

## Behavioral Correctness

- [ ] CHK-015 Scripts default gracefully: `dispatch.sh` and `batch-fab-switch-change.sh` `get_branch_prefix()` return `""` when no git section
- [ ] CHK-016 stage_directives consumed: skills reference `stage_directives`, not `rules`

## Scenario Coverage

- [ ] CHK-017 Config without git section: skills proceed normally (git implicitly enabled)
- [ ] CHK-018 fab-new with Linear ticket: folder slug has no issue ID, status.yaml has issue_id
- [ ] CHK-019 fab-new without Linear ticket: issue_id is null in status.yaml
- [ ] CHK-020 PR title with issue_id: includes the ID
- [ ] CHK-021 PR title without issue_id: omits the ID cleanly

## Edge Cases & Error Handling

- [ ] CHK-022 Missing config.yaml: git-branch assumes enabled, no prefix (existing fallback preserved)

## Code Quality

- [ ] CHK-023 Pattern consistency: skill file structure follows existing patterns (frontmatter, steps, scenarios)
- [ ] CHK-024 No unnecessary duplication: config comment style consistent across project config and scaffold

## Documentation Accuracy

- [ ] CHK-025 Memory files updated: configuration.md, change-lifecycle.md, templates.md reflect new structure
- [ ] CHK-026 Spec files updated: skills.md and architecture.md reference updated field names

## Cross References

- [ ] CHK-027 No stale references: grep for `git.enabled`, `git.branch_prefix`, `naming.format`, `config.rules` returns zero hits in skill files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
