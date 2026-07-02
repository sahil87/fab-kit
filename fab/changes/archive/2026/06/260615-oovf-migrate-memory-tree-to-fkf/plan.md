# Plan: FKF Change 4/4 — Migrate docs/memory/ Tree to FKF + FKF-Aware Reorg Skills

**Change**: 260615-oovf-migrate-memory-tree-to-fkf
**Intake**: `intake.md`

## Requirements

### Memory-Index: Seed-Merge Generation (Q1 — the crux)

Pre-FKF `## Changelog` rows have no live `.status.yaml` `summary:` to regenerate from, so faithful
preservation (DECISION b) requires the generator to accept a curated **seed input** and merge it
into the generated `log.md`. The seed is a per-folder sidecar file `log.seed.md` in the FKF §6.2
entry format; it is an **input** (curated, like `description:` frontmatter), never the generated
output — so the single-writer / byte-stable discipline (FKF §5/§6) is preserved: `fab memory-index`
remains the sole writer of `log.md`, and the seed is just another gathered input.

#### R1: Seed-file read + parse
`fab memory-index` SHALL read a per-folder `log.seed.md` (when present) and parse its date-grouped
FKF §6.2 entries (`## YYYY-MM-DD` headings; `- [**Verb** ][base](/bundle/rel.md) — summary (id)`
lines) into `LogEntry` values, preserving the seed's own date headings (the changelog `Date`
column, independent of git) and the verbatim summary text.

- **GIVEN** a folder `docs/memory/{domain}/` with a `log.seed.md` carrying entries under
  `## 2026-02-09`
- **WHEN** `fab memory-index` runs
- **THEN** those seed entries appear in the generated `log.md` under their original `2026-02-09`
  date heading, with their summary text preserved verbatim

#### R2: Merge-beneath-projected, deterministic
The generator SHALL merge seed entries with the git-projected entries into one `log.md`: union by
date, newest date first; within a date, git-projected entries first then seed entries, each block
ordered by the existing stable secondary sort (file base, then change-id). The merged render SHALL
be a pure function of (git history + registry + parsed seed) so it stays byte-stable.

- **GIVEN** a folder whose git history projects an entry under `## 2026-06-12` and whose
  `log.seed.md` also carries an entry under `## 2026-06-12`
- **WHEN** `fab memory-index` runs
- **THEN** both appear under a single `## 2026-06-12` heading, git-projected entry(ies) first, in a
  deterministic order identical on every run

#### R3: Idempotent seed preservation (Constitution III)
Seed entries SHALL survive every regeneration without duplication. A seed entry that is byte-equal
to a git-projected entry for the same (date, file, id) SHALL be de-duplicated so a re-run is a
no-op. Re-running `fab memory-index` on an already-generated tree SHALL write nothing.

- **GIVEN** a generated `log.md` and an unchanged `log.seed.md`
- **WHEN** `fab memory-index` runs a second time
- **THEN** the on-disk `log.md` is byte-identical (no doubled seed entries) — `--check` exits 0

#### R4: Seed file is a non-topic, non-output artifact
`log.seed.md` SHALL NOT be gathered as a topic file (no `[log.seed]` row in any index) and SHALL
NOT be overwritten by the generator (it is curated input). It is excluded from `gatherFiles` /
`gatherLogEntries` exactly as `index.md` / `log.md` are.

- **GIVEN** a folder containing `log.seed.md` alongside real topic files
- **WHEN** `fab memory-index` runs
- **THEN** the domain index has no `[log.seed]` row and `log.seed.md` is left untouched on disk

#### R5: Loss-tier classification of a seed-merge run (loss.go)
A `log.md` whose drift is driven by a seed-merge SHALL classify as benign (tier 1) on
`--check`, never destructive loss (tier 2). The existing `IsLog` guard already routes all `log.md`
drift to benign; this requirement pins that the seed-merge introduces no new tier-2 category and the
preserved seed is never reported as loss.

- **GIVEN** a `log.md` target whose rendered content includes merged seed entries differing from
  on-disk content
- **WHEN** `fab memory-index --check` runs
- **THEN** the report tier is at most 1 (benign drift), `losses` is empty for that target

#### Non-Goals
- No new `fab` subcommand or `--seed` flag — the sidecar file is auto-discovered per folder (zero
  new CLI surface; Constitution I).
