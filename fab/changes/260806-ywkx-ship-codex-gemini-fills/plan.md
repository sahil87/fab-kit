# Plan: Ship Built-in Codex/Gemini Per-Role Fills

**Change**: 260806-ywkx-ship-codex-gemini-fills
**Intake**: `intake.md`

## Requirements

### Agent defaults: shipped non-claude per-role fills

#### R1: `defaults.yaml` SHALL carry per-role fills for codex and gemini
The embedded `src/go/fab/internal/agent/defaults.yaml` MUST add a `profiles:` map to the `codex` and
`gemini` provider entries, sparse (roles absent fall through to that provider's own `default` entry
per the j9nh precedence). The values MUST be the model IDs and effort levels **verified against the
current codex/gemini CLIs at apply time** (see `## Assumptions` rows 1–3). fab MUST continue to apply
no validation — the strings pass through verbatim.

- **GIVEN** a project with `agent.workers: codex` and no `providers:` block
- **WHEN** `fab resolve-agent apply` runs (apply ∈ `doing`)
- **THEN** it emits codex's `doing` fill — a non-empty `model=` and `effort=xhigh` — not an empty model
- **AND** `fab resolve-agent ship` (ship ∈ `fast`) emits codex's cheaper `fast` model at `low` effort,
  so role differentiation survives the provider swap

#### R2: gemini fills SHALL carry no effort
The gemini CLI has no reasoning-effort flag, so every `providers.gemini.profiles.<role>` entry MUST
set `model` only. Resolution MUST therefore emit no `effort=` line for a gemini-resolved role.

- **GIVEN** `agent.workers: gemini` and no `providers:` block
- **WHEN** `fab resolve-agent review` runs
- **THEN** `model=` carries gemini's `default` fill, `provider=gemini`, and **no** `effort=` line is emitted
- **AND** the composed `dispatch=` command drops nothing but the absent effort token

#### R3: claude's resolved defaults SHALL be byte-identical
Shipping non-claude fills MUST NOT change any resolved profile while both depth knobs are `claude`
(the shipped default). `DefaultProfile(role)` for all six roles MUST be unchanged.

- **GIVEN** an unconfigured project (both knobs at their built-in `claude`)
- **WHEN** every stage resolves
- **THEN** each stage's `{provider, model, effort}` is byte-identical to before this change
- **AND** `TestDefaultRoleProfilesArePinned` passes with its table untouched

### Fill precedence: the legacy flat fill must not be shadowed

#### R4: A user's flat `providers.<name>.model`/`.effort` SHALL outrank a BUILT-IN `profiles.default`
`providerFill` currently reads the flat fill as the **lowest** rung, below `profiles.default`. That was
safe only while no non-claude built-in carried a `profiles.default`. Shipping one would silently
shadow the pre-2.17.0 flat spelling for any config that has not yet run the `2.16.19-to-2.17.0`
migration. `ResolveProvider` MUST therefore fold an override's flat fill **into that override's
`profiles.default`** (the alias semantics the flat spelling is already documented to have) before the
built-in merge, per field.

- **GIVEN** a config carrying only `providers.codex.model: my-pinned-model` and `effort: high`
- **WHEN** `fab resolve-agent apply --provider codex` runs
- **THEN** `model=my-pinned-model` resolves — the user's pin beats the shipped built-in `profiles.default`,
  because apply (`doing`) carries no built-in codex `model` fill of its own and falls through to `default`
- **AND** `effort=xhigh` resolves — codex's built-in **role** fill (`profiles.doing.effort`) outranks the
  folded `profiles.default`, exactly as it would outrank a hand-written `profiles.default.effort`: the flat
  spelling is an alias for `profiles.default`, not a global pin over role fills
- **AND** a user's own `providers.codex.profiles.default.model` still beats their flat fill (the modern
  spelling wins over its alias)
<!-- clarified: rework cycle 1 — the original THEN claimed `effort=high` (a flat pin winning over the
     built-in ROLE fill), which contradicts the alias semantics this same requirement mandates and which
     the shipped, correct implementation follows. Scenario corrected to the alias reading; see
     resolve_agent_test.go TestResolveAgentOverrideProviderTakesFill rationale. -->

### Documentation: the refresh-each-release policy

#### R5: `docs/specs/stage-models.md` SHALL record the refresh policy and the decision lineage
The spec MUST replace its "codex and gemini ship grammar only" claim with the shipped-fill truth, and
MUST carry a short policy section stating that built-in non-claude fills are refreshed at
**kit-release cadence**, are **unvalidated pass-through** values, and are overridden by **one config
line**. It MUST record the `ho9y` → `j3cm` → this-change decision lineage.

- **GIVEN** a reader of `docs/specs/stage-models.md`
- **WHEN** they read § Three built-in providers
- **THEN** the inline-YAML sample shows all three providers' shipped fills, drift-guarded against
  `defaults.yaml`
- **AND** a policy subsection names the refresh cadence, the no-validation contract, and the
  single-line override (`providers.<name>.profiles.<role>.model`)

