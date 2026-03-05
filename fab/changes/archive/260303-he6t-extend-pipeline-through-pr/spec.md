# Spec: Extend Pipeline Through PR

**Change**: 260303-he6t-extend-pipeline-through-pr
**Created**: 2026-03-04
**Affected memory**: No memory files affected — workflow infrastructure change

## Non-Goals

- Changing the archive workflow — hydrate remains the knowledge capture gate; `/fab-archive` still requires `progress.hydrate == done`
- Adding confidence gates for ship/review-pr — these are integration stages, not planning
- Preemptive backfill of existing 6-stage changes — statusman tolerates missing stages
- Changing the `prs[]` array or `.pr-done` sentinel — these remain as supplementary signals

## Workflow Schema: New Pipeline Stages

### Requirement: Ship Stage Definition

The workflow schema (`fab/.kit/schemas/workflow.yaml`) SHALL define a `ship` stage after `hydrate` with the following properties:

- **id**: `ship`
- **name**: "Ship"
- **generates**: `null`
- **description**: "Commit, push, and create PR"
- **requires**: `[hydrate]`
- **required**: `true`
- **initial_state**: `pending`
- **allowed_states**: `[pending, active, done, skipped]`
- **commands**: `[git-pr]`

The `ship` stage SHALL NOT include `failed` in its allowed states. Git-pr fails fast and the user retries; there is no failed state to track.

#### Scenario: Ship stage is added to schema
- **GIVEN** the workflow schema at `fab/.kit/schemas/workflow.yaml`
- **WHEN** `statusman.sh all-stages` is invoked
- **THEN** the output SHALL include `ship` after `hydrate`

#### Scenario: Ship rejects failed state
- **GIVEN** a change with `progress.ship: active`
- **WHEN** `statusman.sh fail <change> ship` is invoked
- **THEN** the command SHALL exit with error "no valid transition"

### Requirement: Review-PR Stage Definition

The workflow schema SHALL define a `review-pr` stage after `ship` with the following properties:

- **id**: `review-pr`
- **name**: "PR Review"
- **generates**: `null`
- **description**: "Process PR review feedback"
- **requires**: `[ship]`
- **required**: `true`
- **initial_state**: `pending`
- **allowed_states**: `[pending, active, done, failed, skipped]`
- **commands**: `[git-pr-review]`

The `review-pr` stage SHALL support the `failed` state, analogous to the `review` stage.

#### Scenario: Review-PR stage is added to schema
- **GIVEN** the workflow schema
- **WHEN** `statusman.sh all-stages` is invoked
- **THEN** the output SHALL include `review-pr` after `ship`

#### Scenario: Review-PR supports fail transition
- **GIVEN** a change with `progress.review-pr: active`
- **WHEN** `statusman.sh fail <change> review-pr` is invoked
- **THEN** `progress.review-pr` SHALL become `failed`

### Requirement: Review-PR Transition Overrides

The workflow schema SHALL include a `review-pr` section under `transitions` with the same structure as `review`:

```yaml
review-pr:
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
    from: [done, ready, skipped]
    to: active
  - event: fail
    from: [active]
    to: failed
```

#### Scenario: Review-PR can restart from failed
- **GIVEN** a change with `progress.review-pr: failed`
- **WHEN** `statusman.sh start <change> review-pr` is invoked
- **THEN** `progress.review-pr` SHALL become `active`

### Requirement: Stage Numbering Update

The `stage_numbers` section in `workflow.yaml` SHALL map all 8 stages:

```yaml
stage_numbers:
  intake: 1
  spec: 2
  tasks: 3
  apply: 4
  review: 5
  hydrate: 6
  ship: 7
  review-pr: 8
```

#### Scenario: Stage numbers include new stages
- **GIVEN** the workflow schema
- **WHEN** `stage_numbers` is read
- **THEN** `ship` SHALL be 7 and `review-pr` SHALL be 8

### Requirement: Completion Rule Update

The `progression.completion` rule SHALL change from "hydrate stage has state=done or state=skipped" to "review-pr stage has state=done or state=skipped". The `progression.current_stage.fallback` SHALL change from `hydrate` to `review-pr`.

#### Scenario: Pipeline completion requires review-pr
- **GIVEN** a change with all stages through hydrate `done` but `ship: pending`, `review-pr: pending`
- **WHEN** pipeline completion is checked
- **THEN** the pipeline SHALL NOT be considered complete

## Status Manager: Stage Machine Extensions

### Requirement: Current Stage Fallback Update

