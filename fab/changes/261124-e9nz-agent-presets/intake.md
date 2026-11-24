# Intake: Agent Presets

**Change**: 261124-e9nz-agent-presets
**Created**: 2026-11-24

## Origin

> Execute step F1 (Agent presets: definitions table, agent.preset key, fab agent --preset, fab agent presets --json, built-in presets, precedence rule) of the plan at /home/sahil/code/sahil87/hexokit/fab/plans/sahil/26-11-24-agent-presets.md. Read the whole plan first (Decisions and Rules bind you), then implement only the F1 Brief.

Interaction mode: one-shot `/fab-new` driven by a cross-repo plan document (hexokit repo, written 2026-11-24
from a `/fab-discuss` session). The plan's **Decisions** table and **Rules** are binding and are restated
below; only the **F1 Brief** is in scope here. H1/H2 (hexokit) and R1 (release) are out of scope.

F1 open decisions asked during intake (`/fab-new` SRAD) and answered by the user:

1. Built-in presets: **`claude` / `codex-workers` / `codex-review`** with the exact shapes in § Built-in presets.
2. Reserved value: **yes — `none`** means "no preset here, even if a lower layer chose one".
3. `fab agent --preset X`: **resolution only** — it does NOT export `FAB_AGENT_PRESET` into the launched
   session (unlike `--workers`), matching the brief's flag caveat.

Follow-up design questions, also answered by the user (all took the recommended option):

4. The precedence splice lives at the **config loader** (the only place layers are distinct), with provenance
   carried on `Config` for `internal/agent`'s `source.*` labels.
5. `fab agent presets --json` is **sparse**, has **no `active` flag**, and its `source` enum gains **`env`**.
6. Preset-supplied fields are labelled **`agent_presets.<name>`** in `-o yaml` `source.*`, plus a top-level
   `preset:` key when a preset is active.
7. **Yes** to dotted `fab config set/unset agent_presets.<n>.<field>` leaves (clis precedent), and **yes**
   to rejecting `--preset` with selector-less `--cli` as a usage error.

The remaining two F1 open decisions were decided by the agent (see Assumptions): the table sits at the
**top level** as `agent_presets`, and the list verb is **`fab agent presets`** (sibling of `fab agent models`).

## Why

Choosing which agent runs which role is harder than it should be:

1. **One scope.** The only durable place to choose is the machine-wide `~/.fab-kit/config.yaml` (HexoKit
   Settings > Agent). There is no way to say "this group runs on Codex workers" or "this workspace is all
   Claude".
2. **Too many parts for a simple choice.** A choice spans two knobs (`agent.session`, `agent.workers`),
   sparse per-role overrides (`agent.profiles.<role>.{cli,model,effort}`) and per-CLI fills
   (`clis.<cli>.profiles.<role>`), resolved across four config layers.
3. **One-off overrides do not reach every dispatch mode.** `fab agent --cli/--model` binds only the native
   Agent-tool arm. `fab dispatch start` re-resolves from config (`agent.Resolve(cfg, stage)` in
   `src/go/fab/cmd/fab/dispatch_start.go`), so a flag never reaches pane or headless workers. Only config does.

