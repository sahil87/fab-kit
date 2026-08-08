# Plan: dispatch.mode — the descent ladder + capability delink

**Change**: 260808-yilt-dispatch-mode-descent-ladder
**Intake**: `intake.md`

## Requirements

### Configuration: Capability Data and Dispatch Preference

#### R1: Provider Capabilities Are Explicit Data

Every provider MUST describe each supported launch capability independently through `session_command`, `dispatch_command`, and an explicit `native` boolean. Command-field presence MUST NOT select dispatch policy. The built-in claude provider SHALL ship `native: true` and a stdin-driven headless `dispatch_command`; codex and gemini SHALL retain their existing command/capability data and SHALL NOT gain native capability.

- **GIVEN** the built-in provider defaults
- **WHEN** claude is resolved
- **THEN** its interactive, native, and headless capabilities are all available as data
- **AND** the headless command uses `claude -p --dangerously-skip-permissions --model {model} --effort {effort}`
- **AND** codex and gemini remain non-native providers with their existing command strings

#### R2: `dispatch.mode` Is the Sole Persistent Mode Preference

The config schema MUST replace `dispatch.watchable` with `dispatch.mode`, accepting exactly `pane`, `native`, or `headless`. The default SHALL be the canonical constant `native`; the field SHALL be scope `both` and advertised in the reference/fence. An absent value SHALL resolve to `native`, while an invalid value MUST emit a `fab: warning:` diagnostic and fail open to `native`.

- **GIVEN** an absent, valid, or invalid `dispatch.mode`
- **WHEN** the nil-safe config accessor resolves it
- **THEN** absent returns `native`, each valid value returns verbatim, and invalid warns then returns `native`
- **AND** the config registry exposes `dispatch.mode` with default `native`, scope `both`, and `advertise: true`
- **AND** no binary accessor or registry row reads `dispatch.watchable`

### Runtime: One Descent Ladder at Both Dispatch Seams

#### R3: Automatic Selection Descends and Never Ascends

Automatic dispatch selection MUST begin at the configured preference and descend only through `pane → native → headless`, choosing the first possible rung. Pane requires tmux availability and `session_command`; native requires `native: true`; headless requires `dispatch_command`. A missing prerequisite SHALL skip its rung rather than fail, and selection MUST fail only when no rung is possible. Selection logic MUST be pure and shared by, or identically reused from, both resolver and launcher seams.

- **GIVEN** any preferred mode, provider capability combination, and tmux-presence input
- **WHEN** automatic mode selection runs
- **THEN** it returns the first possible rung at or below the preference
- **AND** it never returns a more interactive rung than the preference
- **AND** it reports a no-rung error when the provider has no reachable capability

#### R4: `fab resolve-agent` Emits from Resolved Mode

`fab resolve-agent` MUST derive `dispatch=` from `(dispatch.mode, provider capabilities, $TMUX)` rather than command presence. Native resolution SHALL omit the line; pane SHALL emit the profile-substituted `session_command`; headless SHALL emit the profile-substituted `dispatch_command`. A defined or depth-selected unknown provider with no possible rung MUST fail with an actionable error naming the provider and missing capabilities, while an explicit unknown `--provider` lookup SHALL retain its existing earlier error.

- **GIVEN** shipped claude, codex, and gemini providers under default `native`
- **WHEN** their worker roles resolve
- **THEN** claude omits `dispatch=` while codex/gemini emit their unchanged headless commands
- **AND** `mode: pane` resolves to pane inside tmux and descends outside tmux
- **AND** a provider with no capabilities returns a non-zero actionable error

#### R5: `fab dispatch start` and `restart` Re-Derive the Same Ladder

`fab dispatch start` and `restart` MUST use the same automatic descent ladder and current environment as the resolver seam. Existing explicit signals SHALL retain precedence and hard-error semantics: `--pane` and `--headless` remain mutually exclusive, `--timeout` implies headless, and `--server` implies pane. If automatic re-derivation lands on native, the CLI MUST fail actionably and tell the caller to re-run `fab resolve-agent`, because `fab dispatch` cannot launch the native Agent-tool adapter. Every successful automatic launch SHALL surface the selected rung and any descent reason in the existing `dispatched …` line; descent diagnostics SHALL go to stderr.

- **GIVEN** a configured preference, provider capabilities, explicit flags, and current tmux state
- **WHEN** `start` or `restart` selects a launch mode
- **THEN** explicit flags behave exactly as documented and automatic selection matches the resolver for the same inputs
- **AND** a tmux reachability change re-descends rather than substituting one command field for another
- **AND** automatic native resolution fails before any dispatch state is written
- **AND** successful automatic output names the selected rung and why any higher rung was skipped