In `statusman.sh`, the `get_current_stage` function's final fallback (line 312) SHALL return `review-pr` instead of `hydrate`. This ensures the "all done" fallback points to the new terminal stage.

#### Scenario: All stages done returns review-pr
- **GIVEN** a change where all 8 stages are `done`
- **WHEN** `statusman.sh current-stage <change>` is invoked
- **THEN** the output SHALL be `review-pr`

### Requirement: Auto-Logging for Review-PR

The `event_finish` function SHALL auto-log "passed" via logman when `stage == "review-pr"`, identical to the existing `review` auto-log behavior. The `event_fail` function SHALL auto-log "failed" for `review-pr` with the same pattern as `review`.

#### Scenario: Finish review-pr auto-logs passed
- **GIVEN** a change with `progress.review-pr: active`
- **WHEN** `statusman.sh finish <change> review-pr git-pr-review` is invoked
- **THEN** `progress.review-pr` SHALL become `done`
- **AND** logman SHALL be called with `review <change> "passed"`

#### Scenario: Fail review-pr auto-logs failed
- **GIVEN** a change with `progress.review-pr: active`
- **WHEN** `statusman.sh fail <change> review-pr git-pr-review rework-action` is invoked
- **THEN** `progress.review-pr` SHALL become `failed`
- **AND** logman SHALL be called with `review <change> "failed" "rework-action"`

### Requirement: Finish Hydrate Auto-Activates Ship

When `statusman.sh finish <change> hydrate` is invoked, the side-effect SHALL auto-activate the `ship` stage (set `progress.ship` from `pending` to `active`). This extends the existing chain: `finish hydrate → ship active → finish ship → review-pr active`.

#### Scenario: Finish hydrate activates ship
- **GIVEN** a change with `progress.hydrate: active` and `progress.ship: pending`
- **WHEN** `statusman.sh finish <change> hydrate` is invoked
- **THEN** `progress.hydrate` SHALL become `done`
- **AND** `progress.ship` SHALL become `active`

## Status Template: Progress Map Extension

### Requirement: Template Includes New Stages

The status template (`fab/.kit/templates/status.yaml`) SHALL include `ship: pending` and `review-pr: pending` in the progress map, after `hydrate: pending`.

#### Scenario: New change has 8-stage progress map
- **GIVEN** a new change is created via `changeman.sh new`
- **WHEN** `.status.yaml` is read
- **THEN** `progress` SHALL contain all 8 stages: intake, spec, tasks, apply, review, hydrate, ship, review-pr
- **AND** `ship` and `review-pr` SHALL both be `pending`

## Pipeline Skills: Full Pipeline Extension

### Requirement: fab-ff Pipeline Extension

`/fab-ff` SHALL extend its pipeline past hydrate to include ship and review-pr:

1. **Step 7 (existing)**: Hydrate — unchanged
2. **Step 8 (new): Ship** — invoke `/git-pr` behavior. On success: `statusman.sh finish <change> ship fab-ff`. On failure: STOP with error from git-pr.
3. **Step 9 (new): Review-PR** — invoke `/git-pr-review` behavior. On success: `statusman.sh finish <change> review-pr fab-ff`. On failure: STOP with error suggesting re-run.

Ship and review-pr have NO confidence gates. Resumability applies: skip if `progress.ship` or `progress.review-pr` is `done`.

#### Scenario: fab-ff runs through ship
- **GIVEN** a change at hydrate stage with all prior stages done
- **WHEN** `/fab-ff` is invoked and hydrate completes
- **THEN** the pipeline SHALL proceed to invoke `/git-pr` behavior
- **AND** on success, `progress.ship` SHALL become `done`

#### Scenario: fab-ff runs through review-pr
- **GIVEN** a change with `progress.ship: done` and `progress.review-pr: pending`
- **WHEN** `/fab-ff` resumes
- **THEN** the pipeline SHALL invoke `/git-pr-review` behavior
- **AND** on success, `progress.review-pr` SHALL become `done`

#### Scenario: fab-ff skips completed ship
- **GIVEN** a change with `progress.ship: done`
- **WHEN** `/fab-ff` is invoked
- **THEN** ship SHALL be skipped with "Skipping ship — already done."

### Requirement: fab-fff Pipeline Extension

`/fab-fff` SHALL extend its pipeline identically to `/fab-ff` — add Step 9 (Ship) and Step 10 (Review-PR) after hydrate, with the same behavior and status transitions.

#### Scenario: fab-fff completes full 8-stage pipeline
- **GIVEN** a change at intake stage
- **WHEN** `/fab-fff` runs the full pipeline
- **THEN** all 8 stages SHALL reach `done` state
- **AND** the output SHALL include `--- Ship ---` and `--- Review-PR ---` sections

