# Plan: Light Lane — Plan-Time Fork to Inline Execution for Small Changes

**Change**: 260811-3ol6-light-lane-inline-small-changes
**Intake**: `intake.md`

## Requirements

### Orchestration: Inline Plan Co-Generation

#### R1: plan.md co-generated inline at apply entry, both lanes
The `_pipeline.md` bracket MUST co-generate `plan.md` inline in the orchestrator's own context at apply entry (per `_generation.md`'s Plan Generation Procedure), in BOTH the light and full lanes. In the full lane, the dispatched apply worker MUST receive the finished plan via the existing plan-exists seam (`co-generate plan.md ... unless plan.md exists`), so its cold start does task execution only.

- **GIVEN** an intake-gated `/fab-ff` or `/fab-fff` run with no `plan.md`
- **WHEN** the bracket reaches apply entry (Step 1)
- **THEN** the orchestrator co-generates `plan.md` inline before any apply dispatch
- **AND** a subsequently dispatched full-lane apply worker skips plan generation because `plan.md` exists

#### R2: Large-scope guard
When the intake's affected scope is obviously large, the orchestrator MAY skip inline co-gen and dispatch apply-with-co-gen exactly as before — graceful degradation to the shipped path; the exact criterion is left to orchestrator judgment in prose.

- **GIVEN** an intake whose affected scope is obviously large
- **WHEN** the bracket reaches apply entry
- **THEN** the orchestrator MAY dispatch the apply worker with plan generation included, exactly as the pre-change behavior

### Orchestration: The Lane Fork

#### R3: One-time task-count fork
Immediately after inline co-gen, the orchestrator MUST fork once on the number of task entries in `plan.md` `## Tasks` (all phases): count ≤ 5 → LIGHT lane; > 5 → FULL lane. The threshold is hardcoded at 5 in v1 (the `light_max_tasks` config knob is a recorded follow-up). There is NO promotion valve — the lane never changes mid-run; scope growth discovered during rework rides the rework backstop.

- **GIVEN** a freshly co-generated `plan.md`
- **WHEN** the orchestrator counts its `## Tasks` entries
- **THEN** the run proceeds light (≤ 5) or full (> 5), and the lane decision is never revisited mid-run

#### R4: `--light` / `--full` overrides
`/fab-ff` and `/fab-fff` MUST accept per-invocation `--light` and `--full` flags that skip the task-count check and force the lane. Passing both flags MUST be rejected as a usage error in prose.

- **GIVEN** a `/fab-ff --light` or `/fab-fff --full` invocation
- **WHEN** the bracket reaches the fork
- **THEN** the count check is skipped and the named lane is taken
- **AND** passing both flags together is rejected as a usage error

### Orchestration: Light-Lane Execution Loci

#### R5: Inline stages in the light lane
In the LIGHT lane, task execution (apply) and hydrate MUST run inline in the orchestrator's own context, following the same `/fab-continue` Behavior sections the dispatched workers follow. For `/fab-fff` only, ship (`/git-pr` behavior) and review-pr (`/git-pr-review` behavior) MUST also run inline (today's standalone path); `/fab-ff`'s light lane ends at hydrate. Inline stages MUST skip `fab resolve-agent` and run on the session model (an undispatched stage MAY report the configured profile but MUST NOT switch the session model). Inline ship/review-pr keep managing their own stage transitions exactly as standalone.

- **GIVEN** a light-lane run
- **WHEN** the run reaches apply task execution, hydrate, or (fff) ship/review-pr
- **THEN** the orchestrator executes that stage's behavior inline with no sub-agent dispatch and no `fab resolve-agent` call

#### R6: Review always dispatched
Review MUST remain a fresh dispatched worker in BOTH lanes — never inline (reviewer independence). Light-lane rework re-reviews via a fresh review worker exactly as the full lane does.

- **GIVEN** a light-lane run reaching review
- **WHEN** review is dispatched
- **THEN** it is a fresh worker via the Stage Dispatch Procedure, identical to the full lane

