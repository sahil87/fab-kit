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

2. **Stages** — The workflow pipeline in execution order — 6 stages: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. The legacy `tasks` stage was removed in qszh, and the `spec` stage in j6cs; plan generation is an apply-internal sub-step that produces a unified `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`). `allowedStates` contains neither a `tasks` nor a `spec` key, and `isValidStage("tasks")`/`isValidStage("spec")` both return false. `validateStage` returns a deprecation error for either removed stage (`spec` → `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`).
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

The `progress` block contains exactly 6 keys (no `tasks` or `spec` key):

```yaml
progress:
  intake: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
```

`StageOrder` is `["intake", "apply", "review", "hydrate", "ship", "review-pr"]` (length 6). `StageNumber("apply") == 2`; `NextStage("intake")` returns `"apply"`. An orphan `progress.spec` key on an un-migrated `.status.yaml` is tolerated on load (`Validate()` skips it; `GetProgressMap()` omits it; a subsequent `Save` may preserve it via raw-node passthrough) — only the `1.9.7-to-1.10.0` migration removes it. The `set-acceptance` CLI command (`fab status set-acceptance <change> <field> <value>`) updates `plan:` block fields; the legacy `set-checklist` errors immediately with a pointer to `set-acceptance`.

The `Load()` function is tolerant of legacy `.status.yaml` files: it upgrades a `checklist:` block to a `plan:` raw mapping with field migration (`completed → acceptance_completed`, `total → acceptance_count`) and drops `checklist:` when both blocks coexist. The `1.8.0-to-1.9.0.md` migration rewrites in-flight `.status.yaml` files to the new schema (drops `progress.tasks`, replaces `checklist:` with `plan:`); the `1.9.7-to-1.10.0.md` migration drops `progress.spec`; see [migrations.md](migrations.md).

As of j6cs the apply-stage `plan.md` carries a `## Requirements` section (RFC-2119 + GIVEN/WHEN/THEN, the requirement discipline absorbed from the removed `spec.md`) alongside `## Tasks` and `## Acceptance` — these three `##` headings are the stable parser contract.

## `.status.yaml` Confidence Block (`indicative` retired in j6cs)

The `confidence` block holds SRAD scoring: `certain`, `confident`, `tentative`, `unresolved` counts and a derived `score` (0.0–5.0). The `confidence.indicative` flag was **retired in j6cs** — `encodeConfidence` no longer writes it, and `SetConfidence`/`SetConfidenceFuzzy` dropped their `indicative` parameter. The struct keeps a decode-tolerant `Indicative *bool` field so a legacy `indicative: true` key on an un-migrated/archived file round-trips harmlessly (load succeeds, the rest of the block decodes normally, and no write re-emits the key). The `--indicative` CLI flag on `set-confidence`/`set-confidence-fuzzy` is retained for one release as an accepted-but-ignored no-op. `fab score` reads `intake.md` only (the sole scoring source); the migration leaves any `confidence.indicative` key on disk untouched.

## `.status.yaml` `true_impact` Block (ogf2)

`.status.yaml` MAY contain a top-level optional `true_impact` block that records the merge-base-to-HEAD line-count impact of the change at apply-finish and hydrate-finish. The block is created lazily on first computation — there is no template placeholder, and existing `.status.yaml` files without the block remain valid.

```yaml
true_impact:
    added: 142
    deleted: 38
    net: 104
    excluding:               # only present when true_impact_exclude is non-empty
        added: 87
        deleted: 38
        net: 49
    computed_at: 2026-05-07T14:32:00Z
    computed_at_stage: apply
```

Field semantics:
- `added`, `deleted`, `net` — raw `git diff --shortstat <merge-base>...HEAD` results. `net = added - deleted` (signed).
- `excluding` — same fields with `:(exclude)<pattern>` pathspec applied for each entry in `fab/project/config.yaml` `true_impact_exclude` (sister change asvz; default scaffold `[fab/, docs/]`). Sub-block omitted entirely when `true_impact_exclude` is absent/null/empty (consumer treats "no excludes" identically to "excluding == raw"; emitting a duplicate sub-block adds no signal).
- `computed_at` — RFC 3339 UTC timestamp.
- `computed_at_stage` — pipeline stage at which the snapshot was taken: `apply` or `hydrate`.

**Write path**: `WriteTrueImpact(statusPath, base, head, stage)` in `internal/status/true_impact.go` calls `impact.ComputeForRepo` (canonical math in `internal/impact/`) and writes the block via the existing `Save` flow. `status.Finish` invokes the helper for stages `apply` and `hydrate` only — invoked AFTER `applyMetricsSideEffect` and the file save, BEFORE post-hooks. **Best-effort**: on computation failure (e.g., no merge-base resolvable), the helper logs a one-line warning to stderr and returns nil — the stage transition never fails because of a `true_impact` write error. This matches the `fab log command` posture (telemetry hooks never become new failure modes).

**Helper subcommand**: `fab impact <base> <head>` is the canonical CLI for computing the block (consumed by both `WriteTrueImpact` and `/git-pr` Step 3c-impact). It emits the same YAML schema (minus `computed_at_stage` — that is the caller's responsibility) on stdout, exits non-zero with an actionable stderr message on merge-base or `git diff` failure, and reads `true_impact_exclude` from `fab/project/config.yaml` to apply the same `excluding` rule. See `_cli-fab.md` for the full CLI reference.

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
| 260601-j6cs-merge-spec-into-apply | 2026-06-01 | `.status.yaml` schema: `progress.spec` key dropped (6-key progress block, no `tasks`/`spec`). `StageOrder` → `["intake", "apply", "review", "hydrate", "ship", "review-pr"]` (length 6); `StageNumber("apply") == 2`; `NextStage("intake") == "apply"`. Orphan `progress.spec` tolerated on load (`Validate()` skips, `GetProgressMap()` omits, raw-node passthrough on Save) — removed only by the `1.9.7-to-1.10.0` migration. `validateStage` returns a deprecation error for the removed `spec` stage (mirroring `tasks`). Added "`.status.yaml` Confidence Block" section: `confidence.indicative` retired (no longer written; decode-tolerant `Indicative *bool` kept; `SetConfidence`/`SetConfidenceFuzzy` dropped the param; `--indicative` flag is a one-release no-op). Noted `plan.md` now carries a `## Requirements` section. Updated the Stages bullet (6 stages, neither `tasks` nor `spec` in `allowedStates`). |
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
