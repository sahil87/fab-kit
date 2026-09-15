# Plan: code-reorg skill

**Change**: 260823-ekp3-code-reorg-skill
**Intake**: `intake.md`

## Requirements

### Skill: /code-reorg — source-tree structure review

#### R1: Report-only skill file with honest frontmatter
A new user-invocable skill SHALL exist at `src/kit/skills/code-reorg.md` with frontmatter `name: code-reorg`, `helpers: [_srad]`, and a `description` one-liner that names the non-obvious behaviors: read-only findings report on source-tree structure (placement, file names, consolidation), suggestions only — applies nothing, drafts nothing — with `docs/memory/`/`docs/specs/` excluded. The body MUST open with the standard `_preamble` read line. The skill MUST be fully read-only: it modifies no files, creates no changes, runs no `fab status` transition, and creates no git state.

- **GIVEN** a repo with fab initialized
- **WHEN** `/code-reorg` runs to completion
- **THEN** `git status` is unchanged and no `fab/changes/` folder was created

#### R2: Scope resolution with path-based docs carve-out
The skill SHALL take one optional `[<path>]` argument, defaulting to all `source_paths` from `fab/project/config.yaml` swept as one combined scope. It SHALL echo the resolved scope and file count before analysis. The carve-out is path-based, not content-based: a path inside `docs/memory/` or `docs/specs/` is refused with a pointer to `/docs-reorg-memory` / `/docs-reorg-specs`; everything else inside the scoped path is in scope regardless of file type. Errors: a nonexistent or outside-repo path STOPs showing the resolved attempt; missing/empty `source_paths` with no argument STOPs naming the config key.

- **GIVEN** no argument and `source_paths: [src/, scripts/]`
- **WHEN** the skill starts
- **THEN** it echoes the combined scope and its file count before gathering signals
- **GIVEN** the argument `docs/memory/pipeline`
- **WHEN** the skill starts
- **THEN** it refuses with the `/docs-reorg-memory` pointer and performs no analysis

#### R3: Signal gathering with mandatory co-change noise controls
Step 2 SHALL gather: tree-shape signals (depth, fan-out, singleton folders, oversized flat dirs, junk drawers), sibling-naming inconsistencies (casing, plural/singular, stutter), static import-direction (a file whose imports/importers concentrate in a different folder), and git co-change coupling. The co-change walk MUST apply all noise controls: history window (default 12 months), bulk-commit exclusion (default: commits touching > 20 files), rename following (`-M`), a mandated-coupling carve-out derived at run time from the constitution and code-quality files, and a cross-layer whitelist (tests/docs/specs legitimately co-change with the source they cover). The skill file SHALL ship a worked co-change command. Reference density is computed lazily at clustering (step 4), not during detection. Shallow or absent git history SHALL skip the co-change signal with an explicit note in the report while the other signals still run.

- **GIVEN** a repo with sweeping 50-file commits
- **WHEN** co-change coupling is computed
- **THEN** those commits are excluded by the bulk-commit cap and produce no coupling pairs
- **GIVEN** a shallow clone
- **WHEN** the skill runs
- **THEN** the report carries a co-change-skipped note and tree/naming findings are still produced

#### R4: Frame evaluation with the taste guard
Step 3 SHALL evaluate findings against frames in priority order: (a) the project's stated conventions (`context.md`), (b) ecosystem convention, (c) internal sibling consistency. Every finding MUST cite a step-2 signal or a named frame violation; uncited findings are dropped. A finding citing only frame (c) additionally requires a quantified sibling majority (e.g., "7 of 9 siblings use pattern X").

- **GIVEN** a candidate finding with no cited signal and no frame violation
- **WHEN** the taste guard runs
- **THEN** the finding is dropped and does not appear in the report

#### R5: Proposal clustering with blast radius
Step 4 SHALL cluster related findings into coherent proposals. Each proposal carries: the prediction failure it fixes, the concrete move/rename list, blast radius (breaking references — imports, doc cross-links — plus in-flight exposure: open branches / active fab changes touching the affected files), and an SRAD-graded confidence. Folder merges/renames in package-scoped languages (e.g., Go package directories) MUST carry a mandatory elevated blast grade.

- **GIVEN** four related rename findings among siblings
- **WHEN** clustering runs
- **THEN** they form one proposal with one move/rename list, not four proposals

#### R6: Ranked report as the terminal output
Step 5 SHALL present the report ranked highest confidence × lowest blast radius and stop. Content-duplication smells appear in a separate "for `/fab-dedupe`" section, never as proposals. Each proposal MAY carry an informational suggested-next-action line (e.g., `micro: rename directly and commit` or a ready-to-paste `/fab-new <description>`), which the skill never executes. A clean tree ends with a plain "no proposals — structure predicts well" close. The skill emits no pipeline `Next:` line (a documented opt-out per `_preamble.md` § Next Steps Convention, like `/fab-discuss`).

- **GIVEN** a scope where no finding survives the taste guard
- **WHEN** the report renders
- **THEN** it states "no proposals — structure predicts well" and ends without a `Next:` line

