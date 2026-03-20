# Tasks: Fix Idea Docs & --main Flag

**Change**: 260320-9tqo-fix-idea-docs-main-flag
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Go Changes

- [x] T001 Rename `GitRepoRoot()` to `MainRepoRoot()` and add `WorktreeRoot()` in `src/go/idea/internal/idea/idea.go` — `WorktreeRoot()` uses `git rev-parse --show-toplevel`
- [x] T002 Add `--main` persistent bool flag to root command in `src/go/idea/cmd/main.go` — update `Short` description to include worktree guidance
- [x] T003 Update `resolveFile()` in `src/go/idea/cmd/resolve.go` to branch on `mainFlag`: call `MainRepoRoot()` when true, `WorktreeRoot()` when false

## Phase 2: Tests

- [x] T004 Update `src/go/idea/internal/idea/idea_test.go` — rename test references from `GitRepoRoot` to `MainRepoRoot`, add tests for `WorktreeRoot()`

## Phase 3: Documentation Changes

- [x] T005 [P] Remove `# Backlog` section and `fab idea` row from Command Reference table in `fab/.kit/skills/_cli-fab.md`
<!-- clarified: removed fragile line number reference "lines 334–371" — use section heading instead -->
- [x] T006 [P] Add `## idea (Backlog Manager)` section to `fab/.kit/skills/_cli-external.md` after the `wt` section — document as standalone binary with `--main` flag, update frontmatter description
- [x] T007 [P] Update `docs/specs/packages.md` idea section to document `--main` flag and worktree-default behavior
<!-- clarified: added missing task from intake Impact section — packages.md documents the idea binary but doesn't mention --main -->

---

## Execution Order

- T001 blocks T003 (resolve.go depends on renamed/new functions)
- T002 blocks T003 (resolve.go reads `mainFlag` declared in main.go)
- T003 blocks T004 (tests validate the integrated behavior)
- T005, T006, and T007 are independent of each other and of T001–T004
