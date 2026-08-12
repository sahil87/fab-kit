# Config Schema — the per-field metadata table

> **Status:** Design intent (pre-implementation, Constitution VI). This spec is human-curated. It
> records the config-system schema decisions resolved in the 2026-07-08 `/fab-discuss` session and
> written up in the config-upgrade effort's backlog doc (`fab/plans/sahil/config-upgrade.md`, all six
> decisions user-confirmed). It is written across the **three-change** config-upgrade effort:
> **Change 1** (260708-ff2v) — the per-field metadata table + `fab config explain` restructure +
> `--json` — **Change 2** (260708-lpb5) — the four-tier environment > system > project > built-in cascade resolution + scope enforcement +
> the `fab config show [--origin]` / `fab config init --system` visibility commands — and **Change 3**
> (260708-j0qm) — `fab config upgrade` + the managed fence + the `fab_version` → `fab/.fab-version`
> relocation + the migration + registry-driven `fab config init --project` (scaffold config.yaml
> deleted) — are all landed here in authoritative detail. The whole config-upgrade design authority
> lives in one place.
>
> The canonical schema is the Go field table in `src/go/fab/internal/configref/`; this doc is its
> human-readable rationale. Defaults that have a Go symbol are sourced from that symbol, never
> restated here or in the table. The values behind those symbols are single-sourced in
> `src/go/fab/internal/agent/defaults.yaml` (embedded into the binary via `go:embed`). The embedded
> census is exactly: `defaults.yaml` (values) + `internal/configref` (schema/prose) +
> `internal/configscope` (scope taxonomy) + `src/kit/scaffold/` (non-config files) — no stub copy of
> the config exists anywhere.

`fab/project/config.yaml` is the single project-config file the `fab` binary and the markdown skills
read. This spec fixes how its schema is modeled: not as prose, but as an ordered **per-field metadata
table** from which every rendering (the commented-YAML `fab config explain`, the `--json` dump, and
— in later changes — the cascade resolver and the `fab config upgrade` fence) is generated. One
source, no second copy to drift.

---

## Why a metadata table (invert the data/prose relationship)

`fab config explain` originally (change 6nke) rendered the schema from a text template with a few
constants injected: the prose was primary, and there was no machine-readable notion of a field's
default, scope, or override status. The config-upgrade effort needs exactly those: a cascade resolver
needs each field's canonical default and its per-field merge unit; `fab config init --system` needs
each field's `scope`; `fab config upgrade`'s fence generator needs `advertise` and `renamed_from`.

Change 1 inverts the relationship: a **per-field metadata table is primary**, and both the commented
YAML and the JSON are generated from it. The no-drift invariant the template established is preserved —
defaults that have a canonical Go symbol are still referenced from that symbol, never copied. The
table adds *structure*, never a second copy of *values*.

---

## The per-field schema

Each row of the table models one **override unit** — a meaningful override surface, coarser than a
leaf key. Map-valued fields (`providers`, `agent.profiles`, `stage_hooks`) are single rows with
structured defaults, matching the per-field deep-merge semantics [Change 2] uses (maps merge per-key,
lists replace, scalars replace).

| Field | Meaning |
|-------|---------|
| `key` | Dotted path of the override surface (e.g. `agent.profiles`, `project.name`, `true_impact_exclude`). The identity used by the JSON dump and the JSON↔YAML key-parity guard. |
| `default` | The **canonical** built-in default (typed). What the cascade [Change 2] falls back to when no layer overrides the field. A field with no built-in default carries `null` — uniformly, never a typed empty (`[]`/`{}`/`""`). See § Default semantics. |
| expected kind | Internal YAML shape (`string`/`bool`/`int`/`float`/`null`/`sequence`/`mapping`) used for registry resolution and input validation. `fab config set` accepts only scalar leaves whose kind is `string`/`bool`/`int`/`float`; collection kinds remain discoverable for `explain` and environment overlays. This does not alter the established `--json` wire shape. |
| `description` | One-line summary of the field. Required (non-empty) — the registry lint rejects an empty description. Feeds the JSON dump and, later, the generated comment scaffold [Change 3]. |
| `scope` | Override visibility across the cascade layers: `project` / `system` / `both`. See § Scope taxonomy. |
| `advertise` | The "C flag": whether [Change 3]'s managed fence scaffolds this field as a commented reference when it is not overridden. See § Advertise semantics. |
| `renamed_from` | Previous key path for mechanical rename carry-forward. Set on `agent.profiles` (`agent.tiers`) as of 260806-j9nh; `""` on every other row. See § renamed_from. |
| `init-seed` | Whether the field is an A-class **identity** field written LIVE at `fab config init --project` time ([Change 3]) — `project.name`/`project.description`/`source_paths`/`test_paths`. The generator's live block above the fence; every other field is fence territory from day one. Consumed by the init generator, NOT exposed in the `--json` schema dump (like the rendered YAML segment). |

### Defaults are sourced from canonical Go symbols — no second copy

