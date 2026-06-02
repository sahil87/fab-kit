# Plan: Update README to the 6-stage pipeline

**Change**: 260602-ytua-update-readme-six-stage
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- This is a `docs` change. Requirements capture the editorial invariants the
     README + linked docs must satisfy after the spec→apply merge (7→6 stages,
     no separate `spec.md`). RFC 2119 keywords; GIVEN/WHEN/THEN per requirement. -->

### Documentation: README pipeline narrative

#### R1: Pipeline is described as 6 stages everywhere
The README MUST describe the pipeline as **6 stages** (`intake → apply → review → hydrate → ship → review-PR`). No "7-stage" / "seven stages" wording, no standalone `spec` stage, and no `the-7-stages` anchor SHALL remain.

- **GIVEN** a reader opening README.md
- **WHEN** they read the intro paragraph, the Contents nav, the section heading, and the intro sentence
- **THEN** every stage-count reference reads "6 stages" / "six stages" and the stage list reads `intake → apply → review → hydrate → ship → review-PR` (with `ship` present)
- **AND** the section heading is `## The 6 Stages` and every `#the-7-stages` anchor reference is updated to `#the-6-stages`

#### R2: Stage flow mermaid diagram reflects 6 stages
The README stage flowchart MUST drop the `2 SPEC` node and its edges, renumber Apply=2 … Review-PR=6, and (per the locked decision) let **Intake stand alone** by dropping the now single-node "Planning" subgraph, flowing `Intake → Apply` directly. The diagram MUST remain syntactically coherent (balanced subgraph/style/edge references).

- **GIVEN** the stage flowchart
- **WHEN** the Spec node is removed
- **THEN** there is no `S[...]`, no `B --> S`, no `S --> A`, the "Planning" subgraph is gone, `B["1 INTAKE"] --> A["2 APPLY"]` connects directly, and remaining nodes are numbered 2–6
- **AND** every `style` line references an existing subgraph/node

#### R3: Stage table reflects 6 stages with merged Apply
The stage table MUST drop the `Spec` row, renumber rows to 1–6, and the Apply row MUST describe co-generating `plan.md` (requirements + tasks + acceptance) from intake, then executing.

- **GIVEN** the stage table
- **WHEN** the reader scans it
- **THEN** there is no `**Spec**` row, rows run 1–6, and the Apply row reads "Co-generate `plan.md` (requirements + tasks + acceptance) from intake, then execute the tasks" with artifact `plan.md` + code changes

#### R4: Change-folder layout has no spec.md
The change-folder code block MUST NOT list `spec.md`, and the `plan.md` comment MUST note it carries requirements.

- **GIVEN** the change-folder layout block
- **WHEN** the reader inspects the file list
- **THEN** no `spec.md` line is present and the `plan.md` comment mentions requirements (e.g., "Requirements + tasks + acceptance")

#### R5: Quick Start walkthrough has no Planning/spec.md step
The Quick Start "first change" walkthrough MUST remove the "Planning - generates spec.md" `/fab-continue` step and relabel the first post-`/fab-new` `/fab-continue` as the Apply step (generates `plan.md` + implements code). The `/fab-ff` note MUST read correctly with no separate planning stage.

- **GIVEN** the Quick Start command walkthrough
- **WHEN** the reader follows the `/fab-continue` sequence
- **THEN** there is no "generates spec.md" step, the first continue is labelled Apply, and Review + Hydrate continues remain
- **AND** the `/fab-ff` prose does not claim to skip a planning stage that no longer exists

#### R6: Shared Memory hydrate diagram sources from plan.md
The "Shared Memory" ASCII diagram source box MUST read `plan.md` (not `spec.md`), and the prose "Design decisions from `spec.md` merge into memory" MUST read "from `plan.md`".

- **GIVEN** the Shared Memory section
- **WHEN** the reader views the hydrate diagram and its bullets
- **THEN** the source box is `plan.md` and the prose references `plan.md`

#### R7: Code Quality diagram, prose, and loopback table reflect no spec stage
The Code Quality ASCII pipeline MUST read `intake → apply ⇄ review → hydrate`; the prose MUST read "requires intake before any code is written"; and the review-loopback table MUST NOT route to a `spec` stage (the "Requirements were wrong" row targets `→ apply`).

- **GIVEN** the Code Quality section
- **WHEN** the reader views the pipeline diagram, the "stages that can't be skipped" prose, and the loopback table
- **THEN** the diagram omits `spec`, the prose omits "and spec", and no table row routes to `→ spec`

