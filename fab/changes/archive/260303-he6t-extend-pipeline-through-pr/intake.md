# Intake: Extend Pipeline Through PR

**Change**: 260303-he6t-extend-pipeline-through-pr
**Created**: 2026-03-03
**Status**: Draft

## Origin

> Discussion during `/fab-discuss` session. User decided that ff and fff should push all the way through to `git-pr-review`, not stop at hydrate. This led to a deeper question: the statusman state machine should absorb git-pr and git-pr-review as first-class stages (`ship` and `review-pr`) so their state is trackable via `.status.yaml`.

## Framing

This is a **change lifecycle** extension, not a git-specific one. The 6-stage pipeline already tracks non-code activities (intake = requirements, spec = design, hydrate = documentation). Shipping a PR and handling review feedback are lifecycle events — the change isn't done when memory is hydrated, it's done when the work is integrated.

The stages use lifecycle language (`ship`, `review-pr`); the skills that drive them happen to be git-based (`/git-pr`, `/git-pr-review`). If someone later adds a non-git shipping mechanism, they write a different skill and wire it to the same stage. This preserves Constitution Principle VI (Git-Optional) — Fab doesn't couple its identity to git, but it tracks integration status as part of the change lifecycle.

| Phase | Stages | What's tracked |
|-------|--------|----------------|
| Planning | intake → spec → tasks | Requirements, design, breakdown |
| Execution | apply → review | Implementation, validation |
| Completion | hydrate | Knowledge capture |
| Integration | ship → review-pr | Delivery, acceptance |

## Why

Currently the pipeline has a gap: ff/fff run `intake → spec → tasks → apply → review → hydrate` and then stop, printing `Next: /git-pr`. The user must manually invoke `/git-pr` and then `/git-pr-review`. This breaks the "full send" promise of fff and leaves two workflow steps outside the state machine entirely.

PR state is tracked via side-band mechanisms: a `prs[]` array and a gitignored `.pr-done` sentinel. There's no way to answer "has this change been shipped?" or "is it waiting for PR review?" from `.status.yaml` alone — you need to check external signals.

If we don't fix this: fff remains a partial pipeline, the state machine has a blind spot after hydrate, and the backlog items [a4v0]/[9yvv] (trackable states for git-pr) remain unaddressed.

## What Changes

### Add `ship` and `review-pr` stages to statusman

Extend the progress map from 6 stages to 8:

```yaml
progress:
  intake: pending
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending         # NEW: git-pr (commit → push → create PR)
  review-pr: pending    # NEW: git-pr-review (wait → triage → fix)
```

#### `ship` stage

- Driven by `/git-pr`
- States: `pending → active → done` (no `failed` — git-pr fails fast, user retries)
- On `finish ship`: auto-activates `review-pr`, PR URL already in `prs[]`
- `stage_metrics.ship`: `started_at`, `completed_at`, `driver: "git-pr"`

#### `review-pr` stage

- Driven by `/git-pr-review`
- States: `pending → active → done | failed`
- Second stage (alongside `review`) that supports `failed` — review found issues but fixes failed, or timeout
- No new top-level states needed — `active` covers both waiting-for-reviews and working-on-fixes. The `phase` sub-state in `stage_metrics` captures the distinction.
- Sub-state tracking via `stage_metrics.review-pr.phase`: `waiting | received | triaging | fixing | pushed`
- Optional `stage_metrics.review-pr.reviewer`: who reviewed (e.g., `copilot`, `@username`)
- The async waiting concern (polling for reviews) is entirely internal to the `/git-pr-review` skill — the pipeline just sees `active → done` or `active → failed`, same delegation pattern as apply

### Update statusman.sh

- Add `ship` and `review-pr` to the stage order array
- Allow `failed` state for `review-pr` (currently only `review` supports it)
- Add `phase` and `reviewer` as optional fields in `stage_metrics`
- Update `finish hydrate` to auto-activate `ship` (currently hydrate is terminal)
- Update `finish ship` to auto-activate `review-pr`

### Update `.status.yaml` template

Add `ship: pending` and `review-pr: pending` to the progress map in `fab/.kit/templates/status.yaml`.

### Update workflow schema

Add `ship` and `review-pr` entries to `fab/.kit/schemas/workflow.yaml` with their allowed states.

### Update `_preamble.md` state table

Add new states to the state table:

```
| hydrate            | /git-pr, /fab-archive                | /git-pr              |
| ship               | /git-pr-review                       | /git-pr-review       |
| review-pr (pass)   | /fab-archive                         | /fab-archive         |
| review-pr (fail)   | /git-pr-review                       | /git-pr-review       |
```

### Extend ff/fff pipelines

Both `/fab-ff` and `/fab-fff` continue past hydrate:

```
... → hydrate → ship (git-pr) → review-pr (git-pr-review)
```

- `/fab-ff`: extends through ship and review-pr (confidence-gated pipelines still gate at intake/spec, but the later stages run automatically once gated)
- `/fab-fff`: extends through ship and review-pr (full send)
- Both skills invoke `/git-pr` behavior for the ship stage and `/git-pr-review` behavior for review-pr

### Update `/git-pr` to use statusman transitions

Git-pr should call `statusman.sh start` / `statusman.sh finish` for the `ship` stage so state is tracked. The `prs[]` array and `.pr-done` sentinel remain as supplementary signals.

