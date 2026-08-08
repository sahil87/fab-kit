# Config Schema — the per-field metadata table

> **Status:** Design intent (pre-implementation, Constitution VI). This spec is human-curated. It
> records the config-system schema decisions resolved in the 2026-07-08 `/fab-discuss` session and
> written up in the config-upgrade effort's backlog doc (`fab/plans/sahil/config-upgrade.md`, all six
> decisions user-confirmed). It is written across the **three-change** config-upgrade effort:
> **Change 1** (260708-ff2v) — the per-field metadata table + `fab config reference` restructure +
> `--json` — **Change 2** (260708-lpb5) — the three-layer cascade resolution + scope enforcement +
> the `fab config show [--origin]` / `fab config init --system` visibility commands — and **Change 3**
> (260708-j0qm) — `fab config upgrade` + the managed fence + the `fab_version` → `fab/.fab-version`
> relocation + the migration + registry-driven `fab config init --project` (scaffold config.yaml
> deleted) — are all landed here in authoritative detail. The whole config-upgrade design authority
> lives in one place.
>
> The canonical schema is the Go field table in `src/go/fab/internal/configref/`; this doc is its
> human-readable rationale. Defaults that have a Go symbol are sourced from that symbol, never
> restated here or in the table.

`fab/project/config.yaml` is the single project-config file the `fab` binary and the markdown skills
read. This spec fixes how its schema is modeled: not as prose, but as an ordered **per-field metadata
table** from which every rendering (the commented-YAML `fab config reference`, the `--json` dump, and
— in later changes — the cascade resolver and the `fab config upgrade` fence) is generated. One
source, no second copy to drift.

---

## Why a metadata table (invert the data/prose relationship)

`fab config reference` originally (change 6nke) rendered the schema from a text template with a few
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
| `description` | One-line summary of the field. Required (non-empty) — the registry lint rejects an empty description. Feeds the JSON dump and, later, the generated comment scaffold [Change 3]. |
| `scope` | Override visibility across the cascade layers: `project` / `system` / `both`. See § Scope taxonomy. |
| `advertise` | The "C flag": whether [Change 3]'s managed fence scaffolds this field as a commented reference when it is not overridden. See § Advertise semantics. |
| `renamed_from` | Previous key path for mechanical rename carry-forward. Set on `agent.profiles` (`agent.tiers`) as of 260806-j9nh; `""` on every other row. See § renamed_from. |
| `init-seed` | Whether the field is an A-class **identity** field written LIVE at `fab config init --project` time ([Change 3]) — `project.name`/`project.description`/`source_paths`/`test_paths`. The generator's live block above the fence; every other field is fence territory from day one. Consumed by the init generator, NOT exposed in the `--json` schema dump (like the rendered YAML segment). |

### Defaults are sourced from canonical Go symbols — no second copy