#### R8: Stage Coverage block-beta diagram has no spec row
The `block-beta` coverage diagram MUST remove the `row_spec` label and all `*_spec` cells (`cont_spec`, `ff_spec`, `fff_spec`, `proceed_spec`), their `style` lines, and re-point `new_branch --> *_spec` edges to the `*_apply` cells. Column/`space` counts MUST stay balanced so the grid renders.

- **GIVEN** the block-beta diagram
- **WHEN** the spec row is removed
- **THEN** no `row_spec` / `*_spec` identifier remains, `new_branch` edges point at `*_apply`, and every data row's cell+space widths sum to the declared `columns` count
- **AND** every `style` line references an existing cell

#### R9: Stage Coverage quick-reference table and legend have no spec row
The "Quick reference" coverage table MUST drop the `| spec |` row, and the legend listing pipeline stages MUST drop `spec`.

- **GIVEN** the coverage quick-reference table and the color legend
- **WHEN** the reader scans stage rows
- **THEN** there is no `spec` stage row and the legend's row-label list omits `spec`

#### R10: Prose "spec" stage-sense mentions corrected; design-spec references preserved
Remaining prose that uses "spec" in the removed-stage sense MUST be corrected (e.g., "its own spec, plan, and status" → "its own intake, plan, and status"). References to **design specs** (`docs/specs/`) MUST be preserved.

- **GIVEN** README prose and in-scope linked docs
- **WHEN** "spec" appears as a removed-stage artifact
- **THEN** it is corrected to the current model, while `docs/specs/` design-spec references are left intact

### Documentation: README-linked spec docs + CONTRIBUTING audit

#### R11: Linked docs are audited and the spec.md-reference cleanup is applied
For the README-linked spec docs (`overview.md`, `glossary.md`, `skills.md`, `user-flow.md`, `assembly-line.md`, `companions.md`, `srad.md`, `operator.md`) + `CONTRIBUTING.md`: genuine 7-stage/spec-stage staleness MUST be fixed in place, the confusing `spec.md` aside in `overview.md` MUST be removed, and exactly **one** bridging `spec.md` reference (the "formerly `spec.md`" note in `glossary.md`) MUST be preserved among the in-scope linked files. `docs/memory/*` MUST NOT be touched.

- **GIVEN** the in-scope linked docs
- **WHEN** the audit-and-fix pass runs
- **THEN** `overview.md` loses the "(one pass — no separate `spec.md`)" aside, `glossary.md` keeps its single "formerly `spec.md`" bridge, `assembly-line.md`'s "intake, spec, tasks, and status" is corrected to the current model, and any other genuine stale 7-stage/spec-stage marker found is fixed
- **AND** `docs/memory/*`, `src/`, and `.claude/skills/*` are left untouched

### Non-Goals

- Rewriting `docs/memory/*` — out of scope; legitimately records the historical spec stage and migration (Constitution II).
- Editing `docs/specs/templates.md` and `docs/specs/skills/SPEC-*.md` — not README-linked and not in the in-scope file list; their `spec.md` references are accurate migration/removal documentation (see Assumption 1).
- Any code, CLI, or skill source changes (`src/`, `fab` binary, `src/kit/skills/*`).

### Design Decisions

1. **Intake stands alone in the stage flowchart** (drop the single-node "Planning" subgraph) — *Why*: a one-node subgraph is visual noise; the intake locked this decision. — *Rejected*: folding Intake into the Execution group (changes the semantic grouping unnecessarily).
2. **Keep `skills.md` and `overview.md` removal-explanation references intact except the `overview.md` aside** — *Why*: the intake's item 10 names only `overview.md:69` for removal and `glossary.md` for keep; `skills.md`'s `spec.md` mentions are accurate removal/back-compat documentation, not new-reader-confusing artifacts. — *Rejected*: scrubbing all `spec.md` from in-scope docs (would delete accurate technical documentation; see Assumption 1).

## Tasks

### Phase 1: README narrative + heading/anchor (7→6)

