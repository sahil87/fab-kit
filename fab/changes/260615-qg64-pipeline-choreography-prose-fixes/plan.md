# Plan: Pipeline Choreography Prose Fixes (Group B)

**Change**: 260615-qg64-pipeline-choreography-prose-fixes
**Intake**: `intake.md`

## Requirements

<!-- Two prose-only defects. Requirements pin against the Go contract (read-only oracle):
     status.go:627 (Iterations++ only on state=="active"), status.go:646-660 (reset/skip
     cascade PRESERVES Iterations, clears timing only), and the Go regression tests
     (mutators_test.go TestStageMetrics_IterationsAccumulateAcrossReworkCycles,
     status_test.go TestResetCascade_PreservesReviewIterations) which together fix the
     baseline convention. No Go change; src/go/** is off-limits. -->

### Pipeline: Per-Cycle Iteration-Count Choreography (defect a)

#### R1: The Auto-Rework Loop choreography MUST drive exactly one counted review re-activation per cycle

The per-cycle choreography in `src/kit/skills/_pipeline.md` (Auto-Rework Loop) MUST be
explicit that the **only** transition `stage_metrics.review.iterations` counts is a review
`→ active` transition (Go contract: `status.go:627` `Iterations++` fires only on
`state == "active"`), and that the `reset apply` cascade **preserves** the counter without
incrementing it (`status.go:646–660`). Each rework cycle MUST therefore re-enter review via the
`finish apply` auto-activation (item 3) — the counter-incrementing path — and MUST NOT rely on
`reset` to bump or zero the counter, nor re-enter review by any non-`active` path.

- **GIVEN** an Auto-Rework Loop running N rework cycles after an initial review failure
- **WHEN** each cycle runs the `fail review` + `reset apply` pair (item 1), then `finish apply` (item 3, which auto-activates review)
- **THEN** review transitions to `active` exactly once per cycle, incrementing `iterations` by exactly 1 per cycle
- **AND** the `reset apply` cascade never increments and never zeroes `iterations` (it clears timing fields only)

#### R2: The choreography prose MUST document the baseline counting convention so `iterations` equals the rendered cycle count

`_pipeline.md` MUST state the baseline convention determined from the Go regression-test oracle:
`iterations` counts the **initial review entry plus each rework re-entry** — i.e. `iterations`
== the total number of review `→ active` transitions. With an initial review attempt plus N rework
cycles, `iterations == N + 1` (1 initial + N re-entries), and `fab pr-meta` renders this verbatim
as "{iterations} cycle(s)" (`prmeta.go:142–163` `reviewCell` → `{N} cycle{s}`). The prose MUST make
this convention explicit so prose, the Go contract, and `pr-meta` agree and a reader can verify the
count.

- **GIVEN** the oracle `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` (initial activation → iterations=1; then 2 fail→reset→re-finish cycles → iterations=2, then 3)
- **WHEN** the choreography is followed for an initial review + N rework cycles
- **THEN** `stage_metrics.review.iterations == N + 1`
- **AND** `fab pr-meta` renders "✓ {N+1} cycles" (e.g. an initial fail + 2 rework cycles → iterations 3 → "✓ 3 cycles"), never collapsing to "1 cycle"

#### R3: `fab-ff.md` / `fab-fff.md` MUST NOT carry divergent per-cycle choreography wording

The per-cycle choreography is stated exactly once, in `_pipeline.md`. `fab-ff.md` and `fab-fff.md`
are thin wrappers that reference the bracket and MUST NOT restate the per-cycle sequence. They are
edited for defect (a) **only if** they restate the choreography (verification confirms they do not).

- **GIVEN** `fab-ff.md` and `fab-fff.md` reference `_pipeline.md`'s auto-rework loop without restating it
- **WHEN** defect (a) is fixed in `_pipeline.md`
- **THEN** no per-cycle choreography edit is made to `fab-ff.md` or `fab-fff.md` (the single-source property is preserved)

### Review-PR: Copilot Poll Discipline (defect b)

#### R4: The Copilot poll MUST run synchronously to completion — the subagent MUST NOT yield mid-poll

`src/kit/skills/git-pr-review.md` Step 2 Phase 2 (and Step 3 where relevant) MUST carry a permanent,
explicit directive that the Copilot poll runs **synchronously to completion**: the subagent MUST NOT
yield, return, or hand back control while the poll is pending (the 30s × 20 / 10-minute window).
Rationale encoded inline: the subagent stalled/died mid-poll 4× in prior efforts; Copilot lands
~4.5–6.5 min, comfortably inside the window, so the correct behavior is patience-to-completion.

- **GIVEN** `/git-pr-review` has requested a Copilot review and entered the poll
- **WHEN** the review has not yet appeared and attempts remain
- **THEN** the subagent continues polling synchronously without yielding/returning until the review appears or all 20 attempts are exhausted

#### R5: `fab-fff.md` Step 5 dispatch prompt MUST instruct the subagent to complete the poll synchronously

`src/kit/skills/fab-fff.md` Step 5 (the review-pr dispatch prompt) MUST instruct the dispatched
`/git-pr-review` subagent to complete the Copilot poll synchronously and not yield mid-poll, mirroring
R4 into the dispatch seam. The poll STAYS inside `/git-pr-review` (subagent owns request + poll +
triage synchronously) — the wait is NOT moved to the orchestrator (intake Assumption #7).

- **GIVEN** `/fab-fff` Step 5 dispatches `/git-pr-review` as a subagent
- **WHEN** the dispatch prompt is constructed
- **THEN** it includes a directive that the subagent must complete the Copilot poll synchronously and not yield mid-poll
- **AND** the poll remains owned by `/git-pr-review`, not relocated to the orchestrator

#### R6: The poll MUST use correct GitHub query semantics (request login vs. comment-author login; REST for request confirmation)

`git-pr-review.md` Step 2 Phase 2 MUST encode the correct GitHub semantics: the value passed to
`gh pr edit --add-reviewer …` is `copilot-pull-request-reviewer`, and that same string is the
**landed-review author login** — once a Copilot review lands, the review object in the `reviews`
array carries `author.login == "copilot-pull-request-reviewer"` (commonly surfaced as
`copilot-pull-request-reviewer[bot]`). The entry that surfaces under the PR's `requested_reviewers`
(the request side) carries the *different* login `Copilot`. The poll predicate that detects a
*landed review* MUST match the review object's author login `copilot-pull-request-reviewer` (NOT the
`Copilot` login that surfaces under `requested_reviewers`). The skill MUST NOT conflate the two
logins and MUST document the distinction inline. Because GraphQL `reviewRequests` omits bot/app
reviewers, confirming the **request** succeeded MUST use REST `requested_reviewers`
(`gh api repos/{owner}/{repo}/pulls/{number}/requested_reviewers`), not a GraphQL `reviewRequests`
field. Poll cadence is unchanged: 30s × 20 (10-minute window).

- **GIVEN** a Copilot review has been requested and the poll is running
- **WHEN** the poll checks whether the review has landed
- **THEN** it selects review objects whose `author.login == "copilot-pull-request-reviewer"` (the landed-review author login on the `reviews` array), not `Copilot` (the login under `requested_reviewers`)
- **AND** request confirmation uses REST `requested_reviewers` (GraphQL omits bot reviewers)
- **AND** the poll cadence remains 30s × 20

### SPEC Mirrors: Constitution Mirror Rule (mandatory)

#### R7: Every touched skill `.md` MUST update its `docs/specs/skills/SPEC-*.md` mirror in the same change

Constitution: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding
`docs/specs/skills/SPEC-*.md` file." Copilot enforces this strictly. Each skill `.md` whose content
changes MUST have its SPEC mirror updated to reflect the change in the same PR.

- **GIVEN** `_pipeline.md`, `git-pr-review.md`, and `fab-fff.md` change content
- **WHEN** the change is committed
- **THEN** `SPEC-_pipeline.md`, `SPEC-git-pr-review.md`, and `SPEC-fab-fff.md` are each updated to mirror the change
- **AND** `SPEC-fab-ff.md` is left unchanged iff `fab-ff.md` is unchanged

### Non-Goals

- **No Go changes** — `src/go/**` is off-limits; `internal/status` and `internal/prmeta` are confirmed correct.
- **No behavior-schema change** — `.status.yaml` schema, state-machine transitions, and `fab` CLI signatures are unchanged.
- **No timing change** to the poll — cadence stays 30s × 20 (10-minute window).
- **No relocation of the poll** to the orchestrator — it stays inside `/git-pr-review` (Assumption #7).
- **No edits to `.claude/skills/`** — `src/kit/` is canonical; deployed copies are gitignored.

### Design Decisions

1. **Defect (a) root cause is an observability/invariant gap in prose, not a wrong command sequence**: Tracing the *current* `_pipeline.md` choreography against `applyMetricsSideEffect` shows it already drives `finish apply` (review→active) once per cycle, which would count correctly. The under-count arises because the prose never states the **invariant** binding the choreography to the counter (one review `→ active` per cycle is the only counted event; `reset` preserves but never increments), nor documents the baseline convention — leaving room for an implementing agent to re-enter review by a non-counting path or to misread the count. *Why*: the intake confirms the Go layer is correct and the fix is "make the prose drive the correct sequence"; the strongest prose fix is to make the counting invariant explicit and pin it to the contract. *Rejected*: rewriting the command sequence (would risk changing `.status.yaml` history shape, which the Go tests assert) — and any Go edit (off-limits, and the oracle already passes).

2. **Baseline convention = initial entry + each re-entry (N rework cycles → iterations N+1)**: Determined from the oracle. `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` activates review once via `finish apply` (iterations=1), then runs 2 fail→reset→re-finish cycles landing on iterations=3; `reviewCell` renders `iterations` verbatim as "{N} cycle(s)". So `iterations` == count of review `→ active` transitions == 1 (initial) + (rework re-entries). *Why*: the Go test is the deterministic oracle the intake names. *Rejected*: "only rework re-entries count" (N cycles → N) — contradicted by the oracle, which counts the initial activation.

3. **Poll predicate matches `author.login == "copilot-pull-request-reviewer"` for landed-review detection** *(corrected during rework — the apply pass had this backwards)*: The empirical record (`docs/memory/pipeline/execution-skills.md`) establishes the mapping: a *landed* Copilot review object in the `reviews` array carries `author.login == "copilot-pull-request-reviewer[bot]"` (n30u, 4ojc), while the entry under `requested_reviewers` surfaces with login `Copilot` (n30u). u1m1 (the most recent fix, 2026-04-18) deliberately set the Phase 2 `.author.login` filter to `copilot-pull-request-reviewer` "so that incoming Copilot reviews are detected." The poll predicate therefore MUST match `copilot-pull-request-reviewer` (the review-author login), NOT `Copilot`. *Why*: a poll keyed on the requested-reviewer login (`Copilot`) never sees a review that has in fact landed — the exact spurious-timeout bug this change set out to fix. *Rejected*: keying the predicate on `Copilot` (that is the requested-reviewer login, not the review-author login — the apply pass made this inversion and it was the must-fix this rework corrected). *Apparent oddity, recorded as empirical reality*: the value passed to `gh pr edit --add-reviewer` (`copilot-pull-request-reviewer`) happens to equal the landed-review author login, while `requested_reviewers` shows `Copilot`.

## Tasks

### Phase 1: Defect (a) — per-cycle iteration choreography

- [x] T001 Edit `src/kit/skills/_pipeline.md` Auto-Rework Loop "Per-cycle choreography": add an explicit counting invariant — exactly one review `→ active` transition per cycle (via `finish apply` item 3) is the only event `stage_metrics.review.iterations` counts (`status.go:627`); `reset apply` (item 1) preserves the counter without incrementing (`status.go:646–660`); never rely on `reset` to bump/zero it. State the baseline convention: iterations == initial entry + each re-entry, so N rework cycles → iterations N+1, rendered by `fab pr-meta` as "{iterations} cycle(s)". <!-- R1 --> <!-- R2 -->
- [x] T002 Verify `src/kit/skills/fab-ff.md` and `src/kit/skills/fab-fff.md` carry NO divergent per-cycle choreography wording; edit only if they restate it (expected: no change). Confirmed: both only reference the bracket's per-cycle choreography as a pointer; neither restates it — no choreography edit made to either. <!-- R3 -->

### Phase 2: Defect (b) — review-pr poll discipline

- [x] T003 Edit `src/kit/skills/git-pr-review.md` Step 2 Phase 2: add a permanent don't-yield-mid-poll directive (poll runs synchronously to completion; subagent MUST NOT yield/return/hand back while pending; rationale: stalled 4×, Copilot lands ~4.5–6.5 min inside the 10-min window). <!-- R4 -->
- [x] T004 Edit `src/kit/skills/git-pr-review.md` Step 2 Phase 2 (and Step 3 if needed): set the landed-review poll predicate to `author.login == "copilot-pull-request-reviewer"` (the review-author login on the `reviews` array) — distinct from the `Copilot` login that surfaces under `requested_reviewers` — documenting the two-login distinction inline; add that request confirmation uses REST `requested_reviewers` (GraphQL `reviewRequests` omits bot reviewers); keep cadence 30s × 20. <!-- R6 --> <!-- rework: inverted Copilot login predicate corrected -->
  <!-- rework: the apply pass had inverted the mapping (predicate set to "Copilot"), which would never match a landed review and reintroduce the spurious-timeout bug. Reverted to the correct, deliberately-set u1m1 value `copilot-pull-request-reviewer`; corrected the two-login prose direction. Evidence: docs/memory/pipeline/execution-skills.md n30u + u1m1 + 4ojc. -->
- [x] T005 Edit `src/kit/skills/fab-fff.md` Step 5 dispatch prompt: instruct the dispatched `/git-pr-review` subagent to complete the Copilot poll synchronously and not yield mid-poll; affirm the poll stays inside `/git-pr-review` (not relocated to the orchestrator — Assumption #7). <!-- R5 -->

### Phase 3: SPEC mirrors (mandatory)

- [x] T006 [P] Update `docs/specs/skills/SPEC-_pipeline.md` Per-Cycle Rework Choreography section to mirror the counting invariant + baseline convention added in T001. <!-- R7 -->
- [x] T007 [P] Update `docs/specs/skills/SPEC-git-pr-review.md` Phase 2 section to mirror the don't-yield directive + corrected query semantics (Copilot author login, REST requested_reviewers) from T003/T004. <!-- R7 -->
- [x] T008 [P] Update `docs/specs/skills/SPEC-fab-fff.md` Step 5 section to mirror the synchronous-poll dispatch directive from T005. <!-- R7 -->

## Execution Order

- T001 informs T006; T003/T004 inform T007; T005 informs T008 (do skill edits before their SPEC mirrors).
- T002 is verification-only; if it surfaces divergent wording, an extra edit + SPEC mirror would be added.
- T006, T007, T008 are `[P]` (different files) once their source skill edits land.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `_pipeline.md`'s Auto-Rework Loop explicitly states that exactly one review `→ active` transition per cycle (via `finish apply`) is the only event `iterations` counts, and that `reset` preserves-without-incrementing. (Verified: `_pipeline.md:84` Cycle-count invariant + item 3 at `:92`.)
- [x] A-002 R2: `_pipeline.md` documents the baseline convention (iterations == initial entry + each re-entry; N rework cycles → N+1) and ties it to `fab pr-meta`'s "{N} cycle(s)" rendering. (Verified: `_pipeline.md:86` Baseline convention.)
- [x] A-003 R4: `git-pr-review.md` Step 2 Phase 2 carries a permanent don't-yield-mid-poll directive (synchronous to completion). (Verified: `git-pr-review.md:108` + inline note at `:116`.)
- [x] A-004 R5: `fab-fff.md` Step 5 dispatch prompt instructs the subagent to complete the poll synchronously and not yield mid-poll, with the poll kept inside `/git-pr-review`. (Verified: `fab-fff.md:59`.)
- [x] A-005 R6: `git-pr-review.md` landed-review poll predicate matches `author.login == "copilot-pull-request-reviewer"` (the review-author login, NOT the `Copilot` login under `requested_reviewers`), documents the two-login distinction the right way round, and notes REST `requested_reviewers` for request confirmation. (Verified: predicate `git-pr-review.md:118`; two-login note `:102–106`; REST note `:106`,`:115`. <!-- rework: predicate/prose corrected from the inverted apply-pass value -->)

### Behavioral Correctness

- [x] A-006 R1: The corrected prose, when followed, drives `iterations` to advance exactly once per rework cycle — consistent with the unchanged Go oracle (`TestStageMetrics_IterationsAccumulateAcrossReworkCycles`). (Verified: oracle PASSES uncached; status.go:627 increments only on `active`; prose binds item-3 `finish apply` to that one transition.)
- [x] A-007 R6: The poll cadence is unchanged (30s × 20) and `copilot-pull-request-reviewer` is the value passed to `gh pr edit --add-reviewer` (retained). The detection predicate also keys on `copilot-pull-request-reviewer` (the landed review-author login), per the corrected mapping. (Verified: cadence preserved in note + loop; `gh pr edit --add-reviewer copilot-pull-request-reviewer` retained at `git-pr-review.md:112`; predicate at `:118`.)

### Scenario Coverage

- [x] A-008 R2: A worked example/statement makes clear that an initial fail + 2 rework cycles → iterations 3 → "✓ 3 cycles", not "1 cycle". (Verified: worked example at `_pipeline.md:86` and mirrored at `SPEC-_pipeline.md` cycle-count invariant para.)

### Edge Cases & Error Handling

- [x] A-009 R3: `fab-ff.md`/`fab-fff.md` carry no divergent per-cycle choreography wording after the change (single-source property preserved). (Verified: both only point to the bracket — `fab-ff.md:35`, `fab-fff.md:35`; no per-cycle sequence restated; the fab-fff.md edit is solely in the Step 5 poll-dispatch prompt, not choreography.)

### Removal Verification

- [x] A-010 R6: No poll/detection predicate keys on `author.login == "Copilot"` for landed-review detection in `git-pr-review.md` (the `Copilot` login is the requested-reviewer login, not the review-author login). (Verified: the sole detection predicate at `:118` keys on `author.login == "copilot-pull-request-reviewer"`; every `Copilot`-login mention is a requested-reviewers context — the two-login note, the `--add-reviewer` parenthetical, and the REST confirmation note. <!-- rework: removal-verification inverted to match the corrected mapping; the apply-pass `Copilot` predicate was the must-fix -->)

### SPEC Mirror Compliance

- [x] A-011 R7: Every touched skill `.md` (`_pipeline.md`, `git-pr-review.md`, `fab-fff.md`) has its `SPEC-*.md` mirror updated in the same change; `SPEC-fab-ff.md` unchanged iff `fab-ff.md` unchanged. (Verified: all 3 SPEC mirrors modified in working tree; `fab-ff.md` and `SPEC-fab-ff.md` both untouched.)

### Constitution Compliance

- [x] A-012 R7: No `src/go/**` files changed; no `.claude/skills/` files changed; no `.status.yaml`/state-machine/CLI-signature change. (Verified: `git status` shows zero `src/go/` and zero `.claude/` entries; only the 6 content files + this change's own artifacts/.status.yaml are touched.)

### Code Quality

- [x] A-013 Pattern consistency: Edits follow the existing prose/markdown style and cross-reference conventions of each skill and SPEC file. (Verified: blockquote callouts, `(260615-qg64)` change-id tags, and `status.go:NNN`/`prmeta.go` pin-citation style all match surrounding prose conventions.)
- [x] A-014 No unnecessary duplication: The per-cycle choreography stays single-sourced in `_pipeline.md`; the poll discipline is stated where it belongs without restating across files beyond the required dispatch-prompt mirror. (Verified: choreography only in `_pipeline.md`; poll discipline in `git-pr-review.md` + the one required `fab-fff.md` dispatch-prompt mirror.)

### Documentation Accuracy

- [x] A-015 R2: Prose statements about the Go contract (`status.go:627`, `646–660`; `prmeta.go` rendering) accurately reflect the read-only oracle and do not imply any Go change. (Verified against source: `Iterations++` is at status.go:627 inside `case "active":`; the iterations-preserving `pending`/`skipped` branch spans 647–657 (citation 646–660 brackets it correctly); `reviewCell`/`pluralCycle` render `{N} cycle{s}` verbatim. Prose explicitly says "do NOT change `internal/status`".)

### Cross-References

- [x] A-016 R7: SPEC mirrors reference the corresponding skill behavior accurately; any change IDs / section pointers used are consistent with existing conventions. (Verified: each SPEC mirror tags `(260615-qg64)` like sibling entries; SPEC-_pipeline cycle-count invariant, SPEC-git-pr-review poll-discipline bullets, and SPEC-fab-fff Step-5 directive each accurately mirror their skill edits.)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change is purely additive clarifying prose (cycle-count invariant + baseline convention in `_pipeline.md`; synchronous-poll discipline + corrected query semantics in `git-pr-review.md`; a dispatch-prompt mirror in `fab-fff.md`) plus the mandatory SPEC mirrors. It pins existing Go behavior into prose without changing any command sequence or `.status.yaml` history shape, so it makes no existing code, file, branch, or config redundant. (The `status.go` `stageTransitions` dead-override-row deletion candidate noted in prior efforts is a Go-layer item, out of scope here.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Defect (a)'s root cause is an observability/invariant gap in the prose (the choreography already drives `finish apply` once per cycle), so the fix is to make the counting invariant + baseline convention explicit and pin them to the Go contract — not to change the command sequence. | Tracing the current `_pipeline.md` choreography against `applyMetricsSideEffect` shows the counting path is already present; the oracle passes for the documented sequence. The intake frames the fix as "make the prose drive the correct sequence." Changing the command sequence would risk the `.status.yaml` history shape the Go tests assert. One obvious interpretation; reversible (prose). | S:80 R:80 A:80 D:75 |
| 2 | Certain | Baseline convention: `iterations` == initial review entry + each rework re-entry; N rework cycles → iterations N+1; `fab pr-meta` renders it verbatim as "{iterations} cycle(s)". | Determined from the Go regression oracle (`TestStageMetrics_IterationsAccumulateAcrossReworkCycles`: initial activation → 1, +2 cycles → 3) and `prmeta.go` `reviewCell`. Deterministic — the test fixes the convention; no judgment. | S:95 R:80 A:95 D:90 |
| 3 | Certain | The landed-review detection predicate keys on `author.login == "copilot-pull-request-reviewer"` (the review-author login on the `reviews` array), distinct from the `Copilot` login that surfaces under `requested_reviewers`. | Verified against the repo's empirical record: `docs/memory/pipeline/execution-skills.md` n30u (`"Copilot"` in `requested_reviewers` vs `"copilot-pull-request-reviewer[bot]"` in `reviews`), 4ojc (poll waits for `copilot-pull-request-reviewer[bot]`), and u1m1 (deliberately set the Phase 2 `.author.login` filter to `copilot-pull-request-reviewer` so incoming reviews are detected). The original skill on `main` polled this value; the apply pass inverted it to `Copilot` — a must-fix that reintroduced the spurious-timeout bug, corrected in this rework. Deterministic from the memory oracle; no judgment. | S:95 R:80 A:100 D:95 |
| 4 | Certain | Every touched skill `.md` (`_pipeline.md`, `git-pr-review.md`, `fab-fff.md`) updates its `SPEC-*.md` mirror in the same change; `SPEC-fab-ff.md` unchanged iff `fab-ff.md` unchanged. | Constitution rule ("MUST update the corresponding SPEC-*.md"); Copilot enforces strictly. Deterministic, no judgment. | S:100 R:70 A:100 D:95 |
| 5 | Certain | All edits land in `src/kit/skills/` and `docs/specs/skills/` only; no `src/go/**`, no `.claude/skills/`, no schema/CLI change. | Constitution Principle V + explicit intake scope guardrails; `src/go/**` confirmed correct and off-limits. Deterministic. | S:100 R:85 A:100 D:100 |

5 assumptions (4 certain, 1 confident, 0 tentative).
