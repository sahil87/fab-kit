# Plan: Reorg-Memory Orchestrates Frontmatter Backfill for Pre-fab-kit Trees

**Change**: 260614-5ewp-reorg-memory-backfill-orchestration
**Intake**: `intake.md`

## Requirements

<!-- Requirements derived from intake.md. RFC 2119 keywords. This change is
     skill-prose orchestration only — no Go changes. Two skills (docs-reorg-memory,
     docs-hydrate-memory) plus a one-line note in docs-reorg-specs, and the three
     SPEC mirrors. -->

### docs-reorg-memory: Compatibility Detection

#### R1: Detect missing `description:` frontmatter on topic files
During its existing read-all-files diagnosis pass (Step 1), `/docs-reorg-memory` SHALL detect topic files (non-`index.md` `.md` files) that lack a `description:` frontmatter field, reusing `frontmatter.Field` semantics — a file with no frontmatter or no `description:` key counts as missing.

- **GIVEN** a `docs/memory/` tree where some topic files have no `description:` frontmatter
- **WHEN** `/docs-reorg-memory` runs its Step 1 read-all-files pass
- **THEN** it records each such topic file as a compatibility finding (missing frontmatter)
- **AND** files that already carry a `description:` field are not flagged

#### R2: Detect tombstone rows in existing hand-curated index files
`/docs-reorg-memory` SHALL detect tombstone rows: rows in the existing hand-curated index files whose `docs/memory/`-relative link target is absent on disk. Strikethrough syntax (`~~...~~`) SHALL be treated as a corroborating hint that raises confidence but is NOT required — un-struck tombstones are still caught. The signal SHALL be scoped to `docs/memory/`-relative link targets to avoid false positives on intentional external links. Detected tombstone candidates SHALL be surfaced for user confirmation before any relocation.

- **GIVEN** an index file row links to a `docs/memory/`-relative path that does not exist on disk
- **WHEN** `/docs-reorg-memory` scans the existing index files during diagnosis
- **THEN** that row is recorded as a tombstone candidate (whether or not it is struck through)
- **AND** a row whose link target is an external URL or an existing on-disk file is NOT flagged
- **AND** tombstone candidates are presented for explicit user confirmation before relocation

#### R3: Detect custom structural groupings beyond the domains-only table
`/docs-reorg-memory` SHALL detect custom structural headings in the existing root `index.md` beyond the generated domains-only table (e.g., `### Apps`, `### Packages`, `### Cross-cutting`) — groupings the domains-only regen will flatten.

- **GIVEN** the existing root `index.md` contains structural headings beyond a domains-only table
- **WHEN** `/docs-reorg-memory` diagnoses the tree
- **THEN** it records the custom groupings as a compatibility finding (will flatten on regen)

### docs-reorg-memory: Findings Report

#### R4: Surface compatibility findings in the approve-before-mutate report
When any compatibility finding (R1/R2/R3) is present, `/docs-reorg-memory` SHALL surface it in its existing approve-before-mutate findings report as a distinct "Compatibility (pre-fab-kit memory tree detected)" section, listing the counts and the proposed remediation steps (relocate tombstones, backfill frontmatter, rebalance + regenerate).

- **GIVEN** the diagnosis found missing frontmatter, tombstones, and/or custom groupings
- **WHEN** `/docs-reorg-memory` presents its findings report
- **THEN** the report contains a Compatibility section enumerating the findings and the proposed remediation
- **AND** when no compatibility findings exist, no Compatibility section is shown (the skill behaves exactly as before)

### docs-reorg-memory: On-Approval Orchestration

#### R5: Relocate tombstones into `docs/memory/_shared/removed-domains.md` (the one mechanical file)
On user approval, `/docs-reorg-memory` SHALL author exactly ONE content file — `docs/memory/_shared/removed-domains.md` — containing a `description:` frontmatter one-liner (so it round-trips through `fab memory-index`), an H1, and the confirmed tombstone rows lifted verbatim from the old index (preserving the change IDs that explain each removal). This is the ONLY per-file content-authoring action reorg performs; it is bounded to mechanical row relocation and explicitly NOT per-file description synthesis.

