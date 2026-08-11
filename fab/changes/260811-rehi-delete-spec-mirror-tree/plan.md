# Plan: Delete SPEC Mirror Tree and Retire the Mirror Rule

**Change**: 260811-rehi-delete-spec-mirror-tree
**Intake**: `intake.md`

## Requirements

### Specs: Skeleton Fold into `docs/specs/skills.md`

#### R1: Fold condensed Flow skeletons into per-skill sections
Each of the 27 user-invocable skills' existing `## /<skill>` sections in `docs/specs/skills.md` SHALL gain its condensed Flow skeleton from the current post-xy7a mirror (`docs/specs/skills/SPEC-*.md`): the short header, the Flow diagram, and Tools/Sub-agents one-liners. Tools/Sub-agents lines SHALL be included only where they add real information beyond the section's existing prose. The fold MUST happen before any deletion of `docs/specs/skills/` — the mirrors are the source.

- **GIVEN** the 36 condensed mirrors exist in `docs/specs/skills/`
- **WHEN** the fold task completes
- **THEN** every user-invocable skill's `skills.md` section carries its Flow diagram
- **AND** no mirror content needed later is lost before the tree is deleted

#### R2: Fold partial skeletons into § Skill Helpers selectively
The 9 partial mirrors (`SPEC-_*.md`) have no per-skill sections; a partial's Flow skeleton SHALL be folded into `docs/specs/skills.md` § Skill Helpers only where it carries a real flow that adds information (e.g., `_preamble`, `_generation`, `_intake`, `_pipeline`, `_review`, `_srad`). Pure-reference partials (`_cli-fab`, `_cli-external`, `_cli-agents` — no Flow diagram) lose their skeletons without replacement.

- **GIVEN** § Skill Helpers already documents every partial
- **WHEN** the fold completes
- **THEN** partials with real flows have their skeleton present there, and pure-reference partials add nothing

#### R3: Fix in-file live references in `skills.md`
New Skill Checklist item 6 (skills.md:126) SHALL be replaced — the "create `docs/specs/skills/SPEC-{name}.md`" instruction becomes "add the skill's Flow skeleton to its skills.md section". The `/fab-operator` section's "Full spec" line (skills.md:778) SHALL drop its `skills/SPEC-fab-operator.md` link.

- **GIVEN** skills.md:126 mandates creating a SPEC mirror and skills.md:778 links to one
- **WHEN** the fold completes
- **THEN** neither line references the mirror tree

### Specs: Tree Deletion

#### R4: Delete `docs/specs/skills/` and its index row
All 36 `SPEC-*.md` files under `docs/specs/skills/` SHALL be deleted (no `index.md` exists there), and the `skills/` row (last table row) SHALL be removed from `docs/specs/index.md`.

- **GIVEN** the fold (R1–R2) has completed
- **WHEN** the deletion runs
- **THEN** `docs/specs/skills/` no longer exists and `docs/specs/index.md` has no `skills/` row

### Governance: Constitution Amendment

#### R5: Remove the SPEC-mirror rule from the constitution
The Additional Constraints clause "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file …" SHALL be deleted outright from `fab/project/constitution.md`. A dated HTML-comment amendment annotation (matching the five existing `<!-- YYYY-MM-DD (change-id): … -->` blocks) SHALL record the clause removal, the mirror-tree deletion, and where the skeletons went. Version SHALL bump 1.5.0 → 1.6.0 (normative MUST rule removed); **Last Amended** SHALL be 2026-08-11.

- **GIVEN** constitution v1.5.0 with the mirror clause in Additional Constraints
- **WHEN** the amendment completes
- **THEN** the clause is gone, an annotation records the removal, and Governance reads Version 1.6.0 / Last Amended 2026-08-11

### Enforcement Machinery Removal

