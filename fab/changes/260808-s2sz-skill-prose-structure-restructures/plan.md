# Plan: Skill Prose→Structure Restructures (Phase 3)

**Change**: 260808-s2sz-skill-prose-structure-restructures
**Intake**: `intake.md`

## Requirements

### Shared Dispatch Wiring: `_preamble.md`

#### R1: Structured per-stage dispatch contract

`src/kit/skills/_preamble.md` MUST replace the long Per-Stage Model Resolution, CLI-Adapter Dispatch, and Confidence Scoring prose with the tables, concise procedures, and canonical pointers specified by the intake. Every dispatch state, output token, recovery rule, native/CLI seam, override rule, pane-mode constraint, result schema, transition-ownership rule, and confidence-gate fact SHALL survive without behavioral change. `docs/specs/skills/SPEC-_preamble.md` MUST remain synchronized.

- **GIVEN** the current canonical `_preamble.md` at apply entry
- **WHEN** its named sections are restructured
- **THEN** the same executable obligations remain discoverable by heading and protected literal
- **AND** the dispatch-prompt obligations and result schemas remain intact

### CLI Command Reference: `_cli-fab.md`

#### R2: Structured command contracts

`src/kit/skills/_cli-fab.md` MUST restructure the `fab config`, `fab pane map --json`, `fab dispatch`, `fab pr-meta`, `fab memory-index`, `fab batch`, and exit-code commentary named by the intake. Exact usage grammars, flags, mode ladders, output literals, exit codes, severity markers, tier precedence, archive consent matrix, reap contract, and restart behavior SHALL be retained. `docs/specs/skills/SPEC-_cli-fab.md` MUST remain synchronized.

- **GIVEN** the current command-reference sections located by heading
- **WHEN** their narrative is converted to tables, lists, and pointers
- **THEN** every CLI contract remains semantically complete and byte-stable where protected

### Operator and Setup Skills

#### R3: Structured operator and setup procedures

`src/kit/skills/fab-operator.md` and `src/kit/skills/fab-setup.md` MUST receive the Phase 3b structural rewrites described in the intake, with their matching SPEC mirrors updated. The operator runtime no-fence blockquote, frame example, health-emoji table, required cross-repo caveat, setup migration output literals, constitution severity mapping, and ecosystem-to-`test_paths` table SHALL remain verbatim.

- **GIVEN** the current operator and setup instructions
- **WHEN** strategic decisions, frame rules, argument classification, config creation, and migration routing are tabulated
- **THEN** protected examples and operational outcomes remain unchanged

### Planning and Review Helpers

#### R4: Structured helper contracts

`src/kit/skills/_intake.md`, `_review.md`, `_generation.md`, `_srad.md`, and `_cli-agents.md` MUST be restructured per Phase 3b, and each matching SPEC mirror MUST be updated. Parser headings and ID literals (`## Requirements`, `## Tasks`, `## Acceptance`, `T{NNN}`, `A-{NNN}`, `R#`, `<!-- R# -->`), SRAD scoring/markers/assumption shape, and the SRAD Worked Examples SHALL remain intact.

- **GIVEN** the current helper contracts
- **WHEN** repeated rationale and consumer descriptions are converted to compact structure or owner pointers
- **THEN** generation, review, SRAD, intake, and CLI-agent behavior remains equivalent

### Git, Documentation, and Status Skills

#### R5: Structured leaf-skill procedures

`src/kit/skills/git-pr.md`, `git-pr-review.md`, `docs-distill-memory.md`, `docs-reorg-memory.md`, `fab-switch.md`, `fab-status.md`, and `fab-dedupe.md` MUST receive the specified Phase 3b restructures, with every matching SPEC mirror updated. Protected commit/status tokens, staging rationale, Copilot authentication facts, reply prefixes, polling directive, approval gates, and the `bulk approval deliberately NOT offered` rule SHALL survive.

- **GIVEN** each named current skill section located by heading
- **WHEN** ladders, outcome classes, output shapes, properties, and rationale are converted to tables or short lists
- **THEN** users and downstream parsers observe no behavior or token change

