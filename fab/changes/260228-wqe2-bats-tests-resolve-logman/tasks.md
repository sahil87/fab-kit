# Tasks: Add BATS Tests for resolve.sh and logman.sh

**Change**: 260228-wqe2-bats-tests-resolve-logman
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Create `src/lib/resolve/test.bats` — BATS test suite for `resolve.sh` covering: isolated fixture setup (copy resolve.sh into temp dir), all four output modes (--id, --folder, --dir, --status), input forms (4-char ID, substring, full name, fab/current, trailing whitespace), single-change guess fallback (with and without .status.yaml), error cases (no match, multiple matches, missing fab/changes/, no fab/current with multiple changes), archive exclusion, --help flag
- [x] T002 [P] Create `src/lib/logman/test.bats` — BATS test suite for `logman.sh` covering: isolated fixture setup (copy logman.sh + resolve.sh into temp dir, pre-create change directory), command subcommand (append behavior, JSON fields, optional args), confidence subcommand (JSON fields with numeric score), review subcommand (JSON fields, optional rework), append-only behavior (existing lines preserved), file creation (creates .history.jsonl when absent), error cases (no subcommand, unknown subcommand, wrong arg count), --help flag, change resolution integration (4-char ID, unresolvable change)

## Execution Order

- T001 and T002 are independent — both can be implemented in parallel
