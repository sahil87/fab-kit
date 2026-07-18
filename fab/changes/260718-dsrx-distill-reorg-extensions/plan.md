# Plan: Distill/Reorg Extensions — Drain the Accretion Debt

**Change**: 260718-dsrx-distill-reorg-extensions
**Intake**: `intake.md`

## Requirements

### Distill: Four New Removal Classes

#### R1: Change-id heading suffixes stripped
`/docs-distill-memory` MUST recognize and strip change-id tokens from body **headings** — a heading carrying a `(xu0k)`-style or `— 260718-mxgu`-style token has the token removed, keeping the heading text (FKF §3.3: "a heading is `## Dispatch States`, never `### Dispatch States (xu0k)`"). Token recognition follows the registry-gated posture: a full `YYMMDD-XXXX-slug` token always matches; a bare 4-char id matches only when registry-plausible. Displaced provenance worth keeping becomes a trailing `(change-id)` citation in the section body. The class is added to Step 1 (identify), Step 2 (report), and Step 4 (apply), citing `$(fab kit-path)/reference/fkf.md` §3.3.

- **GIVEN** a topic file with a heading `### Dispatch States (xu0k)` (xu0k registry-plausible)
- **WHEN** distillation runs and the file is approved
- **THEN** the heading becomes `### Dispatch States`, and any provenance worth keeping survives as a trailing `(xu0k)` citation in the section body

#### R2: Literal duplicate headings/blocks deduped
`/docs-distill-memory` MUST detect **byte-identical** duplicated heading pairs/blocks within a file and remove the later duplicate on apply. **Near-duplicates MUST be flagged in the Step 2 report for manual review, never auto-merged** — content judgment stays with the human gate. Cited to FKF §3.3.

- **GIVEN** a topic file carrying two byte-identical `## Foo` heading blocks
- **WHEN** distillation runs and the file is approved
- **THEN** the later byte-identical duplicate is removed; a merely *similar* block is instead flagged in the report and left untouched

#### R3: Design-Decisions changelog bullets rewritten
`/docs-distill-memory` MUST handle a `- **{change-id} — retired X**`-shaped changelog bullet inside `## Design Decisions` (the shape FKF §3.3 bans there): when it encodes a durable decision → rewrite to the four-field entry (**Decision** / **Why** / **Rejected** / *Introduced by* — the change-id moves into *Introduced by* or a trailing citation); when it is pure change history already recorded in `log.md`/git → remove under the existing deletion-safety rule. It MUST NOT fabricate rationale: when Why/Rejected content is not derivable, the rewritten entry carries only the fields that exist (Decision + *Introduced by*). Cited to FKF §3.3.

- **GIVEN** a `## Design Decisions` section containing `- **260703-gvxd — retired the poll shim**` with surrounding context describing why
- **WHEN** distillation runs and the file is approved
- **THEN** the bullet becomes a four-field DD entry with the change-id in *Introduced by*; a bullet with no derivable rationale that is pure change history already in `log.md` is removed instead, and no rationale is invented

#### R4: Embedded operational TODOs relocated to backlog (never deleted)
`/docs-distill-memory` MUST **relocate** an embedded operational TODO out of a memory body into `fab/backlog.md` (FKF §3.3: follow-up work items belong in the project backlog or change folder) — never delete it. Relocation appends a standard backlog entry `- [ ] [{fresh-4char-id}] {YYYY-MM-DD}: {TODO text} (relocated from docs/memory/{domain}/{file}.md by /docs-distill-memory)`. When `fab/backlog.md` does not exist it MUST be created with a minimal `# Backlog` header. Relocation honors the Step 3 per-file approval unit — a file the user skips or cherry-picks away keeps its TODOs (no orphaned relocations). Cited to FKF §3.3.

- **GIVEN** a memory body carrying a `TODO: delete the stale gh secret` follow-up and an approved file
- **WHEN** distillation runs
- **THEN** the TODO is appended to `fab/backlog.md` as a fresh-id entry noting the source path, and removed from the memory body; if the user skips that file, its TODO stays in place

