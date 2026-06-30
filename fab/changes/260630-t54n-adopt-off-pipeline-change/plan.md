# Plan: Adopt an off-pipeline change into the Fab pipeline

**Change**: 260630-t54n-adopt-off-pipeline-change
**Intake**: `intake.md`

## Requirements

### Adopt Skill: Orchestrator

#### R1: A new `/fab-adopt` skill brings a completed off-pipeline change into the Fab pipeline
A new skill file `src/kit/skills/fab-adopt.md` SHALL be a thin orchestrator that reuses existing skills as sub-agents (the `/fab-proceed` / `/fab-ff` pattern). It SHALL accept an optional `<slug>` argument (derived from branch name / PR title if omitted), read the `_preamble` always-load layer, and declare `helpers: [_srad, _generation, _review, _pipeline]`. It SHALL enter the real pipeline late with **apply skipped** — every other stage runs for real.

- **GIVEN** a feature branch with code authored without fab and an OPEN (or not-yet-created) PR
- **WHEN** the user runs `/fab-adopt`
- **THEN** the skill reconstructs intake + a thin plan from the diff, marks apply `skipped`, runs review (outward-only) → hydrate → ship (Meta retrofit) → review-pr, and lands the change in the normal pipeline tail

#### R2: Adopt guards reject out-of-scope states before any mutation
`/fab-adopt` Step 0 SHALL STOP (no mutation) when: the branch is detached HEAD or the default branch (reusing `/git-pr`'s guard messages); the PR `state == MERGED` (scenario A — retroactive backfill, out of scope); a fab change already maps to the current branch (`fab change resolve "$(git branch --show-current)"` succeeds — already in the pipeline, point at `/fab-continue`); or the diff against the merge-base is empty (nothing to adopt). `OPEN` and `none` PR states both proceed.

- **GIVEN** a branch whose PR is already MERGED
- **WHEN** `/fab-adopt` runs Step 0
- **THEN** it STOPs with the scenario-A out-of-scope message and makes no `.status.yaml` or git mutation

#### R3: Intake + thin plan are generated in one main-session pass from the diff
`/fab-adopt` SHALL generate `intake.md` and `plan.md` in a single main-session pass (the same agent, not a dispatched apply), reading the diff and PR body once. It SHALL run `fab change new --slug {slug}` against the current branch and activate it, reconstruct `intake.md` via the **Intake-from-Diff Procedure**, present a human-confirmation checkpoint, then on confirm advance+finish intake and write a **minimal** `plan.md` via the **Plan-from-Diff Procedure**.

- **GIVEN** an adopted diff and PR body
- **WHEN** `/fab-adopt` reaches Steps 1–2
- **THEN** the same agent writes both `intake.md` (full) and `plan.md` (thin) without re-reading the diff, with a human-confirmation checkpoint between them

#### R4: Adopt sets honest state via existing transitions — apply skipped, review active
`/fab-adopt` SHALL achieve `apply: skipped, review: active, hydrate/ship/review-pr: pending` using only existing `fab status` transitions: `fab status skip {name} apply` (cascades all downstream → skipped) then `fab status reset {name} review {driver}` (skipped → active, cascades its downstream → pending). It SHALL record the off-pipeline fact via `fab status set-summary {name} "..."`. **No Go change.**

- **GIVEN** an activated adopted change at apply
- **WHEN** `/fab-adopt` runs the Step 2 state transition
- **THEN** `.status.yaml` shows `apply: skipped`, `review: active`, downstream `pending` — and no new Go transition was added

#### R5: Adopt dispatches review outward-only, then hydrate, ship, review-pr
`/fab-adopt` SHALL dispatch review as a sub-agent with `mode: outward-only` (per R6), own the verdict transition (pass → `finish review`; fail → auto-rework per `_pipeline.md` budget or hand back), then dispatch hydrate verbatim (`finish hydrate`), then ship via `/git-pr {name}` (Meta retrofit per R8), then land in review-pr via `finish ship`. It SHALL print an honest-state summary noting only apply is `skipped`.

- **GIVEN** an adopted change at review (active)
- **WHEN** `/fab-adopt` runs Steps 3–6
- **THEN** review runs outward-only, hydrate writes memory, ship retrofits the PR Meta, and the change lands in review-pr with `Next: /git-pr-review`

### Review: Mode Parameter

#### R6: `_review.md` gains a general `mode` parameter (`full` | `outward-only`)
`src/kit/skills/_review.md` SHALL add a **Review Mode** concept: a `mode` parameter with values `full` (default — inward + outward) and `outward-only` (outward sub-agent only). It SHALL NOT add a speculative `inward-only`. The inward Preconditions (`plan.md` MUST exist with `## Tasks`/`## Acceptance`; all tasks `[x]`) SHALL be checked **only** in `full` mode. Parallel Dispatch SHALL dispatch only the sub-agent(s) selected by `mode`. Findings Merge / pass-fail SHALL be unchanged; an empty outward result (all `review_tools` disabled/unavailable → zero findings) SHALL **pass**. Default is `full` when the param is omitted, so all existing callers are unaffected.

- **GIVEN** a review dispatch with `mode: outward-only`
- **WHEN** `_review.md` runs
- **THEN** inward Preconditions are skipped, only the outward sub-agent is dispatched, and zero findings → pass
- **AND** **GIVEN** no `mode` param **WHEN** any existing caller dispatches review **THEN** behavior is identical to today (`full`)

### Generation: Diff-Based Procedures

#### R7: `_generation.md` gains Intake-from-Diff and Plan-from-Diff procedures
`src/kit/skills/_generation.md` SHALL add two procedures. **Intake-from-Diff Procedure**: reconstruct `intake.md` from the branch diff + PR body — Origin = `adopted from {PR or branch}`; Why/What-Changes synthesised from the diff; Affected Memory inferred from which `docs/memory/` domains the diff touches; Impact from changed paths; apply SRAD + `fab score`. **Plan-from-Diff Procedure**: write a deliberately thin `plan.md` — plain-language `## Requirements` (the only part hydrate reads; effort concentrates here), an all-`[x]` `## Tasks` stub, an all-`[x]` `## Acceptance` stub, **no** R#/T#/A# IDs, GIVEN/WHEN/THEN, phases, or `[P]` markers (the apply↔review traceability loop never runs for adopted changes); carry a header note that apply was skipped and the plan is reverse-engineered to feed hydrate.

- **GIVEN** an adopted change
- **WHEN** `/fab-adopt` invokes the two new `_generation.md` procedures
- **THEN** `intake.md` is reconstructed from the diff and `plan.md` carries plain-language requirements with all-`[x]` task/acceptance stubs and no traceability scaffolding
- **AND** the three heading literals `## Requirements` / `## Tasks` / `## Acceptance` (the stable parser contract) are present

### Ship: PR Body Meta-Retrofit

#### R8: `/git-pr` retrofits the `## Meta` block onto an existing OPEN PR
`src/kit/skills/git-pr.md` SHALL close the gap that `## Meta` is injected only on PR **create**. On the existing-OPEN-PR path: if the PR body lacks a `## Meta` block, render it via `fab pr-meta {name} --type {type} --issues "{issues}"` and apply with `gh pr edit --body-file -` (stdin). The retrofit SHALL be gated on body-lacks-`## Meta` for idempotency (a second run is a no-op). **No Go change** — `fab pr-meta` / `prmeta.Render` already exist and are reused.

- **GIVEN** an OPEN PR whose body lacks `## Meta`
- **WHEN** `/git-pr` runs its existing-OPEN-PR path
- **THEN** it renders Meta via `fab pr-meta` and applies it with `gh pr edit --body-file -`
- **AND** **GIVEN** an OPEN PR whose body already has `## Meta` **WHEN** `/git-pr` re-runs **THEN** it makes no edit (idempotent)

### Discoverability & Documentation

#### R9: `/fab-adopt` is discoverable in `/fab-help`
The new skill SHALL be listed by `/fab-help`. `fab fab-help` auto-scans skill frontmatter, so a valid `name`/`description` frontmatter makes it appear; it SHALL be grouped correctly (Planning) via `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go` so it does not fall into the "Other" bucket.

- **GIVEN** the new `fab-adopt.md` skill exists with valid frontmatter
- **WHEN** the user runs `/fab-help`
- **THEN** `/fab-adopt` is listed under the Planning group

#### R10: All SPEC mirrors and aggregate specs enumerating skills/stages are swept
Per Constitution (skill change MUST update its SPEC mirror) and the project Sibling & Mirror Sweep discipline, this change SHALL: create `docs/specs/skills/SPEC-fab-adopt.md`; update `SPEC-_review.md`, `SPEC-_generation.md`, `SPEC-git-pr.md`; and update the aggregate specs that enumerate skills/stages — `docs/specs/skills.md` (new section + helpers row + checklist note), `docs/specs/glossary.md` (Skills table + "adopt" term), `docs/specs/overview.md` (Quick Reference), `docs/specs/user-flow.md` (alternate entry-point). `_preamble.md` § Next Steps State Table SHALL be assessed for an adoption row/note.

- **GIVEN** the skill-file edits in R1–R8
- **WHEN** the sweep is performed
- **THEN** every SPEC mirror and every aggregate spec that enumerates skills/stages references `/fab-adopt` / "adopt", confirmed by grep

### Non-Goals

- Scenario A (retroactive backfill of an already-MERGED PR) — explicitly out of scope; `/fab-adopt` STOPs on MERGED.
- An `inward-only` review mode — no caller exists (parsimony); only `full` and `outward-only` are added.
- Any Go binary change to status transitions or Meta rendering — both reuse existing CLI surface.

### Design Decisions

1. **Adopt is the real pipeline entered late, not a parallel fake pipeline** — only apply is skipped; intake/review/hydrate/ship/review-pr genuinely run. *Why*: honest state (Constitution II — memory must reflect what shipped). *Rejected*: marking all stages "done" (dishonest, loses the hydrate value).
2. **Outward-only review via a general `mode` param on `_review.md`** — not an adopt-specific branch. *Why*: single review authority, inherited by all callers. *Rejected*: an adopt-only review fork (duplicates dispatch logic).
3. **State via `skip apply` + `reset review` composition** — *Why*: yields `apply=skipped, review=active` with existing transitions, no Go change. *Rejected*: a new Go transition (unnecessary).
4. **fab-help grouping via the Go `skillToGroupMap`** — a one-line map entry under "Planning". *Why*: the New Skill Checklist item 8 requires correct grouping, and the auto-scan would otherwise bucket it under "Other" — a visible drift review would flag. *Rejected*: relying on the "Other" bucket (the intake's "no Go change" was about state/Meta mechanics, not the cosmetic help map). This is a graded deviation from the intake's Impact note — see Assumptions.

## Tasks

### Phase 1: Core Skill-Layer Mechanics

- [x] T001 Add the `mode` parameter (Review Mode concept — `full` default | `outward-only`) to `src/kit/skills/_review.md`: gate inward Preconditions on `full`, dispatch only selected sub-agent(s), keep Findings Merge unchanged, document zero-findings-passes for outward-only <!-- R6 -->
- [x] T002 Add the **Intake-from-Diff Procedure** and **Plan-from-Diff Procedure** to `src/kit/skills/_generation.md` (Contents list + two new procedure sections) <!-- R7 -->
- [x] T003 Add the body-retrofit path to `src/kit/skills/git-pr.md` existing-OPEN-PR branch (render via `fab pr-meta`, apply with `gh pr edit --body-file -`, gated on body-lacks-`## Meta`); update Flow + Key Properties idempotency note <!-- R8 -->

### Phase 2: Orchestrator Skill

- [x] T004 Create `src/kit/skills/fab-adopt.md` — the orchestrator (frontmatter with `helpers: [_srad, _generation, _review, _pipeline]`, preamble-read line, Steps 0–6 per intake design, Output, Error Handling + Key Properties tables, `Next:` line) <!-- R1 R2 R3 R4 R5 -->

### Phase 3: Discoverability (Go help grouping)

- [x] T005 Add `"fab-adopt": "Planning"` to `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go` so `/fab-help` groups it correctly <!-- R9 -->

### Phase 4: SPEC Mirrors & Aggregate Sweeps

- [x] T006 [P] Create `docs/specs/skills/SPEC-fab-adopt.md` (Summary + Flow + Tools/Sub-agents tables, mirroring the new skill) <!-- R10 -->
- [x] T007 [P] Update `docs/specs/skills/SPEC-_review.md` (mode parameter, mode-gated preconditions, mode-selected dispatch) <!-- R10 -->
- [x] T008 [P] Update `docs/specs/skills/SPEC-_generation.md` (two new diff-based procedures) <!-- R10 -->
- [x] T009 [P] Update `docs/specs/skills/SPEC-git-pr.md` (body-retrofit path) <!-- R10 -->
- [x] T010 [P] Update `docs/specs/skills.md` (new `/fab-adopt` section, helpers row, New Skill Checklist coverage) <!-- R10 R9 -->
- [x] T011 [P] Update `docs/specs/glossary.md` (Skills table row + "adopt" terminology) <!-- R10 -->
- [x] T012 [P] Update `docs/specs/overview.md` (Quick Reference row) <!-- R10 -->
- [x] T013 [P] Update `docs/specs/user-flow.md` (adoption as alternate pipeline entry-point) <!-- R10 -->
- [x] T014 Assess `src/kit/skills/_preamble.md` § Next Steps State Table for an adoption row/note and update if warranted; sweep `SPEC-_preamble.md` if changed <!-- R10 -->

### Phase 5: Verify

- [x] T015 Run `go test ./...` in `src/go/fab/` for the `fabhelp` package after the T005 map edit; grep the enumeration set to confirm full sweep coverage <!-- R9 R10 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/fab-adopt.md` exists as a thin orchestrator with the correct frontmatter (`helpers: [_srad, _generation, _review, _pipeline]`) and Steps 0–6 implementing the intake design
- [x] A-002 R2: `/fab-adopt` Step 0 STOPs (no mutation) on detached HEAD, default branch, MERGED PR, branch-already-mapped-to-a-change, and empty diff
- [x] A-003 R3: intake + thin plan are generated in one main-session pass with a human-confirmation checkpoint; the change is created via `fab change new --slug` and activated
- [x] A-004 R4: the state transition uses `skip apply` + `reset review {driver}` only (no new Go transition) and yields `apply=skipped, review=active, downstream pending`
- [x] A-005 R5: review dispatches outward-only, hydrate runs verbatim, ship retrofits Meta, the change lands in review-pr; an honest-state summary prints
- [x] A-006 R6: `_review.md` documents `mode` (`full` default | `outward-only`), no `inward-only`; preconditions gated on `full`; dispatch selects sub-agent(s) by mode; outward-only zero findings → pass
- [x] A-007 R7: `_generation.md` documents the Intake-from-Diff and Plan-from-Diff procedures; the thin plan has plain-language requirements, all-`[x]` task/acceptance stubs, no R#/T#/A# scaffolding, and the three heading literals
- [x] A-008 R8: `git-pr.md` documents the body-retrofit path (render via `fab pr-meta`, apply with `gh pr edit --body-file -`, gated on body-lacks-`## Meta`, idempotent)
- [x] A-009 R9: `/fab-adopt` appears in `/fab-help` under the Planning group (frontmatter + `skillToGroupMap` entry)
- [x] A-010 R10: every SPEC mirror (`SPEC-fab-adopt.md` new, `SPEC-_review.md`, `SPEC-_generation.md`, `SPEC-git-pr.md`) and every aggregate spec (`skills.md`, `glossary.md`, `overview.md`, `user-flow.md`) references `/fab-adopt` / "adopt"

### Behavioral Correctness

- [x] A-011 R6: with `mode` omitted, all existing review callers (`/fab-continue`, `/fab-ff`, `/fab-fff`) behave identically to before (default `full`)

### Edge Cases & Error Handling

- [x] A-012 R8: a second `/git-pr` run against a PR whose body already has `## Meta` makes no edit (idempotent)
- [x] A-013 R6: outward-only with all `review_tools` disabled/unavailable returns zero findings and **passes** (best-effort, no hard block)

### Code Quality

- [x] A-014 Pattern consistency: new skill + SPEC follow the structure of sibling files (frontmatter, preamble-read line, Contents/Behavior, Key Properties; SPEC Summary + Flow)
- [x] A-015 No unnecessary duplication: `/fab-adopt` reuses `_pipeline.md` / `_generation.md` / `_review.md` / `/git-pr` as sub-agents rather than inlining their logic; no `inward-only` mode added speculatively
- [x] A-016 Canonical source only: all skill edits are under `src/kit/skills/` — no `.claude/skills/` edits (Constitution V; code-quality.md Anti-Patterns)
- [x] A-017 SPEC-mirror sync (documentation_accuracy): every edited `src/kit/skills/*.md` carries its `docs/specs/skills/SPEC-*.md` update in this change
- [x] A-018 Cross-references (cross_references): the enumeration sweep is complete — grep for `fab-clarify|fab-proceed` over `docs/specs/ src/kit/skills/` confirms `/fab-adopt` is added wherever siblings are listed (or a documented reason it is not)

## Notes

- Check items as you review: `- [x]`
- The Go change (T005) is a single-line `skillToGroupMap` entry — a deliberate, graded deviation from the intake's "no Go change" Impact note (see Assumptions). It requires no test or command-signature change; `fabhelp_test.go` asserts a subset (`expectedMapped`), not a fixed total.

## Deletion Candidates

None — this change adds new functionality (a new `/fab-adopt` orchestrator skill, an additive `mode: outward-only` review path, and an additive `/git-pr` Step 3d body-retrofit branch) without making existing code redundant. The `mode` parameter defaults to `full`, so no existing review path is superseded; Step 3d is a new branch on the existing-OPEN-PR path, not a replacement; and the two `_generation.md` -from-Diff procedures sit alongside (do not replace) the forward Intake/Plan Generation Procedures.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Add a one-line `"fab-adopt": "Planning"` entry to the Go `skillToGroupMap` despite the intake's "no Go change" Impact note | The New Skill Checklist (skills.md item 8) requires correct help grouping; without it the skill silently falls into the "Other" bucket — a visible drift review flags. The intake's "no Go change" was scoped to the state-composition and Meta-retrofit mechanics (Assumptions #8/#9), not the cosmetic help map. No test or command-signature change is needed (`fabhelp_test.go` checks a subset, not a total). | S:80 R:85 A:85 D:75 |
| 2 | Confident | Place `/fab-adopt` in the "Planning" help group (alongside fab-continue/ff/fff/proceed/clarify) | Adopt is a pipeline orchestrator like the other Planning entries; it is not a Start/Navigate or Completion skill | S:85 R:90 A:85 D:80 |
| 3 | Confident | Add an adoption row to `_preamble.md` § Next Steps State Table only if it cleanly fits; otherwise add a derivation note | The State Table keys on pipeline *state*, not entry skill; adoption reaches the same states (review/hydrate/ship/review-pr) so the existing rows already cover the `Next:` lookups — a note clarifying adoption as an alternate entry is the minimal correct change | S:75 R:80 A:80 D:70 |
| 4 | Confident | `/fab-adopt` `helpers:` = `[_srad, _generation, _review, _pipeline]` per the intake; `_srad` is included because the diff-reconstructed intake applies SRAD scoring | Matches the intake's explicit declaration and the planning-skill helper pattern (fab-ff/fff declare the same set) | S:85 R:85 A:85 D:80 |
| 5 | Tentative | Resolve the two Open Questions toward the intake's leanings: handle `pr_state == none` by letting Step 5's `/git-pr` create the PR fresh (no separate `--no-pr` flag); auto-rework applies when run autonomously but the interactive default hands findings back | The intake marks both as "resolve at apply" and states the leaning; a separate `--no-pr` flag is unneeded (the create path already handles it), and hand-back-by-default respects a hand-authored branch | S:70 R:70 A:70 D:65 |

5 assumptions (0 certain, 4 confident, 1 tentative).