- No back-fill of archived-change `summary:` fields (Q1 rejected alternative ii).
- No change to the git-projection / registry-attribution path for changes that DO have a live
  `.status.yaml` — that path is unchanged.

### Memory-Data: One-Time Conversion (group a)

#### R6: Strip `## Changelog` from every memory file (idempotent)
Every topic file under `docs/memory/{_shared,distribution,memory-docs,pipeline,runtime}/*.md`
(excluding `index.md`/`log.md`/`log.seed.md`) SHALL have its trailing `## Changelog` section
(heading through EOF) removed. Re-running SHALL NOT error on already-stripped files (no `## Changelog`
⇒ no-op).

- **GIVEN** a memory file ending in a `## Changelog` table
- **WHEN** the conversion runs
- **THEN** the file ends at the section preceding `## Changelog`; a second run leaves it unchanged

#### R7: Seed the stripped changelog rows into per-folder `log.seed.md` (DECISION b)
Each stripped `## Changelog` row SHALL become one FKF §6.2 seed entry in that file's folder
`log.seed.md`: date heading from the row's `Date` column, bundle-relative link to the file, the
row's `Summary` cell preserved verbatim (with memory↔memory links inside it rewritten per R8), and
the bare 4-char change-id (Q4) from the row's `Change` column prefix in parens. Entries are
date-grouped newest-first. Seeding SHALL be idempotent (a row already present is not duplicated).

- **GIVEN** a row `| 260612-d9rs-docs-reality-sweep | 2026-06-12 | No-target branch added … |` in
  `memory-docs/hydrate-specs.md`
- **WHEN** the conversion runs
- **THEN** `docs/memory/memory-docs/log.seed.md` carries
  `- **Update** [hydrate-specs](/memory-docs/hydrate-specs.md) — No-target branch added … (d9rs)`
  under `## 2026-06-12`

#### R8: Convert memory↔memory links to bundle-relative (FKF §7, idempotent)
Every memory↔memory link SHALL be converted to the bundle-relative form `](/{domain}/{file}.md)`:
the same-domain forms `]({file}.md)` and `](./{file}.md)` resolve against the file's own domain; the
cross-domain form `](../{domain}/{file}.md)` resolves against the named domain (only when `{domain}`
is one of the five real memory domains). Anchors (`#frag`) are preserved. Links OUT of the bundle
(`../specs/…`, `../../README.md`, `../../../src/…`, external URLs) SHALL be left unchanged. The
rewrite SHALL also run on seed-entry summary cells (R7). Already-bundle-relative links are left
unchanged (idempotent).

- **GIVEN** `](../runtime/operator.md)` in `pipeline/execution-skills.md` and `](schemas.md)` in
  the same file and `](../specs/glossary.md)`
- **WHEN** the conversion runs
- **THEN** they become `](/runtime/operator.md)`, `](/pipeline/schemas.md)`, and
  `](../specs/glossary.md)` (the spec link unchanged); a second run changes nothing

#### R9: Ensure `type: memory` frontmatter on every topic file (idempotent)
Every topic file missing `type: memory` SHALL gain it (alongside the existing `description:`); the
3 files that already carry it are left unchanged. Reserved files (`index.md`, `log.md`,
`log.seed.md`) are exempt. No file SHALL end with a doubled `type:`.

- **GIVEN** a file whose frontmatter has `description:` but no `type:`
- **WHEN** the conversion runs
- **THEN** the frontmatter carries `type: memory` and `description:`; a second run adds nothing

#### R10: Regenerate indexes + logs; clean `--check`
After R6–R9, `fab memory-index` (built from `src/go/fab` 2.5.0 source) SHALL regenerate all
`index.md` and `log.md` files, merging seeds (R1–R3). `fab memory-index --check` SHALL exit 0 or 1
(never 2). Zero memory↔memory `](/…)` links SHALL dangle (every target resolves to an extant file).

- **GIVEN** the converted tree
- **WHEN** `fab memory-index` then `fab memory-index --check` run
- **THEN** `--check` exits 0/1 and a dangling-link sweep finds no broken `](/…)` memory link

### Skills: FKF-Aware Reorg (group b)

