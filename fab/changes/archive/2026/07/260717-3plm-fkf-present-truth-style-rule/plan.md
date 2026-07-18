# Plan: FKF Present-Truth Body-Style Rule + Memory-Writer Fixes

**Change**: 260717-3plm-fkf-present-truth-style-rule
**Intake**: `intake.md`

## Requirements

<!-- Prose-only change (no Go/CLI). Requirements capture the normative FKF amendments,
     the dual-file rule, the writer fixes, and the constitution-required mirror sweep. -->

### FKF Spec: Present-Truth Body-Style Rule

#### R1: §3.3 present-truth body-style rule (both FKF files)
FKF §3.3 (Body) SHALL carry a normative body-style rule stating that a memory-file body describes **current truth in present tense**, with **no transition narration** (never "renamed X→Y in {id}", "this inverts/supersedes {id}'s claim", "was `old.value`"), that **superseded behavior is never described in the body** (it belongs to the per-folder generated `log.md` (§6), git history, and archived change folders), that **provenance is limited to** trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions, and that **rationale survives distillation** (a rejected alternative is a durable design fact, not transition narration; token savings come from dropping narration, never rationale). The rule MUST land in BOTH `docs/specs/fkf.md` §3.3 AND `src/kit/reference/fkf.md` §3.3, with the existing §3.3 anchors/section numbers preserved.

- **GIVEN** the FKF §3.3 Body section in either FKF file
- **WHEN** a reader consults it for body-authoring rules
- **THEN** it states the present-truth rule (present tense, no transition narration, no superseded-behavior description, citation-only provenance, rationale-preserved), and the same normative text appears in the other FKF file

#### R2: §3.2 no-change-ids-in-`description:` clarification (both FKF files)
FKF §3.2 (`description`) SHALL state that the curated `description:` one-liner MUST NOT contain change-ids — neither `— xu0k`-style suffixes nor `(d9rs)`-style citations — because the description is a routing signal and provenance citations belong in the body. No enforcement is added (`fab memory-index` validation is NOT extended). The clarification MUST land in BOTH `docs/specs/fkf.md` §3.2 AND `src/kit/reference/fkf.md` §3.2, with anchors preserved.

- **GIVEN** the FKF §3.2 `description` section in either FKF file
- **WHEN** a reader consults it for what may appear in `description:`
- **THEN** it states the no-change-ids ban and points provenance citations to the body, and the same text appears in the other FKF file

### Memory Writers: Present-Truth Rewrite + Citation-Form Provenance

#### R3: Hydrate merge rule keyed on topic/section, rewritten as current truth
`src/kit/skills/fab-continue.md` Hydrate Behavior's "Merge without duplication" rule SHALL change its dedup key from *change name* to *topic/section*, and instruct hydrate to **rewrite the affected section as the new current truth** (removing superseded statements, not narrating them) rather than append or update-in-place a change-keyed delta entry. The rule MUST note the change's dated *what* is already captured once via `fab status set-summary` → `log.md` (the existing C-lite step), so the body carries no transition narration.

- **GIVEN** hydrate integrating a change's requirements/design into an existing memory file
- **WHEN** the target section already documents the topic
- **THEN** hydrate rewrites that section to state current truth (superseded text removed), never appends a "for change X …" delta entry

#### R4: Pattern-capture aligned to citation-form provenance
`src/kit/skills/fab-continue.md` Hydrate Behavior's pattern-capture step (item 6) SHALL drop the narration-inviting "with the change name for traceability" phrasing and instead direct provenance to citation form (a trailing `(change-id)` citation / the `*Introduced by*: {change-name}` field on the Design Decision), keeping traceability without inviting transition narration.

- **GIVEN** hydrate capturing a non-obvious implementation pattern in Design Decisions
- **WHEN** it records provenance
- **THEN** it uses the `*Introduced by*` field / a trailing citation, not free-text change-name narration

#### R5: `/docs-hydrate-memory` merge + description rules
`src/kit/skills/docs-hydrate-memory.md` SHALL gain the present-truth rewrite instruction on its ingest merge path (Step 3 item 4 "If target file exists → merge"), and the no-change-ids-in-`description:` rule on its description-authoring lines (ingest Step 3, and the shared authoring paragraph the generate/backfill modes reference). The rules MUST read consistently with the FKF §3.2/§3.3 amendments (R1/R2).

