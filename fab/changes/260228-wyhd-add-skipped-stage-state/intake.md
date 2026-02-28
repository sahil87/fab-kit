# Intake: Add "skipped" Stage State

**Change**: 260228-wyhd-add-skipped-stage-state
**Created**: 2026-02-28
**Status**: Draft

## Origin

> Add "skipped" as a valid stage state. New skip event in statusman ({pending,active}→skipped) with cascade: all downstream pending stages become skipped. Add skipped to allowed_states for spec/tasks/apply/review/hydrate (not intake). Update get_current_stage, get_progress_line, get_display_stage, validate_stage_state, and _apply_metrics_side_effect to handle skipped. Update workflow.yaml schema with new state and transition. The skip event is the statusman call: `statusman.sh skip <change> <stage> [driver]`. Skipped stages satisfy prerequisites (treated like done). No auto-activate after skip. Resetting a skipped stage brings it to active with normal downstream cascade to pending.

Conversational mode — decisions were made in a `/fab-discuss` session before this change was created. All core design choices are settled.

## Why

Some changes only need an intake — the user wants to capture the intent and context but doesn't need spec, tasks, apply, review, or hydrate. Currently every stage must go through `pending → active → done`, so there's no way to express "I deliberately chose not to do this stage." The only workaround is marking stages `done` without producing artifacts, which is semantically wrong (done = completed successfully with artifact).

A `skipped` state makes the intent explicit: the stage was intentionally bypassed. This enables "intake-only" changes and any future partial-pipeline use cases.

## What Changes

### 1. New `skipped` state in `fab/.kit/schemas/workflow.yaml`

Add a new state definition:

```yaml
- id: skipped
  symbol: "⏭"
  description: "Stage intentionally bypassed"
  terminal: true
```

Add `skipped` to `allowed_states` for all stages **except intake**:

```yaml
# intake stays: [active, ready, done]
# spec becomes: [pending, active, ready, done, skipped]
# tasks becomes: [pending, active, ready, done, skipped]
# apply becomes: [pending, active, ready, done, skipped]
# review becomes: [pending, active, ready, done, failed, skipped]
# hydrate becomes: [pending, active, ready, done, skipped]
```

### 2. New `skip` transition in `workflow.yaml`

Add to the `transitions.default` section:

```yaml
- event: skip
  from: [pending, active]
  to: skipped
```

Also extend the `reset` transition to allow `skipped → active`:

```yaml
# default reset becomes:
- event: reset
  from: [done, ready, skipped]
  to: active
```

And in the review-specific overrides:

```yaml
# review reset becomes:
- event: reset
  from: [done, ready, skipped]
  to: active
```

### 3. New `event_skip` function in `fab/.kit/scripts/lib/statusman.sh`

```bash
# event_skip <status_file> <stage> [driver]
# {pending,active} → skipped. Cascade: all downstream pending stages → skipped.
# No auto-activate of next stage. Metrics cleared (same as pending).
```

Side-effect: iterate all stages after the target; if state is `pending`, set to `skipped`. This mirrors the cascade pattern in `event_reset` but going forward instead of backward.

### 4. New CLI subcommand `skip` in statusman.sh

```
statusman.sh skip <change> <stage> [driver]
```

Dispatches to `event_skip`. Follows the same argument resolution and validation pattern as `start`, `finish`, etc.

### 5. Update `get_current_stage` in statusman.sh

The progression logic currently finds "first active or ready" then falls back to "first pending after last done." It needs to treat `skipped` like `done` for progression — skipped stages are resolved, not blocking.

The fallback logic should find the first `pending` stage after the last `done` or `skipped` stage.

### 6. Update `get_display_stage` in statusman.sh

Add tier for skipped: treat `skipped` similarly to `done` when finding the last resolved stage. The "last done" fallback becomes "last done or skipped."

### 7. Update `get_progress_line` in statusman.sh

Add a `skipped` case in the progress line rendering:

```bash
skipped) parts+=("$stage ⏭") ;;
```

### 8. Update `_apply_metrics_side_effect` in statusman.sh

Add `skipped` case: delete metrics (like `pending` — nothing happened):

```bash
skipped)
  yq -i "del(.stage_metrics.${stage})" "$tmpfile"
  ;;
```

### 9. Update `docs/specs/user-flow.md` diagram 5

Already done by the user — diagram 5 now includes the `skipped` state, `skip` event, and `reset` from `skipped`. The side-effects table has the new `skip` row.

## Affected Memory

- `fab-workflow/schemas`: (modify) Document the new `skipped` state and `skip` event in workflow schema memory
- `fab-workflow/execution-skills`: (modify) Note that skills can now skip stages via `statusman.sh skip`

## Impact

- **workflow.yaml**: New state, new transition, updated allowed_states, updated reset transitions
- **statusman.sh**: New `event_skip` function, new CLI subcommand, updates to 4 query functions
- **user-flow.md**: Already updated (diagram 5, side-effects table)
- **Downstream skills**: No immediate changes — skills continue calling `statusman.sh` as before. The `skip` call is opt-in; existing pipelines don't use it yet
- **Validation**: `validate_status_file` already validates against `allowed_states`, so it will accept `skipped` once the schema is updated

## Open Questions

None — all design decisions were resolved in the discussion session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `skip` event from `pending` or `active` states | Revised — skipping an active stage is valid ("started but don't need it"); `ready` excluded (deliberate reset-then-skip for artifact discard) | S:90 R:85 A:90 D:85 |
| 2 | Certain | Intake cannot be skipped | Discussed — intake is the entry point, always required | S:95 R:85 A:95 D:95 |
| 3 | Certain | `skip` cascades downstream pending stages to `skipped` | Discussed — user agreed with cascade over per-stage | S:95 R:85 A:90 D:90 |
| 4 | Certain | No auto-activate after skip | Discussed — skip is terminal intent, not "move to next" | S:95 R:90 A:90 D:95 |
| 5 | Certain | Skipped stages satisfy prerequisites (like `done`) | Discussed — skipped = resolved for progression | S:90 R:85 A:90 D:90 |
| 6 | Certain | `reset` from `skipped` → `active` with normal downstream cascade to `pending` | Discussed — reuse existing reset mechanics for un-skip | S:90 R:90 A:90 D:95 |
| 7 | Certain | Fully-skipped change is directly archivable | Discussed — intake done + rest skipped = archive-ready, no forced hydrate | S:95 R:80 A:85 D:90 |
| 8 | Certain | Symbol for skipped is `⏭` | Follows the existing pattern (done=✓, failed=✗, active=●, pending=○, ready=◷) | S:80 R:95 A:90 D:85 |
| 9 | Confident | `skipped` state has `terminal: true` in schema | Analogous to `done` — it's a resolved end state. Resettable via explicit `reset` like `done` | S:80 R:90 A:85 D:80 |
| 10 | Confident | No metrics for skipped stages (delete like `pending`) | Nothing happened — no started_at, no completed_at to record | S:75 R:90 A:85 D:85 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
