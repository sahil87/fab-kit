# Quality Checklist: Smart Change Resolution & PR Summary Generation

**Change**: 260224-1jkh-smart-resolve-and-pr-summary
**Generated**: 2026-02-25
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Fallback resolution: `cmd_resolve()` returns single candidate when `fab/current` missing and exactly one valid change folder exists
- [x] CHK-002 Candidate filtering: Folders without `.status.yaml` are excluded from candidates
- [x] CHK-003 Archive exclusion: `archive/` directory is excluded from candidates
- [x] CHK-004 Stderr diagnostic: Guessing emits `(resolved from single active change)` to stderr
- [x] CHK-005 Multi-candidate error: 2+ candidates produce `No active change (multiple changes exist — use /fab-switch).`
- [x] CHK-006 Zero-candidate error: 0 candidates produce `No active change.`
- [x] CHK-007 Intake-derived PR body: PR body contains Summary, Changes, and Context sections when intake exists
- [x] CHK-008 PR title derivation: PR title strips `Intake: ` prefix from H1 heading
- [x] CHK-009 Conditional spec link: Context section includes spec link only when `spec.md` exists
- [x] CHK-010 Fallback to --fill: PR creation falls back to `gh pr create --fill` when no change or no intake

## Behavioral Correctness
- [x] CHK-011 Existing fab/current behavior unchanged: When `fab/current` exists and is valid, resolve returns it without triggering guessing
- [x] CHK-012 Downstream propagation: `preflight.sh` and `/git-branch` benefit from guessing automatically without code changes

## Scenario Coverage
- [x] CHK-013 Single active change guess: Verified via end-to-end test with `fab/current` absent
- [x] CHK-014 Active change with intake — rich PR: PR body structure matches spec format
- [x] CHK-015 No active change — fallback: `gh pr create --fill` used when resolution fails

## Edge Cases & Error Handling
- [x] CHK-016 Non-blocking resolution: Resolution failure in git-pr does not block PR creation
- [x] CHK-017 Folder without .status.yaml excluded: Corrupted change folders are not counted as candidates

## Code Quality
- [x] CHK-018 Pattern consistency: New code follows naming and structural patterns of surrounding code in changeman.sh
- [x] CHK-019 No unnecessary duplication: Existing utilities and patterns reused (e.g., archive exclusion pattern)
- [x] CHK-020 Readability: New code is readable and maintainable per code-quality.md principles

## Documentation Accuracy
- [x] CHK-021 Skill file accuracy: git-pr SKILL.md step 3c accurately describes the new behavior

## Cross References
- [x] CHK-022 Intake references consistent: Links and file paths in PR body use correct relative paths

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
