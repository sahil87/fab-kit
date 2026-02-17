# Quality Checklist: Fold resolve-change into changeman

**Change**: 260216-oinh-DEV-1045-fold-resolve-into-changeman
**Generated**: 2026-02-17
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Resolve from fab/current: `changeman.sh resolve` reads fab/current, strips whitespace, prints name to stdout
- [ ] CHK-002 Resolve from override: `changeman.sh resolve <override>` performs case-insensitive substring matching, exact match priority
- [ ] CHK-003 Resolve error cases: missing fab/current, empty fab/current, no match, multiple matches, missing changes dir — all exit 1 with diagnostic
- [ ] CHK-004 Switch normal flow: resolves name, writes fab/current, git branch integration, structured output
- [ ] CHK-005 Switch deactivation: `--blank` deletes fab/current, handles already-blank
- [ ] CHK-006 Switch config reading: reads git.enabled and branch_prefix from config.yaml via yq, defaults when missing
- [ ] CHK-007 Switch git integration: checkout existing / create new branch, skip when disabled / not a repo, non-fatal on failure
- [ ] CHK-008 Preflight migration: uses `$CHANGEMAN resolve` instead of sourcing resolve-change.sh
- [ ] CHK-009 Batch script migration: both batch scripts use `$CHANGEMAN resolve`
- [ ] CHK-010 resolve-change.sh deleted: no remaining references in fab/.kit/

## Behavioral Correctness

- [ ] CHK-011 Resolve behavior identical: matching algorithm, case-insensitivity, archive exclusion — ported verbatim from resolve-change.sh
- [ ] CHK-012 Preflight output unchanged: same YAML structure and content after migration
- [ ] CHK-013 Switch output format matches spec: `fab/current → {name}`, stage, branch, next command

## Removal Verification

- [ ] CHK-014 resolve-change.sh deleted: file removed from fab/.kit/scripts/lib/
- [ ] CHK-015 No source references: grep confirms no `source.*resolve-change` or `resolve_change` function calls remain
- [ ] CHK-016 src/lib/resolve-change/ deleted: dev directory removed

## Scenario Coverage

- [ ] CHK-017 Resolve exact match: test exists in changeman test.bats
- [ ] CHK-018 Resolve partial match: single and multiple match tests
- [ ] CHK-019 Resolve fab/current: default mode test
- [ ] CHK-020 Switch normal: end-to-end test with git integration
- [ ] CHK-021 Switch blank: deactivation test
- [ ] CHK-022 Preflight integration: preflight tests pass after migration

## Edge Cases & Error Handling

- [ ] CHK-023 Archive folder excluded from resolve matches
- [ ] CHK-024 Git failure non-fatal: switch completes even when git checkout fails
- [ ] CHK-025 Config missing: switch defaults to git enabled, empty prefix
- [ ] CHK-026 Empty fab/current handled: resolve exits 1 with diagnostic

## Code Quality

- [ ] CHK-027 Pattern consistency: new code follows changeman's existing function structure (cmd_new, cmd_rename pattern)
- [ ] CHK-028 No unnecessary duplication: resolve logic not duplicated between resolve and switch

## Documentation Accuracy

- [ ] CHK-029 SPEC-changeman.md updated: resolve and switch subcommands documented
- [ ] CHK-030 fab-switch.md updated: references changeman.sh switch instead of resolve-change.sh

## Cross References

- [ ] CHK-031 Help text updated: changeman --help shows resolve and switch subcommands
- [ ] CHK-032 CLI dispatch updated: resolve and switch entries in case statement

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