- **GIVEN** `/docs-hydrate-memory` merging into an existing file or authoring a `description:`
- **WHEN** it writes body content or a description one-liner
- **THEN** it rewrites the merged section to current truth (no change-keyed delta) and writes a change-id-free description

#### R6: Memory template guidance comments
`src/kit/templates/memory.md` SHALL state the present-truth body style in its guidance comments (present tense, no transition narration, no superseded-behavior description; provenance citations allowed), and its `description:` guidance comment SHALL note the no-change-ids rule. The `*Introduced by*: {change-name}` line on Design Decisions is KEPT (allowed provenance).

- **GIVEN** a memory writer filling `templates/memory.md`
- **WHEN** it reads the guidance comments
- **THEN** they instruct present-truth body style + change-id-free description, and the `*Introduced by*` line remains scaffolded

#### R7: `docs-reorg-memory` verification (assumption 9)
`src/kit/skills/docs-reorg-memory.md` SHALL be inspected during apply. It moves/merges whole files rather than authoring body content, but FKF §3.2 names it a `description:` author. Its description-authoring paths (the `removed-domains.md` `description:` one-liner and the new-topic-file/sub-domain-stub `description:` authoring in Step 5) SHALL carry the no-change-ids-in-`description:` rule if and only if that is where it authors descriptions. No body-style rewrite instruction is added (it authors no body entries). Update only what actually authors memory content.

- **GIVEN** `docs-reorg-memory` authoring a `description:` frontmatter one-liner (new file, sub-domain stub, or `removed-domains.md`)
- **WHEN** the no-change-ids rule applies to that authoring path
- **THEN** the skill notes the description stays change-id-free; no body-style rule is added where the skill authors no body

### Mirror-Class Sweep (Constitution-Required)

