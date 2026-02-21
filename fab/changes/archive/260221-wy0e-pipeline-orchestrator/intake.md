# Intake: Pipeline Orchestrator

**Change**: 260221-wy0e-pipeline-orchestrator
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Add a pipeline orchestrator that reads a live YAML manifest (fab/pipelines/*.yaml) and executes fab changes in dependency order. The manifest is a shared contract between human and orchestrator — human adds change entries with depends_on declarations while the orchestrator updates stage fields as changes progress.

The idea emerged from a discussion about automating the end-to-end fab workflow across multiple dependent changes. The current workflow requires the developer to manually sequence `/fab-ff`, commit/push/PR, then start the next change. When changes have dependencies (B and C depend on A), the developer must track readiness, create worktrees from the right branches, and sequence everything by hand.

The key design insight: the pipeline manifest is a **live contract** — the human writes new entries while the orchestrator processes earlier ones. No upfront planning required. The human is always ahead of the machine.

## Why

1. **Problem**: Multi-change work with dependencies requires manual sequencing — the developer must wait for A to finish, remember to branch B from A's result, activate the right change, and repeat. This is error-prone and wastes human attention on mechanical coordination.

2. **Consequence**: Without automation, developers either serialize everything manually (slow) or skip the fab pipeline for dependent changes (losing quality gates). Complex features that span 3-5 changes become tedious to drive through the full workflow.

3. **Approach**: A shell-based orchestrator that reads a YAML manifest, resolves the dependency DAG, and dispatches each ready change into its own worktree. This fits the "Pure Prompt Play" constitution principle — shell scripts, no runtime frameworks. The orchestrator delegates all artifact generation to existing skills (`fab-ff`, `changes:ship`) via the `claude` CLI.

## What Changes

### Pipeline manifest format

A new YAML file format stored in `fab/pipelines/`. The file is read on every orchestrator loop iteration, allowing the human to add new entries while earlier changes are running.

```yaml
# fab/pipelines/auth-system.yaml
base: main

changes:
  - id: 260221-a7k2-user-model
    depends_on: []
    stage: done              # ← orchestrator writes this

  - id: 260221-b3m1-auth-endpoints
    depends_on: [260221-a7k2-user-model]
    stage: tasks             # ← orchestrator writes current stage

  - id: 260221-c9p4-sessions
    depends_on: [260221-a7k2-user-model]
                             # no stage yet — not started
```

Fields:
- `base` — the base branch for root nodes (nodes with empty `depends_on`)
- `changes[].id` — the change folder name under `fab/changes/` (must already exist with intake + spec)
- `changes[].depends_on` — list of change IDs that must reach `stage: done` before this change is dispatched
- `changes[].stage` — written by the orchestrator as the change progresses; absent means "not started"

The `stage` field mirrors the change's `.status.yaml` active stage. Terminal values: `done` (hydrate complete + shipped) or `failed`.

### Orchestrator scripts

New directory `fab/.kit/scripts/pipeline/` with three scripts:

**`run.sh <manifest>`** — Main entry point. Reads the manifest, runs the dispatch loop:

```
while true:
  re-read manifest from disk (human may have edited)
  for each change with all deps at stage:done and self has no stage:
    dispatch(change)
  for each running change:
    poll .status.yaml in its worktree
    write current stage back to manifest
  if all changes done or all remaining are blocked/failed:
    break
  sleep interval
```

**`dispatch.sh <change-id> <parent-branch> <manifest>`** — Sets up and runs one change:

1. `wt-create --non-interactive --worktree-open skip <parent-branch>` — creates worktree branched from parent
2. In the worktree: copy `fab/changes/<id>/` artifacts if needed (intake.md, spec.md are in main repo's fab/changes/)
3. `claude -p "/fab-switch <change-id>"` — activate the change
4. `claude -p "/fab-ff"` — execute tasks → apply → review → hydrate (confidence-gated)
5. Commit, push, create PR via `claude -p` with appropriate git/ship commands
6. Write `stage: done` to manifest (or `stage: failed` on error)

**`monitor.sh <worktree-path> <change-id>`** — Polls a running change's `.status.yaml` and returns the current active stage. Used by `run.sh` to update the manifest.

### Worktree and branch strategy

Each change runs in its own worktree for complete filesystem isolation:

```
main
 └── 260221-a7k2-user-model        (A's worktree, branched from main)
      ├── 260221-b3m1-auth-endpoints  (B's worktree, branched from A)
      └── 260221-c9p4-sessions        (C's worktree, branched from A)
```

- Root nodes (empty `depends_on`) branch from `base` (typically `main`)
- Dependent nodes branch from their parent's branch — so B's worktree contains A's hydrated code
- Merging to main is NOT automated — left to the human
- PRs for dependent changes target their parent's branch (stacked PRs)

### Prerequisites

Each change listed in the manifest MUST already have:
- A change folder under `fab/changes/<id>/`
- `intake.md` and `spec.md` generated (human has done the thinking)
- A confidence score computed (spec stage calculates this) above the `fab-ff` gate threshold

The orchestrator validates these before dispatching. Changes that don't meet prerequisites are marked `stage: invalid` with a reason.

### Execution model

Serial execution in v1. The orchestrator processes one change at a time, in topological order. When a node completes, it checks the manifest for newly-unblocked nodes.

Parallel execution (dispatch multiple independent nodes simultaneously) is a stretch goal. Would use background processes with PID tracking.

### Example manifest scaffold

`fab/pipelines/example.yaml` — a commented-out, annotated example manifest that ships with the kit. Created by `/fab-setup` or the pipeline scripts' own scaffold step. Shows the manifest format with explanatory comments covering:

- The `base` field and how root nodes use it
- `depends_on` syntax (empty list for roots, list of IDs for dependents)
- How the orchestrator writes `stage` fields (and what values to expect)
- A multi-level dependency example (A → B → D, A → C, showing diamond DAGs)
- Notes on prerequisites (intake + spec + confidence score required per change)
- The live-editing contract (human adds entries while orchestrator runs)

The file is fully commented out so it doesn't interfere with anything — it's purely documentation-as-code. Developers copy and uncomment to create their own pipeline.

### Byobu integration (stretch)

Optional byobu pane layout for visual monitoring:

```
┌─────────────────────┬─────────────────────┐
│   Orchestrator      │   Change A (wt)     │
│                     │   claude /fab-ff     │
│   [A] ● running     ├─────────────────────┤
│   [B] ○ blocked     │   Change B (wt)     │
│   [C] ○ blocked     │   (waiting...)      │
└─────────────────────┴─────────────────────┘
```

The left pane runs `run.sh` and displays DAG status. Right panes are opened via `wt-open --app byobu_tab` or `tmux split-window` as changes are dispatched. This is additive — the core orchestrator works without byobu (headless mode).

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (new) Documents the pipeline manifest format, orchestrator behavior, dispatch lifecycle, prerequisites, and integration with wt-create/fab-ff/changes:ship

## Impact

- **New files**: `fab/.kit/scripts/pipeline/run.sh`, `dispatch.sh`, `monitor.sh`, `fab/pipelines/example.yaml`
- **New directory**: `fab/pipelines/` (for manifest files, gitignored or committed per project)
- **Dependencies**: `wt-create` (existing), `fab-ff` (existing), `claude` CLI (external), `yq` (existing dependency)
- **No changes to existing skills** — the orchestrator composes existing tools, doesn't modify them
- **Constitution**: Fully compliant — shell scripts, no runtime frameworks, markdown/YAML artifacts

## Open Questions

- How should the orchestrator invoke the `claude` CLI for `fab-ff` and shipping? Options: `claude -p` (single prompt, exits), `claude --print` (non-interactive), or piping commands. The right mode depends on whether `fab-ff` needs multi-turn interaction or can complete in a single invocation.
- Should the manifest support partial dependency (e.g., "B can start once A reaches apply stage" rather than waiting for full completion)? This would enable more parallelism but adds complexity.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Serial execution first, parallel as stretch | User explicitly stated "if its too much complication we can do serial" | S:85 R:90 A:80 D:75 |
| 2 | Certain | YAML manifest with live human/orchestrator read/write | Discussed extensively with concrete examples; user confirmed the shared-contract model | S:90 R:85 A:85 D:85 |
| 3 | Certain | Worktrees branched from parent change's branch, not main | User explicitly: "B and C can be tasks that start off from A's branch" | S:90 R:70 A:80 D:85 |
| 4 | Certain | Use fab-ff, not fab-fff | User explicitly stated with rationale: confidence gating is required for unattended execution | S:95 R:85 A:90 D:90 |
| 5 | Certain | Confidence score required — spec stage is prerequisite | User: "I would not do this unless we have a confidence score" | S:90 R:80 A:85 D:85 |
| 6 | Certain | Merging to main is manual, not automated | User explicitly: "merging to main should be left to the human" | S:95 R:80 A:90 D:90 |
| 7 | Certain | Scripts live in fab/.kit/scripts/pipeline/ | Discussed, fits constitution's pure-prompt-play principle | S:75 R:90 A:85 D:80 |
| 8 | Confident | Completion signal via .status.yaml polling + process exit | Both discussed; polling is more robust but process exit is simpler. Defaulting to .status.yaml | S:60 R:80 A:70 D:50 |
| 9 | Confident | Manifests stored in fab/pipelines/ | Discussed briefly, user didn't object. Location is easily changed | S:65 R:90 A:75 D:70 |
| 10 | Tentative | Race condition handling via append-only convention (no flock) | Discussed both options; user acknowledged but didn't commit. Append-only is simpler and sufficient if human only adds entries and orchestrator only updates stages | S:50 R:70 A:60 D:45 |
<!-- assumed: append-only convention for concurrent manifest access — simpler than flock, sufficient if human adds entries and orchestrator updates stages on separate lines -->
| 11 | Certain | Byobu pane integration as stretch goal, not v1 | User described desired layout but acknowledged complexity; core works headless | S:80 R:90 A:80 D:85 |
| 12 | Certain | Ship a commented-out example.yaml as documentation-as-code | User explicitly requested: "Add a commented out fab/pipelines/example.yaml via scaffold" | S:95 R:95 A:90 D:90 |

12 assumptions (9 certain, 2 confident, 1 tentative, 0 unresolved).
