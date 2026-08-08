# Intake: Per-Session Selection — Env Override Layer + Launch Flags

**Change**: 260808-2d1w-env-override-layer-launch-flags
**Created**: 2026-08-08

## Origin

> Implement Change 1 (per-session selection: the env override layer + launch flags) of
> `fab/plans/sahil/config-overhaul.md` — read that full section plus the Orchestration and
> Resolved Decisions sections for scope, decisions, obligations, and rejected alternatives.
> Changes 2, 3, and 6 are running in parallel in sibling worktrees off the same base commit;
> this change has zero dependency on them per the plan's dependency map, so proceed
> independently. Shared-surface files (`_cli-fab.md` and its SPEC mirror,
> `docs/specs/config.md`, `_shared/configuration.md`) will also be touched by the other
> changes in different sections — stay strictly within this change's declared scope in those
> files and don't try to pre-empt or merge the others' work.

One-shot `/fab-new` invocation dispatched by the operator off backlog `[x3cf]`. All design
decisions were resolved in the 2026-08-08 `/fab-discuss` sessions and recorded in
`fab/plans/sahil/config-overhaul.md` (§ Change 1, § Orchestration, § Resolved decisions
items 11–13); that plan doc is the authoritative design source for this intake.

**Parallel-worktree discipline** (operator instruction, binding on apply/review): Changes 2
(verb surface), 3 (dispatch.mode), and 6 (FAB_KIT_PATH) run concurrently in sibling
worktrees. On the shared files — `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md`,
`docs/specs/config.md`, `docs/memory/_shared/configuration.md` — touch ONLY this change's
sections (new env-layer/flag content and the cascade section). Do NOT rename
`session_command`/`dispatch_command` (Change 4), do NOT touch `dispatch.watchable` semantics,
`resolve_agent.go` emission logic, `defaults.yaml`, or `internal/configupgrade` (Changes 2/3/5),
and do NOT document `FAB_DISPATCH_MODE` or `FAB_KIT_PATH` (Changes 3/6 own those).

## Why

1. **The pain**: provider *selection* is per-session intent with no home in the cascade.
   The motivating use case: compare the code two worker providers generate (e.g. kimi3 vs
   codex) by running two parallel worktree sessions in the same project — session A
   dispatches kimi3 workers, session B codex workers, same intake, same pipeline. Provider
   *definition* (a `providers.kimi3` block) is static machine config and already has a home
   (`~/.fab-kit/config.yaml`); selection (`agent.workers`) today bottoms out at per-checkout
   granularity.
2. **The consequence of not fixing it**: the workaround people reach for — editing
   `agent.workers` in `fab/project/config.yaml` per worktree, uncommitted — is a trap:
   `/git-pr`'s `git add -u` sweeps the personal preference into the shipped commit. The
   existing invocation override (`fab resolve-agent --provider`) cannot close the gap: it
   binds the native Agent-tool arm only, because `fab dispatch start` re-resolves from
   config and takes no override flags — so for a cross-provider worker the config edit is
   currently the sole executable path.
3. **Why env over the alternatives**: per-process-tree is per-session *by construction*.
   The harness session's shell calls inherit the variable, so `fab resolve-agent` AND
   `fab dispatch start`'s internal re-resolution read the same environment — the two seams
   cannot disagree, which the invocation-flag override could never guarantee (both are
   verified to load config through `config.Load` → `LoadPath`: `cmd/fab/resolve_agent.go:103`,
   `cmd/fab/dispatch_start.go:220`). Rejected alternatives (plan-recorded, user-confirmed):
   gitignored `config.local.yaml` (fourth file/layer, new plumbing, more "which file wins"
   surface); direnv/.envrc as the *documented* pattern (composes naturally but deliberately
   not built around); per-change stored preference (writes session intent into committed
   artifacts); PR-meta worker-provenance stamp (not adopted here).

Deliberately first in the plan: zero dependencies on Changes 2–6, and it delivers the
comparison use case the moment it ships.

## What Changes

### 1. The env override layer at `internal/config.LoadPath` (generic, scope-gated)

New **top** cascade layer: **env > project > system > built-in defaults**.