Other tools solve this with named presets chosen at launch (vibe-kanban agent variants, Codex profiles, Roo
Code per-mode profiles, Amp's dial). A **preset** — a NAME for a whole role assignment, selected by one scalar
key — gives HexoKit a single value to pass per tab (`FAB_AGENT_PRESET`) and gives CLI users one line of
config. Because selection is config (env layer included), it reaches `fab dispatch start` in all three
dispatch modes, which a flag cannot.

If we don't do this, HexoKit H1/H2 (group/workspace agent presets, Settings dropdown) have nothing to build
on, and fab-kit would have to be parsed/rewritten by HexoKit — which the binding ownership decision forbids.

**Why this approach**: the plan's decisions rule out a new role vocabulary, changes to the stage→role
mapping, or a live cascade. A preset is a sparse bundle of fields fab already has, so the resolver and all
consumers keep their existing semantics; the preset only adds a rung.

## What Changes

### Binding decisions restated (from the plan — do not reopen)

| Decision | Choice |
|---|---|
| Ownership | fab-kit keeps the schema, the cascade and both config files. HexoKit reads/writes only through the `fab` CLI. fab must work without HexoKit. |
| What a preset is | A named, **sparse** set of fields fab already has: `session`, `workers`, and `profiles.<role>.{cli,model,effort}`. No new role vocabulary, no change to stage→role mapping. |
| Selection | One scalar key `agent.preset`, scope `both`. Env form `FAB_AGENT_PRESET` comes from fab's existing mechanical env layer (`FAB_` + uppercased dotted key) — **no special case**. |
| Precedence | Exactly **one** preset is active: the winner of `agent.preset` across env > system > project. Its values enter **at the layer where it was chosen, directly below that layer's own explicit `agent.*` fields**. Invocation flags stay on top. |
| Unknown name | At resolve time: a `fab: warning:` and the key is treated as **unset**. `fab config set agent.preset <name>` **rejects** a name that is not defined. A spawn must never fail for a stale name. |
| Key casing | snake_case keys; plural for a definitions table (`agent_presets`), singular for the selector (`agent.preset`). CLI flags kebab-case (`--preset`). |
| Not in F1 | Changing existing keys, the stage→role mapping, or the `clis` table. Storing a preset on a change (`.status.yaml`). |

### 1. Definitions table — top-level `agent_presets`

A new top-level config key `agent_presets`: a map of preset name → sparse definition. Scope `both`
(definable in system and project files; also reachable via env `FAB_AGENT_PRESETS` mechanically, like
`clis`). Definitions **merge per key across layers** like other maps (`MergeLayers` per-leaf deep merge:
built-in < project < system < env), so a project can add a preset and a system file can tweak one field of a
built-in.

Each definition may set only:

```yaml
agent_presets:
  <name>:
    description: <one line, optional — shown by `fab agent presets`>
    session: <cli>            # same meaning as agent.session
    workers: <cli>            # same meaning as agent.workers
    profiles:                 # same meaning/shape as agent.profiles
      <role>: { cli: <cli>, model: <id>, effort: <level> }
```

`<role>` is one of the six fixed roles (`default`, `operator`, `doing`, `review`, `hydrate`, `fast`). The
reserved name `none` (see § 2) cannot be defined.

Typed shape: add a `Config.AgentPresets map[string]AgentPreset` (`yaml:"agent_presets"`) in
`src/go/fab/internal/config/config.go`, where `AgentPreset` = `{Description string; Session string;
Workers string; Profiles map[string]RoleProfile}` (reusing the existing `RoleProfile`).

### 2. Built-in presets in `src/go/fab/defaults.yaml`

Ship exactly these three (user-confirmed), as a new top-level block in `defaults.yaml`:

```yaml
agent_presets:
  claude:
    description: All roles on Claude (today's defaults)
    session: claude
    workers: claude
  codex-workers:
    description: Pipeline workers on Codex; session on Claude
    workers: codex
  codex-review:
    description: Only the review role on Codex
    profiles:
      review: { cli: codex }
```

Built-in presets are **inert** until selected (presence=intent: a fresh install with nothing set resolves
byte-identically to today). `codex-review` sets only `cli`, so review's model/effort re-derive from codex's
own fills (`clis.codex.profiles.review` → `gpt-5.6-sol` / `xhigh` today).

The built-in tier lives in `internal/agent` (parsed from the embedded `defaults.yaml`); `internal/config`
cannot import `internal/agent` (cycle). Follow the existing `DefaultDispatch*` precedent: `agent`'s
`init()` pushes the parsed built-in presets into a `config` package var (e.g. `config.BuiltinAgentPresets`)
that the loader composes as the lowest tier of the definitions merge.

### 3. Selector — `agent.preset` (and the reserved `none`)

- New scalar `agent.preset` in `AgentConfig` (`yaml:"preset"`), scope `both`, default unset.
- Registered in `src/go/fab/internal/configref/configref.go` with description + scope, **advertised in the
  reference fence** (the `agent:` fence segment gains a commented `# preset: codex-review` style line and a
  `Settable machine-wide: fab config set --system agent.preset <name>` line) — follow the existing
  `agent.session` / `agent.workers` rows.
- `FAB_AGENT_PRESET` works through the existing env layer (`loadEnvLayer` walks `configscope.DottedKeys()`);
  registering the key is sufficient — **no special-case env code**.
- **Reserved value `none`** (user-confirmed): `agent.preset: none` (or `FAB_AGENT_PRESET=none`) participates
  in the selection like any name — it wins across env > system > project — but activates **no** preset.
  This lets a tab (env) or a machine opt out of a lower layer's preset. A definition named `none` is
  rejected/ignored with a warning; `fab agent presets` never lists it. An *empty* `FAB_AGENT_PRESET=` stays
  "unset" per fab's empty-skip rule.

### 4. Resolution — the precedence rule

**The rule.** Let L ∈ {env, system, project} be the highest layer whose `agent.preset` is non-empty.
- If none, or L's value is `none` → no preset; resolution is byte-identical to today.
- If L's value names an undefined preset → `fab: warning: unknown agent preset "<name>" (agent.preset from
  <layer>); ignoring it` on stderr, and resolve as if unset. Never an error.
- Otherwise the preset's fields are spliced in **at layer L, directly below L's own explicit `agent.*`
  fields**: effective tree = `MergeLayers(project', system', env')` where only layer L is replaced by
  `MergeLayers({agent: presetFields}, L)`. Consequences:
  - The preset beats every explicit `agent.*` field in layers **below** L (e.g. env-chosen preset beats a
    system `agent.workers: claude`).
  - An explicit `agent.*` field **at L** beats the preset (so HexoKit's "Customize" grid at the system layer
    still wins over a system-chosen preset).
  - Layers **above** L beat the preset as they do today.
  - Invocation flags (`--cli`/`--model`/`--effort`) stay on top.
- **A CLI set by a preset re-derives model/effort from that CLI's fills** exactly as an
  `agent.profiles.<role>.cli` override does today (the preset's `profiles.<role>.cli` lands in
  `agent.profiles.<role>.cli`, and the existing resolver derives `<p>` fills from the resolved CLI). A
  preset `workers: codex` likewise lands in the depth knob.

**Where.** The brief says "in `internal/agent`, at the single resolve seam, so `fab agent`, `fab dispatch
start` (all three dispatch modes), `fab batch` and the operator launcher all see it." Layer identity only
exists in the loader (`internal/config` `fromProjectMap` / `layersFrom` → `MergeLayers(project, system,
env)`); `internal/agent` receives an already-merged `*config.Config`. So the splice is implemented **once**
(user-confirmed) at the config loader seam that every consumer already goes through (`config.Load` / `LoadPath` /
`LoadNoProject` / `LoadLayers` — the same tree that `fab config show` reports), and the merged `Config`
carries preset provenance (e.g. `Config.Agent` keeps `Preset`, plus a `yaml:"-"` record of the active
preset name, its chosen layer, and which `agent.*` leaves the preset supplied) that
`internal/agent.ResolveRoleWithSource` reads to label `source.*`. This satisfies the brief's intent (one
seam, every consumer sees it, `dispatch start` picks it up from config) without duplicating the cascade.


All existing consumers then see the preset with no per-caller change: `fab agent`, `fab resolve-agent`,
`fab dispatch start` (`agent.Resolve(cfg, stage)` — pane, native and headless), `fab batch`
(`config.LoadPath`), `fab pane open`, `internal/spawn`, `internal/setupcheck`. Verify each call site loads
through the spliced path; any that builds a `Config` some other way must be routed through it.

### 5. `fab config set` / `unset` for `agent.preset`

- `fab config set [--system] agent.preset <name>` validates `<name>` against the **defined** presets (built-in
  ∪ system ∪ project ∪ env definitions as currently resolved) plus the reserved `none`; an undefined name is a
  non-zero usage error naming the defined set (sorted). This is the one place an unknown name is an error.
- `fab config unset [--system] agent.preset` restores inheritance (existing unset behavior).
- `agent_presets.<name>.<field>` dotted leaves: make them settable/unsettable the same way
  `clis.<name>.<field>` dotted leaves are today (`configref.go` dynamic-key matching around lines
  1294–1394: `clis` → mapping, `clis.<n>.<field>` → string, `clis.<n>.profiles.<role>.<field>` → string).
  Shape: `agent_presets.<n>` mapping; `agent_presets.<n>.{description,session,workers}` string;
  `agent_presets.<n>.profiles` mapping; `agent_presets.<n>.profiles.<role>` mapping;
  `agent_presets.<n>.profiles.<role>.{cli,model,effort}` string.

### 6. Invocation — `fab agent [role|stage] --preset <name>`

- New flag `--preset <name>` on `fab agent` only — not on the deprecated, byte-stable `fab resolve-agent`.
- Semantics: **equivalent to `FAB_AGENT_PRESET=<name>` for this one resolution** — the preset is chosen at the
  env layer (beats system/project explicit fields, loses to explicit env `FAB_AGENT_*` fields), and
  `--cli`/`--model`/`--effort` still apply on top. An unknown name warns and resolves as unset (same as
  config). `--preset none` disables any configured preset for this call.
- **Resolution only** (user-confirmed): it does NOT export `FAB_AGENT_PRESET` into an exec'd session (unlike
  `--workers`, which sets `FAB_AGENT_WORKERS`). Help text MUST say so, restating the existing caveat: flags do
  not reach `fab dispatch start`; to make a preset reach dispatched workers set `agent.preset` in config or
  export `FAB_AGENT_PRESET`.
- Interaction with the bare `--cli` form (no selector — bypasses role resolution): `--preset` has no effect
  there; reject the combination with a usage error rather than ignore it silently (mirrors how `--template`
  rejects `--model/--effort`).

### 7. Visibility

**`fab agent presets [--json]`** — new subcommand of `fab agent`, sibling of `fab agent models`
(`cmd/fab/agent_models.go` is the template, including `--repo`). Lists every defined preset (built-in ∪
system ∪ project ∪ env-defined, merged per key), sorted by name; `none` never listed.

- Human output: a table — name, source, description, and a compact summary of the fields it sets.
- `--json`: **a contract HexoKit reads — document it as such** in `_cli-fab.md`. An array (sorted by name):

```json
[
  {
    "name": "codex-review",
    "description": "Only the review role on Codex",
    "source": "built-in",
    "profiles": { "review": { "cli": "codex" } }
  },
  {
    "name": "codex-workers",
    "description": "Pipeline workers on Codex; session on Claude",
    "source": "built-in",
    "workers": "codex"
  }
]
```

  `source` ∈ `built-in | system | project | env` — the **highest** layer that contributed to the
  definition (a system tweak of a built-in reports `system`; a definition supplied or tweaked via
  `FAB_AGENT_PRESETS` reports `env`). `env` extends the plan's `built-in | system | project` enum
  (user-confirmed). Fields a preset does not set are **omitted** (sparse), never emitted empty. No "active"
  flag (user-confirmed): the contract is `{name, description, source}` plus the set fields.

**`fab agent <role|stage> -o yaml`** reports the active preset:
- a top-level `preset: <name>` key (omitted when no preset is active — follow the existing optional-key
  style of `dispatch:`);
- `source.cli` / `source.model` / `source.effort` name the preset for each field it supplied, e.g.
  `source.cli: agent_presets.codex-review` (the dotted path to the definition, matching the existing
  `agent.profiles.review` / `clis.codex.profiles.review` style). A model/effort the preset did not set but
  which was re-derived from the preset-chosen CLI's fills keeps its `clis.<p>.profiles.<role>` source.

**`fab config show --origin`** (and `--json`): `agent.preset` appears like any other key with its layer
origin. Each `agent.*` leaf the preset supplied shows the preset as its origin detail (e.g. origin = the layer
where the preset was chosen, detail = `preset codex-review`). `fab config show` (bare) reports the
**spliced** effective tree, so what `show` prints equals what consumers resolve. `agent_presets` itself shows
with normal per-leaf origins.

### 8. Docs and guards

- `docs/specs/config.md` — `agent_presets` table, `agent.preset` selector, reserved `none`, the precedence
  rule, set-time validation vs resolve-time warning.
- `docs/specs/stage-models.md` — § Fill precedence gains the preset rung (the preset's fields enter at the
  choosing layer, below that layer's explicit fields; flags on top); § Structured resolution gains the
  `preset` key and preset-named `source.*`; § Config schema mentions `agent_presets`.
- `fab config explain` — prose for `agent.preset` and `agent_presets` (registry descriptions in
  `configref.go`), plus the reference fence update.
- Deployed `src/kit/skills/_cli-fab.md` § fab agent — `--preset`, `fab agent presets [--json]` (with the JSON
  contract), `preset` in `-o yaml`. Constitution Principle V: restate rules, never cite `docs/specs/*`.
- Help-dump fixtures for the new flag/subcommand (`cmd/fab/help_dump*`), and any skill pages that enumerate
  `fab agent` flags.
- **Drift guards** extended to the built-in presets: `TestDefaultsFileIsWellFormed`
  (`internal/agent/defaults_test.go`) also validates every built-in preset (known roles, known built-in CLI
  names, only allowed fields, `none` not defined); a pin test for the three built-in presets' exact
  shapes; if docs restate the built-in set (stage-models.md / config.md table), extend
  `TestDocTablesMatchAgentMaps` (`stagemodels_doc_test.go`) or equivalent to cover it.

### 9. Tests (table tests for the precedence rule — required cases from the brief)

1. Preset alone (project `agent.preset: codex-workers`, nothing else) → workers roles resolve codex with
   codex fills; session roles stay claude.
2. Explicit field above it at the same layer (project `agent.preset: codex-workers` + project
   `agent.workers: agy`) → agy wins.
3. Env-chosen preset over system explicit fields (`FAB_AGENT_PRESET=codex-workers`, system
   `agent.workers: claude`) → codex.
4. Project preset shadowed by a system preset (project `agent.preset: codex-review`, system
   `agent.preset: codex-workers`) → only codex-workers applies (review stays on the workers knob = codex,
   doing on codex; nothing from codex-review is merged).
5. Unknown name → `fab: warning:` emitted, resolution equals the unset case.
6. A CLI swap inside a preset (`codex-review`) → review's model/effort come from `clis.codex.profiles.review`,
   source labels name the preset for `cli` and the codex fill for model/effort.
7. `fab dispatch start` picks the preset up from config (headless path test in `dispatch_start_test.go`
   with `agent.preset` set → composed command uses codex).

Plus: reserved `none` at a higher layer cancels a lower layer's preset; `fab config set agent.preset bogus`
fails, `... codex-review` succeeds; `fab agent --preset` equivalence with `FAB_AGENT_PRESET`; `--preset` +
bare `--cli` usage error; `fab agent presets --json` schema/sort/sparse output; `-o yaml` `preset` key;
`fab config show --origin` preset origin detail; no-preset byte-identity (existing golden tests must pass
unchanged).

## Affected Memory

- `runtime/clis-and-profiles`: (modify) add agent presets — definitions table, `agent.preset` selector, reserved `none`, precedence rule (choosing-layer splice), `--preset` flag, `fab agent presets [--json]` contract, `preset`/`source.*` in `-o yaml`
- `_shared/configuration`: (modify) `agent_presets` schema section, `agent.preset` under `#### agent`, set-time validation vs resolve-time warning, `show --origin` preset origin detail
- `distribution/kit-architecture`: (modify) only if it enumerates `defaults.yaml` blocks — note the new `agent_presets` block and the push-at-init var

## Impact

- **Go**: `src/go/fab/defaults.yaml`; `internal/config` (types, loader splice in `fromProjectMap`/`layersFrom`,
  provenance record, built-in presets var); `internal/agent` (parse/push built-in presets, source labels in
  `ResolveRoleWithSource`, drift-guard/pin tests); `internal/configref` (registry rows, dynamic dotted keys,
  reference fence, explain prose, `DefaultsMap` if it projects built-ins); `cmd/fab/agent.go` (`--preset`),
  new `cmd/fab/agent_presets.go` (+ test), `cmd/fab/config*.go` (set validation, `show --origin` detail),
  `-o yaml` emitter, help-dump fixtures; `dispatch_start_test.go` case.
- **Kit**: `src/kit/skills/_cli-fab.md`; any skill page enumerating `fab agent` flags.
- **Docs**: `docs/specs/config.md`, `docs/specs/stage-models.md`; memory files above.
- **Contract**: `fab agent presets --json` becomes a cross-repo contract consumed by HexoKit H1/H2.
- **Behavior with nothing set**: unchanged (must be proven by existing golden/no-preset tests passing).
- No migration needed: all additions are new optional keys.

## Open Questions

- Does `fab batch` / the operator launcher (`internal/spawn`) build its `Config` exclusively via the loader
  functions, or does any path construct a `Config` directly (which would bypass the splice)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Plan's Decisions table (ownership, preset shape, scalar `agent.preset` scope both, env via mechanical layer, precedence, unknown-name handling, key casing) binds this change verbatim | User instruction: "Decisions and Rules bind you"; restated in What Changes | S:95 R:70 A:95 D:95 |
| 2 | Certain | Scope is F1 only — no H1/H2/R1 work, no `.status.yaml` preset storage, no changes to existing keys, stage→role map, or `clis` table | Explicit in user prompt and F1 "Not in this step" | S:95 R:80 A:95 D:95 |
| 3 | Certain | Built-in presets are `claude` {session, workers: claude}, `codex-workers` {workers: codex}, `codex-review` {profiles.review.cli: codex} with the given descriptions | Asked — user chose option 1 | S:95 R:75 A:90 D:90 |
| 4 | Certain | Reserved value `none` cancels lower-layer presets; cannot be defined; never listed | Asked — user chose "Yes, reserve none" | S:95 R:70 A:90 D:90 |
| 5 | Certain | `fab agent --preset` is resolution-only; does not export FAB_AGENT_PRESET; help text states the flag caveat | Asked — user chose "No — resolution only"; matches the brief | S:95 R:80 A:90 D:90 |
| 6 | Confident | `agent_presets` is a top-level key (not `agent.presets`) | Key-casing decision fixes the name `agent_presets`; nesting under `agent` would put `agent.preset`/`agent.presets` side by side and mix definitions with the selector | S:70 R:55 A:80 D:75 |
| 7 | Confident | List verb is `fab agent presets [--json]`, a sibling of the existing `fab agent models` | Plan's Goal, R1 and H1 all name `fab agent presets --json`; `fab agent models` is direct precedent | S:80 R:60 A:85 D:80 |
| 8 | Certain | Splice is implemented at the config loader (the only place layers are distinct), with preset provenance carried on `Config` for `internal/agent`'s `source.*` labels | Asked — user chose the config loader; brief's "in internal/agent" read as the single seam all resolvers share | S:95 R:60 A:90 D:90 |
| 9 | Confident | Built-in presets reach `internal/config` via an `init()` push from `internal/agent`, mirroring `DefaultDispatch*` | Import cycle forbids config → agent; existing documented precedent | S:60 R:75 A:85 D:75 |
| 10 | Confident | `--preset` is equivalent to choosing the preset at the env layer for that one resolution; flags `--cli/--model/--effort` stay on top | Plan: "a preset chosen for a tab (env)" semantics; "Invocation flags stay on top"; keeps `fab agent --preset X` ≡ HexoKit's per-tab FAB_AGENT_PRESET | S:60 R:70 A:70 D:60 |
| 11 | Certain | `--preset` combined with selector-less `--cli` is a usage error | Asked — user confirmed. Bare `--cli` bypasses role resolution so the preset could never apply; mirrors `--template` rejecting `--model/--effort` | S:95 R:80 A:90 D:90 |
| 12 | Confident | `--preset` added to `fab agent` only, not to deprecated `fab resolve-agent` | resolve-agent is a byte-stable compatibility projection; brief names only `fab agent` | S:50 R:85 A:65 D:60 |
| 13 | Certain | `agent_presets.<name>.<field>` dotted leaves are settable via `fab config set/unset`, following the `clis.<name>.<field>` precedent | Asked — user confirmed. Plan notes the clis dotted-leaf precedent for map-valued keys; brief does not require it explicitly | S:95 R:75 A:90 D:90 |
| 14 | Certain | `-o yaml` source label for preset-supplied fields is `agent_presets.<name>`; a top-level `preset:` key is emitted only when a preset is active | Asked — user confirmed. Matches existing dotted-path source labels (`agent.profiles.review`, `clis.codex.profiles.review`) and the optional-key style of `dispatch:` | S:95 R:70 A:90 D:90 |
| 15 | Certain | `fab agent presets --json` omits fields a preset does not set, carries no "active" flag; `source` = highest contributing layer, enum built-in, system, project, env | Asked — user confirmed. Brief specifies `{name, description, source}` plus "the fields the preset sets" — sparse reading; extra fields would widen a cross-repo contract | S:95 R:55 A:90 D:90 |
| 16 | Confident | Unknown preset name at resolve time warns on stderr with `fab: warning:` and resolves as unset; `fab config set` rejects it with a non-zero exit naming the defined set | Binding decision; existing `fab: warning:` fail-open loader convention | S:85 R:70 A:85 D:80 |
| 17 | Confident | No migration needed — all additions are new optional keys; nothing-set resolution is byte-identical | Plan goal "a user who sets nothing sees no change"; presence=intent rule | S:80 R:75 A:85 D:85 |

17 assumptions (10 certain, 7 confident, 0 tentative, 0 unresolved).
