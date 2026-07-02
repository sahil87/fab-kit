# Plan: Ship Memory-File Template in Kit Cache

**Change**: 260616-2fm8-ship-memory-file-template
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md What Changes A–E. This is a refactor: collapse the
     duplicated memory-file shape to one shipped template and repoint the doc
     skills to read it on demand. No behavior change, no Go, no migration. -->

### Templates: Canonical Memory-File Template

#### R1: Ship `src/kit/templates/memory.md`
The kit MUST ship a canonical memory-file template at `src/kit/templates/memory.md` so that FKF §3.1's "the memory-file template" is realized as a real artifact and the conventional memory-file shape has a single source of truth. The template MUST carry leading FKF frontmatter — the `type: memory` constant (FKF §3.1) plus a `description:` placeholder (FKF §3.2) — and the conventional body skeleton per `$(fab kit-path)/reference/fkf.md` §3.3 (Overview / Requirements + Scenario / Design Decisions). The template MUST NOT include a `## Changelog` section (FKF §3.3 — change history lives in the per-folder generated `log.md`). The template MUST demonstrate a bundle-relative cross-link (FKF §7) and MUST carry a guidance comment that cites `$(fab kit-path)/reference/fkf.md` §3.3 rather than re-describing the heading rules.

- **GIVEN** the kit cache at `$(fab kit-path)/templates/`
- **WHEN** an agent or `fab kit-path` resolution reads `templates/memory.md`
- **THEN** it finds a valid-markdown template whose frontmatter parses with `type: memory` and a `description:` placeholder, a body skeleton of Overview / Requirements (+ Scenario) / Design Decisions, no `## Changelog`, a bundle-relative `](/...)` link example, and a guidance comment citing `fkf.md` §3.3

#### R2: Preserve the SHOULD-not-MUST heading nuance
The template's guidance comment MUST preserve FKF §3.3's posture that conventional headings are recommended where they apply, not mandatory — a small reference-pointer file legitimately omits a GIVEN/WHEN/THEN scenario. The guidance comment MUST NOT imply the body sections are required.

- **GIVEN** the guidance comment in `templates/memory.md`
- **WHEN** an agent reads it to author a memory file
- **THEN** the comment communicates that the skeleton is a scaffold (SHOULD-use-where-applicable), not a mandatory section set, consistent with FKF §3.3

### Skills: Repoint Doc Skills to Read the Template

#### R3: Repoint `docs-hydrate-memory.md` (all three modes) to the template
`src/kit/skills/docs-hydrate-memory.md` MUST replace its inlined memory-file shape with an on-demand read of `$(fab kit-path)/templates/memory.md`, mirroring the existing `$(fab kit-path)/templates/intake.md` read pattern. The generate mode's literal ```markdown shape block (Step 3, ~lines 145–160) MUST be replaced with an instruction to read the shape from the template. The ingest mode (Step 3) MUST author from the template rather than the inlined description. The backfill mode MUST reference the template for the **frontmatter shape only** and MUST preserve its pure-frontmatter, body-preserving contract — the repoint MUST NOT widen backfill into a body rewrite.

- **GIVEN** `docs-hydrate-memory.md` after the repoint
- **WHEN** an agent runs generate or ingest mode
- **THEN** it reads the memory-file shape from `$(fab kit-path)/templates/memory.md` instead of an inlined block
- **AND** **WHEN** an agent runs backfill mode
- **THEN** it references the template for the frontmatter shape only and still preserves the body byte-for-byte (no `## Changelog` strip, no body skeleton imposed)

#### R4: Repoint `fab-continue.md` hydrate new-file shape to the template
`src/kit/skills/fab-continue.md` Hydrate Behavior (the "create new files/domains (each carrying FKF frontmatter…)" prose, line 195) MUST reference `$(fab kit-path)/templates/memory.md` for the new-file shape. All surrounding hydrate contracts MUST be preserved verbatim: the `fab status set-summary` C-lite `log.md` source, bundle-relative links, merge-without-duplication, the refuse-before-regen guard, and the shape SHOULD guidance.

- **GIVEN** `fab-continue.md` Hydrate Behavior after the repoint
- **WHEN** an agent creates a new memory file during hydrate
- **THEN** the prose points it at `$(fab kit-path)/templates/memory.md` for the shape
- **AND** the set-summary / bundle-relative-link / merge-without-duplication / refuse-before-regen / shape-guidance contracts are unchanged

