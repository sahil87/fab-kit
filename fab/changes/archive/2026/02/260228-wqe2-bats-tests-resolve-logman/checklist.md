# Quality Checklist: Add BATS Tests for resolve.sh and logman.sh

**Change**: 260228-wqe2-bats-tests-resolve-logman
**Generated**: 2026-02-28
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 resolve.sh output modes: All four modes (--id, --folder, --dir, --status) tested with assertions on exact output format
- [x] CHK-002 resolve.sh input forms: 4-char ID, substring, full folder name, and fab/current fallback all tested
- [x] CHK-003 resolve.sh single-change guess: Fallback tested with .status.yaml present and absent
- [x] CHK-004 resolve.sh error cases: No match, multiple matches, missing fab/changes/, no fab/current with multiple changes — exit codes and stderr message assertions verified
- [x] CHK-005 resolve.sh archive exclusion: Archive directory excluded from resolution
- [x] CHK-006 logman.sh command subcommand: Append behavior, required JSON fields, optional args field tested
- [x] CHK-007 logman.sh confidence subcommand: JSON fields with numeric score verified
- [x] CHK-008 logman.sh review subcommand: Required fields and optional rework field tested
- [x] CHK-009 logman.sh append-only behavior: Existing lines preserved when appending
- [x] CHK-010 logman.sh file creation: .history.jsonl created when absent

## Scenario Coverage
- [x] CHK-011 resolve.sh case-insensitive matching tested
- [x] CHK-012 resolve.sh fab/current with trailing whitespace handled
- [x] CHK-013 logman.sh change resolution via 4-char ID works end-to-end
- [x] CHK-014 logman.sh unresolvable change returns error

## Edge Cases & Error Handling
- [x] CHK-015 resolve.sh --help prints usage with exit 0
- [x] CHK-016 logman.sh no subcommand returns error with "No subcommand" message assertion
- [x] CHK-017 logman.sh unknown subcommand returns error with "Unknown subcommand" message assertion
- [x] CHK-018 logman.sh wrong argument count returns error
- [x] CHK-019 logman.sh --help prints usage with exit 0

## Code Quality
- [x] CHK-020 Pattern consistency: Test files follow setup/teardown isolation pattern from changeman/test.bats and statusman/test.bats
- [x] CHK-021 No unnecessary duplication: Fixture helpers are concise; no reimplementation of test patterns already established
- [x] CHK-022 Readability: Test names are descriptive, following existing @test naming conventions

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
