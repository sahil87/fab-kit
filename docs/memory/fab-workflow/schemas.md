# Schemas

**Domain**: fab-workflow

## Overview

`fab/.kit/schemas/workflow.yaml` is the single source of truth for the Fab workflow: stages, states, transitions, and validation rules. All scripts and skills query this schema (via `statusman.sh`) rather than hardcoding workflow knowledge.

## What workflow.yaml Defines

1. **States** — All valid progress values (`pending`, `active`, `ready`, `done`, `failed`)
   - Each state has: ID, display symbol, description, terminal flag
   - `ready` means "stage work product exists, eligible for advancement or clarification" (non-terminal)
   - Terminal states (`done`) cannot transition without explicit reset

2. **Stages** — The workflow pipeline in execution order
   - Each stage has: ID, name, artifact, description, requirements, initial state, allowed states, commands
   - Stages execute in sequence with dependency validation

3. **Transitions** — Valid state changes for each stage, event-keyed (event, from, to)
   - Default rules apply to all stages
   - Stage-specific overrides (e.g., `review` allows `fail` event)
   - Each transition is triggered by an event command (`start`, `advance`, `finish`, `reset`, `fail`)

4. **Progression** — How to navigate the workflow
   - Current stage detection: first `active` or `ready` stage, or `hydrate` if all done
   - Next stage calculation: first `pending` stage with satisfied dependencies
   - Completion check: `hydrate` is `done`

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

## Future Enhancements

1. **Custom workflows** — Allow `fab/project/config.yaml` to override or extend `workflow.yaml`
2. **Conditional stages** — Skip stages based on change attributes (e.g., docs-only changes skip `apply`)
3. **Parallel stages** — Multiple stages active simultaneously for different artifacts
4. **Stage hooks** — Run scripts before/after stage transitions
5. **State metadata** — Attach timestamps, user info, or exit codes to state transitions

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260215-lqm5-statusman-cli-only | 2026-02-15 | Updated script example from `source statusman.sh` to CLI subprocess pattern (`$STATUSMAN <subcommand>`) |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_preflight.sh` → `lib/preflight.sh` in skill example; updated `src/statusman/README.md` → `src/lib/statusman/README.md` |
| 260213-jc0u-split-archive-hydrate | 2026-02-13 | Updated progression references: terminal stage from `archive` to `hydrate` |
| 260226-6boq-event-driven-statusman | 2026-02-26 | Transitions are now event-keyed (event, from, to) instead of from→to with conditions. Five event commands: `start`, `advance`, `finish`, `reset`, `fail`. |
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | Added `ready` state (artifact exists, eligible for advancement). Removed unused `skipped` state. Updated transitions (`active→ready`, `ready→done`), progression (current stage includes `ready`), and validation (terminal states: `done` only). |
| 260212-4tw0-migrate-scripts-statusman | 2026-02-12 | Moved from `fab/.kit/schemas/README.md`, trimmed statusman API duplication |
