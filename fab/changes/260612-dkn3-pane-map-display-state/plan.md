# Plan: Pane Map Display State

**Change**: 260612-dkn3-pane-map-display-state
**Intake**: `intake.md`

## Requirements

### Pane Map: `display_state` JSON field

#### R1: Nullable `display_state` on pane map JSON rows
`fab pane map --json` rows SHALL carry a `display_state` field (`*string`, `json:"display_state"`, placed immediately after `Stage` in `paneJSON`) populated from the state half of `status.DisplayStage` — the half currently discarded at `src/go/fab/cmd/fab/panemap.go:324`. The state SHALL thread through `paneRow` as a string field defaulting to the em-dash sentinel `—` (set in BOTH the early-return row for non-repo panes and the main constructor, mirroring `stage`), and SHALL map to JSON via the existing `toNullable()` helper: panes with no resolvable change (or an unloadable `.status.yaml`) emit `"display_state": null` — the same nullability contract as `stage` and `change`. The human table output MUST remain unchanged (no new column) — this is a JSON-only additive shape change. Possible values: `active`, `ready`, `done`, `failed`, `pending`, `skipped`, or `null`.

- **GIVEN** a tmux pane whose worktree has an active change with `.status.yaml` where `review-pr: done` (fully shipped, parked)
- **WHEN** `fab pane map --json` resolves that pane
- **THEN** the row contains `"stage": "review-pr"` AND `"display_state": "done"`

- **GIVEN** a tmux pane outside any git repo (or in a worktree with no `fab/` dir or no active change)
- **WHEN** `fab pane map --json` resolves that pane
- **THEN** the row contains `"display_state": null` (em-dash sentinel mapped through `toNullable()`)

- **GIVEN** any set of pane rows
- **WHEN** `fab pane map` renders the human table
- **THEN** the output is byte-identical to the pre-change table (no `display_state` column, no value leakage)

#### R2: `DisplayStage` gains a `failed` tier
`status.DisplayStage` (`src/go/fab/internal/status/status.go`) SHALL insert a `failed` tier between the existing `active` tier and the `ready` tier, returning the FIRST failed stage in pipeline order as `(ss.Stage, "failed")`. Final precedence: active → failed → ready → last done/skipped → first pending. `active` stays first because in-progress work supersedes a parked failure; `failed` outranks `ready`/`done` so a parked failure surfaces instead of being masked by the last-done fallback. Only `review` and `review-pr` can hold `failed` per the per-stage `ValidStates` map.

- **GIVEN** a `.status.yaml` with `review: failed` and no stage `active` or `ready`
- **WHEN** `DisplayStage` is called
- **THEN** it returns `("review", "failed")` (previously fell through to the last done stage, e.g. `("apply", "done")`)

- **GIVEN** a `.status.yaml` with `review: failed` AND a later stage `active` (e.g. `hydrate: active`)
- **WHEN** `DisplayStage` is called
- **THEN** it returns the active stage (`("hydrate", "active")`) — the failed tier does not preempt active work

- **GIVEN** a `.status.yaml` with no `failed` stage
- **WHEN** `DisplayStage` is called
- **THEN** the result is identical to the pre-change derivation (active → ready → last done/skipped → first pending)

#### R3: CLI docs document the new JSON field
`src/kit/skills/_cli-fab.md` § `fab pane map` SHALL document the `display_state` JSON field: its value set (`active`/`ready`/`done`/`failed`/`pending`/`skipped` or `null`), its nullability contract (null when the pane has no resolvable change), and its placement in the JSON field list. Constitution mandate: fab CLI changes MUST update `_cli-fab.md`. Edits are made ONLY under `src/kit/` — never `.claude/skills/` (deployed copies, gitignored).

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** a consumer reads the `--json` flag row of § `fab pane map`
- **THEN** `display_state` appears in the snake_case field list with its full value set and null semantics

### Non-Goals

