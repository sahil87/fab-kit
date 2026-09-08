# Plan: Unconditional Agent Skill Deploy Targets

**Change**: 260908-t513-unconditional-agent-skill-deploy-targets
**Intake**: `intake.md`

## Requirements

### Sync: Always-On Deploy Targets

#### R1: Directory targets deploy unconditionally
`fab-kit sync`'s skill deployment (`src/go/fab-kit/internal/skills.go`, target table at `:52-56`) SHALL deploy the "Claude Code" (`.claude/skills/`, directory format) and "Agents dir" (`.agents/skills/`, directory format) targets on every sync, with no CLI-on-PATH detection. The "OpenCode" (`.opencode/commands/`, flat format) target SHALL remain gated on `opencode` being available.

- **GIVEN** a fab project on a machine with NO agent CLIs on PATH (no `claude`, `codex`, `agy`, `kimi`, `opencode`)
- **WHEN** `fab sync` runs
- **THEN** `.claude/skills/` and `.agents/skills/` are deployed in full (all kit skills + generated `.gitignore` manifest each)
- **AND** `.opencode/commands/` is not created, with the existing `Skipping OpenCode: …` message printed

#### R2: Always-on rows bypass `agentAvailable`; `FAB_AGENTS` governs only gated rows
An always-on target SHALL NOT consult `agentAvailable` (`skills.go:116`) at all — neither PATH lookup nor the `FAB_AGENTS` override can suppress it. `FAB_AGENTS` retains its existing semantics for gated rows only.

- **GIVEN** `FAB_AGENTS=opencode` (or `FAB_AGENTS=claude`, or unset)
- **WHEN** `deploySkills` runs
- **THEN** `.claude/skills/` and `.agents/skills/` deploy regardless of the variable's value
- **AND** the OpenCode row fires iff `opencode` is in `FAB_AGENTS` (when set) or on PATH (when unset)

#### R3: Dead detection output removed
The `Warning: No agent CLIs found in PATH…` branch (`skills.go:85-87`) SHALL be removed — with two targets always firing it is unreachable (delete the `agentsFound` counter too if it has no other reader). The `Skipping {Label}: {missingCLIs}` message SHALL keep firing for gated rows only, and `missingCLIs` (`skills.go:142`) is never consulted for an always-on row. The target-table doc comment (`skills.go:47-51`) SHALL be updated: the "deploys once when any of them is installed" claim now applies only to the gated OpenCode row.

- **GIVEN** the post-change `deploySkills`
- **WHEN** no CLI is on PATH and `FAB_AGENTS` is unset
- **THEN** output shows both always-on targets deploying, one `Skipping OpenCode: opencode not found in PATH` line, and no "No agent CLIs found" warning

#### R4: Manifest machinery untouched
The per-target `.gitignore` manifest machinery (`skills.go:275-357`: read-manifest → deploy → scoped prune → write-manifest) SHALL NOT change. New target dirs self-bootstrap via their manifest on the next sync; no migration file ships.

- **GIVEN** an existing checkout that previously synced only `.claude/skills/` (e.g. only `claude` was on PATH)
- **WHEN** the new binary runs `fab sync`
- **THEN** `.agents/skills/` appears with its own generated manifest, `.claude/skills/` is refreshed via its existing manifest, and no user-owned files are pruned

### Verification: Tests

#### R5: Test suite asserts the unconditional tree
The `src/go/fab-kit/internal` tests SHALL be updated so the suite passes against the new behavior and pins it (Constitution: Go changes ship tests; tests conform to spec):

