# Quality Checklist: Go wt Binary

**Change**: 260310-qbiq-go-wt-binary
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 RepoContext detection: Returns correct main repo root, repo name, and worktrees dir from both main repo and worktrees
- [x] CHK-002 Random name generation: Produces adjective-noun format, retry logic works on collision
- [x] CHK-003 Branch validation: Rejects all git ref naming rule violations
- [x] CHK-004 Change detection: HasUncommittedChanges, HasUntrackedFiles, HasUnpushedCommits match bash semantics
- [x] CHK-005 Hash-based stash: StashCreate uses git stash create (hash-based), stores in reflog, resets working tree
- [x] CHK-006 Rollback stack: LIFO execution, continues on individual failures, Disarm prevents execution
- [x] CHK-007 wt create: All flags work (--worktree-name, --worktree-init, --worktree-open, --reuse, --non-interactive)
- [x] CHK-008 wt list: Default formatted output, --json, --path modes all work
- [x] CHK-009 wt open: App detection, menu, --app flag, worktree selection all work
- [x] CHK-010 wt delete: All flags, resolution order, stash/confirm/branch cleanup all work
- [x] CHK-011 wt init: Finds and runs init script, graceful handling when missing
- [x] CHK-012 Separate binary: `go build ./cmd/wt` produces a standalone binary

## Behavioral Correctness
- [x] CHK-013 Exit codes match bash: 0-6 exit codes preserved for all error categories
- [x] CHK-014 Error format matches bash: "Error: {what}\n  Why: {why}\n  Fix: {fix}"
- [x] CHK-015 Non-interactive mode: stdout has only path, messages on stderr
- [x] CHK-016 Interactive flows: Menu prompts, dirty-state warnings, confirmations match bash UX

## Removal Verification
- [x] CHK-017 Shell scripts removed: All 6 wt-* scripts and wt-common.sh deleted
- [x] CHK-018 Package directory removed: `fab/.kit/packages/wt/` entirely deleted
- [x] CHK-019 No dead references: Fixed — 6 files updated from `wt-create` to `wt create` (dispatch.sh, batch-fab-new-backlog.sh, batch-fab-switch-change.sh, fab-help.sh, fab-operator1.md, run.sh)

## Scenario Coverage
- [x] CHK-020 Exploratory worktree creation: Random name, new branch, init, open
- [x] CHK-021 Branch worktree creation: Local, remote fetch, and new branch paths
- [x] CHK-022 Reuse worktree: --reuse flag returns existing path
- [x] CHK-023 Delete with stash: Changes stashed before deletion, hash printed
- [x] CHK-024 Delete all worktrees: Batch deletion with confirmation
- [x] CHK-025 List JSON output: Correct JSON structure with all fields
- [x] CHK-026 Path lookup: wt list --path returns absolute path or error

## Edge Cases & Error Handling
- [x] CHK-027 Not a git repo: All commands exit with clear error
- [x] CHK-028 Name collision exhaustion: GenerateUniqueName returns error after maxRetries
- [x] CHK-029 Signal handling: SIGINT/SIGTERM triggers rollback during creation
- [x] CHK-030 Remote branch fetch failure: Clear error message
- [x] CHK-031 NO_COLOR support: Error messages have no ANSI codes when NO_COLOR is set

## Code Quality
- [x] CHK-032 Pattern consistency: Follows existing fab-go patterns (cobra commands, internal packages, error handling)
- [x] CHK-033 No unnecessary duplication: Reuses existing worktree.go list/parse logic
- [x] CHK-034 Readability: Functions ≤50 lines where feasible, clear naming
- [x] CHK-035 No god functions: Complex operations broken into composable helpers

## Documentation Accuracy
- [x] CHK-036 **N/A**: Memory files are updated during hydrate stage (Step 7), not apply/review

## Cross References
- [x] CHK-037 justfile recipes: build-wt, build-wt-all, package-kit updated correctly
- [x] CHK-038 env-packages.sh: No change needed (already adds kit/bin to PATH — verified)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