Every default that has a canonical Go symbol is referenced from it, not copied: the claude session
command from `agent.DefaultInteractiveCommand`, the per-role profiles via `agent.DefaultProfile` over
`agent.RoleNames()`, the stage names via `agent.StageNames()`. Those symbols are projections of one
values file, not independent constants: the built-in tier's values — the two depth knobs' `claude`,
the three `dispatch` defaults (`mode: native`, `column_width: 35`, `reap_done: true`), and the four
providers' capability grammars and role fills — live in `src/go/fab/internal/agent/defaults.yaml`,
embedded into the binary via `go:embed` and parsed once. The three dispatch values reach
`internal/config` through its exported `DefaultDispatchMode` / `DefaultDispatchColumnWidth` /
`DefaultDispatchReapDone`, which are **package-level vars carrying no literal of their own**, assigned
by `internal/agent`'s `init()` from the parsed `defaults.yaml`. The push direction is cycle-forced:
agent imports config, so config can never read the values back from agent — assigning into config at
init is the only direction the import graph allows. The nil-safe accessors
(`GetDispatchMode`/`GetDispatchColumnWidth`/`GetDispatchReapDone`) and every other consumer read the
same exported symbols as before. The registry construction fails loud
(returns an error rather than emitting a degraded reference) if a role reported by `RoleNames()` does not
resolve through `DefaultProfile`, or a row has an empty description or
an invalid scope — the same fail-loud discipline the pre-metadata-table renderer applied to its invariants.

### Canonical default vs. rendering example

The `default` is the *canonical* built-in default, **not** the value the reference happens to show as
an example. `source_paths` and `test_paths` render an example (`- src/`, `- "**/*_test.go"`) because a
bare empty list is useless documentation — but their **binary default is empty**, so their `default`
is `null`. The example lives in the field's rendered segment, not in the metadata. This distinction
is load-bearing for [Change 2]: the cascade must fall back to the *canonical* default (empty), never to
a rendering example.

### Default semantics — the uniform empty convention

A field with **no meaningful built-in default** carries `null` — uniformly, never a typed empty
(`[]`, `{}`, or `""`). `null` is the single "the cascade falls back to absent" signal [Change 2]'s
resolver consumes; distinguishing an empty list from an empty map from an empty string would leak a
Go-side implementation detail that carries no cascade meaning and would make `--json` emit
`null`/`[]`/`{}`/`""` inconsistently for the same "no default" concept. So a **non-null** `default`
always denotes a real built-in value (today: the `providers` row's **four built-in providers** —
claude/codex/agy/kimi, each with its capability grammar and per-role fills except kimi, which
deliberately ships none (`260808-rpsr`) — the resolved
`agent.profiles` defaults, the two depth knobs' `claude`, `dispatch.mode`'s `native`,
`dispatch.column_width`'s `35`, and `dispatch.reap_done`'s `true`); every other row is `null`.
**The three `dispatch` rows are
the convention's boundary cases and are deliberately not `null`**: mode is a real string default,
`native`; for width an absent YAML int is indistinguishable from `0` (which the accessor reads as
unset, alongside every other out-of-`1..99` value); and reap has a real `true` default. Each carries a
built-in value the cascade genuinely bottoms out at, not a typed-empty placeholder.
`dispatch.reap_done` is the sharpest of the three and the one that forced a **struct-level** answer as
well as a registry one: its built-in default is **`true`**, so the Go zero value means the *opposite* of
the default, and a plain `bool` would have made an absent key indistinguishable from an explicit
`reap_done: false` — silently disabling reaping for every project that never sets the key. It is
therefore modeled as a **`*bool`** in `internal/config.DispatchConfig` (`nil` = unset = `true`), the one
place the three siblings' shapes diverge; the registry row still carries the plain `true`, because the
*default* is a value, not a pointer. The empty-skip read model does not retire the pointer: `false` is a
real value that survives the merge, but the loader's merged tree carries no built-in-defaults tier (see
§ The defaults tier is materialized for the READ MODEL), so an unset key still reaches Unmarshal as
absent.
The same rule governs the **per-role fills** inside the `providers` default: a built-in's `profiles`
map is projected whenever it ships real fills (`260806-ywkx`), which three of the four do. The
convention applies one level down instead — claude's map is exhaustive (all six roles), while codex's
and agy's are **sparse**, and a role fab-kit ships no fill for is **omitted entirely** rather than
emitted as an empty object, since that would assert a built-in fill that deliberately does not exist
(the omitted roles resolve the provider's `default` entry). `kimi` is that rule taken to its limit
(`260808-rpsr`): it ships **no** fills at all — its `-m` takes a user-config model alias, so any pinned
value would break non-managed installs — so its whole `profiles` key is omitted from the projected
default and `profilesLines` renders no `profiles:` block for it in the reference, leaving the empty
model to drop the `-m` pair and kimi to inherit the user's own `default_model`. The **deprecated flat**
`providers.<name>.model`/`.effort` is likewise absent from every `default`: it exists only as a
read-time alias for `profiles.default` until the `2.16.19-to-2.17.0` migration rewrites a config.

### Section-level prose lives on the row — the segment

One-line `description`s cannot carry the narrative documentation blocks the reference needs (the
providers explanation, the per-provider dispatch/fill notes, the four built-in providers, the fixed
stage→role mapping). Each table row therefore carries — alongside its one-line `description` — the
**rendered YAML segment**: the field's commented block as it appears in the reference. `fab config
explain` is generated by walking the table and concatenating those segments in order; there is no
separate template. The `description` (the machine-readable one-liner, exposed in `--json`) and the
`segment` (the human-readable block, exposed in the YAML) are two projections of **one** row, not a
second copy of the schema to drift — a field's documentation is authored once, on its row. The rows for
map-valued fields (`providers`, the `agent:` block, `stage_hooks`) build their segment by interpolating the
same Go symbols their `default` reads, so the rendered prose carries no literal copy of any value.
The existing reference tests assert those blocks verbatim; the restructure preserves them byte-for-byte.

**Each row also carries the SHORT form of its segment (`ShortSegment`)** — the file-bound advert the
managed fence and the `--system` scaffold render, as distinct from the long `Segment`, which remains
the `fab config explain` surface, unchanged in depth. A short segment is a diet header over the same
YAML block: one to four short description lines with the row's scope tag (`[project]`/`[system]`/
`[both]`, from the `configscope` taxonomy) appended to the last, then two pointer lines — scope-`both`
adverts carry `# Settable machine-wide: fab config set --system <key> <value>` (one line per settable
key of the block), and every advert carries `# Full prose: fab config explain <key>` — followed by the
field's YAML lines, interpolating the same canonical symbols the `default` reads. The registry lint
pins the scope tag and both pointers against the row's metadata, so the annotations are sourced from
the row rather than re-typed. Shape (from the full-document golden tests, over a small synthetic field
set — `branch_prefix` there is a synthetic field name in the tests' own local table, not a live
registry key):

