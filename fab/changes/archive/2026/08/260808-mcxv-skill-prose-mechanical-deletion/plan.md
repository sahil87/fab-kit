# Plan: Skill Prose Mechanical Deletion (Phase 2)

**Change**: 260808-mcxv-skill-prose-mechanical-deletion
**Intake**: `intake.md`

## Requirements

### Skill corpus: present-truth prose

#### R1: Remove historical and transition narration
Canonical skill sources MUST delete obsolete command documentation and rewrite live instructions as present truth across every Phase 2a site identified by the intake and authoritative consolidation plan. The one-release legacy `spec.md` ingestion rule in `_generation.md` SHALL remain as the sole back-compat note, and live credibility anchors and anti-drift prohibitions SHALL remain intact.

- **GIVEN** the current `src/kit/skills/*.md` corpus
- **WHEN** historical markers and change-id archaeology are swept
- **THEN** obsolete narration is absent or expressed as current instruction
- **AND** the protected keep-list remains unchanged

#### R2: Collapse duplicated contracts to their owners
Consumer skills MUST point to the current owning contract instead of restating it. Dispatch consumers SHALL point to `_preamble.md` § CLI-Adapter Dispatch and keep only their local delta; `_pipeline.md` SHALL own one concise Stage Dispatch Procedure reused by its pipeline steps. `_preamble.md` itself MUST NOT be restructured in this phase.

- **GIVEN** the dispatch consumers `_pipeline.md`, `fab-continue.md`, `fab-adopt.md`, `fab-ff.md`, `fab-fff.md`, and `fab-proceed.md`
- **WHEN** their dispatch prose is consolidated
- **THEN** each consumer has one route to the current canon plus only site-specific instructions
- **AND** `fab dispatch reap` remains reachable through that canon

#### R3: Consolidate repeated operational prose
The corpus MUST collapse the Phase 2b duplication families for command logging, external-binary gates, operator coordination, ff/fff framing, documentation-family rules, and small workflow boilerplate without changing their behavior.

- **GIVEN** repeated prose named in Phase 2b of the authoritative plan
- **WHEN** the owning occurrence is retained and consumers are shortened
- **THEN** every load-bearing distinction remains available at its owner or through a direct pointer
- **AND** twin skills remain aligned

### Contract preservation and documentation

#### R4: Preserve protected contracts byte-identically
Every literal and fenced contract in the authoritative plan's § Must NOT be compressed MUST survive byte-identically. Heading-set changes MUST be limited to headings explicitly made obsolete by this phase; parser-contract headings MUST remain unchanged.

- **GIVEN** a before-edit baseline from HEAD
- **WHEN** the Phase 2 edits are complete
- **THEN** protected literals and fenced contracts compare byte-identically
- **AND** `## Requirements`, `## Tasks`, `## Acceptance`, and `## Deletion Candidates` remain intact wherever they govern parsing

#### R5: Keep SPEC mirrors and aggregate specs synchronized
Every modified `src/kit/skills/*.md` file MUST have its corresponding `docs/specs/skills/SPEC-*.md` mirror updated in the same change. The fab-ff/fab-fff twin and aggregate specs `docs/specs/skills.md`, `docs/specs/glossary.md`, and `docs/specs/architecture.md` MUST be swept for moved or renamed claims.

- **GIVEN** the final list of modified canonical skills
- **WHEN** mirror coverage is checked by basename
- **THEN** every modified skill has a modified matching SPEC file
- **AND** aggregate specs contain no stale ownership or dispatch wording

#### R6: Verify skill structure and prose-only scope
Verification MUST run `fab sync`, spot-load deployed copies, compare canonical
heading sets against the HEAD baseline, validate dispatch-pointer reachability,
sweep SPEC mirrors, and confirm the diff changes only canonical Markdown skills,
specs, and change artifacts. No `.claude/skills/` deployed copy SHALL be edited
directly; dev-repo canonical edits remain newer than the pinned release cache.

- **GIVEN** the completed canonical edits
- **WHEN** the phase verification suite runs
- **THEN** sync and deployed-copy inspection complete without a direct deployed edit
- **AND** the repository diff contains no behavioral implementation, test, migration, or memory changes

### Non-Goals

- Restructuring `_preamble.md`'s dispatch or model-resolution canon; that belongs to Phase 3.
- Adding the ownership convention as new policy; that belongs to Phase 4b.
- Changing any skill behavior, command grammar, runtime implementation, migration, test, or memory document.

## Tasks

### Phase 1: Baseline and inventory

- [x] T001 Record HEAD heading sets, protected literals, narration markers, dispatch restatements, and the exact target/mirror inventory before editing `src/kit/skills/*.md` and `docs/specs/skills/SPEC-*.md` <!-- R4 -->

### Phase 2: Mechanical deletion and collapse

