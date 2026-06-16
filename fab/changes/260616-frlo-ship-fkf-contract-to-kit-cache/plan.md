# Plan: Ship FKF Normative Contract to Kit Cache

**Change**: 260616-frlo-ship-fkf-contract-to-kit-cache
**Intake**: `intake.md`

## Requirements

<!-- Content-only distribution fix. Requirements derive from intake "What Changes" §1–§5.
     The "implementation" is markdown authoring + a deploy sync; there is no Go/binary surface. -->

### Distribution: Shipped FKF Normative Contract

#### R1: The FKF normative contract MUST ship inside the kit tree
A new file `src/kit/reference/fkf.md` SHALL exist under a new `reference/` directory (sibling to `templates/`, `migrations/`, `scaffold/`, `skills/`). It SHALL contain the **normative subset** of `docs/specs/fkf.md` — §2 (Conformance), §3 (Concept documents: §3.1 `type`, §3.2 `description`, §3.3 no-`## Changelog` body rule, §3.4 optional frontmatter), §5 (Index files — generated, stub-before-index), §6 (Log files — C-lite model, format, §6.3 `summary:` source), §7 (Cross-links — bundle-relative), §8 (Versioning — `fkf_version`) — and SHALL NOT contain §1, §4 prose rationale, §9, §10, or §11. The original §-anchors (§2, §3.1, §3.2, §3.3, §3.4, §5, §6, §6.3, §7, §8) SHALL be preserved verbatim so repointed citations resolve.

- **GIVEN** a user repo with the kit installed at `$(fab kit-path)`
- **WHEN** an agent opens `$(fab kit-path)/reference/fkf.md`
- **THEN** it finds the normative FKF rules under their original §-anchors, and finds none of the dev-repo-only rationale sections (§1/§4/§9/§10/§11)

#### R2: The shipped contract MUST carry a single-sourcing header note
`src/kit/reference/fkf.md` SHALL open with a short note declaring it the **shipped normative extract** of `docs/specs/fkf.md` (the dev-repo design doc), instructing that any change to FKF normative rules MUST update both files.

- **GIVEN** the shipped contract file
- **WHEN** a maintainer reads its top
- **THEN** the anti-drift instruction (update both files) is explicit

### Distribution: Anti-Drift Reciprocal Pointer

#### R3: `docs/specs/fkf.md` MUST name the shipped extract
`docs/specs/fkf.md` SHALL carry a one-line note near the top naming `src/kit/reference/fkf.md` as the shipped normative extract (the reciprocal mirror of R2). No other restructuring of the spec SHALL occur.

- **GIVEN** the dev-repo design doc `docs/specs/fkf.md`
- **WHEN** a maintainer reads its top
- **THEN** it points at `src/kit/reference/fkf.md` as the shipped extract, and the rest of the spec is byte-unchanged except that pointer

### Skills: Repointed Citations

#### R4: Every load-bearing FKF citation in skills MUST point at the shipped contract
All `docs/specs/fkf.md §X` citations across `src/kit/skills/` that are **user-reachable authority** SHALL be repointed to `$(fab kit-path)/reference/fkf.md §X`. The 5 source files: `fab-continue.md`, `docs-hydrate-memory.md` (3 citations), `docs-reorg-memory.md`, `docs-reorg-specs.md`, `_cli-fab.md`. No user-reachable `docs/specs/fkf.md` citation SHALL remain after the edit (a deliberate dev-repo-provenance mention, if any, is the only acceptable residual and MUST be called out).

- **GIVEN** the deployed skills in a user repo
- **WHEN** an agent follows an FKF citation
- **THEN** the path resolves to `$(fab kit-path)/reference/fkf.md` (which ships), not the never-shipped `docs/specs/fkf.md`

#### R5: The §9 citation in `docs-reorg-specs.md` MUST be rewritten onto Constitution VI
Because §9 (Non-Scope) is NOT in the shipped subset, the `docs-reorg-specs.md` citation SHALL drop the `§9` anchor and stand on **Constitution VI alone**. No §9 anchor SHALL point at the shipped contract, and no §9 stub SHALL ship in `reference/fkf.md`.

