# Tasks: Batch Pipeline Series & Rename

**Change**: 260222-bcfy-batch-pipeline-series-rename
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Rename `fab/.kit/scripts/batch-fab-pipeline.sh` → `fab/.kit/scripts/batch-pipeline.sh` via `git mv`. Update internal script name in frontmatter, comments, and usage text to reference `batch-pipeline.sh`.
- [x] T002 [P] Add `fab/pipelines/.series-*.yaml` pattern to `.gitignore`

## Phase 2: Core Implementation

- [x] T003 Add finite-exit logic to `fab/.kit/scripts/pipeline/run.sh` — read `watch` field from manifest, add all-terminal check in the else branch of the main loop, exit 0 with summary when all terminal and watch is not true
- [x] T004 Replace `origin/` remote branch lookup with local branch ref in `fab/.kit/scripts/pipeline/dispatch.sh` `create_worktree()` — change `git ls-remote` to `git show-ref --verify --quiet "refs/heads/$PARENT_BRANCH"` and remove `origin/` prefix from `git branch` call
- [x] T005 Create `fab/.kit/scripts/batch-pipeline-series.sh` — shell frontmatter, usage/help, argument parsing (`--base`, `-h`/`--help`, positional change IDs), minimum 2 changes validation, manifest generation at `fab/pipelines/.series-{epoch}.yaml`, exec delegation to `pipeline/run.sh`

## Phase 3: Integration & Edge Cases

- [x] T006 Update `fab/.kit/scripts/fab-help.sh` `batch_to_group` mapping — replace `batch-fab-pipeline` key with `batch-pipeline`, add `batch-pipeline-series` key to `"Batch Operations"` group
- [x] T007 Add `watch` field documentation to `fab/pipelines/example.yaml` — commented-out section explaining `watch: true` (infinite) vs default (finite)
- [x] T008 Add BATS tests to `src/scripts/pipeline/test.bats` — test `all_terminal` helper (or inline logic) for the finite-exit check: all done, all failed, mixed terminal, mixed with pending. Update existing `validate_manifest` tests to include manifests with `watch` field (ensure validation still passes).

## Phase 4: Documentation

- [x] T009 [P] Update `docs/memory/fab-workflow/pipeline-orchestrator.md` — rename all `batch-fab-pipeline` references to `batch-pipeline`, document `watch` field and finite-exit behavior, add `batch-pipeline-series.sh` script entry and description, update dispatch.sh local branch ref behavior, update "Infinite Loop" design decision as superseded, add changelog entry
- [x] T010 [P] Update `docs/memory/fab-workflow/kit-architecture.md` — update directory tree listing (`batch-pipeline.sh`, `batch-pipeline-series.sh`), update `batch-fab-pipeline.sh` section description to `batch-pipeline.sh`, add `batch-pipeline-series.sh` description, add changelog entry

---

## Execution Order

- T001 and T002 are independent setup tasks (parallel)
- T003, T004, T005 are independent core implementations (can be done in any order)
- T006 depends on T001 (renamed script must exist for help mapping to be correct)
- T007 is independent
- T008 depends on T003 (finite-exit logic must exist to test)
- T009 and T010 are independent documentation tasks (parallel)