- [x] T002 Remove Phase 2a narration from `src/kit/skills/_preamble.md`, `_generation.md`, `_cli-fab.md`, `fab-clarify.md`, `fab-switch.md`, and `fab-status.md`, retaining the legacy-ingestion note and protected contracts <!-- R1 --> <!-- rework: cycle-2 — grep-discovered site missed: src/kit/skills/_intake.md:116 carries "retired in 1.10.0" archaeology; rewrite as present truth and update docs/specs/skills/SPEC-_intake.md in the same pass -->
- [x] T003 [P] Rewrite the remaining Phase 2a sites in `src/kit/skills/fab-operator.md`, `fab-setup.md`, `git-pr.md`, `git-pr-review.md`, `docs-distill-memory.md`, and `docs-reorg-memory.md` as present truth <!-- R1 -->
- [x] T004 Collapse dispatch restatements in `src/kit/skills/_pipeline.md`, `fab-continue.md`, `fab-adopt.md`, `fab-ff.md`, `fab-fff.md`, and `fab-proceed.md` to one current-canon pointer plus local deltas <!-- R2 -->
- [x] T005 [P] Consolidate `fab log command` and `command -v` gate prose in `src/kit/skills/_preamble.md`, `_cli-external.md`, `fab-discuss.md`, `fab-setup.md`, `fab-switch.md`, `fab-dedupe.md`, `fab-operator.md`, and `fab-help.md` <!-- R3 -->
- [x] T006 Collapse remaining operator and ff/fff duplication in `src/kit/skills/fab-operator.md`, `_pipeline.md`, `fab-ff.md`, and `fab-fff.md` while preserving operator safety directives and driver-specific deltas <!-- R3 -->
- [x] T007 [P] Consolidate documentation-family repetition in `src/kit/skills/docs-hydrate-memory.md`, `docs-distill-memory.md`, and `docs-reorg-memory.md` <!-- R3 -->
- [x] T008 [P] Remove small duplicated boilerplate from `src/kit/skills/fab-new.md`, `fab-draft.md`, `fab-dedupe.md`, `fab-adopt.md`, `git-branch.md`, and `fab-archive.md` <!-- R3 -->

### Phase 3: Mirrors and verification

- [x] T009 Update every corresponding `docs/specs/skills/SPEC-*.md` mirror for the final touched-skill set and sweep `docs/specs/skills.md`, `docs/specs/glossary.md`, and `docs/specs/architecture.md` for stale moved or renamed claims <!-- R5 -->
- [x] T010 Run `fab sync`; compare heading sets and protected literals against HEAD; verify one dispatch pointer per consumer, reap reachability, SPEC mirror coverage, and prose-only file scope <!-- R6 --> <!-- rework: cycle-2 — re-run verification after the _intake.md fix; also re-grep the full 2a marker list across ALL of src/kit/skills/ (incl. _*.md partials) to confirm no other missed site -->

### Phase 4: Cycle-3 rework — restore over-deleted load-bearing content (revise plan)

<!-- Plan revision (rework cycle 3, after 2 consecutive fix-code failures): the original tasks
     never required checking deleted sections/anchors for LIVE REFERENCERS before deletion.
     T011–T013 restore the three over-deletions and T013 adds the missing referencer check. -->

