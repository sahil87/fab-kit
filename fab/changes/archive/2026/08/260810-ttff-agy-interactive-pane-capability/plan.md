# Plan: agy-interactive-pane-capability

**Change**: 260810-ttff-agy-interactive-pane-capability
**Intake**: `intake.md`

## Requirements

### Agent Defaults: agy Interactive Grammar

#### R1: Ship agy as a pane-capable built-in
The embedded built-in provider table MUST define `providers.agy.interactive_command` exactly as `agy --dangerously-skip-permissions --model {model}`. The agent package SHALL expose the value through a `DefaultAgyInteractiveCommand` sibling of the other per-provider command exports, while preserving agy's existing headless grammar, model-only fills, and lack of native capability.

- **GIVEN** a project with no `providers:` override
- **WHEN** the built-in `agy` provider is resolved
- **THEN** both its interactive and headless command grammars are present
- **AND** the interactive grammar is the exact shipped value, carries the full-auto flag, and contains no `{effort}` placeholder

### Runtime and Reference Surfaces: Pane Eligibility

#### R2: Pin and render agy's pane capability
Tests MUST pin agy's interactive command by value and exercise the built-in as pane-capable at the agent, pane-open, dispatch-mode, and rendered-config seams. `fab config explain` SHALL render agy's `interactive_command` from the agent package export before its `headless_command`, without duplicating the command literal outside `defaults.yaml` and the deliberate value pin.

- **GIVEN** a fresh project that names the built-in `agy` provider
- **WHEN** `fab agent --provider agy --print`, provider-generic pane composition, dispatch mode composition, or `fab config explain` is exercised
- **THEN** each surface uses the shipped interactive grammar rather than returning the former missing-capability error or describing agy as headless-only

### Documentation Model: Interactive by Default

#### R3: Reframe provider capability around pane-by-default
Canonical skills, their SPEC mirrors, aggregate specifications, runtime memory, and rendered-reference prose MUST state the current model: every supported agent CLI has an interactive mode; built-in providers are expected to ship pane launch grammar backed by the generic readiness gate; headless grammar is the capability that varies and requires provider probing. The agy entry SHALL document its first-run trust wall as an ordinary bounded readiness-gate judgment round and note that users may additionally seed the exact-match trust store.

An absent `interactive_command` MUST remain a valid user-defined provider configuration: automatic selection skips that rung and forced interactive launch errors actionably. It MUST NOT be presented as a state shipped by fab-kit's built-in provider roster.

- **GIVEN** the four built-in providers and a separate user-defined provider with only `headless_command`
- **WHEN** a reader consults the provider docs or the runtime selects an adapter
- **THEN** all four built-ins are documented as pane-capable
- **AND** the user-defined provider still descends from pane or reports the existing config-key hint when pane is forced

### Verification: Quota-Constrained Follow-up

#### R4: Verify without a live agy pane and preserve the follow-up
This change MUST be verified with exact-value pins, rendered fixtures, and affected Go package tests. A live agy pane run MUST be deferred while the provider quota is exhausted, and the existing `[agik]` backlog entry SHALL record the residual live probe after the approximately 2026-08-16 reset so the unrun check is not lost.

- **GIVEN** agy quota exhaustion through approximately 2026-08-16
- **WHEN** apply verification completes
- **THEN** automated tests and fixtures are green without invoking a live agy worker
- **AND** the backlog names the remaining live trust-wall and delivery probe

### Non-Goals

- Add agy-specific readiness-gate code or trust-store mutation — the provider-neutral gate already owns first-run judgment, and seeding remains a user-side optimization.
- Change `dispatch.mode`, descent diagnostics, or the behavior of user-defined providers that omit `interactive_command`.
- Change agy's headless prompt-delivery grammar, role fills, or effort model.

### Design Decisions