#### R7: Help, specs, and README integration
The skill SHALL be integrated per the New Skill Checklist: `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go` gains `"code-reorg": "Maintenance"` with a corresponding `fabhelp_test.go` update; `docs/specs/skills.md` gains a `## /code-reorg [<path>]` section with Flow skeleton plus a `helpers:` row in § Skill Helpers; `docs/specs/glossary.md` gains the skill's row; `README.md`'s hand-maintained command tables gain the `/code-reorg` row, checked against the shll Toolkit Standards.

- **GIVEN** the change is applied
- **WHEN** `go test ./cmd/fab/...` runs in `src/go/fab`
- **THEN** the fabhelp tests pass with the new map entry

### Non-Goals

- Applying moves/renames, drafting intakes, or deciding fix routing — the report is the entire output
- Judging code content or functionality; semantic duplication routes to `/fab-dedupe` as a pointer
- A `reorg.detectors`-style config knob (v1 uses built-in shell signal gathering)
- Memory-file content (hydrate owns `pipeline/code-reorg.md` and the `pipeline/dedupe.md` boundary note)

### Design Decisions

#### Report-only: the skill finds and presents, never routes
**Decision**: `/code-reorg` ends at its ranked report; it applies nothing, drafts no intakes, and never decides whether a fix is a micro change or a pipeline change.
**Why**: Routing is the user's per-finding call — a one-rename proposal is a micro change that must skip fab, a cross-cutting move deserves a pipeline change, and the skill cannot know which without owning policy it shouldn't. Decoupling also drops the `_intake`/`_generation` dependency entirely.
**Rejected**: The fab-dedupe drafted-intake handoff — it forces routing decisions, conflicts with the micro-change doctrine, and the precedent funnel has produced zero drafted intakes since shipping.
*Introduced by*: 260823-ekp3-code-reorg-skill

#### Path-based docs carve-out, not content-based
**Decision**: The only scope exclusions are `docs/memory/` and `docs/specs/`; everything else in the scoped path is in scope regardless of file type.
**Why**: The skill targets repos consuming fab-kit, where those two trees are the only fab-convention prose with dedicated FKF-aware skills; markdown inside a source tree is just source, judged by the same placement/naming frames. A content-type heuristic would be invented, unpredictable, and unneeded.
**Rejected**: Content-based exclusion of doc-like trees — ambiguous (an apply agent must guess "how much markdown makes a docs tree") and wrong for repos whose source legitimately includes prose.
*Introduced by*: 260823-ekp3-code-reorg-skill

#### Co-change is evidence only after noise controls
**Decision**: The co-change signal ships with a mandatory control set (window, bulk-commit cap, `-M`, constitution-derived mandated-coupling carve-out, cross-layer whitelist) and a worked command in the skill file.
**Why**: Raw co-change is systematically poisoned on sweep-heavy repos with mandated doc-mirror coupling — measured here: a doc mirror co-changes with its CLI source in ~81% of commits while co-location is constitutionally prohibited. Without controls, the strongest-looking signal is the least trustworthy.
**Rejected**: "Weighted highest" raw co-change — admits false positives that pass the taste guard by construction; per-run improvised aggregation — the pairwise computation is the one signal too fiddly to improvise.
*Introduced by*: 260823-ekp3-code-reorg-skill

## Tasks

### Phase 2: Core Implementation

- [x] T001 Author `src/kit/skills/code-reorg.md` scaffolding: frontmatter (`name`, `description` one-liner naming report-only/read-only/docs-trees-excluded, `helpers: [_srad]`), `_preamble` read line, Purpose, Arguments, Contents <!-- R1 -->
- [x] T002 Write Behavior step 1 (scope resolution: source_paths default, combined scope, scope+file-count echo, path-based docs carve-out with docs-reorg pointers) and the Error Handling table (nonexistent/outside-repo path, missing source_paths, docs-path refusal, shallow-history co-change skip) <!-- R2 -->
- [x] T003 Write Behavior step 2 (signal gathering: tree shape, sibling naming, import-direction, co-change with the full noise-control set and a worked command block, lazy reference-density note, shallow-history graceful skip) <!-- R3 -->
- [x] T004 Write Behavior steps 3–4 (frame priority order, taste guard, frame-(c) quantified majority; proposal clustering with the proposal schema incl. in-flight blast radius and the package-scoped elevated blast grade) <!-- R4 -->
- [x] T005 Write Behavior step 5 (ranked report format, "for /fab-dedupe" section, informational suggested-next-action lines, no-proposals success close, documented `Next:`-line opt-out) and the Key Properties table <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Add `"code-reorg": "Maintenance"` to `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go`, update `src/go/fab/cmd/fab/fabhelp_test.go` accordingly, and run `go test ./cmd/fab/...` in `src/go/fab` <!-- R7 -->
- [x] T007 [P] Add the `## /code-reorg [<path>]` section (with Flow skeleton) to `docs/specs/skills.md` and the `helpers: [_srad]` row to its § Skill Helpers table <!-- R7 -->
- [x] T008 [P] Add the `/code-reorg` row to `docs/specs/glossary.md` and to `README.md`'s command tables, checking the README/help edits against `shll standards` (constitution § Toolkit Standards) <!-- R7 -->

