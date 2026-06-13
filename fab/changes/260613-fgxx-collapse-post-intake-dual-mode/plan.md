# Plan: Collapse post-intake dual execution mode — one execution mode, orchestrators as sequencers

**Change**: 260613-fgxx-collapse-post-intake-dual-mode
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md § What Changes (prescriptive) and the two findings docs
     (intake-is-the-context-boundary.md, per-stage-model-tier-application.md).
     This is a behavior reorganization of skill+spec prose plus one Go test —
     no production Go change, no new CLI command. RFC-2119 statements with stable
     R# IDs and GIVEN/WHEN/THEN scenarios. -->

### Skills: `fab-continue.md` post-intake execution model

#### R1: Post-intake `/fab-continue` always dispatches a subagent for its stage
Post-intake `/fab-continue` (apply / review / hydrate) MUST always dispatch a subagent for its
stage — the same dispatch the orchestrators (`_pipeline.md`) already perform per `_preamble.md`
§ Subagent Dispatch. There MUST NOT be an in-session foreground execution path for
apply/review/hydrate. `/fab-continue` in its always-dispatch form is a **one-stage sequencer**: it
owns the `fab status` transitions and runs `fab resolve-agent <stage>` immediately before dispatch,
then reads the returned status/findings and decides proceed / present-menu / stop. Intake
(pre-boundary, main-context) and ship/review-pr (self-managed transitions) are OUT of this rewrite.

- **GIVEN** an active change at apply/review/hydrate and a user runs plain `/fab-continue`
- **WHEN** the skill reaches Step 1 dispatch for that post-intake stage
- **THEN** it runs `fab resolve-agent <stage>`, dispatches a subagent (Agent tool, general-purpose)
  with the resolved model/effort and the "do NOT run `fab status`; return results only" prompt
  contract, and the orchestrating `/fab-continue` itself runs the `finish`/`fail`/`reset` transition
- **AND** there is no documented code path where apply/review/hydrate runs in the foreground session

#### R2: The three dual-mode `do NOT run fab status` conditionals are removed
The THREE caller-aware blockquote conditionals in `fab-continue.md` — Apply Behavior (~line 100),
Review Behavior (~line 156), Hydrate Behavior (~line 176) — each of the form *"When invoked as a
subagent (dispatched by `/fab-ff`/`/fab-fff`): do NOT run any `fab status` command …"* MUST be
removed. With a single post-intake execution mode the conditional has no second branch to express:
the "do NOT run `fab status`; return results only" instruction the orchestrator passes in the
dispatch prompt becomes the **universal block contract**, not a per-caller override living inside the
block.

- **GIVEN** the current `fab-continue.md` with three subagent-only conditionals
- **WHEN** the rewrite is applied
- **THEN** none of the three blockquotes remain in `src/kit/skills/fab-continue.md`
- **AND** the block bodies describe the single dispatched execution mode

#### R3: The removed conditional MUST NOT be reintroduced in a different shape
The block MUST NOT be given a "skip §Verdict when subagent" flag or any other caller-branching
switch. The Review Behavior block ALWAYS returns prioritized findings (must-fix / should-fix /
nice-to-have) + pass/fail as its return value; whether a §Verdict-style decision runs, and who runs
it, is the orchestrator's concern (interactive menu in Path A; autonomous triage in Paths B/C/D). The
block does not branch on caller.

- **GIVEN** the rewritten Review Behavior
- **WHEN** dispatched by any caller (manual `/fab-continue` or `/fab-ff`/`/fab-fff`/`/fab-proceed`)
- **THEN** it returns the same findings payload regardless of caller, with no caller-aware branch
- **AND** no flag re-encodes the removed "skip §Verdict entirely" instruction

#### R4: The interactive § Verdict rework menu (Path A) is preserved
The interactive human rework menu in `fab-continue.md` § Verdict (the **Fail** options table: Fix
code / Revise plan / Revise requirements) MUST remain intact as the Path A failure policy. The human
now reads the **dispatched** review subagent's returned findings rather than findings produced
in-session, but the menu and the human choice stay. This is the invocation-level failure-policy fork
that is correctly orchestrator-side (mirrored by `_pipeline.md` § Auto-Rework Loop for Paths B/C/D).

- **GIVEN** a manual `/fab-continue` review that returns a fail verdict
- **WHEN** the orchestrating `/fab-continue` processes the returned findings
- **THEN** it runs `fab status fail review` then `fab status reset apply`, then presents the
  unchanged Verdict Fail options table for the user's choice
