# Plan: Light Lane Post-Merge Polish

**Change**: 260811-3olp-light-lane-post-merge-polish
**Intake**: `intake.md`

## Requirements

### Kit Skills: `_pipeline.md` Auto-Rework item 3

#### R1: Scope `/fab-adopt`'s loop consumption to the FULL branch
Auto-Rework Loop item 3 ("Re-dispatch apply (resume-first)") SHALL state, in a half-line, that `/fab-adopt`'s partial consumption of the loop uses the **FULL** branch (adoption has no lane fork). The edit MUST be made in the canonical source `src/kit/skills/_pipeline.md`, never the deployed `.claude/skills/` copy.

- **GIVEN** item 3's FULL-lane / LIGHT-lane fork
- **WHEN** a reader (or `/fab-adopt` itself, which reuses § Auto-Rework Loop) consults item 3
- **THEN** the text states that `/fab-adopt` consumes the FULL branch
- **AND** the addition stays a half-line — no restatement of § Worker Continuation mechanics (owner-or-pointer)

### Specs: lane-accurate claims

#### R2: Qualify the stage-models.md mirror claim to the full lane
The `/fab-continue` ship/review-pr bullet in `docs/specs/stage-models.md` § Skill wiring SHALL qualify "mirroring `/fab-fff` Steps 4–5 exactly" to `/fab-fff`'s **full-lane** Steps 4–5.

- **GIVEN** the light lane runs `/fab-fff` Steps 4–5 inline with no `fab resolve-agent`
- **WHEN** the bullet describes `/fab-continue`'s ship/review-pr role resolution
- **THEN** the mirror claim names the full-lane Steps 4–5, not Steps 4–5 unconditionally

#### R3: Make the user-flow.md rework edge label lane-accurate
The mermaid state-diagram edge `review --> apply: auto-rework (sub-agent, fab-ff/fab-fff)` in `docs/specs/user-flow.md` SHALL no longer assert sub-agent rework unconditionally; the label MUST stay terse (a mermaid edge label) and MUST remain valid mermaid syntax.

- **GIVEN** light-lane rework runs inline in the orchestrator's context
- **WHEN** the diagram labels the review → apply auto-rework edge
- **THEN** the label is accurate for both lanes (qualified, or with the unconditional mechanism claim dropped)

#### R4: Trim the glossary "Light lane" entry to definition + pointer
The "Light lane" entry in `docs/specs/glossary.md` SHALL be reduced to a short definition (the inline execution lane of `/fab-ff`/`/fab-fff`, chosen once at apply entry by plan task count, with the full-lane contrast) plus the existing pointer to `_pipeline.md` § Light Lane. Restated mechanics (dispatch/resolve-agent behavior, rework budget, promotion valve) MUST be removed per the owner-or-pointer convention.

- **GIVEN** `_pipeline.md` § Light Lane owns the lane mechanics
- **WHEN** the glossary defines "Light lane"
- **THEN** the entry carries definition + pointer only, no mechanics restatement

#### R5: Sibling sweep of the corrected phrase-classes
Per `code-quality.md` § Sibling Sweeps, the change SHALL sweep the repo (twin skills, aggregate specs, memory files) for other occurrences of the corrected claim classes — (a) unconditional "sub-agent rework" assertions, (b) unqualified "mirrors fab-fff Steps 4–5" claims, (c) light-lane mechanics restated outside the owner — and update any found, or record that none exist.

- **GIVEN** the three phrase-classes corrected by R1–R4
- **WHEN** each class is grepped repo-wide (src/kit/skills/, docs/specs/, docs/memory/)
- **THEN** no other stale occurrence remains

### Non-Goals

- Backlog item 4 (SPEC-`_pipeline.md` glyphs) — OBE; the mirror tree was retired (260811-rehi)
- Any behavior, Go code, template, or memory-content change — this is prose accuracy only

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add a half-line to Auto-Rework item 3 in `src/kit/skills/_pipeline.md` scoping `/fab-adopt`'s loop consumption to the FULL branch <!-- R1 -->
- [x] T002 [P] Qualify "mirroring `/fab-fff` Steps 4–5 exactly" to the full-lane Steps 4–5 in `docs/specs/stage-models.md` § Skill wiring <!-- R2 -->
- [x] T003 [P] Reword the `review --> apply` mermaid edge label in `docs/specs/user-flow.md` to be lane-accurate <!-- R3 -->
- [x] T004 [P] Trim the "Light lane" entry in `docs/specs/glossary.md` to definition + pointer <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Sibling sweep: grep the corrected phrase-classes repo-wide (unconditional sub-agent-rework claims, unqualified Steps 4–5 mirror claims, light-lane mechanics restated outside `_pipeline.md`) and fix any further occurrences — found and fixed two class-(b) occurrences in memory: `docs/memory/pipeline/execution-skills.md` and `docs/memory/_shared/context-loading.md` (both now full-lane-qualified); aggregate specs, twin skills, and `user-flow.md`'s second rework edge (line 69, no mechanism claim) verified already accurate; historical logs/findings frozen-on-write, deliberately untouched <!-- R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_pipeline.md` Auto-Rework item 3 names `/fab-adopt` and scopes its loop consumption to the FULL branch, in a half-line
- [x] A-002 R2: `docs/specs/stage-models.md`'s mirror claim reads full-lane Steps 4–5
- [x] A-003 R3: `docs/specs/user-flow.md`'s rework edge label no longer asserts sub-agent rework unconditionally and remains valid mermaid
- [x] A-004 R4: the glossary "Light lane" entry is definition + pointer only

### Behavioral Correctness

- [x] A-005 R3: the reworded label is consistent with `_pipeline.md` § Light Lane (inline rework, same cycle budget)
- [x] A-006 R4: no lane fact was lost that isn't present in the owner (`_pipeline.md` § Light Lane)

### Scenario Coverage

- [x] A-007 R5: the three phrase-class greps ran repo-wide and every hit was either fixed or verified already accurate

### Code Quality

- [x] A-008 Pattern consistency: edits match each file's surrounding prose/diagram style
- [x] A-009 No unnecessary duplication: no mechanics restated alongside a pointer (owner-or-pointer anti-pattern)
- [x] A-010 Canonical source only: no edits under `.claude/skills/`; the skill edit lands in `src/kit/skills/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Mermaid label strategy: prefer a short lane qualifier; drop the mechanism claim entirely only if the qualified form reads too long on the edge | Backlog allows either; qualified keeps information, terseness is a diagram constraint | S:75 R:90 A:80 D:70 |
| 2 | Confident | Glossary trim keeps the ≤5-task fork value and `--light`/`--full` flag names in the definition | They identify the concept (how the lane is chosen); everything downstream of the choice is mechanics owned by `_pipeline.md` | S:70 R:90 A:80 D:75 |
| 3 | Certain | No `fab sync` / deployed-copy step needed | `.claude/skills/` deploys from the installed kit version, not from this worktree's `src/kit/`; the source edit ships with the next release | S:85 R:90 A:95 D:90 |
| 4 | Confident | The R5 sweep's two memory-file fixes are in scope, superseding the intake's "Affected Memory: none" | code-quality.md § Sibling Sweeps places the memory file documenting a skill's behavior in the sweep class even when Affected Memory missed it; both edits are the same stale claim class (b) the change exists to fix | S:75 R:85 A:85 D:80 |

4 assumptions (1 certain, 3 confident, 0 tentative).
