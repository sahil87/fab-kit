# Plan: Scrub numbered-constitution citations from shipped skills

**Change**: 260616-u4du-scrub-numbered-constitution-citations
**Intake**: `intake.md`

## Requirements

### Documentation Accuracy: Self-contained principle citations in deployable skill/spec text

#### R1: Numbered constitution citations in shipped skills are replaced by named, attributed principles
Every `Constitution <Numeral>` citation in `src/kit/skills/*.md` (the canonical, `fab sync`-deployed skill source) SHALL be rewritten so the unstable Roman-numeral pointer is removed and the principle is named inline and attributed as a fab-kit design principle. The named concept (idempotency / provider neutrality / Pure Prompt Play / specs-are-human-curated) MUST remain present at each former citation site; only the numeral pointer is dropped.

- **GIVEN** a shipped skill cites a fab-kit principle by Roman numeral (e.g. `(idempotency, Constitution III)`)
- **WHEN** the sweep rewrites it
- **THEN** the numeral is gone and the named principle + fab-kit-design-principle attribution remain (e.g. `(idempotency — a fab-kit design principle)`)
- **AND** `grep -rnE "Constitution [IVX]+" src/kit/skills/` returns zero matches

#### R2: Numbered constitution citations in spec mirrors are replaced in lockstep with the skills
Every `Constitution <Numeral>` citation in `docs/specs/skills/SPEC-*.md` SHALL be rewritten by the same naming/attribution rule as R1. This keeps the spec mirrors in sync with their skill bodies, honoring the constitution's "skill changes MUST update the corresponding SPEC mirror" rule.

- **GIVEN** a spec mirror cites a fab-kit principle by Roman numeral
- **WHEN** the sweep rewrites it
- **THEN** the named principle + attribution remain and the numeral is gone
- **AND** `grep -rnE "Constitution [IVX]+" docs/specs/skills/` returns zero matches

#### R3: Multi-occurrence lines and mid-sentence citations are edited per-occurrence, preserving prose
Lines carrying multiple citations (`src/kit/skills/docs-reorg-specs.md` line 20 [×3], `docs/specs/skills/SPEC-docs-reorg-specs.md` line 9 [×3]) and the citation embedded mid-clause in `docs/specs/skills/SPEC-fab-continue.md` line 9 SHALL be edited per occurrence so surrounding prose, block-quotes, and the `per **Constitution VI**` bold-emphasis phrasing read naturally after the edit. A blunt global find-replace MUST NOT be used where it would mangle dense prose.

- **GIVEN** a dense block-quote line with three citations and bold-emphasized `per **Constitution VI**`
- **WHEN** the sweep rewrites it
- **THEN** each occurrence is replaced individually, the sentence still parses, and no stray numeral or broken bold-markup remains

#### R4: Numbered citations in other deployable kit content are scrubbed in lockstep with their dev-source
Every `Constitution <Numeral>` citation in remaining **deployable** kit content SHALL be rewritten by the same naming/attribution rule as R1: `src/kit/reference/fkf.md` (deployed to the kit cache) and `src/kit/migrations/2.4.2-to-2.5.0.md`. The dev-repo single-sources from which deployed content is extracted — `docs/specs/fkf.md` and `docs/specs/index.md` — SHALL be edited in lockstep to preserve single-sourcing. Historical/generated content (`docs/memory/**`, `docs/specs/findings/*`) is explicitly OUT of scope (Non-Goals).

- **GIVEN** deployable kit content (or its dev-source) cites a fab-kit principle by Roman numeral
- **WHEN** the sweep rewrites it
- **THEN** the named principle + attribution remain, the numeral is gone, and the deployed file + its single-source match
- **AND** `grep -rnE "Constitution [IVX]+" src/kit/` returns zero matches, and `docs/specs/fkf.md` + `docs/specs/index.md` return zero

### Non-Goals

- Path/role references to `constitution.md` (the ~69 sites that name it as a file at a known path / describe its role) — these are correct in any consuming repo and stay verbatim.
- `fab/project/constitution.md` itself — its own numbering is legitimate.
- `docs/memory/**` (18 files) — post-implementation history citing the numbering as it was when each change shipped; not deployable text, and `log.md`/`log.seed.md` are generated. Left as historical record.
- `docs/specs/findings/*` — dated point-in-time analysis artifacts; left as historical record.
- `.claude/skills/*` — gitignored deployed copies produced by `fab sync`; never edited directly. `src/kit/skills/` is canonical.

> **Correction (post-review)**: an earlier draft claimed top-level `docs/specs/*.md` had zero numbered citations. Inaccurate — `docs/specs/fkf.md` and `docs/specs/index.md` carried `Constitution VI` (now fixed under R4); `docs/specs/findings/*` still do (intentionally left as historical artifacts).

### Design Decisions