- [x] T011 Restore a concise Auto Mode / `[AUTO-MODE]` protocol contract in `src/kit/skills/fab-clarify.md` — relocate/condense rather than delete (the move-over-delete decision is recorded in `docs/memory/pipeline/clarify.md`, and `_preamble.md` still references Auto Mode); synchronize `docs/specs/skills/SPEC-fab-clarify.md` and sweep `docs/specs/glossary.md` <!-- R1 R3 -->
- [x] T012 Restore the fenced help-dump JSON schema (recursive node fields: name, path, short, usage, text, commands) in `src/kit/skills/_cli-external.md` — fenced contract blocks are protected and survive byte-identically; deduplicate only the surrounding gate prose; synchronize `docs/specs/skills/SPEC-_cli-external.md` <!-- R4 R3 -->
- [x] T013 Referencer check for every deleted or renamed `##`/`###` section heading and named anchor in this diff: grep each old heading/anchor phrase repo-wide (docs/memory/, docs/specs/, src/kit/skills/); for each live referencer either preserve/restore the source anchor or correct the referencing pointer (the intake's pointer-correction allowance covers memory pointer fixes, e.g. `docs/memory/distribution/distribution.md` → `_cli-external.md` § Absent-binary discipline); then re-run the full T010 verification suite <!-- R6 -->

## Execution Order

- T001 blocks every editing task.
- T002–T003 establish the present-truth baseline before the duplication-collapse tasks.
- T004 blocks T006 because `_pipeline.md` becomes the shared ff/fff owner.
- T009 follows T002–T008 and uses the final touched-skill inventory.
- T011–T012 precede T013; T013 (which subsumes the T010 re-verification) is the final task.

## Acceptance

### Functional Completeness

- [x] A-001 R1: All known and grep-discovered Phase 2a narration sites are deleted or expressed as present truth, with the legacy `spec.md` ingestion note and keep-list intact.
- [x] A-002 R2: Every dispatch consumer has one current-canon route and only its local delta; `_pipeline.md` owns one shared Stage Dispatch Procedure.
- [x] A-003 R3: Command logging, binary gates, operator prose, ff/fff framing, docs-family rules, and small boilerplate are consolidated without semantic loss. <!-- review cycle 3: the supported fab-clarify Auto Mode contract and recursive help-dump schema were deleted. -->
- [x] A-004 R4: Protected literals and parser-contract headings survive byte-identically against the HEAD baseline. <!-- review cycle 3: the fenced help-dump JSON schema did not survive the consolidation. -->
- [x] A-005 R5: Every modified canonical skill has a modified matching SPEC mirror, twins align, and aggregate specs contain no stale claim.
- [x] A-006 R6: `fab sync` succeeds, deployed copies are spot-loaded, canonical heading deltas are intentional, and the diff is prose-only within authorized paths.

### Behavioral Correctness

- [x] A-007 R2: Dispatch consumers still reach start/wait/five-state handling/reap/no-state-cleanup, recovery, pane mode, prompt obligations, and the transition-ownership carve-out through `_preamble.md`.
- [x] A-008 R3: The fail-silent versus stop-with-install-hint distinction for external tools and all operator safety boundaries remain explicit.

### Scenario Coverage

- [x] A-009 R4: Before/after literal checks cover output tokens, result schemas, SRAD formula and markers, parser IDs/headings, Copilot/GraphQL facts, approval gates, anti-drift prohibitions, ladders, and ecosystem test-path mapping.
- [x] A-010 R5: A basename sweep proves mirror coverage for the complete touched-skill set, including fab-ff/fab-fff and partials.

### Edge Cases & Error Handling

- [x] A-011 R1: Change-id provenance tied to a live regression test or don't-re-break force remains, while purely historical archaeology is removed.
- [x] A-012 R6: No direct `.claude/skills/` edit, Go change, test change, or migration is present in the tracked diff; memory content is unchanged except pointer corrections to moved/renamed skill anchors, which the intake explicitly allows.

### Removal Verification

- [x] A-018 R1: `fab-clarify.md` carries a live (concise or relocated) Auto Mode / `[AUTO-MODE]` contract; every `_preamble.md` reference to it resolves; SPEC-fab-clarify and the glossary agree.
- [x] A-019 R4: The fenced help-dump JSON schema in `_cli-external.md` survives byte-identically (all recursive node fields present).
- [x] A-020 R6: Every section heading or anchor deleted/renamed by this diff has zero dead referencers repo-wide (docs/memory/, docs/specs/, src/kit/skills/) — each live referencer either still resolves to the source anchor or was pointer-corrected.

### Code Quality

- [x] A-013 Pattern consistency: Markdown remains standard CommonMark and follows neighboring skill structure.
- [x] A-014 No unnecessary duplication: Each consolidated rule is stated at one owner and consumers use pointers.
- [x] A-015 Readability and maintainability: Shortened prose retains enough local context to execute each workflow correctly. <!-- review cycle 3: fab-clarify no longer documents its supported machine-readable Auto Mode. -->
- [x] A-016 Skill-source discipline: Canonical skill edits are confined to `src/kit/skills/` and matching SPEC mirrors are updated.
- [x] A-017 Twin and aggregate sweep: `fab-ff`/`fab-fff` and relevant aggregate specs remain synchronized.

## Notes

- Review marks acceptance items after independently validating the completed apply diff.
- The authoritative plan's line references are content anchors only; all edits use HEAD content.

## Deletion Candidates

- None — this prose-only consolidation removes existing duplication without making implementation code, files, branches, or configuration redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The authoritative Phase 2 known-site lists define the minimum scope, and grep-discovered matches are included only under the same narration/duplication rules | The intake explicitly incorporates the full plan and requires a broader marker sweep | S:95 R:85 A:95 D:90 |
| 2 | Certain | `_generation.md` retains exactly one legacy `spec.md` ingestion compatibility note | The authoritative plan names this as the sole exception to the removed-stage narration sweep | S:100 R:80 A:100 D:100 |
| 3 | Certain | `_preamble.md` remains the current dispatch owner and is edited only for Phase 2a deletions/self-duplication, not structurally reorganized | Explicit user/intake sequencing boundary separates Phase 2 from Phase 3 | S:100 R:75 A:100 D:100 |
| 4 | Confident | SPEC mirrors should express the unchanged present design and ownership after source consolidation rather than add change-history annotations | Specs are pre-implementation design intent; change-id archaeology would conflict with the present-truth goal | S:85 R:90 A:85 D:75 |

4 assumptions (3 certain, 1 confident, 0 tentative).