#### Pane capability follows launch grammar plus the generic gate
**Decision**: Ship pane launch grammar for every built-in provider and handle agy's first-run trust prompt through the existing readiness-gate judgment rounds.
**Why**: Interactive operation is common to agent CLIs; the provider-specific uncertainty is headless prompt grammar, while the gate already mechanizes boot and first-run walls without provider branches.
**Rejected**: Withholding `interactive_command` for any first-run wall, or adding agy-specific trust-store code to the provider-neutral runtime.
*Introduced by*: 260810-ttff-agy-interactive-pane-capability

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add the exact agy `interactive_command` to `src/go/fab/internal/agent/defaults.yaml`, export it from `src/go/fab/internal/agent/agent.go`, and update exact-value/capability pins in `src/go/fab/internal/agent/defaults_test.go` and `src/go/fab/internal/agent/agent_test.go`. <!-- R1 -->
- [x] T002 Update `src/go/fab/internal/configref/configref.go` to render and explain agy's interactive grammar, and update `src/go/fab/cmd/fab/config_test.go` to pin the rendered shape and prose. <!-- R2 -->
- [x] T003 Update the built-in agy behavior cases in `src/go/fab/cmd/fab/agent_test.go`, `src/go/fab/cmd/fab/pane_open_test.go`, and `src/go/fab/cmd/fab/dispatch_start_test.go`, retaining missing-interactive coverage through a user-defined provider. <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T004 [P] Reframe the canonical references in `src/kit/skills/_cli-agents.md` and `src/kit/skills/_cli-fab.md`, and update their required mirrors `docs/specs/skills/SPEC-_cli-agents.md` and `docs/specs/skills/SPEC-_cli-fab.md`. <!-- R3 -->
- [x] T005 [P] Sweep aggregate specs and present-truth memory in `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `docs/specs/config.md`, `docs/specs/glossary.md`, `docs/memory/_shared/configuration.md`, `docs/memory/runtime/providers-and-profiles.md`, `docs/memory/runtime/dispatch.md`, and `docs/memory/runtime/agent-primitives.md`. <!-- R3 -->
- [x] T006 Rewrite the existing `[agik]` entry in `fab/backlog.md` to preserve only the quota-blocked live agy pane follow-up and its post-reset verification scope. <!-- R4 -->

### Phase 4: Polish

- [x] T007 Run `gofmt` on every touched Go file, run scoped Go tests for `src/go/fab/internal/agent`, `src/go/fab/internal/configref`, and `src/go/fab/cmd/fab`, widen verification as warranted, and re-run the stale-claim/mirror sweep. <!-- R4 -->

## Execution Order

- T001 blocks T002 and T003 because both consume the new exported/default command.
- T004 and T005 may proceed alongside T002/T003 after T001; T006 is independent.
- T007 runs after every implementation and documentation task.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `providers.agy.interactive_command` and `DefaultAgyInteractiveCommand` expose the exact command `agy --dangerously-skip-permissions --model {model}` while agy's headless grammar, model-only fills, and non-native posture remain unchanged.
- [x] A-002 R2: Agent, pane, dispatch, and rendered-config surfaces treat built-in agy as interactive/pane-capable and pin its grammar by value.
- [x] A-003 R3: Canonical skills, their SPEC mirrors, aggregate specs, and runtime memory consistently describe pane launch grammar plus the generic gate as the built-in expectation and headless grammar as the provider-varying capability.
- [x] A-004 R4: Automated verification is green and `[agik]` records the quota-blocked live pane probe after the reset window.

### Behavioral Correctness

- [x] A-005 R1: Resolving the built-in agy provider with no project override returns both command forms and no separate effort grammar.
- [x] A-006 R2: `fab config explain` renders agy's interactive command before its headless command from the canonical export, and `fab agent --provider agy --print` composes a model-free full-auto session when no model flag is supplied.
- [x] A-007 R3: No current-truth source claims agy or another built-in is dispatch-only or lacks `interactive_command`; generic missing-capability diagnostics remain correct for user configuration.

### Scenario Coverage

- [x] A-008 R1: Internal agent tests cover exact-value wiring, full-auto grammar, pane eligibility, and the absence of `{effort}` from both agy commands.
- [x] A-009 R2: Command/config tests cover built-in agy session composition, pane composition, dispatch-mode composition, and reference rendering, while a user-defined headless-only provider covers the missing-interactive error.
- [x] A-010 R3: Repository-wide `dispatch-only` / `no interactive_command` / `agik` review finds no stale built-in-agy framing in source, skills, SPEC mirrors, aggregate docs, or present-truth memory.

### Edge Cases & Error Handling

- [x] A-011 R3: A user-defined provider with no `interactive_command` still soft-descends in automatic mode and errors with the existing `providers.<name>.interactive_command` hint when interactive launch is explicit.
- [x] A-012 R4: Verification does not invoke the quota-exhausted agy CLI and clearly preserves the deferred live check.

### Code Quality

- [x] A-013 Readability and maintainability: Comments and tests describe the current capability model without transition narration or stale backlog ownership claims.
- [x] A-014 Pattern consistency: The new export and exact-value pin follow the existing codex/kimi sibling patterns.
- [x] A-015 No unnecessary duplication: Production/rendered prose sources interpolate `DefaultAgyInteractiveCommand`; the exact literal is duplicated only in the deliberate test pin.
- [x] A-016 Sibling and mirror sweep: Both touched canonical skills have synchronized `SPEC-*` mirrors, and aggregate specs/memory carrying the same roster claim are updated.
- [x] A-017 Test alongside and formatting: Relevant Go tests pass and every touched `.go` file is `gofmt`-clean.

## Notes

- Live agy pane verification is intentionally deferred until the provider quota resets around 2026-08-16. The `[agik]` backlog entry is the follow-up owner.
- Review should distinguish stale built-in roster claims from valid generic diagnostics such as `pane unavailable: no interactive_command` for user-defined providers.

## Deletion Candidates

- None — this change adds one provider capability and updates its documentation/tests without making an existing file, symbol, branch, or configuration redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Include `_cli-agents`, both touched skill SPEC mirrors, aggregate specs/glossary, and `_shared/configuration` in the stale-claim sweep | The project sibling/mirror rule requires the whole class up front, and repo-wide grep identified each as a current-truth restatement | S:95 R:90 A:95 D:95 |
| 2 | Confident | Rewrite the existing `[agik]` backlog entry as the residual live-probe owner instead of creating a new backlog ID | The intake names the stale `[agik]` probe ownership and requires a recorded follow-up; updating that entry preserves continuity with minimal churn | S:80 R:90 A:85 D:80 |
| 3 | Certain | Preserve generic missing-`interactive_command` diagnostics and headless-only user-provider examples while removing the claim from the built-in roster | The intake explicitly keeps absent `interactive_command` valid for user configuration; only the shipped state and mental model are changing | S:95 R:90 A:95 D:95 |

3 assumptions (2 certain, 1 confident, 0 tentative).
