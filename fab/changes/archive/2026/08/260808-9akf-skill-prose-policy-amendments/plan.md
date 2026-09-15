# Plan: Skill Prose Policy Amendments (Phase 4)

**Change**: 260808-9akf-skill-prose-policy-amendments
**Intake**: `intake.md`

## Requirements

### Optimizer Policy: Partial Targeting

#### R1: Dedicated Partial Optimization
`src/kit/skills/internal-skill-optimize.md` MUST distinguish consumer-skill optimization from explicitly targeted partial optimization: a consumer pass MUST treat every `_*.md` partial as read-only reference context, while a dedicated pass invoked with a partial name MUST be permitted to apply the same content signals directly to that partial. The Arguments, Pre-flight, Analysis, and Constraints statements MUST express this distinction consistently without changing the existing structural-check behavior.

- **GIVEN** `/internal-skill-optimize fab-continue` targets a consumer skill
- **WHEN** the optimizer loads `_*.md` partials as shared context
- **THEN** it does not trim any partial as a side effect of optimizing `fab-continue`
- **AND** an explicit `/internal-skill-optimize _preamble` invocation is a legitimate dedicated partial-optimization pass

### Optimizer Policy: Content Signals and Contracts

#### R2: Transition Narration Signal
The optimizer's Content signals table MUST identify transition narration such as "no longer", "used to", version/change-ID archaeology, and supersession prose as bloat. It MUST direct the optimizer to rewrite instructions as present truth, preserving the durable prohibition and rationale while dropping history, and MUST cite the deployed `$(fab kit-path)/reference/fkf.md` §3.3 source.

- **GIVEN** instruction prose narrates how a rule changed over time
- **WHEN** content-signal analysis runs
- **THEN** the optimizer flags the narration and preserves only the current rule plus its don't-re-break rationale

#### R3: Protected Output Literals
Optimization Rule 6 MUST apply only to illustrative examples. Output literals that the target skill or a sibling greps or string-matches MUST be treated as contracts and MUST NOT be consolidated or reworded, with the `git-pr` checkmark outputs and `git-pr-review` disposition prefixes retained as explicit examples.

- **GIVEN** a skill contains multiple output literals consumed by matching logic
- **WHEN** the one-output-example rule is evaluated
- **THEN** those literals are classified as contracts rather than illustrative examples
- **AND** `✓ commit`, `Fixed —`, `Deferred —`, and `Skipped —` remain byte-identical in their owning git skills

#### R4: Sibling Duplication Signal (mode-scoped)
The optimizer MUST carry a sibling-duplication content signal **scoped to the files actually loaded in each mode**. In any content pass (single-skill or batch), a rule restated between the analyzed skill and an already-loaded `_*.md` partial MUST be detected — keep the owning statement and reduce the consumer to a pointer, or report the pair as an extraction candidate. Cross-skill comparison (twins such as `fab-ff` ↔ `fab-fff`) is a **batch-mode** duty: after per-file analysis, batch mode MUST compare findings across the analyzed files and report duplicated-rule clusters, report-only. Single-skill mode does not load sibling skills and the signal text MUST NOT claim cross-skill detection there. Constraint 3 MUST remain unchanged.

- **GIVEN** a single-skill pass on a consumer that restates a rule owned by a loaded `_*.md` partial
- **WHEN** content-signal analysis runs
- **THEN** the consumer↔partial duplication is flagged (owner kept; consumer reduced to a pointer or reported)
- **GIVEN** batch analysis finds the same owned rule in sibling files
- **WHEN** per-file analysis completes
- **THEN** the optimizer compares findings across files and reports the duplicated-rule cluster without extracting, merging, or moving content

### Project Policy: Rule Ownership

#### R5: Owner-or-Pointer Convention
`fab/project/code-quality.md` MUST define the project-specific anti-pattern that a skill may state a rule it owns or point to the owning file, but never both, including the drift rationale and PR #539 reap example. Its Sibling & Mirror Sweeps section MUST distinguish reactive sweeps from the convention's preventative role.

- **GIVEN** an apply or review agent edits `src/kit/skills/*.md`
- **WHEN** it loads the project's code-quality policy
- **THEN** it is instructed to choose exactly one of an owned statement or an owner pointer
- **AND** it understands that sibling sweeps catch drift while ownership prevents the duplicate copy

### Documentation and Scope: Mirror Integrity

#### R6: Mirrors, Exclusions, and Structural Invariants
The per-skill SPEC and any aggregate SPEC that restates the amended optimizer behavior MUST mirror R1–R4. The change MUST NOT edit `_preamble`, `git-pr`, `git-pr-review`, the helper allow-list, or Constraint 3; MUST NOT create `_git.md` or `_reorg.md`; and MUST preserve the `##` heading sets of both edited source/policy files. After canonical edits, `fab sync` MUST complete and its installed-kit deployed copies MUST be spot-checked without hand-editing them.

- **GIVEN** the canonical optimizer skill and project policy have been amended
- **WHEN** the mirror sweep and deployment verification run
- **THEN** the per-skill and aggregate SPEC prose describe the same current policy
- **AND** excluded files, protected literals, Constraint 3, and heading sets remain unchanged

### Non-Goals

