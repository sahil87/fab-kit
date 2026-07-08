# Intake: Config Reference Metadata Table

**Change**: 260708-ff2v-config-reference-metadata-table
**Created**: 2026-07-08

## Origin

One-shot `/fab-new` invocation. Raw input:

> Restructure fab config reference (change 6nke, src/go/fab/internal/configref/) from a text-template renderer into a per-field metadata table: default, description, scope (project/system/both), advertise (C flag), renamed_from. fab config reference renders the table as commented YAML (output equivalent to today), add --json. Land the spec at docs/specs/config.md recording the schema decisions. Update _cli-fab.md and tests per constitution CLI-change rule, and the specs index. See fab/plans/sahil/config-upgrade.md Change 1 for full context.

This is **Change 1 of 3** in the config-upgrade effort documented at `fab/plans/sahil/config-upgrade.md` (written 2026-07-08 after a `/fab-discuss` session; **all six open decisions in that doc are user-confirmed** — cascade order, presence=intent, fence contract, auto-run, system config path, scope taxonomy). Changes 2 (cascade resolution + `fab config show --origin` + `fab config init --system`) and 3 (`fab config upgrade` + migration) build on the table this change introduces. Read that plan doc during apply — it is the design authority for cross-change context.

## Why

1. **The pain point**: `fab config reference` (shipped in change 6nke) is a text-template renderer — `internal/configref` holds one 130-line template string with constants injected (`refData`). There is no per-field metadata: no machine-readable notion of a field's default, scope, or override status, and no `--json`. The downstream changes of the config-upgrade effort need exactly that: Change 2's cascade resolver and `fab config init --system` need per-field `scope`; Change 3's `fab config upgrade` fence generator needs `advertise` (which unset fields to scaffold as comments) and `renamed_from` (mechanical rename carry-forward); `fab config show --origin` needs the canonical default set. A prose template cannot answer any of these queries.

2. **The consequence of not doing it**: each downstream consumer regrows its own copy of the schema (defaults, scope lists, comment text), reintroducing exactly the drift the 6nke design eliminated ("no second copy to drift"). The broader effort — retiring the masher-wipes-comments bug class (observed 2026-07-03 and 2026-07-08; `setFabVersion` root cause; stopgap line-splice shipped as 260708-yogn PR #473) — stalls without a canonical, queryable field table as its single source.

3. **Why this approach**: invert the data/prose relationship. Today the prose template is primary and constants are injected into it; after this change a per-field metadata table is primary and both renderings (commented YAML, JSON) are generated from it. The existing no-drift invariant is preserved: defaults in the table are still sourced from the canonical Go constants (`agent.DefaultSessionCommand`, `agent.DefaultTier` over `agent.TierNames`, `agent.StageNames`) — the table adds structure, never a second copy of values. This change is deliberately **additive and independently shippable**: `fab config reference` output stays equivalent, `--json` is new surface, and no user data is touched (no migration).

## What Changes

### 1. Per-field metadata table (`src/go/fab/internal/configref/`)

Replace the `refData` struct + monolithic `referenceTemplate` with a field registry — an **ordered** slice of field entries (order = rendering order, giving deterministic, byte-stable output exactly as today). Per field:

```go
// Illustrative shape — apply owns the final struct layout.
type Field struct {
    Key         string // dotted path, e.g. "agent.tiers", "project.name", "true_impact_exclude"
    Default     any    // canonical built-in default (typed; sourced from Go constants where they exist)
    Description string // drives the generated comment block for this field
    Scope       Scope  // project | system | both
    Advertise   bool   // the C flag: scaffold as commented reference in Change 3's managed fence
    RenamedFrom string // previous key path for mechanical rename carry-forward ("" today)
}
```

Design constraints on the table (from the plan doc + the existing package contract):

- **Single source, no second copy**: every default that has a canonical Go constant is referenced from it, not copied (`agent.DefaultSessionCommand`, per-tier profiles via `agent.DefaultTier`/`agent.TierNames`, stage names via `agent.StageNames`). The existing fail-loud `gatherData` invariants (every tier has a profile and a stage grouping) carry over.
- **Row granularity = override unit**: rows are the meaningful override surfaces, roughly: `project.name`, `project.description`, `project.linear_workspace`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `providers`, `agent.tiers`, `stage_hooks`, `branch_prefix`, `fab_version`. Map-valued fields (`providers`, `agent.tiers`, `stage_hooks`) are single rows with structured defaults — this matches Change 2's per-field deep-merge semantics (maps merge per-key, lists replace, scalars replace).
- **Section-level prose must survive**: today's output carries narrative comment blocks that are not one-line field descriptions (the providers explanation, per-provider notes, the three-provider starter template, the fixed stage→tier mapping comment). The schema must accommodate multi-line/block commentary (e.g. a long-form comment field per row, or interleaved section-prose entries) so the rendered output keeps its current documentation quality and the existing string-assertion tests keep passing. Apply decides the exact representation.
- **Defaults vs. examples**: some current values are *examples*, not built-in defaults (`source_paths: [src/]`, `test_paths: ["**/*_test.go"]` — the binary default for both is empty). The table's `default` must be the *canonical* default (what Change 2's cascade falls back to); rendering-only example values live in the description/comment side, not in `default`. This distinction is load-bearing for Change 2 and is recorded in the spec.

