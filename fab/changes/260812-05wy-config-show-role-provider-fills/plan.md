# Plan: Config Show Honors Per-Role Provider Overrides in Derived Fills

**Change**: 260812-05wy-config-show-role-provider-fills
**Intake**: `intake.md`

## Requirements

### Config Read Model: Derived `agent.profiles` fills honor per-role provider overrides

#### R1: Model/effort fills derive from the overridden provider

`liveRoleProfiles` (`src/go/fab/internal/configref/defaultsmap.go`) MUST derive each role's `model`/`effort` default leaves from a resolution that KEEPS the per-role `provider` override while CLEARING the per-role `model`/`effort` overrides (a `fillsOnly`-style config reduction, applied identically to `agent.profiles` and the legacy `agent.tiers` spelling).

- **GIVEN** a config with `agent.session: claude` and `agent.profiles.operator.provider: codex` (no per-role model/effort)
- **WHEN** `fab config show` composes the effective view
- **THEN** the `agent.profiles.operator` row reads `{provider: codex, model: gpt-5.6-luna, effort: medium}` (codex's operator fill), not the chimera `{provider: codex, model: claude-sonnet-5, effort: medium}`
- **AND** `fab config show agent.profiles.operator.model --origin` shows `gpt-5.6-luna # default (effective)`

#### R2: The provider leaf's own default stays knobs-only-resolved

The derived `provider` leaf MUST keep resolving against the knobs-only config (per-role entries fully stripped), so keyed `--origin` provenance stays truthful — the user's provider override genuinely shadows the knob-derived built-in.

- **GIVEN** the same config as R1
- **WHEN** `fab config show agent.profiles.operator.provider --origin`
- **THEN** output lists `codex # system … (effective)` and `claude # default (shadowed)`

#### R3: The no-echo invariant holds for per-role model/effort overrides

A user's per-role `model`/`effort` override MUST never appear in the defaults tier. With a fully pinned override (`{provider: pinned-provider, model: pinned-model, effort: pinned-effort}`), the derived model/effort defaults come from `pinned-provider`'s own fills (empty for a provider fab ships no fills for — the leaf then falls through under empty-skip and renders absent, matching today's kimi-role rendering), and never equal the pinned values.

- **GIVEN** `agent.profiles.review: {provider: pinned-provider, model: pinned-model, effort: pinned-effort}`
- **WHEN** `DefaultsMapFor(cfg)` composes the defaults tier
- **THEN** the derived `review` row's model/effort are NOT `pinned-model`/`pinned-effort` (they derive from `pinned-provider`'s empty fills), and the derived provider leaf is the knob's provider
- **AND** the existing knob-survival behavior is unchanged (a `workers: codex` knob still governs derived rows for roles without overrides)

#### R4: Spec prose reflects the new composition

The knob-aware drill-down paragraph in `docs/specs/config.md` (§ Six intent-grouped verbs, ~line 500) SHALL be amended: per-role **provider** overrides feed fills derivation; only per-role **model/effort** overrides are stripped. (Memory files `_shared/configuration.md` and `runtime/providers-and-profiles.md` are hydrate's job, not apply's.)

- **GIVEN** the amended spec
- **WHEN** grepping the repo for claims that ALL `agent.profiles`/`agent.tiers` entries are "stripped first"
- **THEN** no spec restatement contradicts the implemented composition (`src/kit/skills/_cli-fab.md` verified at intake to carry no restatement)

### Non-Goals

- No change to runtime resolution (`agent.ResolveRole`, `fab resolve-agent`) — already correct.
- No change to the loader (`internal/config`) — the defaults tier remains read-model-only.
- No new config keys, flags, or migration.

### Design Decisions

#### Two-resolution split in liveRoleProfiles

**Decision**: `liveRoleProfiles` resolves each role twice — once against `knobsOnly(cfg)` for the `provider` leaf, once against a new `fillsOnly(cfg)` (per-role entries reduced to `{provider}` only, model/effort cleared, both `Profiles` and `Tiers` maps, copied without mutating the original) for the `model`/`effort` leaves.
**Why**: The two leaves have different honesty requirements — provider provenance must show what the override shadows (the knob's provider), while model/effort must show the built-in fill *given* the chosen provider. A single resolution cannot serve both; the split keeps each leaf's semantics locally obvious and reuses `agent.ResolveRole` unchanged.
**Rejected**: Post-hoc patching of the knobs-only result with a direct provider-fill lookup — bypasses `ResolveRole`'s documented fill precedence (per-role fill → provider default fill) and would duplicate it in configref.
*Introduced by*: 260812-05wy-config-show-role-provider-fills

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `fillsOnly` helper and the second `agent.ResolveRole` call in `liveRoleProfiles`; take `provider` from the knobs-only resolution and `model`/`effort` from the fills-only resolution. Update the `DefaultsMapFor`, `liveRoleProfiles`, and `knobsOnly` doc comments to state the extended honesty argument (per-role provider overrides feed fills; only per-role model/effort are stripped). File: `src/go/fab/internal/configref/defaultsmap.go` <!-- R1 R2 R3 -->

### Phase 3: Integration & Edge Cases

- [x] T002 Update `src/go/fab/internal/configref/defaultsmap_test.go`: refine `TestDefaultsMapFor_IgnoresUserRoleOverrides` (model/effort expectations change — with the pinned provider kept, they derive from `pinned-provider`'s empty fills, asserting they never echo `pinned-model`/`pinned-effort`; provider-leaf and knob-survival assertions unchanged) and add a chimera regression test (knob `session: claude` + `operator.provider: codex` → derived operator model/effort are codex's fills, derived provider leaf is claude; cover the `agent.tiers` spelling). Run `go test ./src/go/fab/internal/configref/`. <!-- R1 R3 -->
- [x] T003 Extend `src/go/fab/cmd/fab/config_show_init_test.go` alongside `TestConfigShow_ComposesDerivedDefaultsAgainstLiveKnobs` / `TestConfigShowOrigin_DrillDownIsKnobAware`: end-to-end case with a per-role provider override — composed view carries the overridden provider's fills (no chimera), keyed `--origin` shows the R2 provenance split. Run `go test ./src/go/fab/cmd/fab/`. <!-- R1 R2 -->

### Phase 4: Polish

- [x] T004 [P] Amend the knob-aware drill-down paragraph in `docs/specs/config.md` (~line 500) per R4. <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: With a per-role provider override, the composed `agent.profiles.<role>` row carries the overridden provider's model/effort fills — verified by the new unit regression test and the end-to-end `config show` test
- [x] A-002 R2: Keyed `--origin` on the overridden role's `provider` leaf lists the override tier as `(effective)` and the knob-derived built-in as `default (shadowed)`
- [x] A-003 R3: `TestDefaultsMapFor_IgnoresUserRoleOverrides` still asserts pinned model/effort values never appear in the defaults tier, with expectations updated to the fills-only semantics; knob-survival assertion untouched
- [x] A-004 R4: `docs/specs/config.md` knob-aware paragraph states the provider-override-fed fills derivation; repo grep finds no spec/skill prose contradicting it

### Behavioral Correctness

- [x] A-005 R1: `fab config show` and `fab resolve-agent <role>` agree on model/effort for a role whose provider is overridden but model/effort are not

### Scenario Coverage

- [x] A-006 R1: Scoped test packages pass — `go test ./src/go/fab/internal/configref/ ./src/go/fab/cmd/fab/`

### Edge Cases & Error Handling

- [x] A-007 R3: A per-role provider override naming a fill-less/unknown provider derives empty model/effort — the leaves fall through under empty-skip and render absent (no `model: ""` in composed output)

### Code Quality

- [x] A-008 Pattern consistency: `fillsOnly` mirrors `knobsOnly`'s shape, placement, and comment style; no mutation of the caller's config
- [x] A-009 No unnecessary duplication: fill precedence stays owned by `agent.ResolveRole` — configref performs no direct provider-fill lookups

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds the `fillsOnly` reduction alongside `knobsOnly` without making any existing file, function, branch, or config redundant (`knobsOnly` remains in use for the provider leaf).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Factoring = two `ResolveRole` calls (knobs-only for provider, fills-only for model/effort) rather than one call plus patching | Intake row 9 left factoring to apply; the split reuses the resolver's documented precedence without duplication; fully local to defaultsmap.go | S:70 R:90 A:85 D:75 |
| 2 | Confident | Empty fills from a fill-less overridden provider need no rendering code — existing empty-skip drops the leaves (observed today on kimi-governed roles) | Verified in current `config show` output (doing/fast/hydrate/review rows show provider only); no new mechanism | S:65 R:85 A:80 D:75 |
| 3 | Confident | `fillsOnly` deep-copies the per-role maps (entries reduced to `{Provider}`) so neither the original config nor its maps are mutated | Matches `knobsOnly`'s explicit no-mutation contract; map values are small structs | S:70 R:90 A:90 D:80 |

3 assumptions (0 certain, 3 confident, 0 tentative).
