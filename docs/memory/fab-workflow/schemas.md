# Schemas

**Domain**: fab-workflow

## Overview

`$(fab kit-path)/schemas/workflow.yaml` is the single source of truth for the Fab workflow: stages, states, transitions, and validation rules. All scripts and skills query this schema (via `statusman.sh`) rather than hardcoding workflow knowledge.

## What workflow.yaml Defines

1. **States** — All valid progress values (`pending`, `active`, `ready`, `done`, `failed`, `skipped`)
   - Each state has: ID, display symbol, description, terminal flag
   - `ready` means "stage work product exists, eligible for advancement or clarification" (non-terminal)
   - `skipped` means "stage intentionally bypassed" (terminal, symbol `⏭`). Allowed for all stages except intake
   - Terminal states (`done`, `skipped`) cannot transition without explicit reset

2. **Stages** — The workflow pipeline in execution order — 7 stages: `intake`, `spec`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. The legacy `tasks` stage was removed in qszh; plan generation is an apply-internal sub-step that produces `plan.md` (`## Tasks` + `## Acceptance`). `allowedStates` does NOT contain a `tasks` key, and `isValidStage("tasks")` returns false.
   - Each stage has: ID, name, artifact, description, requirements, initial state, allowed states, commands
   - Stages execute in sequence with dependency validation

3. **Transitions** — Valid state changes for each stage, event-keyed (event, from, to)
   - Default rules apply to all stages
   - Stage-specific overrides (e.g., `review` allows `fail` event)
   - Each transition is triggered by an event command (`start`, `advance`, `finish`, `reset`, `fail`, `skip`)
   - `skip` event: `{pending,active} → skipped` with forward cascade (all downstream pending → skipped). No auto-activate
   - `reset` accepts `skipped` as a source state (`skipped → active` with downstream cascade to `pending`)

4. **Progression** — How to navigate the workflow
   - Current stage detection: first `active` or `ready` stage, or first `pending` after last `done`/`skipped`, or `hydrate` if all done/skipped
   - Next stage calculation: first `pending` stage with satisfied dependencies (prerequisites `done` or `skipped`)
   - Completion check: `hydrate` is `done` or `skipped`

5. **Validation** — Rules for `.status.yaml` correctness
   - Exactly 0-1 active stages
   - States must be in `allowed_states` for that stage
   - Prerequisites must be satisfied before activation
   - Terminal states require explicit reset

6. **Stage numbers** — Display numbering for status output (1-indexed positions)

## Referencing from Scripts vs Skills

**In bash scripts**: Invoke `statusman.sh` via CLI subprocess calls:
```bash
STATUSMAN="$(dirname "$(readlink -f "$0")")/statusman.sh"
for stage in $("$STATUSMAN" all-stages); do ...; done
```

**In skills (Claude prompts)**: Reference the schema directly or use bash scripts that call `statusman.sh`:
```markdown
Run `src/kit/scripts/lib/preflight.sh` to get validated stage information.
The script uses `statusman.sh` CLI subcommands internally.
```

For the complete API reference, see `src/lib/statusman/README.md`.

## Design Principles

1. **Single Source of Truth** — One canonical definition, queried by all consumers
2. **Declarative** — Describe *what* the workflow is, not *how* to execute it
3. **Extensible** — Add stages/states/transitions without breaking existing code
4. **Validated** — Schema enforces correctness at runtime
5. **Versionable** — Metadata tracks compatibility and changes

## `.status.yaml` Plan Block (qszh)

Every `.status.yaml` SHALL contain a `plan:` block describing the apply-stage artifact (`plan.md`):

```yaml
plan:
  generated: false      # bool — true after first plan.md write
  task_count: 0         # int — count of - [ ] + - [x] items in ## Tasks section
  acceptance_count: 0   # int — count of - [ ] + - [x] items in ## Acceptance section
  acceptance_completed: 0  # int — count of - [x] items in ## Acceptance section
```

This block replaces the prior `checklist:` block. Field rename: `total → acceptance_count`, `completed → acceptance_completed`. New field: `task_count`. Removed field: `path` (location is fixed at change root).

The `progress` block contains exactly 7 keys (no `tasks` key):

```yaml
progress:
  intake: pending
  spec: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
```

`StageOrder` is `["intake", "spec", "apply", "review", "hydrate", "ship", "review-pr"]` (length 7). `NextStage("spec")` returns `"apply"`. The `set-acceptance` CLI command (`fab status set-acceptance <change> <field> <value>`) updates `plan:` block fields; the legacy `set-checklist` errors immediately with a pointer to `set-acceptance`.

