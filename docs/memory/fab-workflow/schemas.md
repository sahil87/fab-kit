# Schemas

**Domain**: fab-workflow

## Overview

`fab/.kit/schemas/workflow.yaml` is the single source of truth for the Fab workflow: stages, states, transitions, and validation rules. All scripts and skills query this schema (via `statusman.sh`) rather than hardcoding workflow knowledge.

## What workflow.yaml Defines

1. **States** — All valid progress values (`pending`, `active`, `ready`, `done`, `failed`, `skipped`)
   - Each state has: ID, display symbol, description, terminal flag
   - `ready` means "stage work product exists, eligible for advancement or clarification" (non-terminal)
   - `skipped` means "stage intentionally bypassed" (terminal, symbol `⏭`). Allowed for all stages except intake
   - Terminal states (`done`, `skipped`) cannot transition without explicit reset

2. **Stages** — The workflow pipeline in execution order
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
Run `fab/.kit/scripts/lib/preflight.sh` to get validated stage information.
The script uses `statusman.sh` CLI subcommands internally.
```

For the complete API reference, see `src/lib/statusman/README.md`.

## Design Principles

1. **Single Source of Truth** — One canonical definition, queried by all consumers
2. **Declarative** — Describe *what* the workflow is, not *how* to execute it
3. **Extensible** — Add stages/states/transitions without breaking existing code
4. **Validated** — Schema enforces correctness at runtime
5. **Versionable** — Metadata tracks compatibility and changes

## Ephemeral Runtime State

### Agent Block — `.fab-runtime.yaml`

Agent runtime state lives in `.fab-runtime.yaml` at the repository root (gitignored). This file is NOT part of the workflow schema, NOT initialized by templates, and NOT read by statusman or any workflow script. It is managed by Claude Code hook scripts in `fab/.kit/hooks/` via `fab runtime` Go subcommands (`fab runtime set-idle <change>`, `fab runtime clear-idle <change>`), which replace direct yq manipulation.

The file is keyed by full change folder name (`YYMMDD-XXXX-slug` format):

```yaml
260306-1lwf-extract-agent-runtime-file:
  agent:
    idle_since: 1741193400    # unix timestamp — set by Stop hook, cleared by SessionStart hook
```

- **Present** (`{change_folder}.agent.idle_since` set): agent is idle (finished its last response turn)
- **Absent** (no `agent` block for that change): agent is active or no hook has run

Each worktree has its own repo root, so each gets its own `.fab-runtime.yaml` — no cross-worktree contention. The file is created with `{}` on first write by `fab runtime set-idle`. External tools (e.g., pipeline orchestrator) can read this file to detect agent idle state without relying on timing heuristics.

## Future Enhancements

1. **Custom workflows** — Allow `fab/project/config.yaml` to override or extend `workflow.yaml`
2. **~~Conditional stages~~** — *(Partially addressed)* The `skipped` state and `skip` event now enable explicit stage bypassing via `statusman.sh skip`. Skill-level orchestration (automatic skip based on change attributes) remains a future enhancement
3. **Parallel stages** — Multiple stages active simultaneously for different artifacts
4. **Stage hooks** — Run scripts before/after stage transitions
5. **State metadata** — Attach timestamps, user info, or exit codes to state transitions

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260306-6bba-redesign-hooks-strategy | 2026-03-06 | Updated Ephemeral Runtime State: `.fab-runtime.yaml` operations now use `fab runtime set-idle` and `fab runtime clear-idle` Go subcommands instead of direct yq manipulation in hooks. |
| 260306-1lwf-extract-agent-runtime-file | 2026-03-06 | Moved agent runtime state from `.status.yaml` to `.fab-runtime.yaml` (repo root, gitignored, keyed by change folder name). Updated Ephemeral Runtime State section accordingly. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added Ephemeral Runtime State section documenting the optional `agent` block (`agent.idle_since` timestamp) managed by Claude Code hooks, not part of workflow schema or templates |
| 260215-lqm5-statusman-cli-only | 2026-02-15 | Updated script example from `source statusman.sh` to CLI subprocess pattern (`$STATUSMAN <subcommand>`) |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_preflight.sh` → `lib/preflight.sh` in skill example; updated `src/statusman/README.md` → `src/lib/statusman/README.md` |
| 260213-jc0u-split-archive-hydrate | 2026-02-13 | Updated progression references: terminal stage from `archive` to `hydrate` |
| 260226-6boq-event-driven-statusman | 2026-02-26 | Transitions are now event-keyed (event, from, to) instead of from→to with conditions. Five event commands: `start`, `advance`, `finish`, `reset`, `fail`. |
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | Added `ready` state (artifact exists, eligible for advancement). Removed unused `skipped` state. Updated transitions (`active→ready`, `ready→done`), progression (current stage includes `ready`), and validation (terminal states: `done` only). |
| 260228-wyhd-add-skipped-stage-state | 2026-02-28 | Added `skipped` state (`⏭`, terminal) and `skip` event (`{pending,active} → skipped` with forward cascade). Updated `reset` to accept `skipped → active`. Updated progression rules to treat `skipped` alongside `done`. Allowed for all stages except intake. Six event commands: `start`, `advance`, `finish`, `reset`, `fail`, `skip`. |
| 260212-4tw0-migrate-scripts-statusman | 2026-02-12 | Moved from `fab/.kit/schemas/README.md`, trimmed statusman API duplication |
