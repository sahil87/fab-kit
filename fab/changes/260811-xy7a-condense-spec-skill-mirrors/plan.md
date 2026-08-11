# Plan: Condense SPEC Skill Mirrors

**Change**: 260811-xy7a-condense-spec-skill-mirrors
**Intake**: `intake.md`

## Requirements

### Specs: SPEC Mirror Condensation

#### R1: Strip all 36 `docs/specs/skills/SPEC-*.md` files to structural quick-reference
Each SPEC file SHALL be reduced to exactly: title (`# {skill-name}`), a header of at most 2–3 sentences (what the skill does — no dated deltas, no change-id narration), the `## Flow` diagram, the Tools table (`### Tools used` or equivalent), and the `### Sub-agents` section. Everything else — `## Summary` dated-delta prose, `## Contents` TOCs, "Source organization" lines, `## Hooks` / bookkeeping-candidates sections — SHALL be deleted. The total tree SHALL land at 10–20% of 3,721 lines (~370–740).

- **GIVEN** a `docs/specs/skills/SPEC-*.md` file
- **WHEN** the strip is applied
- **THEN** it contains only title + ≤2–3-sentence header + Flow + Tools table + Sub-agents
- **AND** the aggregate tree is ~370–740 lines

#### R2: Migrate-don't-drop load-bearing Summary content
Before deleting a `## Summary` section, content that is (a) load-bearing (a behavior/contract claim a reader would need) and (b) not already present in `docs/memory/` SHALL be merged into the relevant memory file as FKF present-truth (topic-keyed, no change-id narration; rationale in four-field `## Design Decisions` entries where applicable). Deletion is the default; migration is the exception.

- **GIVEN** a SPEC file's `## Summary` section
- **WHEN** audited against the existing `docs/memory/` coverage
- **THEN** uncovered load-bearing claims are merged into memory as present truth and everything else is deleted

### Governance: Mirror Rule Narrowing

#### R3: Constitution mirror rule narrowed to structural changes
The Additional Constraints rule "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file" SHALL be narrowed so a mirror update is required only when a change alters a skill's flow, tool usage, or sub-agent structure. The amendment SHALL follow the constitution's conventions: dated HTML-comment annotation (`<!-- 2026-08-11 (260811-xy7a): … -->`), **Last Amended** updated to 2026-08-11, and a minor version bump (1.4.0 → 1.5.0).

- **GIVEN** `fab/project/constitution.md` at version 1.4.0
- **WHEN** the amendment is applied
- **THEN** the rule names the narrowed trigger, carries a dated annotation, version is 1.5.0, Last Amended is 2026-08-11

#### R4: Enforcement surfaces updated to the narrowed rule
`fab/project/code-review.md`'s "SPEC-mirror sync" must-fix rule and `fab/project/code-quality.md`'s § Sibling & Mirror Sweeps (mirror class entry + CLI-change blockquote) plus the "Shipping a skill change without its SPEC mirror" anti-pattern SHALL be rewritten to the narrowed trigger (flow / tool-usage / sub-agent structure changes only).

- **GIVEN** the enforcement files restating the old unconditional rule
- **WHEN** updated
- **THEN** every restatement triggers only on flow/tool-usage/sub-agent structure changes

#### R5: Sweep class updated consistently
The restatements of the mirror rule/shape in `docs/specs/skills.md` (New Skill Checklist item 6), `docs/specs/index.md` (the `skills/` row description), `src/kit/skills/docs-reorg-specs.md` (reserved-paths note, ~lines 29 and 88), `docs/memory/memory-docs/specs-index.md` (§ Per-Skill SPEC Mirrors + reserved-paths Design Decision), and `docs/memory/pipeline/dedupe.md` (the "constitution-required mirror" line) SHALL be rewritten to the narrowed rule and the condensed shape. Reserved-path protection and mechanical `SPEC-{source-filename}.md` naming are unchanged.

- **GIVEN** a repo-wide grep for SPEC-mirror restatements
- **WHEN** the sweep completes
- **THEN** no file outside historical records restates the old unconditional rule

### Memory: Index Regeneration

#### R6: Memory indexes regenerated after memory writes
After any write under `docs/memory/`, `fab memory-index` SHALL be run and its output taken wholesale (never hand-merged).

- **GIVEN** memory files modified by R2/R5
- **WHEN** `fab memory-index` runs
- **THEN** root/domain/sub-domain indexes and logs reflect the new state

### Non-Goals

- Reclassifying the `docs/specs/skills/` row in `docs/specs/index.md` as generated-adjacent — only the row's shape description changes (intake Assumptions row 6).
- Rewriting historical Design-Decision entries that record past changes.
- Any `.go` change — docs/governance only; no test updates required.

## Tasks

### Phase 1: Governance

- [x] T001 Amend the mirror rule in `fab/project/constitution.md`: narrow to flow/tool-usage/sub-agent triggers, bump version 1.4.0→1.5.0, set Last Amended 2026-08-11, append dated annotation <!-- R3 -->
- [x] T002 [P] Rewrite the "SPEC-mirror sync" rule in `fab/project/code-review.md` (Project-Specific Review Rules) to the narrowed trigger <!-- R4 -->
- [x] T003 [P] Rewrite `fab/project/code-quality.md`: the "Shipping a skill change without its SPEC mirror" anti-pattern, the § Sibling & Mirror Sweeps mirror-class entry, and the CLI-change blockquote <!-- R4 -->

