# Plan: Skill Prose Drift Fixes (Phase 1)

**Change**: 260808-ip4q-skill-prose-drift-fixes
**Intake**: `intake.md`

## Requirements

### Pipeline Dispatch: Branch Selection

#### R1: Cross-Adapter Branching

The `fab-ff` and `fab-fff` **pipeline-stage** dispatch notes MUST direct dispatch sites to resolve once with `fab resolve-agent <stage> --alias`, surface the resolved profile, and branch on `dispatch=` presence through the canonical `_preamble.md` § CLI-Adapter Dispatch contract. The twin `fab-ff` and `fab-fff` wording MUST remain consistent and pointer-shaped.

The `fab-proceed` **prefix-step** dispatch notes (the role-name resolutions `fast`/`default` at its per-stage-model note and resolve bullets) MUST instead state that prefix steps **always dispatch native**: a resolved `dispatch=` line is **surfaced (compliance visibility) but not taken**, because prefix steps are not pipeline stages — `fab dispatch start <change> <stage>` re-resolves its second argument as a pipeline stage (`default`/`fast` are invalid there) and prefix steps carry no `{stage}-result.yaml` contract. The note MUST carry this reason as a one-line clause so the deviation from the standard branch is self-explaining. `/fab-proceed`'s final `/fab-fff` delegation is unaffected (its own pipeline stages branch normally).

- **GIVEN** a pipeline stage whose resolved profile omits `dispatch=`
- **WHEN** an agent follows `fab-ff` or `fab-fff`
- **THEN** it uses the native two-seam dispatch path
- **AND** a profile containing `dispatch=` routes through the canonical CLI-adapter procedure

- **GIVEN** a prefix-step role resolution in `fab-proceed` (e.g. `fab resolve-agent fast --alias` under `agent.workers: codex`) that emits a `dispatch=` line
- **WHEN** an agent follows the prefix-step note
- **THEN** it surfaces the resolved lines and dispatches natively anyway, without invoking `fab dispatch`

### Pipeline Dispatch: Pane Safety and Completion

#### R2: Safe Pane Override Wording

The `fab-continue` skill MUST call the forced mode a “pane worker” and MUST point to the canonical safety constraints before an agent adds `--pane`; it MUST NOT encourage speculative use.

- **GIVEN** an agent considers overriding automatic dispatch mode from outside tmux
- **WHEN** it reads `fab-continue.md`
- **THEN** the prose identifies `--pane` as forcing a pane worker
- **AND** points to `_preamble.md` § CLI-Adapter Dispatch for the safety requirements

#### R3: Canonical Done-Path and Invariants

The `_pipeline`, `fab-continue`, and `fab-adopt` dispatch descriptions MUST make the canonical `done` path reachable: read the result, run `fab dispatch reap <change> <stage>` unconditionally, then perform the sequencer transition. `fab-adopt` MUST also point to the canonical no-session-command-fallback and no-state-cleanup-after-`done` invariants. Added prose MUST be pointer-shaped and MUST preserve reap-is-not-kill and orchestrator-transition ownership.

- **GIVEN** a CLI-dispatched stage returns `done`
- **WHEN** an agent follows any of the three affected dispatch sites
- **THEN** it reads the result and runs `fab dispatch reap <change> <stage>` before the transition
- **AND** it can reach the no-fallback and no-state-cleanup invariants through the canonical pointer

### Operator Display: Glyph Contract

#### R4: Queue Progress Glyphs

The `fab-operator` queue-progress prose MUST use the health-emoji plus `▶` convention defined by its frame rules and MUST NOT prescribe the banned monochrome geometric progress glyphs.

- **GIVEN** a queue containing completed and current entries
- **WHEN** an operator renders its progress
- **THEN** the description agrees with the skill’s health-emoji and `▶` frame convention

### Setup: Deployment Mechanism

#### R5: Sync Mechanism Accuracy

The `fab-setup` deployment and idempotency descriptions MUST both state the actual mechanism implemented by `fab sync` in `src/go/`; the prose MUST be verified against code before editing and MUST not claim a conflicting copy/symlink mechanism.

- **GIVEN** the current `fab sync` implementation
- **WHEN** the setup and idempotency sections are compared
- **THEN** both describe the same code-verified deployment behavior

### Memory Documentation Skills: Portable References

