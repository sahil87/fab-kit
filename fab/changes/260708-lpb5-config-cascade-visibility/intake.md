# Intake: Config Cascade & Visibility Commands

**Change**: 260708-lpb5-config-cascade-visibility
**Created**: 2026-07-08

## Origin

One-shot `/fab-new` invocation. Raw input:

> Implement the three-layer config cascade (project fab/project/config.yaml > system ~/.fab-kit/config.yaml > built-in defaults) in fab config loader, using per-field deep merge: maps merge per-key (existing agent.tiers precedent), lists replace (never concatenate), scalars replace. Enforce scope: a project-scoped field present in the system file is ignored with a warning (fail-open, config must never brick). Add fab config show --origin showing effective config with per-field provenance (git config --show-origin precedent). Add fab config init --system that writes a ~/.fab-kit/config.yaml scaffold containing ONLY scope=system/both fields, all commented, generated from the same field-metadata table shipped in the prior change (260708-ff2v-config-reference-metadata-table) so it cannot drift. See fab/plans/sahil/config-upgrade.md Change 2 for full context.

This is **Change 2 of the three-change config-upgrade effort** designed in the 2026-07-08 `/fab-discuss` session and written up in `fab/plans/sahil/config-upgrade.md` — all six schema decisions there are **user-confirmed** (cascade order, presence=intent, fence contract, auto-run fail-open, system path, scope taxonomy). The forward-looking design intent for this change is already recorded in `docs/specs/config.md` (§ Override cascade [Change 2], § Visibility commands [Change 2]), landed by Change 1.

