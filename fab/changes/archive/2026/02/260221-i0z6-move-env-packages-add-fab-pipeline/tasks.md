# Tasks: Move env-packages.sh to lib & Add fab-pipeline.sh Entry Point

**Change**: 260221-i0z6-move-env-packages-add-fab-pipeline
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: File Move & Reference Updates

- [x] T001 Move `fab/.kit/scripts/env-packages.sh` to `fab/.kit/scripts/lib/env-packages.sh` and update `KIT_DIR` from `"$SCRIPT_DIR/.."` to `"$SCRIPT_DIR/../.."`
- [x] T002 [P] Update `fab/.kit/scaffold/fragment-.envrc` line 4: change `source fab/.kit/scripts/env-packages.sh` to `source fab/.kit/scripts/lib/env-packages.sh`
- [x] T003 [P] Update `src/packages/rc-init.sh` line 14: change `source "$SCRIPT_DIR/../../fab/.kit/scripts/env-packages.sh"` to `source "$SCRIPT_DIR/../../fab/.kit/scripts/lib/env-packages.sh"`

## Phase 2: fab-pipeline.sh Wrapper

- [x] T004 Create `fab/.kit/scripts/fab-pipeline.sh` with: no-args/--list listing (exclude `example.yaml`), -h/--help usage, partial name matching (case-insensitive substring, error on ambiguity/no match), explicit path bypass (contains `/` or ends `.yaml`), `exec` delegation to `pipeline/run.sh` with arg passthrough. Make executable.

## Phase 3: Pipeline Script Improvements

- [x] T005 Update `fab/.kit/scripts/pipeline/run.sh` to resolve manifest change IDs through `changeman resolve` before dispatching — pass resolved full name to `dispatch.sh` instead of raw manifest ID
- [x] T006 [P] Update `fab/.kit/scripts/pipeline/dispatch.sh` `create_worktree()`: add `--worktree-name "$CHANGE_ID"` to the `wt-create` invocation
- [x] T007 [P] Update `fab/.kit/scripts/pipeline/dispatch.sh` `run_pipeline()`: replace raw `yq '.progress.hydrate'` check with `stageman display-stage` to determine stage reached; write `done`/`failed` based on stageman output

## Phase 4: Documentation Updates

- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md`: move `env-packages.sh` from `scripts/` to `lib/` in directory tree, add `fab-pipeline.sh` to `scripts/` listing, update `env-packages.sh` description section path, add `fab-pipeline.sh` description section
- [x] T009 [P] Update `docs/memory/fab-workflow/distribution.md`: change `env-packages.sh` path references to `fab/.kit/scripts/lib/env-packages.sh`
- [x] T010 [P] Update `README.md` Packages > Setup section: change `fab/.kit/scripts/env-packages.sh` reference to `fab/.kit/scripts/lib/env-packages.sh`
- [x] T011 [P] Update `docs/memory/fab-workflow/pipeline-orchestrator.md`: add `fab-pipeline.sh` entry point docs, document changeman resolve for manifest IDs, document `--worktree-name` in dispatch.sh, document stageman-based stage detection

---

## Execution Order

- T001 blocks T002, T003 (file must exist at new path before references update)
- T004 is independent
- T005, T006, T007 are independent of each other
- T008-T011 are independent and parallelizable
