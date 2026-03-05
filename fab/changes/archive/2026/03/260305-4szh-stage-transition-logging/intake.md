# Intake: Stage Transition Logging

**Change**: 260305-4szh-stage-transition-logging
**Created**: 2026-03-05
**Status**: Draft

## Origin

> Backlog item [4szh]: "Improve rework loop logging — log explicit apply re-entry events in .history.jsonl, clarify whether stage_metrics.iterations counts the stage or the apply→review pair, and standardize ad-hoc event names (smoke-test, test-pass) into the schema"

Preceded by a `/fab-discuss` session that explored the current state of rework loop logging, identified the three sub-problems, and converged on design decisions through conversation.

**Key decisions from discussion:**
- New `stage-transition` event type (chosen over extending existing `command` or `review` events)
- Log ALL stage transitions, not just re-entries (complete state machine trace)
- Permissive validation: document canonical values, don't whitelist
- Leave existing archived `.history.jsonl` files untouched

## Why

`.history.jsonl` currently logs three event types: `command`, `confidence`, and `review`. Stage transitions (a stage going `active`) are invisible — you can infer them from command events but there's no first-class record. This makes rework loops hard to trace: when review fails and the pipeline re-enters apply, the only evidence is a second `command` event and a `review` event, with no explicit "re-entered apply because review failed with fix-code."

Additionally, `stage_metrics.iterations` semantics are clear in code (counts per-stage activations) but undocumented — the backlog item itself asks "does it count the stage or the apply→review pair?" Finally, ad-hoc event names (`smoke-test`, `test-pass`) appear in archived history files without schema documentation.

If unfixed: debugging rework loops requires manual correlation across events, `iterations` semantics remain tribal knowledge, and the event schema drifts further from what's actually logged.

## What Changes

### New `stage-transition` event type in `logman.sh`

Add a 4th subcommand to `logman.sh`:

```
logman.sh transition <change> <stage> <action> [from] [reason] [driver]
```

Produces JSON events:

First entry (stage activated for the first time):
```json
{"ts":"ISO-8601","event":"stage-transition","stage":"spec","action":"enter","driver":"fab-ff"}
```

Re-entry (stage re-activated after rework):
```json
{"ts":"ISO-8601","event":"stage-transition","stage":"apply","action":"re-entry","from":"review","reason":"fix-code","driver":"fab-ff"}
```

Fields:
- `event`: always `"stage-transition"`
- `stage`: the stage being activated (e.g., `"apply"`, `"spec"`)
- `action`: `"enter"` (first activation, `iterations` going from 0→1) or `"re-entry"` (`iterations` > 1)
- `from`: present only on re-entries — the stage that triggered the rework (e.g., `"review"`)
- `reason`: present only on re-entries — the rework type (e.g., `"fix-code"`, `"revise-spec"`)
- `driver`: the skill/tool that triggered the transition (e.g., `"fab-ff"`, `"fab-continue"`)

### Emit transition events from `statusman.sh`

In `_apply_metrics_side_effect`, when the `active` case fires, call `logman.sh transition` after incrementing `iterations`. The function already has `stage`, `driver`, and `iterations` (post-increment) available. The `from` and `reason` fields require new parameters passed through from the calling statusman subcommand (`start`, `finish`, `reset`).

Specifically:
- `iterations == 1` → `action=enter`, no `from`/`reason`
- `iterations > 1` → `action=re-entry`, `from` and `reason` passed through from the caller

The `from` and `reason` values flow from the statusman caller context:
- `statusman.sh finish review` auto-activating the next stage → `enter` (not a rework)
- `statusman.sh start <stage>` after a `fail` → `re-entry` with `from` and `reason` from the fail context
- `statusman.sh reset <stage>` → `re-entry` with `from` as the stage that was reset from

### Document `iterations` semantics in memory

Update `docs/memory/fab-workflow/change-lifecycle.md` and `docs/memory/fab-workflow/kit-scripts.md` to clarify:
- `iterations` counts **per-stage activations** (each time a stage transitions to `active`)
- It is NOT the count of apply→review pairs
- First activation = 1, each rework re-entry increments by 1

### Document canonical review result values

Update the `.history.jsonl` schema documentation in both memory files to:
- List `"passed"` and `"failed"` as the canonical `result` values for review events
- Note that validation is permissive — non-canonical values are accepted (useful for ad-hoc debugging)
- Reference the existing ad-hoc values (`"smoke-test"`, `"test-pass"`) as examples of non-canonical usage in archived changes

### Update `_scripts.md` skill reference

Add `logman.sh transition` to the `_scripts.md` subcommand table, callers table, and help text section.

## Affected Memory

- `fab-workflow/kit-scripts`: (modify) Add `stage-transition` event documentation, `logman.sh transition` subcommand, update callers table, document `iterations` semantics
- `fab-workflow/change-lifecycle`: (modify) Add `stage-transition` to event history schema, clarify `iterations` semantics in `stage_metrics` description, document canonical review result values

## Impact

- `fab/.kit/scripts/lib/logman.sh` — new `transition` subcommand (~30 lines)
- `fab/.kit/scripts/lib/statusman.sh` — `_apply_metrics_side_effect` emits transition events, new parameters for `from`/`reason` passthrough
- `fab/.kit/skills/_scripts.md` — add transition subcommand docs
- `docs/memory/fab-workflow/kit-scripts.md` — schema and caller documentation
- `docs/memory/fab-workflow/change-lifecycle.md` — event history schema, iterations semantics
- No changes to: `calc-score.sh`, `preflight.sh`, `changeman.sh`, `resolve.sh`, any skill files

## Open Questions

None — all design decisions resolved during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New `stage-transition` event type (4th event) | Discussed — user chose new event type over extending existing command/review events | S:95 R:80 A:90 D:95 |
| 2 | Certain | Log ALL stage transitions, not just re-entries | Discussed — user explicitly stated "should log all stage transitions" | S:95 R:85 A:85 D:95 |
| 3 | Certain | Permissive validation — document canonical values, don't whitelist | Discussed — user chose permissive approach | S:90 R:90 A:85 D:90 |
| 4 | Certain | Leave existing archived `.history.jsonl` files untouched | Discussed — user said "Nothing" to retroactive cleanup | S:95 R:95 A:90 D:95 |
| 5 | Certain | Change type is `fix` | Discussed — user stated "yes, its a fix" | S:95 R:90 A:90 D:95 |
| 6 | Confident | `action` field: `enter` vs `re-entry` based on iterations count | Discussed — agreed during conversation, exact field name chosen by agent | S:85 R:85 A:80 D:75 |
| 7 | Confident | Re-entries include `from` and `reason` fields; first entries omit them | Discussed — proposed by agent, user agreed | S:80 R:80 A:80 D:75 |
| 8 | Confident | `driver` always present on transition events | Discussed — leverages existing `driver` parameter in `_apply_metrics_side_effect` | S:80 R:85 A:85 D:80 |
| 9 | Confident | `from`/`reason` passthrough via new parameters to `_apply_metrics_side_effect` | Implementation detail — `statusman.sh` callers need to propagate rework context | S:70 R:80 A:75 D:70 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).