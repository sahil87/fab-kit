# Plan: FKF Change 3/4 — Doc-Skill Prose → Author FKF, Stop Writing Per-File Changelogs

**Change**: 260615-8fr5-doc-skill-fkf-prose-stop-changelogs
**Intake**: `intake.md`

## Requirements

<!-- Requirements derived from the intake design. This is a prose-only change to the
     canonical src/kit/skills/ sources + their docs/specs/skills/SPEC-*.md mirrors,
     conforming the doc-skill instructions to the already-shipped FKF contract
     (docs/specs/fkf.md). No Go code, no CLI signature changes. -->

### Doc Skills: fab-continue Hydrate Behavior

#### R1: Hydrate stops listing `Changelog` in the "update existing" section list
`src/kit/skills/fab-continue.md` Hydrate Behavior (Step 4, ≈ line 195) MUST NOT list `Changelog` among the sections updated on an existing memory file. Memory files no longer carry a `## Changelog` section (FKF §3.3). The merge-without-duplication contract (check by change name, update in place) MUST remain intact for the surviving sections (Requirements, Design Decisions).

- **GIVEN** a hydrate run updating an existing memory file
- **WHEN** the agent reads the Hydrate Behavior section list
- **THEN** the list names only `Requirements` and `Design Decisions` (plus keeping `description:` accurate), with no `Changelog` entry
- **AND** the "merge without duplication" / "replaced in place (not duplicated)" phrasing is preserved unchanged

#### R2: Hydrate authors `type: memory` + `description:` FKF frontmatter on new/updated memory files
`src/kit/skills/fab-continue.md` Hydrate Behavior MUST instruct authoring both `type: memory` (constant, FKF §3.1) and a curated `description:` one-liner (FKF §3.2) on each new memory file, not `description:` alone.

- **GIVEN** a hydrate run creating a new memory file
- **WHEN** the agent authors the file's frontmatter
- **THEN** the prose directs writing `type: memory` plus a curated `description:` one-liner

#### R3: Hydrate calls `fab status set-summary` instead of appending a `## Changelog` row
`src/kit/skills/fab-continue.md` Hydrate Behavior MUST replace any per-file changelog write with a single `fab status set-summary {change} "<one-line what-changed>"` call — the C-lite source line (FKF §6.3) that `fab memory-index` joins with git history to generate `log.md`. The summary is authored once during the change at hydrate (the §6.3 "authored at hydrate" path).

- **GIVEN** a hydrate run recording what changed
- **WHEN** the agent records the change history
- **THEN** the prose directs calling `fab status set-summary {change} "<one-line what-changed>"` (not appending a `## Changelog` table row)
- **AND** the prose notes that `fab memory-index` joins this `summary:` with git history when generating per-folder `log.md`

#### R4: Hydrate memory↔memory cross-links use the bundle-relative `/...` form
`src/kit/skills/fab-continue.md` Hydrate Behavior MUST instruct that any memory↔memory cross-link it writes uses the bundle-relative absolute form (`/...`, resolved from `docs/memory/`, FKF §7), while links out of the bundle (source, specs, URLs) stay repo-relative/absolute-URL.

- **GIVEN** a hydrate run writing a cross-link between two memory files
- **WHEN** the agent authors the link
- **THEN** the prose directs the bundle-relative `/...` form for memory↔memory links and leaves out-of-bundle links repo-relative/absolute-URL

### Doc Skills: docs-hydrate-memory (all 3 modes)

#### R5: Ingest mode drops `Changelog` from the created-file section list and stamps `type: memory`
`src/kit/skills/docs-hydrate-memory.md` Ingest Mode Step 3 (≈ line 87) MUST NOT list `Changelog` among the sections of a newly created file, and the authored frontmatter MUST include `type: memory` alongside the existing `description:` authoring (≈ line 90). Memory↔memory links MUST be bundle-relative.

- **GIVEN** ingest mode creating a new memory file
- **WHEN** the agent creates the file
- **THEN** the section list reads "Overview, Requirements, Design Decisions" (no `Changelog`)
- **AND** the frontmatter authoring directs both `type: memory` and `description:`
- **AND** memory↔memory cross-links use the bundle-relative `/...` form

