# Tasks: Add --base flag to wt create

**Change**: 260312-wrk6-add-wt-create-base-flag
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `startPoint` parameter to `CreateWorktree` in `src/go/wt/internal/worktree/crud.go` — when `startPoint` is non-empty and `newBranch` is true, append the start-point to the git command; update all callers
- [x] T002 Add `startPoint` parameter to `CreateBranchWorktree` in `src/go/wt/internal/worktree/crud.go` — pass through to `CreateWorktree` only for the new-branch path (empty string for existing local/remote branches)
- [x] T003 Add `startPoint` parameter to `CreateExploratoryWorktree` in `src/go/wt/internal/worktree/crud.go` — pass through to `CreateWorktree`

## Phase 2: CLI Integration

- [x] T004 Add `--base` flag to `createCmd()` in `src/go/wt/cmd/create.go` — string flag bound to a `base` variable
- [x] T005 Add `--base` ref validation in `create.go` — when `base` is non-empty and will be used (new branch), validate with `git rev-parse --verify <base>` before creating the worktree; error with clear message on failure
- [x] T006 Add `--base` warn-and-ignore logic in `create.go` — when calling `CreateBranchWorktree` with an existing local/remote branch and `base` is non-empty, print warning to stderr; pass empty startPoint to the function
- [x] T007 Wire `base` through to `CreateBranchWorktree` and `CreateExploratoryWorktree` calls in `create.go`

## Phase 3: Tests

- [x] T008 [P] Add `TestCreate_BaseNewBranch` in `src/go/wt/cmd/create_test.go` — create a branch with `--base <other-branch>`, verify marker file present and HEAD matches base branch tip
- [x] T009 [P] Add `TestCreate_BaseExploratoryWorktree` in `src/go/wt/cmd/create_test.go` — create exploratory worktree with `--base`, verify it branches from base tip
- [x] T010 [P] Add `TestCreate_BaseWithExistingLocalBranch` in `src/go/wt/cmd/create_test.go` — pass `--base` with existing local branch, verify warning on stderr and branch unchanged
- [x] T011 [P] Add `TestCreate_BaseWithExistingRemoteBranch` in `src/go/wt/cmd/create_test.go` — pass `--base` with remote branch, verify warning on stderr
- [x] T012 [P] Add `TestCreate_BaseInvalidRef` in `src/go/wt/cmd/create_test.go` — pass invalid `--base` ref, verify error exit and no worktree created
- [x] T013 [P] Add `TestCreate_BaseWithReuse` in `src/go/wt/cmd/create_test.go` — pass `--base` with `--reuse` on existing worktree, verify reuse takes precedence
- [x] T014 [P] Add `TestCreate_BaseDoesNotAffectExistingBehavior` in `src/go/wt/cmd/create_test.go` — verify creating a new branch without `--base` still branches from HEAD

## Phase 4: Documentation & Skill Updates

- [x] T015 [P] Update `docs/specs/packages.md` — add `--base` flag to wt create documentation with behavior table
- [x] T016 [P] Update `fab/.kit/skills/fab-operator2.md` — modify autopilot per-change loop to use `--base` for user-provided ordering

---

## Execution Order

- T001 blocks T002, T003 (callers depend on updated signature)
- T002, T003 block T004-T007 (CLI needs the updated functions)
- T004-T007 block T008-T014 (tests need the flag to exist)
- T015, T016 are independent of all other tasks