### Distribution: Safe Upgrade from Watchable

#### R6: The Watchable Key Migrates Across Both Scopes

The kit MUST ship migration `2.17.2-to-2.18.0.md` and bump `src/kit/VERSION` to `2.18.0`. The migration SHALL atomically sweep both `fab/project/config.yaml` and `~/.fab-kit/config.yaml`: live `dispatch.watchable: true` becomes `mode: pane`, live `false` is removed, commented lines and the managed fence are untouched, and re-running is a no-op. The binary SHALL keep no read-time alias.

- **GIVEN** either config scope contains a live `dispatch.watchable` value
- **WHEN** the migration is applied
- **THEN** true becomes `dispatch.mode: pane`, false disappears, and unrelated bytes are preserved
- **AND** a second run changes nothing

#### R7: Reference Fences Reflect the New Schema

The registry-rendered `dispatch:` segment, config reference JSON, upgrade golden tests, and this repository's managed config fence MUST advertise `dispatch.mode` and the descent ladder while preserving adjacent `column_width` and `reap_done` behavior. The provider reference/default projection MUST include claude's `native` and `dispatch_command` capability data.

- **GIVEN** `fab config reference`, `fab config reference --json`, and `fab config upgrade`
- **WHEN** their outputs are rendered
- **THEN** they expose the new mode/provider capability schema without duplicate YAML parents
- **AND** registry coverage, key-parity, byte-stability, and idempotence guards pass

### Documentation: Canonical Mirror Sweep

#### R8: Kit Skills and Specs State the Ladder Consistently

Canonical files under `src/kit/skills/` and the complete `docs/specs/skills/SPEC-*.md` mirror class MUST replace watchable/presence-as-policy claims with the explicit mode/capability ladder. The affected aggregate specs (`harness-adapters.md`, `stage-models.md`, `config.md`, `architecture.md`, and `glossary.md`) MUST document the same behavior and exact automatic-selection reason strings. Deployed `.claude/skills/` and `docs/memory/` MUST remain untouched.

- **GIVEN** a repository-wide search for the retired doctrine and suffix strings
- **WHEN** the apply sweep completes
- **THEN** every in-scope canonical skill/spec occurrence states the new contract
- **AND** every touched canonical skill has its corresponding SPEC mirror updated
- **AND** historical migration text is changed only where it incorrectly claims current behavior

### Non-Goals

- Do not rename `session_command` or `dispatch_command`; Change 4 owns that sweep.
- Do not add environment-layer behavior, config mutation verbs, source consolidation, or kit-path overrides from Changes 1, 2, 5, or 6.
- Do not change `internal/configscope` behavior; only its stale dispatch-key comment may change because the existing top-level `dispatch` row already has scope `both`.
- Do not edit `.claude/skills/` or `docs/memory/`; canonical skills and pre-implementation specs are the apply-stage surfaces.

### Design Decisions

#### One Pure Capability Ladder

**Decision**: Model native as a third runtime mode and centralize automatic capability descent in `internal/dispatch`, with both `resolve-agent` and `dispatch start` consuming it.
**Why**: One pure table-testable function prevents the two adapter seams from disagreeing while leaving environment reads and provider-specific diagnostics in the cobra layer.
**Rejected**: Duplicating the matrix in `cmd/fab/resolve_agent.go` and `cmd/fab/dispatch_start.go`; the current watchable design already demonstrates how two selection seams drift.
*Introduced by*: 260808-yilt-dispatch-mode-descent-ladder

#### Automatic Reason Strings Describe Preference and Descent

**Decision**: Successful automatic launch suffixes use `mode: <rung> (preferred)` when no descent occurs and `mode: <rung> (descended: <reason>[; <reason>])` when prerequisites force a lower rung; stderr notices use the same reason vocabulary.
**Why**: The output names both the selected rung and the causal missing prerequisite while keeping stdout bounded and machine-oriented diagnostics on stderr per toolkit standards.
**Rejected**: Retaining the four environment-only `auto:` strings, which cannot explain native capability or multi-rung descent.
*Introduced by*: 260808-yilt-dispatch-mode-descent-ladder

## Tasks

### Phase 1: Setup

- [x] T001 Add `ProviderConfig.Native`, `DispatchConfig.Mode`, `DefaultDispatchMode`, the warning accessor, built-in claude capabilities, configref projection/registry/segment updates, and focused tests in `src/go/fab/internal/config/`, `internal/agent/`, and `internal/configref/`. <!-- R1 R2 R7 -->

### Phase 2: Core Implementation