#### R6: Generate mode file template removes the `## Changelog` block and stamps `type: memory`
`src/kit/skills/docs-hydrate-memory.md` Generate Mode file template (≈ lines 145–164) MUST remove the literal `## Changelog` table block and MUST add `type: memory` to the template's frontmatter (alongside the existing `description:`).

- **GIVEN** generate mode synthesizing a memory file from code analysis
- **WHEN** the agent uses the file template
- **THEN** the template frontmatter carries `type: memory` and `description:`
- **AND** the template contains no `## Changelog` block

#### R7: Backfill mode stamps `type: memory` when adding frontmatter, staying body-preserving
`src/kit/skills/docs-hydrate-memory.md` Backfill Mode MUST stamp `type: memory` (alongside `description:`) when adding the leading frontmatter block, so a backfilled file is FKF-conforming (FKF §2 item 2). Backfill MUST remain body-preserving — it MUST NOT strip existing `## Changelog` bodies (that is Change 4's job).

- **GIVEN** backfill mode adding frontmatter to an existing memory file missing it
- **WHEN** the agent writes the leading frontmatter block
- **THEN** the frontmatter carries both `type: memory` and `description:`
- **AND** the file body (including any existing `## Changelog` section) is preserved byte-for-byte

### Documentation: SPEC Mirrors

#### R8: Each changed skill file's SPEC mirror is updated in the same pass
For every `src/kit/skills/*.md` file changed by this change, the corresponding `docs/specs/skills/SPEC-*.md` mirror MUST be updated to reflect the new FKF behavior (Constitution: "Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md` file"). Mirrors of skills NOT changed MUST NOT be edited.

- **GIVEN** this change edits `fab-continue.md` and `docs-hydrate-memory.md`
- **WHEN** the change is complete
- **THEN** `SPEC-fab-continue.md` and `SPEC-docs-hydrate-memory.md` reflect the FKF prose edits (stamp `type: memory`, call `set-summary`, drop `## Changelog`, bundle-relative links)
- **AND** `SPEC-_generation.md` and `SPEC-_review.md` are unchanged (their source skills are no-ops — R9/R10)

### Non-Goals

- Stripping the `## Changelog` sections from the 20 existing memory files (that is Change 4/4) — this change only stops *new* writes.
- Editing `docs/specs/fkf.md` (the authoritative contract — this change conforms prose *to* it).
- Editing `docs/memory/` files — memory hydration happens at the hydrate pipeline stage, not apply.
- Editing `.claude/skills/` deployed copies (gitignored output — Constitution V; canonical sources only).

### Design Decisions

1. **`_generation.md` and `_review.md` confirmed no-ops by close read** — *Why*: a full case-insensitive grep of both files found zero `## Changelog`-writing prose and no memory-file template; `_generation.md` covers only intake/plan generation, `_review.md` only reads memory and writes `## Deletion Candidates`. *Rejected*: editing them speculatively, which would violate the "only mirror changed skills" rule and the intake's strong prior.
2. **Wire the `fab status set-summary` verb despite its absence from the installed `fab 2.4.2` binary** — *Why*: the verb exists in source (`status.go`, `_cli-fab.md`, specs, memory — shipped via `5943`/`bmzo`); this change writes *prose that instructs calling the verb*, it does not invoke it live, so the source-vs-binary skew does not block it. *Rejected*: deferring the wiring until a release lands, which would leave the prose contradicting the shipped FKF format.
3. **Summary authored once at hydrate** — *Why*: FKF §6.3 lists both intake and hydrate as possible authoring points, but task (a) is explicitly hydrate behavior, and hydrate is where the memory write happens. *Rejected*: authoring at intake, which is out of this change's scope.

## Tasks

### Phase 1: fab-continue Hydrate Behavior (skill + mirror)

- [x] T001 Edit `src/kit/skills/fab-continue.md` Hydrate Behavior Step 4 (≈ line 195): drop `Changelog` from the "update existing" section list; instruct authoring `type: memory` + `description:` frontmatter on new files; replace the changelog-row write with a `fab status set-summary {change} "<one-line what-changed>"` call (note `fab memory-index` joins it with git history for `log.md`); require bundle-relative `/...` memory↔memory links. Preserve the merge-without-duplication contract and the refuse-before-regen guard / shape guidance verbatim. <!-- R1 R2 R3 R4 -->
- [x] T002 Update `docs/specs/skills/SPEC-fab-continue.md` HYDRATE STAGE box + prose to mirror T001: `type: memory` + `description:` frontmatter, `fab status set-summary` call (no `## Changelog` write), bundle-relative links. <!-- R8 -->

### Phase 2: docs-hydrate-memory all 3 modes (skill + mirror)

- [x] T003 Edit `src/kit/skills/docs-hydrate-memory.md` Ingest Mode Step 3 (≈ line 87): drop `Changelog` from the created-file section list; stamp `type: memory` alongside `description:` in the frontmatter-authoring prose (≈ line 90); require bundle-relative `/...` memory↔memory links. <!-- R5 -->
- [x] T004 Edit `src/kit/skills/docs-hydrate-memory.md` Generate Mode file template (≈ lines 145–164): add `type: memory` to the template frontmatter; remove the `## Changelog` table block. <!-- R6 -->
- [x] T005 Edit `src/kit/skills/docs-hydrate-memory.md` Backfill Mode (Step 2, ≈ line 192): stamp `type: memory` alongside `description:` when writing the leading frontmatter block; keep it body-preserving (do NOT strip existing `## Changelog` bodies). <!-- R7 -->
- [x] T006 Update `docs/specs/skills/SPEC-docs-hydrate-memory.md` Summary + Backfill Mode prose to mirror T003–T005: all three modes stamp `type: memory`; ingest/generate drop `## Changelog`; bundle-relative links; backfill stays body-preserving and does not strip existing changelogs. <!-- R8 -->

### Phase 3: Investigate-and-record no-ops

- [x] T007 Close-read `src/kit/skills/_generation.md` for any `## Changelog`-writing or memory-file-template prose. Confirmed NO-OP (grep + read: only intake/plan generation; no memory template, no changelog). Make no edit to `_generation.md` or `SPEC-_generation.md`; record the finding in this plan. <!-- R9 -->
- [x] T008 Close-read `src/kit/skills/_review.md` for any `## Changelog`-writing prose. Confirmed NO-OP (it reads memory for consistency checks and writes `## Deletion Candidates`; no changelog writing). Make no edit to `_review.md` or `SPEC-_review.md`; record the finding in this plan. <!-- R10 -->

### Phase 4: Verification

- [x] T009 Grep the four edited/investigated skills to confirm: no NEW `## Changelog`-writing instructions remain in `fab-continue.md` / `docs-hydrate-memory.md`; `type: memory` is present where required; `set-summary` is wired in `fab-continue.md`; bundle-relative `/...` phrasing is present; SPEC mirrors are consistent with their sources. <!-- R1 R2 R3 R4 R5 R6 R7 R8 -->

## Execution Order

- T001 → T002 (mirror follows source); T003/T004/T005 → T006 (mirror follows source). Phase 1 and Phase 2 are independent of each other. Phase 3 (no-op confirmation) is independent. T009 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-continue.md` Hydrate Behavior no longer lists `Changelog` in the "update existing" section list; merge-without-duplication phrasing preserved.
- [x] A-002 R2: `fab-continue.md` Hydrate Behavior directs authoring `type: memory` + curated `description:` on new memory files.
- [x] A-003 R3: `fab-continue.md` Hydrate Behavior directs calling `fab status set-summary {change} "<one-line what-changed>"` instead of appending a `## Changelog` row, and notes the `log.md` join.
- [x] A-004 R4: `fab-continue.md` Hydrate Behavior directs bundle-relative `/...` memory↔memory links (out-of-bundle links unchanged).
- [x] A-005 R5: `docs-hydrate-memory.md` Ingest Mode Step 3 drops `Changelog` from the section list, stamps `type: memory` in frontmatter authoring, and requires bundle-relative links.
- [x] A-006 R6: `docs-hydrate-memory.md` Generate Mode file template carries `type: memory` and no `## Changelog` block.
- [x] A-007 R7: `docs-hydrate-memory.md` Backfill Mode stamps `type: memory` and remains body-preserving (does not strip existing `## Changelog` bodies).
- [x] A-008 R8: `SPEC-fab-continue.md` and `SPEC-docs-hydrate-memory.md` mirror their sources; `SPEC-_generation.md` and `SPEC-_review.md` are unchanged.

### Behavioral Correctness

- [x] A-009 R3: The replaced prose names the verb form `fab status set-summary {change} "..."` exactly (consistent with `_cli-fab.md`), not an invented signature.
- [x] A-010 R7: Backfill mode's body-preservation guarantee (byte-for-byte) is still stated and is not contradicted by the new `type:` stamping.

### Removal Verification

- [x] A-011 R6: No `## Changelog` table block remains in any created-file template or section list across `fab-continue.md` and `docs-hydrate-memory.md` (grep confirms only narrative/Change-4 references, no write instructions).

### Edge Cases & Error Handling

- [x] A-012 R7: The Change-3-vs-Change-4 boundary is honored — no edit strips an existing `## Changelog` body from any memory file; only *new-write* instructions are removed.

### Code Quality

- [x] A-013 Pattern consistency: Edits match each skill's existing prose style (heading style, the existing `description:`-frontmatter phrasing, merge-without-duplication contract wording, FKF §-citations).
- [x] A-014 No unnecessary duplication: No new prose duplicates existing FKF-spec content; edits reuse the canonical phrasing and cite `docs/specs/fkf.md` sections.

### Documentation Accuracy

- [x] A-015 R8: Every changed skill file has its SPEC mirror updated in the same pass; no mirror of an unchanged skill is touched (Constitution rule).

### Cross-References

- [x] A-016 R3: Cross-references to FKF spec sections (§3.1, §3.2, §3.3, §6.3, §7) and to `fab status set-summary` / `fab memory-index` are accurate against the canonical contract.

## Notes

- Check items as you review: `- [x]`
- This change ships PROSE only — no Go code, no `*_test.go` to run. Verification is markdown-prose correctness + SPEC-mirror consistency (T009 grep).
- The `fab status set-summary` verb exists in SOURCE but not in the installed `fab 2.4.2` binary; the prose instructs calling it and is correct against the canonical contract.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Target memory format is dictated by `docs/specs/fkf.md` (drop `## Changelog`, add `type: memory`, `description:` required, bundle-relative `/...` links, `summary:` as `log.md` source) | The spec is the authoritative already-shipped contract; this change conforms prose to it, no design latitude | S:95 R:80 A:100 D:95 |
| 2 | Certain | Edits target canonical `src/kit/skills/` sources; `.claude/skills/` is gitignored deployed output, never hand-edited | Constitution V is explicit | S:100 R:90 A:100 D:100 |
| 3 | Certain | Every changed skill file pairs with a synchronous `docs/specs/skills/SPEC-*.md` mirror update | Constitution "Changes to skill files MUST update the corresponding SPEC-*.md" | S:100 R:75 A:100 D:95 |
| 4 | Certain | Task (c) `_generation.md` is a no-op — close read + grep found no memory template and no `## Changelog`; the generated-file template lives in `docs-hydrate-memory.md` (R6) | Confirmed at apply: only intake/plan generation; `templates/` has no memory template | S:90 R:75 A:95 D:90 |
| 5 | Certain | Task (d) `_review.md` is a no-op — it reads memory and writes `## Deletion Candidates`, writes no `## Changelog` | Confirmed at apply: grep found no changelog-writing prose | S:90 R:75 A:95 D:90 |
| 6 | Confident | `fab status set-summary` is the correct verb to wire into hydrate prose despite its absence from the installed `fab 2.4.2` binary | Verb exists in source (`status.go`, `_cli-fab.md`, specs, memory — shipped `5943`/`bmzo`); prose is correct against canonical contract; this change writes prose, not a live call | S:90 R:85 A:95 D:90 |
| 7 | Confident | `summary:` is authored once at hydrate by `fab-continue` (the §6.3 "authored at hydrate" path), not at intake | §6.3 lists both, but task (a) is explicitly hydrate behavior and the memory write happens at hydrate | S:75 R:80 A:85 D:80 |
| 8 | Confident | SPEC-mirror edits for `fab-continue` and `docs-hydrate-memory` are additive (the mirrors currently mention neither `## Changelog` nor `type: memory`) — add the FKF behavior to the relevant sections, matching existing prose | Grep confirmed mirrors carry no changelog/type-memory/set-summary/bundle-relative text today | S:85 R:80 A:90 D:85 |

8 assumptions (5 certain, 3 confident, 0 tentative).
