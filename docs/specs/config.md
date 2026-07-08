# Config Schema — the per-field metadata table

> **Status:** Design intent (pre-implementation, Constitution VI). This spec is human-curated. It
> records the config-system schema decisions resolved in the 2026-07-08 `/fab-discuss` session and
> written up in the config-upgrade effort's backlog doc (`fab/plans/sahil/config-upgrade.md`, all six
> decisions user-confirmed). It is written across the **three-change** config-upgrade effort:
> **Change 1** (260708-ff2v) — the per-field metadata table + `fab config reference` restructure +
> `--json` — is landed here in authoritative detail. **Changes 2-3** (cascade resolution, visibility
> commands, `fab config upgrade` + migration) are recorded as forward-looking intent, clearly marked
> `[Change 2]` / `[Change 3]`, so the design authority lives in one place.
>
> The canonical schema is the Go field table in `src/go/fab/internal/configref/`; this doc is its
> human-readable rationale. Defaults that have a Go constant are sourced from that constant, never
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
defaults that have a canonical Go constant are still referenced from the constant, never copied. The
table adds *structure*, never a second copy of *values*.

---

## The per-field schema

Each row of the table models one **override unit** — a meaningful override surface, coarser than a
leaf key. Map-valued fields (`providers`, `agent.tiers`, `stage_hooks`) are single rows with
structured defaults, matching the per-field deep-merge semantics [Change 2] uses (maps merge per-key,
lists replace, scalars replace).

| Field | Meaning |
|-------|---------|
| `key` | Dotted path of the override surface (e.g. `agent.tiers`, `project.name`, `true_impact_exclude`). The identity used by the JSON dump and the JSON↔YAML key-parity guard. |
| `default` | The **canonical** built-in default (typed). What the cascade [Change 2] falls back to when no layer overrides the field. A field with no built-in default carries `null` — uniformly, never a typed empty (`[]`/`{}`/`""`). See § Default semantics. |
| `description` | One-line summary of the field. Required (non-empty) — the registry lint rejects an empty description. Feeds the JSON dump and, later, the generated comment scaffold [Change 3]. |
| `scope` | Override visibility across the cascade layers: `project` / `system` / `both`. See § Scope taxonomy. |
| `advertise` | The "C flag": whether [Change 3]'s managed fence scaffolds this field as a commented reference when it is not overridden. See § Advertise semantics. |
| `renamed_from` | Previous key path for mechanical rename carry-forward. `""` on every row today; serves *future* renames. See § renamed_from. |

### Defaults are sourced from constants — no second copy

Every default that has a canonical Go constant is referenced from it, not copied: the claude session
command from `agent.DefaultSessionCommand`, the per-tier profiles via `agent.DefaultTier` over
`agent.TierNames()`, the stage names via `agent.StageNames()`. The registry construction fails loud
(returns an error rather than emitting a degraded reference) if a tier reported by `TierNames()` has no
`DefaultTier` profile, or a tier has no stage grouping, or a row has an empty description or an invalid
scope — the same fail-loud discipline the pre-metadata-table renderer applied to its tier invariants.

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
always denotes a real built-in value (today: the `providers` claude default and the five `agent.tiers`
profiles); every other row is `null`.

### Section-level prose lives on the row — the segment

One-line `description`s cannot carry the narrative documentation blocks the reference needs (the
providers explanation, the per-provider dispatch notes, the three-provider starter template, the fixed
stage→tier mapping). Each table row therefore carries — alongside its one-line `description` — the
**rendered YAML segment**: the field's commented block as it appears in the reference. `fab config
reference` is generated by walking the table and concatenating those segments in order; there is no
separate template. The `description` (the machine-readable one-liner, exposed in `--json`) and the
`segment` (the human-readable block, exposed in the YAML) are two projections of **one** row, not a
second copy of the schema to drift — a field's documentation is authored once, on its row. The rows for
map-valued fields (`providers`, `agent.tiers`, `stage_hooks`) build their segment by interpolating the
same Go constants their `default` reads, so the rendered prose carries no literal copy of any value.
The existing reference tests assert those blocks verbatim; the restructure preserves them byte-for-byte.

---

## Scope taxonomy (decision 6)

`scope` states which cascade layer(s) may override a field. The rationale: the **system** layer
(`~/.fab-kit/config.yaml`, [Change 2]) is restricted to *preference-class* fields — personal model/harness
choices — while *semantics-class* fields stay in the project file so the repo remains reproducible for
teammates and CI.

| scope | Meaning | Fields |
|-------|---------|--------|
| `both` | Overridable in either the project or the system layer (preference-class). | `agent.tiers`, `providers` |
| `project` | Overridable only in the project file (semantics-class, repo-reproducible). | `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, and (conservative default) `stage_hooks`, `branch_prefix`, `fab_version` |
| `system` | Overridable only in the system layer. | *(none today; the value exists for completeness and [Change 2])* |