#### R5: Distill Step 2 report + completion counters extended
`/docs-distill-memory` Step 2 report SHOULD gain matching entries for the four classes (e.g. `strip change-id heading suffixes: N`, `dedupe byte-identical blocks: N (near-duplicates flagged: M)`, `rewrite DD changelog bullets: N`, `RELOCATE TODOs → fab/backlog.md: N`), and the completion line's counters SHOULD extend accordingly.

- **GIVEN** a domain with instances of the four new classes
- **WHEN** the Step 2 report and completion line render
- **THEN** each class is represented with its count in both

### Distill: Survey Consumes the Machine Surface

#### R6: Survey signal source switches to `fab memory-index --check --json`
`/docs-distill-memory` Step 0 survey MUST replace the agent-side grep heuristics with **one `fab memory-index --check --json` invocation**, aggregating per-domain flagged-file counts from: `malformed[]` kinds `description-change-id` + `description-over-cap` (blocking class), and `warnings[]` kinds `description-length` (advisory 501–1000 band) + `narration-density`. A file with multiple findings counts once; a sub-domain file rolls up to its domain (first path segment under `docs/memory/`). The check's exit code MUST NOT gate the survey (exit 1/2 still surveys). Survey output format, the heuristic caveat, and auto-pick semantics are unchanged.

- **GIVEN** a corpus with over-cap descriptions, change-id descriptions, over-length descriptions, and narration-heavy bodies
- **WHEN** the no-arg survey runs against a `--json`-capable binary
- **THEN** per-domain flagged-file counts derive from the JSON `malformed[]`/`warnings[]` arrays (each file counted once, sub-domain files rolled up to their domain), and a non-zero check exit still produces a survey

#### R7: Survey older-binary fallback
`/docs-distill-memory` survey MUST fall back to the current grep heuristics verbatim and warn the user to upgrade `fab` when `--json` is unavailable or the `warnings` key is absent — mirroring `/docs-reorg-memory`'s established older-binary fallback posture.

- **GIVEN** a `fab` binary that does not emit the `warnings` key
- **WHEN** the survey runs
- **THEN** it uses the legacy three-class grep heuristics and warns to upgrade `fab`

### Reorg: File-Splitting

#### R8: Shape Report gains file rows (split candidates)
`/docs-reorg-memory` MUST extend the Shape Report (today folder-only) with **file rows**: any topic file exceeding `[mxgu]`'s thresholds (~400 lines OR ~15KB) is a split *candidate*, sourced from the same `fab memory-index --check --json` call Step 1 already makes (`warnings[]` kind `file-size`, carrying `count` = lines and `bytes`). Older-binary fallback: measure during the read-all-files pass. The reactive posture MUST be preserved — a flagged file is *proposed* for splitting only when its heading clusters show ≥2 genuine topics; a long-but-cohesive file is reported (`⚠ over size — long but cohesive; no split proposed`) and left alone.

- **GIVEN** a 2,000-line topic file whose headings cluster into 5 distinct topics
- **WHEN** reorg's Shape Report renders
- **THEN** the file appears as a `split-file` candidate; a long-but-cohesive file is reported over-size but no split is proposed

#### R9: New Migration Map Kind `split-file`
`/docs-reorg-memory` MUST add a Migration Map `Kind`: `split-file` — fan one multi-topic file into ≥2 topic files in the same domain/sub-domain (file-granularity parallel of `split-domain`). Each new file gets `type: memory` + a fresh change-id-free `description:` (same rule as split-domain's new files). Body content MUST move **verbatim** (restyling remains distill's job). The original path is kept for the dominant topic when one exists, else removed. A split that pushes folder width past ~12 MAY chain into the existing `split-domain` flow. New files target ~300 lines. Link Impact MUST extend to `split-file`: an **anchored** inbound bundle-relative link (`#heading`) follows the file its heading moved to; an **un-anchored** link retargets to the dominant-topic file; ambiguity (no dominant topic + un-anchored inbound links) → the existing abort escape (roll back that migration, regenerate, continue).

- **GIVEN** an approved `split-file` of a multi-topic file with inbound anchored + un-anchored bundle-relative links and a dominant topic
- **WHEN** reorg applies it
- **THEN** the file fans into ≥2 topic files (verbatim bodies, new `type: memory` + change-id-free `description:`), anchored inbound links follow their heading's new file, un-anchored links retarget to the dominant-topic file, and an ambiguous case takes the abort escape

