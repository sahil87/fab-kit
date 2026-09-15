# Plan: Memory Writer Contract — Hydrate Rewrites, Never Appends

**Change**: 260718-wrct-hydrate-rewrite-contract
**Intake**: `intake.md`

## Requirements

> Markdown-only change. It **extends the shipped writer contract in place** — the
> rewrite-not-append core already exists (`fab-continue.md` Hydrate step 4 "Merge as
> current truth", `docs-hydrate-memory.md` ingest Step 3 item 4, FKF §3.3 "Body style").
> The four new rule classes below (heading change-id ban, post-body-edit description
> re-check, post-hydrate self-check, DD-entry-shape + changelog-bullet ban + no-TODOs)
> do not exist anywhere today (grep-verified during the intake's gap analysis) and are
> added as rule additions in the same normative homes.

### FKF: §3.3 Body-Style Additions (both `fkf.md` copies)

The two FKF copies — `docs/specs/fkf.md` (dev-repo design doc) and `src/kit/reference/fkf.md`
(shipped normative extract) — MUST never diverge on normative rules; both receive identical
amendments to the §3.2/§3.3 body-style rule lists (Certain assumption #2).

#### R1: No operational TODOs in a memory body
The §3.3 "Body style" normative bullet list SHALL carry a rule that follow-up work items — TODOs,
"still needs X", next-step checklists — are never memory-body content; they belong in the project
backlog (`fab/backlog.md`) or the originating change folder. A memory body states what IS, not what
remains to be done.

- **GIVEN** a hydrate/`docs-hydrate-memory` writer is authoring a memory body
- **WHEN** it would otherwise embed a follow-up work item (e.g. a "delete the X secret" TODO)
- **THEN** the rule in both `fkf.md` copies directs that item to the backlog or change folder, not the body

#### R2: Headings carry no change-ids
The §3.3 body-style rules SHALL state that heading text names the topic and never a change; a
heading MUST NOT carry a change-id (`### Dispatch States (xu0k)` / `## xu0k — dispatch states`
are prohibited). Change-id provenance stays citation-only in body text, extending the existing
"Provenance is citation-only" bullet.

- **GIVEN** a memory writer writes or edits a section heading
- **WHEN** the heading names a topic
- **THEN** both `fkf.md` copies require the heading text to be change-id-free (provenance stays a trailing `(change-id)` / `*Introduced by*` in the body)

#### R3: Design-Decisions entry shape + changelog-bullet ban
The §3.3 conventional-structure guidance (around the existing four-field DD scaffold) SHALL state
that (a) any *why* / rejected alternative / constraint explanation belongs in a `## Design Decisions`
entry in the four-field shape (**Decision** / **Why** / **Rejected** / *Introduced by*), never as
inline narration in Overview/Requirements prose; and (b) the changelog-bullet shape
(`- **{change-id} — retired X**`) is banned inside `## Design Decisions` — that is change history
(`log.md`'s job, §6), not a design decision. A DD entry heading is a decision *title*, never a change-id.

- **GIVEN** a memory writer has "don't re-break this" rationale to record
- **WHEN** it writes into `## Design Decisions`
- **THEN** both `fkf.md` copies require the four-field entry shape and forbid a change-id-keyed changelog bullet inside the section

### Writer Contract: `fab-continue.md` Hydrate Behavior

#### R4: Heading change-id ban at hydrate
`fab-continue.md` § Hydrate Behavior step 4's current-truth merge bullets SHALL carry the
heading-change-id ban: writers never introduce a change-id-suffixed heading; change-ids appear only
as trailing body citations. (Draining pre-existing such headings in other repos is `[dsrx]`'s scope,
not this change.)

- **GIVEN** the pipeline hydrate block is rewriting a memory section
- **WHEN** it writes or renames a heading
- **THEN** the heading names the topic with no change-id

#### R5: Post-body-edit `description:` re-check
The Hydrate step 4 merge bullets SHALL add an explicit *post-body-edit trigger*: after any body edit,
re-check that the file's `description:` frontmatter still routes accurately — one line, ≤500 chars,
change-id-free (FKF §3.2). Today's rules require an accurate, within-cap `description:` at merge time;
this adds the trigger that the writer re-reads the description *after* body growth (the audit found
descriptions drifting far past cap because nobody re-read them).

- **GIVEN** hydrate has edited a memory file's body
- **WHEN** the body changed
- **THEN** the writer re-checks the `description:` is still an accurate one-line, ≤500-char, change-id-free routing signal

#### R6: Post-hydrate self-check step
`fab-continue.md` § Hydrate Behavior SHALL gain a new numbered step (between step 4 "Hydrate
docs/memory/" and the return step): after all memory writes and before returning/regenerating
indexes, re-read every file touched this run and strip any transition phrasing just introduced — no
"renamed/now/previously/no longer/was `old.value`" narration, no change-keyed delta paragraph left
below an older paragraph on the same topic, no change-ids in headings, `description:` still routes.
The step is scoped to files touched this run (a self-review of this hydrate's own writes, not a corpus
sweep). Insertion at this seam is Confident assumption #5.

- **GIVEN** the hydrate block has finished writing memory files this run
- **WHEN** it reaches the new self-check step (before index regen / return)
- **THEN** it re-reads only the files it touched and strips any transition phrasing / change-id heading / delta paragraph it just introduced, and confirms descriptions still route

#### R7: Pattern-capture aligned to four-field DD shape
`fab-continue.md` Hydrate step 6 (Pattern capture) already routes patterns into Design Decisions with
citation-form provenance; its wording SHALL be aligned with the four-field DD entry shape
(**Decision** / **Why** / **Rejected** / *Introduced by*) so a captured pattern lands as a conforming
DD entry.

- **GIVEN** hydrate captures a non-obvious implementation pattern
- **WHEN** it records the pattern in a memory file's `## Design Decisions`
- **THEN** the step's wording directs a four-field DD entry (not free-text narration)

### Writer Contract: `docs-hydrate-memory.md` (ingest + generate; backfill exempt)

#### R8: Same writer rules on the standalone ingest/generate paths
`docs-hydrate-memory.md` SHALL carry the same three merge-time writer rules (heading change-id ban,
post-body-edit `description:` re-check, DD-entry-shape/changelog-bullet ban) on its ingest Step 3
(items 3–4 and the FKF-frontmatter paragraph below them) and generate Step 3 (which references ingest
Step 3). Backfill mode is exempt from the body rules — it is a pure-frontmatter, body-preserving
operation — except the change-id-free `description:` rule, which it already applies.

- **GIVEN** `/docs-hydrate-memory` runs in ingest or generate mode
- **WHEN** it creates or merges a memory file body
- **THEN** the skill applies the heading-ban, post-body-edit description re-check, and DD-entry-shape/changelog-bullet-ban rules

#### R9: Post-hydrate self-check step (ingest/generate)
`docs-hydrate-memory.md` SHALL gain an equivalent post-hydrate self-check step in ingest and generate
modes, placed before index regeneration, scoped to files touched this run. Backfill mode is exempt
(body-preserving by contract). Placement is Confident assumption #5.

- **GIVEN** `/docs-hydrate-memory` (ingest or generate) has written its memory files
- **WHEN** it reaches the self-check step (before `fab memory-index` regen)
- **THEN** it re-reads the files it touched this run and strips transition phrasing / change-id headings / delta paragraphs; backfill mode does not run this step

### Plan-DD Seam: `_generation.md`

#### R10: Plan `### Design Decisions` aligned to four-field shape
`_generation.md`'s Plan Generation Procedure `### Design Decisions` subsection (currently "summary +
rationale + rejected alternatives") SHALL be aligned to the four-field DD entry shape so hydrate's
pattern capture can lift a plan DD entry into memory DD without reshaping. This is the minimal edit
consistent with the backlog naming `_generation` as a surface (Confident assumption #4). `_generation.md`
contains no memory-writing procedure, so no other writer rules attach here.

- **GIVEN** the Plan Generation Procedure emits an optional `### Design Decisions` subsection
- **WHEN** it records an architectural choice
- **THEN** the subsection guidance names the four-field shape (Decision / Why / Rejected / *Introduced by*), matching memory DD entries

### Template + Mirror Sweep

#### R11: `templates/memory.md` guidance carries the full contract
`src/kit/templates/memory.md` guidance comments already restate §3.3 body style; they SHALL add the
new rules (heading change-id ban, no operational TODOs, DD entry-shape / changelog-bullet ban) so the
template a writer reads at file-creation time carries the full contract.

- **GIVEN** a writer reads `templates/memory.md` at file-creation time
- **WHEN** it fills the scaffold
- **THEN** the guidance comments state the heading ban, the no-TODOs rule, and the DD entry-shape / changelog-bullet ban

#### R12: SPEC mirrors updated (constitution-required)
The SPEC mirror class SHALL be swept: `docs/specs/skills/SPEC-fab-continue.md`,
`SPEC-docs-hydrate-memory.md`, and `SPEC-_generation.md` mirror the corresponding skill edits
(constitution Additional Constraints: a skill change MUST update its `SPEC-*.md`). Certain/Confident
assumption #6.

- **GIVEN** a skill source under `src/kit/skills/` is edited
- **WHEN** the change is applied
- **THEN** the matching `docs/specs/skills/SPEC-*.md` mirror is updated in the same change

#### R13: Aggregate specs updated only where they restate the amended rules
`docs/specs/skills.md` (§ Hydrate Behavior) and `docs/specs/templates.md` (§ Individual File / memory
format) SHALL be updated only where they restate the merge/body-style rules this change amends
(class-sweep discipline, `code-quality.md` § Sibling & Mirror Sweeps). Where an aggregate does not
restate an amended rule, it is left untouched.

- **GIVEN** an aggregate spec restates a body-style/merge rule this change amends
- **WHEN** the sweep runs
- **THEN** that restatement is updated to carry the new rule; aggregates that do not restate it are left unchanged

### Non-Goals

- Enforcement / warnings (`fab memory-index` narration-density, size caps, blocking tiers) — owned by `[mxgu]`, not this change.
- Draining existing debt (change-id headings, over-cap descriptions) in this or other repos — owned by `[dsrx]`'s distill extensions.
- Any Go code change or `fab` CLI-surface change (so no `_cli-fab.md` update).
- Restructuring the shipped rewrite-not-append core — this change extends it in place.

### Design Decisions

#### Extend in place, don't restructure
**Decision**: Add the four residual leak-class rules to the existing writer-contract homes rather than reshape the shipped rewrite-not-append core (`260717-3plm`).
**Why**: only the four residual leak classes are absent (grep-verified); reshaping shipped rules would be churn with no benefit.
**Rejected**: a fresh consolidated "writer contract" section — duplicates the normative homes the existing rules occupy and invites divergence.
*Introduced by*: 260718-wrct-hydrate-rewrite-contract

#### Both `fkf.md` copies amended identically
**Decision**: Apply the §3.2/§3.3 amendments verbatim to both `docs/specs/fkf.md` and `src/kit/reference/fkf.md`.
**Why**: FKF's own single-sourcing note requires it; the same seam `[mxgu]` shares.
**Rejected**: amend only the shipped extract — the design doc would drift from the normative rules it documents.
*Introduced by*: 260718-wrct-hydrate-rewrite-contract

#### Self-check as its own numbered step
**Decision**: Land the post-hydrate self-check as a distinct numbered step scoped to files touched this run, not a corpus sweep.
**Why**: it is a self-review of this hydrate's own writes; a corpus sweep is `[dsrx]`'s job.
**Rejected**: making it a bullet under step 4 — a final re-read pass reads more naturally as its own step at the procedure's tail.
*Introduced by*: 260718-wrct-hydrate-rewrite-contract

## Tasks

### Phase 1: Normative Home (FKF)

- [x] T001 Add the three §3.3 body-style rule additions (R1 no-TODOs, R2 heading change-id ban, R3 DD entry-shape + changelog-bullet ban) to `docs/specs/fkf.md` §3.2/§3.3 <!-- R1 R2 R3 -->
- [x] T002 Mirror the identical additions into `src/kit/reference/fkf.md` §3.2/§3.3 (verbatim rule parity with T001) <!-- R1 R2 R3 -->

### Phase 2: Writer Skills

- [x] T003 Amend `src/kit/skills/fab-continue.md` § Hydrate Behavior: add heading-ban (R4) + post-body-edit `description:` re-check (R5) to step 4 merge bullets; insert the post-hydrate self-check numbered step (R6) between step 4 and the return step; align pattern-capture step 6 to the four-field DD shape (R7) <!-- R4 R5 R6 R7 -->
- [x] T004 Amend `src/kit/skills/docs-hydrate-memory.md`: add the three merge-time writer rules (R8) to ingest Step 3 items 3–4 + the FKF-frontmatter paragraph + generate Step 3; insert the post-hydrate self-check step (R9) in ingest/generate modes before index regen, backfill exempt <!-- R8 R9 -->
- [x] T005 [P] Align the Plan Generation Procedure `### Design Decisions` subsection in `src/kit/skills/_generation.md` to the four-field DD entry shape (R10) <!-- R10 -->

### Phase 3: Template + Mirror Sweep

- [x] T006 [P] Add the new rules (heading ban, no-TODOs, DD entry-shape / changelog-bullet ban) to `src/kit/templates/memory.md` guidance comments (R11) <!-- R11 -->
- [x] T007 Update `docs/specs/skills/SPEC-fab-continue.md` to mirror the fab-continue Hydrate edits — add a change-summary paragraph for the writer-contract additions (R12) <!-- R12 -->
- [x] T008 Update `docs/specs/skills/SPEC-docs-hydrate-memory.md` to mirror the docs-hydrate-memory edits (R12) <!-- R12 -->
- [x] T009 Update `docs/specs/skills/SPEC-_generation.md` to mirror the `_generation` DD-shape alignment (R12) <!-- R12 -->
- [x] T010 Sweep `docs/specs/skills.md` § Hydrate Behavior and `docs/specs/templates.md` § Individual File — update only where they restate the amended merge/body-style rules (R13) <!-- R13 -->

### Phase 4: Verify

- [x] T011 Repo-wide grep sweep to confirm every restatement of the merge/body-style rules in the class carries the new rules, and that no `.claude/skills/` deployed copy was edited (canonical `src/kit/` only) <!-- R12 R13 -->

## Execution Order

- T001 → T002 (write the design-doc copy first, then mirror it verbatim for rule parity)
- T003, T004 depend on T001/T002 landing (the skills cite the FKF sections); T005, T006 are independent `[P]`
- T007–T010 follow their source edits (T007←T003, T008←T004, T009←T005, T010←T003/T004)
- T011 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: Both `fkf.md` copies state that operational TODOs/follow-ups belong in the backlog or change folder, never a memory body
- [x] A-002 R2: Both `fkf.md` copies state that headings carry no change-ids (provenance citation-only in body)
- [x] A-003 R3: Both `fkf.md` copies state the four-field DD entry shape and ban the changelog-bullet shape inside `## Design Decisions`
- [x] A-004 R4: `fab-continue.md` Hydrate step 4 forbids introducing change-id-suffixed headings
- [x] A-005 R5: `fab-continue.md` Hydrate step 4 carries the post-body-edit `description:` re-check trigger (one line, ≤500 chars, change-id-free)
- [x] A-006 R6: `fab-continue.md` Hydrate Behavior has a post-hydrate self-check numbered step between step 4 and return, scoped to files touched this run, backfill-exempt N/A (backfill is docs-hydrate-memory's mode)
- [x] A-007 R7: `fab-continue.md` Hydrate step 6 pattern-capture wording names the four-field DD entry shape
- [x] A-008 R8: `docs-hydrate-memory.md` ingest Step 3 + generate Step 3 carry the heading-ban, description re-check, and DD-entry-shape/changelog-bullet-ban rules; backfill exempt from the body rules
- [x] A-009 R9: `docs-hydrate-memory.md` ingest/generate modes have a post-hydrate self-check step before index regen; backfill mode does not
- [x] A-010 R10: `_generation.md` Plan Generation `### Design Decisions` subsection names the four-field DD entry shape
- [x] A-011 R11: `templates/memory.md` guidance comments state the heading ban, no-TODOs rule, and DD entry-shape / changelog-bullet ban
- [x] A-012 R12: `SPEC-fab-continue.md`, `SPEC-docs-hydrate-memory.md`, and `SPEC-_generation.md` are updated to mirror their skill edits
- [x] A-013 R13: `docs/specs/skills.md` and `docs/specs/templates.md` restatements of the amended rules are updated; non-restating aggregate content is left unchanged

### Behavioral Correctness

- [x] A-014 R1 R2 R3: The two `fkf.md` copies remain rule-identical after the amendment (no normative divergence between the design doc and the shipped extract)
- [x] A-015 R6 R9: The self-check step is explicitly scoped to files touched *this run* and explicitly exempts backfill mode (a corpus sweep or backfill body-rewrite would be wrong)

### Scenario Coverage

- [x] A-016 R4 R8: A writer editing a heading is directed (in both the pipeline and standalone paths) to keep it change-id-free

### Edge Cases & Error Handling

- [x] A-017 R8 R9: Backfill mode's body-preserving contract is not broken — it applies only the change-id-free `description:` rule, not the body-style rules or the self-check step

### Code Quality

- [x] A-018 Pattern consistency: New rule text follows the existing FKF/skill prose style (RFC-2119 phrasing in FKF, present-tense bullets in skills) and existing citation conventions
- [x] A-019 No unnecessary duplication: Rules are added in their normative homes and referenced (not re-explained) in the mirrors; no rule text is duplicated beyond the constitution-mandated `fkf.md` twin and SPEC mirrors
- [x] A-020 Canonical source only: No file under `.claude/skills/` is edited — all skill/template edits are in `src/kit/` (constitution V; code-quality.md § Anti-Patterns)
- [x] A-021 Sibling & Mirror Sweeps: The whole mirror class (SPEC-*.md mirrors + aggregate specs restating the amended rules) is swept up front, not left for review to flag (code-quality.md § Sibling & Mirror Sweeps)

### Documentation Accuracy

- [x] A-022 documentation_accuracy: Every new rule statement is internally consistent with the shipped rewrite-not-append core and cites the correct FKF section numbers (§3.2 for description, §3.3 for body style, §6 for log.md)

### Cross-References

- [x] A-023 cross_references: FKF section citations and skill↔spec cross-links introduced or touched by this change resolve correctly (e.g. `fkf.md` §3.3, `src/kit/reference/fkf.md`, `log.md` §6)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Coordination: PARALLEL with `[mxgu]` (shared only the two `fkf.md` files — merge seam, no ordering dependency). `[dsrx]` is downstream and cites the §3.3 rules this change writes.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is writer rules only — no enforcement signals ([mxgu]) and no debt drain ([dsrx]) | Backlog states the three-change split and each sibling's ownership explicitly; intake Certain #1 | S:95 R:90 A:95 D:95 |
| 2 | Certain | Both `fkf.md` copies receive identical §3.2/§3.3 amendments | FKF single-sourcing note + backlog "Update BOTH"; intake Certain #2 | S:95 R:85 A:95 D:95 |
| 3 | Certain | Extend the shipped current-truth rules in place; only the four new rule classes are absent | Grep-verified (this apply re-confirmed: no heading-ban, no TODO rule, no self-check, no DD changelog-bullet ban exist today); intake Certain #3 | S:85 R:85 A:90 D:80 |
| 4 | Confident | `_generation.md`'s only touchpoint is aligning the Plan-DD subsection to the four-field shape (no memory-writing procedure lives there) | Verified by grep; the plan-DD → memory-DD lift is the real seam and the minimal edit honoring the named surface; intake Confident #4 | S:50 R:80 A:75 D:60 |
| 5 | Confident | Post-hydrate self-check lands as a new numbered step (after step 4 in fab-continue Hydrate; before index regen in docs-hydrate-memory ingest/generate), scoped to files touched this run, backfill exempt | Backlog specifies the check's content but not placement; these are the procedure seams where a final pass fits, and backfill is body-preserving by contract; intake Confident #5 | S:70 R:85 A:80 D:70 |
| 6 | Confident | `templates/memory.md` + the three SPEC mirrors are in-scope; `skills.md`/`templates.md` updated only where they restate amended rules | Constitution mandates SPEC mirrors; the template already restates §3.3 and would contradict the new rules if skipped; class-sweep discipline; intake Confident #6 | S:55 R:90 A:85 D:75 |
| 7 | Confident | SPEC mirrors record the additions as this repo's dated change-summary-paragraph convention (not FKF present-truth bodies) | The `SPEC-*.md` files are skill spec mirrors that already carry per-change dated paragraphs (e.g. `260717-3plm`); FKF §3.3 governs `docs/memory/` bodies only (§9 excludes specs), so the present-truth ban does not apply to these mirrors | S:70 R:85 A:85 D:75 |

7 assumptions (3 certain, 4 confident, 0 tentative).