```
# branch_prefix — worktree branch prefix. [project]
# Full prose: fab config explain branch_prefix
# branch_prefix: ""
```

Generated files carry the diet form so a project's `config.yaml` stays scannable; the essays stay in
`fab config explain`.

**Several rows under one YAML block share a single segment.** Where two or more override units live
under the same top-level key, the segment belongs to the *first* of them and documents them all; the
rest carry an **empty** segment (`project.name` owns the `project:` block for `project.description` and
`project.linear_workspace`; `dispatch.mode` owns the `dispatch:` block for
`dispatch.column_width` **and** `dispatch.reap_done`). This is not an optimisation but a correctness requirement: the reference and
the managed fence render these blocks **commented**, with the documented instruction to uncomment a
whole block, so two separately-uncommentable `# dispatch:` parents would collide into a duplicate YAML
key. It also matches the fence generator, whose override detection is top-level-key scoped: a live
`dispatch:` block suppresses the advertisement of every key under it.

---

## Scope taxonomy (decision 6)

`scope` states which cascade tier(s) may override a field. The rationale: the **system** tier
(`~/.fab-kit/config.yaml`, [Change 2]) is restricted to *preference-class* fields — personal model/harness
choices — while *semantics-class* fields stay in the project file so the repo remains reproducible for
teammates and CI. That restriction is also what makes the system tier's precedence over the project file
(§ Override cascade) safe: the only fields it can outrank are the ones where "my machine beats this
repo's suggestion" is the intended answer.