- `sync_integration_test.go` `TestSync_FullRunProducesExpectedTree` (`:201`, runs under `FAB_AGENTS=claude`): now expects `.claude/skills/` AND `.agents/skills/` present each with a manifest, `.opencode/commands/` still absent.
- `skills_test.go` `TestDeploySkills_GenericDirForNonCodexCLI` (`:83`): keep its guard that `.gemini`/`.agy`/`.kimi`/`.codex` brand dirs are never created; update expectations for the always-on pair.
- A test SHALL exist proving always-on deployment with zero CLIs available (empty/foreign `FAB_AGENTS`), and `TestAgentAvailable_*`/`TestMissingCLIs` keep covering the gated path (function semantics unchanged).
- Re-verify `TestSync_SecondRunIsContentIdenticalNoop` (`:290`), `TestSync_ManifestLifecycle` (`:356`), `TestSync_ProjectOnlyRunsOnlyProjectScripts` (`:417`), `TestSync_ShimOnlySkipsProjectScripts` (`:437`), `TestDeploySkills_PropagatesAgentFailure` (`:459`).

- **GIVEN** the updated package
- **WHEN** `go test ./...` runs scoped to `src/go/fab-kit/internal`
- **THEN** all tests pass with the new assertions

### Docs: Detection Claims Sweep

#### R6: All conditional-deployment claims updated
Every documentation claim that skill deployment is detection-gated SHALL be updated to the new contract (two always-on directory targets + one gated flat target):

- `docs/memory/distribution/kit-architecture.md` § Agent Skill Deployment (~`:104-130`) and § Design Decisions — the "Conditional Multi-Candidate Deployment" entry's decision/why/rejected content is now partially inverted (its "Rejected: Unconditional deployment to all agents" must flip to record the reversal and why: determinism under Constitution III + the `.agents/skills/` cross-client convention; the one-target-per-skill-set and manifest decisions stand).
- `docs/memory/distribution/distribution.md` deployed-target enumerations (~`:194`, `:213`) — verify wording; the target list itself is unchanged.
- `docs/specs/architecture.md` detection prose (~`:489`), target table (~`:490-497` incl. `:497` "deploys once when any of them is on PATH"), and `:539` "deploys skills to detected agents".
- `docs/site/skill.md` (~`:111`), `docs/site/install.md` (~`:118-123`) — verify; likely additive or unchanged.
- `src/kit/skills/_cli-agents.md` (`:209`, `:231`) — verify the "deploys one skill set to `.agents/skills/`" rationale phrasing survives.
- `fab/project/constitution.md` Principle V / `fab/project/context.md` — at most additive wording (mention `.agents/skills/` alongside `.claude/skills/` as deployed copies), wording-only, NO version bump (jjg0/udwv precedent).
- Repo-wide sweep greps before finishing: `Skipping `, `No agent CLIs`, `detected agent`, `conditional`, `on PATH`, `any of them`, including `*_test.go` comments and user-facing string literals.

- **GIVEN** the docs sweep is done
- **WHEN** grepping the repo for the swept phrases
- **THEN** no remaining claim states that `.claude/skills/` or `.agents/skills/` deployment depends on CLI detection

### Non-Goals

- Retiring the `.claude/skills/` target (blocked: Claude Code does not scan `.agents/skills/`; recorded as a follow-up idea, not this change)
- Changing the `.opencode/commands/` target's format, location, or gate
- Any change to manifest read/prune/write logic, scaffolding, direnv, or the router
- A migration file (nothing existing is restructured)

### Design Decisions

#### Explicit `AlwaysOn` field over empty-`CLIs` sentinel
**Decision**: Add `AlwaysOn bool` to `agentConfig` and check it at the detection gate (`if !agent.AlwaysOn && !agentAvailable(agent.CLIs...)`); always-on rows keep their `CLIs` list purely as documentation-free dead weight removed (drop `CLIs` from always-on rows entirely).
**Why**: Explicit beats implicit — an empty-slice sentinel overloads "no candidates" with "always deploy", which reads as a bug to a future editor; a named field self-documents at the table, the single place targets are declared.
**Rejected**: Treating empty `CLIs` as always-available inside `agentAvailable` — smaller diff but hides deploy policy inside a PATH-lookup helper and silently changes that helper's contract for all callers.
*Introduced by*: 260908-t513-unconditional-agent-skill-deploy-targets

