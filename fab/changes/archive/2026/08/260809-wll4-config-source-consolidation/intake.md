# Intake: Config Source Consolidation (Config-Overhaul C5)

**Change**: 260809-wll4-config-source-consolidation
**Created**: 2026-08-09

## Origin

One-shot `/fab-new` invocation picking up the final change of the config-overhaul series:

> Config-overhaul Change 5 (source consolidation) per fab/plans/sahil/26-08-08-config-overhaul.md and backlog [x3cf] item 5: dispatch constants -> defaults.yaml, retire the fab-kit stub config.yaml, add config upgrade --check drift mode, add fence scope annotations, fix the show --origin knob-blind bug; plus the folded items -- config init --print/--force flags, and a config-file prose diet (short per-field fence lines pointing at fab config explain).

The authoritative design is `fab/plans/sahil/26-08-08-config-overhaul.md` § Change 5 (all decisions user-confirmed 2026-08-08/09). All prerequisite changes are **merged into main and present on this branch**: C1 env layer (#553), C2 six-verb surface (#555), C3 dispatch.mode ladder (#554), C4 field renames (#560), and the fp02 config read-model redesign (#557).

**Scope correction applied at intake**: the prompt's "fix the show --origin knob-blind bug" (plan item 5) is **dropped** — fp02 subsumed it (user-confirmed in fp02's intake: "remaining C5 items must rebase on this and drop those two"). Verified in the current source: `cmd/fab/config.go` `readModelDefaults` composes `configref.DefaultsMapFor(cfg)` against the live config, so the derived per-role provider rows already report the provider a depth knob selects. This change implements plan items **1, 2, 3, 4, 6, 7**.

## Why

**The source pain** (plan § Context): the built-in config layer has had multiple value/schema sources plus a stub copy, making "which config comes from where" a real question even for the maintainer. After fp02, values largely live in `internal/agent`'s embedded `defaults.yaml` and are projected into the read model via `configref.DefaultsMap` — but the `dispatch.*` defaults still live as Go constants in `internal/config` (`DefaultDispatchMode`/`DefaultDispatchColumnWidth`/`DefaultDispatchReapDone`), and the fab-kit binary still embeds a stub `config.yaml` as a skew fallback. Target embed census after this change (plan resolved decision 10): **`defaults.yaml` (values) + configref registry (schema/prose) + configscope (scope taxonomy) + `src/kit/scaffold/` (non-config files), zero stub copies.**

**The tooling-story pain**: `fab config upgrade` can repair a drifted file but nothing can *probe* for drift without writing — CI and humans have no "is this repo's config clean?" check. `fab config init` refuses to overwrite and cannot preview, so there is no zero-write way to see what it would write. `--check` and `--print`/`--force` complete the cleanup-tooling story alongside C2's `set`/`unset` (surgical repair) and `show --origin` (find typo'd overrides).

**The fence pain**: the generated fence text is paragraph-length per-field essays (the current project `config.yaml` fence is ~100 lines of prose), duplicating the registry's documentation which `fab config explain` already serves. And the fence never says *which file* a setting belongs in — preference-class (`scope: both`) adverts invite an uncomment-in-repo when the right home is usually `~/.fab-kit/config.yaml`.

If we don't do this: the last two value-source duplications persist (drift risk on every dispatch-default change), config cleanliness stays unprobeable in CI, and every `fab config upgrade` keeps regenerating essays that drift from the registry prose they duplicate.

## What Changes

Six items, each small and independently committable (plan § Change 5). Items 4 and 7 are co-implemented (same renderer, same golden tests).

### 1. One embedded values file — dispatch constants → `defaults.yaml`

Fold the three `dispatch.*` Go-constant defaults out of `src/go/fab/internal/config/config.go` (lines ~254–268):

- `DefaultDispatchMode = "native"`
- `DefaultDispatchColumnWidth = 35`
- `DefaultDispatchReapDone = true`

into `src/go/fab/internal/agent/defaults.yaml` (which already carries `agent:` and `providers:` and is shaped as a user config-file fragment), e.g.:

```yaml
dispatch:
  mode: native
  column_width: 35
  reap_done: true
```