- **AND** the `review`/`failed` dispatch row and the Reset Flow continue to reference this menu

#### R5: The foreground/advisory header note is reconciled to single-mode
The `fab-continue.md` header note (currently "Per-stage model (foreground = advisory-only)", ~line
19) describes the dual mode being collapsed. With no foreground execution path for
apply/review/hydrate it MUST be reconciled so it no longer frames those stages as foreground/advisory;
it MAY shrink to "the one-stage sequencer resolves the tier and dispatches." It MUST NOT contradict
the single-mode model.

- **GIVEN** the header note framing per-stage selection as advisory in `/fab-continue`'s foreground
- **WHEN** the rewrite is applied
- **THEN** the note describes `/fab-continue` as a one-stage sequencer that resolves the tier and
  dispatches its stage's subagent, with no surviving "foreground = advisory" claim for post-intake
  stages

### Skills: `_pipeline.md` and orchestrator wrappers

#### R6: `_pipeline.md` affirms orchestrator-as-pure-sequencer; light prose alignment only
`_pipeline.md` already implements the orchestrator-as-sequencer shape (§ Behavior Dispatch note,
Steps 1–3, Auto-Rework Loop). The change SHOULD be light prose alignment: state that the "do NOT run
`fab status`; return results only" prompt instruction is now the **universal block contract** (not a
per-caller override), since `/fab-continue`'s own post-intake path now dispatches identically. No
structural change to the bracket.

- **GIVEN** `_pipeline.md`'s existing Behavior Dispatch note
- **WHEN** the alignment edit is applied
- **THEN** the prose affirms the universal-contract framing without altering the bracket's steps
- **AND** the orchestrator remains the owner of `fab status` transitions and `fab resolve-agent`

#### R7: The Auto-Rework Loop per-cycle choreography is preserved verbatim in semantics
The Auto-Rework Loop's per-cycle choreography — the `fab status fail review` then
`fab status reset apply` pair that repeats on every failed verdict, and the line-86 invariant *"every
conforming run leaves the same `.status.yaml` history shape"* — MUST be preserved verbatim in
semantics. A uniformly-dispatched apply MUST NOT break it: the orchestrator still issues the same
`fail → reset → finish` sequence on every rework cycle.

- **GIVEN** the Auto-Rework Loop's fail+reset pair and history-shape invariant
- **WHEN** this change's edits are applied
- **THEN** the fail+reset choreography and the line-86 invariant statement are unchanged in meaning
- **AND** no edit alters which `fab status` commands fire per cycle or their order

#### R8: `fab-ff.md` / `fab-fff.md` / `fab-proceed.md` are surveyed; changed only if their prose presumes block-owned transitions
The thin wrappers are already subagent-only and delegate transition ownership to `_pipeline.md`.
They MUST be surveyed; an edit is made ONLY if prose presumes the *block* owns transitions in some
path. Expectation: little/no change. fab-proceed's prefix steps are not pipeline stages and are out
of scope.

- **GIVEN** the three wrapper skills
- **WHEN** surveyed against the single-mode model
- **THEN** any prose presuming block-owned transitions is corrected; otherwise they are left unchanged
  and the survey result is recorded

### Skills: `_preamble.md` advisory-only paragraph (minimal reconcile)

#### R9: `_preamble.md` "Foreground is advisory-only" paragraph is minimally reconciled
The § Subagent Dispatch → Per-Stage Model Resolution "**Foreground is advisory-only**" paragraph
describes the dual mode being collapsed. It MUST be minimally reconciled to stop contradicting the
single-mode model for post-intake stages — it MUST NOT be fully rewritten into Change C's
uniform-tiering story (full Gap-1a closure / effort-knob work is deferred to Change C). The paragraph
is referenced by multiple skills and by `stage-models.md`, so the edit is surgical.

- **GIVEN** the advisory-only paragraph that carves out manual `/fab-continue` apply/hydrate as
  foreground-untiered
- **WHEN** the minimal reconcile is applied
- **THEN** the paragraph no longer asserts that post-intake `/fab-continue` stages run foreground/
  untiered, but it does NOT add the full uniform-tier narrative reserved for Change C
