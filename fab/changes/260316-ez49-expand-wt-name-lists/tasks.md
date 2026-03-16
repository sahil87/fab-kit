# Tasks: Expand wt name lists and fix wt list output

**Change**: 260316-ez49-expand-wt-name-lists
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Expand `adjectives` slice in `src/go/wt/internal/worktree/names.go` from 48 to ≥120 entries. Update the comment from `~50` to the actual count. Keep existing entries, add new ones grouped thematically.
- [x] T002 [P] Expand `nouns` slice in `src/go/wt/internal/worktree/names.go` from 48 to ≥120 entries. Update the comment from `~50` to the actual count. Keep existing entries, add new animals grouped by family.
- [x] T003 [P] Remove the separator row (lines 211-216) from `handleFormattedOutput` in `src/go/wt/cmd/list.go`. Delete the `fmt.Printf` call that prints `strings.Repeat("-", ...)`.

## Phase 2: Test Updates

- [x] T004 [P] Update `TestWordListsNonEmpty` in `src/go/wt/internal/worktree/names_test.go` — change minimum thresholds from 40 to 100 for both lists. Update the error message from `~50` to `~120`.
- [x] T005 [P] Update `TestList_HeaderAndSeparator` in `src/go/wt/cmd/list_test.go` — remove the `assertContains(t, r.Stdout, "----")` assertion. Rename test to `TestList_Header` to reflect it no longer checks separators.

## Phase 3: Validation

- [x] T006 Run `go test ./...` from `src/go/wt/` to verify all tests pass. (4 pre-existing init test failures, no new failures)

---

## Execution Order

- T001, T002, T003 are independent (different files or different sections)
- T004 depends on T001/T002 (tests validate the expanded lists)
- T005 depends on T003 (test validates the removed separator)
- T006 depends on all prior tasks
