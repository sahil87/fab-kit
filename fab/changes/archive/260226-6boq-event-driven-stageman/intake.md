# Intake: Event-Driven Stageman

**Change**: 260226-6boq-event-driven-stageman
**Created**: 2026-02-26
**Status**: Draft

## Origin

> Refactor stageman.sh to an event-driven state machine. Replace the current set-state backstop + transition convenience pattern with 5 explicit events: start, advance, finish, reset, fail. Remove set-state entirely.

Discussion session preceded this change. The user and agent walked through the current state model, identified that `set-state` acts as a backdoor bypassing the transition graph, and designed an event-driven replacement. The user confirmed the 5-event vocabulary and the semantic distinction: `finish` means "stage done, activate next" (works from both `active` and `ready`), `advance` means "move to ready checkpoint."

## Why

The current stageman API inverts the state machine pattern. `set-state` is the primary workhorse — skills name a target state and stageman validates only that the target is in `allowed_states`. The transition rules documented in `workflow.yaml` are aspirational, not enforced. Every skill independently gets the transition logic right (or doesn't). The `transition` command is a convenience wrapper for one specific pattern (`active→done + next→active`) but doesn't handle the `ready` state added in `260226-i9av`.

This means:
1. State machine logic is distributed across ~10 skill files instead of centralized in stageman
2. Prerequisite ordering isn't validated — you can `set-state spec active` while intake is `pending`
3. `transition` doesn't handle `ready` as a from-state — advancing a `ready` stage requires two raw `set-state` calls
4. Any skill can write any allowed state at any time, defeating the purpose of having a transition graph

A well-designed state machine has no `set-state` — the only API is "trigger event," and the machine resolves the new state from (current_state, event).

## What Changes

### 1. New event-based CLI commands in stageman.sh

Replace `set-state` and `transition` with 5 event commands:

```
stageman start   <change> <stage> [driver]    # {pending,failed} → active
stageman advance <change> <stage> [driver]    # active → ready
stageman finish  <change> <stage> [driver]    # {active,ready} → done [+ next pending→active]
stageman reset   <change> <stage> [driver]    # {done,ready} → active [+ downstream → pending]
stageman fail    <change> <stage> [driver]    # active → failed (review only)
```

**`<change>` resolution**: The first positional argument accepts either a raw file path (e.g., `fab/changes/260226-6boq-event-driven-stageman/.status.yaml`) or a change identifier (e.g., `6boq`, `event-driven`, full folder name). When the argument is not an existing file path, stageman resolves it via `changeman.sh resolve`, then appends `/.status.yaml` to construct the full path: `fab/changes/{resolved-name}/.status.yaml`. This eliminates path construction boilerplate from every skill invocation. The same resolution applies to all stageman commands (event and non-event alike).

**`[driver]`** is always optional. It's the name of the skill that triggered the state change (e.g., `fab-continue`, `fab-ff`). When provided, it's recorded in `stage_metrics` for traceability — you can inspect `.status.yaml` to see which skill last activated each stage. Skills are instructed to always pass it; manual/test invocations simply omit it. If omitted, the driver field is left empty in metrics. No validation, no error.

Each command:
- Resolves `<change>` to a `.status.yaml` path (via file or changeman resolution)
- Reads current state from `.status.yaml`
- Validates the transition is legal (current_state + event → new_state)
- Rejects illegal transitions with a diagnostic error
- Applies `stage_metrics` side-effects (same logic as today)
- Writes atomically (temp file → mv)

`finish` has an atomic side-effect: when stage N finishes, if stage N+1 exists and is `pending`, it becomes `active` (implicit `start`). If stage N is hydrate (last), just done — workflow complete.

### 2. Update workflow.yaml transitions section

Replace the current `from→to` transition format with an event-keyed format:

```yaml
transitions:
  default:
    - event: start
      from: [pending]
      to: active

    - event: advance
      from: [active]
      to: ready

    - event: finish
      from: [active, ready]
      to: done

    - event: reset
      from: [done, ready]
      to: active

  review:
    - event: start
      from: [pending, failed]
      to: active

    - event: advance
      from: [active]
      to: ready

    - event: finish
      from: [active, ready]
      to: done

    - event: reset
      from: [done, ready]
      to: active

    - event: fail
      from: [active]
      to: failed
```

### 3. Remove `set-state` and `transition` CLI commands

Delete `set_stage_state()`, `transition_stages()`, and their CLI dispatch cases. All non-event write commands (`set-change-type`, `set-checklist`, `set-confidence`, etc.) remain unchanged.

### 4. Update all skill files that call stageman

Map current calls to new events. With the change-ID shortcut, skills no longer construct file paths — they pass the change identifier directly:

| Current call | New call |
|-------------|----------|
| `stageman.sh set-state <f> intake active fab-new` | `stageman.sh start <change> intake fab-new` |
| `stageman.sh set-state <f> <stage> done` | `stageman.sh finish <change> <stage> <driver>` |
| `stageman.sh set-state <f> <stage> active <driver>` | `stageman.sh start <change> <stage> <driver>` |
| `stageman.sh set-state <f> review failed` | `stageman.sh fail <change> review` |
| `stageman.sh set-state <f> hydrate done` | `stageman.sh finish <change> hydrate <driver>` |
| `stageman.sh transition <f> <from> <to> <driver>` | `stageman.sh finish <change> <from> <driver>` |

Affected files:
- `fab/.kit/scripts/lib/changeman.sh` (1 call: `set-state "$status_file" intake active fab-new` → `start "$status_file" intake fab-new` — keeps raw path since changeman already has it)
- `fab/.kit/skills/fab-continue.md` (~10 calls)
- `fab/.kit/skills/fab-ff.md` (~8 calls)
- `fab/.kit/skills/fab-fff.md` (~8 calls)
- `src/lib/changeman/SPEC-changeman.md` (1 reference)

### 5. Update tests

Extensive rewrite of `src/lib/stageman/test.bats`. The current test suite tests `set-state` and `transition` directly — all of those tests must be replaced with event-based equivalents. New test coverage:

- **Happy path per event**: Each event from each valid from-state succeeds and produces the correct to-state
- **Rejection per event**: Each event from each invalid from-state fails with a diagnostic error (e.g., `start` from `active`, `advance` from `pending`, `finish` from `pending`, `fail` from non-review stage)
- **`finish` side-effect**: Finishing stage N atomically activates stage N+1; finishing hydrate (last stage) just marks done with no side-effect
- **Review-specific**: `fail` only works on review stage, `start` from `failed` only works on review stage
- **Stage metrics**: `start` increments iterations and sets started_at; `finish` sets completed_at on current stage and started_at/iterations on next; `reset` clears metrics; `advance` and `fail` are no-ops on metrics
- **Change-ID resolution**: Event commands accept change identifiers (partial slug, 4-char ID) and resolve to correct `.status.yaml` path
- **Error cases**: Missing file, invalid stage name, missing driver when required

### 6. Update memory docs with state transition table

Add the complete state transition table to `docs/memory/fab-workflow/change-lifecycle.md`:

```
═══════════════════════════════════════════════════════════
  FROM STATE  │  EVENT    │  TO STATE       │  REVIEW ONLY?
═══════════════════════════════════════════════════════════
  pending     │  start    │  active         │  no
  active      │  advance  │  ready          │  no
  active      │  finish   │  done (+next)   │  no
  ready       │  finish   │  done (+next)   │  no
  ready       │  reset    │  active         │  no
  done        │  reset    │  active         │  no
  active      │  fail     │  failed         │  YES
  failed      │  start    │  active         │  YES
═══════════════════════════════════════════════════════════
```

Update `schemas.md`, `execution-skills.md`, and `planning-skills.md` to reference the new event API instead of `set-state`/`transition`.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) State vocabulary, transition model, status mutations, stageman CLI references
- `fab-workflow/schemas`: (modify) workflow.yaml transition format, API reference
- `fab-workflow/execution-skills`: (modify) Status mutations overview, all stageman CLI references
- `fab-workflow/planning-skills`: (modify) Shared generation partial notes, stageman CLI references

