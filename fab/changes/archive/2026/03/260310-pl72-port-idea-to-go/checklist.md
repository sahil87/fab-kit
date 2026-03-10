# Quality Checklist: Port Idea Script to Go Binary

**Change**: 260310-pl72-port-idea-to-go
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Idea struct: `Idea` struct with ID, Date, Text, Done fields exists in `internal/idea/`
- [x] CHK-002 Parse/format: ParseLine and FormatLine correctly round-trip valid idea lines
- [x] CHK-003 Query matching: case-insensitive substring search on ID and text fields
- [x] CHK-004 Load/Save: file operations preserve non-idea lines (headers, blanks)
- [x] CHK-005 Add: appends idea with generated or custom ID/date, auto-creates file
- [x] CHK-006 List: filters by open/done/all, supports --json, --sort, --reverse
- [x] CHK-007 Show: finds single match, displays line or JSON, errors on 0 or >1 matches
- [x] CHK-008 Done/Reopen: toggles status for single match with correct filter
- [x] CHK-009 Edit: modifies text, optional --id (collision check) and --date
- [x] CHK-010 Rm: deletes single match, requires --force (no interactive prompt)
- [x] CHK-011 Cobra registration: `fab idea` with all subcommands in main.go
- [x] CHK-012 Git root resolution: `git rev-parse` exec for repo root
- [x] CHK-013 File override: --file flag > IDEAS_FILE env > default fab/backlog.md

## Behavioral Correctness
- [x] CHK-014 Output messages match Bash format: Added/Done/Reopened/Updated/Removed
- [x] CHK-015 JSON output shape matches: `{"id","date","status","text"}`
- [x] CHK-016 ID collision detected and reported for add and edit
- [x] CHK-017 Multiple match disambiguation shows matching lines with guidance
- [x] CHK-018 Missing file: list returns friendly message, add auto-creates

## Scenario Coverage
- [x] CHK-019 Parse valid open line → correct Idea struct
- [x] CHK-020 Parse valid done line → correct Idea struct
- [x] CHK-021 Parse invalid line → error/skip
- [x] CHK-022 Query match by ID, by text, case-insensitive, multiple, none
- [x] CHK-023 List open (default), done, all filters
- [x] CHK-024 List with --sort id --reverse
- [x] CHK-025 List empty/missing file
- [x] CHK-026 Show single match, multiple matches, no matches
- [x] CHK-027 Done on open idea, done on already-done (error)
- [x] CHK-028 Reopen done idea, reopen already-open (error)
- [x] CHK-029 Edit text only, edit with --id, edit with --id collision
- [x] CHK-030 Rm with --force, rm without --force (error)
- [x] CHK-031 Add with defaults, with custom ID/date, with ID collision
- [x] CHK-032 **N/A** Run outside git repo → error: not unit-testable without mocking exec; GitRepoRoot returns correct error string

## Edge Cases & Error Handling
- [x] CHK-033 Empty backlog file: load returns zero ideas, no error
- [x] CHK-034 Missing file auto-create on add: file and parent dirs created
- [x] CHK-035 Not in git repo: clear error message
- [x] CHK-036 ID collision on add and edit: error with specific message

## Code Quality
- [x] CHK-037 Pattern consistency: follows internal/ package layout, Cobra wiring, and test patterns matching existing code
- [x] CHK-038 No unnecessary duplication: reuses resolve.FabRoot or equivalent, follows existing cmd patterns
- [x] CHK-039 Readability: functions are focused, no god functions >50 lines without clear reason
- [x] CHK-040 No magic strings: constants or named values for format patterns

## Documentation Accuracy
- [x] CHK-041 `_scripts.md` updated if new CLI command signatures added

## Cross References
- [x] CHK-042 **N/A** Memory files (kit-architecture, distribution) reference idea as Go subcommand: hydrate-stage concern

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
