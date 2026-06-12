# Plan: Scaffold/Config Truth + SRAD Scoring Coherence

**Change**: 260612-c5tr-scaffold-config-truth-srad-coherence
**Intake**: `intake.md`

## Requirements

### Distribution: Scaffold & Config Surface Truth

#### R1: Scaffold ships no `stage_directives`
`src/kit/scaffold/fab/project/config.yaml` MUST NOT contain the `stage_directives` block (mapping and its descriptive comment). No directive is relocated — the GIVEN/WHEN/THEN directive is redundant with `_generation.md` step 3 and the `[NEEDS CLARIFICATION]` directive is illegal outside intake post-1.10.0.

- **GIVEN** a fresh project scaffolded by `fab init` + `/fab-setup`
- **WHEN** `fab/project/config.yaml` is created from the scaffold
- **THEN** it contains no `stage_directives` key and no zombie `spec:` stage reference
- **AND** all other scaffold keys (`project`, `source_paths`, `true_impact_exclude`, `test_paths` comment, `checklist`, `agent`) are unchanged

#### R2: Removal migration for populated user configs
A new migration `src/kit/migrations/2.1.6-to-2.2.0.md` MUST drop the dead `stage_directives` key (and, defensively, a re-appeared `model_tiers` key) from existing `fab/project/config.yaml` files. It MUST follow the established migration file format (Summary/Pre-check/Changes/Verification), be idempotent, and leave `stage_hooks` and all other keys untouched. The three existing migrations that preserve/relocate `stage_directives` stay untouched.

- **GIVEN** a user project whose `config.yaml` carries a `stage_directives:` block
- **WHEN** `/fab-setup migrations` applies `2.1.6-to-2.2.0.md`
- **THEN** the block and its descriptive comment are removed, every other key survives, and a re-run is a complete no-op

#### R3: fab-setup drops the `stage_directives` editor
`fab-setup.md` MUST NOT offer `stage_directives` as a config section: removed from the Arguments line, the Config Arguments valid values, and the Config Update menu (menu renumbered, prompts adjusted).

- **GIVEN** a user runs `/fab-setup config`
- **WHEN** the section menu is displayed
- **THEN** it lists project / source_paths / checklist / context.md / code-quality.md / code-review.md / Done with consistent numbering, and `stage_directives` is rejected as an unknown section

#### R4: Docs describe the real config surface
The always-load descriptor for `config.yaml` in `_preamble.md` MUST NOT advertise "naming conventions, model tiers". The same residue in `docs/specs/skills.md` (always-load list), `docs/specs/glossary.md` (`config.yaml` entry), and `docs/specs/architecture.md` (config example with `stage_directives` + `model_tiers` blocks) MUST be cleaned to the real surface (identity, source/test paths, true-impact excludes, plan-acceptance categories, agent spawn command).

- **GIVEN** an agent reading the always-load layer description
- **WHEN** it loads `_preamble.md` (or the spec docs)
- **THEN** the `config.yaml` descriptor names only keys that exist and are consumed

#### R5: This repo's project files carry no residue
This repo's `fab/project/config.yaml` MUST NOT contain the relocated `stage_directives` block, and `fab/project/code-review.md` MUST NOT name the removed "revise spec"/"revise tasks" escalation paths (post-1.10.0 vocabulary: "revise plan" / "revise requirements").

- **GIVEN** this repo after apply
- **WHEN** grepping `fab/project/` for `stage_directives` and `revise spec`
- **THEN** there are no hits

#### R6: Scaffold code-review.md escalation vocabulary
`src/kit/scaffold/fab/project/code-review.md`'s Rework Budget escalation line MUST use the live rework-menu vocabulary: escalate to "revise plan" or "revise requirements" (not "revise spec").

- **GIVEN** a new project's `code-review.md` created from the scaffold
- **WHEN** the review sub-agent reads the Rework Budget section
- **THEN** the escalation paths named are exactly the ones `/fab-continue` and `_pipeline.md` offer

#### R7: "Max cycles" knob wired into the pipeline bracket
`_pipeline.md` MUST derive its rework-cycle cap (`{max_cycles}`) from the `Max cycles: {N}` line under `## Rework Budget` in `fab/project/code-review.md` when the file and line exist, defaulting to 3 otherwise. All hard-coded "3 cycles" mentions in `_pipeline.md` and the driver descriptions (`fab-ff.md`, `fab-fff.md`) MUST reference the knob. The escalation threshold (2 consecutive fix-code) stays fixed.

