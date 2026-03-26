# Quality Checklist: Allow idea shorthand

**Change**: 260326-p4ki-allow-idea-shorthand
**Generated**: 2026-03-26
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Default-to-Add: `idea "text"` adds an idea to the backlog
- [x] CHK-002 Persistent Flags: `idea --main "text"` works with the shorthand
- [x] CHK-003 Error on Empty: `idea ""` returns an error
- [x] CHK-004 _cli-external.md updated with shorthand documentation
- [x] CHK-005 packages.md updated with shorthand documentation

## Behavioral Correctness

- [x] CHK-006 No args still shows help (no regression)
- [x] CHK-007 All existing subcommands (`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`) still work

## Scenario Coverage

- [x] CHK-008 Bare shorthand adds idea: `idea "refactor auth"` produces `Added: [xxxx] ...`
- [x] CHK-009 Equivalence: `idea "text"` behaves identically to `idea add "text"`
- [x] CHK-010 Shorthand with --main flag routes to main worktree backlog

## Edge Cases & Error Handling

- [x] CHK-011 First arg matching a subcommand name dispatches to subcommand, not add (e.g., `idea list` still lists)
- [x] CHK-012 Multiple positional args joined with space (e.g., `idea refactor auth middleware` without quotes)

## Code Quality

- [x] CHK-013 Pattern consistency: Root `RunE` follows same error handling patterns as `addCmd`
- [x] CHK-014 No unnecessary duplication: Reuses `resolveFile()` and `idea.Add()` directly

## Documentation Accuracy

- [x] CHK-015 `_cli-external.md` idea section matches actual CLI behavior
- [x] CHK-016 `docs/specs/packages.md` idea section matches actual CLI behavior

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
