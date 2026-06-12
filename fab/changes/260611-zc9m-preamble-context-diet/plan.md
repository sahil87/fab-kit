# Plan: Preamble Context Diet — Skills Review Batch 3/4

**Change**: 260611-zc9m-preamble-context-diet
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- All skill edits target canonical sources in src/kit/skills/ — never .claude/skills/
     deployed copies. Findings line numbers are vs commit ae79e04c; edits are located by
     content. Memory file updates (Affected Memory) are hydrate's job, not apply's. -->

### Preamble: SRAD Extraction (f003)

#### R1: SRAD content moves to a new `_srad` helper
The `## SRAD Autonomy Framework` block of `src/kit/skills/_preamble.md` (SRAD Scoring table + aggregation, Confidence Grades, Critical Rule, Skill-Specific Autonomy Levels, Worked Examples, Artifact Markers, Assumptions Summary Block) SHALL move verbatim (except R3's example compression) into a new `src/kit/skills/_srad.md` with standard internal-helper frontmatter (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`), mirroring `_generation.md`'s header style. A ~3-line pointer SHALL remain in `_preamble.md` (what SRAD is, that planning skills load `_srad` via `helpers:`, where it lives).

- **GIVEN** a non-planning skill (e.g., `fab-status`, `git-pr`) loads `_preamble.md`
- **WHEN** it reads the preamble end to end
- **THEN** it encounters only the 3-line SRAD pointer, not the framework body
- **AND** the full framework text exists, semantically unchanged, in `_srad.md`

#### R2: `_srad` joins the helper model and the 6 planning skills declare it
`_preamble.md` § Skill Helper Declaration **Allowed values** SHALL gain `_srad`. The six planning skills — `fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify` — SHALL declare `_srad` in their frontmatter `helpers:` list (`fab-clarify` currently has no `helpers:` key and gains one).

- **GIVEN** the post-change skill sources
- **WHEN** `grep -l '^helpers:.*_srad' src/kit/skills/*.md` runs
- **THEN** exactly the 6 planning skills match
- **AND** the preamble's allowed-values line lists `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`

#### R3: Worked Examples compressed; stale SRAD-location pointers updated
Worked Example 1's full scoring table SHALL be compressed to the one-liner style of Examples 2/3 (inside `_srad.md`). `internal-skill-optimize.md`'s pointers stating SRAD/confidence content lives in `_preamble.md` SHALL be updated, and its `_*.md` partial enumerations SHALL include `_srad.md` so the new helper is reference context, never an optimization target.

- **GIVEN** `internal-skill-optimize.md` post-change
- **WHEN** an agent follows its Analysis table or Optimization Rules
- **THEN** SRAD re-explanations are referenced to `_srad.md` (not `_preamble.md`) and `_srad.md` appears in every partial enumeration (Arguments, Pre-flight, Constraints)

### Preamble: Confidence Scoring Diet (f042, f043)

#### R4: Scoring internals move to `_cli-fab` § fab score
The `.status.yaml` schema block, the score formula (+ `expected_min` explanation), and the Template subsection SHALL move from `_preamble.md` § Confidence Scoring into `_cli-fab.md` § fab score (extended). The preamble SHALL keep only **Gate Threshold** (single flat-3.0 intake gate, `--check-gate`) and **Invocation** (who scores, when).

- **GIVEN** the post-change `_preamble.md`
- **WHEN** its Confidence Scoring section is read
- **THEN** no formula, schema yaml, or template internals appear — only Gate Threshold and Invocation
- **AND** `_cli-fab.md` § fab score carries the schema, formula, and template content

#### R5: Bulk Confirm becomes a one-sentence pointer
The preamble's § Bulk Confirm subsection SHALL be replaced by one sentence pointing at `fab-clarify.md` (Step 2, Suggest Mode) as the sole authority, removing the duplicated trigger condition, upgrade semantics, and internal step numbering.

- **GIVEN** the post-change `_preamble.md`
- **WHEN** `grep -c 'confident >= 3' src/kit/skills/_preamble.md` runs
- **THEN** it returns 0 (the trigger lives only in `fab-clarify.md`)

### Preamble: Dormant and Misplaced Sections (f041, f040)

#### R6: [AUTO-MODE] Skill Invocation Protocol moves to `fab-clarify.md`
The `## Skill Invocation Protocol` section SHALL move from `_preamble.md` into `fab-clarify.md` (its sole referencer), leaving a 2-line pointer in the preamble. `fab-clarify`'s Auto Mode is retained — zero behavior change (user decision: move, not delete). The live § Subagent Dispatch mention of `[AUTO-MODE]` SHALL be fixed to reference the new location.

- **GIVEN** the post-change sources
- **WHEN** an orchestrator needs the `[AUTO-MODE]` contract
- **THEN** the full protocol (prefix, placement, detection, transitivity) is defined in `fab-clarify.md`
- **AND** `_preamble.md` § Subagent Dispatch points to it instead of restating it

#### R7: Operator Spawning Rules move to `_cli-external`
The preamble's § Operator Spawning Rules (known-change vs new-change worktree/branch strategy) SHALL move into `_cli-external.md`'s `wt` section. `fab-operator.md` §6 remains the normative step-by-step spawn procedure. While merging, `_cli-external.md` SHALL keep ONE repo-targeting note (the `wt` section's) and drop the duplicate `fab spawn-command --repo` rule at its tmux `new-window` bullet.

- **GIVEN** the post-change sources
- **WHEN** `grep -rn 'Operator Spawning Rules' src/kit/skills/` runs
- **THEN** the heading exists only in `_cli-external.md`
- **AND** the `fab spawn-command --repo <target-repo>` / "never the operator's own config.yaml" rule appears exactly once in `_cli-external.md`

### Preamble: Descriptive Contract (f001, f117)

#### R8: Always-load contract becomes descriptive; Next:-line MUST scoped; fab-switch reconciled
`_preamble.md` §1 SHALL state that a skill's own Context Loading section overrides the always-load default ("unless the skill's own Context Loading section says otherwise"). The Next Steps Convention MUST SHALL apply unless the skill's own Output or Key Properties section defines a different ending (the skill file wins — same basis as the §1 context-loading override; reworded per review so definition and examples agree, since `/git-pr`/`/git-pr-review` do advance pipeline state yet declare their own endings). The fab-switch contradiction SHALL be reconciled in favor of the skill file: `fab-switch` requires no always-load files (preamble's "loads only config.yaml" claim is dropped; fab-switch joins the exception list).

- **GIVEN** a self-exempting skill (e.g., `docs-reorg-specs`, `git-branch`) and the post-change preamble
- **WHEN** both files are read together
- **THEN** the skill's narrower Context Loading no longer contradicts the preamble contract
- **AND** `fab-discuss`/`fab-operator`/`git-pr`-style endings no longer violate a universal Next:-line MUST

#### R9: fab-operator's context loading trimmed
`fab-operator.md` §2 Context Loading SHALL load only `config.yaml`, `constitution.md`, and `context.md` (optional). `fab-operator` SHALL be added to the preamble §1 exception list. Deliberate behavior change, verifier-endorsed (code-quality, code-review, and both doc indexes are used nowhere in the operator).

- **GIVEN** an operator session starting up (or resuming after `/clear`)
- **WHEN** §2 Context Loading executes
- **THEN** it reads 3 project files, not 7
- **AND** the preamble §1 exception list names `fab-operator` with its 3-file load

### fab-continue: Stage-Conditional Helpers (f122)

#### R10: fab-continue loads `_generation`/`_review` per stage; helper semantics extended
`fab-continue.md` SHALL remove `_generation` and `_review` from its frontmatter `helpers:` (keeping `_srad` per R2) and instead carry explicit read instructions at the point of use: read `.claude/skills/_generation/SKILL.md` at apply entry when `plan.md` needs generating AND on the intake-`active` regeneration path (intake generation also lives in `_generation.md`); read `.claude/skills/_review/SKILL.md` when entering Review Behavior. `_preamble.md` § Skill Helper Declaration SHALL be extended to permit stage-conditional in-body loading so the frontmatter contract stays honest. `fab-ff`/`fab-fff` keep `[_generation, _review]` unchanged (finding f074 REFUTED — their orchestrator-level rework loop edits plan.md sections directly).

- **GIVEN** a `/fab-continue` invocation at hydrate, ship, or review-pr (or an apply-resume with `plan.md` present)
- **WHEN** the skill loads its context
- **THEN** neither `_generation.md` nor `_review.md` is loaded
- **GIVEN** a `/fab-continue` invocation at apply entry with no `plan.md` (or intake `active` with no `intake.md`)
- **WHEN** generation begins
- **THEN** the skill body instructs reading `_generation` first

### Pointer Consolidation (f046)

#### R11: fab-proceed and fab-discuss reference the preamble instead of restating it
`fab-proceed.md`'s verbatim Standard Subagent Context 5-file list SHALL be replaced with a pointer to `_preamble.md` § Standard Subagent Context (the pattern `_review.md` already uses). `fab-discuss.md`'s verbatim 7-file always-load list SHALL be replaced with "Load the always-load layer per `_preamble.md` §1", keeping only the do-not-run-preflight / no-change-artifacts deltas.

- **GIVEN** the post-change `fab-proceed.md` and `fab-discuss.md`
- **WHEN** the canonical preamble lists change in the future
- **THEN** neither skill carries a divergent copy — both resolve through the preamble pointer

### Governance, Mirrors, Validation

#### R12: Constitution gains a dated explanatory comment
`fab/project/constitution.md` SHALL gain a dated HTML comment (j6cs precedent style) noting the helper-model extension (`_srad` helper added; stage-conditional loading permitted). No new normative MUST rule. The existing `_cli-fab.md` hard-coding at the CLI constraint is not violated.

- **GIVEN** the post-change constitution
- **WHEN** its Governance section is read
- **THEN** a `<!-- 2026-06-11 (260611-zc9m): ... -->` comment explains the helper-model extension

#### R13: SPEC mirrors and spec docs stay consistent
Every touched skill's `docs/specs/skills/SPEC-*.md` mirror SHALL be updated; a new `SPEC-_srad.md` SHALL be created (constitution: skill changes MUST update mirrors). `docs/specs/skills.md` (helper allowed-values/mapping, Context Loading Convention, Next Steps Convention scope) SHALL be updated. `docs/specs/glossary.md`'s `[AUTO-MODE]` entry SHALL reference the new protocol location. `docs/specs/srad.md` SHALL be checked for consistency (it legitimately carries the formula as design intent; only stale "lives in _preamble" location claims need fixing).

- **GIVEN** the post-change `docs/specs/` tree
- **WHEN** mirrors are compared against their skills
- **THEN** no mirror describes the pre-change content layout (SRAD in preamble, helpers without `_srad`, operator 7-file load, fab-continue unconditional helpers)

#### R14: Deployment and measurement validated
`fab sync` SHALL deploy `_srad` to `.claude/skills/_srad/SKILL.md` with no Go changes (sync auto-deploys any new `.md`). The f004 `wc -c` measurement (preamble bytes; per-skill totals = skill body + `_preamble` + declared helpers + 13,122B always-load layer) SHALL be re-run and before/after recorded in this plan's `## Notes` for the PR description.

- **GIVEN** the post-change sources
- **WHEN** `fab sync` runs
- **THEN** `.claude/skills/_srad/SKILL.md` exists and matches `src/kit/skills/_srad.md`
- **AND** `## Notes` carries the before/after byte table

### Non-Goals

- No deletion of fab-clarify's Auto Mode (user decided: move [AUTO-MODE], retain Auto Mode)
- No change to `fab-ff`/`fab-fff` helper declarations beyond adding `_srad` (f074 refuted)
- No Go code changes (sync.go auto-deploys new skills)
- No prose compression of the preamble beyond the specified relocations (content moves, it doesn't disappear)
- No SPEC files for `_cli-fab`/`_cli-external` (they have none today — finding f048, out of this batch's scope)
- No memory-file (`docs/memory/`) edits at apply — Affected Memory is hydrate's responsibility

### Design Decisions

1. **Scope the Next:-line MUST rather than enumerate exemptions**: scoping to pipeline-state skills is self-maintaining; an exemption list goes stale with every new skill — *Rejected*: explicit exempt-skill list.
2. **fab-switch fully exempt**: its own file says config is not required and it loads no layer files; the skill file wins per the descriptive-contract direction — *Rejected*: keeping the preamble's "loads only config.yaml" claim.
3. **`helpers: [_srad]` stays unconditional for fab-continue**: apply records graded SRAD assumptions, and intake-stage backward-compat work also needs grading; conditional loading is reserved for the two heavyweight helpers — *Rejected*: making `_srad` stage-conditional too.

## Tasks

### Phase 1: SRAD extraction (f003)

- [x] T001 Create `src/kit/skills/_srad.md` — internal-helper frontmatter mirroring `_generation.md`; move the full `## SRAD Autonomy Framework` body from `_preamble.md` (scoring, grades, Critical Rule, autonomy levels, examples, markers, summary block), compressing Worked Example 1 to one-liner style <!-- R1, R3 --> <!-- rework: nice-to-have fixed in passing — _srad.md Artifact Markers table "Placed by: All planning skills (fab-new, fab-continue, fab-ff)" omits fab-draft/fab-fff/fab-clarify; list all six planning skills -->
- [x] T002 Replace the SRAD section in `src/kit/skills/_preamble.md` with a ~3-line pointer (what SRAD is, planning skills load `_srad` via `helpers:`, content location) <!-- R1 -->
- [x] T003 In `src/kit/skills/_preamble.md` § Skill Helper Declaration: add `_srad` to Allowed values and add stage-conditional loading semantics (in-body point-of-use reads as an alternative to frontmatter pre-loads) <!-- R2, R10 --> <!-- rework: _preamble.md:99-101 helpers: YAML example still shows name: fab-continue / helpers: [_generation, _review] — the exact configuration this change removed; switch the example to a still-accurate skill (e.g., fab-ff with [_generation, _review, _srad]) -->
- [x] T004 [P] Add `_srad` to `helpers:` frontmatter of `src/kit/skills/fab-new.md`, `fab-draft.md`, `fab-ff.md`, `fab-fff.md`; add `helpers: [_srad]` to `fab-clarify.md` (no helpers key today); set `fab-continue.md` to `helpers: [_srad]` (removal of `_generation`/`_review` per T011) <!-- R2, R10 -->
- [x] T005 [P] Update `src/kit/skills/internal-skill-optimize.md`: SRAD/confidence pointers now name `_srad.md`; add `_srad.md` to all `_*.md` partial enumerations (Arguments, Pre-flight, Constraints) <!-- R3 -->

### Phase 2: Other preamble relocations (f042, f043, f041, f040)

- [x] T006 Move Schema/Formula/Template subsections from `src/kit/skills/_preamble.md` § Confidence Scoring into `src/kit/skills/_cli-fab.md` § fab score (extended); keep Gate Threshold + Invocation in the preamble <!-- R4 -->
- [x] T007 Replace `src/kit/skills/_preamble.md` § Bulk Confirm with the one-sentence pointer to `fab-clarify.md` Step 2 <!-- R5 -->
- [x] T008 Move `## Skill Invocation Protocol` from `src/kit/skills/_preamble.md` into `src/kit/skills/fab-clarify.md` (new section, Auto Mode retained); leave 2-line pointer in the preamble; fix the § Subagent Dispatch `[AUTO-MODE]` mention; update fab-clarify's own internal cross-reference <!-- R6 -->
- [x] T009 Move § Operator Spawning Rules from `src/kit/skills/_preamble.md` into `src/kit/skills/_cli-external.md`'s wt section; drop the duplicate repo-targeting rule at `_cli-external.md`'s tmux `new-window` bullet <!-- R7 -->

### Phase 3: Contract edits (f001, f117, f122, f046)

- [x] T010 Rewrite `src/kit/skills/_preamble.md` §1 heading/intro as descriptive ("unless the skill's own Context Loading section says otherwise"); exception list gains `fab-operator` (3-file load) and `fab-switch` (none — skill file wins); scope the Next Steps Convention MUST to pipeline-state skills <!-- R8, R9 --> <!-- rework: _preamble.md:248 Next:-exemption examples contradict the definition — "/git-* skills" are exempted as non-pipeline-state, yet /git-pr advances ship and /git-pr-review runs review-pr transitions; reword the exemption basis so definition and examples agree (mirror the fix at docs/specs/skills.md:94) -->
- [x] T011 Trim `src/kit/skills/fab-operator.md` §2 Context Loading to `config.yaml` + `constitution.md` + `context.md` <!-- R9 -->
- [x] T012 Rework `src/kit/skills/fab-continue.md`: per-stage read instructions for `_generation` (apply entry plan-generation path AND intake-active regeneration path) and `_review` (Review Behavior entry) <!-- R10 -->
- [x] T013 [P] Replace restated context lists with preamble pointers in `src/kit/skills/fab-proceed.md` (Standard Subagent Context) and `src/kit/skills/fab-discuss.md` (always-load layer) <!-- R11 -->

### Phase 4: Governance, mirrors, validation

- [x] T014 [P] Add dated explanatory comment to `fab/project/constitution.md` Governance section (260611-zc9m, helper-model extension, no new MUST rule) <!-- R12 -->
- [x] T015 Update SPEC mirrors in `docs/specs/skills/`: SPEC-_preamble.md (subsection inventory, flow tree), create SPEC-_srad.md, SPEC-fab-continue.md (helpers + per-stage loads), SPEC-fab-clarify.md (protocol moved in, helpers), SPEC-fab-operator.md (3-file context), SPEC-fab-new.md, SPEC-fab-draft.md, SPEC-fab-ff.md, SPEC-fab-fff.md (helpers gain `_srad`), SPEC-fab-discuss.md, SPEC-fab-proceed.md (pointer form), SPEC-fab-switch.md (exception reconciliation), SPEC-internal-skill-optimize.md (`_srad` in partial list) <!-- R13 --> <!-- rework: MUST-FIX — SPEC-_generation.md:22 still said "Append ## Assumptions per _preamble.md SRAD framework" (source _generation.md now points at _srad.md; constitution requires the mirror update). Also SPEC-fab-operator.md:150 stale citation "per _preamble.md § Naming Conventions" → now _cli-external.md § Operator Spawning Rules -->
- [x] T016 Update `docs/specs/skills.md`: helper allowed values + current mapping table, Context Loading Convention (descriptive contract + new exceptions), Next Steps Convention scope <!-- R13 --> <!-- rework: skills.md:122 "Adding a New Skill" checklist enumerated only 4 helper values, missing _srad (inconsistent with line 24 of same file); skills.md:34-36 helpers: YAML example still showed fab-continue [_generation, _review] (a removed configuration); skills.md:94 Next:-exemption wording mirrored the preamble contradiction (see T010 rework) -->
- [x] T017 [P] Update `docs/specs/glossary.md` `[AUTO-MODE]` entry (protocol now defined in fab-clarify.md); check `docs/specs/srad.md` for stale location claims (formula content itself stays — it is design intent) <!-- R13 -->
- [x] T018 Run `fab sync`; verify `.claude/skills/_srad/SKILL.md` deploys and planning skills resolve it; run scoped `go test` only if any Go package was touched (none expected) <!-- R14 -->
- [x] T019 Re-run the f004 `wc -c` measurement (preamble + per-skill totals with 13,122B always-load layer); record before/after in `## Notes` <!-- R14 -->

## Execution Order

- T001 blocks T002 (content must land in `_srad.md` before the preamble cut)
- T002/T003 block T019 (measurement needs final preamble)
- T004 depends on T012's frontmatter decision for fab-continue (single edit: `helpers: [_srad]`)
- T015–T017 depend on Phases 1–3 (mirrors describe final state)
- T018 after all skill-source edits; T019 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_srad.md` exists with internal-helper frontmatter and carries the complete SRAD framework (scoring table, aggregation, grades, Critical Rule, autonomy levels, examples, markers, Assumptions Summary block); `_preamble.md` retains only a ~3-line pointer
- [x] A-002 R2: `_srad` is in the preamble's allowed `helpers:` values, and exactly fab-new, fab-draft, fab-continue, fab-ff, fab-fff, fab-clarify declare it
- [x] A-003 R3: Worked Example 1 is one-liner style; `internal-skill-optimize.md` references `_srad.md` and lists it among untouchable partials
- [x] A-004 R4: Preamble Confidence Scoring contains only Gate Threshold + Invocation; `_cli-fab.md` § fab score carries schema, formula, and template content
- [x] A-005 R5: Bulk Confirm in the preamble is a single pointer sentence; the trigger condition appears only in `fab-clarify.md`
- [x] A-006 R6: The [AUTO-MODE] protocol is fully defined in `fab-clarify.md`; the preamble carries a 2-line pointer; § Subagent Dispatch references the new location
- [x] A-007 R7: Operator Spawning Rules live in `_cli-external.md`'s wt section; the `fab spawn-command --repo` rule appears exactly once in `_cli-external.md`; fab-operator §6 untouched as normative procedure
- [x] A-008 R8: Preamble §1 is descriptive with skill-file override; the Next:-line MUST carries a skill-file-declared ending opt-out; no preamble claim that fab-switch loads config.yaml remains
- [x] A-009 R9: fab-operator §2 loads exactly config/constitution/context; preamble §1 exception list names fab-operator
- [x] A-010 R10: fab-continue frontmatter is `helpers: [_srad]`; in-body read instructions cover `_generation` at apply entry + intake-active regeneration and `_review` at Review Behavior entry; preamble helper semantics permit stage-conditional loading; fab-ff/fab-fff keep `[_generation, _review]` (+ `_srad`)
- [x] A-011 R11: fab-proceed and fab-discuss carry preamble pointers, not restated file lists; fab-discuss keeps its no-preflight/no-artifacts deltas

### Behavioral Correctness

- [x] A-012 R1: Zero semantic loss — every SRAD rule (Critical Rule override, weighted mean, grade thresholds, marker placement, summary-block rules incl. plan.md-excludes-Unresolved) survives verbatim or equivalently in `_srad.md`
- [x] A-013 R6: fab-clarify Auto Mode behavior unchanged (mode detection via [AUTO-MODE] prefix still fully specified)
- [x] A-014 R9: Operator behavior change is loading-only — no §1 principle, safety-model, or spawn-procedure text altered by the trim

### Removal Verification

- [x] A-015 R1: No SRAD framework body text (scoring table, worked examples, marker table, summary-block format) remains in `_preamble.md`
- [x] A-016 R4: No score formula, schema yaml, or template subsection remains in `_preamble.md`
- [x] A-017 R7: The known-change/new-change spawning subsections are gone from `_preamble.md` § Naming Conventions

### Scenario Coverage

- [x] A-018 R14: `fab sync` deploys `_srad` to `.claude/skills/_srad/SKILL.md` (verified by running sync and diffing against the source)
- [x] A-019 R14: Before/after byte measurement recorded in `## Notes` (preamble bytes + per-skill totals per the f004 method)

### Edge Cases & Error Handling

- [x] A-020 R10: The rare intake-`active` regeneration path (fab-continue dispatch row) explicitly instructs reading `_generation` before generating the intake
- [x] A-021 R8: Skills with no Context Loading section still default to the full always-load layer (override is opt-in, not opt-out-by-silence)

### Code Quality

- [x] A-022 Pattern consistency: `_srad.md` frontmatter and header style match `_generation.md`/`_review.md`; pointer sentences match the `_review.md:35` reference pattern
- [x] A-023 No unnecessary duplication: no moved content remains duplicated between `_preamble.md` and its new home (grep-verified for trigger conditions, formula lines, spawn rules, protocol text)

### Documentation Accuracy

- [x] A-024 R13: All listed SPEC mirrors describe the post-change layout; SPEC-_srad.md exists; `docs/specs/skills.md` helper mapping matches the actual frontmatter of all skills <!-- review rework met: SPEC-_generation.md now points at _srad.md; skills.md New Skill Checklist item 3 lists all 5 helper values; SPEC-fab-operator.md spawn citation repointed to _cli-external.md § Operator Spawning Rules -->
- [x] A-025 R13: glossary [AUTO-MODE] entry names fab-clarify.md as the protocol home; srad.md carries no stale "defined in _preamble" location claim

### Cross References

- [x] A-026 R13: Every cross-reference into the moved sections (`_preamble.md > Skill Invocation Protocol`, SRAD pointers, bulk-confirm references, spawning-rule references) in `src/kit/skills/*.md` resolves to the new locations
- [x] A-027 R12: Constitution comment present, dated, references 260611-zc9m, and adds no new normative MUST rule

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

### f004 Byte Measurement — Before/After (for the PR description)

Method (finding f004): per-invocation total = skill body + `_preamble.md` + declared helpers + 13,122B always-load layer (7 project files). All sizes `wc -c` on `src/kit/skills/*.md` at this branch.

**Before** (this branch, pre-change): `_preamble` 32,790 · `_generation` 9,486 · `_review` 10,604 · `_cli-fab` 33,144 · `_cli-external` 6,502. Per-skill totals: fab-new 68,162 · fab-draft 64,927 · fab-continue 82,514 · fab-ff 74,452 · fab-fff 77,023 · fab-clarify 54,667 · fab-operator 136,967 · fab-proceed 58,590 · fab-discuss 48,933 · fab-switch 50,570 · fab-help 47,133 · fab-archive 53,413 · git-pr 59,079 · git-pr-review 61,543 · git-branch 50,647 · docs-hydrate-specs 49,264 · docs-reorg-memory 56,318 · docs-reorg-specs 48,984 · internal-retrospect 47,815 · internal-skill-optimize 50,607 · internal-consistency-check 48,348.

**After** (final, post-review-rework): `_preamble` 22,313 (**−10,477, −32.0%**) · `_generation` 9,546 (+60) · `_review` 10,604 (unchanged) · `_cli-fab` 34,811 (+1,667) · `_cli-external` 7,299 (+797) · `_srad` 8,152 (new, opt-in).

| Skill | Before (B) | After (B) | Δ |
|-------|-----------|-----------|---|
| fab-new | 68,162 | 65,923 | −2,239 |
| fab-draft | 64,927 | 62,688 | −2,239 |
| fab-continue | 82,514 | 60,788¹ | −21,726 |
| fab-ff | 74,452 | 72,194 | −2,258 |
| fab-fff | 77,023 | 74,765 | −2,258 |
| fab-clarify | 54,667 | 53,961² | −706 |
| fab-operator | 136,967 | 129,247³ | −7,720 |
| fab-proceed | 58,590 | 47,938 | −10,652 |
| fab-discuss | 48,933 | 38,100 | −10,833 |
| fab-switch | 50,570 | 40,093⁴ | −10,477 |
| fab-help | 47,133 | 36,656 | −10,477 |
| fab-archive | 53,413 | 42,936 | −10,477 |
| git-pr | 59,079 | 48,602 | −10,477 |
| git-pr-review | 61,543 | 51,066 | −10,477 |
| git-branch | 50,647 | 40,170 | −10,477 |
| docs-hydrate-specs | 49,264 | 38,787 | −10,477 |
| docs-reorg-memory | 56,318 | 45,841 | −10,477 |
| docs-reorg-specs | 48,984 | 38,507 | −10,477 |
| internal-retrospect | 47,815 | 37,338 | −10,477 |
| internal-skill-optimize | 50,607 | 40,242 | −10,365 |
| internal-consistency-check | 48,348 | 37,871 | −10,477 |

¹ fab-continue is now stage-dependent: 60,788B at hydrate/ship/review-pr and apply-resumes (`helpers: [_srad]` only); +9,546 (`_generation`) at apply plan-generation entry → 70,334; +10,604 (`_review`) at review → 71,392. Even the worst case is −11,122 vs before.
² fab-clarify absorbed the [AUTO-MODE] protocol (+1,619 body) and opted into `_srad` (+8,152) — its small net saving is by design (it is THE SRAD-consuming skill); every non-planning skill no longer pays for either.
³ fab-operator additionally drops 4 of 7 always-load files (3-file subset = 6,933B vs 13,122B in this constant-AL method): effective startup/`/clear` load ≈ 123,058B (−13,909 vs before).
⁴ fab-switch per the reconciled contract loads no always-load project files; the constant-AL method overstates its total in both columns.

Headline: the always-loaded preamble dropped 32,790 → 22,313B (**−32.0%**, −10.5KB on every one of the ~24 preamble-loading skill invocations). The ~14 non-planning skills save the full 10.5KB; fab-continue saves 11.1–21.7KB depending on stage; fab-proceed/fab-discuss save ~10.7KB; the relocated content (SRAD 8.2KB, scoring internals 1.7KB, spawning rules 0.8KB) is now paid only by the skills that consume it.

## Deletion Candidates

- None — this refactor relocates content and deletes each superseded copy in the same edit (SRAD block, scoring internals, bulk-confirm duplicate, [AUTO-MODE] protocol, spawning rules, duplicate `--repo` rule, restated context lists); no existing content was left redundant or unused. The only dormant pair (`fab-clarify.md` Auto Mode + Skill Invocation Protocol, no live caller since 1.10.0) was explicitly retained by user decision at intake (Assumption #6) and is not a candidate.

## Assumptions

<!-- SCORING SOURCE NOTE: as of 1.10.0, `fab score` reads intake.md only — this
     ## Assumptions section is the apply-agent's record of graded decisions made
     while co-generating ## Requirements (under-specified points resolved inline),
     NOT a scoring source. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Next:-line MUST exemption based on a skill-file-declared ending (Output/Key Properties opt-out), not a "pipeline-state" definition or exempt-skill list | Review rework: the pipeline-state basis contradicted its own examples (/git-pr advances ship); skill-file-wins matches the §1 contract and is self-maintaining | S:70 R:80 A:75 D:70 |
| 2 | Confident | fab-switch becomes a full §1 exception (loads no always-load files) rather than "loads only config.yaml" | Skill-file-wins per intake assumption #8; fab-switch.md's own Context Loading and Key Properties say config/constitution not required | S:75 R:85 A:80 D:75 |
| 3 | Certain | fab-continue keeps `_srad` as an unconditional frontmatter helper (only `_generation`/`_review` go stage-conditional) | Intake mandates `_srad` in all 6 planning skills' `helpers:`; apply always grades assumptions, so the helper is used on the dominant path | S:85 R:90 A:85 D:85 |
| 4 | Certain | Create `SPEC-_srad.md` for the new helper (style of `SPEC-_generation.md`) | Constitution: skill-file changes MUST update corresponding SPEC mirrors; a new skill file needs a new mirror. `_cli-fab`/`_cli-external` have no SPECs today (f048) — out of scope | S:85 R:90 A:90 D:90 |
| 5 | Confident | `_srad.md` added to internal-skill-optimize's partial enumerations (beyond the two SRAD-location pointers the intake names) | Without it, batch mode would treat the new helper as an optimization target — contradicts the partials-are-reference rule the file states three times | S:70 R:85 A:85 D:80 |
| 6 | Confident | Before/after measurement uses current-branch sizes (preamble 32,790B) as "before", not f004's ae79e04c snapshot (32,260B) | Honest like-for-like on this branch; f004 numbers predate merged batches; method (not snapshot) is what the intake mandates re-running | S:70 R:90 A:85 D:75 |
| 7 | Certain | fab-discuss keeps its 2-line ending exemption; trimming its context list does not change its 7-file load | f046 is pointer-consolidation only; fab-discuss's purpose IS surfacing the always-load layer | S:85 R:90 A:90 D:90 |

7 assumptions (3 certain, 4 confident, 0 tentative).
