# Spec: Stage Transition Logging

**Change**: 260305-4szh-stage-transition-logging
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-scripts.md`, `docs/memory/fab-workflow/change-lifecycle.md`

## logman.sh: New `transition` Subcommand

### Requirement: logman.sh SHALL support a `transition` subcommand

`logman.sh` SHALL accept a 4th subcommand `transition` with the following signature:

```
logman.sh transition <change> <stage> <action> [from] [reason] [driver]
```

The subcommand SHALL append a single JSON line to `{change_dir}/.history.jsonl` with event type `"stage-transition"`.

#### Scenario: First entry into a stage
- **GIVEN** a change with `.history.jsonl` (or no history file yet)
- **WHEN** `logman.sh transition 4szh spec enter "" "" fab-ff` is called
- **THEN** a JSON line is appended: `{"ts":"<ISO-8601>","event":"stage-transition","stage":"spec","action":"enter","driver":"fab-ff"}`
- **AND** the `from` and `reason` fields are omitted from the JSON (not present as empty strings)

#### Scenario: Re-entry into a stage after rework
- **GIVEN** a change that has previously completed the apply stage
- **WHEN** `logman.sh transition 4szh apply re-entry review fix-code fab-ff` is called
- **THEN** a JSON line is appended: `{"ts":"<ISO-8601>","event":"stage-transition","stage":"apply","action":"re-entry","from":"review","reason":"fix-code","driver":"fab-ff"}`

#### Scenario: Driver is omitted
- **GIVEN** any transition event
- **WHEN** `logman.sh transition 4szh spec enter` is called without a driver
- **THEN** the `driver` field is omitted from the JSON output

### Requirement: Transition event fields SHALL follow a defined schema

The `stage-transition` event JSON SHALL contain these fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ts` | string | always | ISO-8601 timestamp |
| `event` | string | always | Always `"stage-transition"` |
| `stage` | string | always | Stage being activated (e.g., `"apply"`, `"spec"`) |
| `action` | string | always | `"enter"` (first activation) or `"re-entry"` (rework) |
| `from` | string | re-entry only | Stage that triggered the rework (e.g., `"review"`) |
| `reason` | string | re-entry only | Rework type (e.g., `"fix-code"`, `"revise-spec"`) |
| `driver` | string | when provided | Skill that triggered the transition |

#### Scenario: Validation is permissive
- **GIVEN** a transition event call with `action="custom-debug"`
- **WHEN** `logman.sh transition 4szh apply custom-debug` is called
- **THEN** the event is logged as-is without validation error
- **AND** `"custom-debug"` appears in the `action` field

### Requirement: `transition` SHALL resolve changes via `resolve.sh`

The `transition` subcommand SHALL resolve `<change>` using `resolve_change_dir()` (existing helper that delegates to `resolve.sh --dir`), consistent with the `command`, `confidence`, and `review` subcommands.

#### Scenario: Change resolution failure
- **GIVEN** an invalid change reference `"nonexistent"`
- **WHEN** `logman.sh transition nonexistent spec enter` is called
- **THEN** logman exits with code 1
- **AND** an error message is printed to stderr

## statusman.sh: Transition Event Emission

### Requirement: `_apply_metrics_side_effect` SHALL emit transition events

When `_apply_metrics_side_effect` processes the `active` case (a stage becoming active), it SHALL call `logman.sh transition` after incrementing `iterations`.

The function SHALL accept two new optional parameters: `from` and `reason`, appended after the existing `driver` parameter. The full signature becomes:

```bash
_apply_metrics_side_effect <tmpfile> <stage> <state> [driver] [from] [reason]
```

#### Scenario: First activation (enter)
- **GIVEN** a stage with `iterations == 0` (not yet tracked in `stage_metrics`)
- **WHEN** `_apply_metrics_side_effect` fires for state `active`
- **THEN** `iterations` is incremented to 1
- **AND** `logman.sh transition <change> <stage> enter "" "" <driver>` is called
- **AND** the logman call is best-effort (`2>/dev/null || true`)

#### Scenario: Re-entry (rework)
- **GIVEN** a stage with `iterations == 1` (previously activated once)
- **WHEN** `_apply_metrics_side_effect` fires for state `active` with `from=review` and `reason=fix-code`
- **THEN** `iterations` is incremented to 2
- **AND** `logman.sh transition <change> <stage> re-entry review fix-code <driver>` is called

#### Scenario: Non-active states do not emit transitions
- **GIVEN** any stage
- **WHEN** `_apply_metrics_side_effect` fires for state `done`, `pending`, or `skipped`
- **THEN** no `logman.sh transition` call is made

### Requirement: Change directory SHALL be derived from tmpfile path for logman calls