### Requirement: Pipeline Complete Message Update

Both `/fab-ff` and `/fab-fff` SHALL update their completion message from "Pipeline complete. Change hydrated." to "Pipeline complete." since the pipeline now extends past hydrate.

#### Scenario: Completion message reflects full pipeline
- **GIVEN** a change completing the full pipeline
- **WHEN** the final output is displayed
- **THEN** the message SHALL be "Pipeline complete."
- **AND** `Next:` SHALL show `/fab-archive`

## Pipeline Navigation: State Table Updates

### Requirement: Preamble State Table Extension

The state table in `_preamble.md` SHALL add entries for the new stages:

| State | Available commands | Default |
|-------|-------------------|---------|
| hydrate | /git-pr, /fab-archive | /git-pr |
| ship | /git-pr-review | /git-pr-review |
| review-pr (pass) | /fab-archive | /fab-archive |
| review-pr (fail) | /git-pr-review | /git-pr-review |

The existing `hydrate` row remains unchanged — `/git-pr` is default, `/fab-archive` is available.

#### Scenario: After hydrate, next is git-pr
- **GIVEN** a change with `progress.hydrate: done`
- **WHEN** the state table is consulted
- **THEN** the default command SHALL be `/git-pr`

#### Scenario: After review-pr pass, next is archive
- **GIVEN** a change with `progress.review-pr: done`
- **WHEN** the state table is consulted
- **THEN** the default command SHALL be `/fab-archive`

### Requirement: fab-continue Dispatch Extension

`/fab-continue` SHALL handle the new stages in its dispatch table:

| Derived stage | State | Action |
|---------------|-------|--------|
| `ship` | `active`/`ready` | Execute `/git-pr` behavior → on completion `finish <change> ship fab-continue` |
| `review-pr` | `active`/`ready` | Execute `/git-pr-review` behavior → pass: `finish <change> review-pr fab-continue`. Fail: `fail <change> review-pr` |

The stage argument list SHALL accept `ship` and `review-pr` as valid reset targets. The pipeline description SHALL update from "6-stage" to "8-stage".

#### Scenario: fab-continue dispatches ship
- **GIVEN** a change with `progress.ship: active`
- **WHEN** `/fab-continue` is invoked
- **THEN** it SHALL execute `/git-pr` behavior

#### Scenario: fab-continue dispatches review-pr
- **GIVEN** a change with `progress.review-pr: active`
- **WHEN** `/fab-continue` is invoked
- **THEN** it SHALL execute `/git-pr-review` behavior

## Ship Skill: Git-PR Statusman Integration

### Requirement: Ship Stage Tracking in Git-PR

`/git-pr` SHALL integrate with statusman to track the `ship` stage:

1. **Before Step 3** (Execute Pipeline): If an active change resolves and `progress.ship` is `pending` or `active`, call `statusman.sh start <change> ship git-pr` (idempotent if already active).
2. **After Step 4c** (Write PR Sentinel): If ship stage was started, call `statusman.sh finish <change> ship git-pr`.
3. **On failure**: Do NOT call statusman fail (ship has no failed state). The stage remains `active` for user retry.

The statusman calls SHALL be best-effort — failures are silently ignored to avoid blocking the PR workflow. The existing PR URL recording (Steps 4-4c) is unchanged.

#### Scenario: git-pr tracks ship stage
- **GIVEN** a change with `progress.ship: pending` and an active change resolved
- **WHEN** `/git-pr` successfully creates a PR
- **THEN** `progress.ship` SHALL become `done`
- **AND** `progress.review-pr` SHALL become `active` (via finish auto-activate)

#### Scenario: git-pr without active change skips statusman
- **GIVEN** no active change (changeman resolve fails)
- **WHEN** `/git-pr` creates a PR
- **THEN** no statusman calls SHALL be made
- **AND** the PR creation SHALL succeed normally

#### Scenario: git-pr re-run with ship already done
- **GIVEN** a change with `progress.ship: done`
- **WHEN** `/git-pr` is re-invoked
- **THEN** statusman calls SHALL be skipped (ship is already done)
- **AND** existing PR detection works as before

## Review-PR Skill: Rename and Statusman Integration

### Requirement: Skill Rename

The skill file `fab/.kit/skills/git-review.md` SHALL be renamed to `fab/.kit/skills/git-pr-review.md`. The skill's metadata SHALL update:

- **name**: `git-pr-review` (was `git-review`)
- **description**: unchanged

