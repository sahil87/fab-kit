# Quality Checklist: Delegate fab-switch Name Resolution to Shell

**Change**: 260216-jmy4-DEV-1044-switch-shell-name-resolution
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Argument Flow delegation: Skill instructs LLM to call `resolve-change.sh` via Bash instead of in-prompt matching
- [x] CHK-002 Multiple match handling: Skill instructs LLM to parse stderr comma-separated names on multi-match exit
- [x] CHK-003 No Argument Flow preserved: Listing behavior unchanged — no call to `resolve-change.sh`
- [x] CHK-004 Context Loading updated: `resolve-change.sh` dependency documented in Context Loading section
- [x] CHK-005 File sync: Both `fab/.kit/skills/fab-switch.md` and `.claude/agents/fab-switch.md` have identical body content

## Behavioral Correctness

- [x] CHK-006 No match behavior: On "No change matches" stderr, skill lists all available changes (same UX as before)
- [x] CHK-007 Exact match behavior: Full folder name resolves correctly via shell (the original bug scenario)
- [x] CHK-008 Partial match behavior: Partial slug still resolves to single match

## Scenario Coverage

- [x] CHK-009 Exact match scenario: Spec scenario "Exact match by full folder name" — skill instructions cover this path
- [x] CHK-010 Single partial match scenario: Spec scenario "Single partial match" — covered
- [x] CHK-011 No match scenario: Spec scenario "No match" — covered
- [x] CHK-012 Multiple matches scenario: Spec scenario "Multiple partial matches" — covered
- [x] CHK-013 No argument scenario: Spec scenario "No argument invocation" — covered (unchanged)

## Edge Cases & Error Handling

- [x] CHK-014 Deactivation flow: `--blank` flag behavior unaffected by changes
- [x] CHK-015 Branch integration: `--branch` and `--no-branch-change` flags unaffected
- [x] CHK-016 Error table: All error conditions and actions preserved

## Code Quality

- [x] CHK-017 Pattern consistency: Skill markdown follows existing formatting and section structure
- [x] CHK-018 No unnecessary duplication: No redundant resolution logic alongside shell delegation

## Documentation Accuracy

- [x] CHK-019 Shell invocation syntax: Bash command in skill is correct (`source`, `resolve_change`, `echo`)
- [x] CHK-020 Exit code semantics: Exit 0/1 handling matches actual `resolve-change.sh` behavior

## Cross References

- [x] CHK-021 Consistency with `resolve-change.sh` interface: Variable name `RESOLVED_CHANGE_NAME`, function signature `resolve_change <fab_root> [override]`
- [x] CHK-022 Consistency with preflight memory: `docs/memory/fab-workflow/preflight.md` documents the same source-and-invoke pattern

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