- `src/go/fab/internal/pane/pane.go` `PaneContext` (line 152 discard) — its consumers (`fab pane send`/`capture` confirmation displays) have no use for the state axis
- Pane map table renderer — no new column, output byte-identical
- `CurrentStage()` routing — failed-state routing remains owned by skill dispatch (`/fab-continue`'s review/failed row), not display derivation
- run-kit consumer changes (`app/backend/internal/sessions/sessions.go` `paneMapEntry`, its `docs/specs/api.md`) — separate repo, opts in independently; additive change means run-kit works unmodified

### Design Decisions

1. **`failed` tier position**: active → failed → ready → done/skipped → pending — *Why*: in-progress work supersedes a parked failure (coexistence is possible, e.g. review `failed` while a later pending stage is manually started); `failed` must outrank `ready`/`done` so parked failures surface — *Rejected*: failed-first (would hide active work); failed after ready (Tier 3's "last done" fallback would mask the failure). (Clarified in intake — user-confirmed.)
2. **First failed stage in pipeline order**: matches the existing tier-walk convention (first active, first ready) — *Why*: deterministic, and only `review`/`review-pr` can be `failed` so at most two candidates exist. (Clarified in intake.)
3. **Plumb via existing patterns, no new helpers**: reuse `toNullable()`, em-dash sentinel, and the `paneRow` → `paneJSON` mapping established by `repo` (h3jk) and `pr_url` (r7ju) — *Why*: the data already exists at the call site; this is plumbing, not computation.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Insert the `failed` tier in `DisplayStage` (`src/go/fab/internal/status/status.go`): between the active tier and the ready tier, walk the progress map and return `(ss.Stage, "failed")` for the first `failed` stage; renumber the tier comments (active → failed → ready → last done/skipped → first pending) <!-- R2 -->
- [x] T002 Add `DisplayStage` test cases in `src/go/fab/internal/status/status_test.go`: (a) `review: failed` with nothing active/ready returns `("review", "failed")`; (b) `review: failed` plus a later `active` stage returns the active stage; (c) no-failed regression case preserving the pre-change derivation. Build fixtures via the existing YAML-fixture + `sf.Load` pattern <!-- R2 -->
- [x] T003 Thread `display_state` through `src/go/fab/cmd/fab/panemap.go`: capture both returns of `status.DisplayStage` (line ~324); add a `displayState string` field to `paneRow` defaulting to the em-dash sentinel in BOTH the early-return row and the main constructor (mirroring `stage`); add `DisplayState *string \`json:"display_state"\`` to `paneJSON` immediately after `Stage`; map via `toNullable(r.displayState)` in `printPaneJSON`. Table renderer untouched <!-- R1 -->
- [x] T004 Add pane map tests in `src/go/fab/cmd/fab/panemap_test.go`: (a) `printPaneJSON` emits `display_state` correctly for a change-bearing row and `null` for an em-dash row; (b) `resolvePane` populates `displayState` from a real `.status.yaml` (TestResolvePanePRURL fixture pattern) and leaves the sentinel for a pane without a change; (c) snake_case field-name check includes `display_state`; (d) table output unchanged when `displayState` is set vs cleared <!-- R1 -->

### Phase 4: Polish

- [x] T005 Document `display_state` in `src/kit/skills/_cli-fab.md` § `fab pane map`: add the field to the `--json` flag row's snake_case field list with value set `active`/`ready`/`done`/`failed`/`pending`/`skipped` or `null`, nullability contract, and JSON-only (no table column) note. Edit ONLY under `src/kit/` <!-- R3 -->

## Execution Order

- T001 blocks T002 (tests assert the new tier) and T003 (panemap consumes the state half, including `failed`)
- T003 blocks T004
- T005 is independent, can run any time after T003 settles the field shape

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab pane map --json` rows carry `display_state` immediately after `stage`, sourced from `status.DisplayStage`'s state half and mapped via `toNullable()`
- [x] A-002 R2: `DisplayStage` returns `("review", "failed")` for a review-failed change with nothing active/ready; precedence is active → failed → ready → last done/skipped → first pending
- [x] A-003 R3: `src/kit/skills/_cli-fab.md` § `fab pane map` documents `display_state` with the full value set and null semantics

### Behavioral Correctness

- [x] A-004 R2: `failed` plus a later `active` stage yields the active stage — the new tier never preempts in-progress work; verified by a passing `status_test.go` case

### Scenario Coverage

- [x] A-005 R1: `panemap_test.go` asserts `display_state` is present and correct for a change-bearing pane and `null` for a pane without a change; `go test` passes

### Edge Cases & Error Handling

- [x] A-006 R1: the non-repo early-return row and the unloadable-`.status.yaml` path both carry the em-dash sentinel → JSON `null`; the human table output is byte-identical with and without a populated `displayState`

### Code Quality

- [x] A-007 Pattern consistency: new code mirrors the established nullable-field pattern (`repo`/`pr_url` precedent — em-dash sentinel in `paneRow`, `*string` via `toNullable` in `paneJSON`) and the camelCase field naming of surrounding code
- [x] A-008 No unnecessary duplication: `toNullable()` and existing test fixture patterns are reused; no new helper introduced (production code adds zero helpers; the test-only `displayStageFixture` builder follows the plan-sanctioned YAML+`sf.Load` pattern — see review Should-fix on sharing `loadFixture`'s body)

### Documentation Accuracy

- [x] A-009: the `_cli-fab.md` JSON field list matches the actual emitted fields and value set exactly (no drift between docs and `paneJSON`)

### Cross References

- [x] A-010: edits land only under `src/kit/` (never `.claude/skills/`); `pane.go` `PaneContext`, the table renderer, `CurrentStage()`, and run-kit files are untouched

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/kit/skills/fab-continue.md:42,55` (the `progress.review == failed` compensating guard and its dispatch-row parenthetical) — the parenthetical's premise "preflight's derived stage/state never yields a `failed` tier" is now false: preflight's `display_state` (preflight.go:76 → `status.DisplayStage`) surfaces `review`/`failed` directly, so the `review`/`failed` row could key on the normal dispatch key and the special-case progress-map read retired (skill-side cleanup, not auto-deleted here)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `paneRow` field named `displayState` (camelCase) | Matches surrounding unexported field convention (`windowIndex`, `prURL`) | S:85 R:95 A:95 D:90 |
| 2 | Certain | `DisplayStage` tests build fixtures via YAML + `sf.Load` in a temp dir | Existing `loadFixture` pattern in `status_test.go`; `StatusFile.Progress` is an unexported-raw-backed `yaml.Node`, not directly constructible | S:80 R:95 A:90 D:85 |
| 3 | Certain | No `docs/specs/skills/SPEC-_cli-fab.md` update — file does not exist | Constitution's skill→SPEC constraint has no corresponding SPEC file for the `_cli-fab` helper; intake's Impact section lists only `_cli-fab.md` | S:85 R:90 A:90 D:85 |

3 assumptions (3 certain, 0 confident, 0 tentative).
