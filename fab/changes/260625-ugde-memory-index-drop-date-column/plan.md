# Plan: Drop the "Last Updated" column from generated memory indexes

**Change**: 260625-ugde-memory-index-drop-date-column
**Intake**: `intake.md`

## Requirements

### Renderer: Domain Index Column Shape

#### R1: Domain/sub-domain index renders two columns only
The `RenderDomain` function SHALL render the domain (and sub-domain) file-row table with exactly two columns — `| File | Description |` — dropping the third `Last Updated` column. The root index and the `## Sub-Domains` table (already two-column) SHALL be unchanged. The render SHALL remain a pure function of content (file names + descriptions + structure), making the output branch-independent and idempotent (Constitution III).

- **GIVEN** a `DomainData` with topic files
- **WHEN** `RenderDomain(d)` is called
- **THEN** the table header is `| File | Description |`, the separator is `|------|-------------|`, and each row is `| [base](base.md) | desc |`
- **AND** re-running `fab memory-index` on an unchanged tree produces a byte-identical index regardless of branch/HEAD

#### R2: Dead index-only date plumbing is removed
The renderer package SHALL remove the date-map plumbing that existed solely to feed the index date cell: `FileEntry.LastUpdated`, its population in `gatherFiles` (the `dates.lookup(...)` call), `(*gitDates).lookup`, `gitLastUpdated`, and `gitDates.byPath` (with `parseGitLog` collapsed to return only what `log.md` needs). The batched `git log` pass (`loadGitDates`), `gitDates.commitsByPath`, `gitDates.top`, `gitRelPath`, and everything `gatherLogEntries`/`GatherLogs` consume SHALL be retained — `log.md` still depends on `commitsByPath`.

- **GIVEN** the renderer package after the column drop
- **WHEN** the package is compiled and `log.md` generation runs
- **THEN** no `byPath`/`lookup`/`gitLastUpdated` symbols remain, and `log.md` still renders from `commitsByPath`
- **AND** the package doc comment no longer claims it stamps "Last Updated" / "git log dates" for the index

### `--check` Classifier: Two-Column Parsing

#### R3: `--check` parses the 2-column domain table; exit-code contract unchanged
The `indexparse.go`/`loss.go` destructive-loss detectors SHALL parse the new 2-column domain/sub-domain rows so description/tombstone/grouping detection still works after the format change. The `--check` exit-code contract (0 clean / 1 benign drift / 2 destructive loss) SHALL be unchanged.

- **GIVEN** a committed 2-column `index.md`
- **WHEN** `fab memory-index --check` runs
- **THEN** description-loss, tombstone, and grouping detection behave as before and the exit code maps 0/1/2 unchanged

### CLI Help Text

#### R4: CLI help no longer mentions stamping "Last Updated"
The `memory-index` command's `Long` description and any tier-1 example SHALL no longer reference stamping/refreshing "Last Updated" from git for the index.

- **GIVEN** `fab memory-index --help`
- **WHEN** the help text is read
- **THEN** it describes the index as content-only (no "Last Updated" / git-stamping for the index), while still describing the retained batched pass for `log.md`

### Tests Conform to Spec

#### R5: Tests assert the 2-column domain index
All renderer/classifier tests (`golden_test.go`, `memoryindex_test.go`, `loss_test.go`, and any `freeze_test.go`/`log_test.go`/`seed_test.go` case asserting a 3-column domain table) SHALL be updated to the 2-column form, and `parseGitLog`/`lookup` tests SHALL be updated for the collapsed signature (Constitution VII — tests conform to spec).

- **GIVEN** the updated renderer and classifier
- **WHEN** `go test ./internal/memoryindex/... ./cmd/fab/...` runs
- **THEN** all tests pass against the 2-column output

### Documentation Mirror Sweep

#### R6: Spec + kit-reference mirror move together
`docs/specs/fkf.md` (§2/§5/§6.1), `src/kit/reference/fkf.md` (the normative mirror), `docs/specs/templates.md`, and `src/kit/skills/_cli-fab.md` SHALL drop/reword every live "Last Updated" / index-date reference. The fkf.md dual-file rule requires both fkf files move together.