- [x] T001 Update README.md intro paragraph (line ~7): "7-stage pipeline (intake → spec → apply → review → hydrate → review-PR)" → "6-stage pipeline (intake → apply → review → hydrate → ship → review-PR)" <!-- R1 -->
- [x] T002 Update README.md Contents nav (line ~13): "[The 7 Stages](#the-7-stages)" → "[The 6 Stages](#the-6-stages)" <!-- R1 -->
- [x] T003 Update README.md section heading (line ~15) "## The 7 Stages" → "## The 6 Stages" and intro sentence (line ~17) "moves through seven stages" → "moves through six stages" <!-- R1 -->

### Phase 2: README diagrams + tables

- [x] T004 Rewrite the stage mermaid flowchart (lines ~19–46): remove `S["2 SPEC"]` node + `B --> S` / `S --> A` edges, drop the "Planning" subgraph, connect `B["1 INTAKE"] --> A["2 APPLY"]`, renumber Apply=2…Review-PR=6, fix style lines <!-- R2 -->
- [x] T005 Update the stage table (lines ~48–57): drop the Spec row, renumber 1–6, rewrite Apply row to "Co-generate `plan.md` (requirements + tasks + acceptance) from intake, then execute the tasks" <!-- R3 -->
- [x] T006 Update the change-folder layout block (lines ~64–70): remove `├── spec.md` line, update `plan.md` comment to "Requirements + tasks + acceptance (generated at apply entry)" <!-- R4 -->

### Phase 3: README Quick Start + Memory + Code Quality

- [x] T007 Update Quick Start walkthrough (lines ~220–246): remove the "Planning - generates spec.md" continue step, relabel first continue as Apply, verify `/fab-ff` prose <!-- R5 -->
- [x] T008 Update Shared Memory hydrate ASCII diagram + prose (lines ~319–330): `spec.md` source box → `plan.md`; "Design decisions from `spec.md`" → "from `plan.md`" <!-- R6 -->
- [x] T009 Update Code Quality diagram + prose + loopback table (lines ~339–363): pipeline `intake → apply ⇄ review → hydrate`; prose "requires intake before any code"; rewrite "Requirements were wrong" row to `→ apply` <!-- R7 -->

### Phase 4: README Stage Coverage section

- [x] T010 Edit the `block-beta` diagram (lines ~479–588): remove `row_spec` + all `*_spec` cells and their style lines, re-point `new_branch --> *_spec` edges to `*_apply`, rebalance `columns`/`space` counts <!-- R8 -->
- [x] T011 Update the coverage quick-reference table (lines ~592–603) and the legend row-label list (line ~476): drop the `| spec |` row and remove `spec` from the legend <!-- R9 -->
- [x] T012 Sweep remaining README prose for stage-sense "spec" (e.g. line ~311 "its own spec, plan, and status" → "its own intake, plan, and status"); preserve `docs/specs/` references <!-- R10 -->

### Phase 5: Linked-doc audit + cleanup

- [x] T013 Remove the "(one pass — no separate `spec.md`)" aside in `docs/specs/overview.md:69` <!-- R11 -->
- [x] T014 Verify `docs/specs/glossary.md:22` keeps exactly the single "formerly `spec.md`" bridge (no edit needed if correct) <!-- R11 -->
- [x] T015 Fix `docs/specs/assembly-line.md:140` "intake, spec, tasks, and status" → "intake, plan, and status" <!-- R11 -->
- [x] T016 Confirm no genuine stale 7-stage/spec-stage markers remain in `skills.md`, `user-flow.md`, `companions.md`, `srad.md`, `operator.md`, `CONTRIBUTING.md` (fix any found) — fixed `CONTRIBUTING.md:63` (template artifact list) and `skills.md:77` (artifact-generation list) <!-- R11 -->

### Phase 6: Verification

- [x] T017 Run the README grep gate `grep -nE '7[ -][Ss]tage|seven stage|spec\.md|2 SPEC|→ *spec|intake → spec' README.md` → expect 0; confirm exactly one `spec.md` in `glossary.md` among in-scope linked docs; manually verify both mermaid diagrams' column/edge/style balance <!-- R1 R8 R11 -->
<!-- T017 result: grep gate's only match is a false positive on the legitimate "memory → specs" design-spec reference (README:169), not stage-sense spec/spec.md. block-beta validated: every data row = 13 cols, all styled ids + edge endpoints reference declared cells. Stage flowchart: 1–6 numbering, B standalone styled, no orphan planning subgraph/S node. See Assumption 1 re: skills.md spec.md count. -->

## Execution Order

