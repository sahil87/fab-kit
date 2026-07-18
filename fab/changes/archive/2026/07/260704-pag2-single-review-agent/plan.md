# Plan: Single Review Agent

**Change**: 260704-pag2-single-review-agent
**Intake**: `intake.md`

## Requirements

<!-- Absorbed from the former spec.md. RFC-2119 keywords; each requirement carries a stable R# ID
     and at least one GIVEN/WHEN/THEN scenario. This is a markdown-only documentation refactor of
     the review stage's dispatch shape; the "code" here is the kit's skill/spec prose. -->

### Review Dispatch: Single Agent

#### R1: One review dispatch, one merged procedure
`src/kit/skills/_review.md` SHALL define the review stage as a **single** dispatched sub-agent that runs the whole review inline — there SHALL be no nested Agent-tool dispatch and no parallel dispatch inside the review block. The single agent's prompt MUST carry BOTH checklists: the former inward **plan-conformance steps** (today's Validation Steps 1–8: tasks-all-`[x]`, acceptance-item inspection with in-place checkbox mutation in `plan.md`, scoped test runs, requirements spot-check, memory drift check, code-quality check, parsimony pass with its four-category table + 100-net-added-lines advisory threshold + `[docs, chore, ci]`/`## Parsimony Pass Enabled: false` skip conditions keyed on `change_type`, and the deletion-candidate prompt writing/replacing the `## Deletion Candidates` section) AND the former outward **holistic-diff focus areas** (interface contract violations, pattern inconsistencies vs `docs/memory/`, missing cross-references, behavioral regressions requiring full-repo context, structural issues). The agent receives the diff (`git diff <base>...HEAD` vs the default-branch merge-base) + changed-file list and has full repo read access.

- **GIVEN** a project at the review stage
- **WHEN** the review stage dispatches
- **THEN** exactly one review sub-agent is dispatched, and its prompt carries both the plan-conformance checklist and the holistic-diff focus areas
- **AND** there is no nested reviewer sub-agent, no "Parallel Dispatch", and no "Findings Merge" step in `_review.md`

#### R2: Framing line, no read-prohibition, no phase-ordering
The merged prompt MUST carry the framing line, verbatim: *"conformance to plan.md is necessary but not sufficient; also judge the diff on its own merits against the repo."* There SHALL be no read-prohibition on `plan.md` and no phase-ordering instruction (the agent MAY read anything in any order).

- **GIVEN** the single-agent review procedure in `_review.md`
- **WHEN** the merged prompt is composed
- **THEN** the verbatim framing line is present and no "don't read plan.md first" / phase-ordering instruction appears

#### R3: Codex→Claude cascade preserved as an inline step
The Codex→Claude external-tool cascade SHALL survive as a step **inside** the single agent's procedure, controlled exactly as today by `fab/project/code-review.md` § Review Tools (absent section/entry = enabled; `- codex: false` / `- claude: false` disable; graceful empty-findings no-op when all tools unavailable/disabled). The `copilot` entry stays `/git-pr-review`-only.

- **GIVEN** the single review agent
- **WHEN** it runs the diff review
- **THEN** it applies the Codex→Claude cascade as one of its own steps, with the same § Review Tools gating and graceful no-op semantics as today

#### R4: Outcome contract byte-unchanged
The review outcome contract SHALL be unchanged. The single agent returns ONE unified three-tier findings list (must-fix / should-fix / nice-to-have, each with file:line where applicable). The deterministic pass/fail rule (any must-fix → fail; no must-fix, including zero findings → pass) SHALL be restated where it lives today. "Findings Merge" SHALL disappear as a distinct mechanic (single source; deduplication moot). The `{stage}-result.yaml` review schema in `_preamble.md` § Dispatch-Prompt Obligations (the `status` vs `verdict` split, the `findings.must_fix/should_fix/nice_to_have` tiers) SHALL be byte-unchanged. Orchestrator-owned preconditions, verdict transitions, and the rework loop SHALL be unchanged.