- [x] T002 Implement the pure capability descent ladder, native mode, automatic reason vocabulary, and full matrix tests in `src/go/fab/internal/dispatch/pane_mode.go` and `dispatch_test.go`. <!-- R3 -->
- [x] T003 Wire `src/go/fab/cmd/fab/resolve_agent.go` to the shared ladder, add actionable no-capability errors, and replace watchable cases with mode/capability/default-byte-stability tests in `resolve_agent_test.go`. <!-- R3 R4 -->
- [x] T004 Rework `src/go/fab/cmd/fab/dispatch_start.go` and `dispatch_restart.go` to resolve config/provider capability through the same ladder, preserve explicit overrides, handle tmux-probe descent and native errors, and update command/integration tests. <!-- R3 R5 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Add `src/kit/migrations/2.17.2-to-2.18.0.md`, bump `src/kit/VERSION`, and verify the migration's both-scope, live-key-only, atomic, and idempotent instructions against established migrations. <!-- R6 -->
- [x] T006 Update configupgrade/cmd golden and coverage tests, adjust the `internal/configscope` dispatch comment, regenerate `fab/project/config.yaml` with `fab config upgrade`, and verify reference JSON/YAML/fence idempotence. <!-- R2 R7 -->
- [x] T007 Sweep the affected canonical skill text in `src/kit/skills/` and every corresponding `docs/specs/skills/SPEC-*.md` mirror for the mode/capability/branch-on-presence contract and exact selection messages. <!-- R8 --> <!-- rework: review cycle 1 must-fix — SPEC-_cli-fab.md:30-33 replaced full resolve-agent/config/dispatch inventory rows with abbreviated summaries, deleting unchanged contracts (profile/override precedence, config show/init cascade, dispatch state/wait/restart/reap semantics, exit codes); restore those contracts verbatim and change ONLY the retired watchable/presence claims -->
- [x] T008 Update `docs/specs/harness-adapters.md`, `stage-models.md`, `config.md`, `architecture.md`, `glossary.md`, and any in-scope index descriptions or historical-current claims found by the full mirror search. <!-- R7 R8 --> <!-- rework: review cycle 1 must-fix — sweep incomplete: docs/specs/stage-models.md:31, glossary.md:84, architecture.md:373, and skills.md:65 still describe providers as session_command+dispatch_command only, omitting the native capability flag and dispatch.mode ladder (R8/A-008: one consistent three-capability contract everywhere); also should-fix src/go/fab/internal/agent/agent.go:520-522 ResolveProvider comment still calls missing dispatch_command a failure surface — reword to "no reachable capability at or below the selected mode" -->

### Phase 4: Polish

- [x] T009 Run repository-wide retired-claim sweeps, format Go, run affected package tests followed by the full `src/go/fab` module suite, verify help-dump/CLI standards conformance for the changed command tree text, and confirm no out-of-scope or memory/deployed-skill edits. <!-- R1 R2 R3 R4 R5 R6 R7 R8 -->

## Execution Order

- T001 and T002 establish the config/capability types and shared ladder required by T003 and T004.
- T003 and T004 must agree on identical ladder inputs before documentation in T007-T008 is finalized.
- T006 depends on T001's registry segment and built-in provider projection.
- T007-T008 document the exact strings and behavior finalized by T002-T004.
- T009 validates every prior task and is last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: Provider configs expose independent session, headless, and native capabilities, and claude ships all three with the verified CLI grammar.
- [x] A-002 R2: `dispatch.mode` is the only live mode-preference schema field and resolves absent/invalid/valid inputs as specified.
- [x] A-003 R3: The complete preference × capability × tmux matrix descends only and errors only when no rung is possible.
- [x] A-004 R4: `resolve-agent` omits `dispatch=` only for resolved native mode and emits the correct substituted command for pane/headless.
- [x] A-005 R5: `dispatch start` and `restart` preserve explicit overrides and use the shared automatic ladder for current config/environment.
- [x] A-006 R6: Version 2.18.0 includes an idempotent both-scope migration from live watchable values.
- [x] A-007 R7: Reference YAML/JSON, provider defaults, upgrade fences, and their tests expose the new schema consistently.
- [x] A-008 R8: Canonical skills, every touched skill mirror, and aggregate specs state the same ladder and presence-branch contract.

### Behavioral Correctness

- [x] A-009 R1: Adding or shipping a `dispatch_command` no longer changes a provider's selected mode by itself.
- [x] A-010 R2: Invalid mode values warn on stderr without breaking config loading or changing stdout contracts.
- [x] A-011 R3: Pane preference outside tmux descends to native for claude and to headless for codex/gemini, while native/headless preferences never ascend to pane.
- [x] A-012 R4: Default `native` retains shipped resolver behavior: claude uses the native adapter and codex/gemini use headless CLI.
- [x] A-013 R5: A start-time pane loss re-descends; landing on native yields an actionable re-resolve error before state writes.
- [x] A-014 R6: Stale unmigrated `watchable` is inert and no read-time alias remains.