#### R8: SPEC mirrors for every edited skill file
Per the constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`"), each edited `src/kit/skills/*.md` SHALL have its `docs/specs/skills/SPEC-*.md` mirror updated in this change: `SPEC-fab-continue.md` and `SPEC-docs-hydrate-memory.md` always (their skills are edited), and `SPEC-docs-reorg-memory.md` only if `docs-reorg-memory.md` is edited (per R7's verification outcome).

- **GIVEN** an edited `src/kit/skills/X.md`
- **WHEN** apply completes
- **THEN** `docs/specs/skills/SPEC-X.md` reflects the present-truth / no-change-ids behavior added to X.md (and an unedited skill's mirror is untouched)

#### R9: Aggregate-spec restatements swept
Aggregate specs that restate the memory-body shape or the merge contract SHALL be checked and updated. `docs/specs/templates.md` § Memory File Format currently documents a stale pre-FKF `## Changelog` shape and a "Changelog row" hydration rule; it SHALL be brought current with FKF (no `## Changelog`; present-truth body; change-id-free description; merge = rewrite-as-current-truth). `docs/specs/skills.md`, `docs/specs/glossary.md`, and `docs/specs/architecture.md` SHALL be checked; update any that restate the merge contract / memory-body shape (verification found none requiring change, but the check is required).

- **GIVEN** an aggregate spec that restates the memory-body shape or merge contract
- **WHEN** apply completes
- **THEN** it no longer describes the change-keyed / `## Changelog` model and matches the FKF present-truth rules

#### R10: Repo-wide old-claim grep before finishing
Before finishing apply, the old claims SHALL be grepped repo-wide (e.g. "by change name", "with the change name for traceability", "referencing this change", "change-keyed") and every occurrence in the sweep class (skills, templates, specs) updated. Memory-file occurrences (`docs/memory/**`) are hydrate's responsibility and are recorded for hydrate, not rewritten during apply.

- **GIVEN** the old change-name-keyed phrasings
- **WHEN** apply nears completion
- **THEN** a repo-wide grep confirms no stale occurrence remains in the kit-source/spec sweep class (docs/memory/ deferred to hydrate)

### Non-Goals

- The `docs-distill-memory` corpus-cleanup skill (deferred follow-up NEW skill, built after these writer fixes land).
- Rewriting the existing memory corpus beyond files this change's own hydrate touches.
- Extending `fab memory-index` validation to enforce the rules (no Go/CLI change).
- Any `fkf_version` bump (backward-compatible authoring convention within v0.1; a bump would require a Go change).
- Editing `docs/memory/**` topic files during apply — memory updates are hydrate's stage.

### Design Decisions

1. **Provenance citations kept, transition narration banned** — *Why*: citations are cheap (6 chars) and proven to defend deliberate behavior against future "fixes" (the Copilot poll-predicate precedent); only the description of superseded state is banned. — *Rejected*: zero-provenance (ban change-ids entirely) — loses the defense.
2. **Fix writers + spec first, corpus cleanup later** — *Why*: cleanup without writer fixes is a treadmill (hydrate resumes producing deltas). — *Rejected*: cleanup-only; distill as a mode of `docs-reorg-memory` (not discoverable).
3. **`docs/specs/templates.md` § Memory File Format brought fully current** — it still described the pre-FKF `## Changelog` shape and a "Changelog row" hydration rule, which both restate the memory-body shape/merge contract this change amends; leaving it would ship a stale aggregate mirror that reviewers read strictly. — *Rejected*: touching only the present-truth lines — would leave the `## Changelog` residue self-contradicting the FKF spec it mirrors.

## Tasks

### Phase 1: FKF Spec Amendments (both files)

- [x] T001 Amend `docs/specs/fkf.md` §3.2 with the no-change-ids-in-`description:` clarification (anchors preserved) <!-- R2 --> <!-- rework cycle 2 RESOLVED: dropped "history" from the §3.2 cap enumeration ("requirements, design decisions, prose") byte-identically in docs/specs/fkf.md:101 + src/kit/reference/fkf.md:73; §3.2 shared block re-diffed IDENTICAL -->
- [x] T002 Amend `docs/specs/fkf.md` §3.3 with the present-truth body-style rule (anchors preserved) <!-- R1 -->
- [x] T003 Amend `src/kit/reference/fkf.md` §3.2 with the same no-change-ids clarification (anchors preserved) <!-- R2 -->
- [x] T004 Amend `src/kit/reference/fkf.md` §3.3 with the same present-truth body-style rule (anchors preserved) <!-- R1 -->

### Phase 2: Memory-Writer Fixes

- [x] T005 Rewrite `src/kit/skills/fab-continue.md` Hydrate "Merge without duplication" (item 4) → topic/section-keyed rewrite-as-current-truth <!-- R3 --> <!-- rework cycle 2 RESOLVED: (a) added "free of change-ids" to BOTH description clauses in the fab-continue.md FKF-frontmatter bullet (create-path cap clause + update-existing clause), mirroring docs-hydrate-memory.md, and recorded the §3.2 ban in SPEC-fab-continue.md's 260717-3plm entry. (b) fab-continue.md "the C-lite step below" → "the C-lite step above" (the set-summary bullet sits above the merge bullet) -->
- [x] T006 Align `src/kit/skills/fab-continue.md` Hydrate pattern-capture (item 6) to citation-form provenance <!-- R4 -->
- [x] T007 Add present-truth rewrite + no-change-ids-in-description rules to `src/kit/skills/docs-hydrate-memory.md` (ingest merge path + description-authoring lines) <!-- R5 -->
- [x] T008 Add present-truth body-style + no-change-ids guidance to `src/kit/templates/memory.md` comments (keep `*Introduced by*`) <!-- R6 --> <!-- rework cycle 2 RESOLVED: dropped "history" from the template's description guidance comment (memory.md:5) → "detail (requirements, design decisions, prose) belongs in the BODY sections below"; now agrees with the BODY STYLE comment below it -->
- [x] T009 Inspect `src/kit/skills/docs-reorg-memory.md`; add no-change-ids-in-description rule to its description-authoring paths only if it authors descriptions; no body-style rule <!-- R7 -->

### Phase 3: Mirror-Class Sweep

- [x] T010 Update `docs/specs/skills/SPEC-fab-continue.md` to reflect the topic-keyed rewrite-as-current-truth merge + citation-form pattern-capture <!-- R8 -->
- [x] T011 Update `docs/specs/skills/SPEC-docs-hydrate-memory.md` to reflect present-truth merge + no-change-ids-in-description <!-- R8 -->
- [x] T012 [P] Update `docs/specs/skills/SPEC-docs-reorg-memory.md` ONLY if `docs-reorg-memory.md` was edited in T009 <!-- R8 -->
- [x] T013 Bring `docs/specs/templates.md` § Memory File Format current with FKF (strip `## Changelog` shape + "Changelog row" rule; add present-truth body + change-id-free description + rewrite-as-current-truth merge) <!-- R9 -->
- [x] T014 [P] Check `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md` for merge-contract/memory-body restatements; update any that carry them <!-- R9 --> <!-- rework: review found skills.md:186 still names the old contract ("merged into existing memory files without duplication") — align to "merged as current truth"; also align docs-hydrate-memory.md:33 mode-summary line (should-fix) -->

### Phase 4: Verification

- [x] T015 Grep old claims repo-wide ("by change name", "with the change name for traceability", "referencing this change", "change-keyed"); confirm no stale occurrence remains in the kit-source/spec sweep class (docs/memory/ deferred to hydrate) <!-- R10 --> <!-- rework: review found 2 must-fix residuals the greps missed — src/kit/skills/fab-continue.md:252 Key Properties Idempotent? row ("existing entries for this change are updated in place") and docs/specs/skills/SPEC-fab-continue.md:136-137 HYDRATE STAGE flow-diagram box ("merge without duplication — existing entries for this change updated in place"); reword both to the topic-keyed merge-as-current-truth contract and widen the grep (e.g. "for this change") -->
- [x] T016 Cross-check the two FKF files are byte-consistent on the §3.2/§3.3 additions (dual-update rule) <!-- R1, R2 --> <!-- rework: review verified 2 byte-diffs in the shared normative text — (1) §3.2 trailing note: dev "(This clarifies the routing-signal rationale above; no enforcement is added — `fab memory-index` does not validate against it.)" vs extract "No enforcement is added — `fab memory-index` does not validate against it."; (2) §3.3 Provenance-is-citation-only bullet: final "(Citations are deliberately preserved …)" sentence parenthesized in dev, unparenthesized in extract. Align the SHARED normative text byte-for-byte (dev-only "Why present-truth" rationale blockquote stays dev-only per the extract charter) -->

## Execution Order

- T001–T004 (FKF spec text) precede the writer/mirror edits so the writers and mirrors can cite the amended §3.2/§3.3 consistently.
- T009 outcome gates T012 (SPEC-docs-reorg-memory only if the skill was edited).
- T015/T016 run last (whole-tree verification).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `docs/specs/fkf.md` §3.3 and `src/kit/reference/fkf.md` §3.3 both carry the present-truth body-style rule (present tense, no transition narration, no superseded-behavior description, citation-only provenance, rationale-preserved), anchors preserved
- [x] A-002 R2: `docs/specs/fkf.md` §3.2 and `src/kit/reference/fkf.md` §3.2 both carry the no-change-ids-in-`description:` clarification, anchors preserved
- [x] A-003 R3: `fab-continue.md` Hydrate merge rule is keyed on topic/section and instructs rewrite-as-current-truth (no change-keyed delta)
- [x] A-004 R4: `fab-continue.md` Hydrate pattern-capture uses citation-form provenance, not "with the change name for traceability"
- [x] A-005 R5: `docs-hydrate-memory.md` carries the present-truth merge instruction and the no-change-ids-in-description rule
- [x] A-006 R6: `templates/memory.md` guidance comments state present-truth body style + change-id-free description; `*Introduced by*` line retained
- [x] A-007 R7: `docs-reorg-memory.md` inspected; no-change-ids rule added to its description-authoring paths iff it authors descriptions; no body-style rule added
- [x] A-008 R8: SPEC mirrors updated for every edited skill file (`SPEC-fab-continue.md`, `SPEC-docs-hydrate-memory.md`; `SPEC-docs-reorg-memory.md` iff its skill was edited)
- [x] A-009 R9: `docs/specs/templates.md` § Memory File Format is FKF-current (no `## Changelog` shape/rule; present-truth body; change-id-free description); `skills.md`/`glossary.md`/`architecture.md` checked

### Behavioral Correctness

- [x] A-010 R3: The merge contract described in `fab-continue.md`, `docs-hydrate-memory.md`, and their SPEC mirrors reads consistently (topic-keyed rewrite-as-current-truth), with no residual "by change name" dedup phrasing in the kit-source/spec sweep class
- [x] A-011 R1: The present-truth rule preserves rationale — Design Decisions `Why`/`Rejected` remain durable present-tense intent (a rejected alternative is a design fact, not narration)

### Scenario Coverage

- [x] A-012 R10: Repo-wide grep of the old claims shows no stale occurrence in `src/kit/` or `docs/specs/` (docs/memory/ occurrences recorded for hydrate)
- [x] A-013 R1: The two FKF files are byte-consistent on the §3.2/§3.3 additions (dual-update rule satisfied)


### Edge Cases & Error Handling

- [x] A-014 R6: The `*Introduced by*: {change-name}` provenance line is explicitly KEPT (not removed as "narration") — the allowed-provenance carve-out holds
- [x] A-015 R7: No body-style rewrite instruction is added to `docs-reorg-memory.md` (it authors no body entries) — only description-authoring paths, if any, are touched

### Code Quality

- [x] A-016 Pattern consistency: New prose follows the surrounding FKF/skill/spec voice and structure (RFC-2119 phrasing in normative blocks, blockquote asides where the files use them)
- [x] A-017 No unnecessary duplication: The normative rule is single-sourced per the FKF dual-file contract (dev spec + shipped extract) and cited (not re-described) by the writer skills and mirrors — compressed point-of-use restatements with `(FKF §3.2/§3.3)` citations match the existing 500-char-cap house pattern
- [x] A-018 Canonical source only: All kit edits are under `src/kit/` (no `.claude/skills/` deployed-copy edits)
- [x] A-019 Markdown-only artifacts: All edits are CommonMark markdown; no binary/proprietary formats

### Documentation Accuracy

- [x] A-020 R8 R9: Every edited skill file has its SPEC mirror updated in the same change; aggregate specs restating the merge contract / memory-body shape are current with the FKF amendments (documentation_accuracy + cross_references checklist categories) <!-- mirrors updated + templates.md current; the SPEC-fab-continue.md flow-diagram residual is now reworded (T015) — A-010/A-012 met -->


## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- No Go/CLI changes, no migration, no `fkf_version` bump, no corpus rewrite (deferred `docs-distill-memory`).

## Deletion Candidates

<!-- All prior candidates are resolved:
     - cycle-1: fab-continue.md:252 Key Properties row; SPEC-fab-continue.md:136-137 HYDRATE flow box
       — reworded to the topic-keyed merge-as-current-truth contract.
     - cycle-2: the "history" enumeration token in the §3.2 cap paragraph (docs/specs/fkf.md:101 +
       src/kit/reference/fkf.md:73) and in the template's description guidance comment
       (src/kit/templates/memory.md:5) — dropped ("requirements, design decisions, prose") so the
       §3.2 enumeration no longer contradicts the new §3.3 no-history-in-body rule; FKF fix landed
       byte-identically in both files. No open candidates remain. -->

- *(none — all resolved above)*

## Assumptions

<!-- Graded SRAD decisions made while co-generating Requirements/Tasks/Acceptance.
     The intake pinned all 13 scope decisions; the rows below are apply-time
     refinements within that decided scope. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `docs/specs/templates.md` § Memory File Format is brought fully FKF-current (strip the stale `## Changelog` block + "Changelog row" hydration rule, add present-truth/change-id-free/rewrite-as-current-truth), not just the present-truth line | It is an aggregate spec restating the memory-body shape + merge contract (intake §5 sweep class); leaving the pre-FKF `## Changelog` residue would ship a mirror that self-contradicts the FKF spec — reviewers read the sweep class strictly (code-review.md) | S:80 R:85 A:80 D:80 |
| 2 | Confident | `skills.md`/`glossary.md`/`architecture.md` need no content change (checked; the only hit, glossary's "plan.md (change-scoped)", is accurate) but the check itself is a required task (T014) | Grep during apply found no merge-contract/memory-body restatement in these three; the constitution requires the sweep to cover the whole class, so the check is recorded as a task even when it yields no edit | S:75 R:90 A:80 D:75 |
| 3 | Confident | Memory-file occurrences of the old claims (`docs/memory/pipeline/execution-skills.md`, `memory-docs/*`) are NOT rewritten during apply — they are hydrate's stage (intake lists them as `(modify)` at hydrate) | The pipeline separates apply (source/spec) from hydrate (docs/memory); rewriting memory during apply would double-write and violate the stage boundary — recorded for hydrate instead | S:80 R:85 A:85 D:80 |
| 4 | Confident | `docs-reorg-memory.md`'s description-authoring paths (`removed-domains.md` one-liner, new-file/sub-domain-stub `description:`) receive at most a light no-change-ids note; no body-style rewrite rule is added | Assumption 9 in the intake (verify at apply): reorg moves/merges files and authors only descriptions, not body entries — so only the §3.2 rule can apply, and only where it authors a description | S:70 R:90 A:75 D:75 |

4 assumptions (1 certain, 3 confident, 0 tentative).
