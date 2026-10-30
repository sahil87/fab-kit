# Intake: fab config show --json, set validation, and three config bug fixes

**Change**: 261030-h568-config-show-json-set-validation
**Created**: 2026-10-30

## Origin

One-shot brief handed to this session from a HexoKit (sahil87/hexokit) session working on hexokit change `261030-wf1x-fab-system-config-agents-tab` — a new **Settings → Agents** tab that edits fab-kit's machine-wide config at `~/.fab-kit/config.yaml`: `agent.session`, `agent.workers`, `agent.profiles.<role>.{provider,model,effort}`, and `dispatch.{mode,column_width,min_cols,min_rows,reap_done}`. Invoked as `/fab-new` then `/fab-fff`; the brief says: if the intake gate fails, stop and report; ship ends at a PR (no merge, no release).

Raw brief (condensed, verbatim where it states a requirement):

> HexoKit wraps the `fab config` CLI and never edits the YAML itself. Its daemon runs fab from `$HOME`, so it sees only the env, system and default tiers. The user decided that fab-kit gets a machine-readable output first, so hexokit never has to parse `show --origin` text.
>
> 1. **JSON output for `fab config show`.** Add `--json` (exact flag name is your call; it must fit the existing CLI conventions), in particular together with `--origin`. Return one entry per leaf with: `key`; `value`, typed: string, bool, number or list; `origin`, one of `default | env | system | project`; an origin detail: the env var name, such as `FAB_AGENT_WORKERS`, or the file path. Map-valued fields (`agent.profiles`, `providers`) drill down per leaf, like the text output does today. `fab config show <key> --origin --json` should return that key's tier stack, winner first, each tier marked effective or shadowed. Pick a stable shape, and document it wherever fab-kit documents its CLI contracts (`_cli-fab` skill, docs/specs).
> 2. **`fab config set` validates values.** Today `fab config set agent.workers bogus` and `fab config set dispatch.mode weird` both exit 0. Refuse out-of-range values with a non-zero exit and a clear message: `dispatch.mode` must be one of `pane | native | headless`; `agent.session`, `agent.workers` and `agent.profiles.<role>.provider` must name a provider in the effective `providers` table; `dispatch.reap_done` must be a bool; `dispatch.column_width`, `min_cols` and `min_rows` must be integers in their valid ranges.
> 3. **Panic fix.** With `dispatch.mode: weird` in `~/.fab-kit/config.yaml`, `fab agent review -o yaml` panics with a nil-pointer dereference in `config.(*Config).GetDispatchMode` (`fmt.Fprintf` on a nil writer). An invalid mode should degrade to the default with a warning, never panic.
> 4. **Shadow warning.** The `fab config set` help says that when a higher tier already defines the key, the write succeeds with a warning naming the winning tier. That warning did not fire when `$FAB_AGENT_WORKERS` shadowed a `--system` write. Fix it, or correct the help if env is intentionally excluded.
> 5. **Evaluate, don't necessarily fix: inherited role values are mislabelled.** `show --origin` labels a role provider inherited from `agent.workers` as `# default` … `fab agent <role> -o yaml` reports the source correctly (`source.provider: agent.workers`). Decide whether the JSON output should carry the real source. If so, do it here; otherwise record it as a follow-up in the intake.
>
> Constraints: Never write the developer's real `~/.fab-kit/config.yaml` in tests — isolate HOME the way fab-kit's existing tests do. Follow the constitution and code-quality rules.

### Reproduction done at intake (dev build of this worktree, HOME pinned to a scratch dir, cwd outside any fab project)

| # | Command | Observed |
|---|---------|----------|
| 3 | `printf 'dispatch:\n  mode: weird\n' > $HOME/.fab-kit/config.yaml; fab agent review -o yaml` | `panic: runtime error: invalid memory address or nil pointer dereference` at `fmt.Fprintf({0x0,0x0}, …)` from `internal/config/config.go:1010` (`GetDispatchMode`) |
| 4 | `FAB_AGENT_WORKERS=codex fab config set --system agent.workers agy` from a **non-project** dir | `Set agent.workers in …` exit 0, **no warning** |
| 4 | the same command from **inside** a fab project | warning fires: `fab: warning: agent.workers is shadowed by env $FAB_AGENT_WORKERS — the written value is not in effect` |
| 2 | `fab config set --system agent.workers bogus` / `dispatch.mode weird` / `dispatch.column_width 500` | all exit 0 and write the value |
| 2 | `fab config set --system dispatch.reap_done bogus` | already refused: `config key "dispatch.reap_done" expects a YAML bool, got string` exit 1 (the kind check in `parseSetScalar`) |
| 5 | `fab config show --origin` with `agent.workers: bogus` in the system file | `agent.profiles.doing.provider = bogus  # default` next to `agent.workers = bogus  # <system path>` |

