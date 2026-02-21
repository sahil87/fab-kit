# Tasks: Pipeline Orchestrator

**Change**: 260221-wy0e-pipeline-orchestrator
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create directory `fab/.kit/scripts/pipeline/` and boilerplate for `run.sh` and `dispatch.sh` — `set -euo pipefail`, SCRIPT_DIR resolution, usage/help functions, argument parsing stubs
- [x] T002 [P] Create `fab/pipelines/example.yaml` — fully commented-out annotated example covering `base` field, `depends_on` syntax, `stage` values, multi-level dependency example (A → B, A → C chain), prerequisites notes, live-editing contract, single-dep constraint

## Phase 2: Core — dispatch.sh

- [x] T003 Implement `dispatch.sh` worktree creation and artifact provisioning in `fab/.kit/scripts/pipeline/dispatch.sh` — read `config.yaml` for `git.branch_prefix` via `yq`, derive change branch name, create worktree via `wt-create --non-interactive --worktree-open skip` (root: pass change-branch directly; dependent: `git branch <change-branch> origin/<parent-branch>` first), capture worktree path from last stdout line, copy `fab/changes/<id>/` to worktree if not present
- [x] T004 Implement `dispatch.sh` prerequisite validation in `fab/.kit/scripts/pipeline/dispatch.sh` — check `intake.md` exists, `spec.md` exists, confidence gate via `fab/.kit/scripts/lib/calc-score.sh --check-gate`, write `stage: invalid` to manifest via `yq` on failure with reason logged
- [x] T005 Implement `dispatch.sh` pipeline execution and shipping in `fab/.kit/scripts/pipeline/dispatch.sh` — `claude -p --dangerously-skip-permissions "/fab-switch <id> --no-branch-change"`, `claude -p --dangerously-skip-permissions "/fab-ff"`, on success: `claude -p --dangerously-skip-permissions "Commit all changes and create a PR targeting <target-branch>..."`, read terminal `.status.yaml` state, write `done`/`failed` to manifest via `yq`. Infrastructure failures (wt-create, claude, git) exit non-zero to signal abort to run.sh

## Phase 3: Core — run.sh

- [x] T006 Implement `run.sh` manifest parsing and validation in `fab/.kit/scripts/pipeline/run.sh` — accept manifest path argument, validate YAML structure (`base` field, `changes` list, each entry has `id` and `depends_on`), detect circular dependencies, reject multi-parent `depends_on` (>1 entry), validate `depends_on` references exist
- [x] T007 Implement `run.sh` dispatch loop in `fab/.kit/scripts/pipeline/run.sh` — classify stage values (terminal: done/failed/invalid → skip; intermediate → re-dispatch; absent → dispatch), identify dispatchable changes (all deps at `done`, self not terminal), pick first in list order (serial), call `dispatch.sh`, detect infrastructure failure exit code and abort, idle polling with `sleep 10` when no work available, infinite loop
- [x] T008 Implement `run.sh` SIGINT handling and output in `fab/.kit/scripts/pipeline/run.sh` — `trap` SIGINT, print structured summary (Completed/Failed/Blocked/Skipped/Pending counts with ID lists, worktree paths for in-progress/completed), `[pipeline]` prefixed status lines (Dispatching/Completed/Failed/Waiting), exit code 130

---

## Execution Order

- T001 blocks T003, T004, T005, T006, T007, T008 (directory must exist)
- T003 blocks T005 (worktree creation before pipeline execution)
- T004 blocks T005 (validation before execution)
- T005 blocks T007 (dispatch.sh must be complete before run.sh calls it)
- T006 blocks T007 (validation before loop)
- T002 is independent (scaffold file, no code dependency)