#### R5: `_generation.md` repoint resolves to a no-op (recorded decision)
`src/kit/skills/_generation.md` MUST be left unchanged. It contains only the Intake Generation and Plan Generation procedures (reading `templates/intake.md` and `templates/plan.md`); it has no memory-authoring seam to repoint, and related change `8fr5` left no inlined memory shape in it. The decision to leave it untouched MUST be recorded in `## Assumptions`.

- **GIVEN** `_generation.md`'s actual content (Intake + Plan procedures only)
- **WHEN** the repoint of B is evaluated against it
- **THEN** there is no memory-authoring section to repoint, so the file is left unchanged and the no-op is recorded in `## Assumptions`

#### R6: `docs-reorg-memory.md` repoint is conditional — left as-is (recorded decision)
`src/kit/skills/docs-reorg-memory.md` SHOULD be repointed to cite the template only if doing so reduces duplication without widening reorg's responsibilities. Reorg moves files (preserving FKF frontmatter byte-for-byte) and authors `type: memory` + `description:` only on genuinely-new split files (Step 5 item 3); it already cites `fkf.md` for §5/§7 and authors no memory bodies — there is no inlined body shape to collapse. Therefore reorg MUST be left as-is, and the decision MUST be recorded in `## Assumptions`. Because reorg is unchanged, `SPEC-docs-reorg-memory.md` MUST NOT be touched.

- **GIVEN** `docs-reorg-memory.md`'s frontmatter-only authoring (no body skeleton)
- **WHEN** citing the template is evaluated for duplication reduction
- **THEN** it adds a cross-reference without removing any duplicated shape and risks implying reorg should scaffold bodies, so reorg is left as-is and the decision is recorded

### Skills: Close the `_cli-fab.md` Admission

#### R7: Remove the no-template admission in `_cli-fab.md`
`src/kit/skills/_cli-fab.md` (~line 514) MUST be updated to remove the "there is **no** memory-file template carrying `type: memory` yet (`src/kit/templates/` holds only the intake/plan/status templates)" admission, since the template now ships. The surrounding contract — that `fab memory-index` preserves `type:` when present and does not author/bulk-stamp it — MUST be preserved; only the now-false "no template yet" claim is removed/corrected. `_cli-fab.md` is exempt from SPEC mirrors (there is no `SPEC-_cli-fab.md`).

- **GIVEN** `_cli-fab.md` after the edit
- **WHEN** a reader reaches the `type: memory` frontmatter bullet (~line 514)
- **THEN** it no longer claims no memory-file template exists, and the preserve-when-present / no-bulk-stamp contract for `fab memory-index` is intact

### Specs: SPEC Mirrors (Constitution Rule)

#### R8: Mirror the changed skills into `docs/specs/skills/SPEC-*.md`
Per the Constitution skill→SPEC-mirror rule, every changed skill file MUST have its `docs/specs/skills/SPEC-*.md` mirror updated to reflect the change actually made. `SPEC-docs-hydrate-memory.md` and `SPEC-fab-continue.md` MUST be updated to record that the memory-file shape is now read on demand from `$(fab kit-path)/templates/memory.md`. `SPEC-_generation.md` MUST NOT be changed (its skill is unchanged — R5). `SPEC-docs-reorg-memory.md` MUST NOT be changed (its skill is unchanged — R6).

- **GIVEN** the skill edits made for R3 and R4
- **WHEN** the SPEC mirrors are reviewed
- **THEN** `SPEC-docs-hydrate-memory.md` and `SPEC-fab-continue.md` reflect the template-read repoint, and `SPEC-_generation.md` / `SPEC-docs-reorg-memory.md` are untouched because their skills are untouched

### Non-Goals

- No Go change, no migration, no packaging change — `src/kit/` ships verbatim via `just install` (rsync) and `just dist-kit` (`cp -a`); existing projects pick up the template on their next `fab sync`. (Intake assumption 2.)
- No `fab memory-index` stamping change — task (c) is resolved as "keep the preserve-when-present round-trip; the template + doc skills are the stampers." (Intake assumption 5, What Changes C.)
- No edits to `docs/memory/` — `memory-docs/templates.md` and `distribution/kit-architecture.md` updates are hydrate-stage work, not apply.
- No edits under `.claude/skills/` — that tree is the gitignored `fab sync` deploy copy; all edits go to `src/kit/skills/`.