- **GIVEN** a project whose `code-review.md` sets `Max cycles: 2`
- **WHEN** `/fab-ff` enters the auto-rework loop
- **THEN** the loop stops (fail, no reset) after the 2nd failed cycle
- **GIVEN** a project without `code-review.md`
- **WHEN** the loop runs
- **THEN** the cap is 3 (unchanged default behavior)

#### R8: `stage_hooks` documented (kept live, no Go changes)
`_cli-fab.md` MUST document the live `stage_hooks` config surface: YAML shape (`stage_hooks.{stage}.pre/post`), execution (`sh -c` from the repo root, stdout/stderr passthrough, absent = no-op), pre-hook failure blocking `fab status start` (transition not applied), post-hook running after `finish`'s transition is saved, the fact that `finish`'s auto-activation does NOT fire the next stage's pre hook, and the **failing-post-hook re-run trap** (stage already `done`; re-running `finish` errors on done→done — run the hook command by hand or `reset` first). `config.go`/`status.go` stay untouched.

- **GIVEN** a user whose post hook failed after `fab status finish`
- **WHEN** they consult `_cli-fab.md`
- **THEN** the documentation tells them the stage is already `done` and a bare re-run of `finish` will not re-fire the hook

#### R9: `workflow.yaml` retired
`src/kit/schemas/workflow.yaml` MUST be deleted (nothing consumes it; it still describes the retired 7-stage pipeline with `spec`). `docs/specs/user-flow.md`'s "Source of truth" line MUST repoint to the Go state machine (`src/go/fab/internal/status` + `statusfile`).

- **GIVEN** the kit after apply
- **WHEN** looking for the pipeline's source of truth
- **THEN** no `workflow.yaml` exists under `src/kit/`, and `user-flow.md` points at the Go state machine