- **AND** because `_preamble.md` is a `src/kit/skills/*.md` file, its mandated spec mirror
  `docs/specs/skills/SPEC-_preamble.md` (constitution: skill changes MUST update the corresponding
  `SPEC-*.md`) is reconciled in lockstep — its Summary (the "foreground advisory-only" phrase) and
  the § Subagent Dispatch ASCII-tree leaf no longer contradict single-mode (rework: review must-fix —
  the spec mirror of the edited `_preamble.md` was a plan-scoping gap)
- **AND** any remaining advisory note applies only where a genuine foreground path still exists (a
  bare stage skill run with no orchestrator at all is not a documented post-intake path anymore)

### Specs: SPEC mirrors and stage-models.md

#### R10: `SPEC-fab-continue.md` mirrors the single-mode rewrite
Per the constitution (skill changes MUST update the corresponding `SPEC-*.md`), `SPEC-fab-continue.md`
MUST remove/replace the three conditional descriptions, update the foreground=advisory summary
(line 9) and the Summary to single-mode post-intake dispatch, and KEEP the Verdict rework-menu spec
as the Path A behavior. The "Subagent rule (f006)" note describing the skip-finish/§Verdict behavior
for the three blocks MUST be reconciled to the universal-contract framing.

- **GIVEN** `SPEC-fab-continue.md` describing the dual mode
- **WHEN** the spec edit is applied
- **THEN** the spec describes always-dispatch post-intake `/fab-continue`, the universal
  no-`fab status` contract, and retains the Verdict rework-menu (Path A) spec
- **AND** no spec prose still asserts a foreground execution path for apply/review/hydrate

#### R11: `SPEC-_pipeline.md` aligns to pure-sequencer framing
`SPEC-_pipeline.md` MUST align its dispatch/transition-ownership prose with the pure-sequencer
framing (the universal block contract), without altering the documented per-cycle choreography or the
history-shape invariant.

- **GIVEN** `SPEC-_pipeline.md`'s dispatch note
- **WHEN** the alignment edit is applied
- **THEN** it reflects the universal-contract framing and pure-sequencer ownership
- **AND** the Per-Cycle Rework Choreography section is unchanged in semantics

#### R12: `stage-models.md` "Foreground limitation (advisory only)" section reconciled to single-mode
The "Foreground limitation (advisory only)" section (~lines 263–272), and the line-29 cross-reference
to it, MUST be reconciled so they no longer contradict the single execution mode for post-intake
stages. The section MUST note that full Gap-1a / uniform-tier closure is Change C; it MUST NOT
over-reach into Change C territory.

- **GIVEN** the foreground-limitation section asserting manual `/fab-continue` apply/hydrate/ship run
  untiered in the foreground
- **WHEN** the reconcile is applied
- **THEN** the section reflects that post-intake `/fab-continue` now dispatches (so the carve-out
  narrows to genuinely-foreground cases), with an explicit note that full uniform-tier closure is
  deferred to Change C
- **AND** the cross-reference at line 29 remains coherent

#### R13: `SPEC-fab-ff.md` / `SPEC-fab-fff.md` / `SPEC-fab-proceed.md` updated only if their skill prose changed
These spec mirrors MUST be updated ONLY if the corresponding skill prose (R8) changed. If the
wrappers are unchanged, their specs are left unchanged and that result is recorded.

- **GIVEN** the wrapper specs
- **WHEN** R8's survey concludes
- **THEN** each wrapper spec is edited iff its skill changed; otherwise left as-is

### Go: history-shape invariant test (test-only)

#### R14: A new Go test pins the history-shape-under-uniform-dispatch invariant
A new test MUST be added in `src/go/fab/internal/status/mutators_test.go` near
`TestStageMetrics_IterationsAccumulateAcrossReworkCycles`. It MUST drive the rework status sequence
`Finish(intake) → Finish(apply) → [Fail(review) → Reset(apply) → Finish(apply)]×N` twice — once with
`driver=""` (manual/foreground path) and once with `driver="fab-fff"` (dispatched path) — and assert
the produced `.history.jsonl` stage-transition entries are IDENTICAL in count, stage, action, from,
and reason, differing ONLY in the optional `driver` field. There MUST be NO production Go change
(Constitution VII: Test Integrity — the test verifies existing caller-agnostic behavior). The driver
difference is expected and acceptable (it is the recorded driver name, not a state difference).

- **GIVEN** the caller-agnostic Go state machine (verified: `driver` flows only into
  `applyMetricsSideEffect`, never into a state transition)
