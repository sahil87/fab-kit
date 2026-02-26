# Intake: Add Ready State to Stage Lifecycle

**Change**: 260226-i9av-add-ready-state-to-stages
**Created**: 2026-02-26
**Status**: Draft

## Origin

> Add a `ready` state to the stage lifecycle: `pending → active → ready → done`. Currently `active` is overloaded — it means both "artifact is being generated" and "artifact exists and awaits advancement or clarification." This makes it impossible for agents to distinguish between resuming generation and advancing to the next stage.

Conversational — identified during a `/fab-continue` session where the agent confused the preflight `stage` field (forward-looking: "what to produce next") with the dispatch table's stage column (current-stage: "what stage is active"). The user proposed a three-state model (`pending → active → iterating → done`); discussion refined it to use `ready` instead of `iterating` for clearer semantics.

Key decisions from conversation:
- `ready` chosen over `iterating` — the semantic is "artifact produced, awaiting decision" not "ongoing work"
- The user wants to think more on edge cases within this change — the intake captures the agreed direction, spec stage will refine

**Prior rejection acknowledged**: The `change-lifecycle` design decision "Single Source of Truth: Progress Map with `active` Marker" explicitly rejected adding a `ready` state as "expands vocabulary for narrow edge case." This change revisits that decision based on concrete evidence: a dispatch bug where `stage: spec` + `spec: active` was ambiguous (spec not yet generated vs. spec exists and awaits advancement), and resume-session confusion where agents couldn't distinguish mid-generation from awaiting-user-action. The original rejection was made before the preflight/dispatch semantic mismatch was identified as a real failure mode.
<!-- clarified: acknowledged prior rejection of ready state with concrete evidence for reconsideration -->

## Why

1. **`active` is overloaded**: Currently, `spec: active` could mean either "spec.md is being generated right now" or "spec.md exists and the user can run `/fab-clarify` or `/fab-continue`." An agent resuming a session can't tell which. This caused a concrete bug: the `/fab-continue` dispatch table says `spec` → generate tasks.md (assuming spec.md exists), but preflight returns `stage: spec` when spec.md doesn't exist yet. The mismatch led to a failed `stageman.sh transition` call.

2. **Preflight/dispatch semantic mismatch**: Preflight's `stage` field is forward-looking ("what needs work"), while the dispatch table treats it as current-stage ("what's active"). Adding `ready` eliminates the ambiguity: `active` always means "work in progress", `ready` always means "work product exists, eligible for advancement."

3. **Resume clarity**: When an agent picks up a session, `active` = resume generation (maybe mid-stream), `ready` = artifact exists, ask user what to do next. No guessing.

## What Changes

### 1. New `ready` State + Drop `skipped` in workflow.yaml

Add `ready` to the `states` list in `fab/.kit/schemas/workflow.yaml`. Remove `skipped` — it is defined in the schema but never set by any script or skill (dead code). Update each stage's `allowed_states` to include `ready` and remove `skipped`:

```yaml
states:
  - id: pending
  - id: active
  - id: ready
  - id: done
  - id: failed
```

<!-- clarified: dropping unused skipped state as part of this change — no script or skill ever sets it -->

### 2. Stage Lifecycle: pending → active → ready → done

The transition flow becomes:

- `pending` → stage hasn't started
- `active` → work in progress (generating artifact, executing tasks)
- `ready` → stage's work product exists, eligible for:
  - `/fab-continue` → advance to next stage (`ready → done`, next stage `pending → active`)
  - `/fab-clarify` → deepen current artifact (stays `ready`)
- `done` → advanced past, locked

### 3. Transition Logic in stageman.sh

- `set-state` accepts `ready` as a valid state
- `transition` expects `from_stage` to be `ready` (not `active`) when advancing — this is the key behavioral change
- New convenience: `stageman.sh complete <file> <stage>` sets a stage from `active` to `ready` without advancing (for use after artifact generation)

### 4. Preflight Output Disambiguation

With the `ready` state, preflight's `stage` field becomes unambiguous:
- `stage: spec` + `spec: active` → agent should generate spec.md
- `stage: spec` + `spec: ready` → spec.md exists, `/fab-continue` advances to tasks

### 5. Skill Updates

All skills that transition stages need updating:
- `/fab-continue`: After generating an artifact, set to `ready` (not directly `done`). Separate "generate" dispatch from "advance" dispatch.
- `/fab-ff` and `/fab-fff`: Same pattern — generate → `ready` → advance in sequence
- `/fab-clarify`: Verify stage is `ready` (artifact exists) before scanning