`_apply_metrics_side_effect` operates on a tmpfile (the caller handles atomicity). To call `logman.sh transition`, it SHALL derive the change folder name from the tmpfile's parent directory path (consistent with how `event_finish` and `event_fail` derive it for existing logman calls).

#### Scenario: Folder derivation from tmpfile
- **GIVEN** a tmpfile at `/path/to/fab/changes/260305-4szh-stage-transition-logging/.status.yaml.XXXXXX`
- **WHEN** the function needs the change folder for logman
- **THEN** it extracts the folder name from the grandparent directory path (since tmpfile is a sibling of `.status.yaml` in the change dir)

### Requirement: Callers SHALL propagate `from` and `reason` to `_apply_metrics_side_effect`

The event functions that can trigger `active` state (`event_start`, `event_finish`, `event_reset`) SHALL pass `from` and `reason` through to `_apply_metrics_side_effect`.

#### Scenario: `event_finish` auto-activating next stage
- **GIVEN** `event_finish` completing a stage and activating the next pending stage
- **WHEN** it calls `_apply_metrics_side_effect` for the next stage with `active` state
- **THEN** `from` and `reason` are empty (forward flow is always `enter`, not rework)

#### Scenario: `event_reset` re-activating a stage
- **GIVEN** `event_reset` called with `from` and `reason` parameters
- **WHEN** it calls `_apply_metrics_side_effect` for the target stage with `active` state
- **THEN** `from` and `reason` are passed through to the function

#### Scenario: `event_start` re-activating after failure
- **GIVEN** `event_start` called with `from` and `reason` parameters
- **WHEN** it calls `_apply_metrics_side_effect` for the target stage with `active` state
- **THEN** `from` and `reason` are passed through to the function

### Requirement: `start` and `reset` CLI SHALL accept optional `from` and `reason` parameters

The CLI interface SHALL be extended:

```
statusman.sh start <change> <stage> [driver] [from] [reason]
statusman.sh reset <change> <stage> [driver] [from] [reason]
```

#### Scenario: Reset with from/reason
- **GIVEN** a review failure requiring rework
- **WHEN** `statusman.sh reset 4szh apply fab-ff review fix-code` is called
- **THEN** apply is reset to `active`
- **AND** a `stage-transition` event is logged with `action=re-entry`, `from=review`, `reason=fix-code`

#### Scenario: Start without from/reason (backward compatible)
- **GIVEN** existing callers that use `statusman.sh start <change> <stage> [driver]`
- **WHEN** called without `from` and `reason` parameters
- **THEN** the command works as before (from/reason default to empty strings)
- **AND** the transition event uses `enter` or `re-entry` based on iterations count

## Skill Reference: `_scripts.md` Update

### Requirement: `_scripts.md` SHALL document the `transition` subcommand

The `logman.sh` section in `fab/.kit/skills/_scripts.md` SHALL be updated to include:

1. The `transition` subcommand in the usage section
2. The `transition` subcommand in the callers table (called by `statusman.sh` event functions)
3. Updated CLI signature in the subcommands list

#### Scenario: Developer reads _scripts.md for logman reference
- **GIVEN** a developer or agent reading `_scripts.md`
- **WHEN** they look at the logman.sh section
- **THEN** they see `logman.sh transition <change> <stage> <action> [from] [reason] [driver]` in the subcommands list
- **AND** the callers table shows `statusman.sh _apply_metrics_side_effect` → `logman.sh transition`

### Requirement: `_scripts.md` SHALL document extended `start`/`reset` CLI parameters

The statusman.sh section SHALL reflect the new optional `[from] [reason]` parameters on `start` and `reset` subcommands.

#### Scenario: Updated statusman help text
- **GIVEN** a developer reading the statusman.sh key subcommands table
- **WHEN** they look at `start` and `reset`
- **THEN** the usage column shows `[from] [reason]` as optional trailing parameters

## Memory: Event History Schema Update

### Requirement: `change-lifecycle.md` SHALL document `stage-transition` as a 4th event type

The "Event History (`.history.jsonl`)" section SHALL list `stage-transition` alongside the existing `command`, `confidence`, and `review` events.

#### Scenario: Updated event type list
- **GIVEN** the Event History section in `change-lifecycle.md`
- **WHEN** it lists event types
- **THEN** it includes: `- **stage-transition** — stage activation: {"ts":"...", "event":"stage-transition", "stage":"apply", "action":"enter", "driver":"fab-ff"}` (and a re-entry example with `from`/`reason`)

### Requirement: Event type count SHALL be updated from "Three" to "Four"

The sentence "Three event types:" SHALL be updated to "Four event types:" in the Event History section.

#### Scenario: Count accuracy
- **GIVEN** the Event History section
- **WHEN** it describes the number of event types
- **THEN** it says "Four event types"