#### R6: Runtime-Portable FKF Pointer

The `docs-hydrate-memory` skill MUST point deployed agents to `$(fab kit-path)/reference/fkf.md` rather than a dev-repository-only spec path and MUST omit obsolete migration-trajectory narration.

- **GIVEN** the skill is deployed into a user repository
- **WHEN** an agent follows its FKF guidance
- **THEN** every required reference resolves through the installed kit path

#### R7: Live Reorganization Commands Only

The `docs-reorg-memory` skill MUST NOT claim to supersede the nonexistent `/fab-rebalance-memory` skill.

- **GIVEN** the repository’s current skill set
- **WHEN** an agent reads the reorganization skill’s properties
- **THEN** it sees no reference to `/fab-rebalance-memory`

### Skill Structure: Navigation

#### R8: Dedupe Table of Contents

The `fab-dedupe` skill MUST include a `## Contents` section listing every top-level `##` section in document order, matching the repository’s established skill TOC format.

- **GIVEN** the skill exceeds 100 lines
- **WHEN** a reader opens it
- **THEN** the contents list provides links to all top-level sections

### Tool Permissions: Executable Skill Bodies

#### R9: Allowed-Tools Semantics

The `git-pr` and `git-pr-review` skill frontmatter and their SPEC mirrors MUST reflect current official Claude Code `allowed-tools` semantics. If the field restricts tool access, each skill MUST declare every tool family its body requires; if it is advisory, frontmatter MUST remain unchanged and the mirrors MUST record that finding.

- **GIVEN** current official Claude Code documentation and the tools invoked by each skill body
- **WHEN** `allowed-tools` is evaluated
- **THEN** the skill instructions are executable under the documented permission model
- **AND** both SPEC mirrors record the verified interpretation

### Documentation Integrity: Mirrors and Protected Contracts

#### R10: Mirror and Protected-Set Integrity

Every touched `src/kit/skills/*.md` file MUST have its corresponding `docs/specs/skills/SPEC-*.md` mirror updated in the same change. Aggregate specs MUST be swept for affected claims. The change MUST remain Markdown-only, MUST introduce no behavior change, and MUST preserve command grammars, byte-stable output tokens, exit-code tables, result schemas, parser contracts, and anti-drift prohibitions byte-for-byte while editing around them.

- **GIVEN** the completed Phase 1 source edits
- **WHEN** source-to-SPEC coverage, aggregate claims, and protected literals are checked **against the worktree sources** (`src/kit/skills/` and `docs/specs/skills/`)
- **THEN** every touched skill has a synchronized mirror, no aggregate claim is stale, and protected contracts are unchanged. Deployed-copy spot-checks via plain `fab sync` are explicitly NOT part of this change's verification: `fab sync` deploys the pinned system-cache kit, not this worktree's unshipped `src/kit/`, so such a check cannot validate these edits (they deploy on the next release)

### Non-Goals

- Phases 2–4 of `fab/plans/sahil/skill-prose-consolidation.md` are excluded.
- No Go implementation, test, migration, or `docs/memory/` change is part of this apply stage.
- Existing dispatch restatements are not broadly consolidated; Phase 1 only corrects the verified defects with minimal pointers.

## Tasks

### Phase 1: Verification Setup

- [x] T001 Re-verify every Phase 1 source location at HEAD, inspect the `fab sync` implementation, inventory skill/SPEC mirrors and aggregate-spec occurrences, and record before-edit protected-set grep results without modifying source files <!-- R10 -->

### Phase 2: Core Corrections

