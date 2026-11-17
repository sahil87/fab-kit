# Plan: Native rung descends when the model has no Agent-tool alias

**Change**: 261117-3ms2-native-rung-descends-without-alias
**Intake**: `intake.md`

## Requirements

### Resolution: Native-rung availability

#### R1: A non-empty model without a native alias makes the native rung unavailable
The native dispatch rung SHALL be available for a resolution only when the provider declares
`native: true` AND the resolved model is either empty or has a native Agent-tool alias. "Has a native
alias" MUST be exactly the `model_alias` computation: `agent.IsModelAlias(agent.ModelAlias(model))`
(true for `opus`/`sonnet`/`haiku`/`fable` and versioned Claude IDs such as `claude-opus-5-5`,
`claude-haiku-4-5-20251001`; false for anything else). The predicate MUST live once in
`internal/agent` (e.g. `agent.HasNativeModel(model string) bool`), and `composeAgentResolution`'s
`ModelAlias` assignment MUST reuse the same alias check rather than re-deriving it. The predicate MUST
key only on the model string (provider-neutral).

- **GIVEN** `dispatch.mode: native` and `agent.profiles.doing.model: glm-latest[1m]` (claude)
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** the output carries `dispatch: {rung: headless, command: claude -p … --model glm-latest[1m] --effort high, reason: …}` and `model_alias: ""`

- **GIVEN** the doing role resolves an empty model, `opus`, or `claude-opus-5-5` on claude
- **WHEN** `fab agent apply -o yaml` runs under `dispatch.mode: native`
- **THEN** no `dispatch:` key is emitted (native, unchanged)

- **GIVEN** `dispatch.mode: pane` with tmux available and an alias-less model
- **WHEN** resolution runs
- **THEN** pane resolves (the native rule is never reached)

#### R2: SelectMode reports why native was skipped
`dispatch.SelectMode`'s `native bool` input SHALL become a caller-supplied availability value
(`dispatch.NativeAvailability`, mirroring `TmuxAvailability`) with three states: available, no
capability, and model has no native alias. The skip text for "no capability" MUST stay
byte-identical (`native unavailable`); the skip text for the alias case MUST be
`native unavailable: model has no native alias`. A helper SHOULD build the value from
(capability bool, model-has-alias bool) so callers don't hand-roll the mapping.

- **GIVEN** preference `native`, availability = no alias, headless available
- **WHEN** `SelectMode` runs
- **THEN** it returns `headless` with reason `mode: headless (descended: native unavailable: model has no native alias)`

- **GIVEN** preference `pane`, tmux absent, availability = no alias, headless available
- **WHEN** `SelectMode` runs
- **THEN** reason is `mode: headless (descended: pane unavailable: no tmux; native unavailable: model has no native alias)`

- **GIVEN** availability = no alias and no headless capability
- **WHEN** `SelectMode` runs
- **THEN** it returns `ErrNoMode`

#### R3: `fab agent -o yaml` carries `dispatch.reason`
`agent.DispatchResolution` SHALL gain `Reason string` with YAML tag `reason,omitempty`, ordered after
`command`. It MUST carry the ladder's descent causes (the `; `-joined skip list, without the
`mode: … (descended: …)` wrapper) and MUST be omitted when the selected rung is the preferred one.
`Resolution.Lines` (the `fab resolve-agent` line protocol) MUST NOT gain a line.

- **GIVEN** the R1 glm case
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `dispatch.reason` is `native unavailable: model has no native alias`

- **GIVEN** a role resolving to built-in codex under `dispatch.mode: native`
- **WHEN** `fab agent <stage> -o yaml` runs
- **THEN** `dispatch.reason` is `native unavailable`

- **GIVEN** `dispatch.mode: headless`
- **WHEN** resolution runs
- **THEN** `dispatch:` has no `reason` key

#### R4: Every rung-selection site applies the rule
All automatic `SelectMode` callers MUST compute native availability with the shared predicate over the
profile's resolved model: `dispatchResolutionFor` (`cmd/fab/resolution.go`), `launchFlags.resolveMode`
and its pane-validation re-descent (`cmd/fab/dispatch_start.go`), and `setupcheck.ProbeDispatchMode`
(`internal/setupcheck/probes.go`, over the `doing` profile it already resolves).

- **GIVEN** the R1 config inside a fab project
- **WHEN** `fab dispatch start <change> apply` runs
- **THEN** it launches headless (no `nativeDispatchError`) and stderr prints `dispatch selection: mode: headless (descended: native unavailable: model has no native alias)`

- **GIVEN** the R1 config
- **WHEN** `fab doctor`'s dispatch probe runs
- **THEN** it reports Warn `dispatch.mode "native" would descend at dispatch time: mode: headless (descended: native unavailable: model has no native alias)`

#### R5: The no-rung error names the alias cause
When the alias rule is what leaves the ladder empty (provider declares `native: true`, model has no
alias, no lower rung), the error MUST name the model and say it has no native Agent-tool alias, MUST
NOT suggest configuring `providers.<p>.native`, and MUST still wrap `dispatch.ErrNoMode`. It SHOULD
suggest `providers.<p>.headless_command`, or an alias / empty model.

