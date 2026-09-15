# Intake: Config Read-Model Redesign + UX Follow-ups

**Change**: 260808-fp02-config-read-model-redesign
**Created**: 2026-08-09

## Origin

> /fab-new fp02 — backlog entry `[fp02]` (2026-08-08): "Config read-model redesign + UX follow-ups (post-C2/C5; evolved in 2026-08-08/09 discussion)".

Backlog-driven, conversational intake. The backlog entry pre-flagged two "DECISIONS TO SETTLE FIRST"; both were asked at intake and resolved by the user, plus a third question on sequencing against the not-yet-started C5 (source consolidation):

1. **Precedence (decision a)**: user chose the **proposal** — system overrides project (`Defaults < Project < System < Env`), inverting the shipped git-precedent order.
2. **Scope enforcement (decision b)**: user chose to **keep** it — project-scoped keys stay pruned from the system layer; the drop-entirely option was declined.
3. **C5 relationship**: user chose **proceed now, subsume overlaps** — this change owns the read-model seam (defaults as a real merge layer; the `show --origin` knob-blind drill-down fix folds in here); C5's remaining items stay a separate later change.

The combination of 1+2 is load-bearing: because scope enforcement survives, the precedence inversion only ever affects **preference-class keys** (`scope: both`/`system`) — semantics-class keys (`source_paths`, `test_paths`, …) cannot legally appear in the system layer, so read-side repo hermeticity is fully preserved despite the flip.

