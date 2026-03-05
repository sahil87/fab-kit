# Spec: Split Stage Display from Routing

**Change**: 260218-95xn-split-stage-display-from-routing
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/preflight.md`, `docs/memory/fab-workflow/change-lifecycle.md`

## Non-Goals

- Modifying `get_current_stage` or any routing/transition logic — routing is correct as-is
- Changing the `Next:` convention at the end of `/fab-continue` or other pipeline skill outputs (governed by `_context.md` State Table)
- Modifying the progress table symbols (`✓ ● ○ — ✗`) in `/fab-status`

## Stageman: Display Stage Function

### Requirement: `get_display_stage` function

`stageman.sh` SHALL provide a `get_display_stage` function that returns the stage representing "where you are" in the pipeline, distinct from `get_current_stage` which returns "what's next" for routing.

The function SHALL implement a two-tier fallback:

1. If any stage has state `active`, return that stage
2. Otherwise, return the last stage with state `done` (scanning in pipeline order: intake → spec → tasks → apply → review → hydrate)
3. If no stage is `active` or `done`, return the first stage (`intake`)

The function SHALL also output the state of the returned stage (either `active` or `done`) as a second value, enabling callers to display a state qualifier.

#### Scenario: Stage is active (in-progress work)

- **GIVEN** `.status.yaml` has `intake: done`, `spec: active`, remaining stages `pending`
- **WHEN** `get_display_stage` is called
- **THEN** it returns `spec` with state `active`

#### Scenario: Stage completed, next not started

- **GIVEN** `.status.yaml` has `intake: done`, all other stages `pending`
- **WHEN** `get_display_stage` is called
- **THEN** it returns `intake` with state `done`

#### Scenario: Fresh change (nothing started)

- **GIVEN** `.status.yaml` has all stages `pending`
- **WHEN** `get_display_stage` is called
- **THEN** it returns `intake` with state `pending`
<!-- clarified: Fresh change returns "pending" state — confirmed by user; rare edge case since /fab-new sets intake to done -->

#### Scenario: All stages done

- **GIVEN** `.status.yaml` has all stages `done`
- **WHEN** `get_display_stage` is called
- **THEN** it returns `hydrate` with state `done`

#### Scenario: Review failed

- **GIVEN** `.status.yaml` has `review: failed` and `apply: active`
- **WHEN** `get_display_stage` is called
- **THEN** it returns `apply` with state `active` (the active rework stage, not the failed review)

### Requirement: `display-stage` CLI command

`stageman.sh` SHALL expose `get_display_stage` via a `display-stage` subcommand:

```
stageman.sh display-stage <file>
```

The command SHALL output the stage name and state separated by a colon (e.g., `spec:active`, `intake:done`), consistent with the `key:value` output convention used by other stageman CLI commands (`progress-map`, `checklist`, `confidence`).

#### Scenario: CLI invocation

- **GIVEN** a valid `.status.yaml` with `intake: done`, `spec: active`
- **WHEN** `stageman.sh display-stage .status.yaml` is executed
- **THEN** stdout contains `spec:active`
- **AND** exit code is 0

#### Scenario: CLI with missing file

- **GIVEN** the specified file does not exist
- **WHEN** `stageman.sh display-stage missing.yaml` is executed
- **THEN** exit code is non-zero
- **AND** stderr contains an error message

## Preflight: Display Stage Output

### Requirement: `display_stage` field in preflight YAML

`preflight.sh` SHALL emit a `display_stage` field in its stdout YAML output, alongside the existing `stage` field. The `display_stage` field SHALL contain the display stage name (without state qualifier). A separate `display_state` field SHALL contain the state (`active`, `done`, or `pending`).

The existing `stage` field SHALL remain unchanged — it continues to represent the routing stage for `/fab-continue` dispatch.

#### Scenario: Preflight output after intake done

- **GIVEN** `.status.yaml` has `intake: done`, all other stages `pending`
- **WHEN** `preflight.sh` runs
- **THEN** the YAML output includes `stage: spec` (routing — next to produce)
- **AND** the YAML output includes `display_stage: intake` (display — where you are)
- **AND** the YAML output includes `display_state: done`

#### Scenario: Preflight output with active stage

- **GIVEN** `.status.yaml` has `intake: done`, `spec: active`
- **WHEN** `preflight.sh` runs
- **THEN** the YAML output includes `stage: spec` (routing)
- **AND** the YAML output includes `display_stage: spec` (display)
- **AND** the YAML output includes `display_state: active`

#### Scenario: Preflight output with all done

- **GIVEN** all stages are `done`
- **WHEN** `preflight.sh` runs
- **THEN** `stage: hydrate` (routing)
- **AND** `display_stage: hydrate` (display)
- **AND** `display_state: done`

## Changeman: Switch Display Format

### Requirement: Two-line stage/next display in switch output

The `changeman.sh switch` output SHALL replace the current single `Stage:` line with a two-line format showing the display stage and the next action separately:

```
Stage:  {display_stage} ({N}/6) — {state}
Next:   {next_stage} (via {default_command})
```

Where:
- `{display_stage}` is the display stage name from `get_display_stage`
- `{N}` is the stage number (1-6) of the display stage
- `{state}` is the state qualifier (`done` or `active`; `pending` for fresh changes)
- `{next_stage}` is the routing stage from `get_current_stage` (what `/fab-continue` will produce)
- `{default_command}` is the primary command from the state table

When all stages are done (hydrate complete), the `Next:` line SHALL show only the command: `Next:   /fab-archive`.

The `next_command` helper function SHALL continue to derive commands from the state table in `_context.md`, but the switch output formats it differently from the current format — wrapping it as `{stage} (via {command})` when there is a next stage.

#### Scenario: After `/fab-new` (intake done, nothing else started)

- **GIVEN** `.status.yaml` has `intake: done`, all other stages `pending`
- **WHEN** `changeman.sh switch` completes
- **THEN** the output includes:
  ```
  Stage:  intake (1/6) — done
  Next:   spec (via /fab-continue)
  ```

#### Scenario: Spec in progress

- **GIVEN** `.status.yaml` has `intake: done`, `spec: active`
- **WHEN** `changeman.sh switch` completes
- **THEN** the output includes:
  ```
  Stage:  spec (2/6) — active
  Next:   spec (via /fab-continue)
  ```

#### Scenario: All stages done

- **GIVEN** all stages `done`
- **WHEN** `changeman.sh switch` completes
- **THEN** the output includes:
  ```
  Stage:  hydrate (6/6) — done
  Next:   /fab-archive
  ```

## fab-status: Display Format

### Requirement: Display stage and next action in status output

The `/fab-status` skill SHALL use `display_stage` and `display_state` from the preflight YAML output to render the primary "Stage:" line with a state qualifier. The next action SHALL be shown as a separate line.

The progress table (with `✓ ● ○ — ✗` symbols) SHALL remain unchanged — it already conveys the full picture.

The existing confidence display, checklist display, and version drift check SHALL remain unchanged.

#### Scenario: Status after `/fab-new`

- **GIVEN** `intake: done`, all other stages `pending`
- **WHEN** `/fab-status` runs
- **THEN** the "Stage:" line shows `Stage: intake (1/6) — done`
- **AND** the "Next:" line shows `Next: spec (via /fab-continue)`

#### Scenario: Status with active stage

- **GIVEN** `intake: done`, `spec: active`
- **WHEN** `/fab-status` runs
- **THEN** the "Stage:" line shows `Stage: spec (2/6) — active`
- **AND** the "Next:" line shows `Next: spec (via /fab-continue)`

## fab-switch: Display Format

### Requirement: Consistent display format with fab-status

The `/fab-switch` skill output SHALL use the same two-line stage/next format. Since the switch output is passthrough from `changeman.sh switch`, this requirement is satisfied by the changeman changes. The skill's documentation (canonical output format) SHALL be updated to reflect the new two-line format.

#### Scenario: Switch output matches new format

- **GIVEN** a change with `intake: done`, all other stages `pending`
- **WHEN** `/fab-switch {change-name}` completes
- **THEN** the output includes:
  ```
  fab/current → {name}

  Stage:  intake (1/6) — done
  Next:   spec (via /fab-continue)
  ```

## Design Decisions

### Display Stage Output Format: `stage:state` colon-separated

**Decision**: `display-stage` CLI command outputs `stage:state` as a single colon-separated line (e.g., `spec:active`).

**Why**: Consistent with existing stageman CLI conventions (`progress-map` outputs `stage:state`, `checklist` outputs `key:value`). Callers can split on `:` with the same `IFS=: read -r` pattern already used in `preflight.sh` and other scripts.

**Rejected**: Two separate CLI calls (`display-stage-name` and `display-stage-state`) — doubles subprocess cost for a single concept. JSON output — overengineered for two values.

### Separate `display_stage` and `display_state` fields in preflight

**Decision**: Preflight emits two fields (`display_stage` and `display_state`) rather than a single combined field.

**Why**: Skills need both values independently — the stage name for position display and the state for the qualifier. A combined field (e.g., `display_stage: spec (active)`) would require downstream parsing. Two fields keep the YAML flat and machine-readable.

**Rejected**: Single combined field — requires parsing by consumers. Nested object — inconsistent with preflight's flat field convention.

### Next line shows default command only

**Decision**: The `Next:` line in changeman switch output shows only the default command from the state table (e.g., `/fab-continue`), not the full alternatives list.

**Why**: The switch output is a quick-glance summary. Showing all alternatives (`/fab-continue, /fab-ff, or /fab-clarify`) clutters the display. The full list is already available via `/fab-status` and at the end of each skill's output per the `_context.md` convention.

**Rejected**: Full alternatives list — too verbose for the switch context. No command — users need to know what to do next.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keep `get_current_stage` unchanged for routing | Confirmed from intake #1 — routing logic is correct; only display is misleading | S:95 R:90 A:95 D:95 |
| 2 | Certain | `get_display_stage` returns last active or last done stage | Confirmed from intake #2 — directly implements "where you are" semantics | S:90 R:85 A:90 D:90 |
| 3 | Certain | Add `display_stage` and `display_state` to preflight output | Confirmed from intake #3 — non-breaking addition; skills need both values | S:85 R:90 A:90 D:90 |
| 4 | Confident | State qualifier uses literal state name: "done", "active", "pending" | Confirmed from intake #4 — matches `.status.yaml` vocabulary; "pending" only for fresh changes | S:80 R:90 A:75 D:75 |
| 5 | Confident | `Next:` line shows `{stage} (via {default_command})` format | Confirmed from intake #5 — concise, actionable; full alternatives available via /fab-status | S:75 R:85 A:75 D:70 |
| 6 | Confident | `display-stage` CLI outputs `stage:state` colon-separated | Follows existing stageman CLI pattern (progress-map, checklist); consistent IFS=: parsing | S:70 R:90 A:85 D:80 |
| 7 | Certain | Fresh change (all pending) returns `intake` with state `pending` | Clarified — user confirmed; rare edge case since /fab-new sets intake to done | S:90 R:85 A:90 D:90 |

7 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-18

1. **Fresh change display state**: For a fresh change where all stages are `pending`, what should the display show?
   - **Answer**: `intake` with state `pending` — most accurate, and it's a rare edge case since `/fab-new` sets intake to `done`.