- **GIVEN** `docs-reorg-specs.md`'s "no FKF frontmatter on specs" note
- **WHEN** the citation is rewritten
- **THEN** it cites Constitution VI (which ships in every user repo via `fab/project/constitution.md`) with no §9 anchor, and `reference/fkf.md` contains no §9

#### R6: The `_cli-fab.md` "no memory-file template yet" line MUST be left untouched
Only the `docs/specs/fkf.md` citation in `_cli-fab.md` SHALL be repointed; the separate backlog-tracked line (`2fm8`) about there being no `type: memory` template is out of scope and SHALL NOT be edited.

- **GIVEN** `_cli-fab.md`
- **WHEN** the citation is repointed
- **THEN** the `2fm8` line is byte-unchanged

### Specs: SPEC-Mirror Updates

#### R7: The 4 SPEC mirrors MUST reflect the citation-target change
Per the constitution (a change to `src/kit/skills/*.md` MUST update the corresponding `docs/specs/skills/SPEC-*.md`), `SPEC-fab-continue.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-docs-reorg-memory.md`, and `SPEC-docs-reorg-specs.md` SHALL have their FKF citation references updated to match the repointed source skill (the shipped contract target; for `SPEC-docs-reorg-specs.md`, the §9→Constitution VI rewrite). `_cli-fab.md` is correctly excluded from SPEC mirrors (verified — no `SPEC-_cli-fab.md`). A mirror that does not reference the FKF citation SHALL NOT have one invented.

- **GIVEN** each of the 4 SPEC mirrors
- **WHEN** its source skill's FKF citation changed
- **THEN** the mirror's FKF reference matches the new target (or the §9→Constitution VI rewrite), and `_cli-fab.md` has no mirror to update

### Memory: Kit-Architecture Layout

#### R8: `kit-architecture.md` MUST record the new `reference/` shipped content dir
`docs/memory/distribution/kit-architecture.md` SHALL document `src/kit/reference/` as a new shipped content dir (in the directory-structure tree and the Replaced-on-upgrade list), and SHALL state the packaging invariant explicitly: the whole `src/kit/` tree is copied verbatim (`just install` rsyncs it; `just dist-kit` does `cp -a src/kit/.` then archives the whole tree), so new kit content ships with no Go/packaging change.

- **GIVEN** the kit-architecture memory file
- **WHEN** a reader looks up the kit tree layout
- **THEN** `reference/` appears as a shipped content dir and the verbatim-copy packaging invariant is stated

### Deployment

#### R9: Deployed skill copies MUST be re-synced after editing source skills
After editing `src/kit/skills/*.md`, `fab sync` SHALL be run so the deployed `.claude/skills/` copies reflect the repointed citations (canonical source is `src/kit/skills/`; `.claude/skills/` is never hand-edited — Constitution V).

- **GIVEN** edited `src/kit/skills/*.md` sources
- **WHEN** `fab sync` runs
- **THEN** the deployed `.claude/skills/<skill>/SKILL.md` copies carry the repointed `$(fab kit-path)/reference/fkf.md` citations

### Non-Goals

- No Go/binary change, no packaging/justfile/`release.yml` edit, no `src/kit/migrations/` file — packaging copies `src/kit/` verbatim, so a new file ships automatically (verified at intake).
- No change to `log.md` / `log.seed.md` shipping — correctly generated/curated in-repo.
- No removal/restructure of `docs/specs/fkf.md` beyond the single reciprocal pointer line.
- No fix for the "no `type: memory` template yet" gap (backlog `2fm8`).
- Hydrating memory files other than `kit-architecture.md` is the hydrate stage's job, not apply.

### Design Decisions

1. **`reference/` dir over `templates/`**: the FKF contract is reference-material-to-read, not an instantiable artifact — keeps `templates/` single-purpose. *Rejected*: dropping `fkf.md` into `templates/` (muddies that dir), shipping into the user's `docs/specs/` (violates Constitution VI), a deployed `_fkf.md` helper (can drift from the binary if not re-synced).
2. **Extract the normative subset, keep two files**: the shipped extract stays tight; the dev-repo design doc keeps rationale/history. Reciprocal single-sourcing notes are the anti-drift mitigation for the duplication.
3. **§9 stands on Constitution VI alone**: Constitution VI is the rule's real authority and already ships in every user repo, so dropping the §9 anchor loses nothing and avoids shipping a §9 stub.