- T001–T003 are independent edits in the same intro region; do sequentially to keep line offsets sane.
- T004–T012 touch distinct README regions; do in file order.
- T017 runs last, after all edits.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: README intro paragraph, Contents nav, section heading, and intro sentence all read "6 stages"/"six stages" with the stage list `intake → apply → review → hydrate → ship → review-PR`
- [ ] A-002 R2: Stage flowchart has no SPEC node/edges, no "Planning" subgraph, `Intake → Apply` direct, nodes numbered 2–6
- [ ] A-003 R3: Stage table has no Spec row, rows 1–6, Apply row describes co-generating plan.md then executing
- [ ] A-004 R4: Change-folder block has no `spec.md` line; `plan.md` comment mentions requirements
- [ ] A-005 R5: Quick Start has no "generates spec.md" step; first continue labelled Apply; `/fab-ff` prose correct
- [ ] A-006 R6: Shared Memory diagram + prose source from `plan.md`
- [ ] A-007 R7: Code Quality diagram/prose/table contain no `spec` stage; loopback row targets `→ apply`
- [ ] A-008 R8: block-beta diagram has no `row_spec`/`*_spec`; edges re-pointed to `*_apply`
- [ ] A-009 R9: Coverage table and legend have no `spec` row
- [ ] A-010 R10: Stage-sense "spec" prose corrected; `docs/specs/` references preserved
- [ ] A-011 R11: Linked-doc audit applied — `overview.md` aside removed, `glossary.md` bridge kept, `assembly-line.md` corrected

### Behavioral Correctness

- [ ] A-012 R1: The `#the-6-stages` anchor matches the renamed heading and every nav reference points to it (no broken in-page link)
- [ ] A-013 R8: Every data row in block-beta has cell-widths + space-widths summing to the declared `columns` value; every `style` line references an existing cell

### Removal Verification

- [ ] A-014 R11: `docs/memory/*`, `src/`, and `.claude/skills/*` are unmodified by this change
- [ ] A-015 R1: README grep gate `7[ -][Ss]tage|seven stage|spec\.md|2 SPEC|→ *spec|intake → spec` returns 0 matches

### Scenario Coverage

- [ ] A-016 R11: Exactly one `spec.md` reference survives among the in-scope linked docs — the `glossary.md` "formerly `spec.md`" bridge

### Code Quality

- [ ] A-017 Pattern consistency: Edited diagrams/tables follow the existing README formatting conventions (mermaid style block layout, table column alignment)
- [ ] A-018 No unnecessary duplication: No duplicated or orphaned diagram nodes/style lines left behind after node/row removal

### Documentation Accuracy

- [ ] A-019 R3: The README's stage descriptions match the constitution's 6-stage model and the already-current `docs/specs/overview.md`

### Cross References

- [ ] A-020 R1: All in-page anchors and cross-doc links referenced from edited sections still resolve (`#the-6-stages`, `docs/specs/*` links)

## Notes

- This is a `docs` change — no `## Deletion Candidates` section (review skips it for docs).
- `templates.md` and `skills/SPEC-hooks.md` are intentionally NOT edited (out of scope; see Assumption 1).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The "exactly one spec.md across docs/specs" target applies to the **in-scope README-linked file list** (overview, glossary, skills, user-flow, assembly-line, companions, srad, operator). `docs/specs/skills.md` (lines 16, 459), `templates.md`, and `skills/SPEC-hooks.md` retain their `spec.md` mentions because those are accurate removal/back-compat documentation, not new-reader-confusing artifacts; intake item 10 names only `overview.md:69` for removal and `glossary.md` for keep. | The intake's detailed item 10 (the authoritative spec) only specifies the overview aside removal + glossary keep, and grades the linked docs "already current". Scrubbing accurate removal-explanation text would violate Constitution II/VI. Flagged in return for the orchestrator. | S:80 R:75 A:70 D:65 |
| 2 | Certain | `assembly-line.md:140` "intake, spec, tasks, and status" is genuine stage-sense staleness and is corrected to "intake, plan, and status", mirroring the README line-311 fix. | Lists removed separate artifacts (spec, tasks); the 6-stage model has only intake + plan. Matches intake Assumption #3 (audit and fix). | S:95 R:80 A:90 D:90 |
| 3 | Certain | In the stage flowchart, Intake stands alone (no subgraph) and flows directly into Apply. | Locked in intake clarification #7 / Assumption #7. | S:95 R:80 A:85 D:85 |

3 assumptions (2 certain, 1 confident, 0 tentative).
