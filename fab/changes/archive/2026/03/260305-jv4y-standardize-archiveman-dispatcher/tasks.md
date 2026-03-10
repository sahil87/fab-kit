# Tasks: Standardize archiveman.sh Dispatcher Integration

**Change**: 260305-jv4y-standardize-archiveman-dispatcher
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Change the `*` case in `fab/.kit/scripts/lib/archiveman.sh` (line ~418) from error to `cmd_archive "$@"` (default-to-archive fallback)
- [x] T002 [P] Remove the hardcoded `"archive"` argument from the `archive)` case in `fab/.kit/bin/fab` (line 47)

## Phase 2: Tests

- [x] T003 Update the bash side of `src/go/fab/test/parity/archive_test.go` "archive change" test to invoke `archiveman.sh` without the explicit `archive` subcommand, exercising the default-to-archive path
- [x] T004 Run bats tests (`src/lib/archiveman/test.bats`) to confirm they still pass (updated test 40 for new default-to-archive behavior)
- [x] T005 Run parity tests (`src/go/fab/test/parity/`) to confirm bash/Go output matches

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003 depends on T001 (needs the default-to-archive fallback in place)
- T004 depends on T001
- T005 depends on T001, T002, and T003
