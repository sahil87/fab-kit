# Plan: Reserved `tool-default` model value

**Change**: 261118-djnz-tool-default-model-token
**Intake**: `intake.md`

## Requirements

### Agent resolution: the reserved model value

#### R1: Token constant and collapse helper
`internal/agent` SHALL export the reserved model value as a named constant `ToolDefaultModel = "tool-default"` and a helper `EffectiveModel(model string) string` that returns `""` for exactly that string (case-sensitive, no trimming) and every other string verbatim. No other model validation is added.

- **GIVEN** `EffectiveModel` is called
- **WHEN** the input is `tool-default`
- **THEN** it returns `""`
- **AND** `Tool-Default`, ` tool-default`, `gpt-5.6-sol`, `opus` and `""` are returned unchanged

#### R2: The token wins over lower tiers and collapses after precedence
`agent.ResolveRoleWithSource` SHALL run the full model precedence (flag > `agent.profiles.<role>.model` > `providers.<p>.profiles.<role>.model` > `providers.<p>.profiles.default.model` > empty) with the token treated as an ordinary non-empty value, and THEN collapse the winning model through `EffectiveModel`. `source.Model` SHALL keep the rung that held the token (`agent.profiles.<role>`, `providers.<p>.profiles.<role>`, `providers.<p>.profiles.default`, or `flag`).

- **GIVEN** `agent.profiles.review: { provider: codex, model: tool-default }`
- **WHEN** role `review` resolves
- **THEN** `Model` is `""` (not the codex fill `gpt-5.6-sol`) and `source.Model` is `agent.profiles.review`
- **AND** the composed command contains no `-m` / `{model}` token pair

- **GIVEN** a provider whose `profiles.review.model` (or `profiles.default.model`) is `tool-default`
- **WHEN** the role resolves with no override
- **THEN** `Model` is `""` and `source.Model` is that provider rung's key

- **GIVEN** `--model tool-default` on `fab agent` / `fab resolve-agent`
- **WHEN** resolution runs
- **THEN** `Model` is `""` and `source.Model` is `flag`

#### R3: Provider-only addressing collapses identically
`composeAgentResolution`'s `ProviderOnly` path (which bypasses `ResolveRoleWithSource`) SHALL apply the same `EffectiveModel` collapse to the override model, so `--model tool-default` behaves identically on both addressing forms.

- **GIVEN** `fab agent --provider codex --model tool-default -o yaml` (provider-only addressing)
- **WHEN** it resolves
- **THEN** `model: ""` with `source.model: flag`, and the command carries no model flag pair

#### R4: Native rung keeps inherit; F3's descent does not fire
Because the collapse precedes `HasNativeModel` / `dispatch.NativeAvailabilityFor`, a claude role resolving to the token SHALL keep the native rung (no `dispatch:` block under `dispatch.mode: native`) with `model_alias: ""`. `HasNativeModel`'s doc comment SHALL state this as present truth naming `ToolDefaultModel`.

- **GIVEN** `dispatch.mode: native` and `agent.profiles.doing.model: tool-default` (claude)
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** output has `model: ""`, `model_alias: ""`, `source.model: agent.profiles.doing`, and no `dispatch:` key
- **AND** `fab resolve-agent apply` prints an empty `model=` line (with or without `--alias`) and no `dispatch=` line

- **GIVEN** the same role under `dispatch.mode: headless`
- **WHEN** it resolves
- **THEN** `dispatch.command` has the `--model {model}` pair dropped

#### R5: F3's no-capability error names the token
`noDispatchCapabilityError`'s alias-less-model branch SHALL list `tool-default` as a remedy alongside the existing alias / leave-empty remedy.

- **GIVEN** a provider whose native rung was skipped for an alias-less model and that has no lower rung
- **WHEN** resolution errors
- **THEN** the message mentions `tool-default`

### Config: setting the token