Prereqs already shipped: C1 (env layer, #553), C2 (six-verb surface, #555 — `260808-ihiv`) are in main's history at this branch.

## Why

1. **The presence-semantics pain**: the shipped read model carries explicit-null/presence machinery — `originValue{value, set}` in `cmd/fab/config.go` tracks "present but null" separately from "absent", and `dispatch.reap_done` needs a `*bool` (`nil` = unset = `true`) purely because absent and explicit-`false` were indistinguishable at unmarshal. None of this buys the user anything: an explicit `key: null` shadowing a lower tier is a footgun, not a feature. A uniform per-leaf merge where **empty falls through** deletes the whole distinction.
2. **The precedence pain**: for preference-class keys ("which worker provider do *I* like on *this machine*"), the shipped project-over-system order is backwards — a repo's committed *suggestion* beats the user's machine-wide *choice*, so a personal `~/.fab-kit/config.yaml` preference silently loses in any repo that happens to pin the key. With scope enforcement retained, inverting to system-over-project gives "my machine beats the repo's suggestion" for exactly the preference keys, and nothing else.
3. **The visibility pain**: keyed `fab config show <key> --origin` is winner-only — the shadowed tiers are invisible, so diagnosing "why is my override not taking effect" is archaeology. `set` succeeds silently even when a higher tier shadows the written value; `unset` no-ops with a notice that doesn't say where the key IS live. All three are the same missing feature: the resolver knows the full stack but only ever shows the winner.
4. **The subsumed C5 defect**: the `agent.profiles.*.provider` drill-down rows in `show --origin` are knob-blind — composed from the registry's nil-config `Default`, they read `claude # default` even when a depth knob names another provider (documented under-reporting in `docs/memory/_shared/configuration.md`). The full-stack redesign rewrites exactly this projection, so the fix belongs here (user-confirmed), not in C5.

If not fixed: the `*bool`/`originValue.set` machinery keeps accreting special cases with every new field; per-machine preferences keep losing to repo files; shadowing stays undiagnosable without reading three files by hand.

Why this approach over alternatives: one merge rule (per-leaf, empty-skip) applied identically across all four tiers replaces three different mechanisms (presence tracking, point-of-use fallback special cases, winner-only origin projection). The half-drop of scope enforcement (write allowed, read pruned) was explicitly rejected in the backlog as ignored-write confusion; the full drop was declined by the user at intake.

## What Changes

### 1. Read model: per-leaf deep merge with empty-skip

Effective config becomes, per leaf: **`Defaults < Project < System < Env`** — walk tiers from highest (env) down and take the first tier defining a **non-empty** value for that leaf (lodash `_.merge` semantics; lists replace wholesale, never concatenate).

- **Merge unit**: maps merge per-key recursively; lists and scalars are leaves (replace wholesale).
- **Empty-skip**: a leaf whose value at some tier is `null`, `""` (empty string), `[]`, or `{}` neither wins nor blocks — it falls through to the next tier down. This removes explicit-null/presence semantics from the READ side entirely.
- **`false` and `0` are real values, never skipped** — a bool/int override must survive the merge (`dispatch.reap_done: false` in the project file resolves `false`). The `dispatch.column_width` accessor's out-of-range (`0`/`100`) → default clamp is point-of-use validation and stays.
- Implementation seam: `internal/config.LoadPath` / `LoadLayers` (`src/go/fab/internal/config/config.go`) — the current `deepMerge(deepMerge(systemMap, projectMap), envMap)` becomes the four-tier empty-skip merge in the new order.

### 2. Precedence inversion: Env > System > Project > Defaults

- The system layer moves **above** the project layer. Env stays top; defaults stay bottom.
- **Scope enforcement is retained unchanged**: the system layer still prunes project-scoped keys with a `fab: warning:` (fail-open), `fab config set --system` stays gated on `scope ∈ {system, both}`, and project-scoped env vars stay warn-and-ignore. Consequence: the inversion is observable only for preference-class keys, so repo reproducibility for semantics keys is untouched.
- This is a **behavior change** for any machine where both the project file and the system file define the same preference key (project used to win; system now wins). No on-disk data changes shape, so **no migration file** — the flip is carried by docs and the release notes, not a migration (constitution's migration rule covers user-data *restructuring*, which this is not).
- The fail-open contract is otherwise unchanged (absent system file ⇒ pre-cascade behavior; malformed system file ⇒ warn + skip; malformed project file ⇒ error).

### 3. Defaults become a materialized merge layer

- The built-in defaults tier (the registry's canonical `default` values, `internal/configref`) is projected to a YAML-shaped map and merged as the bottom tier, instead of existing only as point-of-use fallbacks. This is what makes the four-tier model real for the resolver, the origin projection, and the shadow warnings — and it realizes the precondition of C5's deferred "defaults as true layer 0" stretch item (user-confirmed subsumption).
- **Point-of-use fallback seams are retained** (as now-redundant safety) — collapsing them wholesale remains C5 item 1 territory. With defaults materialized, they simply stop triggering.
- Enabled cleanup: `dispatch.reap_done` can drop its `*bool` for a plain `bool` — the merged map always carries a value for the leaf, and an explicit `false` survives the empty-skip merge (false is not empty). <!-- assumed: reap_done *bool→bool collapse lands in this change since the materialized-defaults merge makes it safe; apply may defer it if the unmarshal path still sees pre-merge maps anywhere -->
- `originValue{value, set}` and its helpers (`highestOriginValue`, the set-tracking in `flattenOrigin`) collapse: per-leaf origin becomes "the highest tier with a non-empty leaf" — the same rule as the merge itself, one definition.

### 4. `show <key> --origin`: full-stack layer listing (default; subsumes C5 item 5)

- Keyed `fab config show <key> --origin` changes from winner-only to a **full-stack listing by default**: one line per tier that defines the key (non-empty), with the winner marked. **No new flag**; bare all-keys `show --origin` stays winner-only (one line per leaf).
- Illustrative shape (exact rendering is plan latitude):

  ```
  $ fab config show agent.workers --origin
  agent.workers = codex    # env $FAB_AGENT_WORKERS  (effective)
  agent.workers = kimi3    # system /home/u/.fab-kit/config.yaml  (shadowed)
  agent.workers = claude   # default  (shadowed)
  ```

- Map-valued keys list per-leaf, same drill-down as today, each leaf showing its defining tiers.
- **Knob-blind drill-down fix (from C5 item 5)**: the composed `agent.profiles.*.provider` rows compose against the **live config** instead of the nil-config registry default, so a depth knob naming a non-claude provider is reported honestly. The documented under-reporting note in `_shared/configuration.md` (and its pointer into `runtime/providers-and-profiles.md` § Design Decisions) comes out.

### 5. `set`: shadow warning

- `fab config set <key> <value>` (either target) prints a warning when the **written value is not the effective value** because a higher tier shadows it — e.g. writing to project while system or env defines the key; writing `--system` while env defines it. Exit 0 (the write itself succeeded); the warning names the shadowing tier/variable, e.g. `fab: warning: agent.workers is shadowed by $FAB_AGENT_WORKERS — the written value is not in effect this session`.
- `set <key> ""` (an empty scalar) is refused with guidance: under empty-skip an empty leaf falls through and can never be effective — the right verb is `fab config unset`. <!-- assumed: refuse (not warn-and-write) for empty-scalar set — writing a value the read model is defined to ignore is a pure footgun; unset is always what was meant -->

### 6. `unset`: no-op notice names the live tier

- Unsetting a key not present in the target file stays an exit-0 no-op, but the notice now says **where the key IS live** and suggests the right target, e.g.:
  `fab config unset agent.workers` → `not set in fab/project/config.yaml; live in /home/u/.fab-kit/config.yaml — use: fab config unset agent.workers --system` (and names the env variable when env defines it, which unset cannot remove).

### 7. Docs, help text, and the sweep class

- `docs/specs/config.md` — § Override cascade rewritten (order, empty-skip, defaults-as-layer), § six verbs (`show --origin` keyed full-stack, `set`/`unset` notices), § Default semantics (the `*bool` note if collapsed).
- `src/kit/skills/_cli-fab.md` — § "The config cascade (environment > project > system > defaults)" heading + body, and the `dispatch.reap_done` cascade note (~line 702); + `docs/specs/skills/SPEC-_cli-fab.md` mirror.
- **String-literal sweep** (the ioku lesson): the cobra help text in `cmd/fab/config.go` spells the order out ("environment-over-project-over-system", `--origin` flag help); registry segment prose and the managed-fence advert text saying "settable once machine-wide" (now *winning* machine-wide). Grep both order spellings repo-wide (`project > system`, `system > project`, `environment > project > system`, "project file wins", etc.) including user-facing string literals.
- Memory: see Affected Memory.
- `fab/backlog.md`: `[fp02]` marked done at archive time (standard flow).

## Affected Memory

- `_shared/configuration.md`: (modify) — cascade order + empty-skip semantics in § Override Cascade; the six-verb section gains the full-stack keyed `--origin`, `set` shadow warning, `unset` live-tier notice; the knob-blind drill-down under-reporting paragraph is removed/rewritten as fixed.
- `runtime/providers-and-profiles.md`: (modify) — the "`DefaultProfile` Is Resolution Against a Nil Config" Design Decisions entry updates: the `show --origin` drill-down now composes against the live config.

## Impact

- **Go** (`src/go/fab/`): `internal/config/config.go` (`LoadPath`, `LoadLayers`, `deepMerge` → four-tier empty-skip merge, layer order), `cmd/fab/config.go` (`show`/`--origin` projection rewrite, `originValue` machinery removal, `set` shadow warning + empty refusal, `unset` notice, help-text literals), `internal/configref` (defaults-map projection for the merge tier), `internal/config` struct cleanup (`DispatchConfig.ReapDone *bool` → `bool` candidate). `internal/configscope` unchanged (enforcement kept). Tests throughout: merge-order/empty-skip table tests, origin golden output, shadow-warning cases, existing cascade tests updated for the new order.
- **Kit + specs**: `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md`, `docs/specs/config.md`.
- **Memory**: the two files above.
- **Behavior-change blast radius**: every config consumer reads through `LoadPath`, so the precedence flip and empty-skip apply uniformly (preflight, resolve-agent, dispatch, agent, operator, batch…). Observable change is limited to (a) machines whose system file and project file both set the same preference key, and (b) configs carrying explicit-null/empty leaves that previously shadowed.
- **Not in scope** (stays C5): folding the `dispatch.*` Go-constant defaults into `defaults.yaml`, retiring the fab-kit embedded stub config, `upgrade --check`, fence scope annotations, collapsing the point-of-use fallback seams.

## Open Questions

None — the backlog's two pre-flagged decisions (precedence, scope enforcement) and the C5-overlap question were asked and resolved at intake; see Assumptions #2–4.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Target read model: per-leaf deep merge, lodash `_.merge` semantics, lists replace wholesale, empty/null leaves fall through; removes explicit-null/presence read semantics and the `originValue.set` machinery | Backlog `[fp02]` TARGET MODEL, verbatim | S:90 R:70 A:90 D:90 |
| 2 | Certain | Precedence inverts to Env > System > Project > Defaults | Asked — user chose the proposal ("my machine beats the repo's suggestion"); with scope kept, only preference-class keys are affected | S:95 R:70 A:95 D:95 |
| 3 | Certain | Scope enforcement kept unchanged (system-layer pruning, set gating, env gating) | Asked — user declined the all-or-nothing drop; read-side hermeticity preserved | S:95 R:80 A:95 D:95 |
| 4 | Certain | Proceed now; subsume C5 item 5 (knob-blind drill-down fix) + defaults-as-materialized-layer; the rest of C5 stays a separate later change | Asked — user chose the recommended option; C5 rebases on this | S:90 R:75 A:90 D:90 |
| 5 | Confident | "Empty" = `null`, `""`, `[]`, `{}` — all fall through; `false`/`0` are real values and never skipped | Backlog says "EMPTY/NULL LEAVES SKIPPED"; bool/int overrides must survive or `reap_done: false` breaks | S:65 R:80 A:75 D:65 |
| 6 | Confident | Point-of-use default fallback seams retained as redundant safety; wholesale collapse stays C5 item 1 | Plan doc explicitly deferred the collapse to its own decision; materialized defaults make them inert, not wrong | S:55 R:80 A:70 D:60 |
| 7 | Confident | No migration file — read-semantics change only, no persisted-data restructure; precedence flip carried by docs/release notes | Constitution migration rule covers data restructuring; nothing on disk changes shape | S:70 R:60 A:80 D:75 |
| 8 | Confident | Exact full-stack `--origin` rendering (line format, winner/shadowed markers) is plan latitude; the contract is one line per defining tier, winner marked, no new flag, bare all-keys `--origin` stays winner-only | Backlog fixes the contract, not the format; trivially reversible | S:40 R:85 A:55 D:45 |
| 9 | Confident | `set` shadow warning fires for both targets (project set shadowed by system/env; `--system` set shadowed by env), exit 0, names the shadowing tier | Backlog item 2 is tier-general ("a higher tier shadows it"); warning-not-error since the write itself is valid | S:70 R:85 A:75 D:70 |
| 10 | Confident | Bare all-keys `show` (no `--origin`) keeps NOT materializing defaults in its output | Shipped deliberate behavior; backlog silent on it → preserve; the four-tier model surfaces via keyed show and `--origin` | S:45 R:80 A:60 D:55 |
| 11 | Confident | `set <key> ""` refused (not warn-and-write), pointing at `unset` | Empty leaves are defined to fall through — writing one is a pure footgun; refusal matches set's existing strict input contract | S:40 R:85 A:65 D:50 |
| 12 | Confident | `dispatch.reap_done` `*bool` → plain `bool` lands here (enabled by the materialized-defaults merge) | The merged map always carries the leaf; explicit `false` survives empty-skip; apply may defer if any pre-merge unmarshal path remains | S:45 R:75 A:65 D:55 |

12 assumptions (4 certain, 8 confident, 0 tentative, 0 unresolved).