### Mirrors and Cross-References

#### R6: Complete mirror and reference synchronization

Every touched `src/kit/skills/*.md` file MUST have its matching `docs/specs/skills/SPEC-*.md` changed in the same change. `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md`, `docs/specs/harness-adapters.md`, `docs/specs/stage-models.md`, and the intake's six affected-memory candidates MUST be swept for stale section names, moved ownership, or prose-shape claims; only genuinely stale files SHALL be edited.

- **GIVEN** the completed canonical skill restructures
- **WHEN** mirror and aggregate references are searched by old phrase and section name
- **THEN** no live document relies on a removed wording or outdated ownership claim

### Verification and Scope

#### R7: Prose-only, contract-preserving verification

The change MUST edit no Go source, CLI surface, template, migration, or deployed `.claude/skills/` source. Verification MUST compare heading sets and protected literals against `HEAD`, run `fab sync`, spot-load the deployed copies, confirm every touched skill has a touched SPEC mirror, sweep aggregate references, and confirm each dispatch site reaches the canonical contract through exactly one pointer.

- **GIVEN** all tasks are implemented
- **WHEN** the intake verification suite is run
- **THEN** all checks pass with zero protected-contract loss and no out-of-scope source changes

### Non-Goals

- No behavior, CLI, Go, template, migration, or memory-truth change.
- No Phase 4 policy amendments or new `_git.md`/`_reorg.md` helper extraction.
- No edits to `.claude/skills/`; deployed copies are generated only by `fab sync` for verification.

## Tasks

### Phase 1: Canonical Partials

- [x] T001 Restructure `src/kit/skills/_preamble.md` by current section headings and update `docs/specs/skills/SPEC-_preamble.md`, preserving every protected dispatch and scoring contract. <!-- R1 -->
- [x] T002 Restructure `src/kit/skills/_cli-fab.md` by current command headings and update `docs/specs/skills/SPEC-_cli-fab.md`, preserving usage, output, marker, mode, and exit-code contracts. <!-- R2 --> <!-- rework: batch-new failure row dropped the empty-content backlog-item case and the explicit exit-0 guarantee — restore both qualifiers in the table (review must-fix 2) -->

### Phase 2: Remaining Skill Families

- [x] T003 Restructure `src/kit/skills/fab-operator.md` and `src/kit/skills/fab-setup.md`, then update `docs/specs/skills/SPEC-fab-operator.md` and `docs/specs/skills/SPEC-fab-setup.md`. <!-- R3 --> <!-- rework cycle 2: fab-operator.md:343 Rule 4 says "Numbered/open-ended prompt" — restore to numbered/multi-choice menus per SPEC-fab-operator.md:109-115 and docs/memory/runtime/operator.md:182-189 (review-2 must-fix; cycle-1 fix of the entry-forms mismatch is done and verified) -->
- [x] T004 Restructure `src/kit/skills/_intake.md`, `_review.md`, `_generation.md`, `_srad.md`, and `_cli-agents.md`, then update all five matching `docs/specs/skills/SPEC-*.md` mirrors. <!-- R4 -->
- [x] T005 Restructure `src/kit/skills/git-pr.md`, `git-pr-review.md`, `docs-distill-memory.md`, `docs-reorg-memory.md`, `fab-switch.md`, `fab-status.md`, and `fab-dedupe.md`, then update all seven matching SPEC mirrors. <!-- R5 -->

### Phase 3: Integration and Verification

