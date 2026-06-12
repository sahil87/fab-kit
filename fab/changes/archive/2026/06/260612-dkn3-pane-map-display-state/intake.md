# Intake: Pane Map Display State

**Change**: 260612-dkn3-pane-map-display-state
**Created**: 2026-06-12

## Origin

One-shot `/fab-new dkn3` (backlog ID). Backlog entry from `fab/backlog.md` (found in the main repo's working copy — the entry is an uncommitted edit there and not yet present in this worktree's copy):

> [dkn3] 2026-06-12: Expose the stage state axis in 'fab pane map --json'. DisplayStage (src/go/fab/internal/status/status.go:463-498) computes (stage, state) but both callers discard the state half — panemap.go:324 and internal/pane/pane.go:152 — so downstream consumers (run-kit sidebar) cannot distinguish an actively-worked stage from a parked finished change: a fully-shipped change displays 'review-pr' indefinitely until archived, identical to review-pr actively running. Add display_state (active/ready/done/failed/pending/skipped) as a nullable field on pane map JSON rows (additive shape change; run-kit consumes it via app/backend/internal/sessions/sessions.go paneMapEntry and would update its docs/specs/api.md separately). Unlocks honest per-row attention states in run-kit (failed/ready = needs human, done+parked = quiet row) instead of heuristics over agentState/.fab-runtime.yaml idle_since.

No Linear ID attached. No prior `/fab-discuss` session — all decisions below derive from the backlog entry plus code inspection.

## Why

1. **Pain point**: `fab pane map --json` exposes only the stage *name*, not its *state*. `status.DisplayStage()` already computes both halves, but `panemap.go:324` discards the state (`stage, _ := status.DisplayStage(statusFile)`). A fully-shipped change renders `"stage": "review-pr"` indefinitely until archived — byte-identical to a change whose review-pr is actively running.

2. **Consequence if unfixed**: the run-kit sidebar (the primary JSON consumer, via `app/backend/internal/sessions/sessions.go` `paneMapEntry`) must guess attention states from heuristics over `agent_state` and `.fab-runtime.yaml` `idle_since` — fragile proxies for information fab already has. Parked-done rows stay visually indistinguishable from in-flight rows, defeating at-a-glance triage across many panes.

3. **Why this approach**: the data already exists at the call site — this is plumbing, not computation. An additive nullable JSON field is backward compatible (existing consumers ignore unknown keys) and matches the precedent set by `pr_url`/`pr_number` (change r7ju). The alternative — run-kit deriving state by reading `.status.yaml` itself — would duplicate fab's state-machine logic across repos.

4. **Scope addition discovered during gap analysis**: `DisplayStage`'s tier walk (active → ready → last done/skipped → first pending) never returns `failed`. A review-failed change with nothing active falls through to Tier 3 and displays as `apply`/`done`. The backlog entry explicitly enumerates `failed` among the `display_state` values, and its stated unlock ("failed/ready = needs human") requires `failed` to be reachable — so a `failed` tier must be added to `DisplayStage`, not just plumbed through.

## What Changes

### 1. `fab pane map --json`: new `display_state` field

In `src/go/fab/cmd/fab/panemap.go`:

- At the `DisplayStage` call (line 324), capture both return values:
  ```go
  stage, state := status.DisplayStage(statusFile)
  stageName = stage
  stageState = state
  ```
- Thread the state through `paneRow` as a new string field (initialized to the em-dash sentinel `—`, exactly like `stage`, in both the early-return row for non-repo panes and the main constructor).
- Add to `paneJSON`, placed immediately after `Stage`:
  ```go
  DisplayState *string `json:"display_state"`
  ```
- Map via the existing `toNullable()` helper: panes with no resolvable change (or an unloadable `.status.yaml`) emit `"display_state": null` — the same nullability contract as `stage` and `change`.

Example row (parked shipped change — the motivating case):

```json
{
  "session": "main", "window_index": 2, "pane": "%5", "tab": "dkn3",
  "worktree": "fab-kit.worktrees/dkn3/", "repo": "/home/sahil/code/sahil87/fab-kit",
  "change": "260612-dkn3-pane-map-display-state",
  "stage": "review-pr",
  "display_state": "done",
  "agent_state": null, "agent_idle_duration": null,
  "pr_url": "https://github.com/sahil87/fab-kit/pull/394", "pr_number": 394
}
```

Possible values: `active`, `ready`, `done`, `failed`, `pending`, `skipped`, or `null`. The **table output is unchanged** — this is a JSON-only additive shape change.

### 2. `DisplayStage`: add a `failed` tier

In `src/go/fab/internal/status/status.go` (`DisplayStage`, lines 463–498): insert a `failed` check between the `active` tier and the `ready` tier, returning `(ss.Stage, "failed")` for the first failed stage.
<!-- clarified: failed tier ordering confirmed as active > failed > ready > done/skipped > pending — user accepted recommendation 2026-06-12 -->

Rationale for the position: `active` stays first because in-progress work supersedes a parked failure (a coexistence is possible — e.g. review `failed` while the user manually starts a later pending stage); `failed` must outrank `ready`/`done` so a parked failure surfaces instead of being masked by Tier 3's "last done" fallback. Only `review` and `review-pr` can hold `failed` (`ValidStates` per-stage map, status.go:24-27).