All references to `/git-review` in other skills and documentation SHALL be updated to `/git-pr-review`.

#### Scenario: Skill file renamed
- **GIVEN** the skill file at `fab/.kit/skills/git-review.md`
- **WHEN** the change is applied
- **THEN** the file SHALL exist at `fab/.kit/skills/git-pr-review.md`
- **AND** `fab/.kit/skills/git-review.md` SHALL no longer exist

### Requirement: Review-PR Stage Tracking

`/git-pr-review` SHALL integrate with statusman to track the `review-pr` stage:

1. **On start**: If an active change resolves, call `statusman.sh start <change> review-pr git-pr-review` (handles both pending and failed → active).
2. **On success** (comments processed, pushed): Call `statusman.sh finish <change> review-pr git-pr-review`.
3. **On failure** (Copilot timeout, no reviews, processing error): Call `statusman.sh fail <change> review-pr git-pr-review` if the stage was started. The `fail` call is omitted if no change is active.
4. **On no-op** (no actionable comments, no reviews needed): Call `statusman.sh finish <change> review-pr git-pr-review` — a successful outcome.

Statusman calls SHALL be best-effort — failures silently ignored.

#### Scenario: git-pr-review tracks review-pr stage
- **GIVEN** a change with `progress.review-pr: active`
- **WHEN** `/git-pr-review` processes comments and pushes fixes
- **THEN** `progress.review-pr` SHALL become `done`

#### Scenario: git-pr-review handles failure
- **GIVEN** a change with `progress.review-pr: active`
- **WHEN** `/git-pr-review` encounters a Copilot timeout
- **THEN** `progress.review-pr` SHALL become `failed`

### Requirement: Phase Sub-State Tracking

`/git-pr-review` SHALL update `stage_metrics.review-pr.phase` as it progresses through its workflow:

| Phase value | When |
|-------------|------|
| `waiting` | After requesting Copilot review, before reviews arrive |
| `received` | Reviews detected (existing or just arrived) |
| `triaging` | Classifying comments as actionable vs informational |
| `fixing` | Applying fixes from actionable comments |
| `pushed` | Fixes committed and pushed |

Phase updates use `yq` to write directly to `stage_metrics.review-pr.phase` in `.status.yaml`. The `reviewer` field is set to the reviewer login (e.g., `copilot-pull-request-reviewer[bot]` or `@username`).

#### Scenario: Phase tracking through Copilot review
- **GIVEN** a change with `progress.review-pr: active`
- **WHEN** `/git-pr-review` requests and receives Copilot review
- **THEN** `stage_metrics.review-pr.phase` SHALL progress: `waiting` → `received` → `triaging` → `fixing` → `pushed`
- **AND** `stage_metrics.review-pr.reviewer` SHALL be set

## Changeman: Display Function Updates

### Requirement: Changeman Stage Functions

The `changeman.sh` functions `stage_number()`, `next_stage()`, and `default_command()` SHALL be updated to include the new stages:

**stage_number()**: Add `ship) echo 7 ;;` and `review-pr) echo 8 ;;`. The display format SHALL change from `(N/6)` to `(N/8)`.

**next_stage()**: Add `hydrate) echo "ship" ;;`, `ship) echo "review-pr" ;;`, `review-pr) echo "" ;;`. Remove the current `hydrate) echo "" ;;` terminal.

**default_command()**: Add `ship) echo "/git-pr-review" ;;`, `review-pr) echo "/fab-archive" ;;`. Update `hydrate) echo "/git-pr" ;;` (unchanged).

#### Scenario: Changeman displays 8-stage progress
- **GIVEN** an active change at the ship stage
- **WHEN** `changeman.sh switch <change>` is invoked
- **THEN** the output SHALL show `Stage: ship (7/8) — active`

#### Scenario: Changeman routes after ship
- **GIVEN** an active change at the ship stage
- **WHEN** the next command is derived
- **THEN** the routing SHALL show `review-pr (via /git-pr-review)`

## Backward Compatibility: Missing Stage Tolerance

### Requirement: Statusman Tolerates Missing Stages

When `statusman.sh` reads a `.status.yaml` that lacks `ship` and `review-pr` in the progress map (created before this change), the `get_progress_map` function SHALL default missing stages to `"pending"` via the existing `yq ".progress.${stage} // \"pending\""` pattern.

No migration script is needed. Old changes with 6-stage progress maps continue to work — ship and review-pr default to pending. If `/git-pr` is later invoked on such a change, the statusman `start`/`finish` calls will write the new stage keys into the progress map (lazy append).