so layer 3 (built-in defaults) has a **single config-shaped value source**. Registry rows in `internal/configref` (which today interpolate `config.DefaultDispatchMode` etc. at `configref.go` ~535–565) interpolate the same values from the new source. The fail-open accessor behavior on `config.Config` (`DispatchMode()`/`DispatchColumnWidth()`/`DispatchReapDone()` — invalid/absent values resolve to the default with a warning) is preserved byte-for-byte.

**Known constraint**: `internal/agent` imports `internal/config`, so `internal/config` cannot read the values back from `internal/agent` (the same configref→agent→config cycle fp02 documented). The exact Go seam — where the parsed dispatch defaults are exposed and how the accessors/configref consume them — is an apply-time design decision; the requirement is one value source, existing behavior preserved, drift-guarded by tests naming themselves on failure (the established defaults.yaml pin-test pattern).
<!-- assumed: item 1's Go exposure seam left to apply — cycle constraint documented, requirement is single-source + drift guards, not a prescribed package shape -->

*(Stretch explicitly OUT OF SCOPE, per plan: merging defaults.yaml as a true layer 0 inside `config.LoadPath`. The plan says "do NOT fold this into the same change"; fp02 additionally established the materialized-defaults tier lives at the read-model boundary, not in LoadPath.)*

### 2. Retire the fab-kit embedded stub config.yaml

`src/go/fab-kit/internal/init.go` falls open to a minimal embedded stub config.yaml when the installed fab-go cannot generate one (`stubConfigHeader`, fallback at ~line 178–185, plus `init_test.go` coverage). The skew window is long past — `fab config init --project` shipped in 2.15.x — so the fallback becomes a **clear error telling the user to upgrade fab-go** instead of writing a stub. Zero second copies of config content remain in any binary. Tests updated accordingly (`init_test.go`'s stub-fallback cases become error-path cases).

### 3. `fab config upgrade --check` — the drift probe

New flag on the existing verb (`cmd/fab/config.go` `configUpgradeCmd`, engine in `internal/configupgrade`): exit **non-zero when a run would change the file** — stale fence kit-version stamp, unparked unknown keys, missing fence, any rendered-content delta — with **zero writes**. Exit 0 when clean. This is the drift mode the fence's kit-version stamp was designed for (deferred from the config-upgrade effort's j0qm change). Output names what would change (the upgrade verb already reports its actions; `--check` reports without applying). CI/human probe for "is this repo's config clean?".

### 4. Fence teaches the file split — scope annotations

Annotate each fence field advert with its scope (`[project|system|both]` — in the original config-upgrade worked example but never shipped), and have preference-class (`scope: both`) adverts carry a **"settable machine-wide: `fab config set --system <key> <value>`"** pointer instead of inviting an uncomment-in-repo. Scope data comes from the existing `internal/configscope` taxonomy (already a registry row field). Rendering lives in the configref segment renderer / `configupgrade` fence renderer; byte-stability and golden tests move accordingly. Co-implemented with item 7.

### 6. `fab config init --print` / `--force`

Extends `cmd/fab/config.go` `configInitCmd` (the verb C2 reworked — why this rides in C5):

- `--print` renders the **exact file `init` would write** to stdout, zero writes; composes with both `--project` and `--system`. The preview probe alongside `upgrade --check`'s drift probe.
- `--force` replaces the existing-file refusal with an explicit overwrite (refusal stays the default).

### 7. Config-file prose diet

Both generated files — the project `config.yaml` managed fence and `init --system`'s output — currently carry paragraph-length per-field essays. Tighten the configref rendered segments so each field carries a **short description plus a `fab config explain <key>` pointer** for the full prose. The registry keeps the essays (`fab config explain` output is unchanged in depth); the generated files stop duplicating them. Natural rider on item 4 — both rework the fence's rendered text, so the byte-stability/golden tests move once. Golden-heavy: this rewrites the segment renderer (`configref.go` `providersSegment`/`agentSegment`/`dispatchSegment`/`stageHooksSegment` + per-field segments).

### Dropped: plan item 5 (show --origin knob-blind fix)

