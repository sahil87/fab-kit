# Plan: Repoint Skill Read-Paths to .agents/skills and Gate .claude on Claude

**Change**: 260908-yd9s-repoint-agents-skills-gate-claude
**Intake**: `intake.md`

## Requirements

### Kit Skills: Read-Path Repoint

#### R1: All fab-internal read-path references target `.agents/skills/`
Every read-path reference in `src/kit/skills/*.md` that names `.claude/skills/…` SHALL be changed to `.agents/skills/…`. Measured surface: 37 references across 23 files — 19 instances of the identical header boilerplate line plus ~18 helper/skill read instructions (`_preamble.md` § Skill Helper Declaration's `.claude/skills/{helper}/SKILL.md`, § Subagent Dispatch item 1, `_pipeline.md`/orchestrator dispatch-prompt sections, `fab-continue.md` stage-conditional reads, `_intake` pointers in fab-new/fab-draft/fab-proceed, etc.). `_preamble.md`'s QUOTED canonical boilerplate sentence ("Each skill file should begin with: …") MUST be updated too, or the sweep re-diverges on the next authored skill.

- **GIVEN** the post-change `src/kit/skills/` tree
- **WHEN** grepping it for `.claude/skills`
- **THEN** zero matches remain (all read paths say `.agents/skills/…`; deploy-target descriptions in skill prose, if any, use gated wording without a read-path form)

#### R2: Repointed paths are valid on every machine
The repoint target SHALL be the deployed workspace copy `.agents/skills/{name}/SKILL.md` — guaranteed present post-#647 (`AlwaysOn` row) and byte-identical to the `.claude` copy. No reference may target `$(fab kit-path)/skills/` (version-resolved cache, not the workspace contract).

- **GIVEN** a fresh checkout after `fab sync` with NO agent CLIs on PATH
- **WHEN** a dispatched worker follows any repointed read instruction
- **THEN** the file exists at the referenced `.agents/skills/…` path

### Sync: Re-Tiered Deploy Gates

#### R3: `.claude` targets gated on the `claude` CLI
In `src/go/fab-kit/internal/skills.go`'s `agentConfig` table (lines 53–56): the "Claude Code" row SHALL drop `AlwaysOn: true` and restore `CLIs: []string{"claude"}`; the "Agents dir" row SHALL keep `AlwaysOn: true`; the "OpenCode" row is unchanged. `FAB_AGENTS` semantics are unchanged from #647 — always-on rows bypass `agentAvailable`; the `.claude` row re-joins the gated set it governs.

- **GIVEN** `FAB_AGENTS=codex` (no `claude`)
- **WHEN** `fab sync` runs
- **THEN** `.agents/skills/` deploys fully, `.claude/skills/` is skipped with `Skipping Claude Code: claude not found in PATH`-style output, and existing `.claude/` content is preserved untouched

#### R4: `.claude/settings.local.json` scaffold write shares the gate
The scaffold write of `.claude/settings.local.json` (the generic scaffold tree-walk — `scaffoldTreeWalk` in `src/go/fab-kit/internal/scaffold.go`, called from `sync.go` ~:98–100, merging `src/kit/scaffold/.claude/fragment-settings.local.json`) SHALL be gated on the same `claude` availability result as the deploy row. Mechanism is implementer's choice (front-runner: skip scaffold entries whose destination is under `.claude/` when a single shared `agentAvailable("claude")` result is false); the gate MUST be computed once per sync, not re-derived divergently in two places.

- **GIVEN** a machine without `claude` (and `FAB_AGENTS` unset or lacking `claude`)
- **WHEN** `fab sync` runs in a fresh checkout
- **THEN** no `.claude/` directory is created at all (neither skills nor settings.local.json)
- **AND** with `claude` present, settings.local.json is written exactly as today

### Verification: Tests

#### R5: Test suite pins the re-tiered contract
`src/go/fab-kit/internal/skills_test.go` + `sync_integration_test.go` SHALL be updated (Constitution: Go changes ship tests; tests conform to spec):

- `TestSync_FullRunProducesExpectedTree` (runs under `FAB_AGENTS=claude`): expects `.claude/skills/` AND `.agents/skills/` (each with manifest) AND `.claude/settings.local.json` present; `.opencode/commands/` absent.
- New/adjusted integration case with `FAB_AGENTS=codex` (lacking `claude`): asserts `.claude/` absent entirely while `.agents/skills/` is fully deployed.
- `TestDeploySkills_*` gate/table tests updated for the re-gated `.claude` row; the brand-dir guard (`.gemini`/`.agy`/`.kimi`/`.codex` never created) retained.
- Re-verify `TestSync_SecondRunIsContentIdenticalNoop`, `TestSync_ManifestLifecycle`, scoping tests, `TestDeploySkills_PropagatesAgentFailure`.

- **GIVEN** the updated package
- **WHEN** the scoped `src/go/fab-kit/internal` suite runs
- **THEN** all tests pass with the new assertions

### Docs: Three-Class Sweep

#### R6: Present-truth docs updated under the three-class discipline
Every documentation claim SHALL be handled by its sweep class:

| Class | Rule |
|-------|------|
| (a) Read-path instructions | Repoint to `.agents/skills/` |
| (b) Deploy-target descriptions | Stay `.claude`, wording updated to the gated contract ("when `claude` is present") |
| (c) Historical artifacts | NEVER touched: `src/kit/migrations/2.22.0-to-2.23.0.md`, all `docs/memory/**/log.md` + `log.seed.md`, `docs/specs/findings/*`, `docs/findings/*` (dated research analyses citing historical line numbers) |

Files: `docs/memory/distribution/kit-architecture.md` (§ Agent Skill Deployment + § Design Decisions — RE-SCOPE the #647 always-on-pair decision, don't delete it; record the reversal rationale: unconditional = cross-client standard, gated = brand surfaces), `distribution.md`, `setup.md` (deployment row), `docs/specs/architecture.md` (target table), `docs/specs/overview.md` + `skills.md` (read-path mentions), `docs/memory/_shared/context-loading.md`, `docs/memory/runtime/agent-primitives.md`, `docs/memory/memory-docs/templates.md`, `docs/memory/pipeline/planning-skills.md` + `execution-skills.md`, `fab/project/constitution.md` Principle V + canonical-source constraint and `fab/project/context.md` (wording-only, NO version bump, dated HTML comment per jjg0/udwv precedent), `docs/site/skill.md` + `install.md` (verify-first; any docs/site edit checked against the live shll standards — re-fetch, standing lesson).

- **GIVEN** the sweep is done
- **WHEN** grepping the repo for `.claude/skills`
- **THEN** remaining hits are exactly: class-(b) deploy-target descriptions with gated wording, class-(c) historical artifacts, and Go source/test/scaffold literals that legitimately name the deploy target

### Non-Goals

- Retiring the `.claude/skills/` target (still blocked on Claude Code shipping `.agents/skills/` support; this change is the stepping stone only)
- Changing `.opencode/commands/` gating, format, or location
- Any migration file (verified: nothing existing is restructured; sync's skip path preserves existing dirs)
- Touching deployed-copy directories directly (regenerated by sync)

### Design Decisions

#### Single shared gate result for deploy row and scaffold write
**Decision**: Compute `claude` availability once per sync and feed both the `.claude` deploy row and the scaffold `.claude/`-path skip from that one result.
**Why**: Two independent derivations of the same gate is the drift mechanism — an env-set difference or ordering change would create a half-present `.claude/` (settings without skills or vice versa).
**Rejected**: Independent `agentAvailable("claude")` calls at each site — currently equivalent, but nothing keeps them equivalent.
*Introduced by*: 260908-yd9s-repoint-agents-skills-gate-claude

#### Repoint targets the deployed workspace copy, not the kit cache
**Decision**: References point at `.agents/skills/{name}/SKILL.md` (workspace-relative), never `$(fab kit-path)/skills/`.
**Why**: The deployed copy is the workspace contract, version-pinned per project and visible to every harness; kit-cache paths are version-resolved and machine-specific.
**Rejected**: Kit-cache paths — break the _preamble Path Convention (repo-root-relative) and version-skew silently.
*Introduced by*: 260908-yd9s-repoint-agents-skills-gate-claude

## Tasks

### Phase 2: Core Implementation

- [x] T001 Re-gate the "Claude Code" row in `src/go/fab-kit/internal/skills.go` (:53-56): drop `AlwaysOn: true`, restore `CLIs: []string{"claude"}`; update the table doc comment for the new tiering <!-- R3 -->
- [x] T002 Gate the `.claude/settings.local.json` scaffold write in `src/go/fab-kit/internal/scaffold.go` (+ `sync.go` call site ~:98-100): skip scaffold entries destined under `.claude/` when the shared `claude`-availability result is false; compute the gate once per sync <!-- R4 -->
- [x] T003 Repoint all 37 `.claude/skills` read-path references across the 23 `src/kit/skills/*.md` files to `.agents/skills` — including `_preamble.md`'s quoted canonical boilerplate sentence; verify with `grep -rn '\.claude/skills' src/kit/skills/` → zero hits <!-- R1, R2 -->
- [x] T004 Update `src/go/fab-kit/internal/skills_test.go`: re-gated `.claude` row expectations, keep the brand-dir guard, gated/always-on split coverage <!-- R5 -->
- [x] T005 Update `src/go/fab-kit/internal/sync_integration_test.go`: `TestSync_FullRunProducesExpectedTree` (`FAB_AGENTS=claude`) expects `.claude/skills/` + `.agents/skills/` + `.claude/settings.local.json`; add/adjust a `FAB_AGENTS=codex` case asserting `.claude/` absent entirely, `.agents/skills/` full; re-verify noop/manifest/scoping tests <!-- R4, R5 -->
- [x] T006 Run the scoped test suite (`src/go/fab-kit/internal`, widen via `just test` if fallout appears); gofmt touched files; fix failures <!-- R5 -->

### Phase 3: Documentation Sweep

- [x] T007 Update `docs/memory/distribution/kit-architecture.md` (§ Agent Skill Deployment + re-scope the #647 Design Decision with reversal rationale), `distribution.md`, `setup.md` deployment row <!-- R6 -->
- [x] T008 [P] Update `docs/specs/architecture.md` target table and `docs/specs/overview.md` + `skills.md` read-path mentions <!-- R6 -->
- [x] T009 [P] Repoint read-path references in `docs/memory/_shared/context-loading.md`, `docs/memory/runtime/agent-primitives.md`, `docs/memory/memory-docs/templates.md`, `docs/memory/pipeline/planning-skills.md`, `docs/memory/pipeline/execution-skills.md` <!-- R6 -->
- [x] T010 [P] Constitution Principle V + canonical-source constraint and `context.md` (wording-only, dated HTML comment, no version bump); verify `docs/site/skill.md` + `install.md` against live shll standards (update only stale claims) <!-- R6 -->
- [x] T011 Final three-class sweep: `grep -rn '\.claude/skills'` repo-wide; classify every remaining hit as legitimate class-(b)/(c)/Go-literal; fix any residual class-(a) read path, including `*_test.go` comments and user-facing string literals <!-- R1, R6 -->

## Execution Order

- T001 → T002 (shared gate); T003 independent of Go tasks; T004/T005 after T001-T002; T006 after T004-T005
- T007-T010 after T001-T003 fix the behavior; T011 last (sweeps final state)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `grep -rn '\.claude/skills' src/kit/skills/` returns zero hits; all 23 files reference `.agents/skills/…`
- [x] A-002 R3: With `FAB_AGENTS=codex`, sync deploys `.agents/skills/` fully and skips `.claude/skills/` with the skip message; with `claude` present, `.claude/skills/` deploys as before
- [x] A-003 R4: With no `claude`, a fresh sync creates NO `.claude/` directory at all; with `claude`, `settings.local.json` is written exactly as today

### Behavioral Correctness

- [x] A-004 R4: The claude gate is computed once per sync and shared by the deploy row and the scaffold skip (no second independent derivation)
- [x] A-005 R3: `.agents/skills/` row remains `AlwaysOn` and `.opencode/commands/` gating is untouched (no diff beyond the `.claude` row in the table)

### Scenario Coverage

- [x] A-006 R5: `TestSync_FullRunProducesExpectedTree` passes with the `.claude`+`.agents`+settings expectations; the `FAB_AGENTS=codex` case passes asserting `.claude/` wholly absent
- [x] A-007 R5: Scoped `src/go/fab-kit/internal` suite green; brand-dir guard retained
- [x] A-008 R2: Spot-check — a repointed reference (e.g. `_preamble.md` helper-load path) resolves to an existing file in a synced checkout

### Edge Cases & Error Handling

- [x] A-009 R3: A pre-existing `.claude/` tree on a claude-less machine survives sync untouched (skip path preserves, never prunes)

### Documentation Accuracy

- [x] A-010 R6: kit-architecture.md's #647 Design Decision is re-scoped (not deleted) and records the reversal rationale; distribution.md/setup.md/specs match the new tiering
- [x] A-011 R6: Class-(c) files verifiably untouched (`git diff --name-only` contains no `log.md`, `log.seed.md`, `findings/`, or `migrations/` paths); constitution/context changes are wording-only with a dated HTML comment and no version bump
- [x] A-012 R6: Repo-wide `.claude/skills` grep residue is exactly class-(b) gated-wording descriptions, class-(c) historical artifacts, and legitimate Go/scaffold literals — the reviewer's classification-gap should-fix resolved post-verdict by naming `docs/findings/*` in class (c) (dated research analyses; their historical line-number citations stay verbatim, files untouched)

### Code Quality

- [x] A-013 Pattern consistency: table-driven config and existing sync output style preserved; new gate code follows surrounding conventions
- [x] A-014 No unnecessary duplication: one shared gate result; no duplicated availability logic
- [x] A-015 Owner-or-pointer: skill-prose edits state an owned rule or point at the owner, never both
- [x] A-016 CLI ⇒ docs + tests: no `fab` command signature changes expected; `_cli-fab.md` checked and updated only if sync output it documents changed

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change re-gates existing deploy rows, shares one precomputed availability result, and repoints references without making any existing file, function, branch, or config redundant. (The stale `.claude/skills/` cleanup path a user might want is intentionally NOT code: sync's skip path preserves existing dirs and the generated per-target `.gitignore` manifest enables safe manual deletion — a deliberate non-goal, not a missed deletion.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Scaffold gate mechanism: path-prefix skip of `.claude/`-destined entries in `scaffoldTreeWalk`, keyed on one precomputed shared `claude`-availability result (helper vs bool = implementer's micro-choice) | Intake row 4 delegated the mechanism; the plan pins the front-runner and the single-derivation constraint (drift prevention) | S:65 R:85 A:85 D:60 |
| 2 | Confident | Sweep residue contract for A-012: Go source/test/scaffold literals naming `.claude` deploy paths are legitimate residue (they ARE the deploy target), alongside class-(b)/(c) | Follows directly from the three-class discipline; prevents the reviewer over-flagging deploy-target literals | S:70 R:80 A:85 D:70 |

2 assumptions (0 certain, 2 confident, 0 tentative).