## Impact

- **stageman.sh** — Core rewrite of write functions and CLI dispatch (~150 lines replaced)
- **workflow.yaml** — Transition schema restructured (event-keyed instead of from→to)
- **changeman.sh** — 1 line change (`set-state` → `start`)
- **fab-continue.md** — ~10 stageman call sites updated
- **fab-ff.md** — ~8 stageman call sites updated
- **fab-fff.md** — ~8 stageman call sites updated
- **SPEC-changeman.md** — 1 reference updated
- **test.bats** — Full rewrite of stageman tests
- **4 memory files** — Updated to reflect new API

No user-facing behavior changes. The pipeline stages, states, and progression logic are unchanged — only the mechanism for transitioning between states changes.

## Open Questions

None — the design was fully resolved in the preceding discussion session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | 5 events: start, advance, finish, reset, fail | Discussed — user designed and confirmed the event vocabulary | S:95 R:70 A:95 D:95 |
| 2 | Certain | Remove `set-state` entirely — no backdoor | Discussed — user explicitly chose Option A (event-driven, no backstop) | S:90 R:75 A:90 D:90 |
| 3 | Certain | Remove `transition` — replaced by `finish` | Discussed — `finish` subsumes transition's role with ready-state support | S:85 R:75 A:90 D:90 |
| 4 | Certain | `finish` works from both active and ready | Discussed — user: "no matter where we were in this stage" | S:90 R:75 A:90 D:90 |
| 5 | Certain | `advance` only goes active → ready | Discussed — user: "advance could mean transition from active to ready" | S:90 R:80 A:90 D:95 |
| 6 | Certain | Memory docs updated with state transition table | Discussed — user explicitly requested this as part of the change | S:95 R:90 A:90 D:95 |
| 7 | Confident | Non-event commands unchanged (all-stages, progress-map, checklist, confidence, etc.) | Not discussed but follows logically — these are read-only or non-stage-state writes | S:50 R:85 A:85 D:85 |
| 8 | Confident | changeman.sh updated to use `start` instead of `set-state` | Follows from removing `set-state` — changeman.sh has 1 call site | S:70 R:80 A:85 D:90 |
| 9 | Confident | All skill .md files updated to use event commands | Follows from removing `set-state`/`transition` — skills must use the new API | S:70 R:75 A:85 D:85 |
| 10 | Certain | All commands accept change ID or file path as first arg | Discussed — user proposed resolving via changeman; applies universally to event and non-event commands | S:90 R:80 A:90 D:90 |
| 11 | Certain | Driver is always optional on all event commands | Discussed — skills always pass it per instructions; manual/test callers omit it; no validation needed | S:90 R:85 A:90 D:90 |

11 assumptions (8 certain, 3 confident, 0 tentative, 0 unresolved).