- **Generic mapping over the registry**: one env var per `internal/configref` registry row —
  dotted key uppercased, dots → underscores, `FAB_` prefix. The full mapping over today's
  17 registry rows:

  | Registry row (scope) | Env var | Honored? |
  |---|---|---|
  | `agent.session` (both) | `FAB_AGENT_SESSION` | yes |
  | `agent.workers` (both) | `FAB_AGENT_WORKERS` | yes |
  | `agent.profiles` (both) | `FAB_AGENT_PROFILES` | yes |
  | `providers` (both) | `FAB_PROVIDERS` | yes |
  | `dispatch.watchable` (both) | `FAB_DISPATCH_WATCHABLE` | yes |
  | `dispatch.column_width` (both) | `FAB_DISPATCH_COLUMN_WIDTH` | yes |
  | `dispatch.reap_done` (both) | `FAB_DISPATCH_REAP_DONE` | yes |
  | `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `consolidate.detectors`, `stage_hooks`, `branch_prefix` (project) | `FAB_PROJECT_NAME` etc. | **no — warn + ignore** |

- **Forward walk only**: resolution iterates the registry rows, computes each row's env
  name, and probes the environment (`os.LookupEnv`). It never reverse-parses env names, so
  underscore-vs-dot ambiguity (`dispatch.column_width` → `FAB_DISPATCH_COLUMN_WIDTH`) cannot
  arise. Rows added by later changes (e.g. Change 3's `dispatch.mode`) become env-eligible
  automatically — no per-key work ever recurs.
- **Import-cycle constraint (load-bearing for implementation)**: `internal/config` CANNOT
  import `internal/configref` (configref → internal/agent → internal/config closes a cycle —
  the exact cycle that forced `internal/configscope` into existence as a leaf package).
  The env resolution inside `LoadPath` therefore needs the dotted row-key enumeration from a
  leaf source: extend `internal/configscope` to enumerate the dotted registry keys (it
  already single-sources the per-top-level-key scope taxonomy consumed by both packages),
  and add a parity lint in configref's tests asserting registry keys ⊆/≡ the leaf
  enumeration so the two cannot drift.
- **Values parsed as YAML** (scalar/flow parsing), so a map-valued row can be set whole:
  `FAB_AGENT_PROFILES='{review: {provider: codex}}'`. The parsing helper is introduced here
  and **reused verbatim by Change 2's `set` verb** (the plan's recorded C1↔C2 soft coupling:
  whichever merges second rebases onto the helper — mechanical).
- **Merge semantics**: each honored env var contributes its value nested under the row's
  dotted path (`FAB_AGENT_WORKERS=codex` → `{agent: {workers: codex}}`); the env map merges
  like any other layer via the existing `deepMerge` (maps per-key, lists replace, scalars
  replace): `effective = deepMerge(deepMerge(system, project), env)`.
- **Scope-gated**: only rows with `scope ∈ {both, system}` are honored. A project-scoped
  env var (e.g. `FAB_SOURCE_PATHS`) is ignored with a `fab: warning:` on stderr — the exact
  mirror of the system-file pruning rule (`pruneProjectScoped`), preserving repo
  reproducibility. Unknown `FAB_*` vars in the environment are never scanned at all (forward
  walk — nothing to warn about).
- **Fail-open**: an unparseable env value warns (`fab: warning:`) and is skipped — config
  must never brick (the malformed-system-file precedent, same `warnw` seam).
- **Reach**: the layer lands inside `LoadPath`, so every consumer gets it with zero
  per-caller change — `Load`, `fab agent --repo` (path-based `LoadPath`), preflight, impact,
  status, resolve-agent, dispatch start, operator, batch, spawn, prmeta.

### 2. `fab config show` visibility — the `env` origin

- `config.Layers` (produced by `LoadLayers`) gains the env layer: the env overlay map plus
  per-key provenance of which variable set it. `LoadLayers` runs the same cascade as
  `LoadPath` (its documented no-drift contract), so `Layers.Effective` now includes env —
  plain `fab config show` therefore prints the env-affected effective config (still without
  materializing built-in defaults).
- `fab config show --origin` prints an `env` origin for leaves the env layer set, naming
  the variable (e.g. `agent.workers = codex  # $FAB_AGENT_WORKERS`) — the origin column
  carries the variable name where file-layer leaves carry the file path. Non-negotiable per
  plan: three layers without provenance was archaeology; four without it would be worse.
- `cmd/fab/config.go`'s `renderShowOrigin`/`flattenOrigin` extend from three sources
  (default < system < project) to four (default < system < project < env).

### 3. Launch-flag sugar — `--workers <provider>` on the session-spawning commands

**Pure sugar over the env layer**: no new resolution path, no persisted state. The flag
exports `FAB_AGENT_WORKERS=<provider>` into the spawned session's process environment so
the comparison flow is one argument per session:

```sh
wt create …   # worktree A
fab agent --workers kimi3
wt create …   # worktree B
fab agent --workers codex
```

Per spawn seam (all four verified in source):