#### R10: fab-setup repairs
`fab-setup.md` MUST (a) restore the one-line semver-comparison rule at the Migrations Step 1 three-way `local`/`engine` branch (deleted by #393's f080 dedup), (b) give Config Create Mode a `fab_version` fallback — when no existing key is present, stamp the engine version from `$(fab kit-path)/VERSION` so step 1c's "step 1a guarantees" claim is true, and (c) fix the Next Steps Reference lines to match `_preamble.md`'s State Table (`initialized` → `/fab-new`, `/fab-proceed`, `/docs-hydrate-memory`).

- **GIVEN** an agent at Migrations Step 1 with `local=2.9.7`, `engine=2.10.0`
- **WHEN** it picks the output branch
- **THEN** the in-file rule tells it to compare MAJOR/MINOR/PATCH as integers (2.10.0 > 2.9.7), not lexicographically
- **GIVEN** config create mode with no prior `config.yaml`
- **WHEN** the new file is written
- **THEN** it carries `fab_version` stamped from the engine VERSION

### Pipeline: SRAD Scoring Coherence

#### R11: Half-open grade bands
`_srad.md` (and `docs/specs/srad.md`) MUST map composites to grades via half-open thresholds — ≥85 Certain, ≥60 Confident, ≥30 Tentative, else Unresolved — so continuous composites (59.85, 84.5) always grade deterministically.

- **GIVEN** a decision with composite 84.5
- **WHEN** the agent maps it to a grade
- **THEN** it is Confident (≥60, <85) with no band gap

#### R12: One Critical-Rule number
The Critical Rule MUST have a single numeric definition: the `R < 25 AND A < 25` override. The prose restatements (`_srad.md` § Critical Rule, `docs/specs/srad.md` § The Critical Rule) MUST cite `< 25` explicitly instead of implying the 0–39 Low band.

- **GIVEN** a decision with R:30, A:30 (Low band but ≥25)
- **WHEN** the agent applies the Critical Rule
- **THEN** the override does NOT fire — both documents agree

#### R13: Worked-example arithmetic reaches its grades
`_srad.md` Worked Example 3 MUST be re-dimensioned so its composite actually reaches Certain (S raised to Medium: S:50 R:95 A:100 D:100 → 86 ≥ 85), and its tail must not claim Certain decisions go unmentioned (they are recorded in the Assumptions summary). `docs/specs/srad.md` Example 1 rows 2–3 composites MUST be corrected (15.75, 42.75; grades unchanged) and Example 2 row 3 to 94.35.

- **GIVEN** the worked examples
- **WHEN** their composites are recomputed with the 0.25/0.30/0.25/0.20 weights
- **THEN** every printed composite matches the arithmetic and every grade matches its band

#### R14: Autonomy coverage for all six declaring skills
`_srad.md` § Skill-Specific Autonomy Levels MUST cover all six skills that declare `_srad` — via a covering note for `fab-draft` (fab-new's posture, thin delta) and `fab-clarify` (the escape valve itself). Mirror the same coverage in `docs/specs/srad.md`.

- **GIVEN** an agent loading `_srad` from `fab-draft` or `fab-clarify`
- **WHEN** it consults the autonomy table
- **THEN** its own posture is defined (not absent)

#### R15: Omit-when-zero is output-only; plan walk emits `## Assumptions`
`_srad.md`'s omit-when-zero rule MUST be scoped to the **displayed output summary only**: generated artifacts (intake.md, plan.md) ALWAYS carry the `## Assumptions` section, with a `0 assumptions.` footer and no table rows when empty. `_generation.md`'s Plan Generation walk MUST gain an explicit step that emits the `## Assumptions` section it depends on (three grades, Scores required, footer), and the Intake Generation step 4 notes the always-present rule.

- **GIVEN** an apply run that made zero inline assumptions
- **WHEN** `plan.md` is written
- **THEN** the `## Assumptions` section is present with the `0 assumptions.` footer (and the displayed output omits the summary block)
- **GIVEN** an apply run that made graded assumptions
- **WHEN** the walk completes
- **THEN** a numbered step (not an aside) directed their persistence into `## Assumptions`

#### R16: `docs/specs/srad.md` contract aligned to `_srad`
The spec's Assumptions-table contract MUST match the canonical `_srad.md`: Scores column required on every row (not "optional"), column order `# | Grade | Decision | Rationale | Scores`, all four grades recorded in intake artifacts (Certain rows included; plan.md excludes Unresolved), Unresolved rows carry status context, footer `{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved).`, Certain output visibility "Noted in Assumptions summary", and the output-only omit-when-zero rule.

- **GIVEN** an agent generating an intake from the spec's examples
- **WHEN** `fab score` parses the resulting table
- **THEN** every row parses (no missing Scores, no omitted Certain rows deflating counts/cover)

#### R17: Bulk-confirm evaluated before the zero-gaps exit
`fab-clarify.md` MUST evaluate the bulk-confirm trigger before any zero-gaps early exit: Step 1.5 builds the queue without stopping; the "No gaps found — artifact looks solid." exit moves to Step 2's not-triggered branch.

- **GIVEN** a marker-free intake with 5 Confident, 0 Tentative, 0 Unresolved assumptions sitting below the 3.0 gate
- **WHEN** the user runs `/fab-clarify`
- **THEN** bulk confirm fires (instead of the "artifact looks solid" dead-end), and confirmations can raise the score above the gate

#### R18: Bulk-confirm grades by recomputed composite
The bulk-confirm Artifact Update MUST set S → 95, recompute the composite, and assign the grade by threshold — not label rows Certain by fiat. A confirmed row whose recomputed composite stays < 85 remains Confident (Rationale still records the confirmation).

- **GIVEN** a Confident row with S:70 R:50 A:60 D:60 (composite 59.5)
- **WHEN** the user confirms it (S → 95)
- **THEN** the recomputed composite is 65.75 and the row is labeled Confident, not Certain

#### R19: Audit-trail symmetry
The two audit-trail writers in `fab-clarify.md` (Step 2 bulk confirm, Step 5 Q&A) MUST state identical placement/append rules: append to an existing `## Clarifications` section; create it immediately before `## Assumptions` if absent; skip when the session resolved nothing.

- **GIVEN** either path writes an audit trail
- **WHEN** `## Clarifications` does not yet exist
- **THEN** both create it in the same place with the same session-heading convention

#### R20: fab-new output ordering
`fab-new.md`'s Output template MUST place the Assumptions summary as the final content block immediately before the `Next:` line, per `_srad.md`'s SHALL.

- **GIVEN** `/fab-new` completes with assumptions
- **WHEN** the output renders
- **THEN** the order is intake → Confidence → Activated → Branch → Assumptions → `Next:`

### Distribution: Spec Mirrors

#### R21: SPEC mirrors updated for every touched skill
Every touched `src/kit/skills/*.md` MUST have its `docs/specs/skills/SPEC-*.md` mirror updated where mirrored content is affected: `SPEC-_srad.md`, `SPEC-_generation.md`, `SPEC-fab-clarify.md`, `SPEC-fab-new.md`, `SPEC-fab-setup.md`, `SPEC-_pipeline.md`, plus `SPEC-fab-ff.md` and `SPEC-fab-fff.md` (both carried 3-cycle-cap text and were updated for the `{max_cycles}` knob). `SPEC-_preamble.md` verified content-unaffected; `_cli-fab.md` has no mirror.

- **GIVEN** the mirrors after apply
- **WHEN** compared against their skills
- **THEN** no mirror still describes the pre-change behavior (closed bands, optional Scores, 3-cycle hard-coding, stage_directives menu, fiat-Certain bulk confirm)

### Non-Goals

- No Go changes — `stage_hooks` stays live and undisturbed (`config.go`, `status.go`); `stage_directives`/`model_tiers` have no Go readers to remove
- No regeneration of `workflow.yaml` (retired, not rewritten)
- No change to `fab score`'s parser or formula — coherence fixes are documentation-side
- No wiring of the "After 2 consecutive…" escalation threshold (only the Max-cycles knob was resolved for wiring)
- No memory-file edits (hydrate's job)

### Design Decisions

1. **Migration named `2.1.6-to-2.2.0.md`**: FROM = current released version, TO = next minor — *Why*: follows the j6cs precedent (`1.9.7-to-1.10.0.md`); gap-skip discovery covers locals stranded between 1.10.0 and 2.1.6 — *Rejected*: `1.10.0-to-2.2.0.md` wide range (unneeded; `fab upgrade-repo` self-stamps no-op locals forward, and gap-skips handle the rest)
2. **Covering note over new table columns** for fab-draft/fab-clarify autonomy — *Why*: the 5-column table is already wide; both skills are deltas of existing columns — *Rejected*: two more columns (duplicates fab-new's column almost verbatim)
3. **Zero-gaps exit relocated into Step 2's not-triggered branch** — *Why*: smallest reorder that makes bulk-confirm reachable in its primary scenario while preserving the "solid artifact" UX for genuinely-clean intakes — *Rejected*: deleting the early exit (forces empty question loops on clean artifacts)

## Tasks

### Phase 1: Scaffold & Config Surface Truth

- [x] T001 Remove the `stage_directives` block (incl. comment) from `src/kit/scaffold/fab/project/config.yaml` <!-- R1 -->
- [x] T002 [P] Remove the `stage_directives` block from this repo's `fab/project/config.yaml` <!-- R5 -->
- [x] T003 [P] Reword escalation line 44 in `src/kit/scaffold/fab/project/code-review.md` and this repo's `fab/project/code-review.md` to "revise plan" / "revise requirements" <!-- R6, R5 -->
- [x] T004 Create migration `src/kit/migrations/2.1.6-to-2.2.0.md` (drop `stage_directives`, defensively `model_tiers`; idempotent; format per `docs/memory/distribution/migrations.md`) <!-- R2 -->
- [x] T005 Remove `stage_directives` editor surface from `src/kit/skills/fab-setup.md` (Arguments, Config Arguments, menu + renumber) <!-- R3 -->
- [x] T006 fab-setup repairs in `src/kit/skills/fab-setup.md`: semver one-liner at Migrations Step 1.3, `fab_version` create-mode fallback (step 5 + 1c guarantee), Next Steps Reference rewrite <!-- R10 --> <!-- rework: docs/specs/skills.md:102 fab-setup Next-line row omits /fab-proceed — contradicts the repaired fab-setup.md:431; align it -->

- [x] T007 [P] Rewrite the `config.yaml` always-load descriptor in `src/kit/skills/_preamble.md` <!-- R4 --> <!-- rework: the rewritten enumeration omits review_tools — a live behavior-controlling key (consumed by _review.md cascade + git-pr-review); add it -->

- [x] T008 [P] Clean `docs/specs/skills.md` always-load line, `docs/specs/glossary.md` `config.yaml` entry, `docs/specs/architecture.md` config example <!-- R4 --> <!-- rework: (a) add review_tools to the skills.md:64 + glossary config.yaml enumerations (same omission as T007); (b) architecture.md:286 still says config holds "naming conventions, stage configuration" — same dead-surface residue, clean it -->

- [x] T009 Delete `src/kit/schemas/workflow.yaml`; repoint `docs/specs/user-flow.md` source-of-truth line to the Go state machine <!-- R9 -->
- [x] T010 Wire `{max_cycles}` into `src/kit/skills/_pipeline.md` (definition + all cycle-count mentions) and update the cycle-cap prose in `src/kit/skills/fab-ff.md` / `src/kit/skills/fab-fff.md` <!-- R7 --> <!-- rework: docs/specs/skills.md:388,395 still hard-code "3 cycles"/"3-cycle cap" — align with the {max_cycles} knob -->

- [x] T011 Document `stage_hooks` in `src/kit/skills/_cli-fab.md` (new subsection under `fab status`: shape, semantics, auto-activation caveat, re-run trap) <!-- R8 -->

### Phase 2: SRAD Scoring Coherence

- [x] T012 `src/kit/skills/_srad.md`: half-open thresholds, Critical-Rule prose cites `< 25`, Worked Example 3 re-dimensioned, autonomy covering note, omit-when-zero scoped to output (artifact always-present + `0 assumptions.` footer) <!-- R11, R12, R13, R14, R15 --> <!-- rework: Worked Example 1's parenthetical "(Critical Rule applies: low R + low A)" at _srad.md:65 is the last loose Critical-Rule phrasing — cite R<25 AND A<25; optional while there: add scores to Example 2 to disambiguate the Ex2-Confident vs Ex3-Certain identical qualitative profiles -->

- [x] T013 `src/kit/skills/_generation.md`: insert explicit `## Assumptions` emission step in the Plan Generation walk (renumber write step); add always-present note to Intake step 4 <!-- R15 -->
- [x] T014 `docs/specs/srad.md`: align table contract (Scores required, column order, four grades, footer, Certain visibility), half-open bands, Critical-Rule phrasing, Example 1 composites 15.75/42.75 + Example 2 row 3 94.35, autonomy section coverage <!-- R16, R11, R12, R13, R14 -->
- [x] T015 `src/kit/skills/fab-clarify.md`: move zero-gaps exit after bulk-confirm trigger (Step 1.5/Step 2), grade-by-recomputed-composite in Artifact Update, unify audit-trail rules (Step 2/Step 5) <!-- R17, R18, R19 --> <!-- rework: MUST-FIX — fab-clarify.md:118 inlines the composite formula (0.25*S + ...) and grade thresholds, duplicating _srad.md:30 (the drift class this batch fixes). Replace with a reference: recompute per _srad § SRAD Scoring, grade by its half-open thresholds — restate NO weights/thresholds -->

- [x] T016 [P] `src/kit/skills/fab-new.md`: reorder Output template (Assumptions block last, before `Next:`) <!-- R20 -->

### Phase 3: Mirrors & Verification

- [x] T017 Update SPEC mirrors: `SPEC-_srad.md`, `SPEC-_generation.md`, `SPEC-fab-clarify.md`, `SPEC-fab-new.md`, `SPEC-fab-setup.md`, `SPEC-_pipeline.md`, `SPEC-fab-ff.md`, `SPEC-fab-fff.md` <!-- R21 --> <!-- rework: re-check SPEC-fab-clarify.md for the same inlined composite formula/thresholds the T015 must-fix removes; mirror the de-inlined reference form -->

- [x] T018 Verification sweep: yq-parse both edited config.yaml files; grep live surfaces for residual `stage_directives`/`model_tiers`/`workflow.yaml`/`revise spec`; verify migration range non-overlap; run `go test ./...` baseline (no Go edits); check repointed link target exists <!-- R1, R2, R4, R5, R9 -->

## Execution Order

- T001 before T004 (migration's Summary cites the scaffold state)
- T012 before T013/T014 (canonical `_srad` text settles before its dependents mirror it)
- T017 after all Phase 1–2 tasks (mirrors reflect final skill text)
- T018 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: Scaffold `config.yaml` contains no `stage_directives` key or related comment; all other keys intact
- [x] A-002 R2: `src/kit/migrations/2.1.6-to-2.2.0.md` exists, follows the Summary/Pre-check/Changes/Verification format, and drops `stage_directives` (+ defensive `model_tiers`)
- [x] A-003 R3: `fab-setup.md` no longer lists `stage_directives` anywhere; menu numbering is consistent (7 items incl. Done)
- [x] A-004 R4: `_preamble.md` config.yaml descriptor names the real surface; skills.md/glossary.md/architecture.md carry no `model_tiers`/`stage_directives` references
- [x] A-005 R7: `_pipeline.md` defines `{max_cycles}` (code-review.md Rework Budget, default 3) and no hard-coded cycle count remains in the loop/stop/error text
- [x] A-006 R8: `_cli-fab.md` documents `stage_hooks` incl. pre-blocks-start, post-after-save, auto-activation caveat, and the re-run trap
- [x] A-007 R9: `src/kit/schemas/workflow.yaml` is deleted and `user-flow.md` points at the Go state machine
- [x] A-008 R10: fab-setup carries the semver rule, the `fab_version` fallback, and State-Table-conformant Next Steps
- [x] A-009 R15: `_generation.md` Plan Generation has a numbered `## Assumptions` emission step before the write step
- [x] A-010 R16: `docs/specs/srad.md` requires Scores on every row and shows all four grades recorded at intake

### Behavioral Correctness

- [x] A-011 R11: Both `_srad.md` and `srad.md` grade 84.5 → Confident and 59.85 → Tentative under the new half-open thresholds
- [x] A-012 R12: Both documents state the Critical Rule as `R < 25 AND A < 25` with no 0–39-band reading
- [x] A-013 R17: In `fab-clarify.md`, a zero-gap artifact reaches the bulk-confirm trigger evaluation before any "artifact looks solid" stop
- [x] A-014 R18: The bulk-confirm Artifact Update assigns grade from the recomputed composite (S→95), with sub-85 rows staying Confident
- [x] A-015 R20: `fab-new.md` Output places the Assumptions block immediately before `Next:`

### Removal Verification

- [x] A-016 R1: No `stage_directives` hits remain in `src/kit/scaffold/`, `src/kit/skills/` (live prose), or this repo's `fab/project/`
- [x] A-017 R9: No live reference to `src/kit/schemas/workflow.yaml` remains outside historical changelogs/findings (memory updates deferred to hydrate)
- [x] A-018 R5: Neither `code-review.md` (scaffold or repo) names "revise spec"/"revise tasks"

### Scenario Coverage

- [x] A-019 R13: Worked Example 3's printed dimensions produce a composite ≥ 85; srad.md Example 1 rows compute 12.5/15.75/42.75 and Example 2 row 3 computes 94.35
- [x] A-020 R19: Both audit-trail writers state the same create-before-`## Assumptions` placement and skip-when-empty rule

### Edge Cases & Error Handling

- [x] A-021 R2: Migration pre-check handles missing `config.yaml` and already-absent keys as printed no-op skips; re-run is a complete no-op
- [x] A-022 R15: Zero-assumption artifacts keep the `## Assumptions` section with a `0 assumptions.` footer (output summary omitted instead)
- [x] A-023 R7: With no `code-review.md` (or no `Max cycles:` line), the bracket's cap defaults to 3 — behavior identical to pre-change

### Code Quality

- [x] A-024 Pattern consistency: new migration matches the format/voice of `1.9.7-to-1.10.0.md`; skill edits match surrounding prose conventions
- [x] A-025 No unnecessary duplication: `{max_cycles}` defined once in `_pipeline.md`; bulk-confirm rules single-homed in `fab-clarify.md`; semver rule restored as one line, not a resurrected section

### Documentation Accuracy

- [x] A-026: Every statement added to `_cli-fab.md` about `stage_hooks` is verifiable against `config.go`/`status.go`/`hooks.go` behavior

### Cross References

- [x] A-027: `user-flow.md`'s new source-of-truth target path exists; SPEC mirrors and skills agree on the new band/knob/ordering contracts

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Line refs in the intake were vs `1431a9c3`; each was re-verified against file content before editing (drift noted in apply output where found)

## Deletion Candidates

<!-- re-evaluated at re-review (cycle 2): consumers re-verified against the working tree -->

- `src/benchmark/fixtures/workflow.yaml` — frozen copy of the retired 7-stage schema (its header still claims "Single source of truth for the Fab workflow"). Nothing consumes it: `bench.sh` reads only `fixtures/status.yaml` (bench.sh:8,40) and no `statusman-*` implementation references workflow.yaml at all (cycle 1's claim that they parse it was inaccurate — the candidate stands on stronger grounds: zero consumers). With `src/kit/schemas/workflow.yaml` deleted by R9, this fixture is the last workflow.yaml anywhere in the repo
- `docs/memory/pipeline/schemas.md` § workflow.yaml sections (lines ~10–166 of 200) — still call `$(fab kit-path)/schemas/workflow.yaml` "the single source of truth"; made redundant by R9 (hydrate owns the rewrite — listed for completeness, do not hand-delete before hydrate)

## Assumptions

<!-- SCORING SOURCE NOTE: as of 1.10.0, `fab score` reads intake.md only — this
     ## Assumptions section is the apply-agent's record of graded decisions made
     while co-generating ## Requirements (under-specified points resolved inline),
     NOT a scoring source. Three grades only (Certain/Confident/Tentative) —
     Unresolved is intake-only; apply decides and records, it never leaves a
     decision Unresolved. The Scores column is required for every row. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Migration file named `2.1.6-to-2.2.0.md` (FROM = current release, TO = next minor) | Follows the j6cs `1.9.7-to-1.10.0.md` precedent; gap-skip discovery + upgrade-repo self-stamp cover stranded locals; exact next release number is the release author's call | S:80 R:70 A:85 D:80 |
| 2 | Confident | New migration also defensively drops `model_tiers` | Intake says both keys are "removed everywhere they appear"; the 0.22.0 migration already dropped it for most projects, so this is an idempotent no-op safety net | S:70 R:85 A:85 D:75 |
| 3 | Confident | Same-residue spec docs (`architecture.md` config example, `glossary.md` config.yaml entry, `skills.md` always-load line) cleaned although not enumerated in the intake's Impact list | Intake mandates removal everywhere; leaving stale spec docs would contradict the change's own truth goal | S:75 R:90 A:85 D:80 |
| 4 | Confident | Autonomy coverage via a covering note for `fab-draft`/`fab-clarify` instead of new table columns | Intake explicitly offers "columns (or a covering note)"; both skills are deltas of existing columns and the table is already wide | S:85 R:95 A:85 D:65 |
| 5 | Confident | Worked Example 3 re-dimensioned by raising S to Medium (S:50 R:95 A:100 D:100 → 86) keeping the Certain teaching point; its "no mention" tail corrected to match the grade table's output visibility | Intake offers "raise S or change the expected grade"; the example's purpose is teaching config-determinism → Certain, so the grade is preserved | S:75 R:90 A:80 D:70 |
| 6 | Certain | `{max_cycles}` = integer from `Max cycles: {N}` under `## Rework Budget`, read at bracket entry; default 3 when file/section/line absent; escalation threshold stays hard-coded | Intake resolution row 6 specifies exactly this wiring; the 2-consecutive knob was not part of the resolution | S:90 R:85 A:95 D:90 |
| 7 | Certain | Zero-gaps exit implemented as Step 2's not-triggered branch (Step 1.5 never stops) | Direct consequence of the intake's "evaluate the bulk-confirm trigger before the zero-gaps exit"; preserves the solid-artifact UX | S:85 R:90 A:90 D:75 |
| 8 | Confident | Mirror obligation = update mirrors whose content is affected; eight mirrors updated (incl. `SPEC-fab-ff.md`/`SPEC-fab-fff.md`, which carried 3-cycle-cap text); `SPEC-_preamble.md` verified unaffected; `_cli-fab.md` has no mirror to update | No SPEC-_cli-fab.md exists in docs/specs/skills/ (CLI partials are unmirrored by convention); verified by listing the directory | S:70 R:90 A:80 D:70 |
| 9 | Certain | This repo's config residue removed by direct edit, not by running the new migration here | Intake offers "direct edit or via the migration"; direct edit keeps the worktree deterministic | S:90 R:95 A:95 D:85 |
| 10 | Confident | `srad.md` Example 2 row 3 composite corrected to 94.35 alongside the in-scope Example 1 fixes | Same arithmetic-accuracy class as the mandated fixes; grade unaffected (≥85) | S:65 R:95 A:90 D:80 |

10 assumptions (3 certain, 7 confident, 0 tentative).