### Reorg: Duplicate-Coverage Detection + `_unsorted/` Triage

#### R10: Duplicate-coverage detection pass
`/docs-reorg-memory` MUST add a duplicate-coverage analysis pass flagging the same topic covered in 2+ files (signals: near-identical filenames/descriptions, the same filename in two domains, heavy heading overlap). Output: a `## Duplicate Coverage` table (topic / files / evidence / proposed canonical home). Remediation rides the Migration Map: a **new `Kind`: `merge-file`** (move B's unique sections into canonical file A via the move-section machinery, rewrite all inbound links to A, delete the emptied B — file-granularity parallel of `merge-domain`, with Link Impact + the no-dangling-link guard), or plain `move-section` rows for partial overlap. The report MUST note the tie to the open single-sourcing seam audit as a cross-reference, not scope.

- **GIVEN** two files covering the same topic (e.g. `mock-infrastructure.md` and `msw-mock-infrastructure.md`)
- **WHEN** reorg runs
- **THEN** a `## Duplicate Coverage` table flags them with evidence + a proposed canonical home, and remediation offers a `merge-file` (or `move-section` for partial overlap) with Link Impact and the no-dangling-link guard

#### R11: `_unsorted/` staging triage
`/docs-reorg-memory` MUST add an `_unsorted/` triage listing while keeping `_unsorted/`'s bounds exemption (never split/merged/flattened). Every staged topic file gets a per-file proposal: **`move`** to a named domain (existing kind — the default), or **`delete`** for stale ephemera superseded/recorded elsewhere, each deletion requiring explicit per-file confirmation. Signal: `warnings[]` kind `unsorted-nonempty`; fallback: direct folder listing.

- **GIVEN** `_unsorted/` holding stale session notes for a shipped change plus a genuinely unplaced note
- **WHEN** reorg runs
- **THEN** each staged file gets a per-file `move`/`delete` proposal (`move` default), a `delete` requires explicit per-file confirmation, and `_unsorted/` is never split/merged/flattened

#### R12: Reorg Step 1 records `warnings[]`
`/docs-reorg-memory` Step 1's existing `--check --json` parse MUST additionally record `warnings[]` (`file-size` → Shape Report file rows; `unsorted-nonempty` → triage pass) — one call feeds compatibility detection and the two new passes.

- **GIVEN** the single `fab memory-index --check --json` call Step 1 already makes
- **WHEN** it returns
- **THEN** its `warnings[]` array is recorded and feeds the Shape Report file rows and the `_unsorted/` triage pass, in addition to the existing `losses[]` compatibility detection

### CLI: `description-length` Joins the JSON `warnings[]`

#### R13: `KindDescriptionLength` added to the JSON `warnings[]` switch
`fab memory-index --check --json` MUST include `KindDescriptionLength` (`description-length`) in the JSON `warnings[]` array switch in `src/go/fab/cmd/fab/memory_index.go` (today only `KindNarrationDensity`/`KindFileSize`/`KindUnsorted`/`KindBrokenLink` join the array). The change is additive per the established `warnings[]` contract — existing `tier`/`drift`/`losses`/`malformed`/`warnings` consumers are unaffected, and no CLI signature changes. The finding carries its rune length in `count`.

- **GIVEN** a topic file with a `description:` in the 501–1000 band on an otherwise byte-clean tree
- **WHEN** `fab memory-index --check --json` runs
- **THEN** a `{"kind":"description-length", "path":..., "count":<runes>}` object appears in the `warnings[]` array, the exit code is unchanged (advisory — never blocks), and the pre-existing kinds still appear

#### R14: Go test coverage for the new warnings kind
The Go change MUST ship a test update (Constitution VII / Test Integrity) asserting `description-length` appears in the `--check --json` `warnings[]` array on an advisory-only tier-0 tree, with its `count` carrying the rune length and the existing additive-array contract intact.

- **GIVEN** the extended switch
- **WHEN** the scoped `cmd/fab` tests run
- **THEN** a test asserts the `description-length` warning rides `warnings[]` with a populated `count`, and all pre-existing memory-index tests still pass

### Docs: Constitution Mirror Obligations

#### R15: SPEC mirrors updated for every changed skill source
Every changed `src/kit/skills/*.md` MUST update its `docs/specs/skills/SPEC-*.md` mirror in the same change (Constitution Additional Constraints): `SPEC-docs-distill-memory.md` (R1–R7), `SPEC-docs-reorg-memory.md` (R8–R12), `SPEC-_cli-fab.md` (R13).

- **GIVEN** edits to `docs-distill-memory.md`, `docs-reorg-memory.md`, and `_cli-fab.md`
- **WHEN** the change is reviewed
- **THEN** each source's SPEC mirror reflects the new behavior

#### R16: `_cli-fab.md` § fab memory-index JSON-shape doc updated
`src/kit/skills/_cli-fab.md` § fab memory-index MUST document `description-length` joining the JSON `warnings[]` array (kind enum + count semantics), correcting the prior "deliberately excluded from the array" note. Its SPEC mirror `SPEC-_cli-fab.md` MUST be swept correspondingly (R15).

- **GIVEN** the JSON `warnings[]` gains `description-length`
- **WHEN** `_cli-fab.md` and its mirror render
- **THEN** the JSON-shape enum lists `description-length`, the count semantics are stated, and the stale "excluded" note is gone

#### R17: Aggregate `docs/specs/skills.md` swept
The aggregate `docs/specs/skills.md` MUST be swept per `code-quality.md` § Sibling & Mirror Sweeps — updating the `/docs-distill-memory` and `/docs-reorg-memory` sections (and any restated memory-index/warnings facts) so no stale claim (the survey's agent-side-grep description, the reorg Shape-Report "folder-only" claim, the Migration Map Kind enum) survives repo-wide.

- **GIVEN** the behavior changes in R1–R13
- **WHEN** `grep`-ing the old claims across the mirror class
- **THEN** `docs/specs/skills.md` carries no stale distill/reorg/warnings claim

### Non-Goals

- The no-arg survey mode + dynamic `Next:` line themselves (shipped via 260718-ukpf / PR #498) — R6 only swaps the survey's signal source.
- `[mxgu]`'s warning thresholds and blocking/advisory classification (shipped; consumed as-is).
- `[wrct]`'s writer rules and FKF §3.3 text (shipped; cited as-is).
- Actually running the remediation over any corpus (this change extends the skills; running them is per-repo operator work).
- The single-sourcing seam audit itself (the duplicate-coverage report cross-references it only).
- `docs/memory/` topic-file edits (memory updates happen at hydrate, not apply).
- A migration (no user-data restructuring; the JSON change is additive; no CLI signature changes).

### Design Decisions

#### `split-file` / `merge-file` as file-granularity parallels of `split-domain` / `merge-domain`
**Decision**: Add two new Migration Map kinds `split-file` / `merge-file` rather than new machinery, each reusing the existing split/merge apply path, Link Impact rules, no-dangling-link guard, and abort escape at file granularity.
**Why**: Cleanest fit to the existing Kind enum and apply machinery; reuses the proven bundle-relative link-rewrite rules; keeps reorg's reactive propose-then-apply posture.
**Rejected**: A separate splitting subsystem — duplicates the move/link-rewrite machinery reorg already owns.
*Introduced by*: 260718-dsrx-distill-reorg-extensions

#### Within-file dedup is distill's, cross-file duplication is reorg's
**Decision**: Distill class (b) auto-removes only byte-identical *within-file* duplicates (near-duplicates flagged for manual review); cross-file duplicate coverage belongs to reorg's duplicate-coverage pass.
**Why**: Byte-equality is mechanically safe; near-duplicate and cross-file merging needs content judgment the human gate should see. Within-file = distill (prose remediation), cross-file = reorg (structure) matches the skills' existing division of labor.
**Rejected**: Auto-merging near-duplicates or cross-file duplicates — needs content judgment; risks silent contract loss.
*Introduced by*: 260718-dsrx-distill-reorg-extensions

#### Survey consumes the `[mxgu]` machine surface, not a hybrid grep
**Decision**: The survey's signal source becomes a single `fab memory-index --check --json` call; the residual agent-side length check is retired by adding `description-length` to the JSON `warnings[]`.
**Why**: "One canonical signal source" is the item's stated goal; `_cli-fab.md` already names `[dsrx]` as the `warnings[]` consumer; the kind constant already exists; `warnings[]` is additive by contract.
**Rejected**: Keeping a hybrid agent-side length check — perpetuates the dual-source drift this item removes.
*Introduced by*: 260718-dsrx-distill-reorg-extensions

## Tasks

### Phase 1: CLI (Go) — the machine surface the survey consumes

- [x] T001 Add `memoryindex.KindDescriptionLength` to the JSON `warnings[]` switch in `src/go/fab/cmd/fab/memory_index.go` (the `switch w.Kind` at the advisory-array append), and update the two now-stale flag help/comment notes that say the 501–1000 length nag is "deliberately excluded from the array" (the `--json` flag `Short` and the code comment above the switch) <!-- R13 -->
- [x] T002 Add a scoped test to `src/go/fab/cmd/fab/memory_index_test.go` asserting a `description-length` finding rides `--check --json` `warnings[]` with a populated `count` on an advisory-only tier-0 tree, and the existing additive-array contract is intact <!-- R14 -->
- [x] T003 Run the scoped Go tests (`go test ./cmd/fab/...` under `src/go/fab`); fix failures until green <!-- R14 -->

### Phase 2: Skill sources — distill

- [x] T004 Extend `src/kit/skills/docs-distill-memory.md` Behavior Step 1 (identify), Step 2 (per-file report), and Step 4 (apply) with the four new removal classes — change-id heading suffixes (R1), byte-identical duplicate blocks + near-duplicate flagging (R2), Design-Decisions changelog-bullet rewrite with no-fabrication guard (R3), and operational-TODO relocation to `fab/backlog.md` (R4) — each citing `$(fab kit-path)/reference/fkf.md` §3.3; extend the Step 2 report lines and the completion counter line (R5); update the Output block and Key Properties table to match <!-- R1 --> <!-- R2 --> <!-- R3 --> <!-- R4 --> <!-- R5 -->
- [x] T005 Switch `src/kit/skills/docs-distill-memory.md` Step 0 survey + its Context-Loading survey paragraph to consume a single `fab memory-index --check --json` call (aggregate `malformed[]` `description-change-id`/`description-over-cap` + `warnings[]` `description-length`/`narration-density`; count a multi-finding file once; roll sub-domain files up to their domain; exit code does not gate), with the verbatim older-binary grep fallback + upgrade warning; update the Tools-used/Behavior prose that describes the agent-side grep <!-- R6 --> <!-- R7 --> <!-- rework: review must-fix — the authored claim "the exclusion set is already honored by the primitive... so the survey does not re-apply it" is FALSE (verified vs memoryindex.go frontmatterWarnings: index.md stubs ARE scanned for all three description-tier kinds, and _shared/removed-domains.md is scanned as an ordinary topic file incl. narration-density). Replace with: the survey RE-APPLIES the distillation exclusion set to the JSON finding paths — drop findings whose path is an index.md or _shared/removed-domains.md (log.md/log.seed.md are already skipped by the walker) — preserving the "fully-distilled tree surveys clean" / unchanged auto-pick semantics -->

### Phase 3: Skill sources — reorg

- [x] T006 Extend `src/kit/skills/docs-reorg-memory.md`: Shape Report gains file rows (split candidates from `warnings[]` `file-size`, reactive ≥2-topic posture, long-but-cohesive reported not split) (R8); add Migration Map `Kind: split-file` with verbatim-body / new-frontmatter / dominant-topic rules and the anchored-vs-un-anchored Link Impact + abort escape (R9); record `warnings[]` in Step 1 alongside `losses[]` (R12) <!-- R8 --> <!-- R9 --> <!-- R12 -->
- [x] T007 Extend `src/kit/skills/docs-reorg-memory.md` with the duplicate-coverage detection pass (`## Duplicate Coverage` table + `Kind: merge-file` with Link Impact + no-dangling-link guard, cross-reference to the single-sourcing seam audit) (R10) and the `_unsorted/` staging triage (per-file `move`/`delete` proposals, `delete` needs per-file confirmation, `_unsorted/` keeps its bounds exemption, signal `warnings[]` `unsorted-nonempty` + folder-listing fallback) (R11); update the Output block, Error Handling, and Key Properties table to match <!-- R10 --> <!-- R11 -->

### Phase 4: CLI docs + SPEC mirrors + aggregate sweep

- [x] T008 Update `src/kit/skills/_cli-fab.md` § fab memory-index: add `description-length` to the JSON `warnings[]` kind enum in the `--json` shape line, state its `count` = rune length semantics, and correct the ADVISORY `description-length` bullet + any note claiming it is excluded from the array <!-- R16 -->
- [x] T009 Sweep `docs/specs/skills/SPEC-docs-distill-memory.md` to mirror the four new removal classes (R1–R5) and the machine-surface survey switch + older-binary fallback (R6–R7) <!-- R15 --> <!-- rework: mirror T005's corrected exclusion-set rule (survey re-applies the exclusion set to JSON finding paths) — the SPEC flow line currently restates the false "primitive honors the exclusion set" claim -->
- [x] T010 Sweep `docs/specs/skills/SPEC-docs-reorg-memory.md` to mirror the file-splitting Shape Report + `split-file` kind (R8–R9), duplicate-coverage + `merge-file` + `_unsorted/` triage (R10–R11), and the `warnings[]` Step-1 recording (R12) <!-- R15 -->
- [x] T011 Sweep `docs/specs/skills/SPEC-_cli-fab.md` § fab memory-index row to mirror `description-length` joining the JSON `warnings[]` array (R13/R16) <!-- R15 -->
- [x] T012 Sweep the aggregate `docs/specs/skills.md` `/docs-distill-memory` and `/docs-reorg-memory` sections (and any restated warnings/memory-index facts) so no stale claim survives — the agent-side-grep survey description, the folder-only Shape Report, the Migration Map Kind enum <!-- R17 -->

### Phase 5: Verify docs consistency

- [x] T013 Grep the mirror class repo-wide for the retired claims (agent-side grep survey, "folder-only" Shape Report, the old Migration Map Kind enum lists, "description-length ... excluded from the array") to confirm no stale occurrence remains in `src/kit/skills/` or `docs/specs/` <!-- R17 -->

## Execution Order

- T001 → T002 → T003 (Go: implement, test, run — sequential)
- T002 blocked by T001; T003 blocked by T002
- T004, T005 (distill source) independent of T006, T007 (reorg source) — different files
- T008 (`_cli-fab.md`) can run alongside the skill edits (different file)
- T009 blocked by T004+T005; T010 blocked by T006+T007; T011 blocked by T008 (each SPEC mirror follows its source)
- T012, T013 last (aggregate sweep + repo-wide grep verification, after all sources + mirrors land)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `docs-distill-memory.md` Steps 1/2/4 strip change-id heading suffixes (registry-gated tokens; displaced provenance → trailing citation), citing FKF §3.3
- [x] A-002 R2: `docs-distill-memory.md` auto-removes byte-identical duplicate blocks and flags near-duplicates for manual review (never auto-merged)
- [x] A-003 R3: `docs-distill-memory.md` rewrites DD changelog bullets to the four-field entry (durable) or removes them (pure history), with an explicit no-fabricated-rationale guard
- [x] A-004 R4: `docs-distill-memory.md` relocates operational TODOs to `fab/backlog.md` (append standard entry, create with `# Backlog` header when absent, honor per-file approval — never delete)
- [x] A-005 R5: `docs-distill-memory.md` Step 2 report lines + completion counters cover the four new classes
- [x] A-006 R6: `docs-distill-memory.md` Step 0 survey consumes a single `fab memory-index --check --json` call, aggregating `malformed[]` (`description-change-id`/`description-over-cap`) + `warnings[]` (`description-length`/`narration-density`), counting a file once, rolling sub-domain files up to their domain, not gating on exit code
- [x] A-007 R7: `docs-distill-memory.md` survey falls back to the verbatim grep heuristics + an upgrade warning when `--json`/`warnings` is unavailable
- [x] A-008 R8: `docs-reorg-memory.md` Shape Report carries file rows (split candidates from `file-size`, reactive ≥2-topic posture, long-but-cohesive reported not split)
- [x] A-009 R9: `docs-reorg-memory.md` defines `Kind: split-file` (verbatim bodies, new `type: memory` + change-id-free `description:`, dominant-topic retention, anchored-vs-un-anchored Link Impact, abort escape)
- [x] A-010 R10: `docs-reorg-memory.md` adds duplicate-coverage detection (`## Duplicate Coverage` table + `Kind: merge-file` with Link Impact + no-dangling-link guard + single-sourcing-audit cross-reference)
- [x] A-011 R11: `docs-reorg-memory.md` adds `_unsorted/` triage (per-file `move`/`delete`, `delete` needs per-file confirmation, bounds exemption kept, `unsorted-nonempty` signal + folder-listing fallback)
- [x] A-012 R12: `docs-reorg-memory.md` Step 1 records `warnings[]` (`file-size` → file rows, `unsorted-nonempty` → triage) from the one existing `--check --json` call
- [x] A-013 R13: `memory_index.go` includes `KindDescriptionLength` in the JSON `warnings[]` switch; the finding carries its rune length in `count`; no CLI signature changes
- [x] A-014 R15: each changed skill source (`docs-distill-memory.md`, `docs-reorg-memory.md`, `_cli-fab.md`) has its `docs/specs/skills/SPEC-*.md` mirror updated in this change
- [x] A-015 R16: `_cli-fab.md` § fab memory-index documents `description-length` in the JSON `warnings[]` enum with count semantics; the stale "excluded from the array" note is gone
- [x] A-016 R17: `docs/specs/skills.md` `/docs-distill-memory` + `/docs-reorg-memory` sections carry no stale distill/reorg/warnings claim

### Behavioral Correctness

- [x] A-017 R6: the survey's per-domain counts derive from the JSON arrays (not agent-side grep) when the binary supports `--json`, and a non-zero check exit still surveys
- [x] A-018 R9: an anchored inbound link follows its heading's new file, an un-anchored inbound link retargets to the dominant-topic file, and a no-dominant-topic + un-anchored-links case takes the abort escape
- [x] A-019 R13: the `--check --json` `warnings[]` gains `description-length` while `tier`/`drift`/`losses`/`malformed` and the four pre-existing warning kinds are unchanged (additive, existing consumers unaffected)

### Scenario Coverage

- [x] A-020 R14: a scoped Go test asserts `description-length` rides `--check --json` `warnings[]` with a populated `count`; `go test ./cmd/fab/...` is green
- [x] A-021 R2: a near-duplicate block is flagged in the report and left untouched (not auto-merged) — the human-gate boundary is stated
- [x] A-022 R4: a skipped/cherry-picked-away file keeps its TODOs (no orphaned relocation)

### Edge Cases & Error Handling

- [x] A-023 R7/R8/R11/R12: older-binary fallbacks are stated for the survey (grep), the Shape Report file rows (read-pass measurement), and the `_unsorted/` triage (folder listing) — each with an upgrade posture consistent with reorg's existing fallback
- [x] A-024 R3: the no-fabricated-rationale guard is explicit — a DD-bullet rewrite with no derivable Why/Rejected carries only Decision + *Introduced by*
- [x] A-025 R9: an ambiguous `split-file` (no dominant topic, un-anchored inbound links) rolls back that migration via the existing abort escape and continues

### Code Quality

- [x] A-026 Pattern consistency: the Go switch edit follows the surrounding `warnings`-append pattern (same `WarningFinding` fields), and the skill/SPEC prose follows the existing section structure and citation style (`$(fab kit-path)/reference/fkf.md`)
- [x] A-027 No unnecessary duplication: `split-file`/`merge-file` reuse the existing split/merge Link-Impact + no-dangling-link machinery rather than restating it; the survey uses the one `--check --json` call rather than a second signal source
- [x] A-028 Canonical source only: no edits under `.claude/skills/` (gitignored deployed copies) — only `src/kit/skills/*.md` sources
- [x] A-029 No migration: no `src/kit/migrations/` file (no user-data restructuring; the JSON change is additive)

### Documentation Accuracy

- [x] A-030 R15/R16/R17: every changed skill source's SPEC mirror and the aggregate `skills.md` are accurate against the new behavior; the `_cli-fab.md` JSON-shape change is mirrored in `SPEC-_cli-fab.md`

### Cross-References

- [x] A-031 R1–R4: the four new distill classes cite the deployed extract path `$(fab kit-path)/reference/fkf.md` §3.3 (matching existing skill-prose citation style), not the dev-repo `docs/specs/fkf.md`
- [x] A-032 R10: the duplicate-coverage report notes the tie to the open single-sourcing seam audit as a cross-reference (not scope)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- `docs/memory/memory-docs/distill.md` and `templates.md` are updated at **hydrate**, not apply (intake Affected Memory).
- **Hydrate sweep additions (review should-fix, outside the intake's Affected Memory list)**: `docs/memory/pipeline/schemas.md:232` still claims the 501–1000 description-length nag is "deliberately excluded from the array" (false after R13), and `docs/memory/distribution/kit-architecture.md:320` still enumerates only the four mxgu `warnings[]` kinds — both need the R13 update at hydrate.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (Re-verified at re-review cycle 2: the distill survey's legacy agent-side grep heuristics are deliberately retained verbatim as the R7 older-binary fallback, not made redundant; the Go change is a purely additive switch case with no displaced branch; the corrected "deliberately excluded" flag-help/comment notes in `memory_index.go` were rewritten in place, leaving nothing orphaned.)

## Assumptions

<!-- SCORING SOURCE NOTE: as of 1.10.0, `fab score` reads intake.md only — this
     ## Assumptions section is the apply-agent's record of graded decisions made
     while co-generating ## Requirements. Three grades only (Certain/Confident/Tentative). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The Go change is the single-line switch addition (`KindDescriptionLength` → `warnings[]` append) plus the two stale-note corrections and one test; no CLI signature change, no new constant (the kind already exists in `internal/memoryindex`) | Verified against `memory_index.go` lines 185–195 + the `internal/memoryindex` Kind consts and `Warning` struct; intake states this exactly | S:95 R:90 A:95 D:95 |
| 2 | Certain | Two stale notes must also change with the switch: the `--json` flag `Short` ("deliberately excluded from the array") and the code comment above the switch ("only the mxgu debt-meter kinds join the array") — otherwise the help/comments contradict the new behavior | Both literally present in `memory_index.go` (flag help line + switch comment); leaving them stale is a documentation-accuracy defect review would flag | S:90 R:90 A:95 D:90 |
| 3 | Confident | The full mirror sweep class for this CLI-touching change is `SPEC-docs-distill-memory.md` + `SPEC-docs-reorg-memory.md` + `SPEC-_cli-fab.md` + `docs/specs/skills.md`; `docs/specs/fkf.md`/`src/kit/reference/fkf.md` are NOT swept (they are cited as-is, shipped by `[wrct]`/`[mxgu]`) | code-quality.md § Sibling & Mirror Sweeps + the intake Impact list; FKF text is an explicit Non-Goal | S:80 R:85 A:85 D:80 |
| 4 | Confident | The survey rolls a sub-domain file up to its domain by the FIRST path segment under `docs/memory/` (matching the existing survey's per-domain, domain-table-order reporting) | Intake §4 states "first path segment under `docs/memory/`"; consistent with the existing survey scanning domains in `index.md` domain-table order | S:75 R:85 A:85 D:80 |
| 5 | Confident | `merge-file` (R10) is authored as the file-granularity parallel of `merge-domain`, reusing move-section + link-rewrite + no-dangling-link machinery, matching `split-file`'s parallel to `split-domain` | Intake §3 names `merge-file` explicitly as "parallel to merge-domain at file granularity, with Link Impact + the no-dangling-link guard" | S:80 R:80 A:80 D:75 |
| 6 | Confident | The new distill Step 2 report entries and completion counters are authored as SHOULD (report shape), matching the existing report/completion-line style; exact wording follows the intake's example lines | Intake §1 gives example report lines; the existing skill's Step 2/Output already uses this shape | S:75 R:90 A:85 D:80 |
| 7 | Confident | TODO relocation uses a fresh 4-char id in the backlog entry (author picks a plausible unused id at apply-doc-authoring time; the skill instructs generating a fresh id at run time), format `- [ ] [{id}] {YYYY-MM-DD}: {text} (relocated from {path} by /docs-distill-memory)` under the backlog's `## Open` section | Intake §1(d) + Assumption 9 give the exact format; `fab/backlog.md` uses `- [ ] [{id}] {date}: {text}` under `## Open` | S:80 R:90 A:80 D:80 |

7 assumptions (2 certain, 5 confident, 0 tentative).