## Why

1. **Problem.** HexoKit's Agents tab must render each knob's effective value *and where it came from* (so it can say "set by `$FAB_AGENT_WORKERS`, your edit here won't take effect") and must write values back through `fab config set --system`. Today the only provenance surface is the human-formatted `show --origin` text (`key = value  # origin`, column-aligned, values rendered with `fmt %v` / YAML flow), which loses type information (`45` vs `"45"`, `true` vs `"true"`), has no stable grammar for the origin column (`$VAR`, a bare path, `default`, or in the keyed form `env $VAR` / `system <path>` plus `(effective)`), and is not a contract. Parsing it would couple hexokit to presentation text.
2. **Consequences of not fixing.** (a) hexokit would parse fragile text; (b) `fab config set` silently accepts values that later degrade (bogus provider → a role that can't dispatch; `dispatch.mode weird` → previously a **panic** in every `fab agent`/dispatch call on the machine because the system file is machine-wide); (c) the shadow warning hexokit would surface to the user is silent in exactly hexokit's environment (daemon cwd = `$HOME`, not a fab project); (d) `# default` for a derived role row misleads the UI into showing "built-in" for a value the user actually set via `agent.workers`.
3. **Why this approach.** A `--json` format switch on the existing `show` verb mirrors the established `fab config explain --json` precedent (same verb family, same flag name) rather than inventing a new verb. Validation belongs at the write chokepoint (`fab config set`) — the read path stays fail-open (Constitution III spirit; "a malformed personal preference must not brick every repo"), so a hand-edited bad value still degrades with a warning instead of erroring. The panic and shadow-warning fixes are root-cause fixes at the exact faulty lines (below), not symptom patches.

## What Changes

### 1. `fab config show --json` (all four `show` forms)

`--json` is a pure **format switch** over the four existing `show` forms — every form keeps its current semantics, only the encoding changes. Flag name `--json` (matches `fab config explain --json`; boolean, default false). Text output is byte-unchanged when `--json` is absent.

| Form | JSON output |
|------|-------------|
| `fab config show --json` | The fully composed effective config (defaults ⊕ project ⊕ system ⊕ env — the same tree the plain form prints as YAML) as **one JSON object**, typed values |
| `fab config show <key> --json` | The key's effective value as a JSON value (string / number / bool / array / object / `null`) |
| `fab config show --origin --json` | JSON **array** of leaf rows, WINNER only, same leaves and same order as the text form (registry order of top-level keys, sorted leaf keys within a subtree; map-valued fields `agent.profiles`, `providers` drill down per leaf) |
| `fab config show <key> --origin --json` | JSON **array** of leaf rows forming the key's **full stack**: for a scalar/list key, one row per defining tier, highest precedence first (winner first); a map-valued key drills down per leaf and each leaf contributes its own stack rows in order — exactly the lines the text form prints |

**One row schema for both `--origin --json` forms** (stable; documented as a contract):

```json
{
  "key": "agent.workers",
  "value": "codex",
  "origin": "env",
  "origin_detail": "FAB_AGENT_WORKERS",
  "status": "effective"
}
```

- `key` — dotted leaf key (string).
- `value` — the leaf value, **typed**: JSON string, number, bool, array (lists), or `null`. Never a pre-rendered string. (Numbers: the default tier is normalized through JSON so ints arrive as float64 internally — they must still encode as `45`, not `45.0`/`"45"`.)
- `origin` — exactly one of `"default" | "env" | "system" | "project"` (the existing `tierDefault/tierEnv/tierSystem/tierProject` words).
- `origin_detail` — the environment variable name **without** the `$` (e.g. `"FAB_AGENT_WORKERS"`; for a leaf beneath a map-valued env row, the longest-prefix variable the text form already resolves, e.g. `FAB_AGENT_PROFILES`), or the absolute file path for `system`/`project`, or `""` for `default` (and `""` for the unreachable env-without-recorded-variable case). Always present.
- `status` — `"effective"` or `"shadowed"`. In the bare `--origin --json` form every row is `"effective"` (winner-only). In the keyed form the first row per leaf is `"effective"`, the rest `"shadowed"`.
- `derived_from` — **optional**, present only on derived rows (see §5). Omitted otherwise.

Edge cases, mirroring the text form:
- Keyed form, key defined by no tier (text prints `key = null  # (no tier defines this key)`): JSON prints `[]`.
- Keyed form, a higher tier replaces an ancestor with a non-map (text prints `key = null  # <ancestor tier>  (effective)`): JSON prints one row `{"key": "<requested key>", "value": null, "origin": <ancestor tier word>, "origin_detail": <ancestor detail>, "status": "effective"}`.
- Unknown key: same error + non-zero exit as today (no JSON on stdout).
- Output is a single JSON document followed by a newline; indentation style follows `fab config explain --json` (check `configref.RenderJSON` and match it).

Implementation guidance: reuse the existing provenance walk — `flattenOrigin` / `stackLines` / `keyedOriginLines` / `cascadeTier.origin` in `src/go/fab/cmd/fab/config.go` — so text and JSON cannot diverge. The cleanest shape is to have the walk produce structured rows (key, raw `any` value, tier kind, detail, status, optional derived_from) and render them two ways (aligned text lines vs JSON), instead of a second parallel walk. `originLine` currently stores a pre-rendered `value string` and an `origin string`; extend/replace it so the raw value and tier kind/detail survive to the renderer. The text rendering must remain byte-identical (existing tests in `config_show_init_test.go` / `config_test.go` pin it).

Example (hexokit's case — cwd `$HOME`, system file has `agent.workers: agy`, env `FAB_AGENT_WORKERS=codex`):

```console
$ fab config show agent.workers --origin --json
[
  {"key":"agent.workers","value":"codex","origin":"env","origin_detail":"FAB_AGENT_WORKERS","status":"effective"},
  {"key":"agent.workers","value":"agy","origin":"system","origin_detail":"/home/u/.fab-kit/config.yaml","status":"shadowed"},
  {"key":"agent.workers","value":"claude","origin":"default","origin_detail":"","status":"shadowed"}
]
```

### 2. `fab config set` value-domain validation

After the existing kind check (`parseSetScalar` in `src/go/fab/internal/configupgrade/mutation.go` — already refuses `dispatch.reap_done bogus` and non-int `column_width`), add a **value-domain** check that refuses out-of-range values with exit non-zero, a clear `ERROR:` message naming the key, the bad value, and the accepted set/range, and **writes nothing**. Applies to both project `set` and `set --system`.

| Key | Rule | Accepted-set source (single source of truth, no new literal) |
|-----|------|------|
| `dispatch.mode` | one of `pane`, `native`, `headless` | Export the set once (e.g. `config.ValidDispatchModes`, following the existing `config.ValidAutopilotMergeModes` precedent) and make `GetDispatchMode`'s switch read it too |
| `dispatch.column_width` | int, `1..99` | the range `GetDispatchColumnWidth` already enforces (`w > 0 && w < 100`) — name the bounds once and share them |
| `dispatch.min_cols`, `dispatch.min_rows` | int, `>= 1` (no upper bound) | the `v > 0` rule their accessors already enforce |
| `dispatch.reap_done` | bool | already enforced by the kind check — add a regression test only |
| `agent.session`, `agent.workers`, `agent.profiles.<role>.provider` (any role) | must name a provider in the **effective providers table** = fab-kit's built-in providers ∪ the `providers:` keys of the effective (cascaded) config visible from the cwd | `agent.ProviderNames(cfg)` (already returns built-in ∪ configured, sorted — use it for the error message too) |
| `autopilot.merge_mode` | one of `config.ValidAutopilotMergeModes` | sibling enum key — swept in for consistency (see Assumptions #9) |

Error message shape (example; exact wording is the implementer's, but it MUST name key, value, and accepted set):

```
ERROR: invalid value "weird" for dispatch.mode — must be one of: pane, native, headless
ERROR: invalid value "bogus" for agent.workers — not a provider in the effective providers table (available: agy, claude, codex, kimi)
ERROR: invalid value 500 for dispatch.column_width — must be an integer from 1 to 99
```

The effective providers table is computed from the **same cascade the read path uses** from the current cwd: inside a project, project ⊕ system ⊕ env; outside a project, system ⊕ env (via `config.LoadLayersNoProject`), always ∪ built-ins. Where the validation lives (cmd layer vs `configupgrade`) is the implementer's call; `configupgrade` must not grow a dependency on cwd/project discovery, so the provider list is most naturally computed in `cmd/fab` and passed in (or the whole domain check lives in `cmd/fab` before calling `Set`/`SetSystem`).

Scope boundary: the **read path stays fail-open and provider-neutral** — the resolver still passes an unknown provider through verbatim and `GetDispatchMode` still degrades with a warning. Only the `set` write gate gets stricter. A hand-edited file can still hold an out-of-range value; that is the read path's warn-and-degrade job. `fab config unset` is unchanged (it never inspects the value — it's the repair path).

### 3. Panic fix — `GetDispatchMode` nil writer

Root cause: `src/go/fab/internal/config/config.go:1010` calls `fmt.Fprintf(warnw, …)` directly. `warnw` is a package var that is **nil by design** ("nil means os.Stderr, resolved at CALL time (warnf)"); every other warning site goes through `warnf`, which substitutes `os.Stderr`. Fix: route the warning through `warnf(...)`. Behavior after fix: `fab agent review -o yaml` with `dispatch.mode: weird` prints `fab: warning: invalid dispatch.mode "weird"; using "native"` on stderr and resolves with `native`, exit 0. Add a regression test with `warnw` left nil (the in-package tests currently assign `warnw`, which is why this never fired in tests) — e.g. a test in `cmd/fab` or one that resets `warnw = nil` and captures `os.Stderr`. Sweep: grep `internal/config` (and siblings) for any other direct `warnw` use.

### 4. Shadow warning outside a fab project

Root cause: `effectiveTierFor` (`src/go/fab/cmd/fab/config.go`, used by both `warnIfShadowed` for `set` and `liveTierNotice` for `unset`) starts with `resolve.FabRoot()` and returns `ok=false` on **any** error — so outside a project (hexokit's daemon cwd = `$HOME`) the read model never loads and the warning silently never fires. `fab config show` already handles this correctly: on `resolve.ErrNoFabRoot` it degrades to `config.LoadLayersNoProject()`. Fix: extract the show command's layer-loading branch (FabRoot → `LoadLayers(projectPath)`, `ErrNoFabRoot` → `LoadLayersNoProject()`, other error → error) into one shared helper and use it in both `configShowCmd` and `effectiveTierFor` (keeping `effectiveTierFor` fail-open on real errors). This fixes the `set --system` warning **and** the sibling `unset --system` "live in env …" notice outside a project. Env is NOT intentionally excluded — the help text stays as is (it's correct once the bug is fixed).

Note: `effectiveTierFor` currently re-runs the full loader (re-emitting any fail-open `fab: warning:` lines from a malformed system file on stderr a second time after a `set`). Not in scope to change, but don't make it worse.

### 5. Inherited role values — decision: JSON carries the real source (`derived_from`); text form unchanged (follow-up)

The `agent.profiles.<role>.{provider,model,effort}` rows in the **default tier** are not literal built-ins — `readModelDefaults` → `configref.DefaultsMapFor(cfg)` derives them by resolving each role against the live config (`agent.ResolveRole` on knob-only / fill-only copies). So when `agent.workers: agy` lives in the system file, `agent.profiles.doing.provider = agy` is reported with origin `default`, which is true of the *tier* but hides *why* the value is `agy`.

Decision: in the JSON rows, a leaf `agent.profiles.<role>.{provider|model|effort}` whose **winning tier is `default`** carries an extra `derived_from` object naming the config key that actually determined the value and that key's own winning tier:

```json
{
  "key": "agent.profiles.doing.provider",
  "value": "agy",
  "origin": "default",
  "origin_detail": "",
  "status": "effective",
  "derived_from": {"key": "agent.workers", "origin": "system", "origin_detail": "/home/u/.fab-kit/config.yaml"}
}
```

- The source key comes from `agent.ResolveRoleWithSource` (`Source.Provider` = `agent.workers` / `agent.session` / `agent.profiles.<role>` / `built-in`; `Source.Model`/`Source.Effort` = the provider-fill rung path such as `providers.agy.profiles.doing` or `providers.agy.profiles.default`). For model/effort, `derived_from.key` is the full leaf path (`providers.agy.profiles.doing.model`).
- `derived_from.origin`/`origin_detail` are resolved by running that key through the same provenance descent (`effectiveTierFor`-style, but against the already-loaded layers — do not reload).
- When the source is `built-in` (no user config involved — e.g. no knob set anywhere), **omit** `derived_from`: `origin: default` is then the whole truth.
- Applies to `origin: default` rows only — a row won by env/system/project is a literal user value and carries no `derived_from`. In the keyed full-stack form, the `default` row (effective or shadowed) for these leaves carries it too.
- Resolution must be consistent with `DefaultsMapFor`'s own derivation (provider leaf from the knobs-only config, model/effort from the fills-only config) so `derived_from` never names a key that didn't produce the shown value.

**Follow-up (not in this change)**: the text `show --origin` still prints `# default` for these rows. Annotating it (e.g. `# default (via agent.workers: <system path>)`) is a text-contract change better made separately; record it as a follow-up.

### 6. Documentation

- `src/kit/skills/_cli-fab.md` § fab config (lines ~387–460): add `--json` to the `show` signature block and the four-form table; document the row schema (`key`, `value`, `origin`, `origin_detail`, `status`, optional `derived_from`), the `[]` / ancestor-null edge cases; add the `set` value-domain rules (accepted sets and ranges) to the `set` entry. This partial is the deployed owner of the CLI contract (Constitution: CLI change ⇒ update the owning `_cli-fab*.md` partial).
- `docs/specs/config.md`: add the `show --json` shape and the `set` validation rule next to the existing `show`/`set` sections. Specs are human-curated — keep the addition minimal and factual (design intent: JSON is the machine contract hexokit consumes; validation is write-time only, read path stays fail-open).
- `fab config show` / `fab config set` cobra `Long`/`Example` help strings in `cmd/fab/config.go` (add `--json` example; mention validation in `set`). Check the shll toolkit standards for help/`--json` conventions (`shll standards`) before finalizing the flag text.
- `src/go/fab/cmd/fab/skill.md` if it restates the show forms (grep).
- Memory (hydrate): `_shared/configuration.md` (the `fab config` surface).

## Affected Memory

- `_shared/configuration`: (modify) `fab config show --json` forms and row schema (incl. `derived_from`); `fab config set` value-domain validation (write-time only, read path still fail-open); `GetDispatchMode` warns via `warnf` (no panic); set/unset shadow notices work outside a fab project (shared project-or-projectless layer loader)
- `distribution/kit-architecture`: (modify) only if it restates the `fab config` verb surface or the `show` forms — check and sweep

## Impact

- **Go — `src/go/fab/cmd/fab/config.go`**: `configShowCmd` (`--json` flag, shared layer loader), the provenance walk (`originLine`, `flattenOrigin`, `stackLines`, `keyedOriginLines`, `renderOriginLines`) refactored to structured rows + two renderers, `configSetCmd` (domain validation), `effectiveTierFor` (project-less fallback).
- **Go — `src/go/fab/internal/config/config.go`**: `GetDispatchMode` (`warnf`, shared valid-mode set), possibly exported dispatch range constants.
- **Go — `src/go/fab/internal/configupgrade/mutation.go`**: possibly the domain check hook (or none if validation lives in cmd/fab).
- **Go — `src/go/fab/internal/agent`**: read-only use of `ProviderNames`, `ResolveRoleWithSource`.
- **Tests**: `cmd/fab/config_show_init_test.go`, `cmd/fab/config_test.go`, `internal/config/config_test.go`, `internal/configupgrade/configupgrade_test.go` — all with `t.Setenv("HOME", t.TempDir())` isolation (never the real `~/.fab-kit/config.yaml`); a project-less cwd case (`t.Chdir` to a temp dir) for the shadow warning and the provider-table computation.
- **Docs**: `src/kit/skills/_cli-fab.md`, `docs/specs/config.md`, help strings, memory at hydrate.
- **Consumer**: hexokit change `261030-wf1x-fab-system-config-agents-tab` will depend on the released binary; this change does not touch hexokit.
- No migration (no on-disk data restructuring). No new dependencies.

## Open Questions

- None blocking. The JSON shape, flag name, and the `derived_from` decision are recorded as Confident/Tentative assumptions below; `/fab-clarify` can revisit them.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flag is `--json` on `fab config show`, a pure format switch across all four show forms (bare, keyed, `--origin`, keyed `--origin`) | Brief leaves the name to us "fitting existing conventions"; `fab config explain --json` is the direct sibling precedent | S:85 R:80 A:90 D:85 |
| 2 | Confident | `--origin --json` emits a flat JSON array of rows `{key, value, origin, origin_detail, status[, derived_from]}`; one schema for both bare and keyed forms; bare rows all `status: effective` | Mirrors the text lines 1:1 so text/JSON can't diverge; brief asks for winner-first stack with effective/shadowed marks and a stable shape | S:75 R:60 A:70 D:60 |
| 3 | Confident | `origin_detail` is the env var name without `$`, the absolute file path, or `""` for default — always present | Brief's example is `FAB_AGENT_WORKERS` (no `$`); always-present keeps the shape stable for the consumer | S:80 R:70 A:75 D:70 |
| 4 | Confident | Keyed `--origin --json` for a key no tier defines prints `[]`; an ancestor-replaced descendant prints one `value: null` row with the ancestor's tier | Mirrors the existing text edge cases; empty array is the natural "no tiers" encoding | S:60 R:75 A:75 D:65 |
| 5 | Certain | Panic fix routes `GetDispatchMode`'s warning through `warnf` (nil `warnw` ⇒ os.Stderr) | Root cause verified at config.go:1010; every other site already uses `warnf`; the var's doc comment states nil means stderr | S:95 R:90 A:95 D:95 |
| 6 | Certain | Shadow-warning fix: `effectiveTierFor` uses the same project-or-projectless layer loader as `show` (extracted helper); help text unchanged | Reproduced: warning fires inside a project, silent outside; env is not intentionally excluded; also fixes the sibling `unset` live-tier notice | S:90 R:85 A:90 D:90 |
| 7 | Confident | Provider validation uses the effective providers table from the cwd cascade (`agent.ProviderNames(cfg)` = built-ins ∪ cascaded `providers:` keys); for a `--system` write from a project dir the project's providers count too | Brief says "effective providers table"; reusing the read cascade keeps one definition; the project-providers-in-a-system-write edge is minor and consistent with `show` | S:70 R:70 A:65 D:55 |
| 8 | Confident | Validation is write-time only (`fab config set`); read path stays fail-open (warn + default) and resolver stays provider-neutral; `unset` never validates | Read-path fail-open is a documented design rule ("must not brick every repo"); config.go/defaultsmap.go comments cite provider-neutral pass-through | S:75 R:75 A:80 D:75 |
| 9 | Tentative | Also validate `autopilot.merge_mode` against `config.ValidAutopilotMergeModes` at `set` time | Sibling enum key with an existing exported set (Sibling Sweeps rule); brief didn't list it — easy to drop if unwanted <!-- assumed: sweep autopilot.merge_mode into set validation as a sibling enum key --> | S:40 R:80 A:60 D:50 |
| 10 | Confident | Ranges: `column_width` 1..99, `min_cols`/`min_rows` ≥ 1 with no upper bound — exactly the accessors' existing validity rules, bounds named once and shared | "Valid ranges" per brief; the accessors already define them and the registry descriptions document them | S:75 R:80 A:85 D:80 |
| 11 | Confident | Item 5 decision: JSON rows for default-tier `agent.profiles.<role>.{provider,model,effort}` carry an optional `derived_from {key, origin, origin_detail}` from `agent.ResolveRoleWithSource`; omitted when the source is `built-in`; text `--origin` unchanged and its relabel recorded as a follow-up | Brief invites doing it here if JSON should carry it; hexokit's UI needs the honest source; text change is a separate contract change | S:65 R:65 A:60 D:50 |
| 12 | Confident | Docs: `_cli-fab.md` § fab config is the deployed owner of the contract; `docs/specs/config.md` gets a minimal addition; memory `_shared/configuration` at hydrate | Constitution CLI⇒docs rule names `_cli-fab.md` for core commands; brief names `_cli-fab` + docs/specs | S:85 R:85 A:90 D:85 |
| 13 | Certain | Tests isolate HOME with `t.Setenv("HOME", t.TempDir())` and never touch the real `~/.fab-kit/config.yaml`; no migration needed | Brief constraint; existing tests use this pattern; no on-disk data restructuring | S:95 R:90 A:90 D:95 |

13 assumptions (4 certain, 8 confident, 1 tentative, 0 unresolved).
