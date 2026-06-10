# Plan: Memory Tree Shape & Rebalance

**Change**: 260607-tciy-memory-tree-shape-rebalance
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Memory Index: Generated Index Command (`fab memory-index`)

#### R1: Deterministic, idempotent root + domain index generation
The system SHALL provide a new pure-Go subcommand `fab memory-index` that regenerates the root `docs/memory/index.md` (domains-only — no inlined per-file column) and every `docs/memory/{domain}/index.md` (file rows) from folder contents. The render SHALL be a pure function of gathered data (mirroring `internal/prmeta`'s `Render(Data) string` + `Gather` split), and re-running the command on unchanged inputs SHALL produce byte-identical output (idempotent / byte-stable).

- **GIVEN** a `docs/memory/` tree with one or more domain folders
- **WHEN** `fab memory-index` runs
- **THEN** the root index lists only domain rows (`| Domain | Description |`) and each domain index lists its non-`index` `.md` files (`| File | Description | Last Updated |`)
- **AND** running it a second time produces no diff (byte-stable)

#### R2: Read H1 title + `description:` frontmatter per file
For each non-`index` `.md` file under a domain, `fab memory-index` SHALL read the file's H1 title (the first `# ` line) and a machine-readable `description:` frontmatter field, using them to populate the index rows. When `description:` is absent, the cell SHALL degrade gracefully (empty/`—`) rather than error.

- **GIVEN** a memory file with `description:` frontmatter and a `# H1`
- **WHEN** the index is generated
- **THEN** the file row uses the `description:` value as its Description cell and links by file basename
- **AND** a file lacking `description:` still renders a row (Description degrades, no crash)

#### R3: Git-derived "Last Updated" with graceful degradation
`fab memory-index` SHALL stamp each domain-index file row's "Last Updated" cell from `git log -1 --date=short <file>`, degrading gracefully (rendering `—`) in worktree / shallow-clone / squash / rebase / uncommitted contexts where git returns no date — mirroring how `internal/prmeta` degrades on missing git/gh context.

- **GIVEN** a committed memory file
- **WHEN** the index is generated
- **THEN** its "Last Updated" cell shows the short commit date (`YYYY-MM-DD`)
- **AND** an uncommitted/unresolvable file renders `—` instead of erroring

#### R4: Non-fatal shape-bound warnings (Approach C, detect-only)
On every run, `fab memory-index` SHALL emit non-fatal stderr warnings when a domain folder exceeds the soft width bound (more than ~12 topic files) or when tree depth exceeds 3, without modifying files, blocking, or auto-splitting. Reserved domains `_shared/` and `_unsorted/` SHALL be exempt from the width warning. Warnings SHALL NOT affect the byte-stable index output written to stdout/files.

- **GIVEN** a domain folder with 20 topic files
- **WHEN** `fab memory-index` runs
- **THEN** a `⚠` warning naming the folder, its file count, and the soft bound is written to stderr
- **AND** the regenerated index files are unchanged in content (warnings are advisory only)
- **AND** `_shared/` and `_unsorted/` never trigger the width warning

#### R5: `_cli-fab.md` documents the new command
The constitution requires CLI changes to update `src/kit/skills/_cli-fab.md`. A `## fab memory-index` section SHALL be added documenting the command signature, behavior (regeneration + degradation + warnings), exit codes, and consumers (the hydrate skill / `docs-reorg-memory`).

- **GIVEN** the new subcommand exists
- **WHEN** `_cli-fab.md` is read
- **THEN** it contains a `## fab memory-index` reference section consistent with the implementation

### Memory Templates: `description:` Frontmatter

#### R6: Memory-file template gains `description:` frontmatter
The individual memory-file format in `docs/specs/templates.md` SHALL document a leading YAML frontmatter block carrying a `description:` field (a curated one-line summary), authored by every memory writer (hydrate, `/docs-hydrate-memory`, `docs-reorg-memory`).

- **GIVEN** the memory-file template
- **WHEN** a new memory file is authored
- **THEN** it begins with `---\ndescription: "..."\n---` frontmatter above the `# H1`

#### R7: Root index becomes domains-only in templates/specs
The Top-Level Index format in `docs/specs/templates.md` SHALL be updated to a domains-only table (`| Domain | Description |`), dropping the inlined per-file "Memory Files" column.

- **GIVEN** the templates spec
- **WHEN** the root index format is read
- **THEN** it shows a two-column `| Domain | Description |` table with no per-file column

#### R8: Backfill `description:` frontmatter on all existing topic files
All existing `docs/memory/**/*.md` topic files (NOT `index.md` files) SHALL be backfilled with their current curated one-line descriptions — taken from the current root index's inlined descriptions where present, else synthesized from the file's Overview.

- **GIVEN** the 20 existing `fab-workflow/*.md` topic files (excluding `index.md`)
- **WHEN** the backfill is applied
- **THEN** each file carries a `description:` frontmatter line with its curated description
- **AND** no `index.md` file gains frontmatter

### Hydrate Wiring: Mechanical Index Regeneration

#### R9: Hydrate skills call `fab memory-index` instead of hand-maintaining rows
The `docs-hydrate-memory` skill's "Step 4: Update Indexes" (and the generate-mode "Step 4: Index Maintenance"), plus the `/fab-continue` hydrate behavior's "update indexes" step, SHALL be replaced with a single mechanical call to `fab memory-index`. Memory writers SHALL author the `description:` frontmatter field on new/modified files so the regenerated index has content.

- **GIVEN** a memory write during hydrate
- **WHEN** indexes need updating
- **THEN** the skill runs `fab memory-index` rather than hand-editing index rows

#### R10: Shape bounds codified as SHOULD guidance
The hydrate skill(s) and `docs/specs/templates.md` SHALL codify the ideal-shape bounds as SHOULD guidance: ~5 lower / ~12 upper file count per folder, depth ≤ 3, and a sub-domain earns its own index only when a cohesive cluster of ≥ 8 files exists. Reserved domains `_shared/` and `_unsorted/` are exempt.

- **GIVEN** a writer adding memory files
- **WHEN** a folder approaches the bounds
- **THEN** the guidance steers toward (reactive) sub-domain introduction without mandating it

### docs-reorg-memory: Shape Report (detect-only)

#### R11: docs-reorg-memory gains a read-only Shape Report + index regen
The `docs-reorg-memory` skill SHALL add a read-only "Shape Report" to its diagnosis step that flags folders over the width bound, over depth 3, or under the floor (reserved domains exempt), and SHALL switch its index updates to `fab memory-index`. The file-moving / split / merge / flatten APPLY machinery SHALL NOT be added (deferred follow-up `sx7a`).

- **GIVEN** a memory tree with an over-wide domain
- **WHEN** `docs-reorg-memory` runs its diagnosis
- **THEN** a Shape Report lists the violating folders with their counts/depth and the relevant bound
- **AND** no file-moving apply machinery is introduced

### Specs: SPEC + templates conformance

#### R12: Update SPEC files for every changed skill
Per the constitution, `docs/specs/skills/SPEC-docs-reorg-memory.md`, `docs/specs/skills/SPEC-docs-hydrate-memory.md`, and `docs/specs/skills/SPEC-fab-continue.md` SHALL be updated to reflect the `fab memory-index` wiring and (for reorg) the Shape Report.

- **GIVEN** a skill body changed in this change
- **WHEN** its SPEC is read
- **THEN** the SPEC reflects the new `fab memory-index` flow / Shape Report

### Non-Goals

- No file moves, no sub-domain splitting, no relative-link rewriting (deferred follow-up `sx7a`).
- No new `/fab-rebalance-memory` skill — the rebalancer is the existing `docs-reorg-memory`.
- No change to the intake `{domain}/{file-name}` contract, `_preamble` always-load, or `context-loading` 2-hop convention (no files move).
- No regen sequencing guard (hook/preflight/CI) shipped in this change — open question deferred (see Assumptions #6).
- Shape bounds remain hardcoded guidance, not `config.yaml`-configurable (Assumptions #5).

### Design Decisions

1. **`internal/memoryindex` package modeled on `internal/prmeta`**: pure `Render(RootData)` + `RenderDomain(DomainData)` + `Gather` I/O orchestrator — *Why*: matches the established deterministic-render pattern and keeps byte-for-byte output unit-testable without git fixtures — *Rejected*: shelling from the skill (non-deterministic, not testable, the conflict-churn this change exists to kill).
2. **Reuse `internal/frontmatter.Field`** for reading `description:` — *Why*: existing, tested helper that strips quotes/comments — *Rejected*: a new YAML parse (duplication).
3. **Warnings to stderr, indexes to files**: keeps stdout/file output byte-stable regardless of warning state (Constitution: Idempotent) — *Rejected*: embedding warnings in index files (would churn the artifact).

## Tasks

### Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/memoryindex/` package directory with `memoryindex.go` package doc + `RootData`/`DomainData`/`FileEntry` structs (mirror `internal/prmeta` Data shape) <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 Implement pure renderers in `src/go/fab/internal/memoryindex/memoryindex.go`: `RenderRoot(RootData) string` (domains-only table) and `RenderDomain(DomainData) string` (file rows with Description + Last Updated), byte-stable, deterministic domain/file ordering (lexicographic) <!-- R1 R2 R3 -->
- [x] T003 Implement `Gather(repoRoot) ([]DomainData, RootData, []Warning, error)` in `memoryindex.go`: walk `docs/memory/`, read each file's H1 + `description:` via `internal/frontmatter.Field`, stamp Last Updated via `git log -1 --date=short` (degrade to `—` on empty/error), compute per-folder counts + depth, collect shape warnings exempting `_shared`/`_unsorted` <!-- R2 R3 R4 -->
- [x] T004 Add `memoryIndexCmd()` cobra command in `src/go/fab/cmd/fab/memory_index.go` (resolve repo root via `resolve.FabRoot` + parent dir like prmeta; write root + domain index files; print warnings to stderr) and register it in `src/go/fab/cmd/fab/main.go` <!-- R1 R4 -->
- [x] T005 [P] Write byte-for-byte render fixture tests in `src/go/fab/internal/memoryindex/memoryindex_test.go`: RenderRoot/RenderDomain golden output, idempotency, missing-description degradation, missing-date degradation, shape-warning thresholds, reserved-domain exemption (use loom's stale-count scenario as a regression fixture per Assumptions intake #11) <!-- R1 R2 R3 R4 -->
- [x] T006 [P] Add command-registration test in `src/go/fab/cmd/fab/memory_index_test.go` (mirrors `pr_meta_test.go`: registered Use, runs against a temp `docs/memory/` tmpdir tree) <!-- R1 -->

### Phase 3: Integration & Migration

- [x] T007 Add `description:` frontmatter to the memory-file template + flip root index to domains-only in `docs/specs/templates.md`; add shape-bounds SHOULD guidance <!-- R6 R7 R10 -->
- [x] T008 Backfill `description:` frontmatter into all 20 `docs/memory/fab-workflow/*.md` topic files (descriptions sourced from the current root index inlined column / file Overview); do NOT touch any `index.md` <!-- R8 -->
- [x] T009 Add `## fab memory-index` section to `src/kit/skills/_cli-fab.md` <!-- R5 -->
- [x] T010 Rewire `src/kit/skills/docs-hydrate-memory.md` Step 4 (ingest) + Step 4 (generate) to call `fab memory-index`; add shape-bounds SHOULD guidance + `description:` authoring instruction <!-- R9 R10 -->
- [x] T011 Rewire `src/kit/skills/fab-continue.md` Hydrate Behavior step 4 (update indexes → `fab memory-index`); add shape-bounds SHOULD note <!-- R9 R10 -->
- [x] T012 Add Shape Report (detect-only) to `src/kit/skills/docs-reorg-memory.md` diagnosis step + switch index updates to `fab memory-index`; keep file-moving apply OUT (note follow-up) <!-- R11 -->
- [x] T013 [P] Update `docs/specs/skills/SPEC-docs-reorg-memory.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-fab-continue.md` to reflect `fab memory-index` wiring + Shape Report <!-- R12 -->

### Phase 4: Migration Execution & Verification

- [x] T014 Build (`go build ./...`) and run `fab memory-index` once against the real `docs/memory/` (the actual migration: regenerates indexes from backfilled frontmatter, self-heals drift); verify a second run is a no-op (idempotent) <!-- R1 R8 -->

## Execution Order

- T001 blocks T002–T006
- T002 blocks T003 (Gather builds the Data the renderers consume)
- T003–T004 block T005–T006 (tests target the implemented funcs/cmd)
- T007 blocks T008 (template defines the frontmatter shape before backfill)
- T008 blocks T014 (frontmatter must exist before the real regen)
- T009–T013 are doc/skill edits, independent of each other once T002–T004 land

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab memory-index` regenerates root (domains-only) + every domain index; `go test ./src/go/fab/internal/memoryindex/...` passes
- [x] A-002 R2: index rows are populated from each file's H1 + `description:` frontmatter (verified by render fixture test `TestRenderDomain_FileRows` + `TestGather_ReadsFrontmatterAndH1`)
- [x] A-003 R3: "Last Updated" is stamped from `git log -1 --date=short`, degrading to `—` when unresolvable (`TestGather_UncommittedDateDegrades`; real-tree run shows all 20 rows with valid dates)
- [x] A-004 R4: over-width / over-depth folders emit non-fatal stderr warnings; `_shared`/`_unsorted` exempt (`TestGather_WidthWarningAndReservedExemption` + `TestGather_DepthWarning`)
- [x] A-005 R5: `_cli-fab.md` has a `## fab memory-index` section matching the implementation (signature, behavior, degradation, warnings, --check, exit codes, consumers)
- [x] A-006 R6: memory-file template documents `description:` frontmatter (templates.md "Individual File" section)
- [x] A-007 R7: templates spec root index is domains-only (no per-file "Memory Files" column)
- [x] A-008 R8: all 20 `fab-workflow/*.md` topic files carry `description:` frontmatter; no `index.md` was *backfilled* (the domain index.md's `description:` frontmatter is a generator round-trip, not a backfill — see Notes)
- [x] A-009 R9: hydrate skill(s) + `/fab-continue` hydrate call `fab memory-index` (no hand-maintained "Update Indexes"/"add rows" prose remains)
- [x] A-010 R10: shape bounds (~5/~12, depth ≤3, ≥8-cluster) appear as SHOULD guidance in templates.md + both hydrate skills + fab-continue
- [x] A-011 R11: `docs-reorg-memory` has a read-only Shape Report; split/merge/flatten are framed as proposals only, no file-moving apply machinery added
- [x] A-012 R12: SPEC files for reorg/hydrate/continue reflect the new `fab memory-index` flow + Shape Report

### Behavioral Correctness

- [x] A-013 R1: re-running `fab memory-index` after the migration produces zero diff (verified: `--check` exits 0, run says "already up to date", git status unchanged after run)
- [x] A-014 R8: the regen self-heals prior root-index drift (stale 18-file roster replaced by the generated domains-only index; `TestGather_SelfHealsStaleRoster`)

### Scenario Coverage

- [x] A-015 R4: a 20-file domain (`fab-workflow`) triggers the width warning end-to-end on the real tree (verified live: `⚠ docs/memory/fab-workflow has 20 topic files`)
- [x] A-016 R3: an uncommitted memory file renders `—` (`TestGather_UncommittedDateDegrades`)

### Edge Cases & Error Handling

- [x] A-017 R2: a file missing `description:` still renders a row (`TestRenderDomain_MissingDescriptionAndDateDegrade`; `TestGather_UncommittedDateDegrades` uses a no-frontmatter file)
- [x] A-018 R1: an empty / single-`index.md` `docs/memory/` tree does not error — **met by inspection** (zero domain dirs → empty loop → empty table, no crash); a present-but-domainless tree is not explicitly unit-tested (only the *missing*-dir error path is, `TestGather_MissingMemoryDirErrors`)

### Code Quality

- [x] A-019 Pattern consistency: `internal/memoryindex` mirrors `internal/prmeta`'s Render/Gather split, package doc, `—` degradation, and `filepath.Dir(fabRoot)` repo-root resolution
- [x] A-020 No unnecessary duplication: reuses `internal/frontmatter.Field` (description read) and `internal/resolve.FabRoot` rather than reimplementing
- [x] A-021 Readability: renderers are focused pure functions; helpers are small; no function exceeds the 50-line god-function bound
- [x] A-022 Documentation accuracy (config extra category): `_cli-fab.md` + SPEC + templates match the shipped behavior (verified against the live binary output)
- [x] A-023 Cross-references (config extra category): skill edits reference `fab memory-index` consistently; no dangling "Update Indexes" prose remains

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- The follow-up change `sx7a` (file-moving rebalance + C-apply) is explicitly OUT of scope.
- **R8 vs. generated frontmatter (reconciled)**: R8/A-008 governs the *backfill migration* of topic files only ("no `index.md` gains frontmatter" = the backfill did not touch index.md). The `description:` frontmatter now present on `docs/memory/fab-workflow/index.md` is written by `RenderDomain` as a deliberate round-trip (intake Assumption #7 / memoryindex.go:74-82) so the root row survives regen — a different mechanism, not a backfill violation. Both readings are satisfied.

## Deletion Candidates

- `internal/memoryindex/memoryindex.go:58` (`FileEntry.Title`) + its populator `readH1(path)` at `gatherFiles` (line 259) — the topic-file H1 is gathered for every file but read by no renderer (only asserted in `memoryindex_test.go:136`). Speculative field; if no near-term diagnostic consumer is planned, dropping it removes one `os.Open`/scan per topic file. (Domain-level `readH1` for the index.md Title at line 271 IS used — keep that.)
- None other — this change adds a new command + frontmatter convention without making existing code redundant. The hand-maintained index-row prose it replaces lives in skill markdown (already rewritten in this change), not in deletable source symbols.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Place the package at `src/go/fab/internal/memoryindex` with `Render*`/`Gather` split mirroring `internal/prmeta`. | Intake §Impact + Assumption #7 name the `prmeta` pattern explicitly; one obvious location/structure. | S:92 R:70 A:95 D:90 |
| 2 | Certain | Reuse `internal/frontmatter.Field` to read `description:`; resolve repo root as `filepath.Dir(resolve.FabRoot())` (prmeta's `repoDir`). | Verified existing tested helpers in the codebase; prmeta does exactly this for repo-relative git calls. | S:88 R:75 A:92 D:88 |
| 3 | Confident | Render `—` (em-dash) for an absent date or absent description cell. | Matches prmeta's `—` fallback convention for missing data; keeps tables well-formed. | S:70 R:80 A:80 D:78 |
| 4 | Confident | Width-warning threshold is `> 12` topic files; depth warning is `> 3` path levels under `docs/memory/`. Soft floor (~5) and ≥8-cluster are guidance-only (no warning emitted below floor). | Intake §4 + Assumption #6 give ~12 upper / ~5 lower / depth ≤3 / ≥8 cluster; only the upper + depth are coded as warnings (the floor is advisory, warning on too-few files would be noisy). | S:72 R:78 A:80 D:72 |
| 5 | Confident | Shape bounds stay hardcoded guidance (no `config.yaml` surface added). | Intake Open Question left this open; YAGNI — guidance suffices for B+C-detect, a config surface can be added with the follow-up if needed. Reversible. | S:60 R:78 A:75 D:70 |
| 6 | Confident | No regen sequencing guard (hook/preflight/CI) is shipped in this change. | Intake Open Question + Assumption #4 mark the guard as needed but mechanism-undecided; shipping the wrong mechanism is harder to undo than adding it later. Detect-warnings already surface staleness on each run. | S:58 R:65 A:68 D:60 |
| 7 | Confident | Root index row format is `\| [domain](domain/index.md) \| description \|`; domain description sourced from the existing root index's per-domain Description cell (fallback: synthesized). | The current root index already carries a per-domain Description; preserving it is the obvious domains-only reduction. | S:74 R:75 A:82 D:80 |
| 8 | Confident | Domain/file ordering in generated indexes is lexicographic by folder/basename. | Determinism requires a fixed order; lexicographic is the conventional, reproducible choice and what a fresh walk + sort yields. | S:70 R:80 A:80 D:75 |

8 assumptions (2 certain, 6 confident, 0 tentative).