### Design Decisions

1. **On-demand template read, not inlining**: doc skills read `$(fab kit-path)/templates/memory.md` at point of use — *Why*: this is the exact pattern `_generation.md`/`_intake.md` use for `templates/intake.md`; it collapses the duplicated shape to one source of truth (FKF's whole point) — *Rejected*: keeping the inlined shape in each skill (the current multi-source-of-truth drift FKF was designed to prevent).
2. **Template cites `fkf.md` §3.3 rather than re-describing the rules**: the body's guidance comment points to the reference — *Why*: the template is a scaffold, the reference is the contract; mirrors how `intake.md`/`plan.md` use guidance comments — *Rejected*: re-stating the heading rules in the template (a second place to drift from §3.3).
3. **Backfill references frontmatter shape only**: the repoint touches backfill's frontmatter reference, not its body contract — *Why*: backfill is a pure-frontmatter, body-preserving operation (`docs-hydrate-memory.md:178/190/191`); widening it into a body rewrite would break its contract — *Rejected*: pointing backfill at the full body skeleton.

## Tasks

<!-- Each item carries a <!-- R# --> trace annotation. Phase order is enforced
     by Execution Order below: the template (Phase 1) must exist before the
     skills are repointed to read it (Phase 2), and SPEC mirrors (Phase 3)
     mirror the Phase-2 skill edits. -->

### Phase 1: Setup — Canonical Template

- [x] T001 Create `src/kit/templates/memory.md` — leading FKF frontmatter (`type: memory` constant + `description:` placeholder), body skeleton (Overview / Requirements + Scenario / Design Decisions per `fkf.md` §3.3), NO `## Changelog`, a bundle-relative `](/...)` cross-link example (§7), and a guidance comment citing `$(fab kit-path)/reference/fkf.md` §3.3 that preserves the SHOULD-not-MUST heading nuance <!-- R1 -->
- [x] T002 Verify `templates/memory.md` is valid markdown and its FKF frontmatter parses (`type: memory` + `description:` present); confirm the guidance comment does not imply the body sections are mandatory <!-- R2 -->

### Phase 2: Core Implementation — Repoint Doc Skills

- [x] T003 [P] Repoint `src/kit/skills/docs-hydrate-memory.md` all three modes to read `$(fab kit-path)/templates/memory.md`: replace the generate-mode literal ```markdown shape block (Step 3) with a template-read instruction; point ingest-mode Step 3 authoring at the template; reference the template for backfill's **frontmatter shape only** while preserving its pure-frontmatter, body-preserving contract (no `## Changelog` strip, no body skeleton imposed) <!-- R3 -->
- [x] T004 [P] Repoint `src/kit/skills/fab-continue.md` Hydrate Behavior (line 195) new-file prose to reference `$(fab kit-path)/templates/memory.md` for the shape, preserving the set-summary / bundle-relative-link / merge-without-duplication / refuse-before-regen / shape-SHOULD-guidance contracts verbatim <!-- R4 -->
- [x] T005 [P] Update `src/kit/skills/_cli-fab.md` (~line 514) to remove the now-false "there is no memory-file template carrying `type: memory` yet" admission while preserving the `fab memory-index` preserve-when-present / no-bulk-stamp contract <!-- R7 -->
- [x] T006 Confirm `src/kit/skills/_generation.md` has no memory-authoring seam (Intake + Plan procedures only) and leave it unchanged; record the no-op in `## Assumptions` <!-- R5 -->
- [x] T007 Evaluate `src/kit/skills/docs-reorg-memory.md` against the duplication-reduction test; leave it as-is (frontmatter-only authoring, already cites `fkf.md`, no inlined body shape to collapse); record the decision in `## Assumptions` <!-- R6 -->

### Phase 3: Integration — SPEC Mirrors

- [x] T008 Update `docs/specs/skills/SPEC-docs-hydrate-memory.md` to mirror the T003 repoint — record that all three modes now read the memory-file shape on demand from `$(fab kit-path)/templates/memory.md` (frontmatter-shape-only for backfill) <!-- R8 -->
- [x] T009 Update `docs/specs/skills/SPEC-fab-continue.md` to mirror the T004 repoint — record that hydrate's new-file shape is read from `$(fab kit-path)/templates/memory.md` <!-- R8 -->
- [x] T010 Confirm `docs/specs/skills/SPEC-_generation.md` and `SPEC-docs-reorg-memory.md` are NOT touched (their skills are unchanged per R5/R6) <!-- R8 -->

### Phase 4: Polish — Verification

- [x] T011 Smoke check: `cd src/go/fab && go build ./... && go test ./...` (markdown-only change — confirm nothing Go broke); verify no file under `.claude/skills/` was edited <!-- R1 -->

## Execution Order

- T001 → T002 (template must exist before it is verified)
- T001 blocks T003, T004, T005 (skills are repointed to read the template; the template must exist first)
- T003 → T008, T004 → T009 (SPEC mirrors mirror the skill edits)
- T006, T007 are independent confirmations (no file edits), can run anytime in Phase 2
- T011 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/templates/memory.md` exists with leading FKF frontmatter (`type: memory` constant + `description:` placeholder), the Overview / Requirements (+ Scenario) / Design Decisions skeleton, no `## Changelog`, a bundle-relative `](/...)` link example, and a guidance comment citing `$(fab kit-path)/reference/fkf.md` §3.3
- [x] A-002 R3: `docs-hydrate-memory.md` reads the memory-file shape from `$(fab kit-path)/templates/memory.md` in all three modes (generate block replaced, ingest authored from template, backfill references frontmatter shape only)
- [x] A-003 R4: `fab-continue.md` Hydrate Behavior references `$(fab kit-path)/templates/memory.md` for the new-file shape
- [x] A-004 R7: `_cli-fab.md` no longer claims no memory-file template exists
- [x] A-005 R8: `SPEC-docs-hydrate-memory.md` and `SPEC-fab-continue.md` mirror the template-read repoint

### Behavioral Correctness

- [x] A-006 R2: The template's guidance comment preserves FKF §3.3's SHOULD-not-MUST posture — it does not imply the body sections are mandatory
- [x] A-007 R3: The `docs-hydrate-memory.md` backfill repoint preserves the pure-frontmatter, body-preserving contract — no `## Changelog` strip, no body skeleton imposed on existing files
- [x] A-008 R4: All surrounding `fab-continue.md` hydrate contracts (set-summary, bundle-relative links, merge-without-duplication, refuse-before-regen, shape SHOULD guidance) are unchanged
- [x] A-009 R7: The `_cli-fab.md` `fab memory-index` preserve-when-present / no-bulk-stamp contract is intact after the admission is removed

### Removal Verification

- [x] A-010 R7: The exact phrase "there is **no** memory-file template carrying `type: memory` yet" (and its parenthetical "holds only the intake/plan/status templates") is gone from `_cli-fab.md`

### Scenario Coverage

- [x] A-011 R1: `templates/memory.md` is valid markdown and its YAML frontmatter parses cleanly (`type: memory` + `description:`)
- [x] A-012 R5: `_generation.md` is unchanged (verified no memory-authoring seam) and the no-op is recorded in `## Assumptions`
- [x] A-013 R6: `docs-reorg-memory.md` is unchanged and the conditional decision is recorded in `## Assumptions`
- [x] A-014 R8: `SPEC-_generation.md` and `SPEC-docs-reorg-memory.md` are untouched (their skills are untouched)

### Edge Cases & Error Handling

- [x] A-015 R1: No file under `.claude/skills/` was edited (canonical source is `src/kit/`; `.claude/skills/` is the gitignored deploy copy)

### Code Quality

- [x] A-016 Pattern consistency: The template's guidance-comment style and the skills' template-read phrasing match the existing `$(fab kit-path)/templates/intake.md` read pattern in `_generation.md`/`docs-hydrate-memory.md`; SPEC-mirror voice/structure matches the existing mirrors
- [x] A-017 No unnecessary duplication: The memory-file shape now lives in exactly one source (`templates/memory.md`); the doc skills reference it rather than re-inlining it (verified: the inlined ```markdown shape block in docs-hydrate-memory.md generate mode is removed — `grep -c '^```markdown'` returns 0)

### Documentation Accuracy

<!-- checklist.extra_categories: documentation_accuracy -->

- [x] A-018 Every `fkf.md` section cite in the new/changed files (§3.1/§3.2/§3.3/§6/§7) resolves correctly against `$(fab kit-path)/reference/fkf.md` (verified: all cited sections exist as headings). The `go build/test` half is **N/A**: no Go file is in the diff (`git diff --name-only HEAD | grep '\.go$'` is empty) — this is a markdown-only refactor, so there is no Go to regress

### Cross-References

<!-- checklist.extra_categories: cross_references -->

- [x] A-019 All `$(fab kit-path)/templates/memory.md` and `$(fab kit-path)/reference/fkf.md` path cites in the changed skills and SPEC mirrors are consistent and resolve to real kit-cache paths. `reference/fkf.md` is present in the cache today; `templates/memory.md` is the new canonical source at `src/kit/templates/memory.md` and reaches the version-pinned cache on the next `fab sync` (the deploy-on-sync model — see Non-Goals). The cite string is consistent across all changed files and structurally correct

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/kit/skills/docs-hydrate-memory.md` (generate mode, former Step 3 ```markdown block, ~old lines 145–160) — the inlined memory-file shape block; the canonical `src/kit/templates/memory.md` now owns this shape, so the inlined copy was redundant. Already removed by this change's apply (the dedup is the change's whole point) — recorded here as the candidate this refactor retired, not a new outstanding deletion.
- No further candidates. The `_cli-fab.md` "no template yet" admission was a *false-statement correction*, not redundant code; `fab-continue.md`/ingest/backfill prose was *repointed*, not duplicated. `_generation.md` and `docs-reorg-memory.md` were correctly left untouched (no inlined body shape to retire — recorded decisions, Assumptions 6/7).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship `src/kit/templates/memory.md`; doc skills read it via `$(fab kit-path)/templates/memory.md` (the existing `templates/intake.md` read pattern) | Carried verbatim from intake assumption 1; `fab kit-path` confirmed at `~/.fab-kit/versions/2.5.4/kit`; mirrors `_generation.md`/`_intake.md` | S:100 R:80 A:95 D:95 |
| 2 | Certain | Template body excludes `## Changelog`; carries `type: memory` + `description:` + Overview/Requirements(+Scenario)/Design Decisions + a bundle-relative link example, citing `fkf.md` §3.3 in a guidance comment | Directly dictated by `fkf.md` §3.1/§3.3/§7 (read in-worktree; reference is reachable per shipped frlo) | S:95 R:85 A:100 D:95 |
| 3 | Certain | No Go / packaging / migration change; SPEC mirrors only for the skills actually changed (`docs-hydrate-memory`, `fab-continue`); `_cli-fab.md` SPEC-exempt | Constitution skill→SPEC-mirror rule + intake assumptions 2/4; refactor change_type; `src/kit/` ships verbatim | S:95 R:85 A:95 D:90 |
| 4 | Confident | Backfill mode references the template for the **frontmatter shape only**, not the body skeleton — preserving its pure-frontmatter, body-preserving contract | `docs-hydrate-memory.md:178/190/191` define backfill as body-preserving; the repoint must not widen it (intake assumption 9) | S:75 R:75 A:85 D:70 |
| 5 | Confident | Task (c) resolved: keep `fab memory-index`'s preserve-when-present round-trip; the template + doc skills are the stampers — no Go change | Codebase check: generator round-trips `type:` only, never bulk-stamps topic files; FKF §3.1 assigns stamping to writers (intake assumption 5, verified against `_cli-fab.md:512-518`) | S:75 R:80 A:90 D:85 |
| 6 | Tentative | `_generation.md` repoint is a NO-OP — left unchanged; consequently `SPEC-_generation.md` is also untouched | Confirmed against the actual file: `_generation.md` holds only Intake + Plan procedures (read intake.md/plan.md templates), has no memory-authoring seam, and `8fr5` left no inlined memory shape; real consumers are docs-hydrate-memory + fab-continue (intake assumption 7) | S:70 R:80 A:85 D:65 |
| 7 | Tentative | `docs-reorg-memory.md` left as-is (not repointed); consequently `SPEC-docs-reorg-memory.md` is also untouched | Reorg moves files (preserving frontmatter byte-for-byte) and authors only frontmatter on new split files — it already cites `fkf.md` §5/§7 and has no inlined body shape to collapse; citing the template adds a cross-ref without dedup and risks implying reorg should scaffold bodies (intake assumption 8) | S:65 R:75 A:75 D:60 |

7 assumptions (3 certain, 2 confident, 2 tentative).