**Scope assignments** (plan decision 6, user-confirmed): `agent.tiers`, `providers` = **both**; `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist` = **project**. Rationale: the system layer (`~/.fab-kit/config.yaml`) is restricted to preference-class fields; semantics-class fields stay repo-reproducible for teammates/CI. Fields the plan does not enumerate (`stage_hooks`, `branch_prefix`, `fab_version`) default to **project** (conservative: system-visibility is opt-in per the same rationale; scope enforcement only lands in Change 2, so re-classification later is a one-line data change). `fab_version` additionally stays documented as machine-managed (it leaves config.yaml entirely in Change 3).

**Advertise assignments**: `true` for the optional override surfaces a project has typically *not* set live (`agent.tiers`, `providers`, `checklist.extra_categories`, `true_impact_exclude`, `stage_hooks`, `branch_prefix`, `test_paths`); `false` for scaffold-seeded identity fields (`project.*`, `source_paths`) and machine-managed `fab_version`. The fence example in the plan doc is illustrative, not exhaustive — the final set is recorded in the spec at apply. Note `advertise` has **no behavioral consumer in this change** (the fence generator is Change 3); here it is data + `--json` exposure only.

**`renamed_from`**: mechanism plumbed, value `""` for every row today. Historical renames (`agent.spawn_command` → `providers.claude.session_command`, tykw) were already handled by shipped migrations and are NOT backfilled — the field serves *future* renames so they stop needing hand-written migrations.

### 2. Renderer: commented YAML generated from the table

`Render()` keeps its signature and contract (byte-stable for a given binary version, error on broken invariants) but walks the field table instead of executing one monolithic template. **Output equivalent to today's** means contract-equivalent, not byte-identical:

- Same key coverage (every current key documented, commented or live)
- Same live/commented split (baseline keys live with example values; `stage_hooks`/`branch_prefix`/claude's `dispatch_command`/codex/gemini blocks commented — uncommenting is opting in)
- Same documented semantics that the existing tests assert verbatim: the provider command strings, the "fallback from dispatch_command to session_command" phrase, `{model}`/`{effort}` placeholders, retired keys (`review_tools`, `spawn_command`) absent
- All nine existing tests in `src/go/fab/cmd/fab/config_test.go` keep passing (round-trip via `config.LoadPath`, binary-key coverage via reflection, scaffold-key superset, byte-stability, cobra end-to-end, placeholder/provider/three-provider-template/retired-key contracts)

### 3. New `--json` flag on `fab config reference`

`fab config reference --json` emits the field table as machine-readable JSON to stdout — the tooling surface Changes 2–3 and external tools consume. Suggested shape (final field naming recorded in the spec at apply):

```json
[
  {
    "key": "agent.tiers",
    "default": {
      "default":  {"provider": "claude", "model": "...", "effort": "..."},
      "operator": {"provider": "claude", "model": "...", "effort": "..."},
      "doing":    {"provider": "claude", "model": "...", "effort": "..."},
      "review":   {"provider": "claude", "model": "...", "effort": "..."},
      "fast":     {"provider": "claude", "model": "...", "effort": "..."}
    },
    "description": "Per-stage model override. ...",
    "scope": "both",
    "advertise": true
  },
  {
    "key": "project.name",
    "default": null,
    "description": "Project display name. Read by skills for orientation and PR bodies.",
    "scope": "project",
    "advertise": false
  }
]
```

- Flat JSON array in table (= rendering) order; deterministic and byte-stable like the YAML output
- `renamed_from` omitted when empty (`omitempty`)
- Without the flag, output is the commented YAML exactly as before; the command stays a pure query (no file writes, exit 0 on success, extra positional args still rejected)
- stdlib `encoding/json` only — no new dependencies

### 4. New spec: `docs/specs/config.md` (+ specs index row)

The spec records the schema decisions — this change is where the config-system design intent lands (constitution VI: specs are pre-implementation, human-curated). Contents:

- The per-field metadata schema (fields, granularity rule, defaults-from-constants invariant, defaults-vs-examples distinction)
- The scope taxonomy (project/system/both) with the decision-6 assignments and rationale
- `advertise` semantics (the A/B/C field-category model from the plan doc)
- `renamed_from` carry-forward semantics
- The `--json` output shape
- Effort-level context recorded as forward-looking intent, clearly marked as landing in Changes 2–3: the override cascade (project > `~/.fab-kit/config.yaml` > built-in defaults; per-field deep merge — maps per-key, lists replace, scalars replace), presence=intent (decision 2), the managed-fence contract (decision 3), system config path (decision 5)

Add the `config` row to `docs/specs/index.md` (hand-edited — the specs index is human-curated, unlike the generated memory indexes).

### 5. `_cli-fab.md` + `SPEC-_cli-fab.md` updates

`src/kit/skills/_cli-fab.md` § fab config reference currently states "No flags, no arguments". Update: document `--json` (shape, determinism, pure-query unchanged), and refresh the "Generated, not hand-written" paragraph to describe the per-field metadata table as the generation source. Per the constitution: the CLI change obligates the `_cli-fab.md` update + tests; the skill-file edit obligates the corresponding `docs/specs/skills/SPEC-_cli-fab.md` update. (Deployed copies under `.claude/skills/` are produced by `fab sync` — never edited directly.)

### 6. Tests

Constitution CLI-change rule: test updates accompany the change.

- **Existing** (`config_test.go`): all nine keep passing, adapted only where they reference internals (e.g. anything importing `refData` — currently none; tests call `configref.Render()` and the cobra command, so they should pass largely untouched)
- **New**:
  - `--json` output parses as valid JSON and is byte-stable across renders
  - JSON key set ≡ the YAML reference's documented key set (the two renderings can't drift apart)
  - Every table row has a non-empty `description` and a valid `scope` ∈ {project, system, both} (registry lint — fail-loud like `gatherData` today)
  - Scope assignments match decision 6 for the enumerated fields
  - `--json` with an extra positional arg rejected; plain `reference` output unchanged in contract

## Affected Memory

- `_shared/configuration`: (modify) — `fab config reference` is now generated from a per-field metadata table (default/description/scope/advertise/renamed_from); `--json` added; scope taxonomy recorded
- `distribution/kit-architecture`: (modify) — the 6nke bullet's `internal/configref` description updates from text-template generator to metadata-table generator with dual renderings

## Impact