1. **Name-the-principle-inline rewrite style**: replace `Constitution <Numeral>` with the named concept attributed as a fab-kit design principle — *Why*: the cited rule is sound (fab-kit explaining its own design); keeping the named rationale preserves the "this is a governing principle" signal while dropping only the pointer that rots on deployment — *Rejected*: deleting the rationale (loses the signal); genericizing to "the project constitution" (still implies the consumer's constitution governs a fab-internal choice, and is vaguer).
2. **Per-occurrence editing on dense lines**: edit each citation individually rather than a single global sed — *Why*: line 20 of `docs-reorg-specs.md` and line 9 of `SPEC-docs-reorg-specs.md` carry three citations apiece amid bold-markup and block-quote prose; a blunt replace risks mangled sentences — *Rejected*: global find-replace (unsafe for prose).

## Tasks

### Phase 2: Core Implementation

- [x] T001 [P] Rewrite the `Constitution I` citation in `src/kit/skills/_cli-fab.md` (line ~255, "provider neutrality, Constitution I") to name provider neutrality / Pure Prompt Play as a fab-kit design principle <!-- R1 -->
- [x] T002 [P] Rewrite the `Constitution III` citation in `src/kit/skills/fab-continue.md` (line ~206, "state-wise no-op — Constitution III") to name idempotency as a fab-kit design principle <!-- R1 -->
- [x] T003 [P] Rewrite the `Constitution III` citation in `src/kit/skills/docs-hydrate-memory.md` (line ~191, "idempotency, Constitution III") <!-- R1 -->
- [x] T004 [P] Rewrite the `Constitution III` citation in `src/kit/skills/docs-reorg-memory.md` (line ~161, "idempotency, Constitution III") <!-- R1 -->
- [x] T005 [P] Rewrite the `Constitution VI` citation in `src/kit/skills/docs-hydrate-specs.md` (line ~73, "human-curated (Constitution VI)") to name specs-are-human-curated as a fab-kit design principle <!-- R1 -->
- [x] T006 Rewrite ALL seven `Constitution VI` occurrences in `src/kit/skills/docs-reorg-specs.md` per-occurrence (lines ~18, ~20 [×3], ~83, ~123, ~124), preserving the dense block-quote prose and the "the constitution rejects" / "per **Constitution VI**" phrasings <!-- R3 -->
- [x] T007 [P] Rewrite the `Constitution I` citation in `docs/specs/skills/SPEC-fab-setup.md` (line ~91, "per Constitution I.") <!-- R2 -->
- [x] T008 [P] Rewrite the `Constitution III` citation in `docs/specs/skills/SPEC-git-pr.md` (line ~9, "Re-run contract (Constitution III, ...") <!-- R2 -->
- [x] T009 [P] Rewrite the `Constitution III` citation in `docs/specs/skills/SPEC-fab-draft.md` (line ~9, "Re-run contract (Constitution III)") <!-- R2 -->
- [x] T010 [P] Rewrite the `Constitution III` citation in `docs/specs/skills/SPEC-fab-new.md` (line ~9, "Re-run contract (Constitution III)") <!-- R2 -->
- [x] T011 Rewrite the mid-sentence `Constitution III` citation in `docs/specs/skills/SPEC-fab-continue.md` (line ~9, "a state-wise no-op — Constitution III" inside a long contract clause) inline without breaking the surrounding sentence <!-- R3 -->
- [x] T012 [P] Rewrite both citations in `docs/specs/skills/SPEC-docs-reorg-memory.md` (line ~30 "Idempotency (Constitution III)"; line ~46 "(Constitution I, Pure Prompt Play)") <!-- R2 -->
- [x] T013 Rewrite ALL five `Constitution VI` occurrences in `docs/specs/skills/SPEC-docs-reorg-specs.md` per-occurrence (line ~7, ~9 [×3, incl. "per **Constitution VI**"], ~22), preserving prose and bold-markup <!-- R3 -->

### Phase 2b: Scope expansion (post-review, deployable kit content)

- [x] T015 [P] Rewrite the `Constitution VI` citation in `src/kit/reference/fkf.md` (line ~21, scope note) and its dev-source `docs/specs/fkf.md` (lines ~13 scope note + ~349 §9 Non-Scope) in lockstep <!-- R4 -->
- [x] T016 [P] Rewrite the `Constitution VI` citation in `docs/specs/index.md` (line ~26, fkf index row) <!-- R4 -->
- [x] T017 [P] Rewrite the `Constitution III` citation in `src/kit/migrations/2.4.2-to-2.5.0.md` (line ~131, no-op verification note) <!-- R4 -->

### Phase 3: Verification

- [x] T014 Run `grep -rnE "Constitution [IVX]+" src/kit/skills/ docs/specs/skills/` and confirm ZERO matches; spot-check each edited file to confirm the named-rationale concept still appears at every former citation site <!-- R1 R2 R3 -->
- [x] T018 Run `grep -rnE "Constitution [IVX]+" src/kit/` (all deployable kit) and `grep -rnE "Constitution [IVX]+" docs/specs/fkf.md docs/specs/index.md` — both confirm ZERO matches; confirm `docs/memory/**` (18) and `docs/specs/findings/*` (2) are intentionally untouched <!-- R4 -->

## Execution Order

- T001–T013 are independent edits across distinct files; T006/T011/T013 are the per-occurrence-careful ones (no `[P]` flag — handled deliberately). All edits precede T014.
- T014 (verification grep) MUST run last, after all edits.

## Acceptance

### Functional Completeness

- [x] A-001 R1: Every `Constitution <Numeral>` citation in `src/kit/skills/*.md` is rewritten with the named principle + fab-kit-design-principle attribution; the named concept (idempotency / provider neutrality / Pure Prompt Play / human-curated) remains at each site
- [x] A-002 R2: Every `Constitution <Numeral>` citation in `docs/specs/skills/SPEC-*.md` is rewritten the same way, keeping skills and spec mirrors in sync
- [x] A-003 R3: Multi-occurrence lines (`docs-reorg-specs.md` line 20, `SPEC-docs-reorg-specs.md` line 9) and the mid-sentence `SPEC-fab-continue.md` citation are edited per-occurrence with intact prose and bold-markup
- [x] A-012 R4: `src/kit/reference/fkf.md`, `src/kit/migrations/2.4.2-to-2.5.0.md`, `docs/specs/fkf.md`, and `docs/specs/index.md` are rewritten with named+attributed principles; deployed `reference/fkf.md` and its dev-source `docs/specs/fkf.md` match (single-sourcing preserved)

### Behavioral Correctness

- [x] A-004 R1: `grep -rnE "Constitution [IVX]+" src/kit/skills/` returns zero matches
- [x] A-005 R2: `grep -rnE "Constitution [IVX]+" docs/specs/skills/` returns zero matches
- [x] A-013 R4: `grep -rnE "Constitution [IVX]+" src/kit/` returns zero matches (all deployable kit clean); `docs/specs/fkf.md` + `docs/specs/index.md` return zero; `docs/memory/**` and `docs/specs/findings/*` remain intentionally untouched

### Scenario Coverage

- [x] A-006 R1: Path/role references to `constitution.md` and `fab/project/constitution.md` itself are untouched (no Non-Goal site was edited)

### Edge Cases & Error Handling

- [x] A-007 R3: No stray Roman numeral, broken sentence, or orphaned `**`/`per` markup remains on any rewritten dense or mid-sentence line

### Code Quality

- [x] A-008 Pattern consistency: Rewrite phrasings are consistent across sites (the same canonical attribution form is reused, tuned only for local sentence flow)
- [x] A-009 No unnecessary duplication: No content beyond the numeral pointer was added or removed; each edit is minimal

### Documentation Accuracy

- [x] A-010 R1: Each former citation site reads correctly in-context and survives deployment to an arbitrary consuming repo unchanged (no dangling pointer into fab-kit's principle numbering)

### Cross-References

- [x] A-011 R2: Skill bodies and their SPEC mirrors carry equivalent rewritten rationale (no skill↔SPEC drift introduced by the sweep)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- The intake's prose occurrence-counts (`docs-reorg-specs.md` "5", `SPEC-docs-reorg-specs.md` "3") undercount the true occurrence totals (7 and 5 respectively — line 20 and line 9 each carry THREE). The intake's *line-level* inventory is correct and names every site; the zero-match grep is the authoritative completeness check.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope = `src/kit/skills/` + `docs/specs/skills/` only; constitution self-citation, top-level specs, and `.claude/skills/` left untouched | Intake assumption #1 (user-confirmed "Skills + spec mirrors"); grep confirms top-level specs have zero numbered citations | S:95 R:80 A:95 D:95 |
| 2 | Certain | Rewrite style = name the principle inline + attribute as a fab-kit design principle; keep rationale, drop only the numeral | Intake assumption #2 (user-confirmed "Name the principle inline" over drop / genericize) | S:95 R:75 A:95 D:95 |
| 3 | Confident | Multi-occurrence lines and the mid-sentence `SPEC-fab-continue.md` citation get per-occurrence inline edits, not a blanket sed | Intake assumption #5; a blunt replace would mangle the dense block-quotes and the long contract clause | S:85 R:80 A:80 D:75 |
| 4 | Confident | Verification target is the zero-match grep over actual occurrences (24), not the intake's prose count (22); two dense lines carry 3 citations each | Intake's line-level inventory is correct but its prose occurrence-counts undercount; the grep is the authoritative completeness gate | S:80 R:85 A:90 D:80 |

4 assumptions (2 certain, 2 confident, 0 tentative).