- [x] T006 Sweep and update aggregate/related specs and affected-memory references only where the restructures make them stale; verify a matching touched SPEC mirror exists for every touched skill. <!-- R6 --> <!-- rework: docs/memory/runtime/operator.md:95,97,424,456 still cross-references the retired Working-a-Change 3-row table (review must-fix 1) -->
- [x] T007 Run the full intake verification suite: heading-set and protected-literal comparisons against `HEAD`, dispatch-pointer sweep, `fab sync` success plus sync-source verification per A-014 (as relaxed), mirror/aggregate sweep, and scope/status checks. <!-- R7 --> <!-- rework cycle 3 (user-approved relaxation): fab/.fab-version pins released 2.17.0, so .claude/skills byte-equality is unsatisfiable until this kit ships; verify instead that `just install`'s sync source ~/.fab-kit/local-versions/2.17.1/kit/skills/ matches src/kit/skills/ for every touched skill, and that `fab sync` succeeds -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `_preamble.md` presents the required seam, override, branch, state, recovery, peek, pane, and confidence structures without contract loss.
- [x] A-002 R2: `_cli-fab.md` presents the required command/taxonomy structures without losing command behavior or protected literals.
- [x] A-003 R3: Operator and setup restructures are complete and their protected examples/tables/literals remain intact.
- [x] A-004 R4: Intake, review, generation, SRAD, and CLI-agent helper restructures are complete with parser and calibration anchors intact.
- [x] A-005 R5: Git, documentation, switch, status, and dedupe restructures are complete with protected workflow contracts intact.
- [x] A-006 R6: Every touched skill has a touched matching SPEC mirror and all aggregate/related references are current.
- [x] A-007 R7: Verification passes and no Go, CLI-surface, template, migration, or canonical-source boundary is crossed.

### Behavioral Correctness

- [x] A-008 R1: Native-versus-CLI branching, one-restart recovery, no-send-keys rule, reap behavior, and transition ownership are unchanged.
- [x] A-009 R2: CLI mode ladders, state names, output tokens, severity markers, tier precedence, and exit codes are unchanged.
- [x] A-010 R3: Operator decision routing, watchdog behavior, status-frame formatting, setup modes, and migration routing are unchanged.
- [x] A-011 R4: Intake generation, review mode gating, plan traceability, and SRAD scoring/visibility behavior are unchanged.
- [x] A-012 R5: PR publishing/review, approval gates, status rendering, switching, and dedupe behavior are unchanged.

### Scenario Coverage

- [x] A-013 R7: Heading sets and protected literals compare cleanly against `HEAD`, with any intentional non-parser heading difference explained by a complete reference sweep.
- [x] A-014 R7: `fab sync` succeeds, and the sync-source kit populated by `just install` (`~/.fab-kit/local-versions/2.17.1/kit/skills/`) matches the canonical `src/kit/skills/` sources for every touched skill. (`.claude/skills/` byte-equality is deliberately NOT required: `fab/.fab-version` pins the released 2.17.0 kit, so deployed copies reflect this change only after it ships in a release — per the governing plan's verification intent, which asks only that sync succeeds and deployed copies load.) <!-- clarified: relaxed from deployed-copy byte-equality per user decision after review-3; see T007 rework note -->

### Edge Cases & Error Handling

- [x] A-015 R6: Aggregate specs and affected-memory files with no stale claim remain unchanged rather than receiving cosmetic churn.
- [x] A-016 R7: Dispatch sites retain exactly one canonical pointer and still reach the reap step through `_preamble.md`.

### Code Quality

- [x] A-017: Readability and maintainability improve through tables/lists while the repository's existing CommonMark style is preserved.
- [x] A-018: Pattern consistency is maintained across skill files and their SPEC mirrors.
- [x] A-019: No unnecessary duplication is introduced; owner pointers replace re-derived prose where directed.
- [x] A-020: Canonical-source discipline is maintained: no direct `.claude/skills/` edit and no skill change lacks a SPEC mirror.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Group the 16 named skill files into five implementation tasks plus mirror/reference and verification tasks. | The intake supplies an exhaustive per-file breakdown; grouping changes execution packaging only. | S:95 R:90 A:95 D:90 |
| 2 | Certain | Affected-memory files are read-only sweep candidates unless an actually stale structural reference is found. | The intake explicitly says behavior is unchanged and memory edits are expected to be nil. | S:95 R:95 A:95 D:95 |

2 assumptions (2 certain, 0 confident, 0 tentative).