## Tasks

### Phase 1: Setup

- [x] T001 Create the `src/kit/reference/` directory and author `src/kit/reference/fkf.md` — the normative subset (§2/§3.1/§3.2/§3.3/§3.4/§5/§6/§6.3/§7/§8) extracted faithfully from `docs/specs/fkf.md` with original §-anchors preserved, plus the single-sourcing header note (R2) <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 [P] Add the reciprocal single-sourcing pointer line near the top of `docs/specs/fkf.md` naming `src/kit/reference/fkf.md` as the shipped normative extract; no other restructuring <!-- R3 -->
- [x] T003 [P] Repoint the FKF citation in `src/kit/skills/fab-continue.md:195` (`docs/specs/fkf.md §3.1–§3.2`) to `$(fab kit-path)/reference/fkf.md §3.1–§3.2` <!-- R4 -->
- [x] T004 [P] Repoint the 3 FKF citations in `src/kit/skills/docs-hydrate-memory.md` (lines ~87, ~162, ~190) to `$(fab kit-path)/reference/fkf.md §X` (line 190's load-bearing §3.1 cite repointed; its separate `§10 item 2` migration-trajectory mention left as an explicit dev-repo-provenance reference) <!-- R4 -->
- [x] T005 [P] Repoint the FKF §7 citation in `src/kit/skills/docs-reorg-memory.md:16` to `$(fab kit-path)/reference/fkf.md §7` <!-- R4 -->
- [x] T006 [P] Repoint the FKF citation in `src/kit/skills/_cli-fab.md:468` to `$(fab kit-path)/reference/fkf.md`; leave the `2fm8` "no memory-file template yet" line untouched <!-- R4 R6 -->
- [x] T007 Rewrite the §9 citation(s) in `src/kit/skills/docs-reorg-specs.md` onto Constitution VI alone — drop the `§9` anchor; do not repoint a §9 anchor at the shipped contract (4 references rewritten: lines 9, 20, 83, 123–124) <!-- R5 -->

### Phase 3: SPEC Mirrors & Memory

- [x] T008 [P] Update `docs/specs/skills/SPEC-fab-continue.md` FKF citation reference to the shipped-contract target <!-- R7 -->
- [x] T009 [P] Update `docs/specs/skills/SPEC-docs-hydrate-memory.md` FKF citation reference to the shipped-contract target <!-- R7 -->
- [x] T010 [P] Update `docs/specs/skills/SPEC-docs-reorg-memory.md` FKF citation reference (heading + body §7) to the shipped-contract target <!-- R7 -->
- [x] T011 [P] Update `docs/specs/skills/SPEC-docs-reorg-specs.md` FKF §9 reference(s) to the §9→Constitution VI rewrite (3 references: lines 9, 22, 124) <!-- R7 -->
- [x] T012 Update `docs/memory/distribution/kit-architecture.md` — add `reference/` to the directory tree + Replaced list, and state the verbatim-copy packaging invariant <!-- R8 -->

### Phase 4: Deploy & Verify

- [x] T013 Run `fab sync` so `.claude/skills/` deployed copies reflect the repointed citations; verify it succeeds and spot-check one deployed skill (cache refreshed via `just install` first, since `fab sync` deploys from the version cache, not `src/kit/`) <!-- R9 -->
- [x] T014 Verify `grep -rn 'docs/specs/fkf\.md' src/kit/skills/` leaves no user-reachable citation (call out any deliberate dev-repo-provenance residual); confirm `src/kit/reference/fkf.md` exists with the §-anchored sections <!-- R4 -->

## Execution Order

- T001 blocks T003–T007 (citations must point at a file that exists) and T002 (reciprocal pointer references the new file)
- T003–T007 block T013 (sync deploys the edited sources)
- T008–T011 mirror T003/T004/T005/T007 respectively — author each mirror after its source skill is edited
- T013 blocks T014 (verification runs after sync)

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `src/kit/reference/fkf.md` exists, contains §2/§3.1/§3.2/§3.3/§3.4/§5/§6/§6.3/§7/§8 with original anchors preserved, and omits §1/§4/§9/§10/§11
- [ ] A-002 R2: `src/kit/reference/fkf.md` opens with a single-sourcing header note declaring it the shipped extract of `docs/specs/fkf.md` and instructing "update both"
- [ ] A-003 R3: `docs/specs/fkf.md` carries a one-line reciprocal pointer naming `src/kit/reference/fkf.md`; the rest of the spec is otherwise unchanged
- [ ] A-004 R4: every user-reachable `docs/specs/fkf.md §X` citation across the 5 skill files is repointed to `$(fab kit-path)/reference/fkf.md §X`
- [ ] A-005 R5: `docs-reorg-specs.md`'s §9 citation stands on Constitution VI with no §9 anchor; `reference/fkf.md` ships no §9
- [ ] A-006 R6: the `_cli-fab.md` `2fm8` "no memory-file template yet" line is byte-unchanged
- [ ] A-007 R7: the 4 SPEC mirrors reflect the citation-target change (shipped contract; §9→Constitution VI for docs-reorg-specs); `_cli-fab.md` has no mirror
- [ ] A-008 R8: `kit-architecture.md` records `reference/` as a shipped content dir and states the verbatim-copy packaging invariant
- [ ] A-009 R9: after `fab sync`, a spot-checked deployed `.claude/skills/<skill>/SKILL.md` carries the repointed citation

### Behavioral Correctness

- [ ] A-010 R4: `grep -rn 'docs/specs/fkf\.md' src/kit/skills/` returns no user-reachable citation (any residual is a deliberate provenance mention, explicitly noted)

### Edge Cases & Error Handling

- [ ] A-011 R1: the shipped extract's cross-references between its own sections (e.g. §3.3 → §6, §5 → §8) still resolve within the trimmed file (no dangling §-link to a dropped section)

### Code Quality

- [ ] A-012 Pattern consistency: the new citation form matches the existing `$(fab kit-path)/templates/...` read-path convention; CommonMark markdown only (Constitution IV)
- [ ] A-013 No unnecessary duplication: the shipped extract copies normative rules faithfully (no paraphrase) and the two files are kept in sync via the reciprocal notes rather than a third copy

### Documentation Accuracy

- [ ] A-014 R7: SPEC mirrors and kit-architecture text accurately describe the post-change state (citation target, shipped dir)

### Cross References

- [ ] A-015 R4 R7: repointed citations and SPEC-mirror references are internally consistent (same target form across source skill and its mirror)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

<!-- Apply-stage record of graded decisions; not a scoring source. Carried from the intake's
     resolved assumptions (all already Certain/Confident at intake — no new under-specified
     points arose during plan generation). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Shipped extract preserves the original §-anchors (§2/§3.1/§3.2/§3.3/§3.4/§5/§6/§6.3/§7/§8) so repointed citations resolve | Citations reference specific § numbers; preserving them is the low-risk default (intake #6) | S:80 R:80 A:90 D:85 |
| 2 | Confident | New `src/kit/reference/` dir rather than dropping fkf.md into `templates/` | Clarified with user; `templates/` is for instantiable artifacts, reference-to-read differs (intake #9) | S:95 R:65 A:70 D:60 |
| 3 | Confident | §9 citation in docs-reorg-specs.md rewritten onto Constitution VI; no §9 stub ships | Clarified with user; Constitution VI is the rule's real authority and ships in every user repo (intake #10) | S:95 R:70 A:65 D:55 |
| 4 | Certain | No Go/packaging/migration change — a new `src/kit/` file ships automatically | Verified: `just install` rsyncs `src/kit/` whole; `just dist-kit` does `cp -a src/kit/.` then archives the whole tree (intake #2) | S:90 R:85 A:100 D:95 |
| 5 | Confident | kit-architecture.md is the only memory file edited at apply; other memory hydration deferred to hydrate stage | Intake "Affected Memory" lists only kit-architecture (modify); "What Changes §5" treats it as an apply-time spec-level fact | S:85 R:75 A:85 D:80 |

5 assumptions (1 certain, 4 confident, 0 tentative).