- **`fab agent`** (`cmd/fab/agent.go` — execs via `syscall.Exec("/bin/sh", …, os.Environ())`):
  append `FAB_AGENT_WORKERS=<p>` to the exec'd environment. The flag does NOT change the
  composed session command itself — `fab agent` composes from the *session*-depth roles;
  `--workers` targets what the spawned session's own `fab resolve-agent`/`fab dispatch`
  calls will resolve for Tier-2 workers.
- **`fab batch new` / `fab batch switch`** (`cmd/fab/batch_new.go:139`,
  `cmd/fab/batch_switch.go:140` — spawn via `tmux new-window … shellCmd`): prefix the
  composed shell command with the shell-quoted assignment, e.g.
  `FAB_AGENT_WORKERS='kimi3' <spawnCmd> '/fab-new …'` (portable — avoids a tmux-version
  dependency on `new-window -e`).
- **Operator spawn path** (`cmd/fab/operator.go:91` — same tmux new-window shape):
  `fab operator --workers <p>` prefixes the operator's shellCmd identically.

Flag properties: value passes through verbatim — no validation against known provider names
(provider names are opaque; document-don't-validate, Constitution Principle I). An unknown
provider surfaces downstream at resolve time via the existing unknown-provider error. No
`--session` flag counterpart (plan advertises `FAB_AGENT_SESSION` as a plain env var; the
flag surface stays minimal — decision 12).

### 4. Documentation & advertising

Docs **advertise the handful** — `FAB_AGENT_WORKERS`, `FAB_AGENT_SESSION` — while stating
the mechanism is generic (every `both`/`system`-scoped registry row). `FAB_DISPATCH_MODE`
is mentioned NOWHERE until Change 3 ships it.

- `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md`: `--workers` on the
  `fab agent` / `fab batch new` / `fab batch switch` / `fab operator` entries; the env
  override contract (mapping rule, scope gating, YAML values, fail-open) documented once —
  most naturally alongside `fab config show`'s cascade description; the `--origin` env
  origin. Constitution: CLI changes MUST update `_cli-fab.md`; skill changes MUST update
  the SPEC mirror.
- `docs/specs/config.md` § Override cascade: three → four layers; the env-mapping contract;
  `show`'s env origin appended to § Visibility commands.
- Memory `docs/memory/_shared/configuration.md` § Override Cascade & Scope Enforcement and
  § Visibility Commands; `docs/memory/runtime/providers-and-profiles.md` (per-session
  selection requirement + `fab agent` flag surface). Memory updates happen at hydrate, per
  pipeline; listed here for scope.
- Naming caveat worth one doc sentence: the fab-kit binary's pre-existing `FAB_AGENTS`
  (agents-list override in `src/go/fab-kit/internal/skills.go`) is unrelated to
  `FAB_AGENT_*` config vars — don't conflate them in docs or error text.

### 5. Tests (obligations, from the plan)

- Loader: env-beats-project-beats-system precedence; scope gating (project-scoped var
  warned + ignored); YAML flow parse of a map-valued var (`FAB_AGENT_PROFILES`); merge
  semantics (map per-key merge with file layers); unparseable value warn + skip; empty-var
  handling; no-env byte-stable behavior (zero vars set ⇒ identical to today).
- `show --origin`: env origin line naming the variable; plain `show` including env values.
- Spawn seams: `fab agent --workers` env injection (via `--print`-adjacent or seam test);
  batch new/switch shellCmd prefix; operator shellCmd prefix; quoting of the flag value.
- Parity lint: configscope's dotted-key enumeration ≡ configref registry keys.

**No migration** — net-new, opt-in (no persisted state, no restructure of user data).

## Affected Memory

- `_shared/configuration`: (modify) § Override Cascade & Scope Enforcement gains the env
  layer (fourth layer, mapping rule, scope gating, fail-open); § Visibility Commands gains
  the `env` origin
- `runtime/providers-and-profiles`: (modify) per-session provider selection via
  `FAB_AGENT_WORKERS`/`FAB_AGENT_SESSION`; `fab agent`/`fab batch`/`fab operator`
  `--workers` flag surface

## Impact

- **Go — fab binary** (`src/go/fab/`): `internal/config/config.go` (LoadPath env layer,
  LoadLayers env exposure, YAML value-parse helper) + `config_test.go`;
  `internal/configscope/configscope.go` (dotted row-key enumeration) + tests;
  `internal/configref/configref.go` (parity lint only — no row changes);
  `cmd/fab/config.go` (`show`/`--origin` four-source composition) + tests;
  `cmd/fab/agent.go`, `cmd/fab/batch.go`/`batch_new.go`/`batch_switch.go`,
  `cmd/fab/operator.go` (the `--workers` flag + env injection) + tests.
- **Kit** (`src/kit/skills/_cli-fab.md`) and SPEC mirror (`docs/specs/skills/SPEC-_cli-fab.md`).
- **Specs**: `docs/specs/config.md` (cascade + visibility sections only).
- **Memory** (hydrate): the two files above.
- **Not touched** (parallel-change discipline): `defaults.yaml`, `resolve_agent.go`,
  `internal/dispatch`, `internal/configupgrade`, `cmd/fab/config.go` init/upgrade paths,
  provider field names, `_preamble.md`.
- The fab-kit and fab-go binaries are untouched.

## Open Questions

- None — all design decisions were user-confirmed in the 2026-08-08 discussion and recorded
  in the plan doc; residual implementation choices are graded below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly plan Change 1: env layer + `show --origin` env origin + `--workers` launch flags; nothing from Changes 2–6 | Plan § Change 1 + operator instruction are explicit; dependency map confirms zero coupling | S:95 R:90 A:95 D:95 |
| 2 | Certain | Cascade order env > project > system > built-ins, merged per-field at `LoadPath` via the existing `deepMerge` (maps per-key, lists replace, scalars replace) | Plan-explicit; matches the landed Change-2 (lpb5) merge semantics verified in `internal/config/config.go` | S:90 R:85 A:95 D:90 |
| 3 | Certain | Generic forward registry walk (`FAB_` + uppercase + dots→underscores); scope-gate to `both`/`system`; project-scoped vars warn + ignore; unparseable values warn + skip (fail-open) | Plan-explicit including the never-reverse-parse rule and both warning behaviors | S:90 R:80 A:90 D:90 |
| 4 | Confident | Resolve the import cycle by extending leaf `internal/configscope` with the dotted registry-key enumeration + a configref parity lint; `internal/config` never imports configref | Plan specifies "walks the registry" but not the mechanics; configscope exists precisely for this cycle (its package doc says so), so extending the leaf follows the established single-sourcing pattern | S:70 R:75 A:85 D:70 |
| 5 | Confident | The YAML scalar/flow value-parse helper lives exported in `internal/config`, positioned for Change 2's `set` to reuse | Plan records the C1↔C2 helper coupling but not the package; `internal/config` is where the env layer consumes it and C2's cmd layer can import it cycle-free | S:60 R:85 A:75 D:65 |
| 6 | Confident | A set-but-empty env var (`FAB_AGENT_WORKERS=`) is treated as unset (skipped, no warning), not parsed as YAML null | Shell convention reads empty ≈ unset; parsing "" to null would make env-empty *override* a project value to nil — surprising; easily revisited | S:50 R:85 A:65 D:55 |
| 7 | Certain | `Layers.Effective` includes the env layer, so plain `fab config show` reflects env overrides (defaults still not materialized) | `LoadLayers`' documented contract is "runs the SAME cascade LoadPath runs … so `show` cannot drift from what consumers actually see" | S:75 R:80 A:90 D:85 |
| 8 | Confident | `--origin` env origin renders as the variable name (e.g. `# $FAB_AGENT_WORKERS`) in the origin column, parallel to file-path origins | Plan mandates "an `env` origin (naming the variable)"; exact rendering is presentational and cheap to adjust in review | S:55 R:90 A:70 D:55 |
| 9 | Confident | Env injection mechanics: `fab agent` appends to the `syscall.Exec` environment; batch new/switch and operator prefix the tmux shellCmd with a shell-quoted `FAB_AGENT_WORKERS='<p>'` assignment (no `tmux new-window -e` dependency) | Seams verified in source; assignment-prefix is portable across tmux versions and matches the existing single-quote escaping pattern in those files | S:65 R:85 A:80 D:70 |
| 10 | Certain | `--workers` value is pass-through — no validation against provider names; unknown providers fail downstream at resolve time with the existing error | Document-don't-validate is Constitution Principle I and restated on every provider surface in `internal/config`/`internal/agent` | S:85 R:85 A:95 D:90 |
| 11 | Certain | Flag surface is `--workers` only (no `--session` flag); `fab operator` is included as the fourth spawn seam | Plan decision 12 + § launch-flag sugar name exactly these commands: `fab agent`, `fab batch new`, `fab batch switch`, and the operator spawn path | S:85 R:85 A:85 D:80 |
| 12 | Certain | Docs advertise `FAB_AGENT_WORKERS` + `FAB_AGENT_SESSION` only; `FAB_DISPATCH_MODE` and `FAB_KIT_PATH` are not mentioned (Changes 3/6 own them); no migration ships | Plan-explicit (advertise-the-handful; "No migration — net-new, opt-in"); operator instruction forbids pre-empting parallel changes | S:90 R:85 A:90 D:90 |

12 assumptions (7 certain, 5 confident, 0 tentative, 0 unresolved).
