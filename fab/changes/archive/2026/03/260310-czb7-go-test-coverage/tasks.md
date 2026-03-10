# Tasks: Go Test Coverage and Backend Priority

**Change**: 260310-czb7-go-test-coverage
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Restore `test-go` and `test-go-v` justfile targets and add `test-go` to the `test` recipe in `justfile`
- [x] T002 Reverse backend priority in `fab/.kit/bin/fab` — change default from rust > go to go > rust, update `--version` detection order to match

## Phase 2: Core Implementation

- [x] T003 [P] Add `src/go/fab/internal/resolve/resolve_test.go` — test `ToFolder` (symlink, 4-char ID, substring, full name, ambiguous, no match), `ExtractID`, `FabRoot`, `ToDir`, `ToStatus`, `ToAbsDir`, `ToAbsStatus`
- [x] T004 [P] Add `src/go/fab/internal/log/log_test.go` — test `Command`, `Transition`, `Review`, `ConfidenceLog` (event types, timestamps, append-only, optional field omission)
- [x] T005 [P] Add `src/go/fab/internal/preflight/preflight_test.go` — test `Run` (valid repo, missing config, missing constitution, missing active change, override resolution) and `FormatYAML`
- [x] T006 [P] Add `src/go/fab/internal/score/score_test.go` — test `Compute` (all certain, confident penalties, unresolved zero, cover factor, dimension parsing) and `CheckGate` (pass/fail, intake gate)
- [x] T007 [P] Add `src/go/fab/internal/archive/archive_test.go` — test `Archive` (move + index + pointer clear), `Restore` (move back + index removal, with switch), `List`
- [x] T008 [P] Add `src/go/fab/internal/change/change_test.go` — test `New` (valid slug, explicit ID, invalid slug, ID collision), `Rename`, `Switch`, `SwitchBlank`, `List`

## Phase 3: Integration & Edge Cases

- [x] T009 Verify all tests pass via `cd src/go/fab && go test ./... -count=1` — confirm no `[no test files]` for the 6 target packages

---

## Execution Order

- T001 and T002 are independent setup tasks
- T003–T008 are all parallelizable (different packages, no cross-deps in tests)
- T009 depends on T001–T008 (verification pass)