Already shipped by fp02 (#557, merged). No work here; the `_shared/configuration.md` under-reporting note it referenced was already updated by fp02's sweep.

### Cross-cutting obligations

- **CLI ⇒ docs + tests**: `_cli-fab.md` § fab config (new `--check`, `--print`, `--force` flags; fence-text description updates) + its SPEC mirror; Go tests per item (golden/idempotence for fence changes shared by items 4+7).
- **Specs**: `docs/specs/config.md` (managed-fence section, upgrade/init verb descriptions, embed census).
- **Memory sweep**: see Affected Memory.
- **No migration file**: no user-data restructure — the managed fence self-heals via `fab config upgrade` (kit-version stamp regeneration), and the stub retirement changes only binary fallback behavior on fresh-init edge paths.
- **shll standards check** before finalizing flag naming/help text (constitution § Toolkit Standards).

## Affected Memory

- `_shared/configuration.md`: (modify) fence description (scope annotations + short-line format), `upgrade --check`, `init --print`/`--force`, single-values-file claim for the built-in tier
- `distribution/kit-architecture.md`: (modify) stub config.yaml retirement (fab-kit init fallback → error), embed census (defaults.yaml + configref + configscope + scaffold, zero stubs)
- `runtime/providers-and-profiles.md`: (modify) defaults.yaml now carries `dispatch.*` values alongside agent knobs + providers (single built-in values file)

## Impact

- `src/go/fab/internal/config/config.go` — dispatch constants removed/re-sourced; accessors keep behavior (+ tests)
- `src/go/fab/internal/agent/defaults.yaml` + `internal/agent` parsing/pins — gains `dispatch:` block (+ pinned-test updates)
- `src/go/fab/internal/configref/` — registry default interpolation for dispatch rows; segment renderer rewrite for items 4+7 (golden-heavy)
- `src/go/fab/internal/configupgrade/` — `--check` dry-run mode; fence renderer text changes ride the segment rewrite
- `src/go/fab/cmd/fab/config.go` — `upgrade --check`, `init --print`/`--force` (+ tests)
- `src/go/fab-kit/internal/init.go` + `init_test.go` — stub retirement (fab-kit binary)
- `src/kit/skills/_cli-fab.md` + SPEC mirror — § fab config flag additions
- `docs/specs/config.md`, memory files above
- The project's own `fab/project/config.yaml` fence regenerates (shorter, scope-annotated) on the next `fab config upgrade` after release — expected, not part of this diff

## Open Questions

*(none — all design decisions user-confirmed in fab/plans/sahil/26-08-08-config-overhaul.md, 2026-08-08/09)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop plan item 5 (show --origin knob-blind fix); scope = items 1, 2, 3, 4, 6, 7 | fp02 subsumed it, user-confirmed in fp02's intake ("remaining C5 items must rebase on this and drop those two"); verified shipped in `cmd/fab/config.go` `readModelDefaults` → `DefaultsMapFor` | S:90 R:85 A:95 D:90 |
| 2 | Certain | LoadPath layer-0 defaults merge stays out of scope | Plan says explicitly "do NOT fold this into the same change"; fp02 settled the materialized tier at the read-model boundary | S:95 R:80 A:95 D:95 |
| 3 | Confident | No migration file ships with this change | No user-data restructure: the managed fence regenerates via `fab config upgrade` (existing self-heal path), stub retirement affects only binary fallback on fresh init | S:70 R:65 A:80 D:75 |
| 4 | Tentative | Item 1's Go exposure seam (how defaults.yaml dispatch values reach config accessors + configref) decided at apply | `internal/agent` imports `internal/config` (cycle — fp02 precedent), so the seam needs design; requirement fixed (single source, behavior preserved, drift-guarded), package shape not prescribed | S:55 R:45 A:50 D:40 |
| 5 | Confident | Exact fence-annotation line format (scope-tag placement, pointer wording) decided at apply against golden tests | Plan fixes the semantic (scope tag + `set --system` pointer for both-scoped adverts) but not byte layout; easily tuned, single renderer | S:65 R:70 A:70 D:60 |
| 6 | Confident | Items 4 and 7 land as one renderer rework; golden/byte-stability tests move once | Plan: "natural rider on item 4 … same renderer, same golden tests, do them together" | S:75 R:70 A:85 D:80 |

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved).