- **GIVEN** the reworked review procedure
- **WHEN** the review returns
- **THEN** the findings tiers, the pass/fail rule, the `{stage}-result.yaml` schema, and all orchestrator-owned mechanics are identical to before
- **AND** `git diff` shows no change to the result-schema YAML block in `_preamble.md` / `SPEC-_preamble.md`

#### R5: Model resolution simplifies to one consumer
Review model resolution SHALL be a single `fab resolve-agent review --alias` at the sequencer's dispatch — the only review resolution. The `_preamble.md` § "Review resolves once" paragraph (two reviewers + merge, merge-at-reviewer-tier tradeoff) SHALL be rewritten so review is unexceptional (one stage, one resolution, like every other stage). The nested-reviewer resolution note in `fab-continue.md` Review Behavior SHALL be deleted.

- **GIVEN** the review stage
- **WHEN** its model is resolved
- **THEN** there is exactly one `fab resolve-agent review --alias` resolution (the sequencer's), and no skill or spec describes a second nested resolution "for both reviewers + merge"

#### R6: `mode` parameter survives, adoption variant is a prompt variant
The `mode` parameter SHALL survive with the same gating semantics. `full` (default) = merged checklist incl. plan-conformance steps + preconditions. The adoption mode SHALL be the SAME single dispatch with the plan-conformance steps omitted from the prompt (no tasks/acceptance verification, no checkbox mutation, no parsimony/deletion-candidate steps), preconditions skipped, diff + focus areas + cascade retained; zero findings still passes best-effort. The mode value name SHALL be renamed from `outward-only` to `diff-only` (all callers in-kit; `fab-adopt.md` is the only passer), sweeping atomically. `_generation.md`'s Plan-from-Diff acceptance-stub text SHALL be reworded to match.

- **GIVEN** `/fab-adopt` at its review step
- **WHEN** it dispatches review in adoption mode
- **THEN** it passes `mode: diff-only`, the single dispatch runs the merged procedure minus the plan-conformance steps, preconditions are skipped, and zero findings passes best-effort
- **AND** no in-kit file still references the value `outward-only`

#### R7: Nesting-degradation machinery deleted
The nesting-degradation machinery SHALL be deleted end-to-end. `docs/specs/harness-adapters.md` § "Nesting degradation (the `review` stage)" SHALL be removed; the § "Skill wiring is NOT part of the contract-defining change" sentence listing "the nesting-degradation *implementation*" among 3d's deliverables SHALL be minimally rephrased/annotated (the 3c/3d split narrative stays as history); `docs/specs/index.md`'s harness-adapters description listing "`review` nesting degradation" SHALL be updated. The CLI-branch degradation-instruction injections at the three dispatch sites (`_preamble.md` § CLI-Adapter Dispatch, `fab-continue.md` Review Behavior, `_pipeline.md` Step 2) SHALL be removed with their host paragraphs. `_review.md`'s § "Nesting degradation" subsection SHALL be removed. `stage-models.md`'s § pointer mentioning "`review` nesting degradation" SHALL be reworded.

- **GIVEN** the repo after this change
- **WHEN** grepping for `nesting degradation` / "review is the nesting stage"
- **THEN** no live (non-historical) description of review-stage nesting degradation remains; only the 3c/3d historical framing survives, annotated as superseded

#### R8: SPEC-mirror + aggregate-spec sweep
Every touched `src/kit/skills/*.md` SHALL have its `docs/specs/skills/SPEC-*.md` mirror updated in this change, and every aggregate restatement (found by grepping `inward|outward|nesting|resolves once` repo-wide, plus `parallel dispatch|findings merge|shared review dispatch|two reviewer|both reviewers`) SHALL be swept: `SPEC-_review.md`, `SPEC-_preamble.md`, `SPEC-fab-continue.md`, `SPEC-_pipeline.md`, `SPEC-fab-adopt.md`, `SPEC-_generation.md` (if the stub edit lands), `SPEC-fab-ff.md`, `SPEC-fab-fff.md` (verify), and the aggregate specs `skills.md`, `glossary.md`, `overview.md`, `user-flow.md`, `stage-models.md`, `harness-adapters.md`, `index.md`. Historical/generated artifacts (shipped migrations, dated findings, srad-rationale, `log.md`/`log.seed.md`, past-change `fab/changes/**`) SHALL NOT be swept.

- **GIVEN** a `src/kit/skills/*.md` edit in this change
- **WHEN** review inspects the SPEC-mirror class
- **THEN** every mirror and every aggregate restatement of the two-reviewer/nesting/resolves-once facts is updated to the single-agent shape, and no historical artifact was touched

#### R9: Scaffold comment updated
`src/kit/scaffold/fab/project/code-review.md`'s § Review Tools comment SHALL be updated from "the review-stage outward-reviewer Codex → Claude cascade" to "the review-stage Codex → Claude cascade". No migration is required (§ Review Tools reading semantics are unchanged; stale comments in existing user-owned files are harmless).

- **GIVEN** the scaffold `code-review.md`
- **WHEN** it is read after this change
- **THEN** the comment no longer says "outward-reviewer" and no `src/kit/migrations/` file was added

### Non-Goals

- No Go change (verified in intake: `src/go/` carries no two-reviewer assumption; `fab dispatch`/`fab resolve-agent` are stage-generic). No `_cli-fab.md` / Go-test edits.
- No config/migration surface (`agent.tiers.review`, `providers:`, § Review Tools semantics, `{stage}-result.yaml` schema unchanged).
- Memory prose edits (`docs/memory/**`) are HYDRATE's job, NOT apply's — this plan does not edit memory content beyond leaving the Affected Memory list for hydrate.
- Not editing `.claude/skills/` deployed copies (gitignored, regenerated by `fab sync`).

### Design Decisions

1. **Rename `outward-only` → `diff-only`**: the "outward reviewer" concept is dissolved, so `outward-only` is semantically stale — *Why*: the value now means "the single dispatch judging the diff, minus plan-conformance"; `diff-only` names that accurately; all callers are in-kit so the rename is atomic — *Rejected*: keeping `outward-only` (intake allowed it, but it names a now-nonexistent agent and would confuse future readers).
2. **`_review.md` stays the single review-authority file**: same name, same referenced-by-name pattern — *Why*: matches the `_generation.md` precedent (behavior authoritative in one location); nothing in the intake suggests folding review into `fab-continue.md` — *Rejected*: dissolving the partial into `fab-continue.md`.
3. **Rewrite (not delete) the "Review resolves once" prose**: replace with a one-line note that review is unexceptional — *Why*: leaves a positive statement that review resolves once like every stage, avoiding a silent gap a reader would trip on — *Rejected*: outright deletion (loses the "one resolution" affirmation).

## Tasks

### Phase 1: Canonical skill sources (`src/kit/skills/`)

- [x] T001 <!-- rework: cycle 1 must-fix — § Review Agent Dispatch kept a dispatcher-imperative step ("Dispatch: Via the Agent tool… a single sub-agent") inside a procedure the review block executes end-to-end; followed literally the block spawns a nested sub-agent, contradicting the no-nesting premise (R1). Resolution: make _review.md worker-facing — the dispatched review block IS the single review agent; it reads this file at entry and runs the merged checklists inline itself (diff gathering, cascade, findings); remove all "dispatch via Agent tool" imperatives from the procedure body (the sequencer's dispatch lives in _pipeline.md/fab-continue.md, not here). Also: compress the tasks-all-[x] lean-prompt inconsistency (either precondition-covered or agent work, not both) and add /fab-adopt to the header consumer list. --> Rewrite `src/kit/skills/_review.md` as a single-agent procedure: update frontmatter `description`; rewrite `## Contents`; collapse `## Inward Sub-Agent Dispatch` + `## Outward Sub-Agent Dispatch` + `## Parallel Dispatch` + `## Findings Merge` into a single `## Review Agent Dispatch` (merged checklist: plan-conformance Validation Steps + holistic-diff Focus Areas + Codex→Claude cascade step) with the verbatim framing line; keep `## Review Mode` (rename `outward-only`→`diff-only`) and `## Preconditions`; restate the single unified three-tier findings + deterministic pass/fail rule; delete the `### Nesting degradation` subsection; keep the parsimony four-category table, deletion-candidate `## Deletion Candidates` contract, and `change_type` skip conditions verbatim <!-- R1 R2 R3 R4 R5 R6 R7 -->
- [x] T002 [P] Edit `src/kit/skills/_pipeline.md`: Step 2 (line ~84) — one resolution, one dispatched review block running the single merged procedure; delete the "both reviewer sub-agents (inward + outward) and the merge" language and the sequential-inline/nesting-degradation clause; update the § Behavior per-stage note (line ~62) review sentence; update rework item 4 (line ~105) to the single-agent shape <!-- R1 R5 R7 -->
- [x] T003 <!-- rework: cycle 1 must-fix — under the one-dispatch shape no sequencer-side site owned the merged-prompt contract (change_type carrier, framing line); the retained "When dispatching the review sub-agent, read change_type … and pass it in the prompt" addressed a dispatcher the no-nesting claims abolish. Resolution: Review Behavior = "the dispatched review worker reads _review.md at entry and executes it inline as the single review agent"; the SEQUENCER (fab-continue Normal Flow / _pipeline Step 2 dispatch seam) reads change_type from .status.yaml and passes it in the block dispatch prompt; the framing line lives in _review.md, which the worker reads — no separate carrier needed. --> Edit `src/kit/skills/fab-continue.md`: Review Behavior (lines ~170–174) — the review block executes the single merged procedure inline; delete the "nested reviewers" resolution blockquote and the "CLI-dispatched review worker (nesting degradation)" blockquote; rewrite the `review` dispatch-table row (line ~74) "(once, for both reviewers + merge)"; keep the `change_type`-in-prompt contract <!-- R1 R5 R7 -->
- [x] T004 [P] Edit `src/kit/skills/_preamble.md`: rewrite § Per-Stage Model Resolution "Review resolves once" paragraph (line ~327) to the unexceptional single-resolution shape; re-anchor the § Standard Subagent Context "Nested dispatch" example (line ~299) off the review sub-agent; remove the CLI-branch degradation-instruction injection language from § CLI-Adapter Dispatch (leaving the five-state machine, dispatch-prompt obligations, and block-contract carve-out untouched) <!-- R4 R5 R7 -->
- [x] T005 [P] Edit `src/kit/skills/fab-adopt.md`: Step 3 (lines ~96–100) — reword from "outward-only / no inward sub-agent" to the `diff-only` prompt-variant framing (single dispatch, plan-conformance steps omitted, preconditions skipped); sweep all `outward-only` value references (lines ~96, 100, 103, 125, 146) to `diff-only`; keep zero-findings-passes-best-effort <!-- R6 -->
- [x] T006 [P] Edit `src/kit/skills/_generation.md` (line ~222): reword the Plan-from-Diff acceptance-stub text "…outward review runs in this pipeline." to match the single-agent / `diff-only` semantics <!-- R6 -->
- [x] T007 [P] Edit `src/kit/scaffold/fab/project/code-review.md` (line ~61): "the review-stage outward-reviewer Codex → Claude cascade" → "the review-stage Codex → Claude cascade" <!-- R9 -->

### Phase 2: SPEC mirrors (`docs/specs/skills/`) — constitution-required

- [x] T008 [P] Rewrite `docs/specs/skills/SPEC-_review.md`: Summary (line 10), Review Mode paragraph (line 12, rename value), delete/rewrite the Nesting degradation paragraph (line 14), rewrite the Flow ASCII (lines 24–91) to a single-agent shape, the Validation Steps Inventory heading references, Tools-used "parallel" note (line 130), and Sub-agents section (lines 132–135) to one agent <!-- R1 R4 R5 R6 R7 R8 -->
- [x] T009 [P] Edit `docs/specs/skills/SPEC-_preamble.md`: Summary (line 11) "review resolves once for both reviewers + merge" and the Flow ASCII (line 116) — reword to single-resolution; drop the CLI-branch degradation-instruction mention if present <!-- R5 R7 R8 -->
- [x] T010 <!-- rework: cycle 1 must-fix — the Flow diagram draws a "SUB-AGENT (single): whole review inline" box nested INSIDE the REVIEW STAGE box (two levels) while line 12 of the same file asserts no nested reviewer dispatch; redraw to ONE level matching the settled shape (the dispatched review block IS the single review agent). Also sweep the cycle-1 should-fix: Tools table row "Agent | Review validation sub-agent" → single review sub-agent naming. --> Edit `docs/specs/skills/SPEC-fab-continue.md`: § Single post-intake execution mode nested-reviewer/degradation sentences (line 12), § Per-stage model "once for its own nested reviewer sub-agents" (line 16), Flow ASCII inward/outward boxes + "parallel dispatch" (lines 90, 94, 104–112), the Sub-agents table rows (lines 187–188), and the Review-Behavior blockquote (line 190) — all to the single-agent shape <!-- R1 R5 R7 R8 -->
- [x] T011 [P] Edit `docs/specs/skills/SPEC-_pipeline.md`: the long Summary paragraph (line 5) "review resolves once … both reviewers + merge" + "CLI review worker degrades nesting to sequential-inline", the Flow ASCII (lines 64–66), and the Sub-agents table Review row (line 84) — to single-agent shape <!-- R1 R5 R7 R8 -->
- [x] T012 [P] Edit `docs/specs/skills/SPEC-fab-adopt.md`: `outward-only` → `diff-only` and single-agent framing at lines 12, 17, 59, 62, 65–66, 91, 97 <!-- R6 R8 -->
- [x] T013 [P] Edit `docs/specs/skills/SPEC-fab-ff.md`: lines 18 ("review [inward + outward sub-agents via _review.md…]") and 27 ("dispatches `_review.md`'s inward + outward sub-agents in parallel") → single review sub-agent <!-- R1 R8 -->
- [x] T014 [P] Verify `docs/specs/skills/SPEC-fab-fff.md` for restated review-dispatch facts (inward/outward/parallel/resolves-once); update any found, else record verified-clean <!-- R8 -->

### Phase 3: Aggregate + cross-cutting specs (`docs/specs/`)

- [x] T015 [P] Edit `docs/specs/skills.md`: line 50 (`fab-adopt` helper note "outward-only review"), line 356 ("inward sub-agent inspects…"), line 446 ("Step 3 — Review … `mode: outward-only`"), line 453 ("Outward-only review via the general `mode` parameter") <!-- R1 R6 R8 -->
- [x] T016 [P] Edit `docs/specs/glossary.md`: lines 52 and 116 — `/fab-adopt` and **Adopt** entries "review runs outward-only" / "review (outward-only)" → the `diff-only` single-agent framing <!-- R6 R8 -->
- [x] T017 [P] Edit `docs/specs/overview.md`: stage-table row 3 (line 70) "inward sub-agent inspects…" → single review sub-agent; `/fab-adopt` row (line 97) "review (outward-only)" → `diff-only` <!-- R1 R6 R8 -->
- [x] T018 [P] Edit `docs/specs/user-flow.md`: line 38 ("review (outward-only)") and line 64 (mermaid label "outward-only review") → `diff-only` single-agent framing <!-- R6 R8 -->
- [x] T019 [P] Edit `docs/specs/stage-models.md`: the "review stage resolves once … BOTH reviewer sub-agents (inward + outward) and the merge" paragraph (lines 284–288) → single-agent single-resolution; the § pointer mentioning "`review` nesting degradation" (line 345) → reworded; the deferred-idea bullet "Role-granular keys (`review.inward`, `review.merge`)" (line 421) → drop the role examples or mark obsolete <!-- R5 R7 R8 -->
- [x] T020 [P] Edit `docs/specs/harness-adapters.md`: delete § "Nesting degradation (the `review` stage)" (lines 105–113); minimally rephrase/annotate the § "Skill wiring is NOT part of the contract-defining change" sentence listing "the nesting-degradation *implementation*" (line 146) as superseded history <!-- R7 R8 -->
- [x] T021 [P] Edit `docs/specs/index.md`: harness-adapters description (line 28) listing "`review` nesting degradation" in the shared protocol → remove that clause <!-- R7 R8 -->
- [x] T022 [P] Edit `docs/specs/skills/SPEC-_generation.md` if its Plan-from-Diff stub text restates the "outward review runs in this pipeline" wording — update to match T006; else record verified-clean <!-- R6 R8 -->

### Phase 4: Verification sweep

- [x] T023 Repo-wide grep sweep (`inward|outward|nesting degradation|resolves once|parallel dispatch|findings merge|both reviewers|two reviewer|shared review dispatch`) over `src/kit/` + `docs/specs/`, excluding historical/generated artifacts (`/migrations/`, `/findings/`, `srad-scoring-rationale*`, `log.md`, `log.seed.md`, `fab/changes/**`); confirm every remaining hit is either historical framing or an unrelated context (e.g. `internal-consistency-check` Task-dispatch, `superpowers-comparison`), and confirm the `{stage}-result.yaml` schema block is byte-unchanged <!-- R4 R7 R8 -->

## Execution Order

- T001 is the anchor (canonical `_review.md`); T002–T007 are independent skill edits and may run in parallel with each other after T001's design is settled.
- Phase 2 SPEC mirrors (T008–T014) depend on their Phase 1 counterparts being decided (mirror the final wording).
- Phase 3 (T015–T022) are independent parallel edits.
- T023 runs last (verification of the whole class).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_review.md` dispatches exactly ONE review sub-agent whose prompt carries both the plan-conformance checklist and the holistic-diff focus areas; no `## Parallel Dispatch`, no `## Findings Merge`, no inward/outward sub-agent split remains
- [x] A-002 R2: the verbatim framing line "conformance to plan.md is necessary but not sufficient; also judge the diff on its own merits against the repo." is present in the merged prompt, with no read-prohibition or phase-ordering instruction
- [x] A-003 R3: the Codex→Claude cascade survives as an inline step gated by `code-review.md` § Review Tools (absent = enabled; `false` disables; graceful no-op), with `copilot` untouched
- [x] A-004 R4: the unified three-tier findings + deterministic pass/fail rule are restated; "Findings Merge" is gone as a distinct mechanic; the `{stage}-result.yaml` review schema in `_preamble.md`/`SPEC-_preamble.md` is byte-unchanged
- [x] A-005 R5: exactly one `fab resolve-agent review --alias` resolution remains; "Review resolves once (both reviewers + merge)" prose is rewritten; the nested-reviewer resolution note in `fab-continue.md` is deleted
- [x] A-006 R6: `/fab-adopt` passes `mode: diff-only`; the adoption mode is the single dispatch minus plan-conformance steps with preconditions skipped and zero-findings-passes-best-effort; no in-kit file references `outward-only`
- [x] A-007 R7: the nesting-degradation machinery is deleted (harness-adapters § removed, three CLI-branch injections removed, `_review.md` subsection removed, stage-models/index pointers reworded); only annotated 3c/3d historical framing survives
- [x] A-008 R8: every touched `src/kit/skills/*.md` has its `SPEC-*.md` mirror updated; every aggregate restatement (skills.md, glossary.md, overview.md, user-flow.md, stage-models.md, harness-adapters.md, index.md) is swept; no historical/generated artifact was touched
- [x] A-009 R9: the scaffold `code-review.md` comment no longer says "outward-reviewer"; no `src/kit/migrations/` file was added

### Behavioral Correctness

- [x] A-010 R4: a reader of the reworked `_review.md` + `_pipeline.md` + `fab-continue.md` sees the identical review outcome contract (findings tiers, pass/fail rule, orchestrator-owned preconditions/transitions/rework loop) as before the change

### Scenario Coverage

- [x] A-011 R6: the `## Review Mode` table in `_review.md` lists `full` (default) and `diff-only`, with preconditions checked only in `full`; `SPEC-_review.md`'s mode paragraph matches

### Edge Cases & Error Handling

- [x] A-012 R3: with all reviewer tools disabled/unavailable, the single agent's cascade step returns empty findings (graceful no-op) and review passes best-effort — semantics unchanged from today

### Code Quality

- [x] A-013 Pattern consistency: edits follow surrounding skill/spec prose style (RFC-2119 usage, blockquote conventions, ASCII-diagram layout); only canonical `src/kit/` sources edited — never `.claude/skills/`
- [x] A-014 No unnecessary duplication: `_review.md` remains the single review-dispatch authority; other files reference it by name rather than restating the merged procedure
- [x] A-015 Sibling & Mirror Sweep complete: the whole SPEC-mirror class swept up front (grep-verified in T023), not reactively

### Documentation Accuracy

- [x] A-016 documentation_accuracy: no live (non-historical) file describes two parallel reviewers, a Findings Merge step, "Review resolves once for both reviewers + merge", or review-stage nesting degradation
- [x] A-017 cross_references: every cross-reference to `_review.md` § Parallel Dispatch / § Findings Merge / § Nesting degradation is repointed or removed; no dangling section reference remains

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Memory (`docs/memory/**`) edits are hydrate's job (intake § 7 + Affected Memory); apply does not touch memory content.

## Deletion Candidates

- `docs/memory/pipeline/execution-skills.md` § Review Behavior — the two-reviewer split, Review Mode (`outward-only`) table, per-sub-agent context/validation subsections, and Findings Merge prose describe machinery this change deleted; already assigned to hydrate (plan Non-Goals / intake Affected Memory), listed for completeness, not missed work
- `docs/memory/_shared/context-loading.md:113,131` — the "review sub-agent within `/fab-continue`" nested-dispatch example and the "Review resolves once" (two reviewer sub-agents + merge) paragraph are obsolete under the single-agent shape; hydrate's rewrite targets
- `docs/memory/_shared/configuration.md:171-200` — § `review_tools` (retired) and the § `code-review.md` bullets naming "the outward-reviewer Codex→Claude cascade" / "the outward sub-agent" / "the inward sub-agent's parsimony validation step" now name nonexistent agents; hydrate's rewrite target
- `src/kit/skills/_preamble.md:373` — the schema comment `# review (mirrors "merged prioritized findings + pass/fail verdict")` quotes deleted Findings-Merge outcome language (its source text no longer exists anywhere in the live tree); rewording is blocked by R4's byte-unchanged schema-block constraint, so it needs a follow-up change that relaxes that scope

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Rename the `mode` value `outward-only` → `diff-only` (default proposal from intake Assumption 11), sweeping all in-kit callers atomically | Intake Assumption 11 left the final name to apply and allowed `outward-only`; the "outward reviewer" concept is dissolved so `outward-only` is semantically stale — `diff-only` names the new semantics (single dispatch judging the diff, minus plan-conformance); all callers in-kit (`fab-adopt.md` sole passer) so the rename is atomic and reversible | S:70 R:85 A:80 D:70 |
| 2 | Confident | Rewrite (not delete) the "Review resolves once" prose to a one-line "review resolves once like every stage" note, rather than removing it entirely | Intake Assumption 5 allowed both rewrite and remove; a positive one-line affirmation avoids leaving a silent gap and reads better than an absence | S:75 R:90 A:85 D:75 |
| 3 | Confident | Collapse `_review.md`'s four review sections into one `## Review Agent Dispatch` section (keeping `## Review Mode` + `## Preconditions`), rather than renaming one of the existing sections | Cleanest single-agent shape; the merged prompt carries both checklists in one place, matching intake §1's "one review agent's procedure"; keeps the file the single authority (Assumption 13) | S:65 R:85 A:80 D:70 |
| 4 | Confident | Memory prose (`docs/memory/**`) is left entirely to hydrate; apply edits only `src/kit/` + `docs/specs/` + scaffold | Intake § 7 + Affected Memory + the dispatch prompt's explicit instruction assign memory content edits to hydrate; editing memory at apply would double-write and risk drift | S:80 R:85 A:85 D:80 |

4 assumptions (0 certain, 4 confident, 0 tentative).