- [x] T002 [P] Correct the `dispatch=` branch guidance in `src/kit/skills/fab-ff.md`, `src/kit/skills/fab-fff.md` (branch on `dispatch=` per canon), and `src/kit/skills/fab-proceed.md` (prefix steps: always-native, `dispatch=` surfaced-but-not-taken, with the one-line reason per revised R1); update `docs/specs/skills/SPEC-fab-ff.md`, `SPEC-fab-fff.md`, and `SPEC-fab-proceed.md` <!-- R1 --> <!-- rework: review found the CLI-adapter branch is not executable for prefix-step roles — fab dispatch start only accepts pipeline stages (dispatch_start.go:224, agent.go:436-445); fab-proceed guidance must flip to the plan doc's alternative (always-native, surface-only) -->
- [x] T003 Correct `--pane` wording/safety and the CLI `done`/reap path in `src/kit/skills/fab-continue.md`; update `docs/specs/skills/SPEC-fab-continue.md` <!-- R2, R3 -->
- [x] T004 [P] Add the canonical CLI `done`/reap pointer and dispatch invariants to `src/kit/skills/_pipeline.md` and `src/kit/skills/fab-adopt.md`; update `docs/specs/skills/SPEC-_pipeline.md` and `SPEC-fab-adopt.md` <!-- R3 -->
- [x] T005 [P] Replace the conflicting queue-progress glyph prose in `src/kit/skills/fab-operator.md` and update `docs/specs/skills/SPEC-fab-operator.md` <!-- R4 -->
- [x] T006 [P] Align both deployment-mechanism claims in `src/kit/skills/fab-setup.md` to the code-verified `fab sync` behavior and update `docs/specs/skills/SPEC-fab-setup.md` <!-- R5 -->
- [x] T007 [P] Replace the dev-repository FKF path and remove migration narration in `src/kit/skills/docs-hydrate-memory.md`; remove the dead skill reference in `src/kit/skills/docs-reorg-memory.md`; update `docs/specs/skills/SPEC-docs-hydrate-memory.md` and `SPEC-docs-reorg-memory.md` <!-- R6, R7 -->
- [x] T008 [P] Add a complete top-level TOC to `src/kit/skills/fab-dedupe.md` and update `docs/specs/skills/SPEC-fab-dedupe.md` <!-- R8 -->
- [x] T009 Verify current official Claude Code `allowed-tools` semantics, inspect actual tool use in `src/kit/skills/git-pr.md` and `git-pr-review.md`, then update frontmatter if restrictive or otherwise leave it intact; update `docs/specs/skills/SPEC-git-pr.md` and `SPEC-git-pr-review.md` with the verified result <!-- R9 -->

### Phase 3: Integration and Verification