- Extracting `_git.md` or `_reorg.md`, or changing any git skill — explicitly skipped for this policy-only phase.
- Editing `src/kit/skills/_preamble.md`, `docs/specs/skills/SPEC-_preamble.md`, or the `helpers:` allow-list — the ownership convention is project-local authoring guidance.
- Widening internal-skill-optimize Constraint 3 — content movement remains outside the optimizer's automatic write scope.
- Changing the constitution, shll standards, Go code, tests, migrations, or memory — none is required by this prose-only change.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Update all four partial-targeting statements in `src/kit/skills/internal-skill-optimize.md` to distinguish consumer passes from dedicated partial passes while preserving structural-check semantics. <!-- R1 -->
- [x] T002 Add the FKF-grounded transition-narration row to the Content signals table in `src/kit/skills/internal-skill-optimize.md`. <!-- R2 -->
- [x] T003 Scope Optimization Rule 6 to illustrative examples and protect string-matched output literals in `src/kit/skills/internal-skill-optimize.md`. <!-- R3 -->
- [x] T004 Add the sibling-duplication Content signal and batch-mode cross-file comparison/reporting step in `src/kit/skills/internal-skill-optimize.md` without widening Constraint 3, scoping the signal by mode per R4: consumer↔partial duplication in any content pass (partials are loaded in pre-flight); cross-skill/twin cluster comparison in batch mode only — the signal text must not claim cross-skill detection in single-skill mode. <!-- R4 --> <!-- rework: review cycle 1 — unscoped signal claimed cross-skill detection in single-skill mode, where siblings are never loaded; requirement revised to mode-scoped detection -->
- [x] T005 Add the owner-or-pointer anti-pattern and preventative-vs-reactive sweep cross-reference to `fab/project/code-quality.md`. <!-- R5 -->

### Phase 3: Integration & Verification

- [x] T006 Mirror the amended optimizer policy in `docs/specs/skills/SPEC-internal-skill-optimize.md` and the restating aggregate section in `docs/specs/skills.md`; leave unrelated aggregate specs unchanged. <!-- R6 --> <!-- rework: re-align mirrors to the mode-scoped R4 wording from cycle 1 -->
- [x] T007 Run `fab sync`, spot-check deployed copies, and verify excluded-file diffs, protected literals, Constraint 3, and both edited files' `##` heading sets. <!-- R6 --> <!-- rework: re-verify after cycle-1 signal rewording -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Consumer optimization treats partials as read-only context while an explicitly named partial is a valid content-optimization target at all four policy statement sites.
- [x] A-002 R2: Transition narration is a Content signal that applies FKF §3.3 present-truth guidance through the shipped kit-path reference.
- [x] A-003 R3: Rule 6 is limited to illustrative examples and explicitly protects output literals used by grep or string matching.
- [x] A-004 R4: Consumer↔partial duplication is detected in any content pass; cross-skill/twin clusters are compared and reported in batch mode only, without automatic extraction; the signal text claims no cross-skill detection in single-skill mode.
- [x] A-005 R5: Project code-quality policy states the owner-or-pointer convention and explains how it prevents the duplicated copies that sweeps detect after divergence.
- [x] A-006 R6: The per-skill and aggregate SPEC mirrors match the canonical optimizer behavior; `fab sync` completes and deployed copies are spot-checked without hand edits.

### Behavioral Correctness

- [x] A-007 R1: Structural checks still run on all files, including partials, and the narrowed content exemption does not alter their behavior.
- [x] A-008 R3: `✓ commit`, `Fixed —`, `Deferred —`, and `Skipped —` are byte-identical in `git-pr.md` and `git-pr-review.md`.
- [x] A-009 R4: Constraint 3 remains unchanged and batch sibling findings remain report-only.
- [x] A-010 R6: `_git.md`, `_reorg.md`, `_preamble.md`, both git skills, `SPEC-_preamble.md`, and the helper allow-list are untouched.

### Scenario Coverage

- [x] A-011 R1: Both consumer-skill and explicitly targeted-partial invocation scenarios are represented consistently in the optimizer policy.
- [x] A-012 R2: The signal covers temporal phrases, supersession/version prose, and change-ID archaeology while retaining durable rationale.
- [x] A-013 R4: The signal covers consumer-versus-partial duplication in every mode and twin-skill clusters in batch mode, and its wording matches each mode's actually-loaded file set.
- [x] A-014 R5: The project guidance applies to every `src/kit/skills/*.md` edit and includes the PR #539 reap-drift example.

### Edge Cases & Error Handling

- [x] A-015 R3: Contractual output literals are not mistaken for redundant illustrative examples.
- [x] A-016 R6: No `##` heading is added, removed, or renamed in either edited source/policy file.

### Code Quality

- [x] A-017 Readability and maintainability: Policy wording is concise, imperative, and unambiguous.
- [x] A-018 Pattern consistency: Skill and SPEC edits follow surrounding Markdown structure and terminology.
- [x] A-019 No unnecessary duplication: Owned rules are stated once or referenced by pointer, consistent with the new convention.
- [x] A-020 Canonical source only: No deployed `.claude/skills/` file is hand-edited; deployed changes come only from `fab sync`.
- [x] A-021 SPEC-mirror sync: Every canonical skill policy amendment is represented in `docs/specs/skills/SPEC-internal-skill-optimize.md` and applicable aggregate prose.
- [x] A-022 Sibling and mirror sweep: Repository-wide phrase searches confirm no stale restatement in the applicable mirror class.

## Notes

- Review owns `## Acceptance`; apply does not mark these items.
- This is a prose-only change; no automated code tests are required.

## Deletion Candidates

- None — this change adds new policy functionality without making existing code redundant.

## Assumptions

0 assumptions.