| scope | Meaning | Fields |
|-------|---------|--------|
| `both` | Overridable in either the project or the system layer (preference-class). | `agent.session`, `agent.workers`, `agent.profiles`, `providers`, `dispatch.mode`, `dispatch.column_width`, `dispatch.reap_done` |
| `project` | Overridable only in the project file (semantics-class, repo-reproducible). | `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `consolidate.detectors`, and (conservative default) `stage_hooks` |
| `system` | Overridable only in the system layer. | *(none today; the value exists for completeness and [Change 2])* |

The one field the decision-6 taxonomy does not enumerate (`stage_hooks`) defaults to `project`
— the conservative choice, since system-visibility is opt-in per the same rationale. `dispatch`
(`dispatch.mode`, the pane/native/headless preference ceiling; `dispatch.column_width`, the pane-worker column's
width; `dispatch.reap_done`, whether a finished worker's pane is reclaimed) is `both` by the same
reasoning that puts `agent`/`providers` there: all three keys express how the
**operator** prefers to launch and observe stage workers on **this machine** — the adapter ceiling, how
much of the window that pane takes, and whether a done worker's pane lingers — not what the repo's
pipeline means, so they must be settable once
machine-wide. (`fab_version` was
machine-managed and, as of [Change 3 — landed], left `config.yaml` entirely for the plain-text sibling
`fab/.fab-version`; it is no longer a scoped/registry/config key and carries no scope. The compat-window
config.yaml `fab_version:` fallback that both reader stacks kept has been **closed** [260719-kq7v]: a stale
`fab_version:` in either the PROJECT or the SYSTEM file is now an inert unknown key — `Config.FabVersion` is
tagged `yaml:"-"`, so nothing unmarshals it and it can never bleed into a repo's resolved version.) Scope was metadata-only in
Change 1; **as of Change 2 it is enforced**: the cascade
resolver prunes a project-scoped field found in the system file and emits a `fab: warning:` (fail-open —
config must never brick), and `fab config init --system` scaffolds only the `system`/`both` fields. The
scope enum and the key→scope taxonomy are single-sourced in the leaf package `internal/configscope`
(consumed cycle-free by both the loader `internal/config` and the registry `internal/configref`, which
cannot import each other), so the taxonomy has exactly one definition. Re-classifying a field is still a
one-line data change (in `internal/configscope`).

### Dispatch preference and capability data

`dispatch.mode` accepts exactly `pane`, `native`, or `headless`; absent and invalid values resolve to
the canonical `native` default, with invalid input producing a `fab: warning:` diagnostic. It is a
preference ceiling: automatic resolution descends only through `pane → native → headless`. Provider
fields are independent capabilities—`interactive_command` for pane, `native: true` for the Agent-tool
adapter, `headless_command` for headless—and their presence never chooses policy. Claude ships all
three; codex, agy, and kimi ship pane/headless grammar without native capability. The
`2.17.3-to-2.18.0` migration rewrites live `dispatch.watchable: true` to `mode: pane`, removes live
`watchable: false`, sweeps project and system config, and leaves commented/fence content untouched.
There is no binary read-time alias; an unmigrated legacy key is inert.

---

## `FAB_KIT_PATH` is deliberately outside the registry (Change 6 — landed)

`FAB_KIT_PATH=<dir>` is a per-process kit-**content** resolution override used for fab-kit development.
It is deliberately not a `config.yaml` field, registry row, scoped key, cascade layer, or persisted
machine preference. Kit resolution happens in the router/fab-kit reader path before the fab-go config
cascade exists, and persisting a repository- or machine-specific source path would make teammates or
later shells silently consume stale/arbitrary kit content.

Both content-reader seams special-case the variable directly: fab-go's `internal/kitpath.KitDir()`
feeds `fab kit-path`, templates, reference files, and other workflow reads; fab-kit's resolver feeds
`fab sync` and `fab migrations-status`. A non-empty value is absolutized, must name an existing
directory, and fails loudly without normal-path/cache fallback when invalid. Under the override, sync
prints `kit: <dir> (FAB_KIT_PATH override)` and skips cache resolution plus the cache-version guard.
`fab doctor` prints the same provenance in normal output without adding a health check. `fab init` and
`fab upgrade-repo` refuse to run until the variable is unset because they stamp release-version state.
Binary selection remains version-pinned and unchanged.

This is intentionally separate from any generic `FAB_*` mapping over registry rows: there is no row to
map, and env-only behavior is the hermeticity boundary rather than an additional config layer.

---

## Advertise semantics — the A/B/C field-category model

`advertise` is the "C flag" of the field-category model the config-upgrade effort uses. Under that
model, at [Change 3]'s `fab config upgrade` time, every field is one of:

- **A) user-overridden** → written as live YAML above the managed fence.
- **B) not overridden** → absent from the file (inherited from defaults).
- **C) not overridden but worth advertising** → scaffolded as a commented reference *inside* the
  managed fence, so the user can discover and opt in.

`advertise: true` marks the C-eligible fields — the optional override surfaces a project has typically
*not* set live: `agent.session`, `agent.workers`, `dispatch.mode`, `dispatch.column_width`,
`dispatch.reap_done`, `checklist.extra_categories`,
`consolidate.detectors`, `true_impact_exclude`, `stage_hooks`, `test_paths`. `advertise: false` marks the init-seeded identity fields
(`project.*`, `source_paths`), which are written live at `fab config init --project` time and not
re-advertised in the fence. (`fab_version` is no longer a config-file field — it left `config.yaml` for
`fab/.fab-version` in [Change 3 — landed], so it is neither advertised nor init-seeded.)

`advertise` had no behavioral consumer in Change 1 (data + `--json` exposure only); as of
[Change 3 — landed] the `fab config upgrade` / `fab config init --project` **fence generator** reads it
to decide which un-overridden fields to scaffold into the managed fence.

### Demotion — advertised surface ≠ documented surface (260806-j9nh)

`advertise: false` means "do not scaffold this into every project's fence"; it does **not** mean
"undocumented". The agent machinery — **`agent.profiles`** and the whole **`providers`** table — is
demoted on exactly that basis: the advertised agent surface is the two depth knobs (`agent.session`,
`agent.workers`), and scaffolding the six role profiles plus the built-in provider grammars cost ~90 commented
lines of every project's `config.yaml` for a surface almost nobody overrides (naming a built-in provider
on a knob needs no `providers:` entry at all). Both rows keep their registry entries, their `--json`
defaults, and their rendered segments, so they remain in `fab config explain` — the surface whose job
*is* completeness — and in `fab config init --system` as the diet short-segment form.

Two mechanics follow from the demotion:

- **One segment per YAML block.** The `agent.session` row owns the whole `agent:` segment — the two
  knobs live, plus a commented `profiles:` pointer/example — and `agent.workers` / `agent.profiles`
  carry no segment of their own. Two segments emitting a live `agent:` parent would collide into a
  duplicate YAML key and break the reference's round-trip, the same reason `project.name` owns the
  `project:` block and `dispatch.mode` owns `dispatch:` (§ Several rows under one YAML block).
- **`renamed_from` on `agent.profiles` is metadata, not a working carry.** It records the
  `agent.tiers` → `agent.profiles` rename for `--json` consumers, but `fab config upgrade`'s rename
  carry is a **top-level-key** operation and deliberately skips a same-top-level rename (§ renamed_from).
  The on-disk rewrite is the `2.16.19-to-2.17.0` migration's job; the binary meanwhile keeps reading
  `agent.tiers` per role (and the flat `providers.<name>.model`/`.effort` as an alias for
  `providers.<name>.profiles.default`), so no config goes inert between the two.

---

## renamed_from — mechanical rename carry-forward

`renamed_from` names a field's previous key path so [Change 3]'s `fab config upgrade` can carry a
user's value forward across a rename mechanically, instead of each rename needing a hand-written
migration. Historical renames predating the field (e.g. `agent.spawn_command` →
`providers.claude.interactive_command`, change tykw) were already handled by shipped migrations and are
**not** backfilled. The `--json` dump omits the field when empty.

**One row carries it today**: `agent.profiles`, recording the 260806-j9nh rename from `agent.tiers`.
That row is also the field's documented **limit**. The carry is a **TOP-LEVEL-key** operation — the
upgrader rewrites a column-0 `key:` token and preserves the whole value block verbatim beneath it — so
it can only carry a rename whose old and new keys are *different* top-level keys. `agent.tiers` →
`agent.profiles` collapses to `agent` → `agent`, which `registryTopLevelKeys` deliberately skips (it
would otherwise log a spurious "carried rename" on every run). So on that row `renamed_from` is
**metadata for `--json` consumers**, not a working carry: the on-disk rewrite is the
`2.16.19-to-2.17.0` migration's job, and the binary meanwhile reads `agent.tiers` per role so nothing
goes inert in between. A future nested-aware carry would make the row's metadata operative without a
registry change.

---

## `--json` output shape

`fab config explain --json` emits the field table as a flat JSON array in table (rendering) order,
using stdlib `encoding/json` only (no new dependencies). Each element is a per-field object:

```json
[
  {
    "key": "project.name",
    "default": null,
    "description": "Project display name. Read by skills for orientation and PR bodies.",
    "scope": "project",
    "advertise": false
  },
  {
    "key": "agent.profiles",
    "default": {
      "default":  { "provider": "claude", "model": "...", "effort": "..." },
      "operator": { "provider": "claude", "model": "...", "effort": "..." },
      "doing":    { "provider": "claude", "model": "...", "effort": "..." },
      "review":   { "provider": "claude", "model": "...", "effort": "..." },
      "hydrate":  { "provider": "claude", "model": "...", "effort": "..." },
      "fast":     { "provider": "claude", "model": "...", "effort": "..." }
    },
    "description": "Per-stage model override. ...",
    "scope": "both",
    "advertise": true
  }
]
```

- The `agent.profiles` `default` is a map **keyed by role name** (one entry per `agent.RoleNames()` role —
  `default`, `operator`, `doing`, `review`, `hydrate`, `fast`), each a `{provider, model, effort}`
  profile; the first-level `default` key is the *default role*, not a wrapper. Likewise
  `providers.default` is keyed by provider name (four entries — claude/codex/agy/kimi), each carrying
  its independent `interactive_command`, `headless_command`, and `native` capabilities (omitted when
  unavailable, so each block shows exactly what it ships): claude all three; codex, `agy`, and `kimi`
  both commands without native. agy's interactive grammar is
  `agy --dangerously-skip-permissions --model {model}`; its exact-path trust wall is handled as an
  ordinary readiness-gate judgment round, with live verification retained in backlog `[agik]` until
  quota resets. Every entry carries **no** `model`/`effort` — see § Default semantics.
- `renamed_from` is omitted when empty (`omitempty`), so it appears on the `agent.profiles` object only.
- Output is deterministic and byte-stable, like the commented-YAML rendering — the table is ordered and
  the marshalling is stable.
- Without the flag, bare `fab config explain` prints the commented YAML exactly as before. An
  optional dotted key selects its owning rendered segment; keyed `--json` returns the matching
  owning row(s) in the same array shape. `reference` remains an invisible Cobra alias for
  historical pointers.
- The JSON key set is guarded against drift from the YAML reference's documented key set, so the
  machine-readable and human-readable views cannot silently diverge.

This is the tooling surface [Change 2] and [Change 3] (and external tools) consume: `scope` →
cascade enforcement + `init --system`; `advertise` → the fence scaffold; `renamed_from` → upgrade
carry-forward; `default` → `fab config show --origin`.

---

## Cascade & six-verb command surface (Change 2 + environment layer + 260808-ihiv + 260808-fp02 — landed)

The file cascade, scope enforcement, and two visibility commands landed in Change 2 (260708-lpb5);
the generic environment override layer extends that same loader seam; the read-model redesign
(260808-fp02) fixed the merge rule, the tier order, and the visibility surfaces below. Recorded here in
authoritative detail alongside the Change 1 schema.

### Override cascade

Effective config resolves across four tiers, highest precedence first, at the single loader seam
`internal/config.LoadPath` (so every consumer — preflight, impact, status, resolve-agent, dispatch,
agent, operator, batch, spawn, prmeta — sees effective config with zero per-caller change):

1. **environment** — YAML-valued variables derived from registry keys
2. **system** — `~/.fab-kit/config.yaml` (co-located with the version cache; XDG path rejected — decision 5)
3. **project** — `fab/project/config.yaml`
4. **built-in defaults** — the values in `internal/agent`'s embedded `defaults.yaml` (read through
   the Go symbols this spec's table references), applied at the
   existing point-of-use seams (`internal/agent`'s role/provider resolution, the nil-safe accessors)
   and projected as a materialized read-model tier by `configref.DefaultsMap` (below)

**The system tier outranks the project file.** For a *preference-class* key — "which worker provider do
I like on this machine" — a repo's committed suggestion losing to the user's own machine-wide choice is
the point: the alternative made a personal `~/.fab-kit/config.yaml` preference silently inert in any
repo that happened to pin the key. What makes the order safe is that **scope enforcement is retained**:
only `scope: system`/`both` fields are honored in the system file at all, so a semantics-class key
(`source_paths`, `test_paths`, `stage_hooks`, …) can never appear there and the repo stays reproducible
for teammates and CI. The flip changes no on-disk shape, so it ships **without a migration** — it is
carried by docs and release notes (the constitution's migration rule governs user-data *restructuring*).

All materialized tiers merge at the YAML map level, before unmarshal, **per leaf**: maps merge per-key
(the existing `agent.profiles` precedent), **lists replace** (never concatenate), scalars replace, and
each leaf takes the value of the highest tier that defines it **non-empty**.

**Empty-skip.** A leaf whose value at some tier is `null`, `""`, `[]`, or `{}` neither wins nor blocks —
it falls through to the tier below. This removes explicit-null/presence semantics from the READ side
entirely: an explicit `key: null` shadowing a lower tier was a footgun, not a feature, and modelling it
cost a presence bit threaded through every provenance path. `false` and `0` are **real values and are
never skipped** — a bool/int override must survive (a project `dispatch: {reap_done: false}` resolves
`false`). A mapping whose every leaf is empty defines nothing and falls through wholesale. The rule has
one implementation, `config.MergeLayers` over `config.IsEmptyValue`, which the loader and every
provenance surface share, so visibility cannot disagree with resolution.

Environment names are derived mechanically as `FAB_` plus the uppercase
dotted registry key with dots replaced by underscores; values are YAML-parsed, preserving scalar,
list, and map types. The loader walks the ordered registry-key enumeration rather than the process
environment, accepts only `scope: system`/`both`, and treats an empty value as unset — both a blank
variable and one whose YAML parses empty, so the environment obeys the same empty-skip rule as every
other tier and contributes neither an overlay leaf nor a provenance entry. A malformed value
or a project-scoped environment override emits a `fab: warning:` and is ignored; an unknown/unrelated
`FAB_*` variable is never examined. The supported user-facing examples are `FAB_AGENT_WORKERS` and
`FAB_AGENT_SESSION`; `FAB_AGENTS` (plural) is unrelated and has no config meaning.

### The defaults tier is materialized for the READ MODEL, not for the loader

`configref.DefaultsMap()` projects every registry row's canonical `default` into one YAML-shaped map —
the real bottom tier the visibility and mutation surfaces merge beneath the files and the environment.
`DefaultsMapFor(cfg)` is the same projection with the *derived* `agent.profiles` rows resolved against
the **live** config (below).

The loader deliberately does **not** merge that map. Two reasons, the second decisive:

- `internal/config` cannot import `internal/configref` — `configref → agent → config` would close an
  import cycle, which is why `internal/configscope` exists as a leaf in the first place.
- The `agent.profiles` default is **derived** (resolved from the depth knobs and the provider fills),
  not stored. Merging it into the loader's tree would land a full `{provider, model, effort}` in
  `Config.Agent.Profiles`, which the resolver reads as a **user override** outranking the provider's own
  fills — inverting the documented fill precedence and breaking `--provider` swaps. A registry-defaults
  tier is sound for display and provenance; it is not sound for the resolver.

The retained point-of-use seams therefore remain the loader's fourth tier (their wholesale collapse is
still deferred), and `internal/config.DispatchConfig.ReapDone` stays a `*bool` for the same reason: the
loader's merged tree carries no defaults, so an unset key still reaches Unmarshal as absent and the
pointer is what keeps absent distinguishable from an explicit `false`.

### Presence semantics are gone from the read side

`fab config show --origin` no longer tracks layer presence separately from value. "This tier defines
the leaf" is `!config.IsEmptyValue(value)` — the same test the merge applies — so the winner rule, the
shadow warnings, and the merge are one definition rather than three mechanisms.

The cascade is **fail-open** (config must never brick): an absent system file
is byte-identical to the pre-cascade single-file behavior; a malformed or unreadable system file emits
a `fab: warning:` on stderr and is skipped; a malformed **project** file keeps today's error behavior.
**Scope enforcement**: a project-scoped field appearing in the system file is pruned with a
`fab: warning:` (only `scope: system`/`both` fields are honored there); unknown keys are ignored
silently. The scope taxonomy is single-sourced in the leaf package `internal/configscope`, which both
the loader and the registry `internal/configref` consume without an import cycle. `configscope` also
owns the ordered dotted-key enumeration, with a parity test against `configref` so the generic
environment walk cannot drift from the reference schema.

### Six intent-grouped verbs

- `fab config show [<key>] [--origin]` — a pure query. Plain bare output prints the fully composed
  environment-over-system-over-project-over-built-in-defaults config as YAML. `--origin` adds per-field provenance (exact
  `$FAB_…` variable / system path / project path / `default`, following the
  `git config --show-origin` precedent) with per-key drill-down for map-valued fields. Bare
  `--origin` is **winner-only**: one line per leaf. It surfaces
  typo'd overrides that silently no-op (the intended field shows origin `default`). With a
  known dotted key, scalar/list output is the raw effective value and map output is its YAML subtree;
  unknown keys fail non-zero naming the key.

  **Keyed `--origin` lists the key's FULL STACK** — one line per tier that defines it, highest first,
  the winner marked `(effective)` and the rest `(shadowed)`. No new flag: the winner-only view is the
  all-keys listing's job, and the keyed view is where "why is my override not taking effect" gets
  answered. A map-valued key drills down per leaf, each leaf listing its own defining tiers; a
  descendant under an ancestor a higher tier replaced reports that replacing tier. Rendering:

  ```
  $ fab config show agent.workers --origin
  agent.workers = codex    # env $FAB_AGENT_WORKERS  (effective)
  agent.workers = kimi3    # system /home/u/.fab-kit/config.yaml  (shadowed)
  agent.workers = claude   # default  (shadowed)
  ```

  The keyed listing names the tier (`env`/`system`/`project`/`default`) alongside its label, because a
  stack of two bare file paths is not readable; the winner-only listing keeps the bare
  `git config --show-origin` label vocabulary.

  **The composed `agent.profiles.*` drill-down rows are knob-aware.** They are derived from the depth
  knobs and the provider fills rather than stored, so composing them against the registry's nil-config
  `Default` made every role's provider row read `claude # default` even when a knob named another
  provider. They now compose against the LIVE config (`configref.DefaultsMapFor`), so the reported
  provider (and its fills) is what the role would actually dispatch to. Only the user's per-role
  **model/effort** overrides are stripped before resolving (a defaults tier never echoes a tier above
  it); a per-role **provider** override is KEPT for the model/effort derivation — the built-in fill is
  a function of the provider the role actually dispatches to, so stripping it composed a chimera row
  (the override's provider beside the knob provider's fills) that disagreed with `fab resolve-agent`.
  The provider leaf's own default stays knob-resolved, so keyed `--origin` shows the override
  shadowing the knob's provider while the model/effort leaves report the overridden provider's fill as
  `default (effective)`. Resolution stays provider-neutral: a knob or override naming a provider fab
  ships nothing for is reported verbatim, with empty fills falling through under empty-skip.