- **GIVEN** a user provider `p` with `native: true`, no `headless_command`, preference `native`, model `glm-latest[1m]`
- **WHEN** `fab agent <stage> -o yaml` runs
- **THEN** the error mentions `glm-latest[1m]` and "no native Agent-tool alias", not `providers.p.native`, and `errors.Is(err, dispatch.ErrNoMode)`

#### R6: Empty model keeps inherit; F2 hand-off documented
An empty resolved model MUST keep native available (session-model inherit). Code comments and docs
MUST state that a future "use the tool's default" token resolves to empty before this rule runs.

- **GIVEN** an empty resolved model on claude (`--model=` override)
- **WHEN** `fab agent apply --model= -o yaml` runs under `dispatch.mode: native`
- **THEN** no `dispatch:` key is emitted

### Docs: Skills, specs, config reference

#### R7: Deployed skills and config reference restate the rule
`src/kit/skills/_preamble.md` (§ Per-Stage Model Resolution model-seam row and § CLI-Adapter Dispatch
ladder/reason vocabulary), `src/kit/skills/_cli-fab.md` (§ fab agent ladder + YAML schema
`dispatch.reason` + `model_alias` consequence), and `src/kit/skills/_cli-fab-pane.md` (descent-reason
list) MUST state the native prerequisite and the new reason string, without citing fab-kit-only paths.
`internal/configref` `dispatch.mode` Description and segment prose MUST state native's model
prerequisite; any resulting fence/digest test fallout MUST be handled per the existing digest-catalog
convention. `docs/specs/stage-models.md` and `docs/specs/harness-adapters.md` MUST be updated where
they enumerate native's prerequisites or the descent-reason strings.

- **GIVEN** the updated docs
- **WHEN** a dispatch site reads `_preamble.md`
- **THEN** it learns that the native branch never sees a non-empty alias-less model, and that an empty `model_alias` on the native branch means "empty model → inherit" only

### Non-Goals

- Aliasing custom models (e.g. mapping `glm-latest[1m]` to anything) — F3 Brief non-goal
- Implementing F2's tool-default token — later step
- Renaming `provider` → `cli` — N1

### Design Decisions

#### Descend instead of erroring or warning
**Decision**: A non-empty model with no Agent-tool alias removes the native rung; the ladder descends to the next supported rung (headless for claude).
**Why**: The Agent tool's `model` enum only accepts `opus`/`sonnet`/`haiku`/`fable`, so the native arm silently ran such roles on the session model; `claude -p --model <custom>` honours the model, and descent reuses the existing ceiling/descent ladder every dispatch site already handles.
**Rejected**: Erroring (blocks a deliberately configured stage); warning (leaves the silent model substitution in place); aliasing custom models (out of scope, no faithful mapping).
*Introduced by*: 261117-3ms2-native-rung-descends-without-alias