#### R7: Light rework stays inline, same budget
Light-lane rework MUST stay inline under the same `{max_cycles}` budget (code-review.md Rework Budget, default 3) with the same fail+reset per-cycle choreography; exhaustion MUST park `review: failed` exactly as in the full lane, and a parked light run re-enters however the user chooses, including `--full`.

- **GIVEN** a failed review verdict in a light-lane run
- **WHEN** the auto-rework loop cycles
- **THEN** rework executes inline in the orchestrator's context under the same cycle cap, and exhaustion leaves `review: failed`

### Orchestration: Unchanged Machinery

#### R8: Zero state-machine changes; full lane verbatim
Both state machines (the stage graph and the per-stage states/transitions in `internal/status/status.go`) MUST gain zero new states and zero new transitions; the same `finish`/`fail`/`reset` choreography fires in the same order in both lanes; the review cycle-count invariant (finish apply is the only counted review re-entry) holds verbatim. The FULL lane MUST remain today's bracket verbatim except that the plan pre-exists; all three dispatch adapters, worker continuation, and recovery budgets are unchanged, with `_preamble.md` § Worker Continuation becoming a full-lane-only concern. v1 is skill-prose only: zero Go changes, zero `.status.yaml` schema changes, zero config registry changes.

- **GIVEN** the completed change
- **WHEN** the diff is inspected
- **THEN** no `.go` file, `.status.yaml` schema, or config registry entry has changed
- **AND** the full-lane dispatch path (adapters, worker continuation, recovery budgets) is textually unchanged apart from the plan-exists note

### Documentation: Mirrors and Sweeps

#### R9: SPEC mirrors updated
Because the change alters flow and sub-agent structure of `_pipeline.md`, `fab-ff.md`, and `fab-fff.md`, their condensed structural mirrors (`docs/specs/skills/SPEC-_pipeline.md`, `SPEC-fab-ff.md`, `SPEC-fab-fff.md` — title + header + Flow + Tools + Sub-agents) MUST be updated in the same change (Constitution 1.5.0).

- **GIVEN** the edited skills
- **WHEN** the mirrors are read
- **THEN** their Flow and Sub-agents sections reflect the two-lane bracket, inline co-gen, and the `--light`/`--full` flags

#### R10: Aggregate-spec and repo-wide sweep
Aggregate specs restating per-skill facts MUST be swept (`docs/specs/skills.md` `/fab-ff` and `/fab-fff` sections, `docs/specs/glossary.md` — including a new "light lane" term entry, `docs/specs/architecture.md`), and every repo-wide occurrence of invalidated claims (e.g. every post-intake stage being dispatched, plan co-gen happening inside the apply worker) MUST be updated, including `*_test.go` comments and user-facing string literals if any turn up. The in-tree design reference `docs/findings/light-mode-state-machines.md` MUST have its "Status: design proposal" header updated to reflect adoption.

- **GIVEN** the prose edits
- **WHEN** the old claims are grepped repo-wide
- **THEN** no remaining occurrence asserts the pre-change execution locus, and the findings doc no longer labels the adopted mechanism a proposal

### Non-Goals

- `light_max_tasks` config knob — recorded follow-up, threshold stays hardcoded at 5
- A `weight` event in `.history.jsonl` — recorded follow-up
- Requesting the Copilot review at ship time — recorded follow-up
- Any Go, `.status.yaml` schema, or config registry change — v1 is skill-prose only
- Memory hydration (`docs/memory/` edits) — owned by the hydrate stage, not apply

### Design Decisions

#### Plan-time task-count fork over intake-time type heuristic
**Decision**: The lane decision is a one-time fork on `plan.md` task count (≤ 5 → light) made immediately after inline co-gen, not an intake-time change-type heuristic.
**Why**: Loom-archive evidence (972 completed changes, 927 with recoverable task counts) shows rework rate is a clean gradient over plan size (8% at 1–3 tasks, 7% at 4–5, 11% at 6–8, 16% at 9–12, 29% at 13+), while type is noise (68% of type-eligible changes outgrow the size threshold; 154 small feat/fix changes escape it — recall ~26%, precision ~32%).
**Rejected**: Change-type entry heuristic (infer light from docs/chore/test/ci at intake) — killed by the same data.
*Introduced by*: 260811-3ol6-light-lane-inline-small-changes