### 6. Execution Stages

For apply/review/hydrate:
- `apply: active` → tasks being executed
- `apply: ready` → all tasks checked, eligible for review advancement
- `review: active` → sub-agent review in progress
- `review: ready` → review passed, eligible for hydrate advancement

### 7. Redefine `/fab-ff` Scope

`/fab-ff` is redefined from "fast-forward from spec" to "full pipeline with safety gates." Both `/fab-ff` and `/fab-fff` now start from intake — the difference is that `/fab-ff` can stop at multiple checkpoints while `/fab-fff` forces through.

**New `/fab-ff` behavior:**
- Starts from **intake** stage (was: spec-only)
- Three gates where execution can stop:
  1. **Intake gate**: Refuses to start if indicative confidence score < 3.0 (computed at intake, not persisted — `/fab-ff` must compute or read it)
  2. **Spec gate**: Stops after spec generation if confidence score < threshold (current dynamic per-type thresholds: fix=2.0, feat/refactor=3.0, docs/test/ci/chore=2.0, default=3.0)
  3. **Review gate**: Stops after 3 autonomous review rework cycles (was: interactive rework with no cap)
- On any gate stop, the user can intervene (run `/fab-clarify`, fix issues, then re-run `/fab-ff` to resume)

**New `/fab-fff` behavior** (mostly unchanged):
- Starts from intake (unchanged)
- No gates — forces through all stages regardless of confidence
- Autonomous rework with bounded retry (3-cycle cap, bail on exhaustion — unchanged)
<!-- assumed: fab-fff's 3-cycle review cap stays as-is even with "forces through" semantics — the cap is a practical safety limit, not a confidence gate -->

**Key differences (new model):**

| Aspect | `/fab-ff` (gated) | `/fab-fff` (ungated) |
|--------|-------------------|---------------------|
| Start point | intake | intake |
| Intake gate | indicative score >= 3.0 | none |
| Spec gate | confidence >= threshold | none |
| Review rework | autonomous, 3 cycles, then stop | autonomous, 3 cycles, then bail |
| Frontloaded questions | no | yes (unchanged) |
| Resume after stop | yes — re-run picks up | yes — re-run picks up |

### 8. Script & Test Impact Analysis

#### `stageman.sh` — 7 functions affected

