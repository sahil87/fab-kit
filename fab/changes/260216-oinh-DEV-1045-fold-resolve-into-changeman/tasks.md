# Tasks: Fold resolve-change into changeman

**Change**: 260216-oinh-DEV-1045-fold-resolve-into-changeman
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `resolve` subcommand to `fab/.kit/scripts/lib/changeman.sh` — port `resolve_change()` logic from `resolve-change.sh` as `cmd_resolve()`. Two modes: no-arg reads `fab/current` (strip whitespace, print to stdout); with override does case-insensitive substring match against `fab/changes/` (exclude archive). Exit 0 with name on stdout, exit 1 with diagnostic on stderr. Add `resolve` to CLI dispatch.
- [x] T002 Add `switch` subcommand to `fab/.kit/scripts/lib/changeman.sh` — `cmd_switch()` composes: (1) resolve name via `cmd_resolve`, (2) write `fab/current`, (3) read `config.yaml` via `yq` for `git.enabled`/`git.branch_prefix` with defaults, (4) git branch checkout/create, (5) derive stage via `$STAGEMAN current-stage`, (6) output structured summary. Handle `--blank` for deactivation. Add `switch` to CLI dispatch and help text.

## Phase 2: Caller Migration

- [x] T003 [P] Migrate `fab/.kit/scripts/lib/preflight.sh` — remove `source resolve-change.sh`, add `CHANGEMAN` variable, replace `resolve_change "$fab_root" "$override"` / `$RESOLVED_CHANGE_NAME` with `name=$("$CHANGEMAN" resolve "$override")`. Preserve guidance messages on failure.
- [x] T004 [P] Migrate `fab/.kit/scripts/batch-fab-switch-change.sh` — remove `source resolve-change.sh`, add `CHANGEMAN` variable, replace `resolve_change "$FAB_DIR" "$change"` / `$RESOLVED_CHANGE_NAME` with `match=$("$CHANGEMAN" resolve "$change")`.
- [x] T005 [P] Migrate `fab/.kit/scripts/batch-fab-archive-change.sh` — same pattern as T004.

## Phase 3: Cleanup & Tests

- [x] T006 Delete `fab/.kit/scripts/lib/resolve-change.sh`. Verify no remaining references via `grep -r resolve-change fab/.kit/`.
- [x] T007 Migrate resolve-change tests into `src/lib/changeman/test.bats` — adapt from sourcing `resolve_change()` to invoking `changeman.sh resolve`. Add `switch` tests (normal, blank, git integration, output format). Run full test suite.
- [x] T008 Update `src/lib/preflight/test.bats` — remove the `cp resolve-change.sh` line from setup, add `cp changeman.sh` instead. Verify all preflight tests pass.
- [x] T009 [P] Update `src/lib/changeman/SPEC-changeman.md` — add `resolve` and `switch` subcommand docs, update requirements (add yq dependency), update usage examples.
- [x] T010 [P] Update `fab/.kit/skills/fab-switch.md` — simplify Argument Flow to delegate to `changeman.sh switch`, update Context Loading (remove resolve-change.sh reference), keep No Argument Flow and multi-match interactive selection in skill layer.
- [x] T011 [P] Delete `src/lib/resolve-change/` dev directory (test.bats, test-simple.sh, SPEC-resolve-change.md, any symlinks).

---

## Execution Order

- T001 blocks T002 (switch calls resolve internally)
- T001 blocks T003, T004, T005 (callers need resolve subcommand)
- T002 blocks T010 (skill references switch subcommand)
- T003, T004, T005 block T006 (all callers migrated before deletion)
- T006 blocks T007, T008 (tests updated after old file removed)
- T007 blocks T011 (old dev dir deleted after tests migrated)