#### `FAB_AGENTS` governs gated rows only
**Decision**: Always-on rows never consult `agentAvailable`, so `FAB_AGENTS` cannot suppress them; tests assert the unconditional tree instead of isolating targets.
**Why**: Purest reading of "unconditional" — an env escape would reintroduce per-environment tree variance, the exact failure mode this change removes. Test isolation is not needed: assertions simply include both always-on targets.
**Rejected**: Letting `FAB_AGENTS` suppress always-on rows as a test seam — preserves old test shapes but makes "unconditional" false in CI, the environment where determinism matters most.
*Introduced by*: 260908-t513-unconditional-agent-skill-deploy-targets

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `AlwaysOn bool` to `agentConfig`, mark the "Claude Code" and "Agents dir" rows always-on (dropping their `CLIs` lists), gate the skip check on `!agent.AlwaysOn`, and update the table doc comment in `src/go/fab-kit/internal/skills.go` (`:28-34`, `:47-56`, `:60-64`) <!-- R1, R2 -->
- [x] T002 Remove the unreachable `No agent CLIs found in PATH` warning branch and the `agentsFound` counter (if unreferenced) in `src/go/fab-kit/internal/skills.go` (`:82`, `:85-87`) <!-- R3 -->
- [x] T003 Update `src/go/fab-kit/internal/skills_test.go`: adjust `TestDeploySkills_GenericDirForNonCodexCLI` and `TestDeploySkills_PropagatesAgentFailure` for the always-on pair, add an always-on-with-zero-CLIs deployment test, keep `TestAgentAvailable_*`/`TestMissingCLIs` covering the gated path <!-- R5 -->
- [x] T004 Update `src/go/fab-kit/internal/sync_integration_test.go`: flip `TestSync_FullRunProducesExpectedTree` to expect `.claude/skills/` + `.agents/skills/` manifests and `.opencode/commands/` absent; re-verify `TestSync_SecondRunIsContentIdenticalNoop`, `TestSync_ManifestLifecycle`, and the two scoping tests <!-- R5 -->
- [x] T005 Run the scoped test suite for `src/go/fab-kit/internal` (widen to `src/go/...` if cross-cutting fallout appears) and fix failures <!-- R5 -->

### Phase 3: Documentation Sweep

- [x] T006 Update `docs/memory/distribution/kit-architecture.md` § Agent Skill Deployment and the "Conditional Multi-Candidate Deployment" Design Decision (record the reversal: two always-on targets + one gated; keep one-target-per-skill-set and manifest decisions) <!-- R6 -->
- [x] T007 [P] Verify/update `docs/memory/distribution/distribution.md` deployed-target wording (~`:194`, `:213`) <!-- R6 -->
- [x] T008 [P] Update `docs/specs/architecture.md` detection prose and target table (~`:460-497`, `:539`) <!-- R6 -->
- [x] T009 [P] Verify (update only if stale) `docs/site/skill.md` ~`:111`, `docs/site/install.md` ~`:118-123`, `src/kit/skills/_cli-agents.md` `:209`/`:231`, and constitution Principle V / context.md wording (additive only, no version bump) <!-- R6 -->
- [x] T010 Repo-wide sweep greps (`Skipping `, `No agent CLIs`, `detected agent`, `conditional`, `any of them is`, `on PATH`) covering `*_test.go` comments and user-facing string literals; fix every stale claim in the sweep class <!-- R6 --> <!-- rework: review found docs/memory/distribution/setup.md:146 still claims deployment is "conditional on agent CLI availability" — sweep missed setup.md -->

## Execution Order

- T001 → T002 (same file, warning depends on gate shape); T003/T004 after T001-T002; T005 after T003-T004
- T006-T010 independent of code tasks once behavior is fixed; T010 last (sweeps the final state)

