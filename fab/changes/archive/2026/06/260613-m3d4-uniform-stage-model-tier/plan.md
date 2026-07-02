# Plan: Apply per-stage model tier uniformly + inject effort via subagent prompt

**Change**: 260613-m3d4-uniform-stage-model-tier
**Intake**: `intake.md`

## Requirements

> This change is **Change C** of a three-part refactor, gated on **Change A**
> (`intake-is-the-context-boundary`, landed as `260613-fgxx`). Change A removed the
> post-intake foreground execution path, closing **Gap 1a** of the model-tier finding.
> This change is therefore scoped to **Gap 1b** (compliance visibility), **Gap 2's effort
> half** (inject effort via the subagent prompt), and the **doc reconciliation** that
> follows from the now-uniform behavior. Scope is **skills/prose + docs only — NO Go**.

### Dispatch: Per-stage tier compliance visibility (Gap 1b)

#### R1: Resolved tier surfaced at every per-stage dispatch site
The orchestrator/sequencer MUST surface (echo into the dispatch prompt and/or log) the
resolved `model=/effort=` lines immediately after running `fab resolve-agent <stage>` and
before dispatching the stage's sub-agent, so a skipped or mis-resolved tier is **visible in
output** rather than silently consumed. This MUST apply at every per-stage dispatch site:
`_pipeline.md` Behavior "Per-stage model resolution" note + Steps 1 (apply), 2 (review), 3
(hydrate), and Auto-Rework Loop items 3 (re-apply) and 4 (re-review); and `fab-fff.md` Steps
4 (ship) and 5 (review-pr).

- **GIVEN** an orchestrator about to dispatch the apply sub-agent
- **WHEN** it runs `fab resolve-agent apply` (yielding `model=claude-opus-4-8 effort=high`)
- **THEN** the resolved `model=/effort=` lines are surfaced (echoed into the dispatch prompt and/or logged)
- **AND** a run that skipped resolution shows no such lines, making the miss visible

#### R2: Lightweight non-empty guard where cleanly expressible (Gap 1b)
The dispatch contract SHOULD add a prose-level assertion that the resolved `model=/effort=`
lines are non-empty before dispatch **when it can be expressed cleanly**; where a guard would
add awkward control flow, visibility-in-output alone (R1) satisfies the requirement. Prose
only — NO Go.

- **GIVEN** the canonical Per-Stage Model Resolution contract in `_preamble.md`
- **WHEN** the contract is stated
- **THEN** it notes that an all-empty resolution (both `model=` and `effort=` empty) is a signal worth surfacing/asserting, expressed as prose without awkward control flow
- **AND** no Go code or new skill mechanism is introduced for it

### Dispatch: Effort injection via subagent prompt (Gap 2 effort half)

