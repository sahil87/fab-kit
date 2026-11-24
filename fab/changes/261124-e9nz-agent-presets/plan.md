# Plan: Agent Presets

**Change**: 261124-e9nz-agent-presets
**Intake**: `intake.md`

## Requirements

### Config: Preset definitions and selector

#### R1: `agent_presets` definitions table
The config schema SHALL gain a top-level `agent_presets` map (name → sparse definition `{description, session, workers, profiles.<role>.{cli,model,effort}}`), scope `both`, merging per key across layers (built-in < project < system < env). The reserved name `none` MUST NOT be a definition.

- **GIVEN** a project defines `agent_presets.mine.workers: codex` and a system file tweaks `agent_presets.mine.description`
- **WHEN** the config loads
- **THEN** `mine` carries both fields (per-leaf merge)

#### R2: Built-in presets
`defaults.yaml` SHALL ship exactly `claude` {session, workers: claude}, `codex-workers` {workers: codex} and `codex-review` {profiles.review.cli: codex}, each with the specified description. They are inert until selected: nothing-set resolution is byte-identical to today. The built-in tier reaches `internal/config` through an `init()` push from `internal/agent`.

- **GIVEN** no config and no env
- **WHEN** any role resolves
- **THEN** the result is identical to the pre-change resolution

#### R3: `agent.preset` selector, env, and reserved `none`
`agent.preset` SHALL be a scalar, scope `both`, registered in the registry/fence with env form `FAB_AGENT_PRESET` through the existing mechanical env layer (no special-case env code). `none` participates in selection like a name (highest layer wins) but activates no preset. An empty value is unset.

- **GIVEN** project `agent.preset: codex-review`, env `FAB_AGENT_PRESET=none`
- **WHEN** resolving
- **THEN** no preset applies

### Resolution: precedence