- `fab config explain [<key>] [--json]` — the registry documentation query. Bare forms render the
  full commented YAML or JSON table; keyed forms render the owning segment or its row(s).
  `reference` is an invisible compatibility alias.
- `fab config set <key> <value> [--system]` and `unset <key> [--system]` — surgical writers through
  `internal/configupgrade`, never whole-document YAML marshal. Keys are registry-validated, including
  documented deep scalar leaves. `set` accepts only a single-line, comment-free YAML `string`, `bool`,
  `int`, or `float`; structural map keys, collection-valued leaves, collection/null values, multiline
  input, and YAML comments are refused with manual-edit guidance. An **empty value
  is refused** too, pointing at `fab config unset`: under empty-skip an empty leaf falls through and can
  never be effective, so writing one is a pure footgun rather than a way to clear a key. Emptiness is
  tested on the **parsed** value (`config.IsEmptyValue`), so the quoted-empty spellings (`''`, `""`) and
  an explicit `null` are refused alongside a blank or whitespace-only argument. Fence-only deep
  keys materialize every ancestor from the registry renderer before insertion.

  **`set` warns when a higher tier shadows the write.** The write is valid, so it succeeds and exits 0,
  but a `fab: warning:` on stderr names the tier that actually wins
  (`agent.workers is shadowed by env $FAB_AGENT_WORKERS — the written value is not in effect`). It fires
  for either target — project shadowed by system or environment, `--system` shadowed by environment —
  and NOT for `--system` over a project value, which the system tier outranks. The check is fail-open:
  an unresolvable repo or unreadable layer prints nothing rather than failing a completed write.

  `unset` is deliberately kind-ungated so
  it can repair malformed overrides. System writes accept only `scope: system`/`both`, never add a
  project fence, and create a missing file with the canonical scaffold header. Unsetting an absent
  known key is an exit-zero notice that now **names the tier where the key IS live** plus the command
  that would remove it (`live in system ~/.fab-kit/config.yaml — use: fab config unset agent.workers
  --system`); an environment tier is named as one `unset` cannot remove, and a key supplied only by the
  built-in default keeps the bare notice. The shared YAML value parser retains collections in either flow
  or block style for the environment overlay only; ENV list/map support does not widen the mutation
  contract.