Every default that has a canonical Go symbol is referenced from it, not copied: the claude session
command from `agent.DefaultSessionCommand`, the per-role profiles via `agent.DefaultProfile` over
`agent.RoleNames()`, the stage names via `agent.StageNames()`. The registry construction fails loud
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
always denotes a real built-in value (today: the `providers` row's **three built-in providers** —
claude/codex/gemini, `260805-j3cm`, each with its per-role fills — the resolved `agent.profiles` defaults, the two depth knobs' `claude`, `dispatch.watchable`'s
`false`, `dispatch.column_width`'s `35`, and `dispatch.reap_done`'s `true`); every other row is `null`.
**The three `dispatch` rows are
the convention's boundary cases and are deliberately not `null`**: for a **bool** there is no "absent"
state distinguishable from `false`, and for the width an absent YAML int is indistinguishable from `0`
(which the accessor therefore reads as unset, alongside every other out-of-`1..99` value), so each
carries a real built-in value the cascade genuinely bottoms out at — not the typed-empty placeholder
the convention forbids (the forbidden shapes are the ones that could stand in for "nothing").
`dispatch.reap_done` is the sharpest of the three and the one that forced a **struct-level** answer as
well as a registry one: its built-in default is **`true`**, so the Go zero value means the *opposite* of
the default, and a plain `bool` would have made an absent key indistinguishable from an explicit
`reap_done: false` — silently disabling reaping for every project that never sets the key. It is
therefore modeled as a **`*bool`** in `internal/config.DispatchConfig` (`nil` = unset = `true`), the one
place the three siblings' shapes diverge; the registry row still carries the plain `true`, because the
*default* is a value, not a pointer.
The same rule governs the **per-role fills** inside the `providers` default: every built-in's
`profiles` map is projected, because all three ship real fills (`260806-ywkx`). The convention applies
one level down instead — claude's map is exhaustive (all six roles), while codex's and gemini's are
**sparse**, and a role fab-kit ships no fill for is **omitted entirely** rather than emitted as an
empty object, since that would assert a built-in fill that deliberately does not exist (the omitted
roles resolve the provider's `default` entry). The **deprecated flat**
`providers.<name>.model`/`.effort` is likewise absent from every `default`: it exists only as a
read-time alias for `profiles.default` until the `2.16.19-to-2.17.0` migration rewrites a config.

### Section-level prose lives on the row — the segment

One-line `description`s cannot carry the narrative documentation blocks the reference needs (the
providers explanation, the per-provider dispatch/fill notes, the three built-in providers, the fixed
stage→role mapping). Each table row therefore carries — alongside its one-line `description` — the
**rendered YAML segment**: the field's commented block as it appears in the reference. `fab config
reference` is generated by walking the table and concatenating those segments in order; there is no
separate template. The `description` (the machine-readable one-liner, exposed in `--json`) and the
`segment` (the human-readable block, exposed in the YAML) are two projections of **one** row, not a
second copy of the schema to drift — a field's documentation is authored once, on its row. The rows for
map-valued fields (`providers`, the `agent:` block, `stage_hooks`) build their segment by interpolating the
same Go symbols their `default` reads, so the rendered prose carries no literal copy of any value.
The existing reference tests assert those blocks verbatim; the restructure preserves them byte-for-byte.

**Several rows under one YAML block share a single segment.** Where two or more override units live
under the same top-level key, the segment belongs to the *first* of them and documents them all; the
rest carry an **empty** segment (`project.name` owns the `project:` block for `project.description` and
`project.linear_workspace`; `dispatch.watchable` owns the `dispatch:` block for
`dispatch.column_width` **and** `dispatch.reap_done`). This is not an optimisation but a correctness requirement: the reference and
the managed fence render these blocks **commented**, with the documented instruction to uncomment a
whole block, so two separately-uncommentable `# dispatch:` parents would collide into a duplicate YAML
key. It also matches the fence generator, whose override detection is top-level-key scoped: a live
`dispatch:` block suppresses the advertisement of every key under it.

---

## Scope taxonomy (decision 6)

`scope` states which cascade layer(s) may override a field. The rationale: the **system** layer
(`~/.fab-kit/config.yaml`, [Change 2]) is restricted to *preference-class* fields — personal model/harness
choices — while *semantics-class* fields stay in the project file so the repo remains reproducible for
teammates and CI.

| scope | Meaning | Fields |
|-------|---------|--------|
| `both` | Overridable in either the project or the system layer (preference-class). | `agent.session`, `agent.workers`, `agent.profiles`, `providers`, `dispatch.watchable`, `dispatch.column_width`, `dispatch.reap_done` |
| `project` | Overridable only in the project file (semantics-class, repo-reproducible). | `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `consolidate.detectors`, and (conservative default) `stage_hooks`, `branch_prefix` |
| `system` | Overridable only in the system layer. | *(none today; the value exists for completeness and [Change 2])* |

Fields the decision-6 taxonomy does not enumerate (`stage_hooks`, `branch_prefix`) default to `project`
— the conservative choice, since system-visibility is opt-in per the same rationale. `dispatch`
(`dispatch.watchable`, the watchable-pane opt-in; `dispatch.column_width`, the pane-worker column's
width; `dispatch.reap_done`, whether a finished worker's pane is reclaimed) is `both` by the same
reasoning that puts `agent`/`providers` there: all three keys express how the
**operator** prefers to watch stage workers on **this machine** — whether in a pane at all, how
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
*not* set live: `agent.session`, `agent.workers`, `dispatch.watchable`, `dispatch.column_width`,
`dispatch.reap_done`, `checklist.extra_categories`,
`consolidate.detectors`, `true_impact_exclude`, `stage_hooks`, `branch_prefix`, `test_paths`. `advertise: false` marks the init-seeded identity fields
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
`agent.workers`), and scaffolding the six role profiles plus three provider grammars cost ~90 commented
lines of every project's `config.yaml` for a surface almost nobody overrides (naming a built-in provider
on a knob needs no `providers:` entry at all). Both rows keep their registry entries, their `--json`
defaults, and their rendered segments, so they remain in `fab config reference` and in
`fab config init --system` — the two surfaces whose job *is* completeness.

Two mechanics follow from the demotion:

- **One segment per YAML block.** The `agent.session` row owns the whole `agent:` segment — the two
  knobs live, plus a commented `profiles:` pointer/example — and `agent.workers` / `agent.profiles`
  carry no segment of their own. Two segments emitting a live `agent:` parent would collide into a
  duplicate YAML key and break the reference's round-trip, the same reason `project.name` owns the
  `project:` block and `dispatch.watchable` owns `dispatch:` (§ Several rows under one YAML block).
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
`providers.claude.session_command`, change tykw) were already handled by shipped migrations and are
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

`fab config reference --json` emits the field table as a flat JSON array in table (rendering) order,
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
  `providers.default` is keyed by provider name (three entries — claude/codex/gemini), each carrying
  only the command fields that exist for that built-in (`session_command` and, for codex/gemini,
  `dispatch_command`; both `omitempty`) and **no** `model`/`effort` — see § Default semantics.
- `renamed_from` is omitted when empty (`omitempty`), so it appears on the `agent.profiles` object only.
- Output is deterministic and byte-stable, like the commented-YAML rendering — the table is ordered and
  the marshalling is stable.
- Without the flag, `fab config reference` prints the commented YAML exactly as before; the command
  stays a pure query (no file writes, exit 0 on success, extra positional args rejected by `cobra.NoArgs`).
- The JSON key set is guarded against drift from the YAML reference's documented key set, so the
  machine-readable and human-readable views cannot silently diverge.

This is the tooling surface [Change 2] and [Change 3] (and external tools) consume: `scope` →
cascade enforcement + `init --system`; `advertise` → the fence scaffold; `renamed_from` → upgrade
carry-forward; `default` → `fab config show --origin`.

---

## Cascade & visibility commands (Change 2 — landed)

The three-layer cascade, scope enforcement, and the two visibility commands landed in Change 2
(260708-lpb5). Recorded here in authoritative detail alongside the Change 1 schema.

### Override cascade [Change 2 — landed]

Effective config resolves across three layers, highest precedence first, at the single loader seam
`internal/config.LoadPath` (so every consumer — preflight, impact, status, resolve-agent, dispatch,
agent, operator, batch, spawn, prmeta — sees effective config with zero per-caller change):

1. **project** — `fab/project/config.yaml`
2. **system** — `~/.fab-kit/config.yaml` (co-located with the version cache; XDG path rejected — decision 5)
3. **built-in defaults** — the Go tables in the `fab` binary (this spec's table), applied at the
   existing point-of-use seams (`internal/agent`'s role/provider resolution, the nil-safe accessors)

The two **files** merge at the YAML map level, before unmarshal, by **per-field deep merge**: maps
merge per-key (the existing `agent.profiles` precedent), **lists replace** (never concatenate), scalars
replace — project wins. The cascade is **fail-open** (config must never brick): an absent system file
is byte-identical to the pre-cascade single-file behavior; a malformed or unreadable system file emits
a `fab: warning:` on stderr and is skipped; a malformed **project** file keeps today's error behavior.
**Scope enforcement**: a project-scoped field appearing in the system file is pruned with a
`fab: warning:` (only `scope: system`/`both` fields are honored there); unknown keys are ignored
silently. The scope taxonomy is single-sourced in the leaf package `internal/configscope`, which both
the loader and the registry `internal/configref` consume without an import cycle.

### Visibility commands [Change 2 — landed]

- `fab config show [--origin]` — a pure query. Plain output prints the merge of the two FILES
  (project over system) as YAML; built-in defaults are NOT materialized here (they apply at
  point-of-use), surfaced explicitly only by `--origin`, which adds per-field provenance (project
  path / system path / `default`, the `git config --show-origin` precedent) with per-key drill-down
  for map-valued fields. It surfaces typo'd overrides that silently no-op today (the intended field
  shows origin `default`).
- `fab config init --system` — writes a `~/.fab-kit/config.yaml` scaffold containing ONLY
  `scope: system`/`both` fields, all commented — generated from this same table so it can't drift.
  Refuses to overwrite an existing file (no `--force`); bare `fab config init` is a usage error.

## `fab config upgrade` + migration (Change 3 — landed)

The mechanical upgrader, the `fab_version` relocation, the scaffold retirement, and the migration
landed in Change 3 (260708-j0qm). Recorded here in authoritative detail alongside Changes 1/2.

### Presence = intent [Change 3 — landed] (decision 2)

Any live field in a config file is an **override**, even if its value equals the default. `fab config
upgrade` never auto-removes a live field; B-hygiene ("these fields equal current defaults — remove?")
is advisory only. A value-diff classifier cannot distinguish "deliberately pinned" from "never touched",
and auto-dropping would silently change behavior when the default later moves.

### The managed fence [Change 3 — landed] (decision 3)

`fab config upgrade` (the single, comment-aware writer of `config.yaml` going forward) regenerates a
byte-stable, idempotent **managed fence** of commented C-fields (`advertise: true`, not currently
overridden), delimited by byte-exact `>>>`/`<<<` splice anchors carrying a kit-version stamp
(`# >>> fab reference (kit X.Y.Z) >>> …` / `# <<< end fab reference <<< …`, dash-padded). Upgrade
rewrites ONLY between the markers; everything outside — including the user's own comments on A-fields —
is the user's. Every scaffolded block is **fully commented including its parent keys** (a live `agent:`
over comment-only children is exactly the `agent: null` the old whole-file masher produced), with every
comment marker at **column 0**: the comment-out helper skips only a line whose `#` is ALREADY at column
0 (fence-level prose), so a line the segment ships deliberately commented at an INDENT (claude's
`# dispatch_command:`, the `# codex:` / `# gemini:` blocks) gains the fence prefix like a live line —
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
`fab config upgrade` is the **only** writer of `config.yaml`, which retires the comment-clobbering
`setFabVersion` bug class at the root. `fab upgrade-repo` **auto-runs** the upgrader after sync
(decision 4, fail-open: a fab-go predating the subcommand prints a reminder and the upgrade continues).
The kit-version stamp in the BEGIN line makes staleness visible and enables a *later* `--check` drift
mode (not in this change).

### `fab_version` → `fab/.fab-version` [Change 3 — landed] (decision 1)

`fab_version` moves out of `config.yaml` entirely into a new plain-text sibling `fab/.fab-version`
(one line, bare semver + newline, committed, sibling to `fab/.kit-migration-version`). This is what lets
`fab config upgrade` be config.yaml's single writer — the one machine-managed field the old masher owned
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
`fab config init --project`; when that fab-go predates the subcommand, it falls open to a minimal
**embedded stub** `config.yaml` (a fresh repo must never fail preflight for lack of a config.yaml — not a
printed instruction). The registry carries the init/seed metadata (`InitSeed`) marking which fields are
written live at init; fab-kit's mechanical detection — the repo folder name, an existing `src/`, and the
ecosystem marker-table `test_paths` — becomes generator input, passed as `--name`/`--source-path`/
`--test-path` flags so those A-class fields land live (`project.description` is not mechanically
detectable and is added interactively by `/fab-setup`). An empty-seed `fab config init --project`
(no flags) emits the header + fence only.
The former `TestConfigReferenceSupersetsScaffoldKeys` guard is re-anchored to a registry-internal
invariant (the init-seeded key set ⊆ the registry key set), since there is no scaffold file to compare
against.