### Removal Verification

- [x] A-015 R2: `GetDispatchWatchable`, the watchable registry row, and current-schema references are removed from implementation surfaces.
- [x] A-016 R6: No binary code reads `dispatch.watchable`; only historical migration/intake artifacts may retain it where historically accurate.
- [x] A-017 R8: The retired presence-selects-mode doctrine and old `auto:` suffix set are absent from all current canonical skill/spec surfaces.

### Scenario Coverage

- [x] A-018 R3: Table tests cover all three preferences across native-capable, command-only, mixed, and no-capability providers with and without tmux.
- [x] A-019 R4: Resolver tests cover shipped defaults, pane descent, no-rung providers, depth-selected unknown providers, and explicit unknown-provider precedence.
- [x] A-020 R5: Launcher tests cover explicit pane/headless/timeout/server signals, direct automatic selections, one- and two-rung descents, tmux-probe failure, native landing, and restart re-derivation.
- [x] A-021 R7: Reference, JSON parity, managed-fence, golden, byte-stability, and idempotence tests pass with the new dispatch block.

### Edge Cases & Error Handling

- [x] A-022 R2: Nil config, absent mode, invalid mode, and all valid mode values have deterministic accessor coverage.
- [x] A-023 R3: Providers with no capabilities cannot silently resolve native or launch an empty command.
- [x] A-024 R4: Explicit `--provider` unknown-name lookup still fires before capability selection, while config-selected unknown providers report missing capabilities.
- [x] A-025 R5: Explicit pane prerequisites remain hard errors, automatic missing prerequisites descend with notices, and no partial dispatch state is written on native/no-rung failure.
- [x] A-026 R6: Migration instructions preserve comments/fences/unrelated YAML and handle absent files and repeat execution safely.

### Code Quality

- [x] A-027 Pattern consistency: New Go code follows surrounding naming, error-wrapping, table-test, cobra-stream, and package-boundary patterns.
- [x] A-028 No unnecessary duplication: Resolver and launcher consume one pure descent implementation and reuse existing profile substitution/config loading utilities.
- [x] A-029 Readability and maintainability: Selection state and reason values make the ladder explicit without oversized functions or hidden policy.
- [x] A-030 Composition: Existing config, agent, spawn, pane, and dispatch helpers remain the seams for loading, profile substitution, and tmux operations.
- [x] A-031 No god functions: Launcher restructuring keeps mode selection, prerequisite validation, command composition, and process launch focused.
- [x] A-032 No magic strings or numbers: Mode names, defaults, and reusable reason/error vocabulary are named constants.
- [x] A-033 Canonical-source discipline: No `.claude/skills/` or `docs/memory/` files are modified, and every canonical skill edit has its SPEC mirror.
- [x] A-034 Migration discipline: The user-data schema rewrite ships only through the versioned migration and managed fence regeneration.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds the new mode/capability contract while removing its planned watchable-era counterparts; no additional existing code became redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Claude's headless grammar is `claude -p --dangerously-skip-permissions --model {model} --effort {effort}` with the prompt on stdin | The installed Claude CLI help explicitly supports `-p`, stdin print mode, `--dangerously-skip-permissions`, `--model`, and `--effort` together; the intake pins this posture | S:95 R:90 A:95 D:95 |
| 2 | Confident | `internal/dispatch` owns a pure shared automatic ladder and adds `ModeNative`; native is selectable but never persisted as a launched dispatch record | Both current seams already import/use `internal/dispatch`; a shared pure selector is the clearest way to satisfy the no-disagreement invariant without introducing a package cycle | S:80 R:85 A:90 D:80 |
| 3 | Confident | Automatic output uses `mode: <rung> (preferred)` or `mode: <rung> (descended: <reason>[; <reason>])`, with matching descent notices on stderr | The plan pins rung+reason visibility but delegates exact strings; this form is bounded, actionable, stdout/stderr-correct, and directly testable | S:65 R:85 A:80 D:60 |
| 4 | Certain | The existing top-level `dispatch` entry in `internal/configscope` remains behaviorally unchanged; only its stale watchable comment changes | Scope is keyed by top-level YAML key and already resolves `dispatch` to `ScopeBoth`, exactly satisfying the new field | S:95 R:95 A:95 D:95 |
| 5 | Confident | Historical migration files retain watchable/presence wording only when they describe the historical version being migrated; current-behavior claims in active skills/specs are rewritten | Rewriting history would make old migrations inaccurate, while the binding sweep requires current canonical contracts to drop the retired doctrine | S:75 R:90 A:85 D:75 |

5 assumptions (2 certain, 3 confident, 0 tentative).