#### Native availability as a reason-carrying value
**Decision**: `SelectMode` takes a caller-computed `NativeAvailability` (like `TmuxAvailability`) instead of a bool, and `dispatch.reason` exposes the skip list generally (any descent), not only the alias cause.
**Why**: Keeps `SelectMode` pure while letting the reason name the real cause; one uniform `reason` field explains every descent.
**Rejected**: A second bool parameter (no place for the reason text); an F3-only reason field (ad hoc, inconsistent with `dispatch start`'s stderr line).
*Introduced by*: 261117-3ms2-native-rung-descends-without-alias

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add `agent.HasNativeModel(model string) bool` (empty or `IsModelAlias(ModelAlias(model))`) in `src/go/fab/internal/agent/agent.go` with doc comment covering the F2 hand-off; table test in `src/go/fab/internal/agent/agent_test.go` <!-- R1 -->
- [x] T002 In `src/go/fab/internal/dispatch/pane_mode.go`: add `NativeAvailability` type + constants + a constructor helper (capability, hasAlias), change `SelectMode`'s native param to it with the two skip texts, and add an `AutoReason` method that returns the descent list (empty when preferred/explicit); update `TestSelectMode`, `TestSelectModeAutomaticMatrix` and add alias cases in `src/go/fab/internal/dispatch/dispatch_test.go` <!-- R2 -->
- [x] T003 `src/go/fab/internal/agent/resolution.go`: add `DispatchResolution.Reason` (`reason,omitempty`, after `command`); `src/go/fab/cmd/fab/resolution.go`: compute native availability from `prov.Native` + `agent.HasNativeModel(profile.Model)`, set `Reason` from the returned `AutoReason`, reuse the predicate/alias check for `ModelAlias` <!-- R1, R3, R6 -->
- [x] T004 `src/go/fab/cmd/fab/dispatch_start.go`: pass the resolved profile's native availability into `resolveMode` and the pane-validation re-descent <!-- R4 -->
- [x] T005 `src/go/fab/cmd/fab/resolve_agent.go` `noDispatchCapabilityError` (+ its callers): when the alias rule emptied the ladder, name the model/alias cause and drop the `native` remedy; keep `ErrNoMode` wrapped <!-- R5 -->
- [x] T006 `src/go/fab/internal/setupcheck/probes.go` `ProbeDispatchMode`: apply the predicate to the doing profile; update its doc comment's reason list; add a probe test <!-- R4 -->

### Phase 2: Integration tests

- [x] T007 `src/go/fab/cmd/fab` tests: `fab agent -o yaml` glm case (dispatch headless + reason + empty model_alias), empty/`opus`/`claude-opus-5-5` regression (no dispatch), codex `reason: native unavailable`, headless preference no reason, `--model=` empty override stays native; `fab dispatch start` glm case launches headless with the stderr descent line; no-rung error test for a native-only user provider; update existing exact-YAML expectations that now include `reason` <!-- R1, R3, R4, R5, R6 -->

### Phase 3: Docs

- [x] T008 Update `src/kit/skills/_preamble.md`, `src/kit/skills/_cli-fab.md`, `src/kit/skills/_cli-fab-pane.md`, `src/go/fab/internal/configref/configref.go` (dispatch.mode Description + segment; handle digest/fence test fallout per convention), `docs/specs/stage-models.md`, `docs/specs/harness-adapters.md`; run `go test ./...` in `src/go/fab` <!-- R7 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `agent.HasNativeModel` exists in `internal/agent`, returns true for empty/aliases/versioned Claude IDs and false otherwise, and is the only native-alias predicate used by resolution and rung selection
- [ ] A-002 R2: `SelectMode` takes a `NativeAvailability`; the no-capability skip text is still exactly `native unavailable`, and the alias skip text is `native unavailable: model has no native alias`
- [ ] A-003 R3: `fab agent -o yaml` emits `dispatch.reason` after `command` when a descent happened and omits it otherwise; `fab resolve-agent` line output is unchanged in shape
- [ ] A-004 R4: `dispatchResolutionFor`, `resolveMode`, the pane re-descent, and `ProbeDispatchMode` all apply the predicate
- [ ] A-005 R5: The no-rung error for the alias case names the model and the missing alias, omits the `native` remedy, and wraps `ErrNoMode`
- [ ] A-006 R6: An empty resolved model stays native; the F2 hand-off is documented in the predicate's comment and in the docs
- [ ] A-007 R7: `_preamble.md`, `_cli-fab.md`, `_cli-fab-pane.md`, configref prose, `stage-models.md`, and `harness-adapters.md` state the native model prerequisite and the new reason string

### Behavioral Correctness

- [ ] A-008 R1: A claude role pinned to `glm-latest[1m]` under `dispatch.mode: native` now resolves `dispatch.rung: headless` with the model in the command, instead of native
- [ ] A-009 R4: `fab dispatch start` for that stage launches headless instead of refusing with `nativeDispatchError`

### Scenario Coverage

- [ ] A-010 R2: Dispatch-package tests cover native-preference alias descent, pane+no-tmux two-rung descent, alias-caused `ErrNoMode`, and pane-with-tmux unaffected
- [ ] A-011 R3: cmd tests cover glm reason, codex `native unavailable` reason, and no reason under headless preference
- [ ] A-012 R4: a `fab dispatch start` test and a setupcheck probe test cover the alias descent

### Edge Cases & Error Handling

- [ ] A-013 R6: `--model=` (empty override) and unpinned alias fills keep the native rung (no `dispatch:`)
- [ ] A-014 R5: native-only user provider + alias-less model produces the R5 error

### Code Quality

- [ ] A-015 Pattern consistency: New code follows naming and structural patterns of surrounding code (`TmuxAvailability` idiom, pure selector, cobra-layer env reads)
- [ ] A-016 No unnecessary duplication: the alias check exists once and every site reuses it
- [ ] A-017 Readability over cleverness: no god functions introduced; `resolveMode`/`dispatchResolutionFor` stay focused
- [ ] A-018 No magic strings: skip texts are named constants/values in `internal/dispatch`
- [ ] A-019 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`; deployed skill edits cite no fab-kit-only paths
- [ ] A-020 Go changes ship tests and the CLI reference partials (`_cli-fab.md`, `_cli-fab-pane.md`) are updated

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Predicate name `agent.HasNativeModel`; `dispatch.NativeAvailability` with a constructor helper | Intake left names to the plan; mirrors existing `TmuxAvailability` idiom | S:75 R:90 A:85 D:75 |
| 2 | Confident | `dispatch.reason` value is the bare skip list extracted from `AutoReason` via a method on `AutoReason` | Intake § 3 fixed the value shape; a method keeps parsing next to the format's owner | S:80 R:85 A:80 D:75 |
| 3 | Confident | Update `docs/specs/stage-models.md` and `harness-adapters.md` only where they enumerate native prerequisites / descent strings | Keeps specs true without rewriting design intent; F1/D1 precedent edited the same spec | S:70 R:85 A:75 D:75 |
| 4 | Confident | configref prose updated; if the fence segment change trips the digest catalog test, append the prior digest per convention | Same mechanism F1/D1 used for fence text changes | S:70 R:80 A:80 D:70 |

4 assumptions (0 certain, 4 confident, 0 tentative).