Blast radius of this tier (all become *more* honest for review-failed changes, no consumer breaks):
- `fab preflight` `display_stage`/`display_state` → reports `review`/`failed` instead of `apply`/`done`. `/fab-continue`'s dispatch table already has a `review`/`failed` row (added in batch 1, PR #390) — today it compensates by reading `progress.review` directly, so the new output aligns with, rather than contradicts, the skill.
- `fab change list` (`name:display_stage:display_state:score`) and `/fab-status`'s "Stage:" line → show `review — failed`.
- `CurrentStage()` (routing) is deliberately **not** touched — routing for failed states is owned by skill dispatch logic, not display derivation.

### 3. Tests and CLI docs

- `src/go/fab/cmd/fab/panemap_test.go`: assert `display_state` is present and correct for a change-bearing pane, and `null` for a pane without a change.
- `src/go/fab/internal/status/status_test.go`: new `DisplayStage` cases — review `failed` with nothing active returns `("review", "failed")`; `failed` + a later `active` stage returns the active stage.
- `src/kit/skills/_cli-fab.md` § `fab pane map`: document the new JSON field and its value set (constitution: CLI changes MUST update `_cli-fab.md`).

## Affected Memory

- `runtime/pane-commands`: (modify) document the `display_state` JSON field on `fab pane map --json` rows — value set, nullability, additive-shape rationale
- `pipeline/change-lifecycle`: (modify) record the `DisplayStage` `failed` tier — review-failed changes now display `review — failed` in `/fab-status`, `fab change list`, and preflight output instead of falling through to the last done stage

## Impact

- **Code**: `src/go/fab/cmd/fab/panemap.go` (capture + paneRow + paneJSON + nullable mapping), `src/go/fab/internal/status/status.go` (failed tier), `src/go/fab/cmd/fab/panemap_test.go`, `src/go/fab/internal/status/status_test.go`
- **Docs**: `src/kit/skills/_cli-fab.md` (pane map JSON shape)
- **Untouched**: `src/go/fab/internal/pane/pane.go:152` (`PaneContext`) — cited in the backlog as evidence of the discard pattern, but its consumers (`fab pane send`/`capture` confirmation displays) have no use for the state axis; pane map table output; `CurrentStage()` routing
- **Downstream (out of scope, separate repo)**: run-kit consumes the field via `app/backend/internal/sessions/sessions.go` `paneMapEntry` and updates its `docs/specs/api.md` independently. Additive change — run-kit works unmodified until it opts in.
- **Compatibility**: purely additive JSON shape; `fab preflight`/`fab change list` display becomes more accurate for review-failed changes (a behavior change in display only, aligned with existing skill dispatch)

## Open Questions

- None — the backlog entry is high-signal (exact call sites, field name, value set, nullability, and consumer named) and code inspection resolved the remaining design points into the graded assumptions below.

## Clarifications

### Session 2026-06-12

**Q1: Where should the `failed` tier sit in DisplayStage's precedence walk?**
A: active → failed → ready → done/skipped → pending (accepted recommendation). In-progress work stays visible; parked failures outrank ready/done.

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 2 | Confirmed | — |
| 4 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Deliverable is a nullable `display_state` field on pane map JSON rows only; table output unchanged | Backlog states it verbatim ("additive shape change", "nullable field on pane map JSON rows") | S:95 R:90 A:95 D:95 |
| 2 | Certain | Add a `failed` tier to `DisplayStage` so `display_state` can actually return `failed` | Clarified — user confirmed | S:95 R:70 A:85 D:75 |
| 3 | Certain | Tier precedence: active → failed → ready → done/skipped → pending | Clarified — user confirmed | S:95 R:80 A:55 D:45 |
| 4 | Certain | Do not extend `PaneContext` (pane.go:152) or the table renderer | Clarified — user confirmed | S:95 R:90 A:75 D:70 |
| 5 | Certain | Go change ships with test updates and a `_cli-fab.md` pane-map doc update | Constitution mandates both for fab CLI changes | S:90 R:95 A:100 D:95 |
| 6 | Certain | `CurrentStage()` routing untouched — failed-state routing remains owned by skill dispatch (`/fab-continue`'s review/failed row) | Clarified — user confirmed | S:95 R:75 A:80 D:70 |
| 7 | Certain | `failed` tier returns the first failed stage in pipeline order | Clarified — user confirmed | S:95 R:85 A:80 D:75 |
| 8 | Certain | `paneRow` carries the state as a string defaulting to the em-dash sentinel; `DisplayState` placed immediately after `Stage` in `paneJSON` | Clarified — user confirmed | S:95 R:90 A:90 D:80 |
| 9 | Certain | run-kit consumer changes (sessions.go `paneMapEntry`, its `docs/specs/api.md`) are out of scope | Backlog states run-kit updates its side "separately"; separate repo | S:95 R:90 A:95 D:95 |

9 assumptions (9 certain, 0 confident, 0 tentative, 0 unresolved).
