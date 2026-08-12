# Intake: Config Show Honors Per-Role Provider Overrides in Derived Fills

**Change**: 260812-05wy-config-show-role-provider-fills
**Created**: 2026-08-12

## Origin

> Fix `fab config show`'s derived `agent.profiles` rows to honor per-role provider overrides when deriving model/effort fills.

Dispatched promptless by `/fab-proceed` from a live conversation in which the bug was reproduced (fab 2.19.8), the mechanism was traced to source, and the fix was **decided by the user**. This is a follow-up bug fix to the shipped change `260810-rvza-config-show-compose-defaults` (which introduced the live-knob composition), not a re-activation of it.

## Why

**Problem (verified on fab 2.19.8).** With the system config `~/.fab-kit/config.yaml` setting only:

```yaml
agent:
  session: claude
  workers: kimi
  profiles:
    operator:
      provider: codex
```

`fab config show` displays `agent.profiles.operator` as `{provider: codex, model: claude-sonnet-5, effort: medium}` — a **chimera row that no resolution path ever produces**. Runtime resolution is correct: `fab resolve-agent operator` returns `model=gpt-5.6-luna effort=medium provider=codex` (codex's own operator fill, per `src/go/fab/internal/agent/defaults.yaml`). Only the composed `config show` view — and keyed `--origin`, which shows `model = claude-sonnet-5 # default (effective)` — is misleading.

**Consequence if unfixed.** The read model (`config show`) and the resolver (`resolve-agent`) disagree about which model a role runs at whenever a per-role `provider` override is set — exactly the escape hatch the config reference advertises (`profiles: operator: { provider: codex }` is the shipped example in the managed fence). Users debugging "which model will my operator run?" get a fabricated answer from the primary inspection surface.

**Mechanism (confirmed in `src/go/fab/internal/configref/defaultsmap.go`).**

- The `agent.profiles` built-in-defaults tier is DERIVED, not stored: `DefaultsMapFor` → `liveRoleProfiles` resolves each role via `agent.ResolveRole` against `knobsOnly(cfg)`, which strips ALL of the user's `agent.profiles` (and legacy `agent.tiers`) entries:

  ```go
  func knobsOnly(cfg *config.Config) *config.Config {
      if cfg == nil {
          return nil
      }
      stripped := *cfg
      stripped.Agent.Profiles = nil
      stripped.Agent.Tiers = nil
      return &stripped
  }
  ```

- With the operator's `provider: codex` override stripped, the operator role falls back to the depth knob `session: claude` → claude's operator fill (`claude-sonnet-5`/`medium`) becomes the derived "built-in" row.
- The four-tier per-leaf merge (env > system > project > defaults) then combines the `provider` leaf from the system tier (codex) with the stale `model`/`effort` leaves from the defaults tier (claude's fills) — producing the chimera.
- The stripping exists for a legitimate reason (doc comment at `defaultsmap.go:40-48`): the defaults tier must never echo back a higher tier's value (otherwise a user's own `agent.profiles.review.model` override would be reported as its own built-in on the `default (shadowed)` origin line). The fix must preserve this no-echo invariant.

**Why this approach.** Keeping only the per-role `provider` override during fills derivation makes the derived model/effort values ones that appear in **no higher tier** (they come from the overridden provider's own built-in fill map), so the no-echo invariant holds by construction. The file's own doc comment already makes this honesty argument for the depth knobs ("composing against cfg reports the provider (and its fills) honestly") — the fix extends the same argument to per-role provider overrides.

## What Changes

### `liveRoleProfiles` keeps the per-role provider override when deriving fills

In `src/go/fab/internal/configref/defaultsmap.go`, when deriving each role's **model/effort** fills, strip only the per-role **model/effort** overrides but KEEP the per-role **provider** override — a `fillsOnly`-style variant of `knobsOnly` (per-role entries reduced to `{provider}` only, `model`/`effort` cleared; applies identically to `agent.profiles` and the legacy `agent.tiers` spelling). The operator row's derived model/effort then honestly read codex's fills.

Expected behavior with the reproduction config above:

```
$ fab config show   # agent.profiles.operator, after the fix
operator: {provider: codex, model: gpt-5.6-luna, effort: medium}
```

### The `provider` leaf's own default stays knobs-only-resolved

The derived `provider` leaf keeps resolving against `knobsOnly(cfg)` (claude, via the session knob), so `--origin` provenance remains truthful — the system tier's codex genuinely shadows the built-in claude:

```
$ fab config show agent.profiles.operator.provider --origin
agent.profiles.operator.provider = codex    # system ~/.fab-kit/config.yaml  (effective)
agent.profiles.operator.provider = claude   # default  (shadowed)

$ fab config show agent.profiles.operator.model --origin
agent.profiles.operator.model = gpt-5.6-luna    # default  (effective)
```

Composition is therefore per-leaf split: **provider** from the knobs-only resolution, **model/effort** from the provider-override-kept resolution. The exact factoring (e.g., a second `agent.ResolveRole` call per role against the `fillsOnly` config, taking model/effort from it) is left to apply within these semantics.

### Tests (same change, per Constitution VII / code-quality test-alongside)

- `src/go/fab/internal/configref/defaultsmap_test.go`:
  - **New regression test**: a config with a depth knob plus a per-role `provider` override derives that role's model/effort from the **overridden provider's** fills (the chimera case above), while the derived `provider` leaf stays the knob's provider.
  - **Refine `TestDefaultsMapFor_IgnoresUserRoleOverrides`** (line 89): its current expectation that a pinned `{provider, model, effort}` override leaves all three derived leaves equal to the nil-config built-in changes for model/effort — with `provider: pinned-provider` kept, model/effort now derive from `pinned-provider`'s (empty) fills, not claude's. The pinned `model`/`effort` values must still never echo (the no-echo assertion survives, its expected values change). The provider-leaf expectation is unchanged. The trailing knob-survival assertion (`review provider = codex` under a workers knob) is unaffected.
- `src/go/fab/cmd/fab/config_show_init_test.go`: extend or add alongside `TestConfigShow_ComposesDerivedDefaultsAgainstLiveKnobs` (line 110) / `TestConfigShowOrigin_DrillDownIsKnobAware` (line 543) to cover the composed end-to-end view with a per-role provider override — no chimera row, `--origin` shows `model … # default (effective)` with the overridden provider's fill.

### Documentation prose (verified restatements only)

- `docs/specs/config.md` (~line 500, "The composed `agent.profiles.*` drill-down rows are knob-aware"): amend the prose that says the rows are "derived from the depth knobs and the provider fills" with the user's `agent.profiles`/`agent.tiers` entries stripped — after the fix, per-role **provider** overrides also feed fills derivation, and only per-role model/effort overrides are stripped.
- `src/kit/skills/_cli-fab.md`: **no edit** — verified it does not restate the knobs-only composition detail, and no CLI command signature changes.

### Non-goals

- No change to runtime resolution (`agent.ResolveRole`, `fab resolve-agent`) — it is already correct.
- No change to the loader (`internal/config`) — the defaults tier remains read-model-only (the `DefaultsMap` doc comment's loader rationale is untouched).
- No new config keys, flags, or migration (no user-data restructuring).

## Affected Memory

- `_shared/configuration`: (modify) § Six-Verb Surface, the "composed `agent.profiles.<role>` drill-down rows are knob-aware" paragraph (line ~75) — currently states the user's `agent.profiles`/`agent.tiers` entries are "stripped first"; after the fix only per-role model/effort overrides are stripped and the provider override feeds fills derivation.
- `runtime/providers-and-profiles`: (modify) § Design Decisions, the "registry row is the STATIC default; the read model composes a live one" consequence (line ~535) — same stale "entries stripped first" claim.

(The originating discussion guessed `distribution`; verified against `docs/memory/index.md` and grep — the config read-model prose lives in `_shared` and `runtime`, and `distribution/` has no restatement.)

## Impact

- **Code**: `src/go/fab/internal/configref/defaultsmap.go` (`liveRoleProfiles`, `knobsOnly` + new `fillsOnly`-style variant; doc comments updated to state the extended honesty argument).
- **Tests**: `src/go/fab/internal/configref/defaultsmap_test.go`, `src/go/fab/cmd/fab/config_show_init_test.go`. Run scoped: `go test ./src/go/fab/internal/configref/ ./src/go/fab/cmd/fab/` first.
- **Specs**: `docs/specs/config.md` (one paragraph).
- **Memory (hydrate)**: the two files above.
- **No CLI surface change**: command signatures, flags, and output formats are unchanged — only the composed values become truthful.

## Open Questions

None — the fix mechanism, provenance semantics, and scope were all decided in the originating discussion; the remaining choices (exact helper factoring, empty-fill leaf rendering) are graded below and decided at apply.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix = strip only per-role model/effort overrides in `liveRoleProfiles`, KEEP the per-role provider override (a `fillsOnly`-style variant of `knobsOnly`) | Discussed — user made this exact decision; mechanism verified in source | S:95 R:85 A:95 D:95 |
| 2 | Certain | The `provider` leaf's own derived default stays knobs-only-resolved (claude), so keyed `--origin` keeps `provider = codex # system (effective)` / `claude # default (shadowed)` while `model = gpt-5.6-luna # default (effective)` | Discussed — user specified this provenance split explicitly | S:90 R:80 A:90 D:85 |
| 3 | Certain | New follow-up change; the shipped `260810-rvza-config-show-compose-defaults` is not re-activated | Discussed — stated in the dispatch; rvza is completed/shipped | S:100 R:95 A:100 D:100 |
| 4 | Certain | Tests ship in the same change: a new chimera regression test plus refined expectations in `TestDefaultsMapFor_IgnoresUserRoleOverrides` and an end-to-end `config show`/`--origin` case | Constitution VII + code-quality test-alongside; existing test files located and read | S:90 R:90 A:100 D:95 |
| 5 | Certain | Affected memory = `_shared/configuration` + `runtime/providers-and-profiles` (both carry the now-stale "entries stripped first" claim), correcting the discussion's `distribution` guess | Verified by grep against `docs/memory/` per the procedure; `distribution/` has no restatement | S:70 R:85 A:90 D:85 |
| 6 | Certain | `docs/specs/config.md` drill-down paragraph is amended in-change; `src/kit/skills/_cli-fab.md` untouched (no restatement found, no signature change) | Dispatch said "update ONLY if prose restates the detail (check first)" — checked: config.md:500 restates it, _cli-fab.md does not | S:80 R:90 A:85 D:80 |
| 7 | Confident | A per-role provider override naming an unknown/fill-less provider (e.g. `kimi`, or a name fab ships nothing for) derives empty model/effort — under empty-skip the leaf falls through/renders absent — mirroring `resolve-agent`'s honest empty resolution | Follows the existing provider-neutral pass-through design (`TestDefaultsMapFor_UnknownProviderPassesThrough`); exact rendering decided at apply | S:60 R:80 A:75 D:65 |
| 8 | Confident | Legacy `agent.tiers` per-role provider overrides get identical treatment to `agent.profiles` (provider kept, model/effort stripped) | `knobsOnly` already treats the two spellings symmetrically; no signal the fix should diverge | S:60 R:85 A:85 D:75 |
| 9 | Confident | Implementation factoring — e.g. a second `agent.ResolveRole` call per role against the `fillsOnly` config, taking model/effort from it while the provider leaf keeps the `knobsOnly` resolution — is left to apply within the agreed semantics | Small, reversible, fully local to `defaultsmap.go`; the semantics (rows 1–2) are fixed, only the code shape is open | S:70 R:90 A:80 D:70 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).
