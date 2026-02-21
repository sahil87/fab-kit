# Pipeline Orchestrator

**Domain**: fab-workflow

## Overview

The pipeline orchestrator automates multi-change execution in dependency order. It reads a YAML manifest (`fab/pipelines/*.yaml`), resolves the dependency chain, and dispatches each change into an isolated worktree where `fab-ff` runs the full pipeline (tasks → apply → review → hydrate). The manifest is a live contract — the human adds entries while the orchestrator processes earlier ones.

V1 is serial (one change at a time). The orchestrator runs indefinitely until killed with Ctrl+C.

## Requirements

### Manifest Format

Pipeline manifests are YAML files in `fab/pipelines/` with:
- `base` — branch that root nodes branch from (typically `main`)
- `changes[]` — list of entries, each with `id`, `depends_on`, and optional `stage`

The `stage` field is written by the orchestrator. Valid values: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate` (intermediate), `done`, `failed`, `invalid` (terminal).

`depends_on` supports at most one entry in v1. Multi-parent dependencies require manual merge of parent branches first.

### Orchestrator Core (run.sh)

`run.sh` accepts a manifest path and runs an infinite dispatch loop:

1. Re-reads the manifest from disk (live editing support)
2. Validates: schema, circular deps, single-dep constraint, reference integrity
3. Identifies dispatchable changes (deps all `done`, self not terminal)
4. Dispatches first dispatchable in list order (serial, deterministic)
5. If nothing dispatchable: sleeps 10 seconds (configurable via `PIPELINE_POLL_INTERVAL`), re-reads

**Stage classification for resumability**:
- Terminal (`done`, `failed`, `invalid`) — skip permanently
- Intermediate (pipeline stage names) — re-dispatch into fresh worktree
- Absent — dispatch normally

On SIGINT: prints structured summary (Completed/Failed/Blocked/Skipped/Pending with worktree paths), exits 130.

Output: full Claude output passthrough, `[pipeline]`-prefixed status lines between dispatches.

### Change Dispatch (dispatch.sh)

`dispatch.sh` handles a single change:

1. **Worktree creation** — branch name is `{branch_prefix}{change-id}`. Root nodes: `wt-create --non-interactive --worktree-open skip <change-branch>`. Dependent nodes: `git branch <change-branch> origin/<parent-branch>` first, then `wt-create`.
2. **Artifact provisioning** — copies `fab/changes/<id>/` from source repo to worktree if not present
3. **Prerequisite validation** — intake.md, spec.md, confidence gate. Writes `invalid` on failure.
4. **Pipeline execution** — `claude -p --dangerously-skip-permissions` for fab-switch (with `--no-branch-change`) and fab-ff
5. **Shipping** — `claude -p --dangerously-skip-permissions` for commit/push/PR. PRs target parent's branch (stacked PRs) or `base` for roots.
6. **Stage reporting** — reads terminal `.status.yaml`, writes `done`/`failed` to manifest via `yq`

Infrastructure failures (wt-create, claude, git) abort the orchestrator entirely. Pipeline failures (`fab-ff` non-zero) mark the change `failed` and continue.

Worktrees are left in place after dispatch (success or failure) for manual inspection.

### Example Scaffold

`fab/pipelines/example.yaml` — fully commented-out annotated example covering manifest format, dependency syntax, stage values, prerequisites, and the live-editing contract.

## Scripts

| Script | Location | Purpose |
|--------|----------|---------|
| `run.sh` | `fab/.kit/scripts/pipeline/run.sh` | Main orchestrator loop |
| `dispatch.sh` | `fab/.kit/scripts/pipeline/dispatch.sh` | Per-change dispatch |

## Design Decisions

### Serial Execution in V1
**Decision**: Process one change at a time in topological order.
**Why**: Avoids concurrent worktree management, manifest race conditions, and interleaved output. Parallel is a documented stretch goal.
**Rejected**: Background processes with PID tracking.

### Infinite Loop with SIGINT Exit
**Decision**: run.sh runs indefinitely, polling for new manifest entries.
**Why**: Supports the live-contract model — the human stays ahead, adding entries at their pace. The orchestrator is a daemon-like process, not a batch job.
**Rejected**: Exit when all changes done (prevents the human from adding more entries).

### Claude -p for Pipeline and Shipping
**Decision**: All pipeline steps (fab-switch, fab-ff, shipping) use `claude -p --dangerously-skip-permissions`.
**Why**: Print mode is non-interactive and exits after completion. `--dangerously-skip-permissions` is an explicit opt-in for automated execution. Claude generates contextual commit messages and PR descriptions.
**Rejected**: Direct shell for shipping (generic messages), interactive Claude sessions (not automatable).

### Single-Dependency Restriction
**Decision**: `depends_on` limited to one entry in v1.
**Why**: Multi-parent dependencies create a branch topology problem — D depending on B and C can't branch from both. Resolution requires manual merge.
**Rejected**: Branching from last-completed parent (loses other parent's code).

### Infrastructure Failures Abort
**Decision**: wt-create/claude/git failures abort the orchestrator entirely.
**Why**: Infrastructure failures indicate a broken environment. Continuing would likely fail on subsequent changes.
**Rejected**: Mark individual change failed and continue (masks environment issues).

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260221-wy0e-pipeline-orchestrator | 2026-02-21 | Initial implementation — serial orchestrator with run.sh + dispatch.sh, YAML manifest format, wt-create integration, Claude CLI execution, stacked PRs, SIGINT handling, example scaffold |