#### R4: Choosing-layer splice
Exactly one preset is active: the value of `agent.preset` at the highest layer (env > system > project) with a non-empty value. Its fields SHALL be spliced in at that layer directly below that layer's own explicit `agent.*` fields (`MergeLayers(project', system', env')` with only the choosing layer replaced by `MergeLayers({agent: presetFields}, L)`). The splice lives in the config loader (`fromProjectMap` and `layersFrom`, identically) so every consumer and `fab config show` see the same tree. Invocation flags stay on top.

- **GIVEN** project `agent.preset: codex-workers` + project `agent.workers: agy`
- **WHEN** a worker role resolves
- **THEN** agy wins (explicit field at the choosing layer beats the preset)
- **GIVEN** env `FAB_AGENT_PRESET=codex-workers` + system `agent.workers: claude`
- **THEN** codex (the preset beats lower layers' explicit fields)
- **GIVEN** project preset `codex-review` shadowed by system preset `codex-workers`
- **THEN** only codex-workers applies; nothing from codex-review is merged

#### R5: Unknown name
An unknown preset name at resolve time SHALL emit `fab: warning: unknown agent preset "<name>" (agent.preset from <layer>); ignoring it` on stderr and resolve as if no preset were chosen. It MUST NOT error.

#### R6: CLI re-derivation and provenance
A CLI set by a preset SHALL re-derive model/effort from that CLI's fills exactly as an `agent.profiles.<role>.cli` override does. `Config` SHALL carry the active preset name, choosing layer, and the set of `agent.*` leaves the preset actually supplied, so `ResolveRoleWithSource` labels those fields `agent_presets.<name>` in `source.*`; fills re-derived from the preset-chosen CLI keep their `clis.<p>.profiles.<role>` label.

- **GIVEN** `agent.preset: codex-review`
- **WHEN** `review` resolves
- **THEN** cli=codex with `source.cli: agent_presets.codex-review`, and model/effort from `clis.codex.profiles.review`

### Commands

#### R7: `fab config set/unset agent.preset` and dotted leaves
`fab config set [--system] agent.preset <name>` SHALL reject a name that is neither defined (built-in ∪ system ∪ project ∪ env) nor `none`, with a non-zero usage error naming the defined set sorted. `agent_presets.<n>[.description|session|workers]`, `agent_presets.<n>.profiles[.<role>[.cli|model|effort]]` dotted keys SHALL be settable/unsettable like `clis.<n>.<field>`.

#### R8: `fab agent --preset`
`fab agent [role|stage] --preset <name>` SHALL be equivalent to choosing the preset at the env layer for this one resolution (loses to explicit env `FAB_AGENT_*` fields; flags still on top; unknown name warns; `none` disables). It is resolution-only — it does NOT export `FAB_AGENT_PRESET` into an exec'd session; the help text states that flags do not reach `fab dispatch start`. Combined with selector-less `--cli` it is a usage error. It is not added to `fab resolve-agent`.

#### R9: `fab agent presets [--json]`
A new `fab agent presets [--repo <path>] [--json]` SHALL list every defined preset sorted by name (never `none`). Human output: table of name, source, description, set-field summary. `--json`: array of `{name, description, source, [session], [workers], [profiles]}` — sparse, no `active` flag; `source` ∈ `built-in|system|project|env` = the highest layer contributing to the definition. This is a cross-repo contract and is documented as such.

#### R10: `-o yaml` and `show --origin`
`fab agent -o yaml` SHALL emit a top-level `preset: <name>` key only when a preset is active. `fab config show --origin` SHALL attribute each preset-supplied `agent.*` leaf to the choosing layer with detail `preset <name>`; bare `fab config show` prints the spliced tree.

### Docs and guards

#### R11: Docs, reference, fixtures, drift guards
Reference fence/explain prose, `docs/specs/config.md`, `docs/specs/stage-models.md`, `src/kit/skills/_cli-fab.md` (no `docs/specs` or `src/go` citations), and help-dump fixtures SHALL describe the feature. `TestDefaultsFileIsWellFormed` SHALL validate built-in presets and a pin test SHALL fix the three shapes.

### Non-Goals
- H1/H2 (HexoKit), R1 (release), `.status.yaml` preset storage, changes to existing keys, the stage→role map, or the `clis` table.

### Design Decisions

#### Splice at the config loader
**Decision**: the preset splice is implemented once at `internal/config` (`fromProjectMap`/`layersFrom`), with provenance carried on `Config` for `internal/agent`.
**Why**: layer identity exists only there; every consumer already loads through it.
**Rejected**: splice in `internal/agent` — receives an already-merged `*Config`, layers are gone.
*Introduced by*: 261124-e9nz-agent-presets

## Tasks

### Phase 1: Setup

- [x] T001 Add `agent_presets` block to `src/go/fab/defaults.yaml` <!-- R2 -->
- [x] T002 Add `AgentPreset` type, `Config.AgentPresets`, `AgentConfig.Preset`, `Config.Preset` provenance, `BuiltinAgentPresets` var in `src/go/fab/internal/config/config.go`; register `agent.preset` and `agent_presets` in `internal/configscope/configscope.go` <!-- R1, R3 -->

### Phase 2: Core Implementation

- [x] T003 Implement the splice (selection, definitions merge, unknown/none handling, provenance) in new `src/go/fab/internal/config/preset.go`, wire into `fromProjectMap` and `layersFrom`, add `FromLayers` and an env-preset override entry point <!-- R4, R5, R6 --> <!-- rework: cycle 1 must-fix — make LoadPath/Load/LoadNoProject one-line delegations to the *WithPreset variants (config.go:461-492); route the env-only fallback (config.go:488,:527) through applyPreset so FAB_AGENT_PRESET/--preset survive a schema-incompatible system file; extract applyPreset supplied-leaf walk (preset.go:141-169) into a helper to get under 50 lines; fix preset.go:83 comment (presetFields → presetRoleLeaves); export the preset layer names so cmd/fab withPresetTier (config.go:743) compares against shared constants -->
- [x] T004 `internal/agent/agent.go`: parse built-in presets in `init()` push; label preset-supplied fields `agent_presets.<name>` in `ResolveRoleWithSource`; add `Preset` to `Resolution` in `resolution.go` <!-- R2, R6, R10 -->
- [x] T005 Registry rows, segments, fence line, explain prose and dynamic dotted keys in `src/go/fab/internal/configref/configref.go` (+ `defaultsmap.go` if needed) <!-- R3, R7, R11 -->
- [x] T006 `fab config set` validation for `agent.preset`; `show --origin` preset tier; `FromLayers` use in `cmd/fab/config*.go` <!-- R7, R10 --> <!-- rework: cycle 1 should-fix — config_validate.go:35 must reject set/unset of agent_presets.none[.…] (reserved name) at write time -->
- [x] T007 `fab agent --preset` and `fab agent presets [--json]` in `cmd/fab/agent.go`, new `cmd/fab/agent_presets.go`, `-o yaml` `preset` key in `cmd/fab/resolution.go` <!-- R8, R9, R10 -->

### Phase 3: Tests and Integration

- [x] T008 Config-loader/precedence table tests (7 required cases + `none`) in `src/go/fab/internal/config/preset_test.go` and `internal/agent` <!-- R4, R5, R6 -->
- [x] T009 cmd tests: `agent_presets_test.go`, `--preset` equivalence + bare `--cli` rejection, `config set agent.preset`, `show --origin`, `dispatch_start_test.go` headless preset case <!-- R7, R8, R9, R10 -->
- [x] T010 Drift guards: extend `internal/agent/defaults_test.go` (well-formed + pin), doc table guard if docs restate built-ins <!-- R11 --> <!-- rework: cycle 1 should-fix — extend TestDocTablesMatchAgentMaps (stagemodels_doc_test.go) to check docs/specs/stage-models.md § Agent presets table against config.BuiltinAgentPresets -->
- [x] T011 Fix existing fixtures/goldens affected by the new registry rows and subcommand (help-dump, configref, configscope tests) <!-- R11 -->

### Phase 4: Polish

- [x] T012 [P] Docs: `docs/specs/config.md`, `docs/specs/stage-models.md` <!-- R11 --> <!-- rework: cycle 1 should-fix — add `fab agent presets` to the config-free command list in src/go/fab/cmd/fab/skill.md:22 and docs/site/skill.md:22 -->
- [x] T013 [P] Kit: `src/kit/skills/_cli-fab.md` § fab agent (JSON contract; no docs/specs or src/go citations) <!-- R11 -->
- [x] T014 Run `go test ./...` and `go vet ./...` <!-- R11 -->

## Execution Order

- T002 blocks T003; T003 blocks T004, T006, T007
- T005 depends on T002; T008–T011 follow their implementation tasks

## Acceptance

### Functional Completeness

- [x] A-001 R1: `agent_presets` merges per key across built-in/project/system/env and `none` is not definable
- [x] A-002 R2: the three built-in presets ship with exact shapes and are inert when unselected
- [x] A-003 R3: `agent.preset` registered (scope both, fence line) and `FAB_AGENT_PRESET` works without special-case env code
- [x] A-004 R4: choosing-layer splice implemented once in the loader, identical in `fromProjectMap` and `layersFrom`
- [x] A-005 R5: unknown name warns with the specified text and resolves as unset
- [x] A-006 R6: `source.*` names `agent_presets.<name>` for preset-supplied fields; re-derived fills keep their `clis.*` label
- [x] A-007 R7: `config set agent.preset` rejects undefined names, accepts defined and `none`; dotted `agent_presets.*` leaves are settable
- [x] A-008 R8: `fab agent --preset` works, is resolution-only, rejects selector-less `--cli`, absent from `resolve-agent`
- [x] A-009 R9: `fab agent presets [--json]` emits the documented sparse sorted contract with `source`
- [x] A-010 R10: `-o yaml` carries `preset:` only when active; `show --origin` attributes preset leaves
- [x] A-011 R11: docs, `_cli-fab.md`, fixtures, drift guards updated (well-formed check lives in the sibling `TestDefaultsFileBuiltinPresetsAreWellFormed`; `TestDocTablesMatchAgentMaps` now also checks the stage-models.md § Agent presets table against `config.BuiltinAgentPresets`)

### Behavioral Correctness

- [x] A-012 R4: with nothing set, resolution and all pre-existing tests are byte-identical to before
- [x] A-013 R4: all seven brief precedence cases plus reserved `none` pass as table tests

### Scenario Coverage

- [x] A-014 R4: `fab dispatch start` headless path picks the preset up from config (composed command uses codex)
- [x] A-015 R8: `--preset X` ≡ `FAB_AGENT_PRESET=X` for one resolution

### Edge Cases & Error Handling

- [x] A-016 R3: empty `FAB_AGENT_PRESET=` is unset; `none` at a higher layer cancels a lower preset
- [x] A-017 R5: a stale name never fails a spawn

### Code Quality

- [x] A-018 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [x] A-019 No unnecessary duplication: Existing utilities (MergeLayers, configvalue, dynamicMatch, agent_models template) reused; `Load`/`LoadPath`/`LoadNoProject` are now one-line delegations to their `*WithPreset` variants (internal/config/config.go:393, :456, :496)

## Notes

- Check items as you review: `- [x]`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the cycle-1 candidate — duplicated `Load*` bodies — was resolved by making the originals delegate to their `*WithPreset` twins)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | An unknown name at the highest choosing layer yields NO preset (no fallthrough to a lower layer's preset) | Brief: "the key is treated as unset" for the winner; test case 5 says resolution equals the unset case | S:70 R:70 A:75 D:70 |
| 2 | Confident | `Config.AgentPresets` holds only user-layer (file+env) definitions; built-ins are composed via a config helper/var, like the `clis` table | Matches the `CLIs`/`ResolveCLI` precedent; Layers.Effective never carries the built-in tier | S:65 R:75 A:80 D:75 |
| 3 | Confident | A preset leaf is "supplied" only when no layer at or above the choosing layer carries a non-empty value at that `agent.*` path | Makes `source.*` truthful: explicit higher-layer fields keep their normal label | S:70 R:70 A:80 D:75 |
| 4 | Confident | `show --origin` renders the preset tier as the choosing layer's kind with detail `<layer label> (preset <name>)` and JSON `origin_detail` `preset <name>` | Brief gives "origin = layer, detail = preset <name>" without exact text | S:60 R:80 A:70 D:65 |
| 5 | Confident | `--preset` unknown-name warning reports layer `--preset flag`; `fab: warning:` prefix is unchanged | Distinguishes flag from env var in the message; format otherwise per brief | S:60 R:85 A:75 D:70 |
| 6 | Tentative | `fab config set agent_presets.<n>.session/workers/profiles.*.cli` does not validate CLI names | Brief demands validation only for `agent.preset`; CLI names are opaque and may be defined later | S:55 R:80 A:60 D:55 |

6 assumptions (0 certain, 5 confident, 1 tentative).
