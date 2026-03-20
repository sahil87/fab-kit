# Quality Checklist: Fix Idea Docs & --main Flag

**Change**: 260320-9tqo-fix-idea-docs-main-flag
**Generated**: 2026-03-20
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Remove Backlog Section from `_cli-fab.md`: No `# Backlog` section, no `fab idea` row in Command Reference table
- [ ] CHK-002 Add Idea Section to `_cli-external.md`: `## idea (Backlog Manager)` section present with correct `fab/.kit/bin/idea` invocation
- [ ] CHK-003 `_cli-external.md` frontmatter updated: description field includes `idea`
- [ ] CHK-004 Preserve existing `_cli-external.md` content: `wt`, `tmux`, `/loop` sections unchanged
- [ ] CHK-005 Default resolution uses `--show-toplevel`: `WorktreeRoot()` function exists and is called by default
- [ ] CHK-006 `--main` flag resolves via `--git-common-dir`: `MainRepoRoot()` called when `--main` is set
- [ ] CHK-007 `--main` is a persistent flag: Available to all subcommands
- [ ] CHK-008 Help text updated: Root command `Short` mentions "current worktree" and "--main"
- [ ] CHK-009 `WorktreeRoot()` function added: Uses `git rev-parse --show-toplevel`
- [ ] CHK-010 `GitRepoRoot()` renamed to `MainRepoRoot()`: Old name no longer exists
- [ ] CHK-011 `resolveFile()` branches on `mainFlag`: Calls correct root function based on flag

## Behavioral Correctness
- [ ] CHK-012 Default behavior changed: Without `--main`, `idea` operates on current worktree (not main)
- [ ] CHK-013 `--file` precedence preserved: `--file` > `IDEAS_FILE` > default, root determined by `--main`

## Scenario Coverage
- [ ] CHK-014 `idea list` from linked worktree without `--main`: Reads worktree's own `fab/backlog.md`
- [ ] CHK-015 `idea --main list` from linked worktree: Reads main worktree's `fab/backlog.md`
- [ ] CHK-016 `idea list` from main worktree: Same behavior with or without `--main`
- [ ] CHK-017 `idea --help` output: Contains worktree guidance and `--main` flag

## Edge Cases & Error Handling
- [ ] CHK-018 Not in a git repo: Both `WorktreeRoot()` and `MainRepoRoot()` return meaningful error

## Code Quality
- [ ] CHK-019 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [ ] CHK-020 No unnecessary duplication: `WorktreeRoot()` and `MainRepoRoot()` share error handling pattern

## Documentation Accuracy
- [ ] CHK-021 `_cli-external.md` idea section matches actual CLI behavior (subcommands, flags, output formats)
- [ ] CHK-022 `docs/specs/packages.md` updated with `--main` flag documentation

## Cross References
- [ ] CHK-023 No stale references to `fab idea` as a fab subcommand in skill files
- [ ] CHK-024 No stale references to `GitRepoRoot()` in test files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