#### R11: `docs-reorg-memory` preserves FKF frontmatter + rewrites bundle-relative links
The `src/kit/skills/docs-reorg-memory.md` move logic SHALL preserve each moved file's FKF
frontmatter (`type: memory` + `description:`) and SHALL rewrite **bundle-relative** links (the path
after `/` changes when a file changes domain/sub-domain), replacing the relative-link Link-Impact
examples. It SHALL note that sibling-relative breakage largely disappears under bundle-relative
links (FKF §7 rationale) and confirm the split/merge stub-before-index flow writes the
`description:`-only stub before `fab memory-index`.

- **GIVEN** a `move` migration relocating `runtime-agents.md` from `pipeline/` to
  `pipeline/runtime/`
- **WHEN** the skill rewrites links
- **THEN** inbound links rewrite `](/pipeline/runtime-agents.md)` → `](/pipeline/runtime/runtime-agents.md)`,
  the moved file keeps `type: memory` + `description:`, and the Link-Impact note shows the
  bundle-relative form

#### R12: `docs-reorg-specs` guards against stamping FKF frontmatter on specs (Q3)
The `src/kit/skills/docs-reorg-specs.md` SHALL carry a guard that it never stamps FKF frontmatter
(`type:`/`description:`) on spec files during moves (specs stay human-curated, Constitution VI / FKF
§9). It SHALL NOT add a specs-index generator or borrow (Q3 declined).

- **GIVEN** `docs-reorg-specs` moving a spec file
- **WHEN** the move executes
- **THEN** the moved spec carries no added `type:`/`description:` frontmatter and no specs-index
  generator is introduced

### Specs: SPEC Mirrors (group c)

#### R13: Mirror reorg-skill changes to their SPEC files (Constitution)
`docs/specs/skills/SPEC-docs-reorg-memory.md` and `SPEC-docs-reorg-specs.md` SHALL be updated to
mirror R11/R12 respectively.

- **GIVEN** the R11/R12 skill edits
- **WHEN** the mirrors are reviewed
- **THEN** each SPEC reflects the FKF-awareness (bundle-relative rewrites + frontmatter preservation
  for memory; the no-FKF-stamp guard for specs)

### Docs: `_cli-fab.md` + Memory-Doc Updates

#### R14: Update `_cli-fab.md` § fab memory-index for the seed-merge (Constitution)
The `## fab memory-index` section SHALL document the `log.seed.md` seed input, the merge-beneath
semantics, idempotent preservation, and that seed-merge drift stays benign (tier 1).

- **GIVEN** the Go seed-merge change
- **WHEN** `_cli-fab.md` is read
- **THEN** it documents `log.seed.md` as a curated seed input and the merge/idempotency/loss-tier
  behavior

### Migration: Conversion-as-Record (Q2)

#### R15: Document the one-time conversion in a migration file
A migration file SHALL record the six-part conversion (strip changelog, seed log.seed.md, rewrite
links, add `type:`, build seed-merge, regenerate) so other projects can replay it; this change's
apply stage is the executor for THIS repo (Q2: migration-as-record + apply-as-executor). The
migration SHALL be idempotent.

- **GIVEN** another fab-kit project upgrading to 2.5.0
- **WHEN** `/fab-setup migrations` applies it
- **THEN** the migration instructs the same idempotent conversion steps

## Tasks

### Phase 1: Go Seed-Merge (the crux — do first, the data conversion depends on it)