- **GIVEN** the user approved remediation and confirmed N tombstone rows
- **WHEN** `/docs-reorg-memory` performs orchestration
- **THEN** it writes/updates `docs/memory/_shared/removed-domains.md` with a `description:` frontmatter line, an H1, and the N tombstone rows copied verbatim (change IDs preserved)
- **AND** reorg authors no other memory content file (per-file descriptions route to hydrate's backfill)

#### R6: Merge-not-duplicate when `removed-domains.md` already exists (idempotency)
If `docs/memory/_shared/removed-domains.md` already exists, `/docs-reorg-memory` SHALL merge new tombstone rows into it without duplicating rows already present. Re-running reorg on an already-converted tree SHALL add no duplicate rows.

- **GIVEN** `docs/memory/_shared/removed-domains.md` already contains some tombstone rows
- **WHEN** `/docs-reorg-memory` relocates tombstones again
- **THEN** only genuinely new rows are appended; existing rows are not duplicated
- **AND** a second run with no new tombstones produces no change to the file

#### R7: Dispatch hydrate backfill as a general-purpose sub-agent (no manifest passed)
After relocating tombstones, `/docs-reorg-memory` SHALL dispatch `/docs-hydrate-memory`'s backfill mode as a general-purpose sub-agent (per `_preamble.md` § Subagent Dispatch — standard subagent context, the 5 project files). The dispatch prompt SHALL name the operation ("backfill this tree"), NOT pass a file manifest — hydrate re-scans `docs/memory/` independently (assumption #9). The dispatch SHALL signal that this is the reorg-dispatched (caller) form so backfill defers `fab memory-index` to reorg.

- **GIVEN** the user approved remediation and tombstone relocation has completed
- **WHEN** `/docs-reorg-memory` dispatches backfill
- **THEN** it spawns a general-purpose sub-agent running `/docs-hydrate-memory` backfill mode with the standard subagent context
- **AND** the dispatch prompt names the operation, passes no file list, and marks the call as reorg-dispatched (defer regen)

#### R8: Run rebalance + `fab memory-index` once at the end of orchestration
After the backfill sub-agent returns, `/docs-reorg-memory` SHALL run its existing rebalance (split/merge/flatten) logic and then `fab memory-index` exactly once — this is the single regeneration for the whole orchestration.

- **GIVEN** the backfill sub-agent has returned
- **WHEN** `/docs-reorg-memory` completes orchestration
- **THEN** it runs its existing rebalance logic, then `fab memory-index` once
- **AND** the index regeneration is not run by the backfill sub-agent (reorg owns the single regen)

#### R9: Approval gate — decline leaves the tree intact
Backfill and tombstone relocation SHALL run only on explicit user approval (reorg's existing posture). If the user declines, `/docs-reorg-memory` SHALL report the compatibility findings and stop without mutating any file — the user keeps their hand-curated tree intact.

- **GIVEN** compatibility findings were surfaced
- **WHEN** the user declines the proposed remediation
- **THEN** `/docs-reorg-memory` reports the findings and stops without writing `removed-domains.md`, without dispatching backfill, and without running `fab memory-index`

### docs-hydrate-memory: Backfill Mode

#### R10: Add backfill as a third mode alongside ingest and generate
`/docs-hydrate-memory` SHALL support a third mode — **backfill** — beside ingest and generate, reachable both when invoked directly by a user over an existing tree whose topic files lack `description:` frontmatter, and when dispatched as reorg's sub-agent. The mode-routing prose / Argument Classification SHALL be updated so backfill is reachable and unambiguous relative to ingest and generate (generate **creates** files from source-code gaps; backfill **adds frontmatter to existing** memory files).

- **GIVEN** an existing `docs/memory/` tree whose topic files lack `description:` frontmatter
- **WHEN** `/docs-hydrate-memory` is invoked in backfill mode (directly or via reorg dispatch)
- **THEN** it routes to backfill behavior, distinct from ingest and generate
- **AND** the routing prose makes the backfill trigger unambiguous against ingest/generate

#### R11: Backfill re-scans `docs/memory/` independently (no caller manifest)
Backfill mode SHALL re-scan `docs/memory/` itself to find every topic file (non-`index.md` `.md`) lacking a `description:` field — it SHALL NOT receive a file list from its caller. This holds for both the direct-user form and the reorg-dispatched form (assumption #9).

- **GIVEN** backfill mode is entered (directly or via reorg dispatch with no file list)
- **WHEN** backfill discovers files to process
- **THEN** it walks `docs/memory/` itself and selects topic files missing `description:`
- **AND** it does not depend on a manifest from the caller

#### R12: Synthesize `description:` from the file's own content, body-preserving
For each discovered topic file missing `description:`, backfill SHALL synthesize a one-line `description:` from the file's own content (Overview / first section / H1). Where an existing curated index row maps file-by-file to the file, backfill SHOULD prefer the curated row text as the source (higher quality than re-synthesis). Backfill SHALL write the `description:` as the leading frontmatter block and SHALL preserve the file body byte-for-byte — it only prepends/edits frontmatter, never the content.

- **GIVEN** a topic file missing `description:` with content (Overview/H1) and possibly a matching curated index row
- **WHEN** backfill processes it
- **THEN** it writes a one-line `description:` frontmatter block (preferring matching curated index-row text when it maps file-by-file) and leaves the body unchanged byte-for-byte

#### R13: Idempotent — skip files already carrying `description:`
Backfill SHALL skip any topic file that already has a `description:` field. A second backfill pass over an already-converted tree SHALL be a no-op (no frontmatter rewrites, no body changes).

- **GIVEN** a tree where some files already have `description:` and some do not
- **WHEN** backfill runs
- **THEN** only the files missing `description:` are touched; already-described files are skipped
- **AND** a second run after the first produces no further changes

#### R14: Create missing domain/sub-domain `index.md` description-only stubs
Backfill SHALL create any missing domain/sub-domain `index.md` description-only stub the same way ingest/generate modes do (the stub-before-index pattern per the Index Ownership model), so `fab memory-index` has the domain description to read.

- **GIVEN** a domain or sub-domain folder lacks an `index.md` (or its `index.md` lacks `description:`)
- **WHEN** backfill runs
- **THEN** it creates/sets a `description:`-only `index.md` stub for that folder before any index regeneration

#### R15: Caller-aware regen deferral
Backfill SHALL be caller-aware: when dispatched by reorg, it SHALL NOT run `fab memory-index` (reorg runs it once at the end). When invoked directly by a user, it SHALL run `fab memory-index` as its final step like the other modes. The mode SHALL learn its caller from the dispatch prompt (a flag/prefix passed by reorg).

- **GIVEN** backfill was dispatched by reorg (caller signal present)
- **WHEN** backfill finishes synthesizing frontmatter and stubs
- **THEN** it does NOT run `fab memory-index` (defers to reorg)
- **GIVEN** backfill was invoked directly by a user (no reorg caller signal)
- **WHEN** backfill finishes
- **THEN** it runs `fab memory-index` as the final step

### docs-reorg-specs: No-Symmetry Note

#### R16: Add an explicit no-backfill/compatibility note for specs
`/docs-reorg-specs` SHALL carry a one-line note stating that no backfill/compatibility step applies to specs, with the rationale: there is no specs-index generator, the specs index is hand-rewritten, and constitution VI keeps specs human-curated. This prevents a future contributor from "fixing the asymmetry."

- **GIVEN** a contributor reads `docs-reorg-specs.md`
- **WHEN** they consider whether a compatibility/backfill step is missing relative to docs-reorg-memory
- **THEN** an explicit note tells them no such step applies to specs and why (no generator; hand-curated index; constitution VI)

### SPEC Mirrors

#### R17: Update SPEC mirrors for every changed skill
The SPEC mirror files SHALL be updated to reflect the skill changes, matching each mirror's existing structure/format: `docs/specs/skills/SPEC-docs-reorg-memory.md` (compatibility detection + orchestration + tombstone guard), `docs/specs/skills/SPEC-docs-hydrate-memory.md` (backfill mode), `docs/specs/skills/SPEC-docs-reorg-specs.md` (no-symmetry note).

- **GIVEN** `docs-reorg-memory.md`, `docs-hydrate-memory.md`, and `docs-reorg-specs.md` were edited
- **WHEN** the change is complete
- **THEN** each corresponding `SPEC-*.md` mirror reflects the new behavior in its established format (Summary/Flow/tables)

### Non-Goals

- **No Go changes** — the `fab memory-index` generator, `--check` flag, and `frontmatter.Field` already exist and are correct (assumption #2). This change touches no `src/go/` file.
- **No new fab subcommand and no migration file** — deferred (assumption #8); a future migration could orchestrate these same two skills.
- **No specs-index generator and no spec-frontmatter backfill** — specs are human-curated; out of scope (assumption #1).
- **Direct-user backfill does NOT detect/relocate tombstones** — tombstone detection/relocation is reorg's structural concern; backfill stays a pure frontmatter operation (resolves the intake's open question; see ## Assumptions row 11).

### Design Decisions

1. **reorg orchestrates, hydrate synthesizes**: reorg detects + relocates the one mechanical file (`removed-domains.md`) + dispatches; hydrate's backfill mode owns per-file description synthesis. — *Why*: preserves the restructure/author competence seam, mirroring how `/fab-proceed` orchestrates sub-skills without absorbing their logic. — *Rejected*: backfill-synthesis-in-reorg (crosses the seam — two skills both authoring content); a new top-level `/docs-make-compatible` skill (duplicates ~80% of hydrate; poor standing-command surface for a once-per-repo task).
2. **Loose seam — backfill re-scans, no manifest** (assumption #9): reorg dispatches "backfill this tree"; hydrate walks `docs/memory/` itself. — *Why*: two independently-invocable skills, idempotent, trivial double-scan cost, robust to drift. — *Rejected*: passing a manifest (tighter coupling, drift-prone).
3. **Tombstone auto-relocate over hard-block** (intake): one-command DX; the tradeoff (reorg authors one mechanical file) is acceptable because relocating existing rows is mechanical movement, not prose synthesis. — *Rejected*: warn-only (silent irreversible loss); hard-block hand-off (three invocations, not one command).
4. **Tombstone detection = unresolved `docs/memory/`-relative link target (primary) + strikethrough hint + user confirmation** (assumption #10): catches struck and un-struck tombstones; relative-path scope avoids external-link false positives; confirmation gates relocation.

## Tasks

<!-- All edits target src/kit/skills/*.md (canonical source) — NEVER .claude/skills/.
     Markdown-only change. SPEC mirrors updated in the same change (constitution). -->

### Phase 1: docs-hydrate-memory backfill mode (the synthesis muscle reorg calls)

- [x] T001 In `src/kit/skills/docs-hydrate-memory.md`, update the Purpose mode list and the `## Argument Classification & Mode Routing` section to introduce backfill as a third mode (alongside ingest/generate), making its trigger unambiguous: backfill = existing tree whose topic files lack `description:` (direct-user form + reorg-dispatched form), distinct from generate (creates files from code gaps) <!-- R10 -->
- [x] T002 In `src/kit/skills/docs-hydrate-memory.md`, add a `## Backfill Mode Behavior` section: re-scan `docs/memory/` independently (no caller manifest) to find topic files missing `description:`; synthesize a one-line `description:` from the file's own content, preferring a file-by-file-mapping curated index row; write frontmatter as the leading block, body byte-preserved; skip files already having `description:` (idempotent); create missing domain/sub-domain `description:`-only `index.md` stubs (stub-before-index per Index Ownership) <!-- R11 --> <!-- R12 --> <!-- R13 --> <!-- R14 -->
- [x] T003 In `src/kit/skills/docs-hydrate-memory.md` Backfill Mode Behavior, specify caller-aware regen deferral: when dispatched by reorg (caller signal in the dispatch prompt) do NOT run `fab memory-index`; when invoked directly by a user, run it as the final step like the other modes <!-- R15 -->
- [x] T004 In `src/kit/skills/docs-hydrate-memory.md`, extend `## Idempotency` (and `## Output` / `## Error Handling` as needed) to cover backfill mode's no-op-second-pass and body-preservation guarantees <!-- R13 -->

### Phase 2: docs-reorg-memory detection + findings + orchestration

- [x] T005 In `src/kit/skills/docs-reorg-memory.md` Step 1 (Read All Memory Files), fold in compatibility detection: missing `description:` frontmatter on topic files (`frontmatter.Field` semantics); tombstone rows (index row whose `docs/memory/`-relative link target is absent on disk — primary signal; strikethrough `~~...~~` corroborating hint; user-confirmed); custom structural groupings in the root `index.md` beyond the domains-only table <!-- R1 --> <!-- R2 --> <!-- R3 -->
- [x] T006 In `src/kit/skills/docs-reorg-memory.md`, add a `## Compatibility (pre-fab-kit memory tree detected)` findings-report block to the approve-before-mutate output (Step 3/4 region + Output section), surfacing the R1/R2/R3 counts and the proposed remediation; omit the block entirely when no compatibility findings exist <!-- R4 -->
- [x] T007 In `src/kit/skills/docs-reorg-memory.md` Step 5 (User Confirmation & Apply), add the on-approval compatibility orchestration in strict order: (1) relocate confirmed tombstones to `docs/memory/_shared/removed-domains.md` — reorg authors this ONE mechanical file (description: frontmatter + H1 + verbatim tombstone rows incl. change IDs), merge-not-duplicate if it exists; (2) dispatch `/docs-hydrate-memory` backfill mode as a general-purpose sub-agent (standard subagent context, names the operation, no manifest, reorg-dispatched/defer-regen signal); (3) rebalance + `fab memory-index` once at the end <!-- R5 --> <!-- R6 --> <!-- R7 --> <!-- R8 -->
- [x] T008 In `src/kit/skills/docs-reorg-memory.md`, make the approval gate explicit for compatibility remediation (decline = report findings and stop without mutating), and reflect the new sub-agent dispatch + the one-mechanical-file authoring in `## Key Properties` (the "Modifies memory files" / idempotency / sub-agent rows) and `## Output` <!-- R9 --> <!-- R5 --> <!-- R6 -->

### Phase 3: docs-reorg-specs no-symmetry note

- [x] T009 In `src/kit/skills/docs-reorg-specs.md`, add a one-line note (e.g., under Purpose or Reserved Paths) stating no backfill/compatibility step applies to specs, with the rationale: no specs-index generator, hand-rewritten index, constitution VI keeps specs human-curated <!-- R16 -->

### Phase 4: SPEC mirrors (constitution requirement — same change)

- [x] T010 [P] Update `docs/specs/skills/SPEC-docs-hydrate-memory.md` to reflect backfill mode (third mode in Summary; Flow branch; tools/sub-agents notes), matching the mirror's existing Summary/Flow/tables structure <!-- R17 -->
- [x] T011 [P] Update `docs/specs/skills/SPEC-docs-reorg-memory.md` to reflect compatibility detection + findings report + on-approval orchestration (tombstone relocation, hydrate backfill sub-agent dispatch, single end-of-run regen) and the new Sub-agents row, matching the mirror's existing structure <!-- R17 -->
- [x] T012 [P] Update `docs/specs/skills/SPEC-docs-reorg-specs.md` to record the no-symmetry note (no compatibility/backfill step for specs; rationale), matching the mirror's existing structure <!-- R17 -->

### Phase 5: Verify

- [x] T013 Confirm no `src/go/` files were touched; confirm all skill edits landed in `src/kit/skills/` (not `.claude/skills/`); run `just test` (or `go test ./...` in `src/go/fab`) as a regression guard even though no Go changed; spot-check idempotency wording across the two skills <!-- R13 --> <!-- R6 -->

## Execution Order

- Phase 1 (hydrate backfill) before Phase 2 (reorg orchestration), since reorg's dispatch contract (R7) references backfill's caller-signal and re-scan behavior — co-design the seam in hydrate first.
- T001 before T002–T004 (mode routing scaffolds the behavior section).
- T005 before T006 (detection produces what the findings report surfaces); T006/T007/T008 sequential within docs-reorg-memory (same file).
- Phase 4 mirrors after the skills they mirror (Phases 1–3); T010/T011/T012 are `[P]` (different files).
- T013 last.

## Acceptance

<!-- Declarative outcomes for review. Requirement-derived items name their R#. -->

### Functional Completeness

- [ ] A-001 R1: `docs-reorg-memory.md` Step 1 detects topic files missing `description:` frontmatter (frontmatter.Field semantics; already-described files not flagged)
- [ ] A-002 R2: `docs-reorg-memory.md` detects tombstone rows by unresolved `docs/memory/`-relative link target (primary), with strikethrough as a corroborating hint, scoped to relative paths, user-confirmed before relocation
- [ ] A-003 R3: `docs-reorg-memory.md` detects custom structural groupings in the root index beyond the domains-only table
- [ ] A-004 R4: `docs-reorg-memory.md` surfaces a Compatibility findings section (counts + proposed remediation) in its approve-before-mutate report, omitted when no findings exist
- [ ] A-005 R5: `docs-reorg-memory.md` on approval authors exactly one mechanical file `docs/memory/_shared/removed-domains.md` (description: frontmatter + H1 + verbatim tombstone rows incl. change IDs) and no other content file
- [ ] A-006 R6: `docs-reorg-memory.md` merges into an existing `removed-domains.md` without duplicating rows (idempotent re-run adds nothing)
- [ ] A-007 R7: `docs-reorg-memory.md` dispatches `/docs-hydrate-memory` backfill as a general-purpose sub-agent with standard subagent context, naming the operation (no manifest) and signaling reorg-dispatched/defer-regen
- [ ] A-008 R8: `docs-reorg-memory.md` runs rebalance + `fab memory-index` once at the end of orchestration (not in the sub-agent)
- [ ] A-009 R9: `docs-reorg-memory.md` runs remediation only on explicit approval; on decline it reports findings and mutates nothing
- [ ] A-010 R10: `docs-hydrate-memory.md` documents backfill as a third mode with unambiguous routing vs. ingest/generate
- [ ] A-011 R11: `docs-hydrate-memory.md` backfill re-scans `docs/memory/` independently (no caller manifest), both direct-user and reorg-dispatched
- [ ] A-012 R12: `docs-hydrate-memory.md` backfill synthesizes a one-line `description:` from file content (preferring a file-by-file curated index row), writes it as leading frontmatter, body byte-preserved
- [ ] A-013 R13: `docs-hydrate-memory.md` backfill skips files already having `description:`; second pass is a no-op
- [ ] A-014 R14: `docs-hydrate-memory.md` backfill creates missing domain/sub-domain `description:`-only `index.md` stubs (stub-before-index)
- [ ] A-015 R15: `docs-hydrate-memory.md` backfill is caller-aware — defers `fab memory-index` when reorg-dispatched, runs it when invoked directly
- [ ] A-016 R16: `docs-reorg-specs.md` carries the one-line no-backfill/compatibility note with rationale (no generator; hand-curated index; constitution VI)
- [ ] A-017 R17: All three SPEC mirrors (`SPEC-docs-reorg-memory.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-docs-reorg-specs.md`) are updated to reflect the changes in their established format

### Behavioral Correctness

- [ ] A-018 R7: The reorg→hydrate dispatch contract is consistent across both skill files — reorg's dispatch prompt (names operation, no manifest, defer-regen signal) matches hydrate's documented expectations (re-scan, caller-aware deferral)
- [ ] A-019 R15: The single-regen invariant holds end-to-end: reorg runs `fab memory-index` exactly once and the backfill sub-agent runs it zero times in the dispatched path

### Edge Cases & Error Handling

- [ ] A-020 R6: Idempotency (constitution III) — re-running reorg on an already-converted tree is a no-op: no duplicate tombstone rows, frontmatter-present files skipped by backfill, byte-stable index
- [ ] A-021 R2: Tombstone detection does not false-positive on intentional external links (signal scoped to `docs/memory/`-relative targets) and catches un-struck tombstones (strikethrough not required)

### Code Quality

- [ ] A-022 Pattern consistency: New skill prose follows the existing structure, tone, and section conventions of the edited files (Index Ownership model, stub-before-index pattern, Subagent Dispatch contract); SPEC mirrors match their existing Summary/Flow/table format
- [ ] A-023 No unnecessary duplication: Backfill reuses hydrate's existing Index Ownership / stub-before-index model rather than re-stating it; the competence seam is preserved (reorg does not duplicate hydrate's synthesis)
- [ ] A-024 Canonical source: All skill edits are in `src/kit/skills/*.md`; none in `.claude/skills/`; no `src/go/` file touched

### Documentation Accuracy

- [ ] A-025 Documentation accuracy: Skill prose and SPEC mirrors accurately describe the shipped behavior (e.g., `frontmatter.Field` semantics, domains-only root index, the strict orchestration order) with no claims contradicted by the Go generator

### Cross-References

- [ ] A-026 Cross-references: Cross-skill references resolve (reorg references hydrate's backfill mode by name; both reference the Index Ownership model and `_preamble.md` § Subagent Dispatch); no dangling references introduced

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Markdown-only change (skill prose + SPEC mirrors). No Go. Tests are a regression guard only.

## Assumptions

<!-- Graded SRAD decisions made while co-generating ## Requirements. Three grades
     only (Certain/Confident/Tentative). Rows 1-10 carried forward from intake;
     row 11 resolves the intake's remaining open question inline. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Specs are out of scope — no backfill/compatibility step for `docs-reorg-specs` (only the one-line no-symmetry note) | Verified: no specs-index generator; `docs-reorg-specs.md` hand-rewrites the index; constitution VI keeps specs human-curated. No compatibility contract to violate. | S:95 R:90 A:100 D:95 |
| 2 | Certain | No Go changes — generator, `--check`, `frontmatter.Field` already exist and are correct | Read `memoryindex.go` (RenderRoot domains-only; gatherFiles reads `frontmatter.Field(path,"description")`; Gather walks on-disk folders only). Change is skill-prose only. | S:90 R:85 A:100 D:95 |
| 3 | Certain | All skill edits target `src/kit/skills/`, never `.claude/skills/` | `context.md:9` + constitution: canonical source vs. gitignored deployed copies. | S:100 R:80 A:100 D:100 |
| 4 | Certain | SPEC mirrors updated for every changed skill, matching each mirror's existing format | Constitution: skill edits MUST update the corresponding `docs/specs/skills/SPEC-*.md`. | S:100 R:75 A:100 D:100 |
| 5 | Confident | reorg orchestrates (detect + relocate + dispatch); hydrate owns per-file synthesis | Intake-confirmed; preserves the restructure/author competence seam; mirrors `/fab-proceed` orchestration. | S:95 R:65 A:80 D:80 |
| 6 | Confident | Backfill is a mode of `/docs-hydrate-memory`, not a new top-level skill | Intake-confirmed; avoids ~80% logic duplication; once-per-repo task is poor standing-command surface. | S:95 R:70 A:85 D:85 |
| 7 | Certain | Backfill is body-preserving — only prepends/edits `description:` frontmatter | Intake-confirmed; idempotency (constitution III) + the convention (descriptions in frontmatter, body is the user's content). | S:95 R:75 A:90 D:85 |
| 8 | Confident | Migration file deferred — not part of this change | Intake-confirmed; heavier, only pays off at scale; a future migration could orchestrate these skills. | S:95 R:80 A:75 D:80 |
| 9 | Confident | Backfill re-scans `docs/memory/` independently — reorg dispatches "backfill this tree", no manifest passed | Intake-confirmed; loose seam between two independently-invocable skills; idempotent; trivial double-scan cost; robust to drift. | S:95 R:65 A:55 D:85 |
| 10 | Confident | Tombstone detection = index row whose `docs/memory/`-relative link target is absent on disk (primary), strikethrough `~~...~~` a corroborating hint, always user-confirmed | Intake-confirmed; catches struck + un-struck tombstones; relative-path scope avoids external-link false positives; confirmation gates relocation. | S:95 R:60 A:60 D:85 |
| 11 | Confident | Direct-user backfill does NOT detect/relocate tombstones — that stays reorg-only; backfill is a pure frontmatter operation | Resolves the intake's remaining Open Question ("leaning reorg-only, to be finalized at apply"). The competence seam (assumption #5) puts structural concerns in reorg; tombstone relocation requires authoring `removed-domains.md` (reorg's one mechanical file). Keeping backfill frontmatter-only preserves the body-preserving guarantee (#7) and avoids a second skill authoring structural content. Reversible — backfill could gain it later if a direct-user need emerges. | S:80 R:70 A:80 D:80 |

11 assumptions (5 certain, 6 confident, 0 tentative).
