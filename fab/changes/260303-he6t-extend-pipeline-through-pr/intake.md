# Intake: Extend Pipeline Through PR

**Change**: 260303-he6t-extend-pipeline-through-pr
**Created**: 2026-03-03
**Status**: Draft

## Origin

> Discussion during `/fab-discuss` session. User decided that ff and fff should push all the way through to `git-fix-pr-reviews`, not stop at hydrate. This led to a deeper question: the statusman state machine should absorb git-pr and git-fix-pr-reviews as first-class stages (`ship` and `review-pr`) so their state is trackable via `.status.yaml`.

## Why

Currently the pipeline has a gap: ff/fff run `intake → spec → tasks → apply → review → hydrate` and then stop, printing `Next: /git-pr`. The user must manually invoke `/git-pr` and then `/git-fix-pr-reviews`. This breaks the "full send" promise of fff and leaves two workflow steps outside the state machine entirely.

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
  review-pr: pending    # NEW: git-fix-pr-reviews (wait → triage → fix)
```

#### `ship` stage

- Driven by `/git-pr`
- States: `pending → active → done` (no `failed` — git-pr fails fast, user retries)
- On `finish ship`: auto-activates `review-pr`, PR URL already in `prs[]`
- `stage_metrics.ship`: `started_at`, `completed_at`, `driver: "git-pr"`

#### `review-pr` stage

- Driven by `/git-fix-pr-reviews`
- States: `pending → active → done | failed`
- Second stage (alongside `review`) that supports `failed` — review found issues but fixes failed, or timeout
- Sub-state tracking via `stage_metrics.review-pr.phase`: `waiting | received | triaging | fixing | pushed`
- Optional `stage_metrics.review-pr.reviewer`: who reviewed (e.g., `copilot`, `@username`)

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
| ship               | /git-fix-pr-reviews                  | /git-fix-pr-reviews  |
| review-pr (pass)   | /fab-archive                         | /fab-archive         |
| review-pr (fail)   | /git-fix-pr-reviews                  | /git-fix-pr-reviews  |
```

### Extend ff/fff pipelines

Both `/fab-ff` and `/fab-fff` continue past hydrate:

```
... → hydrate → ship (git-pr) → review-pr (git-fix-pr-reviews)
```

- `/fab-ff`: extends through ship and review-pr (confidence-gated pipelines still gate at intake/spec, but the later stages run automatically once gated)
- `/fab-fff`: extends through ship and review-pr (full send)
- Both skills invoke `/git-pr` behavior for the ship stage and `/git-fix-pr-reviews` behavior for review-pr

### Update `/git-pr` to use statusman transitions

Git-pr should call `statusman.sh start` / `statusman.sh finish` for the `ship` stage so state is tracked. The `prs[]` array and `.pr-done` sentinel remain as supplementary signals.

### Update `/git-fix-pr-reviews` to use statusman transitions

The new skill (from change i58g) should call `statusman.sh start` / `statusman.sh finish` (or `fail`) for the `review-pr` stage. Update `phase` in stage_metrics as it progresses.

## Affected Memory

No memory files affected — this is a workflow infrastructure change.

## Impact

- `fab/.kit/scripts/lib/statusman.sh` — modified (new stages, new allowed states)
- `fab/.kit/templates/status.yaml` — modified (new stages in progress map)
- `fab/.kit/schemas/workflow.yaml` — modified (new stage definitions)
- `fab/.kit/skills/_preamble.md` — modified (state table)
- `.claude/skills/git-pr/SKILL.md` — modified (statusman integration)
- `.claude/skills/git-fix-pr-reviews/SKILL.md` — modified (statusman integration) — depends on i58g completing first
- `.claude/skills/fab-ff/SKILL.md` — modified (pipeline extension)
- `.claude/skills/fab-fff/SKILL.md` — modified (pipeline extension)
- Backlog items [a4v0] and [9yvv] — partially addressed by this change

## Open Questions

- Should `ship` support `failed` state, or is "fail fast + user retries" sufficient?
- Should `review-pr` be skippable? (e.g., repos without any review setup)
- For ff: does the confidence gate still only apply at intake/spec, or should ship/review-pr have their own gate logic?
- Should existing in-flight changes (already past hydrate) get the new stages added to their `.status.yaml`?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Both ff and fff extend through ship and review-pr | Discussed — user said "we should push ff and fff all the way through to git-pr-fix-reviews" | S:95 R:65 A:90 D:95 |
| 2 | Certain | Add `ship` and `review-pr` as first-class stages | Discussed — user asked for deep dive into state machine absorbing these | S:90 R:55 A:85 D:90 |
| 3 | Confident | `review-pr` supports `failed` state | Analogous to existing `review` stage; review failures are a real workflow state | S:70 R:70 A:80 D:75 |
| 4 | Confident | Sub-state tracking via `stage_metrics.phase` (Option B from discussion) | Extends existing pattern without new schema blocks; recommended in discussion | S:75 R:80 A:75 D:70 |
| 5 | Confident | `ship` does NOT support `failed` (fail fast, user retries) | Git-pr already fails fast; adding failed state adds complexity for no clear benefit | S:65 R:75 A:70 D:65 |
| 6 | Tentative | Existing in-flight changes don't get backfilled with new stages | Migration adds complexity; old changes can complete without ship/review-pr | S:50 R:60 A:55 D:50 |
| 7 | Tentative | ff confidence gates remain at intake/spec only — ship and review-pr have no gate | These stages are execution, not planning — confidence gating doesn't apply | S:55 R:70 A:60 D:55 |
| 8 | Tentative | `review-pr` is skippable (repos without review setup skip it) | Some repos won't have Copilot or mandatory reviews — need graceful handling | S:50 R:75 A:50 D:50 |

8 assumptions (2 certain, 3 confident, 3 tentative, 0 unresolved).