**Dependency state**: Change 1 (`260708-ff2v`, the per-field metadata registry + `fab config reference --json`) is fully shipped (PR #474) and its implementation commit is cherry-picked onto this branch (`ee21b0e5`), so `internal/configref`'s 12-row `[]Field` registry — with `Scope`, `Default`, `Advertise`, `RenamedFrom` per row — is available to build on.

## Why

1. **The pain point.** There is no system layer today: `internal/config.Load` reads only `fab/project/config.yaml`. Personal preference-class settings (model tiers, provider commands) must be re-declared in every repo's project file, polluting repo-reproducible config with per-user choices. And the ff2v registry's `scope` metadata is data-only — nothing enforces or consumes it yet.
2. **The consequence of not fixing it.** The config-upgrade effort's end state (config.yaml as a sparse overrides file, mechanically reconciled by `fab config upgrade` in Change 3) presupposes a working cascade: without layer resolution there is nothing for "B) not overridden → absent, inherited" to inherit *from* at the user level, and Change 3's upgrader has no defined semantics to preserve. Separately, three layers **without provenance is archaeology**: a typo'd override (`agent.teirs:`) silently no-ops today, and adding a second file doubles that failure surface. The visibility commands are the non-negotiable companion.
3. **Why this approach.** The merge semantics reuse the in-repo precedent (`internal/agent`'s per-field merge of project tiers over built-in defaults, tykw) rather than inventing new rules; scope enforcement and both new commands are generated from/checked against the single ff2v registry, so no second schema copy exists to drift — the same no-drift invariant Change 1 established.

## What Changes

### 1. Three-layer cascade in the fab config loader (`src/go/fab/internal/config`)

`config.LoadPath` (the single seam every consumer goes through — preflight, impact, status, resolve-agent, dispatch, agent, operator, batch, spawn, prmeta) gains system-layer resolution:

1. **project** — `fab/project/config.yaml` (highest precedence)
2. **system** — `~/.fab-kit/config.yaml` (decision 5: co-located with the version cache; XDG rejected). Resolved via `os.UserHomeDir()`; tests override with `t.Setenv("HOME", …)`.
3. **built-in defaults** — the Go constants already applied at existing point-of-use seams (`internal/agent`'s tier/provider merge, the nil-safe accessor fallbacks).

**Merge semantics** (per-field deep merge, recursive):

- **maps merge per-key** — the existing `agent.tiers` precedent, applied at every map level: project `agent.tiers.review.model` + system `agent.tiers.review.effort` compose into one effective `review` profile; a system-only `doing` tier survives alongside project-only `review`.
- **lists replace** — never concatenate. A project `source_paths: [src/]` fully replaces any system-file list (moot in practice — lists are project-scoped — but the rule is uniform).
- **scalars replace** — project value wins.

The two *files* merge generically (YAML map level, before unmarshal into `Config`); the built-in-defaults layer stays where it lives today (point-of-use fallbacks), which composes to identical three-layer semantics with no changes to any consumer. **Absent system file ⇒ byte-identical behavior to today** (empty layer, no error). **Malformed system file ⇒ warn + skip the layer** (fail-open — a broken personal file must not brick every repo on the machine); a malformed project file keeps today's error behavior.

The cascade applies wherever config is loaded — including explicit-path callers (`fab agent --repo`, `fab batch`, spawn) and no-project runs (`fab operator` outside a repo): the system layer is user-global by definition.

### 2. Scope enforcement (fail-open)

Before merging, the system file's top-level override units are checked against the ff2v registry's `Scope`:

- `scope: both` (`agent.tiers`, `providers`) or `scope: system` (none today) → honored.
- `scope: project` (`project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `stage_hooks`, `branch_prefix`, `fab_version`) → **pruned from the system layer with a stderr warning**, e.g.:

```
fab: warning: ignoring project-scoped field "source_paths" in ~/.fab-kit/config.yaml (project-scoped fields belong in fab/project/config.yaml)
```

Fail-open throughout: warnings never change the exit code, and stdout contracts (preflight YAML, resolve output) are unaffected. Unknown keys in the system file are ignored silently, matching project-file behavior today (typo surfacing is `show --origin`'s job; a `fab config validate` linter remains the recorded future non-goal).

**Import-cycle constraint** (verified): `internal/config` cannot import `internal/configref` — configref imports `internal/agent`, which imports `internal/config`. The scope/key metadata must therefore reach the loader cycle-free: extract the key+scope table (not the defaults, which need `agent`) into a leaf package both sides consume, or inject it as data. Exact packaging is a plan-time decision; the invariant is that scope values are **never duplicated** outside the registry's single source.

### 3. `fab config show [--origin]` — new subcommand

Pure query in the `reference` family. Bare `fab config show` prints the **effective** (post-cascade) config; `--origin` adds per-field provenance (the `git config --show-origin` precedent). Provenance is computed by walking the full registry — every field row reports its effective value and origin, with **per-key drill-down for map-valued fields** (maps merge per-key, so per-key is the honest granularity). Illustrative `--origin` shape (exact format is plan-time):

```
project.name = fab-kit                          # fab/project/config.yaml
agent.tiers.review.model = claude-fable-5       # fab/project/config.yaml
agent.tiers.doing.model = claude-opus-4-8       # ~/.fab-kit/config.yaml
agent.tiers.fast.effort = low                   # default
providers.claude.session_command = claude …     # default
```

A typo'd override surfaces because the *intended* field shows `origin: default` when the user expected their file to win. Registry `Default`s (the canonical `null`-vs-real-value convention from `docs/specs/config.md` § Default semantics) feed the `default`-origin rows — this is the `--json`/table consumption Change 1 built for.

### 4. `fab config init --system` — new subcommand

Writes the `~/.fab-kit/config.yaml` scaffold: a header explaining the system layer, then **ONLY `scope: system`/`both` fields** (today: `agent.tiers`, `providers`), **all commented**, generated from the same registry (segments/defaults) so the scaffold cannot drift from the schema. Sketch:

```yaml
# ~/.fab-kit/config.yaml — system-level fab config (all repos on this machine).
# Resolves below any project's fab/project/config.yaml and above built-in defaults.
# Only preference-class fields (scope: system/both) are honored here; project-scoped
# fields in this file are ignored with a warning.
#
# agent:
#     tiers:
#         review:
#             provider: claude
#             model: …
#             effort: …
# providers:
#     claude:
#         session_command: …
```

Refuses to overwrite an existing `~/.fab-kit/config.yaml` (non-zero exit, message naming the path) — the file is user-owned once created. No `--force` in v1. `fab config init` without `--system` is a usage error (a project-scaffold mode is not part of this change — `/fab-setup` owns project bootstrap).

### 5. Documentation, spec, and test obligations

- **`src/kit/skills/_cli-fab.md`** § fab config: document `show [--origin]` and `init --system` alongside `reference` (constitution: CLI changes MUST update `_cli-fab.md` + tests).
- **`docs/specs/skills/SPEC-_cli-fab.md`**: mirror sweep in the same change (code-quality § Sibling & Mirror Sweeps).
- **`docs/specs/config.md`**: flip the `[Change 2]` forward-looking sections (§ Override cascade, § Visibility commands, the scope-enforcement note in § Scope taxonomy) to landed status — same treatment ff2v gave Change 1.
- **Go tests** (constitution VII, test-alongside): cascade merge table tests (maps per-key incl. nested profile fields, lists replace, scalars replace, absent/malformed system file), scope-enforcement warning + pruning, `show`/`show --origin` byte-stable rendering, `init --system` write/refuse-overwrite — all system-path tests via `t.Setenv("HOME", …)`.

### 6. Explicitly NOT in this change

- **No migration file**: no existing user data is restructured — the system file is net-new and opt-in; `config.yaml` is never written (single-writer discipline arrives with Change 3).
- **No `fab config upgrade`, no fence, no `fab_version` move, no `setFabVersion` deletion** — all Change 3.
- **The fab-kit binary is untouched** (`ResolveConfig`/`readFabVersion` move in Change 3); the cascade lands in the fab module's `internal/config` only.
- **No `advertise`/`renamed_from` consumers** — those activate in Change 3.

## Affected Memory

- `_shared/configuration.md`: (modify) add the three-layer cascade semantics (layer order, per-field deep merge rules, fail-open scope enforcement), the system file `~/.fab-kit/config.yaml`, and the `fab config show [--origin]` / `fab config init --system` command surfaces
- `distribution/kit-architecture.md`: (modify) extend the `fab config` command-group entry (6nke/ff2v) with the two new subcommands and the loader's cascade/scope-enforcement behavior

## Impact

- **Go (fab module)**: `internal/config` (loader — cascade merge, scope pruning, warning emission), `internal/configref` (scope/key metadata exposed cycle-free — possible leaf-package split), `cmd/fab/config.go` (two new subcommands), corresponding `_test.go` files. All ~12 `config.Load`/`LoadPath` call sites get effective config with **zero per-caller changes**.
- **Kit markdown**: `src/kit/skills/_cli-fab.md` (§ fab config).
- **Specs**: `docs/specs/skills/SPEC-_cli-fab.md` (mirror), `docs/specs/config.md` ([Change 2] sections → landed).
- **Behavioral risk**: low for existing users — no system file means byte-identical behavior; the new layer only activates when `~/.fab-kit/config.yaml` exists.
- **Dependency**: ff2v registry (present on this branch via cherry-pick `ee21b0e5`; shipped upstream as PR #474).

## Open Questions

None — all six schema decisions were user-confirmed in the 2026-07-08 `/fab-discuss` session (recorded in `fab/plans/sahil/config-upgrade.md` § Resolved decisions and `docs/specs/config.md`).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Layer order (project > system > defaults), merge rules (maps per-key deep, lists replace, scalars replace), and system path `~/.fab-kit/config.yaml` are fixed as specified | User-confirmed decisions (config-upgrade.md cascade + decision 5), restated in docs/specs/config.md § Override cascade | S:90 R:70 A:95 D:95 |
| 2 | Certain | Scope enforcement is fail-open — project-scoped fields in the system file are pruned with a stderr warning; taxonomy = the ff2v registry's Scope values (decision 6) | User-confirmed; the registry already encodes scope per row | S:90 R:80 A:95 D:90 |
| 3 | Certain | `init --system` scaffold contains ONLY scope system/both fields (today `agent.tiers` + `providers`), all commented, generated from the registry | Explicit in the request and plan doc — "generated from the same field-metadata table so it cannot drift" | S:90 R:80 A:90 D:90 |
| 4 | Confident | The two config *files* merge in the loader; the built-in-defaults layer stays at existing point-of-use seams (internal/agent merge, nil-safe accessors); registry Defaults are consumed by `show --origin` for display only | Minimal diff preserving the ye8r single-parser and tykw merge architecture; file-merge + existing fallbacks compose to identical three-layer semantics | S:60 R:70 A:80 D:70 |
| 5 | Confident | Scope/key metadata reaches `internal/config` cycle-free (leaf-package extraction or data injection — plan picks the packaging); scope values stay single-sourced in the registry | Verified import chain configref → agent → config forbids a direct import; both resolutions are internal refactors | S:55 R:75 A:70 D:60 |
| 6 | Confident | Malformed or unreadable system file ⇒ warn + skip the layer (fail-open); project-file error behavior unchanged | "Config must never brick" governs the new user-global layer; a broken personal file must not break every repo on the machine | S:60 R:80 A:75 D:70 |
| 7 | Confident | Bare `fab config show` prints effective config without provenance; `--origin` adds it; both are pure queries in the `reference` family | `git config --show-origin` precedent named in the request; flag-adds-annotation is the natural reading | S:65 R:85 A:75 D:70 |
| 8 | Confident | `show --origin` walks the full registry (every field: effective value + origin ∈ {project path, system path, default}) with per-key drill-down for map-valued fields | "Per-field provenance" + the registry as single source; per-key is the honest granularity where maps merge per-key; output format trivially reversible | S:45 R:85 A:65 D:50 |
| 9 | Confident | The scope warning fires on every config load (stderr, `fab: warning:` prefix), not only in `show` | "Ignored with a warning" is load-time semantics; stderr never disturbs stdout contracts or exit codes | S:50 R:80 A:70 D:55 |
| 10 | Confident | `init --system` refuses to overwrite an existing system file (non-zero exit + message); no `--force` in v1; bare `fab config init` is a usage error | Conservative default — the file is user-owned once created; overwrite support is a cheap later addition | S:40 R:80 A:65 D:55 |
| 11 | Confident | The cascade applies at `LoadPath` (the single seam), so explicit-path callers (`--repo`, batch, spawn) and no-project runs also see the system layer | The system layer is user-global by definition; a per-caller opt-out would contradict "effective config" | S:35 R:70 A:60 D:50 |
| 12 | Confident | No migration file ships — nothing restructures existing user data (system file is net-new + opt-in; config.yaml is never written) | The migration obligation triggers on restructuring *existing* data (code-quality anti-patterns); Change 3 owns the config.yaml restructure | S:55 R:75 A:80 D:75 |

12 assumptions (3 certain, 9 confident, 0 tentative, 0 unresolved).