- [x] T001 Add seed parsing to `src/go/fab/internal/memoryindex/` — a `parseSeedLog(content string) []LogEntry` pure function (new file `seed.go` or in `log.go`) that parses FKF §6.2 date-grouped entries: `## YYYY-MM-DD` headings and `- [**Verb** ][base](/path.md) — summary (id)` lines into `LogEntry{Date, Verb, FileBase, BundleRelPath, Summary, ChangeID}`. Tolerate missing verb/id, multi-word summaries, links inside summary. <!-- R1 -->
- [x] T002 Add `mergeSeedEntries(projected, seed []LogEntry) []LogEntry` pure function that unions the two, de-duplicating any seed entry byte-equal to a projected entry for the same (Date, FileBase, BundleRelPath, ChangeID, Summary); RenderLog's existing date-group + stable sort handles ordering. Ensure git-projected entries sort before seed entries within a date when otherwise equal (stable secondary key). <!-- R2 R3 -->
- [x] T002b Read the seed file in `buildLogTarget` / `GatherLogs`: load `{folderDir}/log.seed.md` (when present), `parseSeedLog` it, `mergeSeedEntries` into the gathered entries before `RenderLog`. A folder with only seed entries (no git history) MUST still emit a `log.md` (relax the GatherLogs `dates==nil` short-circuit only if needed — keep degrade-gracefully when git is wholly absent). <!-- R1 R2 -->
- [x] T003 Exclude `log.seed.md` from `gatherFiles` and `gatherLogEntries` (add `|| name == "log.seed.md"` to both suffix filters in `memoryindex.go`) so it is never a topic row and never re-read as history. Generator never writes it (it is only read). <!-- R4 -->
- [x] T004 Confirm/extend `loss.go` so a seed-merged `log.md` stays tier ≤1 — the `IsLog` guard already routes log.md drift to benign; add a regression test asserting a seed-merge-driven log.md drift classifies benign with empty losses. No new tier-2 category. <!-- R5 -->
- [x] T005 [P] Tests in `memoryindex` package: `parseSeedLog` round-trips RenderLog output (parse∘render identity on §6.2 entries); `mergeSeedEntries` de-dups exact matches and preserves distinct ones; a real-git-repo GatherLogs test with a seeded `log.seed.md` proves merged output + second-pass byte-stability (idempotent, no dupes); seed-only folder (no git) emits a log.md; `log.seed.md` excluded from topic rows. <!-- R1 R2 R3 R4 -->
- [x] T006 Build (`cd src/go/fab && go build ./...`) and install/locate the freshly-built binary for use in the data-conversion validation (the installed 2.4.2 binary lacks seed-merge — MUST use the 2.5.0 build). <!-- R10 -->

### Phase 2: One-Time Data Conversion (apply-driven; depends on Phase 1 binary)

- [x] T007 Build the seed `log.seed.md` files: for each of the 5 domain folders, parse every topic file's `## Changelog` rows and emit a `log.seed.md` (FKF §6.2: date-grouped newest-first, bundle-relative link, verbatim Summary with R8 link rewrite applied, bare 4-char id). This is the faithful capture of all 651 rows BEFORE stripping. Idempotent (merge-not-duplicate if a seed file already exists). <!-- R7 -->
- [x] T008 Convert memory↔memory links to bundle-relative across all 20 topic files AND inside the seed summaries written in T007: same-domain `]({f}.md)`/`](./{f}.md)` → `](/{thisdomain}/{f}.md)`; cross-domain `](../{domain}/{f}.md)` → `](/{domain}/{f}.md)` only for the 5 real domains; preserve `#anchors`; leave `../specs/…`, `../../README.md`, `../../../src/…`, URLs untouched. Idempotent. <!-- R8 -->
- [x] T009 Add `type: memory` to the frontmatter of the 17 topic files missing it (leave the 3 that have it); never double-stamp. <!-- R9 -->
- [x] T010 Strip the trailing `## Changelog` section (heading→EOF) from all 20 topic files. Run AFTER T007 (seeding reads the changelog) and T008 (link rewrite touches changelog summary cells before they are seeded — order T007→T008 inside seeding, then strip). Idempotent. <!-- R6 -->
- [x] T011 Regenerate indexes + logs with the Phase-1 binary (`fab memory-index`), then validate `fab memory-index --check` exits 0/1 (never 2). <!-- R10 -->
- [x] T012 Dangling-link sweep: assert every memory↔memory `](/…)` link across `docs/memory/**` resolves to an extant file (zero dangling). <!-- R10 -->

### Phase 3: Skills + Specs + Docs (depends on nothing in Phase 2; can follow Phase 1)