The `Load()` function is tolerant of legacy `.status.yaml` files: it upgrades a `checklist:` block to a `plan:` raw mapping with field migration (`completed → acceptance_completed`, `total → acceptance_count`) and drops `checklist:` when both blocks coexist. The `1.8.0-to-1.9.0.md` migration rewrites in-flight `.status.yaml` files to the new schema (drops `progress.tasks`, replaces `checklist:` with `plan:`); see [migrations.md](migrations.md).

## `.status.yaml` `true_impact` Block (ogf2)

`.status.yaml` MAY contain a top-level optional `true_impact` block that records the merge-base-to-HEAD line-count impact of the change at apply-finish and hydrate-finish. The block is created lazily on first computation — there is no template placeholder, and existing `.status.yaml` files without the block remain valid.

```yaml
true_impact:
    added: 612
    deleted: 38
    net: 574
    excluding:               # only present when true_impact_exclude is non-empty
        added: 540           # raw minus true_impact_exclude → the "total" row
        deleted: 38
        net: 502
    tests:                   # only present when test_paths is non-empty (7t5a)
        added: 400           # test-only lines, within the scaffolding-excluded universe
        deleted: 0
        net: 400
    computed_at: 2026-05-07T14:32:00Z
    computed_at_stage: apply
```

Field semantics:
- `added`, `deleted`, `net` — raw `git diff --shortstat <merge-base>...HEAD` results. `net = added - deleted` (signed).
- `excluding` — same fields with `:(exclude)<pattern>` pathspec applied for each entry in `fab/project/config.yaml` `true_impact_exclude` (sister change asvz; default scaffold `[fab/, docs/]`). Sub-block omitted entirely when `true_impact_exclude` is absent/null/empty (consumer treats "no excludes" identically to "excluding == raw"; emitting a duplicate sub-block adds no signal).
- `tests` — same `added`/`deleted`/`net` fields measured by a **third** `git diff --shortstat <merge-base>...HEAD` pass scoped to the project's `test_paths` patterns (config field, attribution axis — see [configuration.md](configuration.md)). This pass applies BOTH the test-path includes AND the `true_impact_exclude` excludes, so the test count lives strictly **inside the scaffolding-excluded universe** — a test fixture under an excluded path (e.g. `docs/`) is not double-counted (the exclude wins). Include patterns carry `:(glob)` pathspec magic so `**` matches any depth (`**/*_test.go` matches root-level AND nested test files); excludes keep literal matching to preserve `true_impact_exclude`'s directory-prefix behavior. Sub-block **lazily omitted** (`yaml:"tests,omitempty"`) when `test_paths` is absent/null/empty — matching the `excluding` lazy-omit posture; behavior collapses to today's single-number display. Placed after `excluding`, before `computed_at`.
- `computed_at` — RFC 3339 UTC timestamp.
- `computed_at_stage` — pipeline stage at which the snapshot was taken: `apply` or `hydrate`.

**`impl` is NOT stored** — there is no `impl` field in `.status.yaml`, the `fab impact` YAML, or the engine `Result`. The implementation count is the **render-time residual** `impl = max(0, total − tests)`, where `total` is `excluding.net` (else raw `net`) and the clamp is applied **per component** (`added`/`deleted`/`net` independently). The engine (`internal/impact`) is pure measurement — storing only the three measured passes (raw, `excluding`, `tests`); the residual and its clamp are derived by the two render consumers (`/git-pr` PR body assembly and `impactColumn` in `internal/change`), never persisted, so no derived field can drift between the separate diff passes. Each consumer emits a one-line stderr warning when any component clamp triggers (best-effort posture, consistent with `WriteTrueImpact`); a negative `impl` line or component is never rendered.

**Write path**: `WriteTrueImpact(statusPath, base, head, stage)` in `internal/status/true_impact.go` calls `impact.ComputeForRepo` (canonical math in `internal/impact/`) and writes the block via the existing `Save` flow. `status.Finish` invokes the helper for stages `apply` and `hydrate` only — invoked AFTER `applyMetricsSideEffect` and the file save, BEFORE post-hooks. **Best-effort**: on computation failure (e.g., no merge-base resolvable), the helper logs a one-line warning to stderr and returns nil — the stage transition never fails because of a `true_impact` write error. This matches the `fab log command` posture (telemetry hooks never become new failure modes).