#### R6: `fab config set` accepts the token and the empty refusal points at it
`fab config set` SHALL accept `tool-default` on model keys (`agent.profiles.<role>.model`, `providers.<p>.profiles.<role>.model`) with no new validation. When an empty value is refused for a key whose last segment is `model`, the error SHALL additionally say `fab config set <key> tool-default` uses the tool's own default model; other keys keep today's message. The command's `Long` help SHALL mention the token next to the empty-refusal sentence.

- **GIVEN** `fab config set --system agent.profiles.review.model tool-default`
- **WHEN** it runs
- **THEN** it exits 0 and writes the leaf, and `fab agent review -o yaml` then reports `model: ""` with `source.model: agent.profiles.review`

- **GIVEN** `fab config set agent.profiles.review.model ""`
- **WHEN** it runs
- **THEN** it exits non-zero and the error mentions both `fab config unset` and `tool-default`

- **GIVEN** `fab config set dispatch.mode ""`
- **WHEN** it runs
- **THEN** the error does not mention `tool-default`

### Model discovery: the token row

#### R7: `fab agent models` prepends a reserved row per tool
`agentmodels` SHALL prepend, as the first row for every tool (built-in adapters, `models_command` tools, adapter-less tools like agy), a row `{cli: <tool>, id: "tool-default", label, efforts, default: false, source: "reserved"}` where:
- `source` is a new exported constant `SourceReserved = "reserved"`;
- `label` is `tool default (<id of the row with default: true>)` when one exists, else `tool default`;
- `efforts` is the default row's efforts when a default row exists and has efforts, else the tool's known set (claude's effort set for claude), else `[]` (never `null`);
- the row is never merged/deduped with other rows (a tool-reported row whose id is literally `tool-default` is dropped so the reserved row stays unique).

- **GIVEN** a `models_command` tool reporting `m1` as `default: true` with efforts `["high"]`
- **WHEN** `fab agent models mytool --json` runs
- **THEN** row 0 is `{"cli":"mytool","id":"tool-default","label":"tool default (m1)","efforts":["high"],"default":false,"source":"reserved"}` and the `m1` row keeps `default: true`

- **GIVEN** discovery fails (codex not on PATH)
- **WHEN** `fab agent models codex --json` runs
- **THEN** row 0 is the reserved row labelled `tool default`, followed by the fill rows, with the warning still on stderr

### Non-Goals

- `effort: tool-default` — the token is model-only; an effort value passes through verbatim.
- The provider → cli rename (N1), hexokit H3 UI, and the D1 stale-pin warning follow-up.
- A migration — no user data is restructured.

### Design Decisions

#### Collapse after precedence, not at load
**Decision**: The token is an ordinary non-empty value through the whole precedence chain and is mapped to `""` only on the winning model, in the shared resolver (plus the provider-only path).
**Why**: It wins over lower tiers for free (empty-skip treats it as a value), `source.model` naturally names the key that held it, and every resolution consumer (agent, resolve-agent, dispatch start, pane open, spawn, setupcheck, configref) inherits the behavior from one site. F3's native check then sees `""` and keeps the inherit path.
**Rejected**: Mapping at config load (would make it indistinguishable from empty and fall through to fills); special-casing in `spawn.WithProfile` / the rung selector (multiple sites, and `model_alias`/source would disagree).
*Introduced by*: 261118-djnz-tool-default-model-token

#### Token row identified by `source: reserved`
**Decision**: The `fab agent models` token row carries a distinct `source: "reserved"` and `default: false`.
**Why**: Consumers (hexokit H3) can find it without string-matching an id, and the tool's real default model keeps sole ownership of `default: true`.
**Rejected**: `source: tool` / `default: true` (would claim the token is a model the tool reports).
*Introduced by*: 261118-djnz-tool-default-model-token

## Tasks

### Phase 1: Core Implementation

- [x] T001 In `src/go/fab/internal/agent/agent.go` add `ToolDefaultModel` and `EffectiveModel`, apply `EffectiveModel` to `p.Model` at the end of `ResolveRoleWithSource` (source untouched), and rewrite `HasNativeModel`'s doc comment as present truth. Add tests in `src/go/fab/internal/agent/agent_test.go`: helper table (exact-match only), token at `agent.profiles.<role>.model`, at `providers.<p>.profiles.<role>.model`, at `providers.<p>.profiles.default.model`, and via `Overrides{Model: "tool-default", ModelSet: true}` — each yields `Model == ""` with the right `source.Model`; and the token beats a provider fill. <!-- R1 R2 -->
- [x] T002 In `src/go/fab/cmd/fab/resolution.go` apply `agent.EffectiveModel` to the `ProviderOnly` override model. Add cmd-level tests (in `src/go/fab/cmd/fab/agent_surface_test.go` or `resolve_agent_test.go`, following their helpers): `fab agent review -o yaml` with a codex `tool-default` pin (model `""`, source key, no model flag in `command`/`dispatch.command`); provider-only `--provider codex --model tool-default`; claude `tool-default` under `dispatch.mode: native` keeps no `dispatch:` key and `model_alias: ""`; under `headless` the dispatch command drops `--model`; `fab resolve-agent` prints an empty `model=` line with and without `--alias`. <!-- R2 R3 R4 -->
- [x] T003 In `src/go/fab/cmd/fab/resolve_agent.go` add the `tool-default` remedy to `noDispatchCapabilityError`'s alias-less branch; update/extend the test that pins that message (grep `has no native Agent-tool alias` under `src/go/fab/cmd/fab/*_test.go`). <!-- R5 -->
- [x] T004 In `src/go/fab/cmd/fab/config.go` extend the empty-value refusal with the `.model`-key hint and mention the token in `configSetCmd`'s `Long`; add tests in `src/go/fab/cmd/fab/config_test.go`: set `tool-default` on `agent.profiles.review.model` (and a `providers.<p>.profiles.<role>.model` key) succeeds and resolves to `model: ""` with the key as source; empty on a model key mentions `tool-default`; empty on a non-model key does not. <!-- R6 -->
- [x] T005 In `src/go/fab/internal/agentmodels/models.go` add `SourceReserved` and prepend the reserved row in `discoverWith` after normalization (label/efforts per R7, drop any tool-reported `tool-default` id). Update existing tests whose row indices/counts shift (`internal/agentmodels/models_test.go`, `cmd/fab/agent_models_test.go`, adapter tests if they go through `discoverWith`) and add tests for: labelled-with-default, plain label on failed discovery, claude efforts fallback, `[]` efforts for adapter-less tools, the table output's first data row, and dropping a tool-reported `tool-default`. <!-- R7 -->

### Phase 2: Documentation

- [x] T006 In `src/go/fab/internal/configref/configref.go` add one sentence about `tool-default` to the `agent.profiles` Description and to the providers precedence prose/Description (real value, beats lower tiers, resolves to an empty model — tool default on CLI rungs, session-model inherit on native). If the changed text sits in a digested managed-fence/advert segment, add the previous digest to `internal/configupgrade`'s legacy list per the F1 precedent and refresh any golden fixtures; run `go test ./internal/configref/... ./internal/configupgrade/... ./cmd/fab/...`. <!-- R2 R6 -->
- [x] T007 Update `src/kit/skills/_cli-fab.md`: `fab agent` / `fab resolve-agent` resolution (token collapses to `model: ""`, `source.model` keeps the key, native inherit, `--model tool-default`), `fab agent models` (the first `reserved` row, label and efforts rules, `source` enum now `tool|alias|fill|reserved`), `fab config set` (accepts the token; `.model` refusal hint). Grep `src/kit/skills/` (incl. `_preamble.md`, `_cli-fab-pane.md`) for "inherit"/empty-model wording that needs a one-clause token note; keep owner-or-pointer. <!-- R2 R4 R6 R7 -->
- [x] T008 Update specs: `docs/specs/stage-models.md` (replace the "a future 'use the tool's default' token" wording near line 643 with the present rule; add the token to the precedence/default-profile prose), `docs/specs/config.md` (empty refusal + `tool-default` as the way to request a tool's default), `docs/specs/glossary.md` (new `tool-default` entry). Grep `docs/specs/` and `src/` for "future \"use the tool's default\"" / "a future" token wording and make it present truth. <!-- R2 R4 R6 R7 -->

## Execution Order

- T001 blocks T002–T005 (they use `agent.EffectiveModel` / `agent.ToolDefaultModel`)
- T006–T008 after Phase 1 so docs describe the shipped behavior

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `agent.ToolDefaultModel == "tool-default"` and `agent.EffectiveModel` maps only that exact string to `""`, covered by a table test
- [ ] A-002 R2: The token at each config tier and via `--model` resolves `Model == ""` with `source.Model` naming that tier, and beats a lower-tier fill
- [ ] A-003 R3: Provider-only `--model tool-default` resolves `model: ""` with `source.model: flag`
- [ ] A-004 R4: Claude `tool-default` under `dispatch.mode: native` yields no `dispatch:` key and `model_alias: ""`; `HasNativeModel` doc comment names `ToolDefaultModel` in present tense
- [ ] A-005 R5: The alias-less no-capability error mentions `tool-default`
- [ ] A-006 R6: `fab config set … model tool-default` exits 0 and the resolved profile shows `model: ""` with the key as source; the `.model` empty refusal names `tool-default`
- [ ] A-007 R7: Every tool's `fab agent models` output starts with the `reserved` row (correct label, efforts, `default: false`)

### Behavioral Correctness

- [ ] A-008 R2: Composed commands (`command`, `dispatch.command`, `fab dispatch start`'s `spawn.WithProfile`) drop the model token pair exactly as for an empty model — the literal `tool-default` never appears in a composed command
- [ ] A-009 R6: Empty-value refusal on non-model keys is byte-identical to before

### Scenario Coverage

- [ ] A-010 R4: Tests cover native-kept (claude, native) and headless-dropped (`--model` absent from dispatch command)
- [ ] A-011 R7: Tests cover labelled default, plain label on failed discovery, claude effort fallback, adapter-less `[]`, table first row, and a tool-reported `tool-default` being dropped

### Edge Cases & Error Handling

- [ ] A-012 R1: Case variants / whitespace-padded variants pass through verbatim (not collapsed)
- [ ] A-013 R7: Discovery failure still yields the reserved row first plus fills, with the warning on stderr

### Code Quality

- [ ] A-014 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [ ] A-015 No unnecessary duplication: The collapse lives in one helper reused by both resolution paths; no re-implemented precedence
- [ ] A-016 Readability over cleverness: helper and row construction are small, named, and commented in the surrounding style
- [ ] A-017 No magic strings: the token and `reserved` source are named constants, referenced everywhere (tests may use the literal to pin the contract)
- [ ] A-018 CLI reference + tests: `_cli-fab.md` documents every changed CLI behavior and each Go change ships tests (constitution)
- [ ] A-019 No fab-kit-only paths cited from deployed `src/kit/` content; sibling sweep done (specs aggregates, glossary, F3's "future token" wording)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Helper named `EffectiveModel`, constant `ToolDefaultModel` in `internal/agent` | Intake suggested these names; `internal/agent` already owns `ModelAlias`/`HasNativeModel` | S:70 R:90 A:80 D:70 |
| 2 | Confident | A tool-reported row with id `tool-default` is dropped so the reserved row stays unique | Intake says the token is reserved and never merged; dropping is the simplest uniqueness guarantee | S:55 R:85 A:70 D:65 |
| 3 | Confident | The `.model` refusal hint keys on the last dotted segment being `model` | Covers both `agent.profiles.<role>.model` and `providers.<p>.profiles.<role>.model` with one rule | S:60 R:90 A:80 D:75 |
| 4 | Confident | Advert legacy-digest bump only if the edited text is inside a digested fence segment | Intake row 10; F1 precedent | S:70 R:85 A:70 D:70 |

4 assumptions (0 certain, 4 confident, 0 tentative).