- [x] T010 Sweep `docs/specs/skills.md`, `docs/specs/glossary.md`, and `docs/specs/architecture.md` plus the full skill/SPEC mirror class for affected old claims; make only necessary aggregate corrections and confirm every touched skill has a mirror delta <!-- R10 -->
- [x] T011 Verify against worktree sources only: confirm every dispatch site reaches the reap step, compare protected-set literals and parser-contract headings before/after, and verify the final diff is Markdown-only and prose-only. Do NOT use plain `fab sync` deployed-copy spot-checks as verification (it deploys the pinned system-cache kit, not this worktree's `src/kit/`) and do NOT hand-edit `.claude/skills/` <!-- R10 --> <!-- rework: review found T011's deployed-copy spot-check unsound — all touched skills necessarily mismatch their deployed copies until the next kit release; verification must target src/kit/ + docs/specs/ directly -->
- [x] T012 Fix the stale OpenCode deployment claim in `docs/specs/architecture.md` (~L464, Agent Integration table): it says flat-file symlinks where `src/go/fab-kit/internal/skills.go:35` configures `Mode: copy` — align it to the code-verified copy mechanism, consistent with the fab-setup R5 wording. Then re-sweep the aggregate specs (`docs/specs/skills.md`, `glossary.md`, `architecture.md`) for any OTHER symlink/copy deployment claims touching the R5 mechanism <!-- R10 --> <!-- rework: cycle-2 review found the R10 aggregate-spec sweep incomplete — architecture.md:464 still carried the symlink claim -->
- [x] T013 Record a backlog follow-up in `fab/backlog.md` (matching its existing entry format, with a fresh 4-char id and today's date) for the stale symlink claims in `docs/memory/distribution/kit-architecture.md` (~L115, L188, L374 — say OpenCode uses/resolves symlinks, contradicting `skills.go:35` and the file's own all-four-copy statement). Memory edits stay OUT of this change (Non-Goals); the follow-up preserves present-truth accountability <!-- R10 --> <!-- rework: cycle-2 review should-fix — resolved as a recorded follow-up per the reviewer's alternative, keeping the no-memory Non-Goal intact -->

## Execution Order

- T001 blocks all source edits because every location and protected literal must be verified at HEAD first.
- T003 and T004 both edit dispatch-restatement sites; apply their source changes sequentially so the canonical done-path wording stays consistent.
- T009 depends on current official documentation and local skill-body inspection, but is otherwise independent of T002–T008.
- T010 follows all source and mirror edits; T011 is the terminal verification task.

## Acceptance

### Functional Completeness

- [x] A-001 R1: The `fab-ff`/`fab-fff` pipeline-stage dispatch notes branch on `dispatch=` through the canonical pointer with twin wording aligned; the `fab-proceed` prefix-step notes state always-native dispatch with `dispatch=` surfaced-but-not-taken and carry the one-line reason.
- [x] A-002 R2: `fab-continue` calls the forced mode a pane worker and points to the canonical safety constraints before `--pane` is used.
- [x] A-003 R3: `_pipeline`, `fab-continue`, and `fab-adopt` all carry a reachable result-read → unconditional reap → sequencer-transition path, and adopt points to both missing invariants.
- [x] A-004 R4: `fab-operator` queue-progress prose uses the health-emoji plus `▶` convention and no longer prescribes `●` or `◌` there.
- [x] A-005 R5: Both `fab-setup` mechanism claims agree with the inspected `fab sync` implementation.
- [x] A-006 R6: `docs-hydrate-memory` uses the installed-kit FKF reference and contains no obsolete migration aside at the defect site.
- [x] A-007 R7: `/fab-rebalance-memory` is absent from `docs-reorg-memory`.
- [x] A-008 R8: `fab-dedupe` has a complete, ordered TOC matching all of its top-level sections.
- [x] A-009 R9: `git-pr` and `git-pr-review` permissions are consistent with current official `allowed-tools` semantics and their actual tool use, with both mirrors recording the result.
- [x] A-010 R10: Every touched canonical skill has a matching SPEC mirror delta, with aggregate specs consistent and no out-of-scope source or memory edits.

### Behavioral Correctness

- [x] A-011 R1: In `fab-ff`/`fab-fff`, a missing `dispatch=` still denotes native dispatch while a present `dispatch=` denotes the CLI adapter, and the edits introduce no third branch; `fab-proceed`'s prefix-step carve-out never routes a role-name resolution into `fab dispatch`.
- [x] A-012 R3: Reap remains pane hygiene rather than state cleanup or kill, and the orchestrator remains the sole owner of stage transitions.
- [x] A-013 R5: Deployment wording describes current behavior without changing `fab sync` itself.
- [x] A-014 R9: Permission metadata does not grant unrelated tool access beyond what each skill body requires.

### Scenario Coverage

- [x] A-015 R2: The outside-tmux force-pane scenario directs the reader to the canonical prerequisite/hard-error rules rather than implying `--pane` is always safe.
- [x] A-016 R3: The CLI `done` scenario is traceable from each affected dispatch site through result reading, unconditional reap, and normal transition handling.
- [x] A-017 R6: The FKF reference scenario works in a user repository that lacks fab-kit’s dev-only `docs/specs/fkf.md`.

### Edge Cases & Error Handling

- [x] A-018 R3: The pointer wording does not weaken `failed`, `failed (no-result)`, `orphaned`, restart-budget, or never-send-keys handling around the edited dispatch prose.
- [x] A-019 R9: The allowed-tools conclusion is supported by official documentation rather than inferred solely from current local behavior.

### Code Quality

- [x] A-020 Readability and maintainability: Corrections are minimal, pointer-shaped, and easy to trace to the canonical owner.
- [x] A-021 Pattern consistency: Skill TOC, dispatch pointers, frontmatter, and SPEC updates follow neighboring file conventions.
- [x] A-022 No unnecessary duplication: New prose does not restate canonical dispatch or pane-safety contracts.
- [x] A-023 Canonical source only: No direct edits were made under `.claude/skills/`; deployed-copy changes arise only from `fab sync`.
- [x] A-024 SPEC-mirror sync: Every edited `src/kit/skills/*.md` file has its corresponding `docs/specs/skills/SPEC-*.md` updated.
- [x] A-025 CLI stability: No Go file or CLI command signature changed, so `_cli-fab.md` and Go tests require no update.
- [x] A-026 Protected-set integrity: Exact command grammars, byte-stable tokens, exit-code tables, result schemas, parser contracts, and anti-drift prohibitions remain byte-identical.

## Notes

- Check acceptance items during review, not apply.
- No Go tests are required because this change is Markdown-only.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|

0 assumptions.
