# Tasks: Operator Spawn Add Fab Sync

**Change**: 260405-xh08-operator-spawn-add-fab-sync
**Spec**: `spec.md`
**Intake**: `intake.md`

<!--
  TASK FORMAT: - [ ] {ID} [{markers}] {Description with file paths}

  Markers (optional, combine as needed):
    [P]   — Parallelizable (different files, no dependencies on other [P] tasks in same group)

  IDs are sequential: T001, T002, ...
  Include exact file paths in descriptions.
  Each task should be completable in one focused session.

  Tasks are grouped by phase. Phases execute sequentially.
  Within a phase, [P] tasks can execute in parallel.
-->

## Phase 1: Core Implementation

<!-- Both source changes are independent — different files, no shared dependency. -->

- [x] T001 [P] In `src/go/wt/internal/worktree/context.go`, change `InitScriptPath()` default return value from `"fab-kit sync"` to `"fab sync"` (line 180).

- [x] T002 [P] In `src/go/wt/internal/worktree/context_test.go`, update `TestInitScriptPath_Default` to assert `"fab sync"` instead of `"fab-kit sync"` (line 74).

- [x] T003 In `src/go/wt/cmd/create.go`, add init script call inside the `--reuse` collision block (lines 179–185), before `return nil`. Introduce `existingWtPath := filepath.Join(ctx.WorktreesDir, finalName)`, gate on `worktreeInit == "true"`, and call `_ = wt.RunWorktreeSetup(existingWtPath, "force", initScript, ctx.RepoRoot)`. Replace the existing `fmt.Println(filepath.Join(ctx.WorktreesDir, finalName))` with `fmt.Println(existingWtPath)`.

## Phase 2: Tests

<!-- T004 depends on T003 being in place so the test can exercise the new code path. -->

- [x] T004 In `src/go/wt/cmd/create_test.go`, add `TestCreate_ReuseRunsInitScript` following the pattern of `TestCreate_InitScriptRuns`: create test repo, call `createInitScript`, commit it, pre-create the target worktree via `createWorktreeViaWt`, then run `wt create --non-interactive --reuse --worktree-name <name>` with `WORKTREE_INIT_SCRIPT=scripts/worktree-init.sh`, and assert `.init-script-ran` exists in the reused worktree path.

## Phase 3: Verification

- [x] T005 Run `go test ./internal/worktree/...` in `src/go/wt/` and confirm `TestInitScriptPath_Default` passes asserting `"fab sync"`.

- [x] T006 Run `go test ./cmd/...` in `src/go/wt/` and confirm `TestCreate_ReuseRunsInitScript` passes, `TestCreate_ReuseExisting` still passes (stdout path unchanged), and `TestCreate_InitScriptRuns` still passes.

---

## Execution Order

- T001 and T002 are independent of each other and of T003 — all three can run in parallel.
- T004 depends on T003 (the new reuse init code must exist before the test can exercise it).
- T005 depends on T001 and T002.
- T006 depends on T003 and T004.