#### R6: Every mirror of the "grammar only / ships no model ID" claim SHALL be swept
The claim is restated across specs, the rendered config reference, and the skill CLI references. Per
`fab/project/code-quality.md` § Sibling & Mirror Sweeps the **whole class** MUST be updated in this
change: `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `docs/specs/config.md`,
`docs/specs/glossary.md`, `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md`,
`src/kit/skills/_cli-agents.md` + `docs/specs/skills/SPEC-_cli-agents.md`, and
`src/go/fab/internal/configref/configref.go` (whose rendered text reaches every user).

- **GIVEN** a repo-wide grep for `grammar only`, `rot at CLI cadence`, `ships no model ID`, `no fills`
- **WHEN** the sweep is complete
- **THEN** no surviving occurrence outside `fab/changes/` asserts that codex or gemini carries no fill
- **AND** every doc that quotes a concrete fill value is either drift-guarded or explicitly marked as
  an example

#### R7: The rendered `fab config reference` SHALL show the shipped fills, interpolated not copied
`configref.providersSegment()` MUST render codex's and gemini's real shipped fills, sourced from
`agent.ResolveProvider(nil, …).Profiles` rather than as literal strings (the package's
no-literal-copy rule), and MUST keep them **commented** (a block restating a built-in default
registers no project override). The `providers` registry row's JSON `default` MUST project the
non-claude `profiles` maps that now exist.

- **GIVEN** `fab config reference` output
- **WHEN** the codex/gemini blocks are read
- **THEN** each shows its shipped `profiles:` entries with an "override to pin a newer model" note
- **AND** the blocks still parse as **absent** from `Config` (commented), and the rendering stays
  byte-stable (roles emitted in `agent.RoleNames()` order, no map range-iteration)

### Tests

#### R8: The drift guards SHALL cover the non-claude fills
The pinned-profile guard, the embedded-defaults validation, the doc-mirror guard, and the registry
projection assertions MUST be extended so a fill bump is an explicit, test-acknowledged edit rather
than a silent one, and so the inverted "must carry no fills" assertions become "must carry fills".

- **GIVEN** an edit that changes a shipped codex or gemini fill in `defaults.yaml`
- **WHEN** `go test ./internal/agent/... ./cmd/fab/...` runs
- **THEN** a pinned-values test fails and names itself, exactly as a claude model bump does today
- **AND** the doc-mirror guard checks the codex/gemini inline-YAML samples against the resolved
  provider fills, not against claude's

#### R9: End-to-end resolution SHALL be covered for `agent.workers: codex` and `: gemini`
A test MUST exercise the flagship UX: one knob line, every pipeline stage resolving that provider's
per-role fill through the j9nh precedence chain.

- **GIVEN** `agent.workers: codex` (then `: gemini`)
- **WHEN** every stage in `agent.StageNames()` resolves
- **THEN** each Tier-2 stage carries that provider's role fill and each Tier-1 role stays on claude
- **AND** the expectations are derived from `ResolveProvider(nil, name).Profiles`, not restated literals

### Non-Goals

- **No staleness automation** — no CI check against provider APIs, no model-catalog fetch. The policy
  is documentation-only (intake assumption 5; fab stays validation-free).
- **No config seeding** — fills ship as embedded `defaults.yaml` data, never written into a user's
  `config.yaml` (intake assumption 1).
- **No CLI, schema, or migration change** — the addition is additive data on an existing shape.
- **No memory-file edits** — `docs/memory/runtime/providers-and-profiles.md` and
  `docs/memory/_shared/configuration.md` are the intake's **Affected Memory** list, which is hydrate's
  input. Apply writes code and specs; hydrate writes memory.

### Design Decisions

#### Gemini fills use the CLI's stable model ALIASES, not versioned IDs
**Decision**: `providers.gemini.profiles` ships `default: { model: pro }` and `fast: { model: flash }` —
the gemini CLI's own stable aliases — rather than a dated ID such as `gemini-3-pro-preview`.
**Why**: `resolveModel()` in gemini-cli maps `pro`/`flash`/`flash-lite`/`auto` to whatever the current
best model is for the caller's entitlement, and downgrades gracefully when the user lacks preview
access. That makes the fill **rot-immune**, which is the exact failure mode the ho9y/j3cm no-fills
stance was defending against — so it strengthens the reversal rather than merely accepting its cost.
**Rejected**: `gemini-2.5-pro`/`gemini-2.5-flash` (the intake's placeholders) — the 2.5 line is
scheduled for shutdown; `gemini-3-pro-preview` / `gemini-3.1-pro-preview` — preview-suffixed IDs churn
faster than kit releases and would need a bump every few weeks.
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills

#### Codex fills are pinned to concrete slugs from the installed CLI's live model catalog
**Decision**: `default: { model: gpt-5.6-sol, effort: high }`, `doing`/`review` at `effort: xhigh`,
`fast: { model: gpt-5.6-luna, effort: low }`.
**Why**: the codex CLI exposes no alias mechanism — `-m` takes a slug — so concrete IDs are the only
option. They were read from the installed `codex-cli` 0.146.0's own model catalog rather than from
documentation, which is the closest thing to an authoritative source: `gpt-5.6-sol` is the
priority-1 "latest frontier agentic coding model" and `gpt-5.6-luna` the "fast and affordable" one,
and both list `xhigh`/`low` among their supported reasoning levels.
**Rejected**: `gpt-5.3-codex` / `gpt-5.3-codex-mini` (the intake's placeholders) — absent from the
current catalog entirely; `gpt-5.4` / `gpt-5.4-mini` — present but carrying explicit deprecation
notices pointing at the 5.6 line.
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills

#### The legacy flat fill becomes a real alias for `profiles.default`
**Decision**: `ResolveProvider` folds an override's flat `model`/`effort` into that override's
`profiles.default` (per field) before merging the built-in table, instead of `providerFill` reading
the flat value as a rung *below* `profiles.default`.
**Why**: the flat spelling is **documented** as an alias for `profiles.default`, but was implemented as
a lower-precedence rung. That was indistinguishable from the alias while no non-claude built-in
carried a `profiles.default`; shipping one makes the difference load-bearing, and the rung form would
silently shadow a pre-migration user's pinned model with fab-kit's shipped one.
**Rejected**: leaving the rung and accepting the shadowing (a silent regression for exactly the
un-migrated configs the alias exists to serve); special-casing "built-in vs user" inside
`providerFill` (it receives an already-merged `ProviderConfig` and cannot tell them apart).
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills

#### The doc-mirror drift guard becomes provider-aware
**Decision**: `TestMirrorDocsMatchDefaultProfiles` tracks the enclosing `providers.<name>:` key and
compares each role-fill line against `ResolveProvider(nil, name).Profiles[role]`, with the `effort`
half of the pattern optional.
**Why**: the guard is shape-driven, so a live codex fill line written into `stage-models.md` would
otherwise be matched as a *claude* role fill and fail spuriously; and gemini's effort-less fills would
not match at all, leaving them unguarded. Provider-awareness is a strict generalization — for claude,
`ResolveProvider(nil, "claude").Profiles[role]` is exactly what `DefaultProfile(role)` returns.
**Rejected**: exempting the codex/gemini samples with `# example` markers (leaves the newly-shipped
values as the only unguarded mirror in the tree, which is what the guard exists to prevent).
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills

