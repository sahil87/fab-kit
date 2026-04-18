# Checklist: wt open — Support "default" as App Value

**Change**: 260409-5z32-wt-open-default-medium
**Created**: 2026-04-09

## Functional Completeness

- [ ] CHK-001 `ResolveDefaultApp()` returns correct `AppInfo` when default is detected
- [ ] CHK-002 `ResolveDefaultApp()` returns error when no default detected
- [ ] CHK-003 `wt open --app default` resolves and opens via `DetectDefaultApp`
- [ ] CHK-004 `wt open --app default` errors when no default detected
- [ ] CHK-005 `wt create --worktree-open default` resolves and opens via `DetectDefaultApp`
- [ ] CHK-006 `wt create --worktree-open default` warns and continues when no default detected

## Behavioral Correctness

- [ ] CHK-007 `SaveLastApp` is called when `"default"` resolves successfully
- [ ] CHK-008 `"default"` is case-sensitive — `"Default"` falls through to `ResolveApp`
- [ ] CHK-009 `wt open --app default` shows no interactive menu
- [ ] CHK-010 `wt create --worktree-open default` still prints worktree path on no-default

## Edge Cases & Error Handling

- [ ] CHK-011 `DetectDefaultApp` returns -1 (empty app list) — handled gracefully
- [ ] CHK-012 Existing `"prompt"` and `"skip"` behavior in `--worktree-open` unchanged

## Code Quality

- [ ] CHK-013 Pattern consistency — follows existing `ResolveApp`/`DetectDefaultApp` patterns
- [ ] CHK-014 No unnecessary duplication — shared helper used by both callers
- [ ] CHK-015 Readability — new code is self-documenting with clear function name
- [ ] CHK-016 No magic strings — `"default"` keyword used directly (consistent with `"skip"`, `"prompt"`)