- [x] T013 [P] Update `src/kit/skills/docs-reorg-memory.md`: rewrite the Link Impact note + rewrite rule to the bundle-relative form, state that moves preserve FKF frontmatter (`type: memory` + `description:`), note sibling-relative breakage disappears under §7, confirm stub-before-index writes the `description:`-only stub before `fab memory-index`. <!-- R11 -->
- [x] T014 [P] Update `src/kit/skills/docs-reorg-specs.md`: add the guard that it never stamps FKF frontmatter on spec moves (Constitution VI / FKF §9); keep the existing "no specs-index generator" note (Q3 declined the borrow). <!-- R12 -->
- [x] T015 [P] Update `docs/specs/skills/SPEC-docs-reorg-memory.md` mirror for T013 (bundle-relative rewrites + frontmatter preservation). <!-- R13 -->
- [x] T016 [P] Update `docs/specs/skills/SPEC-docs-reorg-specs.md` mirror for T014 (no-FKF-stamp guard). <!-- R13 -->
- [x] T017 [P] Update `src/kit/skills/_cli-fab.md` § fab memory-index: document the `log.seed.md` seed input, merge-beneath-projected, idempotent preservation, seed-merge drift stays benign (tier 1). <!-- R14 -->
- [x] T018 [P] Add a migration file in `src/kit/migrations/` recording the idempotent conversion steps (Q2: migration-as-record). Fold into `2.4.2-to-2.5.0.md` or add a new `*.md` — choose to match the existing migration cadence. <!-- R15 -->

### Phase 4: Memory-doc consistency (mechanical, follows the cutover)

- [x] T019 Update `docs/memory/memory-docs/templates.md` and `hydrate.md` prose to reflect the post-cutover state (tree fully FKF: all files `type: memory`, no `## Changelog`, bundle-relative links; reorg FKF-awareness from T013). These are themselves topic files — apply the conversion (T008/T009/T010) to them too, then re-regenerate. NOTE: hydrate (a later pipeline stage) owns deep memory writes; here keep edits minimal + consistency-only, and re-run `fab memory-index` after. <!-- R10 -->

## Execution Order

- Phase 1 (T001→T006) is the foundation: the seed-merge binary must exist before the data
  conversion can be validated.
- Inside Phase 2, the per-folder order is: T007 (seed, with T008 link-rewrite applied to summaries)
  → T009 (type:) → T010 (strip changelog) → T011 (regen) → T012 (dangling sweep). Seeding MUST read
  the changelog before stripping.
- Phase 3 is independent of Phase 2 (skills/specs/docs) and may run alongside it after Phase 1.
- T019 (Phase 4) runs last and triggers a final `fab memory-index` regen.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `fab memory-index` reads + parses `log.seed.md` into LogEntry values preserving seed dates and verbatim summaries
- [ ] A-002 R2: seed and git-projected entries merge under unified date headings in a deterministic order
- [ ] A-003 R3: a second `fab memory-index` run is byte-stable (no duplicated seed entries); `--check` exits 0
- [ ] A-004 R4: `log.seed.md` is never a topic-index row and is never overwritten by the generator
- [ ] A-005 R5: a seed-merge-driven `log.md` drift classifies benign (tier ≤1) with empty losses on `--check`
- [ ] A-006 R6: all 20 memory files have their `## Changelog` removed; re-strip is a no-op
- [ ] A-007 R7: all 651 changelog rows are preserved as `log.seed.md` seed entries (date + verbatim summary + bare 4-char id), bundle-relative links applied inside summaries
- [ ] A-008 R8: every memory↔memory link is bundle-relative; out-of-bundle links unchanged; re-run idempotent
- [ ] A-009 R9: every topic file carries `type: memory` (17 added, 3 untouched); no doubled `type:`
- [ ] A-010 R10: `fab memory-index --check` (2.5.0 build) exits 0/1 (never 2) on the converted tree
- [ ] A-011 R11: `docs-reorg-memory.md` preserves FKF frontmatter on moves and rewrites bundle-relative links (Link Impact note updated)
- [ ] A-012 R12: `docs-reorg-specs.md` guards against stamping FKF frontmatter on spec moves; no specs-index generator added
- [ ] A-013 R13: both SPEC mirrors reflect the skill changes
- [ ] A-014 R14: `_cli-fab.md` § fab memory-index documents the seed-merge input + semantics
- [ ] A-015 R15: a migration file records the idempotent conversion steps

### Behavioral Correctness

- [ ] A-016 R8: `../specs/glossary.md`, `../../README.md`, `../../../src/…` links remain repo-relative (not bundle-relativized)
- [ ] A-017 R7: seed entry dates come from the changelog `Date` column (not git commit dates)

### Scenario Coverage

- [ ] A-018 R12: zero dangling memory↔memory `](/…)` links across `docs/memory/**` after conversion

### Edge Cases & Error Handling