## Tasks

### Phase 1: Data + resolution

- [x] T001 Add sparse `profiles:` maps to the `codex` and `gemini` entries of `src/go/fab/internal/agent/defaults.yaml` (codex: `default`/`doing`/`review`/`fast`; gemini: `default`/`fast`, model-only), and rewrite the file's `providers` header comment so it no longer claims the non-claude built-ins are grammar-only <!-- R1 R2 -->
- [x] T002 Fix the flat-fill precedence in `src/go/fab/internal/agent/agent.go`: fold an override's flat `Model`/`Effort` into its `profiles.default` inside `ResolveProvider` (per field, modern spelling wins), and update the `providerFill` doc comment to describe the alias rather than a bottom rung <!-- R4 -->
- [x] T003 Update the package doc and the `defaultProviders` / `DefaultCodex*` / `DefaultGemini*` var comments in `src/go/fab/internal/agent/agent.go` — the non-claude built-ins are no longer grammar-only, and the refresh-at-kit-cadence policy replaces the "model IDs rot in weeks" rationale <!-- R1 R5 -->

### Phase 2: Go tests

- [x] T004 Invert the grammar-only assertions in `src/go/fab/internal/agent/defaults_test.go`: `TestDefaultsFileProviders` must assert codex/gemini now carry `profiles`, that gemini's fills carry no effort, and that no built-in uses the deprecated flat fill <!-- R8 -->
- [x] T005 Extend `TestDefaultRoleProfilesArePinned` in `src/go/fab/internal/agent/agent_test.go` with a pinned table for the codex/gemini provider fills, so a bump is a deliberate two-place edit <!-- R8 -->
- [x] T006 Update the empty-resolution tests in `src/go/fab/internal/agent/agent_test.go` — `TestKnobWithoutFillsResolvesEmpty`, the `TestResolveRoleWithOverrides` gemini-swap case, and `TestResolveProvider_BuiltInCodexAndGemini` — to assert the shipped fills instead of emptiness <!-- R1 R2 R8 -->
- [x] T007 Add `TestFlatProviderFillBeatsBuiltInDefault` to `src/go/fab/internal/agent/agent_test.go` covering R4 both ways (flat fill wins over the built-in; a user's own `profiles.default` wins over their flat fill) <!-- R4 -->
- [x] T008 Add the end-to-end knob test to `src/go/fab/internal/agent/agent_test.go`: `agent.workers: codex` and `: gemini` resolve every Tier-2 stage to that provider's role fill while Tier-1 roles stay on claude, with expectations derived from `ResolveProvider` <!-- R9 -->
- [x] T009 Make `TestMirrorDocsMatchDefaultProfiles` in `src/go/fab/internal/agent/defaultprofiles_mirrors_test.go` provider-aware (track the enclosing provider key; make the `effort` half optional; raise the coverage backstop accordingly) <!-- R8 -->

### Phase 3: Rendered reference + its tests

- [x] T010 Render the shipped codex/gemini fills in `configref.providersSegment()` (`src/go/fab/internal/configref/configref.go`), interpolated from `agent.ResolveProvider(nil, …).Profiles` in `agent.RoleNames()` order, kept commented, with an "override to pin a newer model" note; update the providers-row `Description` and the package-doc empty-default paragraph <!-- R7 -->
- [x] T011 Update `src/go/fab/cmd/fab/config_test.go`: the registry-default assertions must require non-claude `profiles` instead of forbidding them, and the row description phrases must match the new wording <!-- R7 R8 -->
- [x] T012 [P] Update `TestResolveAgentOverrideProviderNoFill` in `src/go/fab/cmd/fab/resolve_agent_test.go` — `--provider codex` on a bare config now resolves codex's shipped fill; verify `TestResolveAgentOverrideProviderTakesFill` (the flat-fill guard for R4) still passes unchanged <!-- R1 R4 -->
- [x] T013 [P] Update the bare-invocation tests in `src/go/fab/cmd/fab/agent_test.go` (`TestAgentPrintBuiltinCodexNoFill`, `TestAgentProviderBuiltinCodexNoConfig`, and the gemini append-mode case) to expect the shipped fills <!-- R1 R2 -->

### Phase 4: Specs + skill references (the mirror sweep)

- [x] T014 Rewrite `docs/specs/stage-models.md` § Three built-in providers to show all three providers' shipped fills, and add the **refresh-each-release policy** subsection (kit cadence, unvalidated pass-through, one-line override, `ho9y` → `j3cm` → `ywkx` lineage); update the § Out of scope "Non-claude default fills" line <!-- R5 R6 -->
- [x] T015 [P] Update `docs/specs/config.md` § Default semantics (the codex/gemini `profiles` are now projected, not omitted) and `docs/specs/architecture.md`'s config-fence sample prose <!-- R6 R7 -->
- [x] T016 [P] Update `src/kit/skills/_cli-fab.md` (§ fab resolve-agent examples and the `providers:` description) and mirror the change into `docs/specs/skills/SPEC-_cli-fab.md` <!-- R6 -->
- [x] T017 [P] Update `src/kit/skills/_cli-agents.md` and `docs/specs/skills/SPEC-_cli-agents.md` so their grammar-only framing distinguishes the markdown dictionary's no-model-IDs rule (unchanged) from the binary's now-shipped fills; check `docs/specs/glossary.md`'s `providers` row and `src/kit/migrations/2.16.19-to-2.17.0.md` <!-- R6 -->
- [x] T018 Run the repo-wide sweep grep (`grammar only`, `rot at CLI cadence`, `rot in weeks`, `ships no model ID`, `no fills`, `gpt-5.3-codex`, `gemini-2.5-`) outside `fab/changes/` and close every surviving occurrence <!-- R6 -->

### Phase 5: Verification

- [x] T019 Run `go test ./...` from `src/go/fab` and fix every failure <!-- R1 R2 R3 R4 R7 R8 R9 -->

### Phase 6: Rework (cycle 1 — review findings)

- [x] T020 Sweep the flat-fill ALIAS semantics into `src/go/fab/internal/config/config.go` (the `ProviderConfig` type doc's fill-precedence ladder at ~:86 and the `Model`/`Effort` field prose at ~:103-104 still describe the pre-change "rung BELOW profiles.default" reading) and the same stale claim in `src/go/fab/internal/config/config_test.go:~497-499` — both must state the fold-into-`profiles.default` alias semantics shipped by T002 <!-- R4 R6 -->
- [x] T021 Fix the activation warning in `src/kit/migrations/2.16.19-to-2.17.0.md` (~:35-44, ~:137-152): (a) correct the rule to "the `default` role, plus every role with no shipped `profiles.<role>` entry" (claude ships an exhaustive map, so no claude role changes; the current wording contradicts its own parenthetical); (b) state the codex/gemini upgrade impact — under the pre-2.17.0 binary those providers shipped no fills, so a flat `providers.codex.model` was active for EVERY role, and after upgrade it stops applying to `doing`/`review`/`fast` where fab-kit now ships a role fill (the user-visible loss the warning must name) <!-- R6 -->
- [x] T022 Make `TestConfigReferenceProvidersMatchBuiltins` in `src/go/fab/cmd/fab/config_test.go` (~:516-588) compare `entry["profiles"]` against `p.Profiles` verbatim (model AND effort per role), so the doc comment's "per-role fill map cross over verbatim" claim is actually enforced <!-- R7 R8 -->
- [x] T023 De-literal the two prose mentions of `gpt-5.6-sol`/`gpt-5.6-luna` in `docs/specs/stage-models.md` (~:247, ~:266) — reword to role-relative phrasing (e.g. "the `default` fill" / "the cheaper `fast` model") or mark explicitly as examples, so A-017 holds; add one clause (defaults.yaml gemini comment + stage-models § gemini fills) noting the `pro`/`flash` alias mechanism has a gemini-CLI version floor (older CLIs require full versioned IDs) <!-- R5 R6 -->
- [x] T024 Delete the unreachable trailing `prov.Model`/`prov.Effort` rung in `providerFill` (`src/go/fab/internal/agent/agent.go:~464-474`, per plan § Deletion Candidates item 1): collapse to the role→default two-rung read and correct its doc comment (drop the impossible hand-built-`ProviderConfig` caller justification) <!-- R4 -->
- [x] T025 Re-run `go test ./...`, `go vet ./...`, and `gofmt -l` from `src/go/fab` and fix every failure <!-- R4 R6 R7 R8 -->

### Phase 7: Rework (cycle 2 — re-review findings)

- [x] T026 Fix the two stale `Model discovery` rows in `src/kit/skills/_cli-agents.md` (~:155 codex, ~:167 gemini): delete the claims `fab ships no codex model ID and does not validate one` / `fab ships no gemini model ID` — fab now SHIPS fills for both (codex `default`/`doing`/`review`/`fast`; gemini `default`/`fast` as CLI aliases `pro`/`flash`) — and replace the deprecated flat-fill pinning advice (`providers.codex.model` / `providers.gemini.model`) with the modern spelling `providers.<name>.profiles.<role>.model` (a flat fill is an alias for `profiles.default` and is OUTRANKED by the shipped role fills, so the old advice silently fails for `doing`/`review`/`fast`). After editing, re-verify `docs/specs/skills/SPEC-_cli-agents.md` — it does not currently restate these rows, so confirm it needs no matching edit (or make one if the edit changes something it does restate) <!-- rework: R6 sweep miss — per-provider literal variants escaped T018's grep -->
- [x] T027 Fix the overbroad clause in `src/kit/skills/_cli-agents.md` ~:131: `All three carry per-role fills too, so naming one resolves a real model for every role rather than an empty one` is FALSE for the `fab agent --provider` path (which bypasses both fill sources — this same file's caveat at ~:68). Scope the clause to the fill-consuming paths (a depth knob, `agent.profiles.<role>.provider`, `fab resolve-agent --provider`) and explicitly exclude `fab agent --provider` <!-- rework: contradicts the :68 caveat; verified `fab agent --provider codex --print` prints bare `codex` -->
- [x] T028 Resolve the prose/render asymmetry in `src/go/fab/internal/configref/configref.go` (~:628-631 vs ~:655-703): the prose says all three built-ins ship per-role fills but the rendered block shows a `profiles:` map only under codex/gemini. Add one clause to the prose (or the claude block comment) noting claude's six fills are projected in `fab config reference --json` / resolved by `fab resolve-agent`, OR render claude's map commented like its `dispatch_command` — pick the smaller diff that keeps `Render()` byte-stable across two calls and updates the affected config tests <!-- rework: should-fix — reader of the rendered reference can conclude claude ships no fills -->
- [x] T029 Re-run the R6 sweep with the literal patterns that escaped last time — case-insensitive `ships no [a-z]* model ID`, `no (codex|gemini) model`, and flat-fill pinning advice `providers\.(codex|gemini)\.model` — outside `fab/changes/`, `docs/memory/` (hydrate's), and frozen historical migrations; close every surviving occurrence. Then run `go test ./...`, `go vet ./...`, `gofmt -l` from `src/go/fab` <!-- rework: T018's pattern set proved too narrow; user-facing string literals are the recurring sweep gap -->

### Phase 8: Rework (cycle 3 — review #3 findings)

- [x] T030 Close the R6 sweep miss in `docs/specs/stage-models.md` § Harness-adapter boundary (~:723, ~:726-727): reword the `*Claude-flavored data (overridable):*` bullet so it says the CLAUDE fills use Claude model IDs while codex/gemini ship their own (per § Three built-in providers), and rewrite the v1-scope bullet `No non-claude default fills (that is ywkx)` to drop the deferral — this change IS ywkx and shipped them — while keeping the still-true half (`no provider-detection, no non-Claude integration test`). Then grep the tree (outside `fab/changes/`, `docs/memory/`, frozen migrations) for remaining variants of THIS phrasing class: `non-claude default fills`, `fills use Claude model IDs`, `that is .?ywkx` — and close any survivor <!-- rework: R6 — the spec's own later section named this change as future work -->
- [x] T031 Add drift-guard coverage for the rendered codex/gemini fill lines in `fab config reference` (the one user-facing surface this change added with no test): in `TestConfigReferenceDocumentsProviderFill` (`src/go/fab/cmd/fab/config_test.go`), assert `configref.Render()` output contains one commented `<role>: { ... }` line per shipped codex/gemini fill, DERIVED from `agent.ResolveProvider(nil, name).Profiles` with the same omitempty shaping the renderer uses (so gemini's model-only and codex's effort-only rows are both pinned; no literal model strings in the test). Verify the guard actually bites: temporarily nil-swap the `profilesLines(...)` call sites in `src/go/fab/internal/configref/configref.go` (~:663, ~:667) and confirm the new assertion fails, then restore. Re-verify A-020 <!-- rework: reviewer proved the rendered fills can be deleted with the suite staying green -->
- [x] T032 Fix the overclaim in `src/kit/migrations/2.16.19-to-2.17.0.md` Verification step 5 (~:186): the rewrite is NOT resolver-neutral in a mixed cascade (a system-layer `profiles.default` vs a project-layer flat fill — rewriting only the project layer flips the winner; the inversion itself is pre-existing). Either scope the claim to "unchanged when both layers are rewritten together" or name the mixed-layer exception alongside the two warnings the migration already documents <!-- rework: should-fix — false claim in one reachable shape -->
- [x] T033 One-line mirror fix in `docs/specs/skills/SPEC-_cli-agents.md` (~:80): the Key Properties row `| Records model IDs? | **No** — discovery recipes only, by design |` is the file's only unqualified restatement of the claim this change qualified everywhere else — reword per reviewer, e.g. `**No catalogs** — discovery recipes only; the binary's shipped fills are named, not enumerated`. Then run `go test ./...`, `go vet ./...`, `gofmt -l` from `src/go/fab` <!-- rework: mirror-class consistency -->

## Execution Order

- T001 → T002 → T003 (data first; the resolution fix is what makes T001 safe for un-migrated configs)
- T004–T009 depend on T001–T002
- T010 depends on T001; T011 depends on T010
- T012, T013 depend on T001–T002 and are independent of each other
- T014–T018 depend on T001 (the values they document) and are independent of the Go tests
- T019 last
- Phase 6 (rework): T020–T024 are independent of each other; T025 last
- Phase 7 (rework cycle 2): T026–T028 independent; T029 last
- Phase 8 (rework cycle 3): T030–T032 independent; T033 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/go/fab/internal/agent/defaults.yaml` carries sparse `profiles:` maps on both `codex` and `gemini`, with values verified against the current CLIs at apply time
- [x] A-002 R2: every `providers.gemini.profiles.<role>` entry sets `model` only — no `effort` key anywhere in the gemini block
- [x] A-003 R3: `TestDefaultRoleProfilesArePinned` passes with its claude table unmodified, and every stage's resolved profile is unchanged for an unconfigured project
- [x] A-004 R4: `ResolveProvider` folds an override's flat fill into its `profiles.default`, so a pre-migration flat pin wins wherever a role resolves through `default` (a built-in **role** fill still outranks it, per R4's corrected alias semantics)
- [x] A-005 R5: `docs/specs/stage-models.md` carries a refresh-policy subsection naming the kit-release cadence, the no-validation contract, the one-line override, and the ho9y → j3cm → ywkx lineage
- [x] A-006 R7: `fab config reference` renders the shipped codex/gemini fills, sourced from `agent.ResolveProvider` rather than from literals <!-- verified manually at review cycle 3: the rendered block emits codex's four and gemini's two fill lines, commented, in agent.RoleNames() order (default, doing, fast, review). The rendering is CORRECT; what is missing is the automated guard — see A-020 -->


### Behavioral Correctness

- [x] A-007 R1: with `agent.workers: codex` and no `providers:` block, `apply`/`review` resolve `effort=xhigh` and `ship` resolves the cheaper `fast` model at `low` — role differentiation survives the swap
- [x] A-008 R2: with `agent.workers: gemini`, resolution emits no `effort=` line for any role
- [x] A-009 R4: a user's own `providers.<name>.profiles.default` still beats their flat `providers.<name>.model` (the modern spelling wins over its alias)
- [x] A-010 R7: the rendered codex/gemini reference blocks still parse as **absent** from `Config`, and `Render()` output is byte-stable across two calls

### Scenario Coverage

- [x] A-011 R9: a test exercises `agent.workers: codex` and `: gemini` across every stage in `agent.StageNames()`, with expectations derived from `ResolveProvider` rather than restated
- [x] A-012 R8: `TestMirrorDocsMatchDefaultProfiles` checks the codex and gemini inline-YAML samples in `docs/specs/stage-models.md` against the resolved provider fills
- [x] A-013 R4: a test covers the flat-fill-vs-built-in-default precedence in both directions

### Edge Cases & Error Handling

- [x] A-014 R2: gemini's effort-less fills do not cause an empty `effort=` line, a stray `--effort` token, or a `{effort}` placeholder to appear in any composed command
- [x] A-015 R8: a fill bump in `defaults.yaml` fails a test that names itself in the output (the deliberate-change pin), for the non-claude providers as it already does for claude

### Code Quality

- [x] A-016 Pattern consistency: new YAML rows, test tables, and doc prose follow the shapes established by j9nh/2j2i (sparse per-role maps, derived expectations, one pinned table)
- [x] A-017 No unnecessary duplication: no model ID is written as a literal outside `defaults.yaml` and the single pinned test table; rendered docs interpolate from Go symbols <!-- verified rework cycle 1: the `gpt-5.6-sol`/`gpt-5.6-luna` prose quotes are gone (stage-models.md:246-249, 271-274 now read role-relative); the only surviving occurrences are stage-models.md:233,236 inside the drift-guarded `role: { model: … }` sample, `_cli-fab.md`'s `<codex-model-id>` placeholders, and configref's interpolation from `ResolveProvider`. Residual nice-to-have: `pro`/`flash` at stage-models.md:255 are prose literals outside the guarded shape -->

- [x] A-018 Canonical source only: every kit edit lands in `src/kit/`, none under `.claude/skills/`
- [x] A-019 SPEC-mirror sync: every `src/kit/skills/*.md` edit carries its `docs/specs/skills/SPEC-*.md` update
- [x] A-020 Go changes ship tests: every behavior change in `internal/agent` and `internal/configref` is covered by an updated or new test in the same change <!-- MET (review cycle 4, closed by T031): the gap cycle 3 found — `configref.profilesLines`'s output into the RENDERED providers segment — is now guarded by TestConfigReferenceDocumentsProviderFill (config_test.go:438-471), which derives one expected `  #     <role>: { … }` line per shipped codex/gemini fill from agent.ResolveProvider with the renderer's own omitempty shaping. Re-verified the guard BITES by repeating cycle 3's mutation: replacing both `profilesLines(providers["codex"|"gemini"].Profiles, roleOrder)` call sites with `profilesLines(nil, roleOrder)` now fails with all six expected fill lines named (codex default/doing/fast/review, gemini default/fast); restored, suite green. -->
- [x] A-021 Migrations: no user-data restructuring is introduced, so no `src/kit/migrations/` file is required (additive embedded data only)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/internal/agent/agent.go:544-545` (the `mergeField(&resolved.Model, …)` / `mergeField(&resolved.Effort, …)` pair in `ResolveProvider`) — with the flat fill folded into `Profiles[RoleDefault]` by `withFlatFillAlias`, the resolved struct's own flat fields have NO production reader. Re-verified this cycle by grepping every `ResolveProvider` call site (`agent.go:419`, `cmd/fab/{resolve_agent,agent,operator,dispatch_start}.go`, `internal/spawn/spawn.go`, `internal/configref/configref.go:249`): each reads only `SessionCommand`, `DispatchCommand`, or `Profiles`. `providerFill` stopped consulting them at T024 and `configref.providerDefault` deliberately refuses to project them; the only remaining reads are the *never-set* assertions in `agent_test.go` / `defaults_test.go`, which cover the nil-config case and would still hold. Keep only if the resolved `ProviderConfig` is meant to round-trip the user's literal spelling.
- `src/go/fab/internal/configref/configref.go:700-702` (the `len(set) == 0 → continue` guard in `profilesLines`) — a `providerProfileDefault` carrying neither `model` nor `effort` cannot come from `defaults.yaml`: every shipped fill sets at least one field, and `defaults_test.go:93-96` pins `profiles.default.model` non-empty per provider. A dead branch rather than a real skip case.
- `src/go/fab/internal/configref/configref.go:680-682` (the `len(profiles) == 0 → ""` early return in `profilesLines`) — **weaker candidate than last cycle's phrasing claimed**: it IS reachable, but not for the reason its comment gives. `defaults_test.go` now forbids a fill-less built-in, so the stated case ("a future grammar-only built-in") cannot occur; what actually reaches the branch is a *drift* case — `providersSegment` looks the maps up by the hardcoded string keys `"codex"`/`"gemini"`, so renaming either provider in `defaults.yaml` yields a zero `providerDefault` and an empty map here. Correct fix is to either delete the branch and key the lookup off a canonical symbol, or keep it and rewrite the comment to name the drift case.
- `docs/specs/stage-models.md:841` § Out of scope, the "Non-claude default fills — *no longer deferred*" bullet — an out-of-scope list carrying an in-scope item; the two live pointers (§ Three built-in providers, § Refreshing the non-claude fills) make the bullet redundant.
- The hand-written omitempty fill shaping (`if fill.Model != "" { … }` / `if fill.Effort != "" { … }`) now exists in FOUR places: `configref.profilesLines` (production, `configref.go:694-702`) plus three test re-derivations — `cmd/fab/config_test.go:452-462` (rendered lines), `cmd/fab/config_test.go:637-648` (JSON projection), `internal/configupgrade/configupgrade_test.go:378-388` (equals-default fixture). The three test copies are deliberately independent of the renderer (a guard that called it would be tautological), but they are identical to each other, so one exported test helper on `internal/agent` would retire two of the three without weakening any guard.

Nothing else: the change is additive data plus a precedence correction, and `withFlatFillAlias` and `profilesLines` are both single-purpose with live call sites. (The trailing `prov.Model`/`prov.Effort` rung inside `providerFill`, carried as the first candidate of an earlier cycle, was deleted by T024 and is no longer a candidate.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Codex fills are `default: {gpt-5.6-sol, high}`, `doing`/`review: {effort: xhigh}`, `fast: {gpt-5.6-luna, low}` | VERIFIED at apply time against the installed `codex-cli` 0.146.0's own model catalog (`~/.codex/models_cache.json`, fetched 2026-08-07): `gpt-5.6-sol` is the priority-1 "latest frontier agentic coding model", `gpt-5.6-luna` the "fast and affordable" one, and both list `xhigh` and `low` among supported reasoning levels. The intake's `gpt-5.3-codex`/`gpt-5.3-codex-mini` placeholders are absent from the catalog entirely | S:90 R:85 A:90 D:85 |
| 2 | Confident | Gemini fills use the CLI's stable aliases `pro` and `flash` rather than versioned IDs | VERIFIED against gemini-cli's `packages/core/src/config/models.ts`: `resolveModel()` maps `auto`/`pro`/`flash`/`flash-lite` to the current best model for the caller's entitlement and downgrades gracefully without preview access. Rot-immune, which is precisely the risk the ho9y/j3cm stance was defending against. The intake's `gemini-2.5-*` placeholders are the legacy fallback line (2.5 Pro is scheduled for shutdown); `gemini-3-pro-preview` churns faster than kit releases. See § Design Decisions | S:75 R:85 A:80 D:70 |
| 3 | Confident | The codex catalog read is per-account and per-installed-version, so a user without `gpt-5.6` entitlement will fail loudly at the codex CLI | Accepted by intake assumption 2's own reasoning (loud, cheap failure with a one-line config override) — and it is the strictly better failure than the silent role-differentiation loss the empty-model alternative produces | S:70 R:85 A:75 D:75 |
| 4 | Confident | The flat-fill precedence fix (R4) is in scope for this change rather than a follow-up | Shipping a built-in `profiles.default` is what makes the existing rung-vs-alias discrepancy load-bearing; leaving it would ship a silent regression for un-migrated configs *created by this change*. Fixing the root cause here is the constitution's "fix root causes, not symptoms" reading, and an existing test (`TestResolveAgentOverrideProviderTakesFill`) already pins the desired behavior | S:65 R:75 A:85 D:75 |
| 5 | Confident | Memory files (`runtime/providers-and-profiles`, `_shared/configuration`) are left to hydrate, not edited during apply | The intake lists them under **Affected Memory**, which is hydrate's input by pipeline design; apply writes code and specs. The code-quality sweep class is applied in full to the spec/skill mirrors, which apply does own | S:80 R:90 A:85 D:70 |
| 6 | Confident | The rendered reference keeps the codex/gemini blocks COMMENTED even though they now carry real shipped fills | Unchanged from j3cm: a commented block registers no project override, and presence=intent holds for behavior — a built-in is inert until a knob, role override, or flag names it. Rendering them live would write an accidental override into every project | S:75 R:80 A:90 D:80 |

| 7 | Confident | The gemini alias caveat (T023) names a **version floor** without naming a version number | The mechanism is established by assumption 2's source read — `resolveModel()` maps `pro`/`flash` CLIENT-side, so a CLI predating it sends the alias as a model ID and fails there. The floor's exact version is NOT verifiable here (no gemini CLI installed in this worktree), and pinning a number we cannot check is worse than the mechanism statement, which is what a reader acts on. Failure mode and fix are identical to a stale slug (loud, one config line) | S:75 R:95 A:60 D:80 |

| 8 | Confident | T028 is resolved by the **prose clause**, not by rendering claude's `profiles:` map | The task offered both and asked for the smaller diff that keeps `Render()` byte-stable. The clause is 4 comment lines; rendering claude's six fills would add ~8 live-value lines to every project's fence, duplicating the same values `agent.profiles`'s registry Default already projects and enlarging the byte-stable surface a fill bump must move. The clause is also the accurate statement — `providerDefaults()` walks *all* providers, so `--json` really does project claude's map | S:80 R:90 A:85 D:80 |
| 9 | Confident | `fab/project/config.yaml`'s managed fence is OUT of T029's sweep scope despite carrying the retired `GRAMMAR ONLY` / `fab ships no model ID` text | That fence is *generated output* of the INSTALLED binary (stamped `kit 2.16.19`) and is overwritten wholesale by `fab config upgrade`; it is one release behind on every j9nh field too (it still renders `agent.tiers`), not just on this claim. Hand-editing it would be reverted at the next upgrade and would not reach any user. The generating source — `internal/configref` — is what this change fixes, and it is fixed | S:85 R:95 A:80 D:85 |

| 10 | Confident | T030's "close any survivor" sweep also rewrites the § Drift guard sentence at `docs/specs/stage-models.md:822-823` ("no non-claude built-in carries a fill"), which none of the task's three grep terms match | It is the same claim class — a Claude-only-fills statement this change falsified — and it escaped the reviewer's greps only because the phrase is LINE-WRAPPED (`no\nnon-claude built-in carries a fill`), so a line-based grep cannot see it. It also describes `TestDefaultsFileProviders`, which now asserts the opposite (all three built-ins carry fills, `profiles.default.model` non-empty per provider, none using the flat spelling), making it a live drift-guard inaccuracy rather than a wording nit. Leaving it would hand review the same finding a fourth time | S:85 R:90 A:85 D:80 |

10 assumptions (1 certain, 9 confident, 0 tentative).