## Memory: `iterations` Semantics Documentation

### Requirement: `change-lifecycle.md` SHALL clarify `iterations` semantics in `stage_metrics`

The `stage_metrics` field description SHALL explicitly document:
- `iterations` counts per-stage activations (each time a stage transitions to `active`)
- First activation = 1, each rework re-entry increments by 1
- It is NOT the count of apply→review pairs

#### Scenario: Developer checks iterations meaning
- **GIVEN** a developer reading the `stage_metrics` description in `change-lifecycle.md`
- **WHEN** they look for `iterations` semantics
- **THEN** they find an explicit definition: "counts per-stage activations, incremented each time the stage transitions to `active`"

### Requirement: `kit-scripts.md` SHALL document `iterations` in the stage metrics section

The `_apply_metrics_side_effect` documentation in `kit-scripts.md` SHALL note the `iterations` increment behavior and its relationship to transition event `action` field.

#### Scenario: Developer reads kit-scripts for iterations
- **GIVEN** a developer reading `kit-scripts.md`
- **WHEN** they look at the stage metrics or `_apply_metrics_side_effect` documentation
- **THEN** they see that `iterations` is incremented on every `active` transition, and `iterations == 1` maps to `action=enter` while `iterations > 1` maps to `action=re-entry`

## Memory: Canonical Review Result Values

### Requirement: `change-lifecycle.md` SHALL document canonical review result values

The review event documentation SHALL list `"passed"` and `"failed"` as canonical `result` values, note that validation is permissive (non-canonical values accepted), and reference existing ad-hoc values as examples.

#### Scenario: Developer checks valid review results
- **GIVEN** a developer reading the review event schema in `change-lifecycle.md`
- **WHEN** they look for valid `result` values
- **THEN** they see `"passed"` and `"failed"` listed as canonical values
- **AND** a note that non-canonical values (e.g., `"smoke-test"`, `"test-pass"`) are accepted for ad-hoc debugging

### Requirement: `kit-scripts.md` SHALL document canonical review result values

The `.history.jsonl` format section in `kit-scripts.md` SHALL list canonical review result values alongside the existing event format documentation.

#### Scenario: Kit-scripts review event reference
- **GIVEN** a developer reading the review event format in `kit-scripts.md`
- **WHEN** they look at the review event JSON example
- **THEN** they see a note about canonical values (`"passed"`, `"failed"`) and permissive validation

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New `stage-transition` event type (4th event) | Confirmed from intake #1 — user chose new event type | S:95 R:80 A:90 D:95 |
| 2 | Certain | Log ALL stage transitions, not just re-entries | Confirmed from intake #2 — user explicitly stated this | S:95 R:85 A:85 D:95 |
| 3 | Certain | Permissive validation — document canonical values, don't whitelist | Confirmed from intake #3 — consistent with existing logman approach | S:90 R:90 A:85 D:90 |
| 4 | Certain | Leave existing archived `.history.jsonl` files untouched | Confirmed from intake #4 — no retroactive changes | S:95 R:95 A:90 D:95 |
| 5 | Certain | Change type is `fix` | Confirmed from intake #5 — user stated | S:95 R:90 A:90 D:95 |
| 6 | Certain | `action` field: `enter` (iterations==1) vs `re-entry` (iterations>1) | Upgraded from intake #6 — code confirms iterations logic in `_apply_metrics_side_effect` | S:90 R:85 A:90 D:90 |
| 7 | Confident | Re-entries include `from` and `reason`; first entries omit them | Confirmed from intake #7 — requires new parameters on `_apply_metrics_side_effect` | S:80 R:80 A:80 D:75 |
| 8 | Certain | `driver` always present when provided to statusman | Upgraded from intake #8 — code confirms driver is already plumbed through `_apply_metrics_side_effect` | S:85 R:85 A:90 D:85 |
| 9 | Confident | `from`/`reason` passthrough via new parameters on `_apply_metrics_side_effect`, `event_start`, `event_reset` CLI | Confirmed from intake #9 — extends existing parameter pattern | S:70 R:80 A:75 D:70 |
| 10 | Certain | `event_finish` auto-activate always passes empty from/reason (forward flow is not rework) | New — code analysis confirms finish → next is always a first entry | S:90 R:90 A:90 D:90 |
| 11 | Certain | logman transition call is best-effort (`2>/dev/null \|\| true`) | New — consistent with existing logman calls in `event_finish` and `event_fail` | S:90 R:95 A:90 D:90 |
| 12 | Certain | Derive change folder from tmpfile parent directory (existing pattern in event_finish/event_fail) | New — code confirms this is how existing logman calls derive the folder | S:90 R:90 A:95 D:90 |

12 assumptions (10 certain, 2 confident, 0 tentative, 0 unresolved).