**Helper subcommand**: `fab impact <base> <head>` is the canonical CLI for computing the block (consumed by both `WriteTrueImpact` and `/git-pr` Step 3c-impact). It emits the same YAML schema (minus `computed_at_stage` — that is the caller's responsibility) on stdout, exits non-zero with an actionable stderr message on merge-base or `git diff` failure, and reads `true_impact_exclude` and `test_paths` from `fab/project/config.yaml` to apply the same `excluding` and `tests` rules. The `tests` sub-block is emitted only when `test_paths` is non-empty, placed after `excluding` and before `computed_at` (matching the `.status.yaml` field order) so `/git-pr` can parse it via `yq`. See `_cli-fab.md` for the full CLI reference.

## `.status.yaml` Identity Fields

### `id` Field

The `id` field is a top-level field in `.status.yaml` containing the 4-character change ID (the `XXXX` component of the folder name). It is derived from the `name` at creation time and is immutable.

```yaml
id: x2tx
name: 260307-x2tx-status-symlink-pointer
created: 2026-03-07T16:54:29+05:30
```

The `id` field makes the change ID directly available from reading `.status.yaml` without needing to parse the folder name. This is especially useful when reading status via the `.fab-status.yaml` symlink — the consumer gets the ID from the file content rather than having to parse the symlink target path.

### `.fab-status.yaml` Symlink

`.fab-status.yaml` is a symlink at the repository root pointing to the active change's `.status.yaml`. It is the active change pointer — the replacement for the former `fab/current` text file. The symlink target is always a relative path: `fab/changes/{name}/.status.yaml`. See [change-lifecycle.md](change-lifecycle.md) for full lifecycle documentation.

Together with `.fab-runtime.yaml`, these two sibling files at the repo root form the complete ephemeral per-worktree state surface, scannable with a single glob.

## Ephemeral Runtime State

### Agent State — `.fab-runtime.yaml`

Agent runtime state lives in `.fab-runtime.yaml` at the repository root (gitignored). This file is NOT part of the workflow schema (distinct from `workflow.yaml`, which this doc describes), NOT initialized by templates, and NOT read by statusman or any workflow script. It is managed by Claude Code hook scripts via the `fab hook stop|session-start|user-prompt` subcommands.

**Schema and write pipeline**: See [runtime-agents.md](runtime-agents.md) for the authoritative documentation. The file uses a top-level `_agents` map keyed by Claude's `session_id` (UUID from hook stdin) with `change`, `pid`, `tmux_server`, `tmux_pane`, and `transcript_path` as optional entry properties, plus a top-level `last_run_gc` timestamp that throttles an inline GC sweep. Entries populate regardless of active-change state, so agents running in discussion mode are tracked the same as change-associated agents.

Each worktree has its own repo root, so each gets its own `.fab-runtime.yaml` — no cross-worktree contention. External tools can read this file to detect agent idle state and correlate agents to panes without relying on timing heuristics.

## Future Enhancements

1. **Custom workflows** — Allow `fab/project/config.yaml` to override or extend `workflow.yaml`
2. **~~Conditional stages~~** — *(Partially addressed)* The `skipped` state and `skip` event now enable explicit stage bypassing via `statusman.sh skip`. Skill-level orchestration (automatic skip based on change attributes) remains a future enhancement
3. **Parallel stages** — Multiple stages active simultaneously for different artifacts
4. **Stage hooks** — Run scripts before/after stage transitions
5. **State metadata** — Attach timestamps, user info, or exit codes to state transitions

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260530-7t5a-true-impact-test-split | 2026-05-30 | Added optional `tests` sub-block (`added`/`deleted`/`net`) to the `true_impact` block, placed after `excluding` / before `computed_at` and lazily omitted (`yaml:"tests,omitempty"`) when `test_paths` is empty. Computed via a third `git diff --shortstat` pass applying BOTH the `test_paths` includes (with `:(glob)` magic so `**` matches any depth) AND the `true_impact_exclude` excludes — so tests are counted within the scaffolding-excluded universe (no double-counting of fixtures under excluded paths). The engine (`internal/impact`) stays pure measurement (`Result.Tests *Pair`); `impl` is NOT stored anywhere — it is the render-time residual `max(0, total − tests)` (per-component clamp, total = `excluding.net` else raw `net`) derived by the `/git-pr` PR body and `impactColumn` consumers, never persisted. `fab impact` emits `tests` after `excluding`. Migration not needed — old `.status.yaml` files without `tests` parse unchanged (tag-driven decode + omitempty). |
| 260507-ogf2-restrain-ai-code-bloat | 2026-05-07 | Added `.status.yaml` `true_impact` block: optional top-level mapping with `added`/`deleted`/`net`/`computed_at`/`computed_at_stage` (values `apply` or `hydrate`) plus optional `excluding` sub-block (omitted when `true_impact_exclude` is empty). Block is lazily created — no template placeholder; existing files without the block remain valid. Written by `status.Finish` for stages `apply` and `hydrate` via `WriteTrueImpact` (best-effort: stderr warning on failure, never propagates). Canonical math lives in `internal/impact/`; CLI surface is `fab impact <base> <head>`. |
| 260423-qszh-merge-tasks-checklist | 2026-05-06 | `.status.yaml` schema: `progress.tasks` key dropped entirely (no rename). `checklist:` block replaced by `plan:` block with fields `generated`, `task_count`, `acceptance_count`, `acceptance_completed` (rename: `total → acceptance_count`, `completed → acceptance_completed`; new: `task_count`; removed: `path`). Added "`.status.yaml` Plan Block" section documenting the new schema, 7-key progress block, `StageOrder`/`NextStage` updates, `set-acceptance` CLI replacing `set-checklist`, and `Load()` tolerance of legacy files. Updated Stages bullet to note 7-stage pipeline and removal of `tasks` from `allowedStates`/`isValidStage`. |
| 260419-o5ej-agents-runtime-unified | 2026-04-19 | Replaced the in-file `.fab-runtime.yaml` schema description with a cross-reference to the new [runtime-agents.md](runtime-agents.md) (authoritative doc for the `_agents[session_id]` + `last_run_gc` schema, hook write pipeline, GC, grandparent PID walker, and pane-map matching rule). Clarified that `.fab-runtime.yaml` is a distinct schema from `workflow.yaml` — this file documents the latter. |
| 260307-x2tx-status-symlink-pointer | 2026-03-07 | Replaced `fab/current` pointer file with `.fab-status.yaml` symlink at repo root. Added `id` field to `.status.yaml`. Updated resolution, switch, rename, pane-map, hooks, and dispatch. Migration `0.32.0-to-0.34.0` covers conversion. |
| 260306-6bba-redesign-hooks-strategy | 2026-03-06 | Updated Ephemeral Runtime State: `.fab-runtime.yaml` operations now use `fab runtime set-idle` and `fab runtime clear-idle` Go subcommands instead of direct yq manipulation in hooks. |
| 260306-1lwf-extract-agent-runtime-file | 2026-03-06 | Moved agent runtime state from `.status.yaml` to `.fab-runtime.yaml` (repo root, gitignored, keyed by change folder name). Updated Ephemeral Runtime State section accordingly. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added Ephemeral Runtime State section documenting the optional `agent` block (`agent.idle_since` timestamp) managed by Claude Code hooks, not part of workflow schema or templates |
| 260215-lqm5-statusman-cli-only | 2026-02-15 | Updated script example from `source statusman.sh` to CLI subprocess pattern (`$STATUSMAN <subcommand>`) |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_preflight.sh` → `lib/preflight.sh` in skill example; updated `src/statusman/README.md` → `src/lib/statusman/README.md` |
| 260213-jc0u-split-archive-hydrate | 2026-02-13 | Updated progression references: terminal stage from `archive` to `hydrate` |
| 260226-6boq-event-driven-statusman | 2026-02-26 | Transitions are now event-keyed (event, from, to) instead of from→to with conditions. Five event commands: `start`, `advance`, `finish`, `reset`, `fail`. |
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | Added `ready` state (artifact exists, eligible for advancement). Removed unused `skipped` state. Updated transitions (`active→ready`, `ready→done`), progression (current stage includes `ready`), and validation (terminal states: `done` only). |
| 260228-wyhd-add-skipped-stage-state | 2026-02-28 | Added `skipped` state (`⏭`, terminal) and `skip` event (`{pending,active} → skipped` with forward cascade). Updated `reset` to accept `skipped → active`. Updated progression rules to treat `skipped` alongside `done`. Allowed for all stages except intake. Six event commands: `start`, `advance`, `finish`, `reset`, `fail`, `skip`. |
| 260212-4tw0-migrate-scripts-statusman | 2026-02-12 | Moved from `$(fab kit-path)/schemas/README.md`, trimmed statusman API duplication |