#### No promotion valve
**Decision**: Light rework stays inline under the same `max_cycles` budget with exhaustion parking; there is no light→full mid-run promotion.
**Why**: Every diff — light or full — passes the same independent review, so a misclassified small change wastes bounded inline rework cycles but can never ship unreviewed; zero mid-run mode machinery.
**Rejected**: Promotion valve (light→full on 2nd rework cycle) — bounded rework + exhaustion parking is a sufficient backstop.
*Introduced by*: 260811-3ol6-light-lane-inline-small-changes

## Tasks

### Phase 1: Core Skill Edits

- [x] T001 Edit `src/kit/skills/_pipeline.md`: inline plan co-gen step at apply entry (both lanes, with the large-scope MAY-skip guard), the one-time ≤5-task fork, light-lane execution rules (inline apply task execution + hydrate; review always dispatched), and rework-locus notes (inline light rework; Worker Continuation becomes full-lane-only) <!-- R1 R2 R3 R5 R6 R7 R8 --> <!-- rework: resume lane-derivation gap — state the explicit re-derivation rule (count plan.md ## Tasks; --light/--full per invocation) for resumed runs where Step 1/co-gen is skipped -->
- [x] T002 [P] Edit `src/kit/skills/fab-ff.md`: `--light`/`--full` arguments and the light-lane scope note (task execution + hydrate; its terminal) — point at `_pipeline.md` for lane mechanics <!-- R3 R4 R5 --> <!-- rework: collapse the review-stays-dispatched restatement at fab-ff.md:36 to a pointer at _pipeline.md § Light Lane -->
- [x] T003 [P] Edit `src/kit/skills/fab-fff.md`: `--light`/`--full` arguments; Steps 4–5 (ship, review-pr) run inline in the light lane; the Step 5 synchronous-poll directive becomes moot in the light lane <!-- R4 R5 --> <!-- rework: owner-or-pointer violation — keep the fff-specific binding (Steps 4-5 inline; poll directive moot) but collapse the mechanical glosses ('no fab resolve-agent and no dispatch', 'today's standalone path', yield-seam restatement) to the § Light Lane pointer -->

### Phase 2: Mirrors and Sweeps

- [x] T004 Update the condensed structural mirrors `docs/specs/skills/SPEC-_pipeline.md`, `SPEC-fab-ff.md`, `SPEC-fab-fff.md` (Flow + Tools + Sub-agents) to match the Phase-1 edits <!-- R9 --> <!-- obsoleted mid-pipeline: PR #582 deleted the SPEC tree and retired the mirror rule (constitution 1.6.0); the reviewed mirror content was transplanted into docs/specs/skills.md's absorbed _pipeline flow skeleton + /fab-fff flow lines at rebase -->
- [x] T005 [P] Update aggregate specs: `docs/specs/skills.md` (`/fab-ff` and `/fab-fff` sections — inline co-gen, lane fork, flags), `docs/specs/glossary.md` (new "light lane" term entry; adjust any invalidated per-skill rows), `docs/specs/architecture.md` (only if a claim is invalidated) <!-- R10 -->
- [x] T006 [P] Update `docs/findings/light-mode-state-machines.md` "Status: design proposal" header to reflect adoption <!-- R10 -->
- [x] T007 Repo-wide grep sweep of invalidated claims (every post-intake stage dispatched; plan co-gen inside the apply worker), including `*_test.go` comments and user-facing string literals; update every occurrence in the sweep classes <!-- R10 --> <!-- rework: sweep miss — fab-continue.md:120/:198 apply+hydrate blockquotes still claim 'always runs in a dispatched sub-agent' (must-fix); also scope the universal claims at fab-continue.md:34/:61 to the sequencer, matching the stage-models.md carve-out; optional: short scoping clause at _preamble.md:322 -->

### Phase 3: Verification

- [x] T008 Verify: `git status` shows zero `.go` changes and zero edits under `.claude/skills/`; re-grep swept phrases clean; each new rule stated once with pointers elsewhere <!-- R8 R10 --> <!-- rework: re-verify after rework edits -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `_pipeline.md` Step 1 states inline `plan.md` co-generation at apply entry in both lanes, with the full-lane apply worker receiving the finished plan via the plan-exists seam
- [x] A-002 R2: The large-scope guard is stated as a MAY that degrades to dispatching apply-with-co-gen as before
- [x] A-003 R3: The one-time fork on `## Tasks` count (≤ 5 → light, > 5 → full, threshold hardcoded) is documented; no promotion valve exists
- [x] A-004 R4: `--light`/`--full` are documented on both drivers and stated to be mutually exclusive
- [x] A-005 R5: Light-lane inline loci are documented (apply task execution + hydrate; fff adds ship/review-pr); inline stages skip `fab resolve-agent` and run on the session model; inline ship/review-pr keep their own transitions
- [x] A-006 R6: Review is documented as a fresh dispatched worker in both lanes
- [x] A-007 R7: Light-lane rework is documented as inline under the same `{max_cycles}` budget and fail+reset choreography, with exhaustion parking `review: failed`
- [x] A-008 R8: The prose asserts zero new states/transitions and full lane = today's bracket (plan pre-exists); worker continuation is scoped full-lane-only; no `.go`, `.status.yaml` schema, or config registry change appears in the diff
- [x] A-009 R9: `SPEC-_pipeline.md`, `SPEC-fab-ff.md`, and `SPEC-fab-fff.md` reflect the two-lane bracket, inline co-gen, and the flags
- [x] A-010 R10: Aggregate specs and the glossary are swept (incl. a "light lane" glossary entry); the findings doc header reflects adoption; a repo-wide grep of the old claims returns no stale occurrences

### Scenario Coverage

- [x] A-011 R3/R4: A reader of `_pipeline.md` + `fab-ff.md` can trace: gate → inline co-gen → count ≤ 5 → inline tasks → dispatched review → inline hydrate (ff), and the `--full` override path to today's dispatched bracket

### Edge Cases & Error Handling

- [x] A-012 R2/R3: The large-scope MAY-skip, mid-rework scope growth (rides the rework backstop), and parked-light re-entry via `--full` are each covered in prose

### Code Quality

- [x] A-013 Pattern consistency: new prose matches the surrounding skill files' style and density; the bracket owns lane mechanics and drivers point at it
- [x] A-014 No unnecessary duplication: each new rule is stated exactly once (owner) with pointers elsewhere; no restatement of lane mechanics in driver files
- [x] A-015 Canonical sources only: edits land in `src/kit/skills/` and `docs/` — zero edits under `.claude/skills/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Task breakdown is 8 prose-editing tasks in 3 phases (core skills → mirrors/sweeps → verification); no code or test tasks | Intake fixes v1 as skill-prose only; impact list names every edit target | S:90 R:90 A:90 D:90 |
| 2 | Confident | `--light` and `--full` are mutually exclusive; passing both is a usage error in prose | Intake assumption 15; obvious default for contradictory flags | S:35 R:90 A:75 D:65 |
| 3 | Confident | The findings doc's "Status: design proposal" header is updated to reflect adoption | Intake assumption 17; present-truth would be violated by shipping an adopted mechanism labeled "proposal" | S:30 R:90 A:60 D:50 |
| 4 | Certain | `/fab-continue`'s apply contract is unchanged — its dispatched apply worker still co-generates `plan.md` when absent; the bracket's inline co-gen rides the existing plan-exists seam | Intake impact list names only `_pipeline.md`, `fab-ff.md`, `fab-fff.md`; the seam is already in the apply contract | S:90 R:90 A:90 D:90 |

4 assumptions (2 certain, 2 confident, 0 tentative).

## Deletion Candidates

None — this change adds lane prose without making existing code or prose redundant (the fab-fff Step 5 synchronous-poll directive stays load-bearing for the full lane). Recorded by the sequencer on the review worker's behalf (read-only block contract).