### Phase 2: Memory Rule Updates

- [x] T004 Rewrite `docs/memory/memory-docs/specs-index.md` § Per-Skill SPEC Mirrors and the reserved-paths Design Decision to the narrowed rule + condensed shape; refresh `description:` frontmatter <!-- R5 -->
- [x] T005 [P] Reword the "constitution-required mirror" sentence in `docs/memory/pipeline/dedupe.md` <!-- R5 -->

### Phase 3: Strip the 36 SPEC Mirrors

Each strip task includes the R2 migrate-don't-drop audit: scan the deleted Summary/prose for load-bearing claims absent from memory; migrate the rare exception into the matching memory file as present truth.

- [x] T006 Strip partial mirrors `docs/specs/skills/SPEC-_*.md` (9 files: _preamble, _generation, _review, _srad, _pipeline, _intake, _cli-fab, _cli-external, _cli-agents) to title + header + Flow + Tools + Sub-agents <!-- R1 R2 -->
- [x] T007 Strip fab-skill mirrors `docs/specs/skills/SPEC-fab-*.md` to the condensed shape <!-- R1 R2 -->
- [x] T008 Strip remaining mirrors `docs/specs/skills/SPEC-git-*.md`, `SPEC-docs-*.md`, `SPEC-internal-*.md` to the condensed shape <!-- R1 R2 -->

### Phase 4: Sweep & Verify

- [x] T009 Update sweep-class files: `docs/specs/skills.md` New Skill Checklist item 6, `docs/specs/index.md` `skills/` row, `src/kit/skills/docs-reorg-specs.md` reserved-paths note <!-- R5 -->
- [x] T010 Run `fab memory-index` to regenerate memory indexes/logs; take output wholesale <!-- R6 -->
- [x] T011 Verify: `wc -l docs/specs/skills/SPEC-*.md` totals ~370–740 lines; every file matches the condensed shape; repo-wide grep finds no remaining old-rule restatement outside historical records <!-- R1 R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: All 36 `docs/specs/skills/SPEC-*.md` files contain only title, ≤2–3-sentence header, Flow diagram, Tools table, Sub-agents section
- [x] A-002 R1: Total `docs/specs/skills/` tree is 370–740 lines (10–20% of 3,721)
- [x] A-003 R2: No load-bearing behavioral claim from a deleted Summary exists nowhere else (migrated to `docs/memory/` as present truth where uncovered)
- [x] A-004 R3: Constitution mirror rule names the narrowed trigger; version 1.5.0; Last Amended 2026-08-11; dated annotation present
- [x] A-005 R4: `code-review.md` and `code-quality.md` mirror rules trigger only on flow/tool-usage/sub-agent changes
- [x] A-006 R5: All five sweep-class locations restate the narrowed rule / condensed shape; mechanical SPEC naming and reserved-path protection unchanged
- [x] A-007 R6: `fab memory-index` output is committed wholesale; no hand-edited index rows

### Behavioral Correctness

- [x] A-008 R3: A prose-only skill edit no longer triggers a mirror-update obligation anywhere in governance/enforcement docs

### Scenario Coverage

- [x] A-009 R1: `docs/specs/skills/SPEC-fab-discuss.md`-style shape (the exemplar) is the uniform result across all 36 files

### Edge Cases & Error Handling

- [x] A-010 R1: Files carrying `## Hooks`/bookkeeping/`## Contents` sections (e.g. `SPEC-_preamble.md`, `SPEC-fab-ff.md`) have them fully removed
- [x] A-011 R2: Memory files touched by migration remain FKF-conforming (present-truth, `description:` ≤500 chars, change-id-free headings)

### Code Quality

- [x] A-012 Pattern consistency: Amended docs follow existing annotation/section conventions of their files
- [x] A-013 No unnecessary duplication: Rule is stated by owners and pointed to, not restated divergently

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Constitution version bump is minor (1.4.0 → 1.5.0) | Carried from intake row 5: narrowing a normative MUST rule is material, above cosmetic wording | S:70 R:85 A:70 D:60 |
| 2 | Confident | Strip work batched into 3 tasks (partials / fab-* / rest) rather than 36 | Mechanical, same-shape edits; per-file tasks add bookkeeping without changing the work | S:80 R:85 A:85 D:75 |
| 3 | Confident | Existing Flow diagrams and Tools/Sub-agents tables are preserved verbatim where present (heading names kept as-is per file), not redrawn | The intake names the sections to keep; redrawing risks introducing errors into content that already matches the target shape | S:75 R:90 A:80 D:75 |
| 4 | Certain | The `docs-reorg-specs.md` reserved-paths edit is a prose-only skill edit needing no mirror update beyond the strip this change performs | Intake row 9; the strip touches every mirror anyway | S:55 R:85 A:60 D:50 |
| 5 | Confident | Historical dated entries inside memory `## Design Decisions` and log files are not rewritten | Intake row 8: they record past changes, not current rule restatements | S:80 R:80 A:80 D:70 |

5 assumptions (1 certain, 4 confident, 0 tentative).