- **Code**: `src/go/fab/internal/configref/configref.go` (restructure, ~257 lines today), `src/go/fab/cmd/fab/config.go` (add `--json`), `src/go/fab/cmd/fab/config_test.go` (adapt + extend)
- **Docs**: `docs/specs/config.md` (new), `docs/specs/index.md` (add row), `src/kit/skills/_cli-fab.md` (§ fab config reference), `docs/specs/skills/SPEC-_cli-fab.md`
- **Not touched**: `fab/project/config.yaml` semantics, the scaffold, `internal/config` (the loader — cascade is Change 2), no migration (nothing restructures user data), no new dependencies
- **Downstream**: Changes 2 and 3 of the config-upgrade effort consume the table (`scope` → cascade enforcement + `init --system`; `advertise` → fence scaffold; `renamed_from` → upgrade carry-forward; `default` → `show --origin`)

## Open Questions

- None — the design was resolved in the 2026-07-08 `/fab-discuss` session recorded in `fab/plans/sahil/config-upgrade.md` (all six decisions user-confirmed).

## Assumptions

<!-- STATE TRANSFER: This table is the sole continuity mechanism between the intake-stage
     agent and the apply-entry agent (which co-generates plan.md). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The field table is the single source; defaults referenced from canonical Go constants (`agent.DefaultSessionCommand`, `agent.DefaultTier`, `agent.StageNames`) — no second copy | Mandated by both the existing package doc (6nke no-drift invariant) and the plan doc ("the single source") | S:90 R:85 A:95 D:90 |
| 2 | Confident | "Output equivalent to today" = contract equivalence (same keys, live/commented split, tested prose strings, all nine existing tests pass), not byte-identical | Plan says "should stay equivalent"; the existing test suite is the executable definition of the contract; byte-identity would defeat the restructure | S:70 R:80 A:75 D:65 |
| 3 | Confident | `--json` shape: flat JSON array in table order, per-field objects `{key, default, description, scope, advertise, renamed_from(omitempty)}`, deterministic; final naming recorded in spec | Plan says only "add `--json` for tooling"; array-of-fields is the obvious dump of an ordered table; easily revised before Change 2 consumes it | S:60 R:85 A:80 D:60 |
| 4 | Confident | Fields the plan's scope taxonomy doesn't enumerate (`stage_hooks`, `branch_prefix`, `fab_version`) get scope=project | Decision-6 rationale ("system layer restricted to preference-class fields") + conservative default; enforcement only lands in Change 2, so re-classification is a one-line data change | S:65 R:85 A:70 D:60 |
| 5 | Confident | advertise=true for optional override surfaces (`agent.tiers`, `providers`, `checklist.extra_categories`, `true_impact_exclude`, `stage_hooks`, `branch_prefix`, `test_paths`); false for `project.*`, `source_paths`, `fab_version` | The plan's fence example is illustrative, not exhaustive; no behavioral consumer until Change 3, so the set is cheap to revise; final set recorded in the spec at apply | S:45 R:88 A:50 D:35 |
| 6 | Certain | `renamed_from` ships empty on every row; historical renames (tykw's `agent.spawn_command` move) are NOT backfilled | Plan frames it as "future field renames"; historical renames already shipped as migrations that no longer re-apply | S:70 R:90 A:85 D:75 |
| 7 | Confident | `docs/specs/config.md` covers the full effort's design intent (cascade, presence=intent, fence, system path) clearly marked as Changes 2–3, with Change 1's table schema in authoritative detail | Plan: "this is where the schema decisions are recorded"; specs are pre-implementation intent (constitution VI), so recording the confirmed forward design fits; trivially editable | S:60 R:90 A:75 D:55 |
| 8 | Confident | The table schema carries block-level/section prose (multi-line commentary) in addition to per-field descriptions, so today's narrative comment blocks survive | Existing tests assert verbatim prose (provider notes, no-fallback phrase); a one-line description per field cannot reproduce today's output quality; exact representation is apply's call | S:50 R:80 A:60 D:45 |
| 9 | Certain | No migration ships with this change | Constitution migration rule triggers on user-data restructure; this change writes no user data (`fab config reference` stays a pure query; config.yaml untouched until Change 3) | S:85 R:80 A:95 D:90 |

9 assumptions (3 certain, 6 confident, 0 tentative, 0 unresolved).
