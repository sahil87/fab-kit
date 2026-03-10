# Tasks: Clean Break — Go Only

**Change**: 260305-u8t9-clean-break-go-only
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Removals

- [x] T001 [P] Delete 7 ported shell scripts from `fab/.kit/scripts/lib/`: `statusman.sh`, `changeman.sh`, `archiveman.sh`, `logman.sh`, `calc-score.sh`, `preflight.sh`, `resolve.sh`
- [x] T002 [P] Delete `fab/.kit/packages/wt/bin/wt-status`
- [x] T003 [P] Delete `src/packages/wt/tests/wt-status.bats`

## Phase 2: Dispatcher & PATH

- [x] T004 Update `fab/.kit/bin/fab` — remove shell fallback case block (lines 30–49), remove `LIB_DIR`, change `--version` backend from `"shell"` to `"none"` when no binary found, add error message when no backend
- [x] T005 Update `fab/.kit/scripts/lib/env-packages.sh` — add `$KIT_DIR/bin` to PATH before the packages loop

## Phase 3: Go Binary — fab status show

- [x] T006 Add `fab status show` subcommand to `src/go/fab/cmd/fab/status.go` — implement worktree discovery via `git worktree list --porcelain`, fab state resolution using existing `internal/resolve` and `internal/statusfile` packages, support `--all`, `--json`, and `[<name>]` arguments
- [x] T007 Add Go implementation for worktree discovery and fab state resolution in `src/go/fab/internal/` (new package or extend existing)

## Phase 4: Pipeline, Tests & Documentation

- [x] T008 Update `fab/.kit/scripts/pipeline/dispatch.sh` `validate_prerequisites` function — replace direct `calc-score.sh` path check and invocation with `fab score --check-gate <change-id>` via worktree's `fab/.kit/bin/fab`
- [x] T009 Update `fab/.kit/skills/_scripts.md` — remove shell fallback references, update Backend Priority to `rust > go > error`, note Cobra help support, remove `[fab] using shell backend` reference
- [x] T010 Update `src/go/fab/test/parity/parity_test.go` `runBash` function — add skip guard that checks if the target bash script exists in `repoRoot/fab/.kit/scripts/lib/` before copying; when missing, return a sentinel that causes the calling test to skip with `t.Skipf("bash script %s not found — shell scripts removed")`

---

## Execution Order

- T001, T002, T003 are independent (all [P]) — can run in parallel
- T004 depends on T001 (dispatcher references lib/ scripts)
- T005 is independent of T004
- T006 depends on T007 (command wires up the implementation)
- T008 is independent of T006/T007
- T009 is independent
- T010 depends on T001 (skip guard only relevant after scripts deleted)
