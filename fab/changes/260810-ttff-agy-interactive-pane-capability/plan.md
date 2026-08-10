# Plan: agy-interactive-pane-capability

**Change**: 260810-ttff-agy-interactive-pane-capability
**Intake**: `intake.md`

## Requirements

### Runtime: Agy Pane Capability

#### R1: Ship `providers.agy.interactive_command`
`defaults.yaml` SHALL ship an `interactive_command` for the `agy` built-in provider, matching its headless full-auto posture (`--dangerously-skip-permissions`). The model ID embeds the reasoning level, so the command SHALL NOT carry an `{effort}` placeholder. 

- **GIVEN** a project configured to use the `agy` built-in provider
- **WHEN** `fab dispatch open` is called on a stage mapped to it
- **THEN** the worker opens as an interactive pane, since the provider carries an `interactive_command`
- **AND** the readiness gate answers the first-run trust wall as an ordinary parked dialog

#### R2: Sweep "dispatch-only" framing
The codebase and memory MUST NOT refer to `agy` as "dispatch-only" or claim it "carries no `interactive_command`" (or similar contrastive claims that single it out as lacking pane capability). This applies to Go comments, rendered documentation, specs, and memory files.

- **GIVEN** a search for "dispatch-only" or "no interactive_command" across the codebase
- **WHEN** reviewing the results
- **THEN** no results should falsely claim `agy` lacks interactive pane capability
- **AND** test assertions (e.g., in `defaults_test.go`) MUST pin the shipped command by value instead of asserting its absence

#### R3: Reframe Capability Model
Memory and specs SHALL reflect that pane capability is the default expectation for any provider (because it relies on generic launch grammar and the readiness gate). Headless grammar is what varies and needs probing. An absent `interactive_command` remains a valid user configuration but is no longer a shipped state for built-ins.

- **GIVEN** `docs/memory/runtime/providers-and-profiles.md` and `docs/specs/stage-models.md`
- **WHEN** reading about the built-in providers
- **THEN** all four built-ins are documented as pane-capable by default
- **AND** the open question per provider is correctly framed as "first-run behavior" rather than prompt grammar

#### R4: Provide Go Exports
`agent.go` SHALL export the default interactive command string for `agy` as a package-level variable (e.g., `DefaultAgyInteractiveCommand`), matching the pattern used for other providers.

- **GIVEN** `src/go/fab/internal/agent/agent.go`
- **WHEN** `internal/configref` renders the `agy` block
- **THEN** it interpolates the exported `DefaultAgyInteractiveCommand` variable

## Tasks

### Phase 1: Core Implementation

- [x] T001 Modify `src/go/fab/internal/agent/defaults.yaml` to add `interactive_command: 'agy --dangerously-skip-permissions --model {model}'` to the `agy` block and rewrite comments about the first-run trust wall. <!-- R1 -->
- [x] T002 Modify `src/go/fab/internal/agent/agent.go` to add `DefaultAgyInteractiveCommand` and interpolate it in `internal/configref/configref.go`. <!-- R4, R2 -->
- [x] T003 Modify `src/go/fab/internal/agent/defaults_test.go` to assert the exact value of `agy`'s `interactive_command` instead of asserting its absence. <!-- R2 -->

### Phase 2: Documentation & Memory Sweeps

- [x] T004 [P] Sweep and update `docs/specs/stage-models.md`. Rewrite the "One of the four is dispatch-only" section. <!-- R2, R3 -->
- [x] T005 [P] Sweep and update `docs/memory/runtime/providers-and-profiles.md`. Replace the "dispatch-only built-in" section with pane-by-default framing. <!-- R2, R3 -->
- [x] T006 [P] Sweep and update `docs/memory/runtime/dispatch.md` to rephrase the "no-interactive_command" example. <!-- R2, R3 -->
- [x] T007 [P] Sweep and update `docs/memory/runtime/agent-primitives.md` to update the `agy` grammar entry and remove the pane caveat. <!-- R2, R3 -->
- [x] T008 [P] Sweep `src/kit/skills/_cli-fab.md` and any other `.md` files for "dispatch-only", "no interactive_command", "agik", and similar phrases. Fix as needed. <!-- R2 --> <!-- rework: sweep missed stale DISPATCH-ONLY claims in Go test comments (config_test.go:715, agent_test.go:1048) -->

### Phase 3: Integration & Polish

- [x] T009 Run `cd src/go/fab && go test ./...` and update any failing tests (especially rendered-reference fixtures under `internal/configref`). <!-- R2 -->
- [x] T010 Run `cd src/go/fab && gofmt -l -w $(find . -name "*.go")` to format touched Go files. <!-- R1, R2, R4 -->

## Execution Order

- T001, T002, T003 must be completed before T009.
- T004 through T008 can be executed in parallel.
- T010 must be the final step for Go files.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` contains `interactive_command` for `agy`.
- [x] A-002 R4: `agent.go` exports `DefaultAgyInteractiveCommand` and it is used in `configref.go`.

### Behavioral Correctness

- [x] A-003 R2: `defaults_test.go` asserts the value of `agy`'s `interactive_command` rather than its absence.
- [x] A-004 R2: `go test ./...` passes, confirming rendered reference fixtures reflect the updated `interactive_command`.

### Removal Verification

- [x] A-005 R2: Grepping the codebase for "dispatch-only" in relation to `agy` returns no results.

### Code Quality

- [x] A-006 Pattern consistency: New exported variables and test assertions match the patterns used for `codex` and `kimi`.
- [x] A-007 Formatting: `gofmt -l` returns no unformatted Go files.
- [x] A-008 Sibling Sweeps: Any edits to `src/kit/skills/` are mirrored in `docs/specs/skills/` (if applicable).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship agy's `interactive_command` as built-in data in `defaults.yaml`, not user-config guidance | User-stated in discussion; identical shape to kimi's ki9v flip | S:90 R:85 A:90 D:90 |
| 2 | Certain | First-run trust wall is handled by the existing readiness-gate judgment rounds — no agy-specific code | Generic mechanism; kimi's identical wall is cleared this way in production today | S:85 R:80 A:90 D:85 |
| 3 | Confident | Command value `agy --dangerously-skip-permissions --model {model}` (no effort/prompt placeholders) | Mirrors agy's own shipped headless posture; effort rides the ID suffix; delivery is post-launch since 1lah. | S:70 R:85 A:75 D:70 |
| 4 | Confident | No trust-store seeding shipped — the gate suffices; seeding stays a user-side optimization noted in memory | Gate is the generic path and needs no per-provider file format knowledge | S:65 R:90 A:80 D:75 |
| 5 | Confident | Live agy pane verification deferred past quota reset; change verifies by tests + fixtures | Quota exhaustion is external; pin-by-value test covers the shipped grammar; follow-up recorded | S:75 R:80 A:70 D:70 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).

## Deletion Candidates

- `None` — this change adds new functionality without making existing code redundant