Fields the decision-6 taxonomy does not enumerate (`stage_hooks`, `branch_prefix`, `fab_version`)
default to `project` — the conservative choice, since system-visibility is opt-in per the same
rationale. `fab_version` is additionally machine-managed (it leaves `config.yaml` entirely in
[Change 3]). Scope is **metadata-only in Change 1** — no code branches on it yet. Enforcement (a
project-scoped field in the system file is *ignored with a warning*, fail-open) lands in [Change 2],
so re-classifying a field is a one-line data change until then.

---

## Advertise semantics — the A/B/C field-category model

`advertise` is the "C flag" of the field-category model the config-upgrade effort uses. Under that
model, at [Change 3]'s `fab config upgrade` time, every field is one of:

- **A) user-overridden** → written as live YAML above the managed fence.
- **B) not overridden** → absent from the file (inherited from defaults).
- **C) not overridden but worth advertising** → scaffolded as a commented reference *inside* the
  managed fence, so the user can discover and opt in.

`advertise: true` marks the C-eligible fields — the optional override surfaces a project has typically
*not* set live: `agent.tiers`, `providers`, `checklist.extra_categories`, `true_impact_exclude`,
`stage_hooks`, `branch_prefix`, `test_paths`. `advertise: false` marks scaffold-seeded identity fields
(`project.*`, `source_paths`) and machine-managed `fab_version`, which are not re-advertised.

`advertise` has **no behavioral consumer in Change 1** — it is data + `--json` exposure only. The fence
generator that reads it is [Change 3], so the set is cheap to revise until then.

---

## renamed_from — mechanical rename carry-forward

`renamed_from` names a field's previous key path so [Change 3]'s `fab config upgrade` can carry a
user's value forward across a rename mechanically, instead of each rename needing a hand-written
migration. It is `""` on **every row today**: historical renames (e.g. `agent.spawn_command` →
`providers.claude.session_command`, change tykw) were already handled by shipped migrations and are
**not** backfilled. The field serves *future* renames only. The `--json` dump omits it when empty.

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
    "key": "agent.tiers",
    "default": {
      "default":  { "provider": "claude", "model": "...", "effort": "..." },
      "operator": { "provider": "claude", "model": "...", "effort": "..." },
      "doing":    { "provider": "claude", "model": "...", "effort": "..." },
      "review":   { "provider": "claude", "model": "...", "effort": "..." },
      "fast":     { "provider": "claude", "model": "...", "effort": "..." }
    },
    "description": "Per-stage model override. ...",
    "scope": "both",
    "advertise": true
  }
]
```

- The `agent.tiers` `default` is a map **keyed by tier name** (one entry per `agent.TierNames()` tier —
  `default`, `operator`, `doing`, `review`, `fast`), each a `{provider, model, effort}` profile; the
  first-level `default` key is the *default tier*, not a wrapper. Likewise `providers.default` is keyed
  by provider name.
- `renamed_from` is omitted when empty (`omitempty`), so it is absent from every object today.
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

## Forward-looking intent (Changes 2-3)

Recorded here so the config-system design lives in one place. These are **not** implemented in
Change 1.

### Override cascade [Change 2]

Effective config resolves across three layers, highest precedence first:

1. **project** — `fab/project/config.yaml`
2. **system** — `~/.fab-kit/config.yaml` (co-located with the version cache; XDG path rejected — decision 5)
3. **built-in defaults** — the Go field table in the `fab` binary (this spec's table)

Merge is **per-field deep merge**: maps merge per-key (the existing `agent.tiers` precedent), **lists
replace** (never concatenate), scalars replace. Scope enforcement is **fail-open**: a project-scoped
field appearing in the system file is ignored with a warning — config must never brick.

### Visibility commands [Change 2]

- `fab config show --origin` — the effective config with per-field provenance (the
  `git config --show-origin` precedent). It surfaces typo'd overrides that today silently no-op.
- `fab config init --system` — writes a `~/.fab-kit/config.yaml` scaffold containing ONLY
  `scope: system`/`both` fields, all commented — generated from this same table so it can't drift.

### Presence = intent [Change 2] (decision 2)

Any live field in a config file is an **override**, even if its value equals the default. `fab config
upgrade` never auto-removes a live field; B-hygiene ("these fields equal current defaults — remove?")
is advisory only. A value-diff classifier cannot distinguish "deliberately pinned" from "never touched",
and auto-dropping would silently change behavior when the default later moves.

### The managed fence [Change 3] (decision 3)

`fab config upgrade` regenerates a byte-stable, idempotent **managed fence** of commented C-fields
(`advertise: true`, not currently overridden), delimited by byte-exact `>>>`/`<<<` splice anchors
carrying a kit-version stamp. Upgrade rewrites ONLY between the markers; everything outside — including
the user's own comments on A-fields — is the user's. Unknown fields are parked in a
`# removed in X.Y.Z, your value was:` block below the fence, never silently deleted. After this change,
`fab config upgrade` is the *only* writer of `config.yaml`, which retires the comment-clobbering
`setFabVersion` bug class at the root. `fab_version` moves out of `config.yaml` to `fab/.fab-version`
(decision 1), and `fab upgrade-repo` auto-runs the upgrader (decision 4, fail-open).