#### R6: Strip the enforcement/sweep machinery
Four surfaces SHALL lose their SPEC-mirror machinery:
- `fab/project/code-review.md` — delete the **SPEC-mirror sync** must-fix rule (first bullet under § Project-Specific Review Rules)
- `fab/project/code-quality.md` — delete the anti-pattern "**Shipping a structural skill change without its SPEC mirror**", the mirror entry (first bullet) in § Sibling & Mirror Sweeps' class list, **and** the section's trailing mirror blockquote; the other sweep classes (twin skills, aggregate specs, memory files) stay
- `docs/specs/skills.md` — checklist item 6 replacement (covered by R3)
- `src/kit/skills/docs-reorg-specs.md` — remove the reserved-path carve-outs (the "Reserved paths" note item and the "never migrate reserved paths (`docs/specs/skills/SPEC-*.md`)" constraint); `.claude/skills/` deployed copies MUST NOT be edited

- **GIVEN** the four enforcement surfaces exist
- **WHEN** the stripping completes
- **THEN** no live rule mandates or references SPEC-mirror maintenance

### Live Reference Sweep

#### R7: Rewrite live references to present truth
Every live reference to the mirror tree SHALL be removed or rewritten:
- `docs/memory/memory-docs/specs-index.md` — remove § Per-Skill SPEC Mirrors and the reserved-path-exemption Design Decision; describe the folded skeletons' home in `skills.md`; keep the `description:` accurate and ≤500 chars, change-id-free
- `docs/memory/pipeline/dedupe.md` — drop the `SPEC-fab-dedupe.md` mirror sentence (line 19)
- `docs/memory/pipeline/execution-skills.md` — drop the `SPEC-git-pr.md` Flow-tree mirror clause (line 67 trailing clause)
- `docs/memory/_shared/context-loading.md` — drop the `SPEC-_preamble.md` parenthetical (line 93)
- `docs/memory/runtime/operator.md` — trim the dead "three SPEC mirrors were aligned" clause in the `_cli-external` Design Decision (line 460)
- `docs/specs/fkf.md` — fix the live claims at lines 204 and 229
- `fab/plans/sahil/skill-prose-consolidation.md` — update execution constraint 1 and the line ~398 sweep-checklist item

- **GIVEN** the intake's live-reference inventory
- **WHEN** the sweep completes
- **THEN** no listed file carries a live mirror-tree reference

#### R8: Historical records stay untouched
`docs/specs/findings/*`, all dated `log.md`/`log.seed.md` files, `fab/backlog.md` completed `[x]` entries, `fab/changes/*` artifacts, and `docs/specs/srad-scoring-rationale-v1-to-v2.md` MUST NOT be modified.

- **GIVEN** historical records referencing the mirror tree
- **WHEN** the change completes
- **THEN** those files are byte-identical to before

### Verification & Hygiene

#### R9: Memory indexes regenerated after memory edits
After the memory file edits, `fab memory-index` SHALL be run and its output taken wholesale (never hand-merged); `fab memory-index --check` MUST exit 0.

- **GIVEN** memory files were edited
- **WHEN** the regen runs
- **THEN** all `index.md` files reflect the edits and `--check` exits 0

#### R10: Zero surviving live references
A final repo-wide grep for `docs/specs/skills/` and `SPEC-` SHALL confirm no live reference survives outside the historical carve-outs (R8). Go sources and `scripts/` MUST be verified clean (intake verified none exist; the CLI⇒docs+tests rule is not triggered).

- **GIVEN** all edits and deletions are done
- **WHEN** the final grep runs
- **THEN** every remaining match is inside a historical carve-out

#### R11: Deployed copies untouched; markdown-only
Nothing under `.claude/skills/` SHALL be edited (canonical source is `src/kit/skills/`; `fab sync` redeploys). All artifacts remain plain markdown/YAML (Constitution IV).

- **GIVEN** the change touches a canonical skill source file
- **WHEN** the change completes
- **THEN** `git status` shows no modifications under `.claude/skills/`