## Acceptance

### Functional Completeness

- [x] A-001 R1: With zero agent CLIs on PATH and `FAB_AGENTS` unset, `fab sync` deploys all kit skills to `.claude/skills/` and `.agents/skills/` (each with a generated `.gitignore` manifest) and skips `.opencode/commands/` with the skip message
- [x] A-002 R2: `FAB_AGENTS` value (any, including empty or foreign names) cannot suppress the two always-on targets; the OpenCode row still honors it
- [x] A-003 R3: The `No agent CLIs found in PATH` warning string no longer exists in `src/go/`; the skip message fires only for gated rows

### Behavioral Correctness

- [x] A-004 R1: `TestSync_FullRunProducesExpectedTree` (still under `FAB_AGENTS=claude`) passes asserting `.claude/skills/.gitignore` AND `.agents/skills/.gitignore` exist and `.opencode/commands/` does not
- [x] A-005 R4: No diff under the manifest helpers (`manifestEntry`/`parseManifestEntry`/`readSkillManifest`/`writeSkillManifest`/`cleanStaleSkills`) and no migration file added

### Scenario Coverage

- [x] A-006 R5: A test exercises always-on deployment with no CLIs available; the scoped `src/go/fab-kit/internal` suite is green
- [x] A-007 R5: `TestDeploySkills_GenericDirForNonCodexCLI`'s brand-dir guard (`.gemini`/`.agy`/`.kimi`/`.codex` never created) is retained and passing

### Edge Cases & Error Handling

- [x] A-008 R4: A pre-existing target dir with user-owned (non-manifest) entries survives the first always-on sync untouched (no-manifest-prunes-nothing rule still holds)

### Documentation Accuracy

- [x] A-009 R6: kit-architecture.md, distribution.md, specs/architecture.md carry no remaining claim that `.claude/skills/` or `.agents/skills/` deployment is detection-gated; the Design Decision records the reversal with rationale
- [x] A-010 R6: Sweep greps return zero stale occurrences (docs, site, kit skills, test comments, string literals); constitution/context wording at most additive with no version bump

### Code Quality

- [x] A-011 Pattern consistency: New code follows naming and structural patterns of surrounding code (table-driven config, existing output style)
- [x] A-012 No unnecessary duplication: No new helper duplicates existing utilities; gate logic stays single-sourced at the loop
- [x] A-013 **N/A**: No `src/kit/skills/*.md` prose was edited, so the owner-or-pointer rule has no changed-file application
- [x] A-014 CLI ⇒ docs + tests: `_cli-fab.md` checked — updated only if `fab sync` output it documents changed

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `agentAvailable` multi-candidate behavior (`src/go/fab-kit/internal/skills.go:108-125`) and `TestAgentAvailable_AnyCandidate` — only the single-candidate OpenCode row remains gated, so the variadic any-match path has no production caller
- `missingCLIs` plural branch (`src/go/fab-kit/internal/skills.go:135`) and its multi-candidate assertions — no shipped gated target now supplies more than one CLI, so this branch is retained only for possible future rows

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Always-on mechanism = explicit `AlwaysOn bool` field (not empty-`CLIs` sentinel) | Intake left it implementer's choice preferring the clearer diff; a named field self-documents and keeps `agentAvailable`'s contract untouched | S:75 R:85 A:80 D:65 |
| 2 | Confident | `FAB_AGENTS` governs gated rows only; always-on rows bypass `agentAvailable` entirely (resolves intake's deferred row per its recorded recommendation) | Purest "unconditional"; env-suppressible always-on would reintroduce per-environment variance; tests assert the unconditional tree | S:70 R:75 A:75 D:60 |
| 3 | Confident | The unreachable warning branch is removed (not reworded) | Skip message already reports the gated row; rewording keeps dead ceremony | S:65 R:90 A:80 D:60 |

3 assumptions (0 certain, 3 confident, 0 tentative).
