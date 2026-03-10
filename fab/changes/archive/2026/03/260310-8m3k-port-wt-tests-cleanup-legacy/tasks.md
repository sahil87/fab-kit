# Tasks: Port wt Tests & Cleanup Legacy

**Change**: 260310-8m3k-port-wt-tests-cleanup-legacy
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create test helper utilities in `src/go/wt/cmd/testutil_test.go` — shared functions for creating temp git repos, creating worktrees via the CLI, asserting worktree/branch existence, cleanup. Mirrors the patterns from `src/packages/wt/tests/test_helper.bash`

## Phase 2: Core Implementation — Test Porting

- [x] T002 Port init tests to `src/go/wt/cmd/init_test.go` — cover script execution, missing script guidance, not-in-git-repo error, idempotency. Derive from `src/packages/wt/tests/wt-init.bats`
- [x] T003 [P] Port create tests to `src/go/wt/cmd/create_test.go` — cover exploratory creation, branch-based creation (local/remote/new), --worktree-name, name collision, --reuse flag, init script integration, porcelain output (stdout=path, stderr=messages), branch-off-HEAD behavior. Derive from `src/packages/wt/tests/wt-create.bats`
- [x] T004 [P] Port delete tests to `src/go/wt/cmd/delete_test.go` — cover delete by name, branch cleanup (--delete-branch, --delete-remote), --stash flag, --delete-all, discard in non-interactive, error for nonexistent. Derive from `src/packages/wt/tests/wt-delete.bats`
- [x] T005 [P] Port list tests to `src/go/wt/cmd/list_test.go` — cover formatted output (repo name, total count), --path flag, --json flag with all fields, mutual exclusivity, dirty/unpushed indicators, NO_COLOR support. Derive from `src/packages/wt/tests/wt-list.bats`
- [x] T006 [P] Port open tests to `src/go/wt/cmd/open_test.go` — cover opening by name/path, error for nonexistent worktree, error for unknown app, error from main repo without target. Derive from `src/packages/wt/tests/wt-open.bats`

## Phase 3: Integration & Edge Cases

- [x] T007 Port edge case tests to `src/go/wt/cmd/edge_test.go` — cover corrupted state (external rm + prune), invalid branch name cleanup, detached HEAD, multiple unique names, name collision errors. Derive from `src/packages/wt/tests/edge-cases.bats`
- [x] T008 Port integration tests to `src/go/wt/cmd/integration_test.go` — cover full lifecycle (create→list→delete), create-multiple→delete-all, non-interactive automation, git state integrity. Derive from `src/packages/wt/tests/integration.bats`
- [x] T009 Port any missing common utility tests — check `src/packages/wt/tests/wt-common.bats` against existing `src/go/wt/internal/worktree/*_test.go` and add any uncovered behaviors (hash-based stash, branch validation edge cases)

## Phase 4: Cleanup

- [x] T010 Remove `src/packages/` directory (includes wt shell package, rc-init.sh, and tests subdir)
- [x] T011 [P] Remove `src/tests/` directory (bats submodule libs)
- [x] T012 [P] Remove `.gitmodules` file and clean up git submodule references
- [x] T013 Search for and update any references to `src/packages/`, `src/tests/`, or bats in docs, scripts, and config files
- [x] T014 Run `go test ./...` in `src/go/wt/` to verify all ported tests pass

---

## Execution Order

- T001 blocks T002-T009 (test helpers needed first)
- T003-T006 are parallelizable (independent command tests)
- T007-T008 depend on T001 but are independent of each other
- T010-T012 are parallelizable and depend on T009 (port complete before removing source)
- T013 depends on T010-T012 (references checked after removal)
- T014 depends on T002-T009 (tests must exist before running)