### Non-Goals

- No Go, script, or test changes — intake verified zero code references to the mirror tree
- No changes to historical records (`docs/specs/findings/`, logs, backlog `[x]` entries, change artifacts, `srad-scoring-rationale-v1-to-v2.md`)
- No reorganization of `skills.md` beyond the fold — sections keep their existing prose

## Tasks

### Phase 1: Fold (source of truth still present)

- [x] T001 Fold all 27 user-invocable skills' condensed Flow skeletons (header + Flow + informative Tools/Sub-agents) into their `docs/specs/skills.md` sections; fold informative partial skeletons (`_preamble`, `_generation`, `_intake`, `_pipeline`, `_review`, `_srad`) into § Skill Helpers <!-- R1, R2 -->
- [x] T002 Replace New Skill Checklist item 6 and remove the `skills/SPEC-fab-operator.md` link (skills.md:778) in `docs/specs/skills.md` <!-- R3 -->

### Phase 2: Governance & machinery

- [x] T003 [P] Remove the SPEC-mirror clause from `fab/project/constitution.md`, add the dated amendment annotation, bump to 1.6.0 / Last Amended 2026-08-11 <!-- R5 -->
- [x] T004 [P] Delete the SPEC-mirror sync rule in `fab/project/code-review.md` and the mirror anti-pattern + sweep entry + trailing blockquote in `fab/project/code-quality.md` <!-- R6 -->
- [x] T005 [P] Remove the reserved-path carve-outs in `src/kit/skills/docs-reorg-specs.md` (canonical source; `.claude/skills/` untouched) <!-- R6, R11 -->

### Phase 3: Delete & sweep

- [x] T006 Delete `docs/specs/skills/` (36 files) and remove the `skills/` row from `docs/specs/index.md` <!-- R4 -->
- [x] T007 Sweep memory files: `docs/memory/memory-docs/specs-index.md`, `docs/memory/pipeline/dedupe.md`, `docs/memory/pipeline/execution-skills.md`, `docs/memory/_shared/context-loading.md`, `docs/memory/runtime/operator.md` <!-- R7 -->
- [x] T008 Sweep `docs/specs/fkf.md` (lines 204, 229) and `fab/plans/sahil/skill-prose-consolidation.md` (constraint 1, sweep-checklist item) <!-- R7 -->

### Phase 4: Verify

- [x] T009 Run `fab memory-index` (output wholesale) and confirm `fab memory-index --check` exits 0 <!-- R9 -->
- [x] T010 Final grep sweep: `docs/specs/skills/` and `SPEC-` repo-wide — no live references outside historical carve-outs; confirm `*.go` and `scripts/` clean; confirm `.claude/skills/` untouched <!-- R8, R10, R11 -->

## Execution Order

- T001 and T002 MUST complete before T006 — the mirrors are the fold's source
- T007 MUST complete before T009 — the regen consumes the memory edits
- T010 runs last, after everything

## Acceptance

### Functional Completeness

- [x] A-001 R1: Every user-invocable skill's `skills.md` section contains its Flow diagram from the retired mirror; Tools/Sub-agents appear only where they add information
- [x] A-002 R2: § Skill Helpers carries the informative partial skeletons (`_preamble`, `_generation`, `_intake`, `_pipeline`, `_review`, `_srad`); pure-reference partials add nothing
- [x] A-003 R3: Checklist item 6 instructs folding the Flow skeleton into the skills.md section; no `skills/SPEC-*.md` link remains in skills.md
- [x] A-004 R4: `docs/specs/skills/` does not exist; `docs/specs/index.md` has no `skills/` row
- [x] A-005 R5: Constitution has no SPEC-mirror clause, carries a dated 2026-08-11 annotation, and reads Version 1.6.0 / Last Amended 2026-08-11
- [x] A-006 R6: code-review.md, code-quality.md, and `src/kit/skills/docs-reorg-specs.md` carry no SPEC-mirror enforcement machinery; other sweep classes intact
- [x] A-007 R7: All seven listed live-reference sites are rewritten to present truth
- [x] A-008 R9: `fab memory-index` regenerated indexes are committed to disk and `fab memory-index --check` exits 0