- `fab config init [--system] [--print] [--force]` — bare init generates the project file; `--project` remains a
  compatible explicit spelling. `--system` writes a `~/.fab-kit/config.yaml` scaffold containing ONLY
  `scope: system`/`both` fields, all commented (the SHORT per-field adverts, not the essays) — generated from this same table so it can't drift.
  Both modes refuse to overwrite an existing file unless `--force` is given (`--force` explicitly
  overwrites the target; refusal stays the default), and the two mode flags are
  mutually exclusive. `--print` renders the exact would-be file to stdout with **zero writes**: it
  composes with both modes, is never blocked by an existing file, and `--print --force` is a pure
  preview of what an overwrite would write.
- `fab config upgrade` remains the whole-project-file reconciliation verb described below, now with a
  `--check` drift probe (§ The managed fence).

There is no reserved `validate` verb. Unknown-key refusal and system-scope enforcement happen at
the mutation seam where they are actionable.

## `fab config upgrade` + migration (Change 3 — landed)

The mechanical upgrader, the `fab_version` relocation, the scaffold retirement, and the migration
landed in Change 3 (260708-j0qm). Recorded here in authoritative detail alongside Changes 1/2.

### Presence = intent [Change 3 — landed] (decision 2)

Any live field in a config file is an **override**, even if its value equals the default. `fab config
upgrade` never auto-removes a live field; B-hygiene ("these fields equal current defaults — remove?")
is advisory only. A value-diff classifier cannot distinguish "deliberately pinned" from "never touched",
and auto-dropping would silently change behavior when the default later moves.