- [ ] A-019 R1: a folder with no `log.seed.md` behaves exactly as before (no error, pure git projection)
- [ ] A-020 R6: a file with no `## Changelog` (already stripped) is a strip no-op

### Code Quality

- [ ] A-021 Pattern consistency: new Go code follows the package's pure-function-plus-Gather-I/O split, named constants over magic strings, and the existing test idioms (writeFile/gitDateRun, golden byte-compares)
- [ ] A-022 No unnecessary duplication: seed parsing reuses RenderLog's entry shape and the existing date/sort helpers rather than reimplementing them
- [ ] A-023 Readability: `parseSeedLog`/`mergeSeedEntries` stay focused (<50 lines each) per code-quality.md anti-patterns

### Documentation Accuracy & Cross-References (extra_categories)

- [ ] A-024 documentation_accuracy: `_cli-fab.md`, the two SPEC mirrors, the migration, and templates.md/hydrate.md describe the shipped behavior exactly
- [ ] A-025 cross_references: bundle-relative links resolve; SPEC mirrors name their `src/kit/skills/` sources; the migration cites `docs/specs/fkf.md` §10

## Notes

- Check items as you review: `- [x]`
- The 2.5.0-source binary (not the installed 2.4.2) MUST drive every `fab memory-index` validation.
- `src/kit/` is canonical; `.claude/skills/` is gitignored — never edit the deployed copies.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Q2 → migration-file-as-record + this change's apply stage as executor for THIS repo | Intake recorded this as the leaning default (Tentative there); the 651-row faithful prose conversion is an agent-driven pass ill-suited to a declarative migration, while the migration file documents replay for other projects. Apply executing it here is the natural fit; reversible (migration text is editable) | S:60 R:65 A:65 D:60 |
| 2 | Certain | Q4 → seed-log change-id is the bare 4-char id `(d9rs)`, matching what Change-2's generator already emits | Verified the live generated logs: `docs/memory/pipeline/log.md` emits `(5943)` (bare 4-char), and the golden tests assert `(l3ja)`/`(aaaa)`. The §6.2 spec example shows the full `(YYMMDD-XXXX)` but the generator (the alignment target per Q4) emits bare 4-char | S:90 R:90 A:100 D:95 |
| 3 | Confident | Seed mechanism = per-folder `log.seed.md` sidecar input (not in-place log.md parsing, not a CLI flag) | Keeps `log.md` a pure generated output (single-writer, FKF §5/§6), makes the seed an explicit curated input like `description:` frontmatter, is trivially idempotent (stable file on disk), and adds zero CLI surface (Constitution I). In-place log.md parsing can't distinguish seed from projected reliably; a `--seed` flag adds CLI surface for no gain | S:70 R:55 A:75 D:65 |
| 4 | Confident | Seed entries preserve the full Summary cell verbatim (not truncated to a §6.2 one-liner) | DECISION b mandates faithful preservation of all 651 rows; the §6.2 "one line, not paragraph-prose" guidance governs git-projected entries authored going forward, while the seed is a one-time faithful capture of pre-FKF history. The seed is curated input, so it may carry the original prose | S:80 R:60 A:85 D:70 |
| 5 | Confident | Seed entry date = the changelog row's `Date` column, not the file's git commit date | The pre-FKF rows carry historical authored dates (e.g. 2026-02-09) that predate/differ from git; faithfulness (DECISION b) requires preserving the row's own date as the §6.2 date heading | S:85 R:70 A:85 D:80 |
| 6 | Confident | Bundle-relative cross-domain rewrite fires ONLY for `../{d}/` where `{d}` ∈ {_shared, distribution, memory-docs, pipeline, runtime}; `../specs/` and deeper `../../` escapes stay repo-relative | FKF §7: only memory↔memory links are bundle-relative; `../specs/glossary.md` resolves to docs/specs (out of bundle). Gating on the five real domains prevents mis-relativizing spec/README/src links | S:85 R:75 A:90 D:85 |
| 7 | Confident | T019 keeps memory-doc edits minimal/consistency-only (hydrate, a later stage, owns deep memory writes) | Constitution II + the pipeline model: hydrate is the memory writer; apply edits these two files only for cutover-consistency (they are themselves topic files needing the mechanical conversion) and re-regenerates | S:70 R:70 A:75 D:65 |

7 assumptions (1 certain, 6 confident, 0 tentative).