### Rename `/git-review` to `/git-pr-review`

Rename the skill to distinguish it from the internal `review` stage (code validation sub-agent). `/git-pr-review` processes *PR* reviews (external), while `review` is the internal validation stage. The `git-` prefix signals "this touches git"; the `-pr-review` suffix clarifies the scope.

### Update `/git-pr-review` to use statusman transitions

The skill (renamed from `/git-review`, extracted in change i58g) should call `statusman.sh start` / `statusman.sh finish` (or `fail`) for the `review-pr` stage. Update `phase` in stage_metrics as it progresses.

## Affected Memory

No memory files affected — this is a workflow infrastructure change.

## Impact

- `fab/.kit/scripts/lib/statusman.sh` — modified (new stages, new allowed states)
- `fab/.kit/templates/status.yaml` — modified (new stages in progress map)
- `fab/.kit/schemas/workflow.yaml` — modified (new stage definitions)
- `fab/.kit/skills/_preamble.md` — modified (state table)
- `fab/.kit/skills/git-pr.md` — modified (statusman integration)
- `fab/.kit/skills/git-review.md` — renamed to `git-pr-review.md`, modified (statusman integration) — depends on i58g completing first
- `fab/.kit/skills/fab-ff.md` — modified (pipeline extension)
- `fab/.kit/skills/fab-fff.md` — modified (pipeline extension)
- Backlog items [a4v0] and [9yvv] — partially addressed by this change

## Resolved Questions

- **`ship` failed state?** — No. Fail fast + user retries is sufficient. Git-pr already fails fast; adding a failed state adds complexity for no clear benefit.
- **Confidence gates for ship/review-pr?** — No. These are integration stages, not planning. Confidence gating applies to planning decisions only (intake/spec).
- **`active` state for waiting?** — No new state needed. `active` already means "in progress." Whether the agent is polling for reviews or fixing code, it's still active. The `phase` sub-state in `stage_metrics` captures the distinction internally. The async waiting concern is entirely internal to `/git-pr-review` — the pipeline just sees `active → done` or `active → failed`.
- **Skill naming?** — `/git-review` renamed to `/git-pr-review` to distinguish from the internal `review` stage (code validation sub-agent). Stage names use lifecycle language (`ship`, `review-pr`); skill names use tool language (`/git-pr`, `/git-pr-review`).
- **Constitution VI tension?** — No conflict. This is a change lifecycle extension, not git coupling. The stages track integration status using lifecycle concepts; the skills that drive them happen to be git-based. Non-git projects skip these stages.
- **Backfill existing changes?** — No preemptive backfill. Statusman tolerates missing stages in the progress map — treat absent `ship`/`review-pr` as implicitly `skipped` for old changes. If `/git-pr` is invoked on an old 6-stage change, lazy-append the stages on first use. Zero migration code, zero risk.
- **`review-pr` skippable?** — Always attempted after ship succeeds. Auto-skip (via pipeline) only when no git or no remote — same conditions that skip `ship`. The skill handles "no reviews arrive" as a normal `done` (phase: `waiting → done`), not `skipped`. `skipped` is reserved for environments where integration is inapplicable. No `required: false` needed in the schema.

## Open Questions

(none — all resolved through discussion)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Both ff and fff extend through ship and review-pr | Discussed — user explicitly directed full pipeline extension | S:95 R:65 A:90 D:95 |
| 2 | Certain | Add `ship` and `review-pr` as first-class stages | Discussed — user confirmed lifecycle framing for state machine | S:90 R:55 A:85 D:90 |
| 3 | Certain | `ship` does NOT support `failed` (fail fast, user retries) | Discussed — agreed git-pr fails fast, no benefit from failed state | S:90 R:75 A:85 D:90 |
| 4 | Certain | ff confidence gates remain at intake/spec only — ship and review-pr have no gate | Discussed — integration stages, not planning; gating doesn't apply | S:90 R:70 A:85 D:90 |
| 5 | Certain | No new top-level states — `active` covers waiting and working | Discussed — phase sub-state in stage_metrics handles the distinction | S:90 R:80 A:90 D:90 |
| 6 | Certain | Rename `/git-review` to `/git-pr-review` | Discussed — distinguishes from internal review stage; user confirmed | S:95 R:70 A:90 D:95 |
| 7 | Confident | `review-pr` supports `failed` state | Analogous to existing `review` stage; review failures are a real workflow state | S:70 R:70 A:80 D:75 |
| 8 | Confident | Sub-state tracking via `stage_metrics.phase` | Extends existing pattern without new schema blocks | S:75 R:80 A:75 D:70 |
| 9 | Confident | No preemptive backfill for in-flight changes | Discussed — statusman tolerates missing stages (treat absent as `skipped`); lazy append on first use if `/git-pr` invoked on old 6-stage change | S:75 R:75 A:80 D:75 |
| 10 | Confident | `review-pr` always attempted after ship; auto-skip only when no git/no remote | Discussed — skill handles "no reviews" as normal `done` (not `skipped`); `skipped` only for inapplicable environments. No `required: false` needed. | S:75 R:75 A:75 D:75 |

10 assumptions (6 certain, 4 confident, 0 tentative, 0 unresolved).
