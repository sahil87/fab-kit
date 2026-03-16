# Quality Checklist: Expand wt name lists and fix wt list output

**Change**: 260316-ez49-expand-wt-name-lists
**Generated**: 2026-03-16
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Adjective list expansion: `len(adjectives)` >= 120
- [x] CHK-002 Noun list expansion: `len(nouns)` >= 120
- [x] CHK-003 Comment accuracy: Comments above both lists reflect actual counts
- [x] CHK-004 Separator removal: `wt list` formatted output has no dash separator row
- [x] CHK-005 No logic changes: `GenerateRandomName()` and `GenerateUniqueName()` function bodies unchanged

## Behavioral Correctness
- [x] CHK-006 Formatted output: Header row followed directly by data rows, columns still align
- [x] CHK-007 JSON output: `wt list --json` unchanged
- [x] CHK-008 Path output: `wt list --path` unchanged

## Scenario Coverage
- [x] CHK-009 Test validates expanded list thresholds (≥100)
- [x] CHK-010 Test validates header exists without dash separator assertion
- [x] CHK-011 All existing tests pass (`go test ./...`) — 4 pre-existing init test failures unrelated to this change

## Edge Cases & Error Handling
- [x] CHK-012 No duplicate entries in adjective list
- [x] CHK-013 No duplicate entries in noun list
- [x] CHK-014 All adjectives are lowercase, ≤7 chars preferred (1 at 8 chars: "polished")
- [x] CHK-015 All nouns are real animals, lowercase — "sphinx" replaced with "earwig" per review

## Code Quality
- [x] CHK-016 Pattern consistency: New word entries follow the same formatting style (8 per line, quoted, comma-separated)
- [x] CHK-017 No unnecessary duplication: No near-synonyms in either list

## Documentation Accuracy
- [x] CHK-018 **N/A** Packages spec: Word list sizes not mentioned in specs or memory — no update needed

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