### The managed fence [Change 3 — landed] (decision 3)

`fab config upgrade` (the whole-file reconciler in the shared comment-aware writing engine) regenerates a
byte-stable, idempotent **managed fence** of commented C-fields (`advertise: true`, not currently
overridden), rendered from each field's SHORT segment (`ShortSegment` — the 1–4-line
scope-tagged description header plus the machine-wide and `fab config explain` pointers, §
Section-level prose), never the long essay, delimited by byte-exact `>>>`/`<<<` splice anchors carrying a kit-version stamp
(`# >>> fab reference (kit X.Y.Z) >>> …` / `# <<< end fab reference <<< …`, dash-padded). Upgrade
rewrites ONLY between the markers; everything outside — including the user's own comments on A-fields —
is the user's. Every scaffolded block is **fully commented including its parent keys** (a live `agent:`
over comment-only children is exactly the `agent: null` the old whole-file masher produced), with every
comment marker at **column 0**: the comment-out helper skips only a line whose `#` is ALREADY at column
0 (fence-level prose), so a line the segment ships deliberately commented at an INDENT (the
`  # profiles:` / `  #   review: { provider: codex }` agent-block examples) gains the fence prefix like a live line —
which both keeps the fence visually flush and makes "strip the leading `# ` from every line of a block"
restore the segment byte-exactly. The fence
**omits fields already overridden** above it. Omission is at **top-level-key granularity**: a live
top-level key (e.g. `agent:`) suppresses the entire scaffolded block for every registry row under that
key, since the override unit and the system-file merge both land at the top-level key — the fence never
half-advertises a partially-overridden block. A legacy file with no fence gets one **appended at the
bottom**. Everything OUTSIDE the fence is the user's and is **never dropped**: content the user places
BELOW the fence (a live override appended after the END anchor) is **hoisted above** the fence on the
next run and then classified like any other live key (kept if known, parked if unknown) — the layout is
self-healing, not a silent-loss trap. Unknown fields (a live key no longer in the registry) are
**parked** in a `# removed in … (parked by fab config upgrade — delete when done):` block below the
fence, the value serialized in the comment — appended **exactly once**, never regenerated away. Before
writing, the reconciled document is **validated as YAML** and a run that would produce an unparseable
file is **refused** (the original left untouched) rather than bricking the repo. A live field matching a
registry row's `renamed_from` is **carried** to the new key mechanically (value verbatim), replacing the
per-rename hand-written-migration pattern. Output is byte-stable and idempotent (the `fab memory-index`
discipline — golden + idempotence tests); the write is atomic (`internal/atomicfile`). After this change,
`internal/configupgrade` is the **only writing engine** for existing `config.yaml` files: `upgrade`
owns whole-file reconciliation while `set`/`unset` use its surgical path splice and the same fence
renderer. This retires the comment-clobbering
`setFabVersion` bug class at the root. `fab upgrade-repo` **auto-runs** the upgrader after sync
(decision 4, fail-open: a fab-go predating the subcommand prints a reminder and the upgrade continues).
The kit-version stamp in the BEGIN line makes staleness visible and feeds the **`--check` drift
probe**: `fab config upgrade --check` shares Upgrade's entire compute path (`configupgrade.Check`
calls the same `computeUpgrade`) but writes NOTHING — it prints what a run would change and exits
non-zero when the file has drifted (a stale fence kit-version stamp, unparked removed keys, a missing
fence, or a missing file, which a real run would create), 0 when the file is clean. The shared compute
path means the probe can never disagree with an applying run about what would change.