#### R3: Resolved effort injected as an imperative subagent-prompt instruction
Because the Claude Code Agent tool has **no `effort` parameter**, the dispatching skill MUST
inject the resolved effort into the **subagent prompt** as an explicit imperative instruction
(the model half stays on the Agent tool's `model` param, unchanged). This MUST apply at every
per-stage dispatch site named in R1. Where the resolved effort is **empty**, the instruction
MUST be omitted (mirroring the existing "empty effort ⇒ omit" contract).

- **GIVEN** an orchestrator dispatching the review sub-agent with resolved `effort=xhigh`
- **WHEN** it builds the Agent-tool dispatch prompt
- **THEN** the prompt carries an imperative instruction such as ``Operate at `xhigh` reasoning effort for this task.``
- **AND** when the resolved effort is empty, no effort instruction is added

#### R4: Review resolves once → same effort instruction to both reviewers + merge
The `review` stage MUST resolve `fab resolve-agent review` **once** and apply the **same**
effort instruction to BOTH reviewer sub-agents (inward + outward) and the merge — preserving
the existing "review resolves once" contract.

- **GIVEN** the review block resolving its tier once (`effort=xhigh`)
- **WHEN** it dispatches the inward and outward reviewer sub-agents
- **THEN** both prompts carry the identical `xhigh` effort instruction, and the merge runs at the same tier

### Docs: Reconcile findings + spec + preamble + header note with uniform behavior

#### R5: Finding doc reflects Gap 1a closed, Gap 1b + Gap 2-effort addressed, harness ask residual
`docs/findings/per-stage-model-tier-application.md` MUST be updated so: Gap 1a's "Closed by..."
note reads as **closed** by the landed Change A (not a forward projection); `**Status:**` and
the per-path table reflect that there is no longer a foreground-vs-subagent split; Gap 1b and
Gap 2's effort-half are recorded as **addressed by this change** (visibility + prompt-injection);
and § Suggested directions item 4 (per-subagent `effort` param on the Agent tool) **remains** and
is marked as the **residual** after this change.

- **GIVEN** the finding doc post-Change-A
- **WHEN** this change lands
- **THEN** Gap 1a is marked closed, Gap 1b + Gap 2-effort marked addressed (with the seam used), and item 4 remains the residual harness ask
- **AND** no Go code or skill mechanism is built for the harness ask

#### R6: Spec reconciled with uniform behavior + effort-via-prompt
`docs/specs/stage-models.md` § Foreground limitation and § Skill wiring MUST be reconciled
with the now-uniform behavior: no foreground advisory path for post-intake stages (already
reflected post-A — reconcile only the tier-mechanism residue), and effort is now **injected via
the subagent prompt** rather than dropped. As human-curated pre-implementation design intent
(Constitution VI), this section is **edited**, not auto-generated.

- **GIVEN** § Skill wiring stating "Empty effort → omit the effort flag" (which read as "effort is dropped")
- **WHEN** this change lands
- **THEN** the spec states effort is injected via the subagent prompt (omitted only when empty), and the § Foreground limitation scope note no longer defers the effort half as unwritten

#### R7: Preamble documents the effort-via-prompt seam + harness-adapter alias detail
`src/kit/skills/_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution MUST document the
**effort-via-prompt** seam: model is injected via the Agent tool's `model` param, effort is
injected via an explicit subagent-prompt instruction (no Agent-tool effort param). It MUST
reconcile the residual-advisory paragraph for the tier mechanism (Gap 1a already removed by A —
do not re-remove what A removed) and add the compliance-visibility (R1) expectation. It SHOULD
note the harness-adapter detail that the Agent tool's `model` param takes a short alias
(`opus`/`sonnet`/`haiku`/`fable`), NOT the full `claude-opus-4-8` id that `fab resolve-agent`
emits — the orchestrator maps the resolved id to the alias.

- **GIVEN** the canonical Per-Stage Model Resolution contract
- **WHEN** this change lands
- **THEN** it documents effort-via-prompt as the effort seam, surfaces/asserts the resolved lines (visibility), and notes the id→alias mapping at the harness-adapter boundary
- **AND** it does not re-remove the foreground-advisory language Change A already removed

#### R8: fab-continue header note reconciled (Gap 1a residue)
`src/kit/skills/fab-continue.md` header per-stage-model note MUST be reconciled with the uniform
behavior. If Change A already removed the foreground-advisory caveat (it did — the note already
reframes to the one-stage sequencer with "no foreground execution path... to leave the tier
merely advisory"), leave it; this change only adds the effort-via-prompt mention if it improves
local consistency, without duplicating A's reconciliation.

- **GIVEN** the `fab-continue.md` header note already reframed by Change A
- **WHEN** this change lands
- **THEN** the note is consistent with effort-via-prompt + the canonical contract, with no re-removal of A's already-deleted advisory language

### Docs: SPEC mirrors (Constitution requirement)

#### R9: Every edited skill file's SPEC mirror updated
Per the Constitution Additional Constraint (skill changes MUST update the corresponding
`docs/specs/skills/SPEC-*.md`), the SPEC mirror of every edited `src/kit/skills/*.md` MUST be
updated to reflect the effort-via-prompt + visibility behavior. Affected mirrors:
`SPEC-_pipeline.md`, `SPEC-_preamble.md`, `SPEC-fab-fff.md`, `SPEC-fab-continue.md`.

- **GIVEN** edits to `_pipeline.md`, `_preamble.md`, `fab-fff.md`, `fab-continue.md`
- **WHEN** this change lands
- **THEN** each corresponding `SPEC-*.md` mirror documents the new effort-via-prompt + visibility behavior and the Gap 1a removal attribution

### Non-Goals

- **No Go change** — `src/go/**` untouched; `fab resolve-agent`'s signature is unchanged, so `_cli-fab.md` is untouched.
- **Not building the harness ask** — the per-subagent `effort` param on the Agent tool is documented as residual only; no skill mechanism or Go is added.
- **Not re-doing Change A's work** — the every-post-intake-stage-dispatches-a-subagent mechanic and the broad foreground-advisory deletions are A's deliverable, already landed; this change reconciles only the tier-mechanism-specific residue.
- **No migration** — no user-data restructure (config / `.status.yaml` / archive layout).
- **Memory not touched at apply** — `pipeline/execution-skills`, `pipeline/planning-skills` are handled at HYDRATE, not apply.

### Design Decisions

1. **Effort instruction wording**: ``Operate at `xhigh` reasoning effort for this task.`` (imperative, backtick-quoted level, single sentence). — *Why*: imperative + unambiguous per the intake's stated criterion; reads naturally appended to a dispatch prompt; backticks make the level scannable. — *Rejected*: "Use xhigh effort" (less imperative), a multi-sentence block (verbose for a per-dispatch line).
2. **Visibility seam = echo into the dispatch prompt AND log line**: surface the resolved `model=/effort=` in the dispatch prompt (it travels with the effort instruction anyway) and instruct the orchestrator to emit it in its step output. — *Why*: the prompt already must carry the effort line for R3, so co-locating the model line there is zero marginal cost and makes the per-dispatch tier self-documenting; the output-log half is what makes a *skipped* resolution visible. — *Rejected*: log-only (doesn't help the subagent self-document), prompt-only (a skipped dispatch with no prompt line is invisible in the orchestrator's own output).
3. **Guard expressed as prose assertion in the canonical contract only** (`_preamble.md`), not repeated at every site. — *Why*: the canonical Per-Stage Model Resolution contract is the single source the dispatch sites already defer to; asserting non-emptiness there avoids awkward per-site control flow (Assumption #9's "fall back to visibility if a guard would add awkward control flow"). — *Rejected*: a per-site `if empty then…` branch (awkward control flow, six duplicated copies).

## Tasks

### Phase 1: Canonical contract (the single source the sites defer to)

- [x] T001 Update `src/kit/skills/_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution: (a) document the **effort-via-prompt** seam — model via Agent tool `model` param, effort via an explicit imperative subagent-prompt instruction (omit when empty); (b) add the **compliance-visibility** expectation (surface the resolved `model=/effort=` into the dispatch prompt and/or orchestrator output) plus the prose non-empty-assertion note; (c) reconcile the harness-adapter paragraph to note the Agent `model` param takes a short alias (`opus`/`sonnet`/`haiku`/`fable`), not the full `claude-*` id `fab resolve-agent` emits (orchestrator maps id→alias); (d) update the trailing parenthetical in the "Per-stage selection applies on every post-intake stage" paragraph so the per-sub-agent effort knob is described as injected-via-prompt-now with the Agent-tool effort param as the residual harness ask (do NOT re-remove A's already-deleted foreground-advisory language). <!-- R2 --> <!-- R3 --> <!-- R7 -->

### Phase 2: Dispatch sites (defer to the contract; add visibility + effort line)

- [x] T002 [P] Update `src/kit/skills/_pipeline.md`: (a) the Behavior "Per-stage model resolution" note — extend so the resolved `model=/effort=` is surfaced and the effort half is injected into the dispatch prompt; (b) Step 1 (apply), Step 2 (review — once, both reviewers + merge), Step 3 (hydrate) dispatch instructions; (c) Auto-Rework Loop item 3 (re-apply) and item 4 (re-review) — each: surface the resolved tier and add the effort instruction to the dispatch prompt (omit effort when empty). <!-- R1 --> <!-- R3 --> <!-- R4 -->
- [x] T003 [P] Update `src/kit/skills/fab-fff.md`: Step 4 (ship — `fab resolve-agent ship`) and Step 5 (review-pr — `fab resolve-agent review-pr`), plus the "Per-stage model" blockquote note: surface the resolved tier and inject the resolved effort instruction into the `/git-pr` / `/git-pr-review` dispatch prompts (omit effort when empty). <!-- R1 --> <!-- R3 -->
- [x] T004 [P] Reconcile `src/kit/skills/fab-continue.md` header per-stage-model note (~line 19) and the Review Behavior "Per-stage model resolution (nested reviewers)" note: ensure consistency with effort-via-prompt + the canonical contract; the nested-reviewers note applies the same once-resolved effort instruction to both reviewers + merge. Do NOT re-remove A's already-deleted foreground-advisory language. <!-- R4 --> <!-- R8 -->

### Phase 3: Findings + spec doc reconciliation

- [x] T005 [P] Update `docs/findings/per-stage-model-tier-application.md`: (a) `**Status:**` line; (b) Gap 1a's "Closed by..." note → marked CLOSED by landed Change A (`260613-fgxx`); (c) the per-path table (~lines 40–46) → reflect no foreground-vs-subagent split; (d) Gap 1b + Gap 2-effort marked addressed by this change (visibility + prompt-injection) with the residual being the harness ask; (e) ensure § Suggested directions item 4 (per-subagent effort param) remains and is marked the residual. <!-- R5 -->
- [x] T006 [P] Update `docs/specs/stage-models.md` § Skill wiring and § Foreground limitation: § Skill wiring — effort is injected via the subagent prompt (omitted only when empty), not silently dropped; § Foreground limitation scope note — remove the deferral of the effort half as "intentionally NOT written here" now that this change writes it; keep the residual harness-ask framing for the Agent-tool effort param. Human-curated design intent — edit, do not auto-generate. <!-- R6 -->

### Phase 4: SPEC mirrors (Constitution requirement)

- [x] T007 [P] Update `docs/specs/skills/SPEC-_preamble.md` Summary + the Per-Stage Model Resolution Flow node: document the effort-via-prompt seam, compliance visibility, and the id→alias harness-adapter detail (attribute to this change). <!-- R9 -->
- [x] T008 [P] Update `docs/specs/skills/SPEC-_pipeline.md`: document that each per-stage dispatch site surfaces the resolved tier and injects the effort instruction into the dispatch prompt (Steps 1–3 + rework items 3/4). <!-- R9 -->
- [x] T009 [P] Update `docs/specs/skills/SPEC-fab-fff.md`: document that Steps 4–5 surface the resolved tier and inject effort into the `/git-pr` / `/git-pr-review` dispatch prompts. <!-- R9 -->
- [x] T010 [P] Update `docs/specs/skills/SPEC-fab-continue.md`: document the effort-via-prompt seam in the one-stage sequencer + nested reviewers (consistent with the canonical contract); note Gap 1a closed by A and the effort half now addressed by this change. <!-- R9 -->

### Phase 5: Meta-consistency pass

- [x] T011 Cross-check that `_pipeline.md` / `_preamble.md` / `fab-fff.md` / `fab-continue.md` and their SPEC mirrors and the finding/stage-models docs are mutually consistent: the effort-via-prompt seam, the once-resolved-review rule, the visibility expectation, and the Gap 1a-closed / harness-ask-residual framing all agree across files. No new claim contradicts another. <!-- R1 --> <!-- R3 --> <!-- R5 --> <!-- R6 --> <!-- R7 -->

### Phase 6: Sibling-class completeness (rework — review cycle 1)

<!-- rework cycle 1: review found A-014 violation — the change updated fab-fff.md / SPEC-fab-fff.md to the
     two-seam (model-param + effort-via-prompt) phrasing but left their behaviorally-identical siblings
     fab-ff.md / SPEC-fab-ff.md stating the now-false single-seam "passes the resolved model+effort to the
     Agent dispatch" framing, creating a repo-internal contradiction. fab-ff was never in the dispatch-site
     list (intake named only _pipeline.md + fab-fff.md), so no Phase-1–5 task covered it. Sweep by class. -->

- [x] T012 Reconcile `src/kit/skills/fab-ff.md` per-stage-model blockquote note (~line 37) — the structural twin of `fab-fff.md`'s note (T003): bring it to the same two-seam phrasing (surface the resolved `model=/effort=`; model via the Agent `model` param; effort via the imperative subagent-prompt instruction, omitted when empty), OR shorten it to defer to `_preamble.md` § Per-Stage Model Resolution without restating the now-stale single-seam mechanics. `fab-ff.md` runs the same `_pipeline.md` bracket as `fab-fff.md`, so its dispatch behavior (Steps 1–3) is already covered by T002; this fixes ONLY the fab-ff.md note that pre-states the superseded contract. Do NOT re-remove Change A's already-deleted foreground-advisory language. <!-- R3 --> <!-- R4 -->
- [x] T013 Update `docs/specs/skills/SPEC-fab-ff.md` (~line 5) to mirror the T012 edit and match the `SPEC-fab-fff.md` (T009) reconciliation — the two-seam phrasing with the `(…, 260613-m3d4)` change attribution, so the SPEC mirror does not contradict its skill file or its `SPEC-fab-fff.md` sibling. Constitution skill→SPEC requirement. <!-- R9 -->
- [x] T014 Soften `docs/specs/skills/SPEC-fab-continue.md` line ~7 (the unedited `260613-fgxx` paragraph) so it no longer pre-states the superseded single-seam framing ("passes the resolved model/effort to the Agent dispatch") that the next paragraph (line ~11, edited in T010) overrides — defer to the "Per-stage model" description below it. Should-fix internal-consistency wrinkle of the same class as T012/T013. <!-- R9 -->
- [x] T015 Re-run the T011 meta-consistency pass with scope widened to the **whole `fab-ff`/`fab-fff` orchestrator sibling class** (both skill files + both SPEC mirrors + `_pipeline.md`/`_preamble.md`/`fab-continue.md` + finding/stage-models docs): grep the repo for the stale single-seam phrasing ("passes the resolved model+effort to the Agent dispatch" / "model/effort to the Agent dispatch") and confirm every in-scope occurrence is reconciled. Memory files (`docs/memory/_shared/context-loading.md` etc.) carrying the stale phrasing are OUT of scope — deferred to hydrate — and must be left untouched. <!-- R1 --> <!-- R3 --> <!-- R9 -->

## Execution Order

- T001 first (the canonical contract the dispatch sites defer to — establishes the wording/seam the rest mirror).
- T002–T006 may run in parallel after T001 (independent files; each mirrors the contract).
- T007–T010 (SPEC mirrors) run after their source skill edits (T007 after T001; T008 after T002; T009 after T003; T010 after T004), but are mutually independent `[P]`.
- T011 last in the original pass (consistency pass over the finished set).
- T012–T014 (rework cycle 1) run after T011; T013 after T012 (SPEC mirror follows its skill edit); T014 independent. T015 last (re-run the consistency pass with class-wide scope).

## Acceptance

### Functional Completeness

- [x] A-001 R1: Every per-stage dispatch site in `_pipeline.md` (Behavior note + Steps 1–3 + Auto-Rework items 3/4) and `fab-fff.md` (Steps 4–5) surfaces the resolved `model=/effort=` (echoed into the dispatch prompt and/or output).
- [x] A-002 R3: Every per-stage dispatch site injects the resolved effort as an imperative subagent-prompt instruction, with explicit "omit when empty" handling.
- [x] A-003 R5: `docs/findings/per-stage-model-tier-application.md` marks Gap 1a closed (by landed Change A), Gap 1b + Gap 2-effort addressed by this change, and item 4 the residual harness ask.
- [x] A-004 R6: `docs/specs/stage-models.md` § Skill wiring + § Foreground limitation state effort-via-prompt (not dropped) and carry no unreconciled foreground-advisory tier residue.
- [x] A-005 R7: `src/kit/skills/_preamble.md` documents the effort-via-prompt seam, the compliance-visibility expectation, and the id→alias harness-adapter detail.
- [x] A-006 R9: `SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-fff.md`, `SPEC-fab-continue.md` each document the effort-via-prompt + visibility behavior and the Gap 1a-closed attribution.

### Behavioral Correctness

- [x] A-007 R3: Where resolved effort is empty, the dispatch prompt carries NO effort instruction (mirroring the existing empty-effort-omit contract), stated explicitly at the canonical contract.
- [x] A-008 R4: The review stage resolves `fab resolve-agent review` once and the SAME effort instruction governs both reviewer sub-agents (inward + outward) and the merge — the "review resolves once" contract is preserved, not duplicated per reviewer.
- [x] A-009 R8: The `fab-continue.md` header note is consistent with the canonical contract and effort-via-prompt; Change A's already-removed foreground-advisory language is NOT re-removed or re-introduced.

### Removal Verification

- [x] A-010 R2: No Go file under `src/go/**` is modified; `_cli-fab.md` is unchanged (resolver signature unchanged); the harness ask is documented but not built.

### Edge Cases & Error Handling

- [x] A-011 R2: The non-empty guard is expressed as prose (a surface/assert note in the canonical contract), not as awkward per-site control flow; if a clean guard was infeasible, visibility-in-output alone is present and the assumption row records the fallback.

### Code Quality

- [x] A-012 Pattern consistency: New prose matches the surrounding skill-file prose density, idiom, and blockquote-note conventions; SPEC-mirror edits match the existing `(260613-xxxx)` change-attribution style.
- [x] A-013 No unnecessary duplication: The effort/visibility behavior is stated canonically in `_preamble.md` and referenced (not re-derived verbatim) at each dispatch site; edits do not duplicate Change A's reconciliation.

### Documentation Accuracy (checklist.extra_categories)

- [x] A-014 Cross-references between `_pipeline.md` / `_preamble.md` / `fab-fff.md` / `fab-ff.md` / `fab-continue.md` and their SPEC mirrors and the finding/stage-models docs are mutually consistent (the meta-consistency pass, T011 + the class-wide re-run T015) — no file claims a behavior another contradicts. **In particular, the `fab-ff`/`fab-fff` orchestrator sibling pair (and their SPEC mirrors) agree on the two-seam dispatch contract** — neither retains the superseded single-seam "passes the resolved model+effort to the Agent dispatch" phrasing.
- [x] A-015 R9: No in-scope skill or SPEC file retains the stale single-seam phrasing ("passes the resolved model+effort to the Agent dispatch" / "model/effort to the Agent dispatch") — verified by a class-wide grep (T015). Memory files carrying the phrasing are out of scope (deferred to hydrate) and remain untouched.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Markdown-only / CommonMark throughout; NO Go.
- This change is itself about the per-stage-model dispatch contract — the dispatch-site edits, their SPEC mirrors, and the `_preamble.md` contract must be mutually consistent (meta-consistency is load-bearing here).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change A landed as `260613-fgxx`; scope is Gap 1b + Gap 2-effort + doc reconciliation only (Gap 1a already closed, not duplicated) | Verified in source: `_preamble.md` line 350, `fab-continue.md` line 19, and `stage-models.md` § Foreground limitation all reference `260613-fgxx` and state no foreground post-intake path remains. Intake "IMPORTANT DEPENDENCY" + both finding docs corroborate. | S:95 R:75 A:95 D:95 |
| 2 | Certain | Edit ONLY `src/kit/skills/*` (canonical) + `docs/*`; never `.claude/skills/*` (gitignored deployed copies); no Go | Constitution Principle V + Additional Constraints; intake constraints; `fab resolve-agent` signature unchanged | S:100 R:75 A:100 D:100 |
| 3 | Certain | Skill-file edits update their SPEC mirrors: `SPEC-_pipeline.md`, `SPEC-_preamble.md`, `SPEC-fab-fff.md`, `SPEC-fab-continue.md` | Constitution Additional Constraint (skill change → SPEC update); all four mirrors confirmed to exist | S:95 R:75 A:95 D:90 |
| 4 | Confident | Effort-instruction wording: ``Operate at `xhigh` reasoning effort for this task.`` (imperative, backtick level, single sentence) | Intake gives this as the example and explicitly delegates final wording to apply ("imperative, unambiguous, omitted when empty"); chosen for naturalness appended to a dispatch prompt | S:90 R:85 A:80 D:75 |
| 5 | Confident | Visibility seam = surface resolved `model=/effort=` into the dispatch prompt AND instruct the orchestrator to emit it in step output | Intake item 1 says "echoed into the dispatch prompt and/or logged"; both halves are cheap (the prompt already carries the effort line) and together cover both a mis-resolved tier (prompt) and a skipped resolution (output) | S:90 R:80 A:80 D:75 |
| 6 | Confident | Lightweight guard expressed as a prose non-empty assertion in the canonical `_preamble.md` contract only (not per-site control flow) | Assumption #9 (confirmed): add guard "when cleanly feasible; fall back to visibility-in-output alone if a guard would add awkward control flow." A canonical-only prose assertion is the clean form; per-site branching is the awkward form correctly avoided | S:90 R:80 A:75 D:75 |
| 7 | Confident | Capture the harness-adapter id→alias detail (Agent `model` param takes `opus`/`sonnet`/`haiku`/`fable`, not the full `claude-opus-4-8` id) in `_preamble.md` § Per-Stage Model Resolution | Orchestrator instruction names this as a real harness detail "worth capturing... mention it if it fits naturally, keep it concise"; it is exactly the harness-adapter boundary the section already describes | S:85 R:85 A:80 D:80 |
| 8 | Confident | `fab-continue.md` header note needs only light touch (A already reframed it); no re-removal of A's deleted advisory language | Intake "Note on overlap with Change A" + verified source: the header note already reads "no foreground execution path... to leave the tier merely advisory" | S:90 R:80 A:80 D:80 |

8 assumptions (3 certain, 5 confident, 0 tentative).