### Phase 4: Polish

- [x] T009 Sibling sweep: grep repo-wide (skills, specs, README) for claims made stale by a source-tree reorg skill existing — reorg-family docs-only scope claims, "no skill covers X" phrasing, fab-dedupe boundary statements — and update every occurrence outside `docs/memory/` (memory is hydrate's) <!-- R1 -->

## Execution Order

- T001 blocks T002–T005 (same file, sequential sections)
- T006–T008 independent of each other after T005; T009 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/code-reorg.md` exists with the specified frontmatter (name, behavior-naming description, `helpers: [_srad]`), the `_preamble` read line, and explicit report-only/read-only statements
- [x] A-002 R2: Scope resolution covers the default combined `source_paths` sweep, the scope+file-count echo, and the path-based docs carve-out with sibling-skill pointers
- [x] A-003 R3: All four signal families are specified; the co-change section carries every noise control (window, bulk cap, `-M`, mandated-coupling carve-out, cross-layer whitelist) and a worked command block
- [x] A-004 R4: The taste guard and frame priority are stated with the frame-(c) quantified-majority rule
- [x] A-005 R5: The proposal schema includes prediction failure, move/rename list, blast radius with in-flight exposure, SRAD confidence, and the package-scoped elevated blast grade
- [x] A-006 R6: The report contract covers ranking, the /fab-dedupe section, informational next-action lines, the no-proposals close, and the documented `Next:` opt-out
- [x] A-007 R7: fabhelp.go map entry + test update, skills.md section + helpers row, glossary row, and README rows are all present

### Behavioral Correctness

- [x] A-008 R1: The skill prose contains no instruction that writes a file, creates a change, or runs a `fab status` transition
- [x] A-009 R7: `go test ./cmd/fab/...` passes in `src/go/fab` with the new map entry

### Scenario Coverage

- [x] A-010 R2: The docs-path refusal scenario is verifiable from the skill prose (refusal message + pointer, no analysis performed)
- [x] A-011 R3: The shallow-history scenario is verifiable (co-change skipped with a report note; other signals still run)

### Edge Cases & Error Handling

- [x] A-012 R2: The Error Handling table enumerates the four error cases from the intake's error contract

### Code Quality

- [x] A-013 Pattern consistency: the skill file follows the structure of sibling analysis skills (docs-reorg-*, fab-dedupe) — Contents, Behavior steps, Error Handling + Key Properties tables
- [x] A-014 Owner-or-pointer: rules owned elsewhere (preamble conventions, SRAD grading) are pointed at, never restated alongside a pointer
- [x] A-015 Canonical source only: all kit edits are under `src/kit/`; nothing under `.claude/skills/` is hand-edited
- [x] A-016 Sibling sweep done: no stale reorg-family scope claims or contradicted "no skill covers" phrasing remain outside `docs/memory/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | fabhelp group `"Maintenance"` (not "Planning" where fab-dedupe sits) | code-reorg is an analysis/report skill like docs-reorg-*; fab-dedupe is Planning because it drafts intakes — code-reorg doesn't | S:75 R:85 A:80 D:70 |
| 2 | Certain | Tasks exclude `docs/memory/` edits | Hydrate owns memory writes per the pipeline contract; intake's Affected Memory feeds hydrate, not apply | S:85 R:90 A:95 D:95 |
| 3 | Confident | Default thresholds written as named defaults in the skill file: 12-month window, 20-file bulk cap; junk-drawer and co-change-fraction criteria stated as judgment guidance rather than numeric thresholds | Intake assumption #11 defers exact values to apply; window/cap have intake-stated defaults, the rest resist honest numeric pinning | S:60 R:75 A:65 D:55 |
| 4 | Confident | skills.md Flow skeleton mirrors the docs-reorg-* section shape | Existing sections are the template; § Partial Flow Skeletons documents the format | S:70 R:85 A:85 D:80 |
| 5 | Confident | `/code-reorg` does NOT join the hardcoded TYPICAL FLOW "Maintain docs:" line in fabhelp.go | The line enumerates docs-tree maintenance commands; code-reorg is a source-tree skill — adding it would mislabel its scope | S:75 R:85 A:80 D:70 |
| 6 | Confident | README row placed in the existing Documentation command table alongside the reorg family (no new table created) | Keeps the reorg family colocated; the description itself disambiguates the source-tree scope | S:60 R:85 A:70 D:60 |
| 7 | Certain | T009 sibling sweep required zero edits outside `docs/memory/` — no stale reorg-family docs-only scope claims, "no skill covers X" phrasing, or contradicted fab-dedupe boundary statements found | Swept src/kit, docs/specs, docs/site, README: overview.md Quick Reference omits the whole reorg family (curated, not exhaustive); docs-distill-memory's "structural moves belong to /docs-reorg-memory" is docs/memory-scoped and stays true (code-reorg refuses those paths) | S:80 R:85 A:85 D:80 |

7 assumptions (2 certain, 5 confident).