- **GIVEN** the doc sweep
- **WHEN** the specs/reference are read
- **THEN** no live spec/reference describes the index as carrying a git-stamped "Last Updated" column; the batched-pass description now attributes dates to `log.md` only

#### R7: Skills + their SPEC mirrors move together
Each touched `src/kit/skills/*.md` (`docs-hydrate-memory.md`, `fab-continue.md`, `docs-reorg-memory.md`, `git-pr.md`) SHALL be edited together with its `docs/specs/skills/SPEC-*.md` mirror (Constitution Additional Constraints). Edits SHALL be on canonical `src/kit/skills/` sources only — never `.claude/skills/`.

- **GIVEN** a skill edit dropping the "Last Updated" reference
- **WHEN** the change is reviewed
- **THEN** every edited skill has a matching SPEC-*.md edit in the same change

#### R8: 3a-bis is retained with a log.md-only rationale
`/git-pr` sub-step 3a-bis SHALL NOT be deleted. Its rationale in `git-pr.md`, `SPEC-git-pr.md`, and `pipeline/execution-skills.md` SHALL be narrowed to "log.md only" — log.md still needs the post-commit projection to capture the change's own entry pre-squash; the index-regen half becomes a reliable no-op.

- **GIVEN** the narrowed rationale
- **WHEN** 3a-bis prose is read
- **THEN** it justifies the post-commit regen by `log.md`'s freeze-on-write needs, not by index date drift, and the sub-step still exists

### Memory Prose

#### R9: Post-impl memory prose describing the column is reworded
`docs/memory/memory-docs/{hydrate,hydrate-generate,templates}.md`, `docs/memory/distribution/kit-architecture.md`, `docs/memory/pipeline/{execution-skills,schemas}.md` SHALL drop/reword the `| File | Description | Last Updated |` references and date-stamping prose, narrowing the 3a-bis design-decision prose to log.md-only. Frozen `log.md`/`log.seed.md` historical entries and generated `docs/memory/**/index.md` files SHALL be left untouched (the migration regenerates the indexes). The root `docs/memory/index.md` SHALL NOT be touched.

- **GIVEN** the memory prose sweep
- **WHEN** the memory files are read
- **THEN** no active memory prose describes the index date column; frozen log entries are preserved verbatim

### Migration

#### R10: Migration re-baselines every index.md to 2-column form
A `src/kit/migrations/` file SHALL re-baseline every `docs/memory/**/index.md` to the 2-column form by running the new `fab memory-index`, matching the existing migration format/versioning convention, with a pre-check that the installed binary produces 2-column output before rewriting. This is the last churn the repo sees from the date column.

- **GIVEN** an existing project with 3-column domain indexes
- **WHEN** `/fab-setup migrations` applies this migration
- **THEN** the indexes are re-baselined to 2-column and `fab memory-index --check` is byte-stable afterward (never tier 2)
- **AND** the migration aborts cleanly if the running binary still emits a 3-column index

### Design Decisions