- **WHEN** the new test drives the rework sequence twice with `driver=""` and `driver="fab-fff"`
- **THEN** the two runs' `.history.jsonl` stage-transition entries match in count, stage, action,
  from, reason — differing only where `driver` is recorded
- **AND** `go test ./internal/status/...` passes with no production source change

### Non-Goals

- Change B (extract `_intake.md`) — pre-boundary de-duplication; independent of this change.
- Change C (uniform per-stage model tier) — depends on this change; full Gap-1a closure / effort-knob
  work belongs there. This change only stops *contradicting* single-mode; it does not write the
  uniform-tiering story.
- intake / ship / review-pr behavioral rewrites — intake is the boundary; ship/review-pr already
  self-manage transitions.
- Any production Go code change, any new `fab` CLI command, any `_cli-fab.md` change.
- `runtime/operator.md` reconcile — that is a HYDRATE concern (Assumption #8), not apply.

### Design Decisions

1. **Orchestrator owns `fab status` + `fab resolve-agent` (pure-sequencer)** — the block never owns
   transitions. *Why*: matches the Lego model (invocations control order/grouping/model-tier; blocks
   control work) and the finding's open-seam #3 recommendation; `_pipeline.md` already implements it.
   *Rejected*: keeping the `fab status` call inside the block as a "free implementation choice" — it
   would re-tangle the bookkeeping skin the change exists to remove.
2. **The "do NOT run `fab status`" prompt instruction becomes the universal block contract** rather
   than a per-caller conditional. *Why*: with one execution mode the instruction is invariant across
   callers; encoding it in the dispatch prompt (orchestrator side) keeps the block caller-agnostic.
   *Rejected*: a block-level flag (`skip §Verdict when subagent`) — explicitly forbidden by intake
   § 4 and R3.
3. **Test-only Go component, no production change** — the Go state machine is already caller-agnostic
   (KEY VERIFIED EVIDENCE in the intake), so the manual and dispatched paths issue identical CLI call
   sequences. *Why*: the test pins the invariant so a future skill-layer regression that diverges the
   two sequences is caught. *Rejected*: relying solely on the existing iteration test (Assumption #7
   resolved Certain — a dedicated history-shape test is added regardless).

## Tasks

### Phase 1: Skills source — `fab-continue.md` (highest-risk, independently revertable)

- [x] T001 In `src/kit/skills/fab-continue.md`, remove the three dual-mode `do NOT run fab status` blockquote conditionals (Apply Behavior ~L100, Review Behavior ~L156, Hydrate Behavior ~L176), identified by verbatim text <!-- R2 -->
- [x] T002 In `src/kit/skills/fab-continue.md`, rewrite Normal Flow (Step 1 dispatch table rows for apply/review/hydrate, Step 3 SRAD+Generation table, Step 4 status update, Step 5 output) and the Apply/Review/Hydrate Behavior sections so the post-intake path always dispatches a subagent and `/fab-continue` runs `fab resolve-agent <stage>` before dispatch as a one-stage sequencer that owns the transitions <!-- R1 -->
- [x] T003 In `src/kit/skills/fab-continue.md`, ensure the rewritten Review Behavior block always returns findings with no caller-branching flag — verify no "skip §Verdict when subagent" instruction is reintroduced in any shape <!-- R3 -->
- [x] T004 In `src/kit/skills/fab-continue.md`, keep the interactive § Verdict rework menu (Fail options table) intact and confirm the `review`/`failed` dispatch row and Reset Flow still reference it as Path A <!-- R4 -->
- [x] T005 In `src/kit/skills/fab-continue.md`, reconcile the header note (~L19) to single-mode: `/fab-continue` is a one-stage sequencer that resolves the tier and dispatches; drop the "foreground = advisory-only" framing for post-intake stages <!-- R5 -->

### Phase 2: Skills source — pipeline + preamble prose alignment

- [x] T006 In `src/kit/skills/_pipeline.md`, light prose alignment affirming orchestrator-as-pure-sequencer and that the "do NOT run `fab status`; return results only" prompt instruction is the universal block contract; make no structural change to the bracket <!-- R6 -->
- [x] T007 In `src/kit/skills/_pipeline.md`, verify the Auto-Rework Loop per-cycle choreography (fail+reset pair) and the line-86 history-shape invariant are preserved verbatim in semantics — no edit to which `fab status` commands fire or their order <!-- R7 -->
- [x] T008 [P] Survey `src/kit/skills/fab-ff.md`, `src/kit/skills/fab-fff.md`, `src/kit/skills/fab-proceed.md` for prose presuming block-owned transitions; edit only where found, otherwise leave unchanged and record the survey result <!-- R8 -->
- [x] T009 In `src/kit/skills/_preamble.md`, minimally reconcile the "Foreground is advisory-only" paragraph (§ Subagent Dispatch → Per-Stage Model Resolution) to stop contradicting single-mode for post-intake stages; do NOT write Change C's uniform-tiering story <!-- R9 -->

### Phase 3: Spec mirrors + stage-models.md

- [x] T010 In `docs/specs/skills/SPEC-fab-continue.md`, remove/replace the three conditional descriptions, update the advisory-only summary (L9) and the Summary to single-mode post-intake dispatch, reconcile the "Subagent rule (f006)" note to the universal contract, and KEEP the Verdict rework-menu (Path A) spec <!-- R10 -->
- [x] T011 [P] In `docs/specs/skills/SPEC-_pipeline.md`, align dispatch/transition-ownership prose to pure-sequencer / universal-contract framing without altering the Per-Cycle Rework Choreography semantics <!-- R11 -->
- [x] T012 [P] In `docs/specs/stage-models.md`, reconcile the "Foreground limitation (advisory only)" section (~L263–272) and the L29 cross-reference to single-mode for post-intake stages; note full Gap-1a/uniform-tier closure is Change C; do not over-reach <!-- R12 -->
- [x] T013 [P] Update `docs/specs/skills/SPEC-fab-ff.md` / `SPEC-fab-fff.md` / `SPEC-fab-proceed.md` only if T008 changed the corresponding skill prose; otherwise leave unchanged and record <!-- R13 -->
- [x] T017 In `docs/specs/skills/SPEC-_preamble.md`, reconcile the two "foreground advisory-only" sites left stale by T009's `_preamble.md` edit — the `## Summary` phrase (it lists "foreground advisory-only" among the Per-Stage Model Resolution items) and the § Subagent Dispatch ASCII-tree leaf (`│ foreground is advisory-only.`) — to single-mode post-intake dispatch (e.g. "per-stage selection applies on every post-intake stage; residual advisory only for a genuinely no-dispatch run"), annotated `(260613-fgxx)`; do NOT add Change C's uniform-tiering story <!-- R9 --> <!-- rework: review must-fix — mandated spec mirror of the edited _preamble.md omitted from the original spec phase -->

> **Rework note (cycle 1)**: T017 added to close the review must-fix — the constitution requires that an edited `src/kit/skills/*.md` file (here `_preamble.md`, via T009/R9) update its corresponding `docs/specs/skills/SPEC-*.md` mirror; the original Phase 3 scoped SPEC-fab-continue / SPEC-_pipeline / stage-models / wrappers but omitted SPEC-_preamble.

### Phase 4: Go history-shape test

- [x] T014 In `src/go/fab/internal/status/mutators_test.go`, add a new test near `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` that drives `Finish(intake) → Finish(apply) → [Fail(review) → Reset(apply) → Finish(apply)]×N` twice (driver="" and driver="fab-fff"), reading each run's `.history.jsonl`, and asserts the stage-transition entries are identical in count/stage/action/from/reason and differ only in the optional `driver` field; NO production Go change <!-- R14 -->
- [x] T015 Run `go test ./internal/status/...` from `src/go/fab/` and ensure the new test and the existing suite pass <!-- R14 -->

### Phase 5: Redeploy

- [x] T016 Run `fab sync` to redeploy the edited `src/kit/skills/*.md` to `.claude/skills/` and confirm success <!-- R1 -->

## Execution Order

- T001 → T002 → T003 → T004 → T005 (sequential within fab-continue.md — same file)
- T006 → T007 (same file `_pipeline.md`); T008 and T009 independent of each other and of T006/T007
- T010, T011, T012 independent ([P]); T013 depends on T008's outcome; T017 (SPEC-_preamble mirror) independent, depends only on T009 being done
- T014 → T015 (test must exist before running); T015 gates correctness
- T016 (fab sync) runs after all skill-source edits (T001–T009) are complete

## Acceptance

### Functional Completeness

- [x] A-001 R1: Post-intake `/fab-continue` (apply/review/hydrate) always dispatches a subagent and runs `fab resolve-agent <stage>` before dispatch; no foreground execution path remains for those stages
- [x] A-002 R2: None of the three dual-mode `do NOT run fab status` blockquotes remain in `src/kit/skills/fab-continue.md`
- [x] A-003 R3: The rewritten Review Behavior returns findings unconditionally with no caller-branching flag; the removed "skip §Verdict entirely" instruction is not reintroduced in any shape
- [x] A-004 R4: The interactive § Verdict rework menu (Fail options table) is intact and still referenced by the `review`/`failed` dispatch row and Reset Flow
- [x] A-005 R5: The `fab-continue.md` header note is reconciled to single-mode (one-stage sequencer resolves tier + dispatches); no "foreground = advisory" claim survives for post-intake stages
- [x] A-006 R6: `_pipeline.md` affirms orchestrator-as-pure-sequencer and the universal block contract with no structural bracket change
- [x] A-007 R8: The three wrappers are surveyed; edits made only where prose presumed block-owned transitions, otherwise unchanged with the survey recorded
- [x] A-008 R9: `_preamble.md`'s advisory-only paragraph is minimally reconciled (stops contradicting single-mode) without writing Change C's uniform-tiering story
- [x] A-009 R10: `SPEC-fab-continue.md` mirrors the single-mode rewrite, reconciles the f006 subagent-rule note, and keeps the Verdict rework-menu (Path A) spec
- [x] A-010 R11: `SPEC-_pipeline.md` aligns to pure-sequencer framing with the choreography unchanged in semantics
- [x] A-011 R12: `stage-models.md`'s foreground-limitation section and L29 cross-reference are reconciled to single-mode with an explicit Change-C deferral note
- [x] A-012 R13: Wrapper specs updated iff their skills changed; otherwise unchanged and recorded
- [x] A-013 R14: A new history-shape test exists in `mutators_test.go` driving the rework sequence twice (driver="" / "fab-fff") and asserting `.history.jsonl` entries identical except `driver`

### Behavioral Correctness

- [x] A-014 R7: The Auto-Rework Loop per-cycle choreography (fail+reset pair) and the line-86 history-shape invariant are unchanged in meaning; the same `fab status` commands fire per cycle in the same order
- [x] A-015 R14: `go test ./internal/status/...` passes (new test + existing suite); no production Go source was modified

### Removal Verification

- [x] A-016 R2: A repo-wide search confirms the three verbatim "do NOT run any `fab status` command" subagent conditionals are gone from `src/kit/skills/fab-continue.md` and not relocated elsewhere in the block bodies

### Scenario Coverage

- [x] A-017 R1: Dispatching the same change's stage from a cold session vs. mid-conversation would produce identical block output (the testable Lego invariant — verified by inspection of the rewritten dispatch contract)
- [x] A-018 R3: The Review Behavior block returns the same findings payload regardless of which caller (manual or orchestrator) dispatched it

### Edge Cases & Error Handling

- [x] A-019 R4: The `review`/`failed` dispatch row still resets apply and presents the rework menu directly (manual Path A), and is not collapsed into the autonomous loop
- [x] A-020 R12: The reconcile of `stage-models.md` does not introduce Change-C content (no uniform-tier/effort-knob rewrite) — scope discipline holds

### Code Quality

- [x] A-021 Pattern consistency: New/edited markdown follows the surrounding skill/spec prose conventions (blockquote notes, change-ID annotations where the file uses them, RFC-2119 phrasing); the Go test follows the host file's table-driven/fixture patterns and reads `.history.jsonl` consistent with the package's existing helpers
- [x] A-022 No unnecessary duplication: The universal block contract is stated once (in the dispatch prompt / `_pipeline.md` note) and referenced, not duplicated into each block; the test reuses existing fixture/load helpers rather than reimplementing them

### Documentation Accuracy

- [x] A-023 R10 R11 R12: Spec mirrors and stage-models.md accurately reflect the shipped skill behavior (constitution: docs are source of truth; specs are pre-implementation design intent reconciled to shipped behavior) with no residual dual-mode description
- [x] A-025 R9: `docs/specs/skills/SPEC-_preamble.md` (the mandated mirror of the edited `_preamble.md`) is reconciled — neither the `## Summary` nor the § Subagent Dispatch ASCII tree asserts "foreground advisory-only" in a way that contradicts single-mode post-intake dispatch; no Change-C uniform-tiering content added

### Cross References

- [x] A-024 R9 R12: Cross-references between `_preamble.md`'s advisory-only paragraph, `fab-continue.md`'s header note, and `stage-models.md`'s foreground section remain mutually consistent after the minimal reconcile (no dangling "see § Foreground limitation" pointer that now contradicts single-mode)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Edit ONLY `src/kit/skills/*.md` (canonical source); `.claude/skills/*.md` is gitignored deploy output — run `fab sync` after editing.

## Deletion Candidates

The change removed three blockquote conditionals from `src/kit/skills/fab-continue.md` (the Apply/Review/Hydrate "When invoked as a subagent: do NOT run `fab status`…" notes) — but those were inline prose inside still-live behavior sections, not separable code units, and they were replaced in place by the single-mode block framing. No call site, function, constant, or file became dead as a result of this change. The only new code (`mutators_test.go`'s `TestHistoryShape_IdenticalRegardlessOfDriver` + its three helpers) is additive test coverage; it supersedes nothing.

- **None — no existing code was made redundant or unused.** The three removed conditionals were prose, replaced in place (R2); the existing `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` is NOT redundant with the new history-shape test (it asserts iteration *accumulation*, the new one asserts *driver-blind history shape* — orthogonal invariants), so it is retained. `runtime/operator.md` reconcile and the three memory-file updates are HYDRATE-stage concerns, not deletions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Edit only `src/kit/skills/*.md`; run `fab sync` to redeploy; the committed change is the `src/kit/` edits, never `.claude/skills/`. | Constitution + intake Assumption #2 mandate canonical-source-only edits. No interpretation. | S:95 R:90 A:100 D:95 |
| 2 | Certain | No production Go change; the only Go change is a new test in `mutators_test.go` (intake Assumption #7, Certain). No new `fab` CLI command; `_cli-fab.md` unchanged. | Intake states it explicitly; Go state machine verified caller-agnostic. | S:95 R:90 A:95 D:90 |
| 3 | Certain | Requirements organized by affected-file domain (fab-continue / pipeline+preamble / specs+stage-models / Go test) rather than a runtime feature taxonomy. | This is a prose-reorganization change with no runtime domains; file-area grouping is the only sensible structure and matches the intake's § Impact layout. | S:90 R:85 A:90 D:90 |
| 4 | Confident | The three conditionals are uniquely identified by their verbatim blockquote text, not line numbers (numbers may have drifted from the intake's ~100/156/176). | Intake Assumption #4 (Confident); verified by reading the current `src/kit/skills/fab-continue.md` — the three blockquotes are present at L100/L156/L176 in the source as of apply. | S:95 R:80 A:90 D:85 |
| 5 | Confident | Header-note (R5) and `_preamble.md`/`stage-models.md` (R9/R12) wordings are an apply-time judgment band — minimal reconcile that neutralizes the contradiction without writing Change C's uniform-tiering story. | Intake Assumption #9 (Confident): minimal-reconcile chosen after clarify; exact wording is judgment, hence Confident not Certain. | S:90 R:75 A:80 D:80 |
| 6 | Confident | The new Go test builds a real `fab/changes/{name}/` layout in a temp dir and passes that `fab/` path as `fabRoot` so `log.Transition`'s `resolve.ToAbsDir(fabRoot, statusFile.Name)` resolves and `.history.jsonl` is actually written and readable (the existing iteration test passes a bare temp dir as fabRoot, so its best-effort log silently no-ops — that shape will not work for asserting on history). | Verified by reading `log.go` (`appendJSON` → `{changeDir}/.history.jsonl`), `resolve.go` (`ToAbsDir` = `fabRoot/changes/{folder}`), and `status.go` (`applyMetricsSideEffect` calls `log.Transition(fabRoot, statusFile.Name, …)`). Assertion shape from intake § 5; test-harness plumbing is an apply-time decision. | S:80 R:80 A:80 D:75 |
| 7 | Confident | Each rework run in the new test uses a distinct change folder/`.status.yaml` (so the two `.history.jsonl` files are independent), then the two transition-entry sequences are compared field-by-field (stage/action/from/reason) ignoring `ts` and `driver`. | The `ts` field is a timestamp and `driver` is the intentional difference; both must be excluded from the equality assertion. Matches the intake's "identical in count, stage, action, from, and reason, differing only in driver" contract. | S:85 R:80 A:80 D:80 |

7 assumptions (3 certain, 4 confident, 0 tentative).