### Removal Verification

- [x] A-009 R4/R6: `git status`/filesystem confirm the 36 mirror files deleted and the four enforcement surfaces stripped — no orphaned references to them
- [x] A-010 R10: Repo-wide grep for `docs/specs/skills/` and `SPEC-` matches only historical carve-outs (findings/, log.md/log.seed.md, backlog `[x]`, fab/changes/*, srad-scoring-rationale)

### Edge Cases & Error Handling

- [x] A-011 R8: `docs/specs/findings/`, dated `log.md`/`log.seed.md`, `fab/backlog.md` `[x]` entries, `fab/changes/*`, and `docs/specs/srad-scoring-rationale-v1-to-v2.md` are unmodified
- [x] A-012 R11: No file under `.claude/skills/` is modified; all changed artifacts are plain markdown/YAML

### Code Quality

- [x] A-013 Pattern consistency: Folded skeletons and annotations follow the surrounding files' existing conventions (skills.md section style, constitution annotation block style)
- [x] A-014 No unnecessary duplication: Tools/Sub-agents lines omitted where the section prose already states them; no restatement of the retired rule left anywhere

## Notes

- Change type: `docs` — markdown-only, zero Go/scripts/tests
- Fold-before-delete ordering is load-bearing (T001/T002 before T006)
- `fab memory-index` output is taken wholesale; index/log files are never hand-merged

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fold shape per section: short header line + fenced Flow diagram + Tools/Sub-agents one-liners only where informative, appended at the end of each `## /<skill>` section | Intake specifies the skeleton shape (title + short header + Flow + Tools + Sub-agents) and the add-information filter; title/heading already exists as the section heading | S:90 R:80 A:85 D:85 |
| 2 | Confident | Partial skeletons folded into § Skill Helpers as compact per-partial subsections for `_preamble`, `_generation`, `_intake`, `_pipeline`, `_review`, `_srad`; `_cli-fab`/`_cli-external`/`_cli-agents` dropped (no Flow) | Intake assumption row 7 — "only where it carries a real flow that adds information"; the six carry real flows, the three reference partials do not | S:40 R:85 A:65 D:55 |
| 3 | Confident | `docs/memory/runtime/operator.md` line 503's `*Introduced by*: 260811-xy7a-... (migrated from the condensed SPEC mirrors)` provenance citation stays — citation-only provenance is allowed by FKF §3.3 | Only line 460's dead clause is in the sweep inventory; provenance citations are not live claims | S:70 R:90 A:75 D:70 |
| 4 | Certain | `fab/plans/sahil/skill-prose-consolidation.md` constraint 1 and the sweep-checklist item are rewritten (not deleted) to drop the mirror requirement while keeping the plan's other constraints intact | The plan is a live execution doc; its mirror constraints are dead rules but the surrounding constraints stay valid | S:80 R:85 A:85 D:80 |
| 5 | Confident | The final grep surfaced three live sites beyond the intake inventory, swept the same way: `fab/plans/sahil/reuse-awareness-codemap.md:119` and `fab/plans/sahil/agent-state-divestment.md:121` (live plan docs — same class as skill-prose-consolidation.md) and the open `[ ]` backlog entry `[ruaw]`'s dead "full SPEC-mirror sweep" surface phrase; `docs/findings/index.md`'s dated review-round summary and the constitution's dated amendment annotations stay as historical records | The intake's sweep rule is "update every live reference"; plan docs and an open backlog entry are live, dated findings/annotation records are not | S:65 R:85 A:80 D:75 |

5 assumptions (2 certain, 3 confident, 0 tentative).