#### Scenario: Old 6-stage change treated correctly
- **GIVEN** a `.status.yaml` with only intake through hydrate in the progress map
- **WHEN** `statusman.sh progress-map <change>` is invoked
- **THEN** `ship` SHALL show `pending` and `review-pr` SHALL show `pending`

#### Scenario: Lazy append on first use
- **GIVEN** an old 6-stage change with `progress.hydrate: done`
- **WHEN** `statusman.sh start <change> ship git-pr` is invoked
- **THEN** `progress.ship` SHALL be written as `active` into `.status.yaml`
- **AND** the change SHALL function normally with the new stages

## Deprecated Requirements

### Git-Review Skill Name

**Reason**: Renamed to `/git-pr-review` to distinguish from the internal `review` stage (code validation sub-agent).
**Migration**: All references to `/git-review` in skills and documentation are updated to `/git-pr-review`. The deployed copy in `.claude/skills/` is updated via `fab-sync.sh`.

## Design Decisions

1. **Lifecycle language for stages, tool language for skills**: Stage names (`ship`, `review-pr`) use lifecycle concepts; skill names (`/git-pr`, `/git-pr-review`) reference the driving tool. This preserves Constitution Principle V (Portability) — non-git projects can skip these stages or wire different skills.
   - *Why*: Decouples the state machine from the specific tool implementation
   - *Rejected*: Using `git-pr`/`git-pr-review` as stage names — couples the state machine to git

2. **No `failed` state for ship**: Git-pr fails fast and the user retries. Adding a failed state would require a recovery path with no clear benefit.
   - *Why*: Simplicity — fail fast + retry is sufficient for a "create PR" operation
   - *Rejected*: Adding failed state for symmetry with review-pr — unnecessary complexity

3. **No preemptive backfill**: Statusman already defaults missing progress keys to `pending`. Lazy append on first use avoids migration code entirely.
   - *Why*: Zero migration code, zero risk to existing changes, yq handles missing keys gracefully
   - *Rejected*: Migration script to add stages to all existing changes — risk for no benefit

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Both ff and fff extend through ship and review-pr | Confirmed from intake #1 — user explicitly directed full pipeline extension | S:95 R:65 A:90 D:95 |
| 2 | Certain | Add `ship` and `review-pr` as first-class stages in workflow.yaml | Confirmed from intake #2 — user confirmed lifecycle framing | S:90 R:55 A:85 D:90 |
| 3 | Certain | `ship` does NOT support `failed` state | Confirmed from intake #3 — fail fast, user retries | S:90 R:75 A:85 D:90 |
| 4 | Certain | No confidence gates for ship/review-pr | Confirmed from intake #4 — integration stages, not planning | S:90 R:70 A:85 D:90 |
| 5 | Certain | `active` state covers both waiting and working (no new states) | Confirmed from intake #5 — phase sub-state handles distinction | S:90 R:80 A:90 D:90 |
| 6 | Certain | Rename `/git-review` to `/git-pr-review` | Confirmed from intake #6 — user confirmed naming | S:95 R:70 A:90 D:95 |
| 7 | Certain | `review-pr` supports `failed` state | Confirmed from intake #7 — analogous to review stage, review failures are real | S:85 R:70 A:80 D:85 |
| 8 | Certain | Sub-state tracking via `stage_metrics.review-pr.phase` | Confirmed from intake #8 — extends existing pattern | S:80 R:80 A:80 D:80 |
| 9 | Certain | No preemptive backfill for in-flight changes | Confirmed from intake #9 — lazy append on first use | S:80 R:80 A:85 D:80 |
| 10 | Certain | `review-pr` always attempted after ship; auto-skip only when no git/remote | Confirmed from intake #10 — skill handles "no reviews" as normal done | S:80 R:75 A:80 D:80 |
| 11 | Confident | Auto-log for review-pr uses same logman `review` subcommand as review | Same logging mechanism; logman review subcommand is already generic | S:70 R:85 A:75 D:70 |
| 12 | Confident | Archive hydrate guard unchanged — archive remains available after hydrate | Archive is about knowledge capture (hydrate done); ship/review-pr are integration | S:75 R:80 A:80 D:70 |
| 13 | Confident | git-pr statusman calls are best-effort (silently ignored on failure) | Consistent with existing git-pr pattern for PR URL recording | S:75 R:85 A:80 D:75 |
| 14 | Confident | Phase values: waiting, received, triaging, fixing, pushed | Maps naturally to git-pr-review's Step 2-5 workflow phases | S:70 R:85 A:70 D:65 |

14 assumptions (10 certain, 4 confident, 0 tentative, 0 unresolved).