| Function | Lines | Impact | What Changes |
|----------|-------|--------|-------------|
| `get_progress_line()` | 133-167 | High | Add `ready` case with new symbol (e.g., `◷`) |
| `get_current_stage()` | 203-241 | High | `ready` should NOT be treated as "active" for routing — a `ready` stage means the agent should advance, not generate. Likely treat `ready` similar to `done` for routing purposes |
| `get_display_stage()` | 243-278 | Moderate | Add `ready` as Tier 1.5 (after active, before done) — if a stage is `ready`, show it as the display stage |
| `_apply_metrics_side_effect()` | 169-197 | Moderate | `ready` = no-op (preserve existing metrics from `active`). Don't reset, don't update timestamps |
| `set_stage_state()` | 309-347 | Low | No `driver` required for `ready` (it's not a work-in-progress state; it signals completion of work) |
| `validate_status_file()` | 689-730 | Low | `ready` does NOT count toward active-stage limit — only `active` is limited to exactly 0-1 |
| `transition_stages()` | 349-420 | Moderate | Currently hardcodes `from→done, to→active`. Must change: `from→done` (via `ready`), `to→active`. The `ready→done` step happens BEFORE the transition call |

#### `changeman.sh` — NO changes needed

All state logic is fully delegated to `stageman.sh`. `changeman.sh` calls `stageman.sh` for state operations and display — it will automatically pick up the new `ready` state behavior.

#### `workflow.yaml` — Schema updates

- Add `ready` state definition (symbol, description, terminal: false)
- Remove `skipped` state definition and all references in transitions
- Update each stage's `allowed_states` to include `ready`
- Add transition rules: `active → ready` (artifact generation complete), `ready → done` (advancement)

#### `preflight.sh` — Routing logic

- `stage` field (routing): `ready` should route the same as if the stage were done and the next stage is pending — the artifact exists, so the next action is advancement
- `display_stage`/`display_state`: `ready` is shown as-is (e.g., `Stage: spec (2/6) — ready`)

#### Test files (`src/lib/stageman/test.bats`)

Tests needing updates or additions:
- **`progress-line` tests** (9 existing): update expected outputs for `ready` symbol
- **New `set-state ready` tests**: verify `ready` is accepted, no `driver` required
- **New `stage-metrics` tests**: verify `ready` is a no-op for metrics
- **New `current-stage` tests**: verify `ready` stages route correctly (advance, not generate)
- **New `display-stage` tests**: verify `ready` shows in display output
- **Existing `transition` tests**: verify `ready → done` transitions work
- **`changeman/test.bats`**: no changes needed (inherits from stageman)

#### `calc-score.sh` — Gate logic

- Add `--stage intake` support for computing indicative confidence at intake stage (for `/fab-ff` intake gate)
- Currently only runs at spec stage; needs to handle intake-stage `expected_min` thresholds

## Affected Memory

- `fab-workflow/schemas`: (modify) document new `ready` state and removal of unused `skipped` state in workflow.yaml
- `fab-workflow/change-lifecycle`: (modify) update state vocabulary and transition rules
- `fab-workflow/planning-skills`: (modify) update stage transition descriptions
- `fab-workflow/execution-skills`: (modify) update apply/review/hydrate transition descriptions

## Impact

- **`fab/.kit/schemas/workflow.yaml`**: New `ready` state, drop `skipped`, updated allowed_states and transitions per stage
- **`fab/.kit/scripts/lib/stageman.sh`**: 7 functions updated for `ready` state (progress line, routing, display, metrics, validation, transitions)
- **`fab/.kit/scripts/lib/preflight.sh`**: Disambiguation logic using ready state, display_stage/display_state for `ready`
- **`fab/.kit/scripts/lib/calc-score.sh`**: Add `--stage intake` support for `/fab-ff` intake gate
- **`fab/.kit/skills/fab-continue.md`**: Split generate/advance dispatch, use `ready` intermediate
- **`fab/.kit/skills/fab-ff.md`**: Targeted edits (~30% changes) — new starting point (intake), insert spec generation + spec gate, swap review fallback from interactive to stop
- **`fab/.kit/skills/fab-fff.md`**: Update contrast with `/fab-ff`, clarify "forces through" semantics
- **`fab/.kit/skills/fab-clarify.md`**: Stage guard accepts `ready` (artifact exists) for scanning
- **`fab/.kit/skills/_preamble.md`**: State table, confidence scoring, gate thresholds
- **`src/lib/stageman/test.bats`**: New tests for `ready` state (progress-line, set-state, metrics, routing, display)
- **`src/lib/changeman/test.bats`**: No changes (inherits from stageman)

## Open Questions

- ~~Should `/fab-continue` on a `ready` stage auto-advance without user confirmation, or should it prompt?~~ **Resolved**: Auto-advance — same UX as today. The `ready` state's purpose is agent/preflight disambiguation, not adding user friction. `/fab-clarify` remains the opt-in deepening step.
<!-- clarified: fab-continue auto-advances from ready — no user confirmation added -->
- ~~Does `failed` need a parallel: `review: failed` means "go back to apply" — does it stay as-is or interact with `ready`?~~ **Resolved**: `failed` stays as-is (review-only). After rework, the reworked stage goes through `ready` (consistent model: regenerate → `ready` → auto-advance to `done`). Same lifecycle pattern everywhere — no special cases.
<!-- clarified: failed unchanged, rework stages go through ready for consistency -->
- ~~For `/fab-ff` and `/fab-fff`, should the pipeline pause at each `ready` state or auto-advance through?~~ **Resolved**: Auto-advance — pipelines set `ready` then immediately transition to `done`/next. `ready` is transient in pipeline context, not a pause point.
<!-- clarified: pipeline commands auto-advance through ready states -->

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five states: `pending → active → ready → done` + `failed` | Discussed — user proposed three-state extension, naming refined from `iterating` to `ready` | S:95 R:70 A:85 D:90 |
| 2 | Certain | `ready` means "artifact exists, eligible for advancement or clarification" | Discussed — user specified the semantic: can fab-continue (advance) or fab-clarify (deepen) | S:90 R:70 A:85 D:90 |
| 3 | Certain | `active` means "work in progress, generation/execution ongoing" | Discussed — user specified: if active, we know artifact is still to be created | S:90 R:70 A:85 D:90 |
| 4 | Confident | `transition` requires `ready` (not `active`) for advancement | Logical consequence — prevents advancing before artifact exists. Major behavioral change but eliminates the root cause of the dispatch confusion | S:80 R:50 A:80 D:75 |
| 5 | Tentative | New `stageman.sh complete` subcommand for active→ready | Reasonable convenience to avoid two separate set-state calls, but may be unnecessary if skills just call `set-state <stage> ready` | S:60 R:85 A:70 D:55 |
| 6 | Certain | Pipeline commands (`/fab-ff`, `/fab-fff`) auto-advance through `ready` | Clarified — user confirmed auto-advance; `ready` is transient in pipeline context, not a pause point | S:90 R:60 A:85 D:90 |
| 7 | Certain | `failed` unchanged; rework stages go through `ready` | Clarified — `failed` stays review-only; after rework, reworked stages go `ready → done` for consistency | S:90 R:55 A:85 D:90 |
| 8 | Certain | `/fab-continue` auto-advances from `ready` | Clarified — same UX as today; `ready` is for agent disambiguation, not user friction | S:90 R:70 A:90 D:90 |
| 9 | Certain | Drop unused `skipped` state from workflow.yaml | Clarified — `skipped` defined in schema but never set by any script or skill; dead code cleanup | S:95 R:80 A:95 D:95 |
| 10 | Certain | Prior rejection of `ready` state acknowledged | Clarified — change-lifecycle DD rejected `ready`; revisited due to concrete dispatch bug and resume ambiguity evidence | S:95 R:60 A:90 D:90 |
| 11 | Certain | `/fab-ff` starts from intake (like `/fab-fff`) | Discussed — user specified both commands start from intake; difference is gates, not starting point | S:95 R:45 A:90 D:90 |
| 12 | Certain | `/fab-ff` intake gate: refuses if indicative score < 3.0 | Discussed — user specified threshold of 3.0 at intake stage | S:90 R:50 A:85 D:85 |
| 13 | Confident | `/fab-ff` spec gate uses existing dynamic per-type thresholds | User said "check what this condition is now" — current thresholds are dynamic (fix=2.0, feat=3.0, etc.). Keeping dynamic thresholds is consistent | S:75 R:55 A:80 D:70 |
| 14 | Certain | `/fab-ff` review: autonomous rework, 3 cycles, then stop | Discussed — user specified "review fails 3 times" as a stop condition, replacing interactive rework | S:90 R:50 A:85 D:85 |
| 15 | Tentative | `/fab-fff` 3-cycle review cap unchanged | User says fff "forces through" — the 3-cycle cap is a practical safety limit, not a confidence gate. Unclear if "forces through" means removing the cap | S:60 R:40 A:60 D:50 |
<!-- assumed: fab-fff's 3-cycle review cap stays as-is — interpreting "forces through" as "no gates" rather than "no safety limits" -->
| 16 | Confident | `changeman.sh` needs no changes | Analysis confirmed: all state logic delegated to stageman.sh; changeman only calls stageman CLI | S:90 R:85 A:95 D:90 |
| 17 | Confident | `ready` state: no driver required, no metrics side-effect, not counted toward active limit | `ready` signals completion (not work-in-progress); metrics already tracked from `active` phase; only `active` should be limited to 0-1 | S:75 R:70 A:75 D:70 |

17 assumptions (10 certain, 4 confident, 2 tentative, 0 unresolved). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-26

Q: The change-lifecycle memory explicitly rejected adding a `ready` state. Should the intake acknowledge this?
A: Yes — added acknowledgment to Origin section with concrete evidence (dispatch bug, resume ambiguity) that changed the calculus.

Q: The schemas memory lists `skipped` as a valid state but the intake omits it. What to do?
A: Investigation showed `skipped` is defined in workflow.yaml but never set by any script/skill. Decision: drop it as dead code in this change.

Q: Should `/fab-continue` auto-advance from `ready` or prompt?
A: Auto-advance — same UX as today. `ready` is for agent disambiguation, not user friction.

Q: Should pipeline commands auto-advance through `ready`?
A: Yes — `ready` is transient in pipeline context, not a pause point.

Q: Does `failed` interact with `ready`? After rework, should stages go through `ready`?
A: `failed` unchanged (review-only). After rework, stages go through `ready → done` for consistency.

Q: Redefine `/fab-ff` scope — starts from intake with safety gates?
A: Yes — `/fab-ff` now starts from intake (like `/fab-fff`). Three gates: intake indicative score >= 3.0, spec confidence >= threshold, review 3-cycle cap. `/fab-fff` forces through with no gates.

Q: What changes are needed in stageman.sh and changeman.sh for the ready state?
A: `stageman.sh`: 7 functions affected (progress line, routing, display, metrics, validation, state setter, transitions). `changeman.sh`: no changes (delegates to stageman). `workflow.yaml`: add state, update transitions, drop `skipped`. Test files: new cases for progress-line, set-state, metrics, routing, display.