### `fab_version` → `fab/.fab-version` [Change 3 — landed] (decision 1)

`fab_version` moves out of `config.yaml` entirely into a new plain-text sibling `fab/.fab-version`
(one line, bare semver + newline, committed, sibling to `fab/.kit-migration-version`). This is what lets
`internal/configupgrade` be config.yaml's sole writing engine — the one machine-managed field the old masher owned
leaves the file. **Gitignore caveat** [2.15.2 — 8ken]: "committed" is only true if `.gitignore` does not
swallow the file. The `.fab-*` line (added for the root runtime files `.fab-status.yaml`/`.fab-backend`/
`.fab-runtime.yaml`) is **unanchored** — with no slash it matches at any depth, so it also ignores
`fab/.fab-version`. A negation line `!fab/.fab-version` (immediately after `.fab-*`) is what un-ignores
the file and makes the "committed" guarantee real. The shipped scaffold fragment
(`src/kit/scaffold/fragment-.gitignore`) carries the negation, so every `fab sync` self-heals a project's
`.gitignore`; the `2.15.1-to-2.15.2` migration verifies the negation and commits the file for
already-shipped repos, and `stampFabVersion`'s callers emit a fail-open `fab: warning:` when the file is
still ignored so the written-but-ignored version file can never go quiet again. Both reader stacks (the fab-kit router's pinned-version resolution and the fab-go
preflight staleness check) read `fab/.fab-version` as the **sole** version source — the one-compat-window
config.yaml `fab_version:` fallback they briefly kept has been closed [260719-kq7v]. The `fab_version` row is
removed from the registry and from the `internal/configscope` taxonomy, and `Config.FabVersion` is tagged
`yaml:"-"`, so a stale `fab_version:` in either config file is an inert unknown key — nothing unmarshals it
and it can never contribute to a repo's resolved version. The `2.14.0-to-2.15.0` migration (a user-data
restructure per the constitution) moves the value and deletes the key for pre-2.15 repos; it is
sentinel-guarded and idempotent. Historical
comment-backfill migrations (e.g. `2.13.1-to-2.13.2`) are left untouched; that pattern is **retired going
forward** — field adds/renames/removals are mechanical registry data reconciled by `fab config upgrade`.

### Scaffold config.yaml deleted — init generates from the registry [Change 3 — landed]

The hand-maintained scaffold `src/kit/scaffold/fab/project/config.yaml` (the last drift-prone copy of the
defaults/comment prose) is **deleted**. `fab config init --project` generates the initial `config.yaml`
from the registry: the **A-class identity fields** (`InitSeed` rows — `project.name`,
`project.description`, `source_paths`, `test_paths`) written live above the managed fence, then the fence
of commented C fields. No `agent:` key is pinned at init (presence=intent — an init-pinned knob or role profile
would be an accidental override). `fab init` (the fab-kit binary) shells out to the pinned fab-go's
`fab config init --project`; when that fab-go cannot generate the project config (e.g. it predates the
subcommand), `fab init` **fails with a clear error** naming the remedy — upgrade fab-go (e.g. `brew
upgrade fab-kit`) and re-run `fab init`. The fab-kit binary's former embedded stub `config.yaml`
fallback is **retired**: the binary ships zero stub copies of the config, keeping the embed census at
`defaults.yaml` (values) + `internal/configref` (schema/prose) + `internal/configscope` (scope
taxonomy) + `src/kit/scaffold/` (non-config files). The registry carries the init/seed metadata (`InitSeed`) marking which fields are
written live at init; fab-kit's mechanical detection — the repo folder name, an existing `src/`, and the
ecosystem marker-table `test_paths` — becomes generator input, passed as `--name`/`--source-path`/
`--test-path` flags so those A-class fields land live (`project.description` is not mechanically
detectable and is added interactively by `/fab-setup`). An empty-seed `fab config init --project`
(no flags) emits the header + fence only.
The former `TestConfigReferenceSupersetsScaffoldKeys` guard is re-anchored to a registry-internal
invariant (the init-seeded key set ⊆ the registry key set), since there is no scaffold file to compare
against.