1. **Drop the column (Option A)** rather than freeze-on-write / pin-to-ref / relax `--check`: only Option A makes the index a pure function of content — true idempotency. *Rejected*: B (still moves on any touch), C (env-dependent), D (file still churns). (Intake assumption #1.)
2. **`parseGitLog` collapses to a single return** (`commitsByPath` only): `byPath` has no remaining consumer once the index date cell is gone. Keeping a dead 2-tuple would be misleading. (Intake OQ — apply-stage judgment.)

### Non-Goals

- No flag/API changes to `fab memory-index` (`--check`, `--json`, `--rebuild` keep their signatures).
- No edits to frozen `log.md`/`log.seed.md` history or to generated `index.md` files by hand (the migration regenerates indexes).

## Tasks

### Phase 1: Core Renderer

- [x] T001 In `src/go/fab/internal/memoryindex/memoryindex.go`, change `RenderDomain` to a 2-column table (header `| File | Description |`, separator `|------|-------------|`, row `| [%s](%s.md) | %s |`); reword the "do not hand-edit" note to drop "dates from `git log`"; update the package doc comment to drop "git log dates" / "stamping Last Updated" for the index <!-- R1 -->
- [x] T002 In `memoryindex.go`, remove `FileEntry.LastUpdated`, the `dates.lookup(...)` call in `gatherFiles`, `(*gitDates).lookup`, `gitLastUpdated`, and `gitDates.byPath`; collapse `parseGitLog` to return only `commitsByPath`; update `loadGitDates` and all callers + doc comments; keep `loadGitDates`/`commitsByPath`/`top`/`gitRelPath`/`gatherLogEntries`/`GatherLogs` intact <!-- R2 -->

### Phase 2: `--check` Parser + CLI Help

- [x] T003 In `src/go/fab/internal/memoryindex/indexparse.go` and `loss.go`, update the row-parsing doc comments/examples to expect 2-column domain rows; verify `parseIndexRows` (cell[0]=link, cell[1]=desc) is column-count-tolerant so detection still works; keep the 0/1/2 contract <!-- R3 -->
- [x] T004 In `src/go/fab/cmd/fab/memory_index.go`, reword the `Long` description to drop "stamping \"Last Updated\" from git" (index) and reword the tier-1 example "a refreshed `Last Updated`"; keep the log.md batched-pass description <!-- R4 -->

### Phase 3: Tests

- [x] T005 Update `golden_test.go`, `memoryindex_test.go`, and `loss_test.go` to the 2-column domain index; update `parseGitLog`/`lookup`/`LastUpdated` test expectations to the collapsed signature; run `cd src/go/fab && go test ./internal/memoryindex/... ./cmd/fab/...` and fix until green <!-- R5 -->

### Phase 4: Spec + Reference + CLI-ref Sweep

- [x] T006 Edit `docs/specs/fkf.md` (§2 line ~59, §5 lines ~208-209, §6.1 line ~244) and `src/kit/reference/fkf.md` (lines ~38, ~146-147) together — drop the "Last Updated" column from the Domain tier, the conformance note, and reword §6.1's "same date source the index uses" to log.md-only <!-- R6 --> <!-- rework: M2 — §5.1 "Generated Indexes" prose in BOTH fkf.md (~line 195) and src/kit/reference/fkf.md (~line 133) STILL reads "a pure function of (folder contents + description + git dates), so the output is byte-stable/idempotent" — the exact falsehood this change removes. Drop "+ git dates" from the basis clause in BOTH files; keep them byte-identical -->
- [x] T007 Edit `docs/specs/templates.md` (lines ~418, ~429, ~463, ~541, ~571) — drop the `| File | Description | Last Updated |` tables, the design-rationale date sentence, and the "never hand-edit Last Updated cells" instruction <!-- R6 -->
- [x] T008 Edit `src/kit/skills/_cli-fab.md` (§ fab memory-index, lines ~495, ~506, ~573, ~578, ~636) — column description, batched-pass note (log.md only), tier-1 drift example <!-- R6 -->

### Phase 5: Skills + SPEC Mirrors

- [x] T009 [P] Edit `src/kit/skills/docs-hydrate-memory.md` (lines ~37, ~113) + `docs/specs/skills/SPEC-docs-hydrate-memory.md` (drop "Last Updated" cell references) <!-- R7 --> <!-- rework: M3 — docs-hydrate-memory.md lines ~164 and ~196 STILL say regenerate indexes "from folder contents + frontmatter + git dates"; the index no longer consumes git dates and the per-file git fallback was deleted. Drop "+ git dates" on both lines -->
- [x] T010 [P] Edit `src/kit/skills/fab-continue.md` (line ~209) + `docs/specs/skills/SPEC-fab-continue.md` (drop "Last Updated" cell reference) <!-- R7 -->
- [x] T011 [P] Edit `src/kit/skills/docs-reorg-memory.md` + `docs/specs/skills/SPEC-docs-reorg-memory.md` (line ~41, drop "Last Updated" cell reference) <!-- R7 -->
- [x] T012 Edit `src/kit/skills/git-pr.md` (line ~213) + `docs/specs/skills/SPEC-git-pr.md` (line ~18) — rewrite 3a-bis rationale to log.md-only; do NOT delete the sub-step <!-- R8 -->

### Phase 6: Memory Prose

- [x] T013 [P] Edit `docs/memory/memory-docs/hydrate.md` (lines ~87, ~97, ~120) and `docs/memory/memory-docs/hydrate-generate.md` (line ~89) — index is content-only, drop date-stamping prose <!-- R9 --> <!-- rework: M3 — hydrate.md ~line 99 STILL says "The index tiers are a pure function of folder contents + frontmatter + git dates" (contradicts the reworded line 97 "index carries no dates"). Drop "+ git dates" -->
- [x] T014 [P] Edit `docs/memory/memory-docs/templates.md` (lines ~140-141, ~187, ~254) — domain/sub-domain rows are `| File | Description |`; recency lives in log.md <!-- R9 --> <!-- rework: M3 — templates.md ~line 137 STILL says "The index render is a pure function of folder contents + ... frontmatter + git dates ... the per-file `git log -1` spawn is retained solely as fallback" (the fallback was DELETED; contradicts line 140) and ~line 253 Decision line STILL says "generated ... from a ... description: frontmatter field + git dates". Drop "+ git dates" on both and delete the deleted-fallback claim on ~137 -->
- [x] T015 [P] Edit `docs/memory/distribution/kit-architecture.md` (lines ~324, ~331) — drop date-stamping description + dead `byPath`/`lookup`/`gitLastUpdated` references; note `loadGitDates`/`commitsByPath` retained for log.md <!-- R9 -->
- [x] T016 [P] Edit `docs/memory/pipeline/execution-skills.md` (lines ~3, ~63, ~222) and `docs/memory/pipeline/schemas.md` (line ~156) — narrow 3a-bis rationale to log.md-only; batched pass now yields only `commitsByPath` <!-- R9 R8 -->

### Phase 7: Migration

- [x] T017 Create `src/kit/migrations/2.6.6-to-{next}.md` matching the existing migration format — re-baseline every `docs/memory/**/index.md` to 2-column via `fab memory-index`, with a pre-check that the installed binary produces 2-column output before rewriting <!-- R10 -->
- [x] T018 Bump `src/kit/VERSION` to `2.7.0` (the migration's target version). <!-- R10 --> <!-- rework: M1 — a feature/behavior migration bumps src/kit/VERSION to its target IN THE SAME CHANGE; precedent 73771c2b (2.5.5-to-2.6.0) did exactly this and migrations.md:110 states the rule. The migration file's own Verification asserts VERSION reads 2.7.0; leaving it at 2.6.6 is self-contradictory -->
- [x] T019 Catalog the new migration in `docs/memory/distribution/migrations.md` — add a `### 2.6.6-to-2.7.0` section mirroring the `### 2.5.5-to-2.6.0` entry's shape (re-baseline mechanism, the rendered-output binary pre-check as a second output-probe precedent, no `fab/`/`.status.yaml` change, VERSION bump to 2.7.0) and extend the frontmatter `description:` clause to mention the `2.6.6-to-2.7.0` migration. <!-- R10 --> <!-- rework: SF1 — precedent 73771c2b bundled the migrations.md catalog entry in the same commit; this is the documenting-memory-file of the shipped migration (cross_references sweep class). migrations.md frontmatter is generated-index-adjacent prose — edit the body + the description: clause -->

## Execution Order

- T001, T002 (renderer) before T003 (parser comments reference renderer shape) and T005 (tests assert renderer output)
- T005 runs after T001-T004 (compiles + asserts final behavior)
- T006-T016 (doc sweep) are independent of the Go work and of each other (different files), but T012/T016 share the 3a-bis narrative — keep them consistent
- T017 (migration) last — it documents the shipped behavior

## Acceptance

### Functional Completeness

- [x] A-001 R1: `RenderDomain` emits `| File | Description |` (2-column) for domain and sub-domain indexes; root and `## Sub-Domains` tables unchanged
- [x] A-002 R2: `FileEntry.LastUpdated`, `(*gitDates).lookup`, `gitLastUpdated`, `gitDates.byPath` are gone; `parseGitLog` returns only `commitsByPath`; `loadGitDates`/`commitsByPath`/`gatherLogEntries`/`GatherLogs` retained and `log.md` still generates
- [x] A-003 R3: `--check` parses 2-column domain rows; description/tombstone/grouping detection works; exit codes 0/1/2 unchanged
- [x] A-004 R4: `fab memory-index --help` no longer references stamping/refreshing "Last Updated" for the index
- [x] A-005 R5: renderer/classifier tests assert 2-column output and the collapsed `parseGitLog` signature
- [x] A-006 R6: `docs/specs/fkf.md`, `src/kit/reference/fkf.md`, `docs/specs/templates.md`, `src/kit/skills/_cli-fab.md` carry no live index "Last Updated" reference
- [x] A-007 R7: each edited `src/kit/skills/*.md` has its matching `docs/specs/skills/SPEC-*.md` edit; no `.claude/skills/` edits
- [x] A-008 R8: 3a-bis still exists in `git-pr.md`/`SPEC-git-pr.md`/`execution-skills.md` with a log.md-only rationale
- [x] A-009 R9: memory prose in the 6 listed files reworded; frozen log.md/log.seed.md and generated index.md untouched; root index.md untouched
- [x] A-010 R10: a `src/kit/migrations/` file re-baselines every index.md to 2-column with a binary pre-check, matching the existing migration convention

### Behavioral Correctness

- [x] A-011 R1: re-running `fab memory-index` on an unchanged tree is byte-stable (second run reports "already up to date") and branch-independent
- [x] A-012 R3: the `--check` exit-code contract is unchanged (verified by passing classifier tests)

### Removal Verification

- [x] A-013 R2: no dead date-map symbols remain (`go build` clean; grep finds no `byPath`/`lookup`/`gitLastUpdated`/`LastUpdated` in the index path)

### Edge Cases & Error Handling

- [x] A-014 R10: migration pre-check aborts cleanly when the running binary still emits a 3-column index (no partial rewrite)

### Code Quality

- [x] A-015 Pattern consistency: new code/doc follows surrounding naming, comment density, and idiom
- [x] A-016 No unnecessary duplication: reuses existing renderer/parser utilities; no reimplementation
- [x] A-017 Migrations for user-data restructuring: index column-shape change ships as a `src/kit/migrations/` file, not an ad-hoc script (code-review.md project rule)
- [x] A-018 Go changes ship tests: the `.go` change carries updated tests (Constitution VII)

### documentation_accuracy

- [x] A-019: every live spec/skill/template/active-memory reference to the index "Last Updated" column is dropped or reworded; only frozen historical `log.md`/`log.seed.md` entries and generated `index.md` files retain it
- [x] A-020: CLI help (`_cli-fab.md` + `memory_index.go` Long) and the memory `kit-architecture` description accurately describe the retained batched pass as serving `log.md` only

### cross_references

- [x] A-021: the skill ↔ SPEC-*.md mirror class and the `fkf.md` dual-file (spec + kit reference) are swept together — no half-updated mirror
- [x] A-022: 3a-bis prose is internally consistent across `git-pr.md`, `SPEC-git-pr.md`, `pipeline/execution-skills.md` (and `pipeline/index.md` if it restates the line)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop the `Last Updated` column entirely (Option A) | User explicitly selected Option A in the discuss session; only option making the index a pure function of content (Constitution III) | S:95 R:70 A:95 D:95 |
| 2 | Certain | Collapse `parseGitLog` to return only `commitsByPath` (single return value) | `byPath` has no remaining consumer once the index date cell is gone; a dead 2-tuple would mislead readers | S:85 R:80 A:90 D:85 |
| 3 | Confident | Update `loss_test.go` existing/rendered fixtures to 2-column rather than leaving them 3-column | Tests should reflect the real shipped format (Constitution VII); the classifier is column-tolerant but fixtures must model reality | S:75 R:80 A:85 D:80 |
| 4 | Certain | Name the migration `2.6.6-to-2.7.0.md` (next minor after the installed 2.6.6) and bump kit VERSION to 2.7.0 | Matches the existing migration naming convention (`{from}-to-{to}.md`); a column-shape change is a minor bump, consistent with 2.5.5→2.6.0 (the prior memory-index migration) | S:80 R:75 A:85 D:80 |
| 5 | Confident | Migration uses a plain `fab memory-index` re-baseline (not `--rebuild`) with a 2-column binary pre-check | The index drop is byte-stable on a plain run; `--rebuild` is the destructive log.md escape hatch and is not needed for an index column change. The pre-check probes the rendered output, mirroring 2.5.5-to-2.6.0's capability probe | S:75 R:75 A:80 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative).
