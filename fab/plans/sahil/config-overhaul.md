# Config management overhaul — per-session env layer, verbs, dispatch modes, field names, source consolidation

> Backlog detail doc — written 2026-08-08 after a `/fab-discuss` session (the env-layer
> change added later the same day from a follow-on discussion; Change 6 folded in from
> backlog `[kpth]` the same day). All open decisions below are **resolved** (user-confirmed
> in that session). The work is split into six changes, each independently shippable, in
> dependency order — intended to be picked up by other agents via `/fab-new`.
>
> **Reordered 2026-08-08 (user-confirmed)**: the env override layer runs **first** — it has
> zero dependencies on the rest of the plan and delivers runtime provider switching (the
> kimi3-vs-codex comparison use case) immediately; the other four changes shift down one.
>
> **Supersedes change `260807-k0v3-consolidate-config-surface`**: its intake and backlog
> entry are deleted; its implementation branch (worktree nimble-ravine, apply done through
> 5 review-rework cycles, never merged) is **discarded as stale** — the design survives,
> re-derived fresh in Change 2, and the review-cycle findings survive as day-one test cases
> (see Change 2 § Lessons from the discarded branch).

## Goal

Six related cleanups to how fab-kit's configuration is inspected, mutated, and resolved:

1. **Per-session selection** — a scope-gated environment override layer
   (env > project > system > defaults) plus launch-flag sugar, so two parallel sessions in
   one project can dispatch different worker providers.
2. **Verb surface** — `fab config` becomes six intent-grouped verbs (`show`/`explain`/`set`/
   `unset`/`init`/`upgrade`), with surgical fence-engine writes replacing hand-editing.
3. **Dispatch-mode semantics** — how a stage worker is launched (native sub-agent / tmux
   pane / headless CLI) becomes an explicit, impossible-state-free preference knob
   (`dispatch.mode`), delinked from the presence/absence of provider command fields.
4. **Field naming** — `session_command`/`dispatch_command` renamed to the honest axis:
   `interactive_command`/`headless_command`.
5. **Source consolidation** — one embedded values file, zero stub copies, a `--check` drift
   mode, and a fence that teaches which file each setting belongs in.
6. **Kit-dev resolution override** — a per-process `FAB_KIT_PATH` honored at the
   kit-resolution seam (shim + `kit-path`), so a kit-dev worktree can run its own unshipped
   `src/kit/`. The env-override *pattern* of Change 1 applied to infrastructure resolution —
   deliberately not a config field.

## The source map (current state, for orientation)

**Read path** — effective config resolves across three layers at `internal/config.LoadPath`:
project `fab/project/config.yaml` > system `~/.fab-kit/config.yaml` (scope-pruned) >
built-ins. The built-in layer has **two** value sources: `internal/agent`'s embedded
`defaults.yaml` (agent knobs + the whole `providers:` table, applied at point-of-use seams)
and Go constants in `internal/config` (the `dispatch.*` defaults). Two metadata sources
shape everything: the `internal/configref` registry (per-field schema + rendered segments)
and the `internal/configscope` taxonomy (leaf package; cycle-forced).

**Write path** — `fab config init --project` (registry-generated: InitSeed fields live +
managed fence), `fab config upgrade` (the single writer; fence regeneration, parking,
hoisting), `fab config init --system` (registry filtered to `scope ∈ {system, both}`, all
commented), plus `src/kit/scaffold/` for the non-config project files and one **embedded
stub config.yaml in the fab-kit binary** (the skew fallback — the last true second copy).

## Orchestration — parallelism & dependency map

> For the operator dispatching these changes as parallel worktree agents (one change =
> one worktree = one full pipeline run off backlog `[x3cf]`).

```
t=0 (parallel):  C1 env layer    C2 verbs    C3 mode ladder    C6 FAB_KIT_PATH
                      │              │             │
                      │              │             ▼
                      │              │        C4 renames        (strictly after C3)
                      │              │             │
                      └──────────────┴──────┬──────┘
                                            ▼
                                     C5 consolidation           (last)
```

**Hard edges** (semantic — never parallelize across them):

- **C3 → C4**: both rewrite the same provider blocks (`defaults.yaml` comments, the same
  spec/memory sections); the second to land owns the sweep of the first's text, so running
  them concurrently guarantees a semantic merge.
- **C2 → C5 and C3 → C5**: C5's fence pointers render `fab config set --system` (C2's
  verb), its item 1 folds `dispatch.mode: native` into `defaults.yaml` (C3's knob), and it
  extends the same `internal/configupgrade` engine + `cmd/fab/config.go` that C2 reworks.
- **C4 → C5** (recommended, not hard): landing C5 first forces C4 to re-sweep C5's new
  fence/defaults text — wasted rework, not wrongness.

**Fully parallel at t=0**: C1, C2, C3, C6 — no semantic coupling. Two soft couplings to
expect as *mechanical* rebases (operator stacked-merge rule: mechanical rebase OK,
semantic conflict hands back):

- **C1 ↔ C2** share the YAML value-parsing helper (C1 introduces it; C2's `set` reuses
  it). Whichever merges second rebases onto the helper — small and mechanical.
- **C1/C2/C3** all edit `_cli-fab.md` (+ SPEC mirror), `docs/specs/config.md`, and
  `_shared/configuration.md` — in *different sections*; textual adjacency only.

**C6 is edge-free**: different seam (shim + kit-path resolution), mostly a different
binary; only trivial `_cli-fab.md` / doctor-output adjacency. Slot it wherever a pane is
free.

**Conflict-surface table** (what overlaps where):

| Shared surface | Touched by | Nature |
|---|---|---|
| `_cli-fab.md` + SPEC mirror | C1 C2 C3 C4 C6 | different sections — mechanical |
| `docs/specs/config.md` | C1 C2 C3 C5 | different sections — mechanical |
| `_shared/configuration.md` | C1 C2 C3 C4 C5 | different sections — mechanical |
| `defaults.yaml` | C3 C4 C5 | semantic — serialized by C3 → C4 → C5 |
| `internal/configupgrade`, `cmd/fab/config.go` | C2 C5 | semantic — C5 after C2 |
| `resolve_agent.go`, `internal/dispatch` | C3 | exclusive |
| shim + kit-path resolution | C6 | exclusive |

**Critical path**: C3 → C4 → C5 — start C3 at t=0 regardless of how many other panes run.
**Conservative waves** (fewer panes): A = {C1, C2, C6} → B = {C3} → C = {C4} → D = {C5}.
**Single-agent fallback**: plan order 1 → 2 → 3 → 4 → 5, with 6 slotted anywhere.

---

## Change 1 — per-session selection: the env override layer + launch flags

> Motivating use case (follow-on `/fab-discuss`, 2026-08-08): compare the code two worker
> providers generate (e.g. kimi3 vs codex) by running two parallel worktree+agent sessions
> in the same project — session A dispatches kimi3 workers, session B codex workers.
> Provider **definition** (a `providers.kimi3` block — grammar + fills) is static config and
> already has a home: `~/.fab-kit/config.yaml`, once, machine-wide. Provider **selection**
> (`agent.workers`) is per-session intent, and no cascade layer holds it today — the
> invocation override (`fab resolve-agent --provider`) binds the native arm only
> (`fab dispatch start` re-resolves from config), so for a cross-provider worker the config
> override is currently the sole executable path. The workaround people will reach for —
> editing `agent.workers` per worktree, uncommitted — is a trap: `/git-pr`'s `git add -u`
> sweeps the preference into the shipped commit.
>
> **Deliberately first in the plan**: this change has zero dependencies on the other four
> and delivers the comparison use case the moment it ships.

### The env override layer (generic, scope-gated)

- **New top cascade layer** at `internal/config.LoadPath`:
  **env > project > system > built-in defaults**.
- **Generic mapping over the registry**: one env var per registry row — dotted key
  uppercased, dots → underscores, `FAB_` prefix (`agent.workers` → `FAB_AGENT_WORKERS`,
  `dispatch.watchable` → `FAB_DISPATCH_WATCHABLE`). Resolution walks the registry
  **forward** (compute each row's env name, probe the environment) — it never
  reverse-parses env names, so underscore-vs-dot ambiguity (`dispatch.column_width`)
  cannot arise. Rows added by later changes (e.g. Change 3's `dispatch.mode`) become
  env-eligible automatically — the mapping is generic, so no per-key work ever recurs.
- **Values parsed as YAML** (scalar/flow parsing — introduced here, and reused verbatim by
  Change 2's `set` verb), so a map-valued row can be set whole:
  `FAB_AGENT_PROFILES='{review: {provider: codex}}'`. The env layer merges per-field like
  any other layer (maps per-key, lists replace, scalars replace).
- **Scope-gated**: only `scope ∈ {both, system}` rows are honored. A project-scoped env
  var (e.g. `FAB_SOURCE_PATHS`) is ignored with a `fab: warning:` — the exact mirror of
  the system-file pruning rule, preserving repo reproducibility. An unparseable env value
  likewise warns and is skipped (fail-open — config must never brick, the malformed-
  system-file precedent).
- **Why env closes the cross-provider gap structurally**: per-process-tree is per-session
  by construction. The harness session's shell calls inherit the variable, so
  `fab resolve-agent` AND `fab dispatch start`'s internal re-resolution read the same
  environment — the two seams cannot disagree, which is exactly what the invocation-flag
  override could never guarantee.
- **Visibility**: `fab config show --origin` gains an `env` origin (naming the variable).
  Non-negotiable — three layers without provenance was archaeology; four without it would
  be worse.

### Launch-flag sugar

- `--workers <provider>` on the session-spawning commands — `fab agent`, `fab batch new`,
  `fab batch switch`, and the operator spawn path — exporting `FAB_AGENT_WORKERS` into the
  spawned session's process environment. **Pure sugar over the env layer**: no new
  resolution path, no persisted state; the flag exists so the comparison flow is one
  argument per session.
- Docs **advertise the handful** (`FAB_AGENT_WORKERS`, `FAB_AGENT_SESSION`, and — once
  Change 3 ships it — `FAB_DISPATCH_MODE`) while the mechanism stays generic (every
  `both`/`system`-scoped registry row).

The comparison flow this enables:

```sh
# once, machine-wide: define the provider (grammar + per-role fills)
#   providers.kimi3: {session_command: …, dispatch_command: …, profiles: …}
#   (fields renamed interactive_command/headless_command by Change 4)
# then, two sessions:
wt create …  # worktree A
fab agent --workers kimi3
wt create …  # worktree B
fab agent --workers codex
```

Same intake, same pipeline, different Tier-2 workers; nothing written to any config file.

### Rejected alternatives (recorded)

- **Gitignored local config file** (`config.local.yaml`, the git `--worktree`-config
  precedent) — a fourth file, a fourth layer, new `set`/`unset` plumbing, and new
  "which file is winning" surface; everything Change 5 works to reduce. Env covers the
  need without a file.
- **direnv/.envrc worktree persistence** — composes with the env layer by nature, but
  deliberately **not** built around or documented as the pattern (user decision).
- **Per-change stored preference** — writes session preference into committed artifacts.
- **PR-meta worker-provenance stamp** — discussed, not adopted in this change.

**Obligations**: loader + tests (env layer, scope gating, YAML parse, fail-open);
`show --origin` env origin + tests; spawn-seam env injection for the flags
(`fab agent`/`fab batch`/operator) + tests; `_cli-fab.md` + SPEC mirror (new flags + the
env contract); `docs/specs/config.md` § cascade (three → four layers); memory
`_shared/configuration.md` § Override Cascade, `runtime/providers-and-profiles.md`.
**No migration** (net-new, opt-in).

---

## Change 2 — the six-verb `fab config` surface

> Re-derivation of the k0v3 design with one rename (`docs` → `explain`). Fresh
> implementation — the discarded branch is not salvaged. Surveyed against git 2.46+
> (`get/set/unset/list/edit`), npm (`get/set/delete/list/edit/fix`), gh, gcloud, kubectl:
> the `get/set/unset/list` core is covered; `explain` follows the `kubectl explain`
> precedent for per-key schema docs, which most tools lack entirely.

Target surface (six verbs, three cobra command groups):

```
# Inspect — "what IS"
fab config show [<key>] [--origin]        # effective post-cascade config; with <key>, just that value

# Inspect — "what CAN be"
fab config explain [<key>] [--json]       # registry documentation (rename of `reference`); with <key>, that field's block

# Modify — surgical, fence-aware
fab config set <key> <value> [--system]   # write one field through the configupgrade engine
fab config unset <key> [--system]         # remove an override, restoring inheritance/default

# Lifecycle — whole-file
fab config init [--system]                # bootstrap; bare = project mode (--project flag retained)
fab config upgrade                        # mechanical reconcile against the registry (unchanged)
```

Carried-forward design detail (all user-confirmed in the original k0v3 session):

- **`show <dotted.key>`** prints the effective value with built-in defaults composed (unlike
  bare `show`, which deliberately doesn't materialize defaults — the single-key form answers
  "what is in effect"). Scalar/list leaves print raw (scriptable, no `key =` prefix);
  map-valued keys print the YAML subtree. `--origin` works at both depths. Unknown key ⇒
  non-zero exit naming the key.
- **`explain [<key>]`** — bare form prints exactly what `fab config reference` prints today;
  with `<key>`, only that field's rendered segment (keys documented inside another field's
  segment resolve to the owning segment). `--json` bare = full field table; with `<key>` =
  matching row(s). **`reference` is retained as an invisible cobra alias** — it protects the
  `# Full reference of all available options: fab config reference` pointer comments seeded
  by migration 2.9.2-to-2.10.0 and keeps historical migration texts functional. All current
  kit/doc text migrates to `explain`.
- **`set`** writes through the `internal/configupgrade` fence engine — never a YAML marshal
  round-trip (the yogn/#473 comment-mashing class is the whole motivation). Live field ⇒
  splice; fence-only field ⇒ materialize live above the fence (presence=intent: setting
  pins). Refuses unknown keys (suggests `fab config explain`); scope-enforced via
  `internal/configscope` (this + unknown-key refusal delivers the typo-linting the reserved
  `validate` slot was held for — **`validate` is dropped**). Deep dotted keys into
  map-valued fields supported (`set agent.profiles.review.model …`,
  `set providers.codex.profiles.default.model …`). `<value>` parsed as a YAML scalar, lists
  in flow syntax — the same value parsing Change 1's env layer introduced. `--system`
  creates `~/.fab-kit/config.yaml` (scaffold header) when missing — git-config precedent.
- **`unset`** removes the live override through the same engine; the fence regains the
  commented reference line. Restores inheritance (system value, else built-in) — semantic
  under presence=intent. Unsetting an unset key = exit-0 no-op with notice (idempotent).
  Deliberately ungated on value type (the repair path).
- **`init`** — bare becomes project mode (today a usage error; error→behavior is the safe
  breaking direction). `--project` retained (fab-kit's `fab init` shells out to it);
  `--system` unchanged; both together still an error; overwrite refusal unchanged.
- **No `edit` verb** ($EDITOR sugar over a fence-managed file invites mangling; `set`/`unset`
  is the nudge). **No `fix`/`doctor` verb** (`upgrade` already hoists/parks/regenerates/
  refuses-on-unparseable — extend it, don't duplicate it; see Change 5's `--check`).

### Lessons from the discarded branch (day-one test cases)

The k0v3 implementation went through 5 review-rework cycles; every cycle found a real
"silent-wrong-at-exit-0" escape in the surgical writers. The fresh implementation writes
these as regression tests **before** the writer code:

- block-form orphaning (setting a leaf must not orphan its map block's siblings)
- duplicated renderer between set-materialize and the fence renderer (single-source the segment)
- key-axis typos (a valid-looking dotted path that matches no registry row must refuse)
- value-type holes (a map value where a scalar belongs — the expected-kind registry signal)
- quote holes (leading-quote-only comment scanner; quoted-key refusal)
- comment preservation end-to-end (the yogn class: user comments survive every set/unset)

**Obligations**: CLI change ⇒ `_cli-fab.md` § fab config rewrite + SPEC mirror + tests;
`docs/specs/config.md` surface section + set/unset-as-surgical-upgrade framing +
validate-slot removal; `docs/specs/index.md` row; kit-text sweep `reference` → `explain`
(fab-setup.md + SPEC mirror, `src/go/fab/cmd/fab/skill.md`, `docs/site/skill.md`); memory
sweep (`_shared/configuration.md` primary, `distribution/kit-architecture.md` incl. the
stale `cobra.NoArgs` claim and the "leaves room for a future validate" sentence,
`runtime/providers-and-profiles.md`, `distribution/migrations.md` — the quoted seeded
comment stays VERBATIM); historical migration files left verbatim (the alias keeps them
functional); **shll standards check** before finalizing naming/help text (constitution
§ Toolkit Standards).

---

## Change 3 — `dispatch.mode`: the descent ladder + capability delink

> The semantic heart. Today the dispatch mode is *f*(dispatch_command presence,
> `dispatch.watchable`, `$TMUX`) — capability data doubles as mode policy
> (`resolve_agent.go` `dispatchLineFor`, `fab dispatch start`'s auto ladder). Consequences:
> claude can never ship its headless grammar as data (presence would flip everyone's
> out-of-tmux dispatches to headless CLI), and a user adding it themselves silently changes
> mode for every dispatch. This change makes provider fields **pure capability grammar** and
> mode an **explicit preference**.

### Provider fields become pure grammar

- **claude ships a `dispatch_command`** (headless grammar exists: `claude -p`, prompt on
  stdin — exact flags to be settled at implementation, incl. `--dangerously-skip-permissions`
  and `{model}`/`{effort}` placement). After this change, presence means "here's how",
  never "do it".
- **claude gains an explicit native-capability flag** (e.g. `native: true` on its provider
  block in `defaults.yaml`). Native dispatch (the Agent tool) only exists for claude-family
  models, and provider names are opaque — fab never infers from names — so the capability
  must be shipped data, not a presence signal or a name match. Overridable like every other
  provider field.

### The knob: `dispatch.mode: pane | native | headless` (default `native`, scope `both`)

The setting is a **preference ceiling, not a command**. Per dispatch, resolution starts at
the preferred rung and **descends the fixed ladder `pane → native → headless`, never
ascending**: take the first rung that is *possible* given provider capability + environment;
surface the chosen rung and reason on the existing `dispatched …` line; error only when no
rung is possible (a custom provider with no commands at all).

Rung prerequisites: **pane** = tmux reachable ∧ the interactive command; **native** =
provider's native flag; **headless** = the headless command. A rung whose prerequisite is
missing is skipped, not errored — this is what kills the impossible states.

| `dispatch.mode` | claude worker | codex/gemini worker | outside tmux |
|---|---|---|---|
| `pane` ("let me watch") | pane; no tmux → native | pane; no tmux → headless | descends past pane |
| `native` (default, "quiet in-context") | native | headless (no native seam; notice says so) | same — tmux irrelevant |
| `headless` ("detached processes, always") | headless | headless | same |

Properties worth stating in the spec:

- **Never ascend** is the safety property: a descent is always *less* interactive, so an
  unattended run can never be surprised by a pane, and `native` never becomes a pane.
- Mixed setups just work: mode resolves per dispatch against the stage's resolved provider,
  so `agent.session=claude` + `agent.workers=codex` gives a claude session and headless
  codex stage workers under the default.
- The default `native` reproduces today's `watchable: false` behavior byte-for-byte for the
  built-ins (claude → native, codex/gemini → headless), so an unconfigured repo sees zero
  behavior change.
- The old "no fallback between the two command fields" doctrine is **replaced by the ladder
  between modes**: each mode still requires its own field — the ladder descends *modes*, and
  a mode whose field is missing is simply not possible. No field ever substitutes for another.
- `dispatch.mode` is `scope: both` — so via Change 1's generic mapping it is settable
  per-session as `FAB_DISPATCH_MODE`, with zero additional work in this change.

### What dissolves and what survives

- **`dispatch.watchable` dissolves.** Migration rewrites `watchable: true` → `mode: pane`
  in both scopes; absent/`false` → nothing (the `native` default). No read-time alias kept —
  the knob is young (260806-mnri) and machine-local; the migration closes it.
- **`resolve-agent`'s `dispatch=` contract is unchanged in shape**: line absent ⇔ resolved
  mode is native; line present carries the mode's command. The skills' branch-on-presence
  seam needs no restructuring; the emission matrix in `dispatchLineFor` is rewritten to read
  `(mode, capability, $TMUX)` instead of `(presence, watchable, $TMUX)`.
- **Per-invocation `--pane`/`--headless` on `fab dispatch start` survive** as one-shot
  overrides; `start`'s auto ladder re-derives from the same `(mode, capability, env)` inputs
  so the two seams cannot disagree.

**Obligations**: registry + configscope rows (add `dispatch.mode`, drop `dispatch.watchable`;
paired entries); migration file (user-data restructure — constitution); `defaults.yaml`
(claude `dispatch_command` + native flag) with its pinned-test updates; `resolve_agent.go` +
`internal/dispatch` mode selection + tests; `_cli-fab.md` + SPEC mirror; `_preamble.md`
§ CLI-Adapter Dispatch + watchable prose (and every skill restating it) + SPEC mirrors;
specs `harness-adapters.md`, `stage-models.md`, `config.md`; memory `_shared/configuration.md`
§ dispatch, `runtime/providers-and-profiles.md`, `runtime/dispatch.md`,
`_shared/context-loading.md`. This is the largest sweep class of the five changes — treat
the whole watchable/presence-signal mirror class as in-scope up front.

---

## Change 4 — rename `session_command`/`dispatch_command` → `interactive_command`/`headless_command`

> Deliberately **not** tier-aligned naming (session/workers): a Tier-2 pane worker runs the
> `session_command`, so tier names would encode a false invariant. The fields split by
> **interaction mode** — "launch an interactive session a human can watch/steer" vs "run
> headless, prompt on stdin, exit when done" — and the new names say exactly that.
> `headless_command` also breaks the word-collision between the field, the `fab dispatch`
> verb, and the `dispatch.*` config block.

- **Read-time alias** (the `agent.tiers` → `agent.profiles` precedent): `ProviderConfig`
  keeps the two deprecated fields; resolution prefers the new spelling per field, so a
  half-migrated config resolves everything during the window.
- **Migration** rewrites both scopes on disk. These are *nested* keys, so `renamed_from`'s
  top-level mechanical carry cannot fire — set the metadata anyway (`--json` consumers), as
  documented on the `agent.profiles` row.
- **Full kit-text sweep**: `defaults.yaml` comments, configref segments, `_cli-fab.md`,
  `_preamble.md`, every skill + SPEC mirror restating the field names, specs
  (`harness-adapters.md`, `stage-models.md`, `config.md`, `architecture.md`), memory
  (`_shared/configuration.md` § providers, `runtime/providers-and-profiles.md`,
  `runtime/dispatch.md`). Grep both old names repo-wide including user-facing string
  literals (the ioku lesson).
- **Ordering note**: Changes 3 and 4 touch the same provider blocks. Recommended order is
  3 → 4 (mode semantics first; the rename sweep then also renames Change 3's additions).
  If executed 4 → 3 instead, Change 3 lands directly on the new names — either works; the
  second to land owns the sweep of the first's text.

---

## Change 5 — source consolidation, `upgrade --check`, fence clarity

Rides along after the dust settles; each item is small and independently committable.

1. **One embedded values file.** Fold the `dispatch.*` Go-constant defaults
   (`DefaultDispatchColumnWidth`, `DefaultDispatchReapDone`, watchable's successor
   `dispatch.mode: native`) into `internal/agent`'s `defaults.yaml` so layer 3 has a single
   config-shaped value source. Registry rows interpolate as before. *(Stretch, explicitly
   optional and deferred to its own decision: merge `defaults.yaml` as a true layer 0 in
   `LoadPath`, collapsing the point-of-use fallback seams — the file was shaped for that
   future; do NOT fold this into the same change.)*
2. **Retire the fab-kit embedded stub config.yaml.** The skew window is long past (the
   `init --project` subcommand shipped in 2.15.x); the fallback becomes a clear error
   telling the user to upgrade fab-go. Zero second copies remain.
3. **`fab config upgrade --check`** — the drift mode the fence's kit-version stamp was
   designed for (deferred in the config-upgrade effort's Change 3, j0qm): exit non-zero
   when a run would change the file (stale fence stamp, unparked unknown keys, missing
   fence), zero writes. The CI/human probe for "is this repo's config clean?" — this is the
   cleanup tooling story, alongside Change 2's `set`/`unset` (surgical repair) and
   `show --origin` (find typo'd overrides).
4. **Fence teaches the file split.** Annotate each fence line/block with its scope
   (`[project|system]` — in the original config-upgrade worked example but never shipped),
   and have preference-class (`scope: both`) adverts carry a
   "settable machine-wide: `fab config set --system <key> <value>`" pointer instead of
   inviting an uncomment-in-repo. Byte-stability/golden tests move accordingly.
5. **Fix `show --origin`'s knob-blind provider drill-down rows** — the composed per-role
   provider rows read `claude # default` even when a depth knob names another provider
   (documented under-reporting in `_shared/configuration.md`); compose against the live
   config instead of the nil-config registry default.

**Obligations**: per item — tests (golden/idempotence for fence changes), `_cli-fab.md` +
SPEC mirrors for `--check`, `docs/specs/config.md`, memory sweep. Item 2 touches the fab-kit
binary (`internal/init.go`).

---

## Change 6 — `FAB_KIT_PATH`: session-scoped kit-resolution override (kit development)

> Folded in from backlog `[kpth]` (2026-08-08). Today **nothing** can make a kit-dev
> worktree run its own unshipped `src/kit/`: `fab/.fab-version` correctly pins the last
> released kit (it is a data-migration stamp — bumping it early would lie), the shim
> consults `~/.fab-kit/local-versions/{v}` only for the PINNED version (a `just
> install`-populated 2.17.1 is invisible while the stamp says 2.17.0), and the shim binary
> carries zero `FAB_*` env overrides (verified via `strings`, 2026-08-08). Consequence:
> `fab sync` perpetually redeploys the released kit over `.claude/skills/`, and
> `$(fab kit-path)` serves stale templates / `reference/fkf.md` / migrations — agents in
> the dev repo exercise one-release-behind skills. Live evidence: change 260808-s2sz burned
> a full review cycle + a failed T007 + a user decision on exactly this (acceptance A-014
> "deployed copies match sources" was unsatisfiable and had to be relaxed to sync-SOURCE
> equality).

- **Honor a per-process `FAB_KIT_PATH=<dir>` at the kit-resolution seam** — the shim and
  the fab binary, wherever the kit directory is resolved — so `sync`, `kit-path`,
  templates, reference docs, and migrations ALL follow one override. Deliberately **not**
  a `fab sync --source` flag: a sync-only override recreates the score-binary/source
  version-skew disease for every other kit reader.
- **A sibling of Change 1, deliberately NOT part of its mechanism.** Change 1's generic
  mapping walks *registry rows*; the kit path is *not a config field* and must never
  become one — a committed `kit.path` would break hermeticity for teammates, a persistent
  machine-wide setting would recreate the stale-kit disease this change diagnoses, and the
  shim resolves the kit *before* any config cascade exists. Env-only is the guardrail, so
  this is a special-cased variable at a different seam, landed in both binaries.
- **Guardrails** (from `[kpth]`):
  1. **Provenance is mandatory** — `fab sync` output and `fab doctor` print
     `kit: <dir> (FAB_KIT_PATH override)`, so a lingering shell export can never silently
     mix kits.
  2. **Hermeticity untouched** — per-process env only; no stamp or cache mutation; user
     repos see zero behavior change.
  3. **Set-once ergonomics via the dev repo's own existing `.envrc`** — in-repo autodetect
     rejected (it would make dev-repo behavior diverge invisibly from user repos). This
     does not contradict Change 1's direnv rejection: that decision declined to *document
     direnv as the end-user pattern* for worker selection; here `.envrc` is the fab-kit dev
     repo's own tooling, not a user-facing pattern.
- Independent of all other changes; ships any time (it is kit-dev tooling only).

**Obligations**: shim + `src/go` kit-path resolution + tests; `_cli-fab.md` (§ fab sync /
kit-path env note) + SPEC mirror; `fab doctor` output; this plan / `docs/specs/config.md`
env-override note; memory `distribution/kit-architecture.md` (kit-resolution section).
**No migration** (env-only, opt-in, no persisted state).

---

## Resolved decisions (all user-confirmed 2026-08-08)

1. **Discard the k0v3 branch** (worktree nimble-ravine) — the code it patched is stale;
   design re-derived fresh in Change 2; review findings preserved as test cases. Intake and
   backlog entry `[k0v3]` deleted, replaced by this plan's backlog entry.
2. **`explain`, not `docs`** — verb-shaped, kubectl-established, avoids "open the website"
   reading. Invisible `reference` alias retained.
3. **No `edit`, no `fix`/`doctor`** — `set`/`unset` are the nudge; `upgrade` (+ `--check`)
   is the repair story.
4. **`dispatch.mode: pane | native | headless`, default `native`** — a preference ceiling
   resolved per-dispatch down the fixed ladder pane → native → headless, never ascending;
   first possible rung wins; rung + reason surfaced; error only when nothing is possible.
   No `auto` value — `pane` *is* auto (pane-preference-with-fallback).
5. **`dispatch.watchable` dissolves via migration, no alias.**
6. **Provider capability is shipped data** — claude gets a real `dispatch_command`
   (headless grammar) and an explicit native flag; presence of a command field never selects
   a mode again.
7. **`interactive_command` / `headless_command`** — interaction-mode naming; tier-aligned
   naming rejected (pane workers are Tier-2 and run the interactive command).
8. **The two config files stay separate** — semantics-class (repo, committed, reproducible)
   vs preference-class (machine, personal); no merge. The cascade already lets a repo pin a
   preference when it truly is repo semantics.
9. **Fence gets scope annotations + `set --system` pointers** for preference-class adverts.
10. **`configscope` stays a leaf package** (cycle-forced, tiny, lint-guarded); target embed
    census after Change 5: `defaults.yaml` (values) + configref registry (schema/prose) +
    configscope (scope taxonomy) + `src/kit/scaffold/` (non-config files), zero stub copies.
11. **Per-session selection = the env override layer** — generic scope-gated `FAB_*` mapping
    over the registry (forward walk, YAML-parsed values), env > project > system > defaults;
    project-scoped env vars warn-and-ignore. Rejected: gitignored local config file,
    direnv-documented pattern, per-change stored preference, dirty per-worktree
    config.yaml (the `git add -u` trap).
12. **Launch flags are pure sugar** — `--workers` on the session-spawning commands exports
    the env var into the spawned session; no new resolution path, no persisted state.
13. **The env layer ships first** (reorder, user-confirmed 2026-08-08) — it has no
    dependency on the verb surface, the mode ladder, the renames, or the consolidation, and
    it is the change that unlocks runtime provider switching; the former Changes 1–4 shift
    down to 2–5 with relative order preserved.
14. **`FAB_KIT_PATH` is Change 6, not part of Change 1** (folded from backlog `[kpth]`,
    user-confirmed 2026-08-08) — same env-override philosophy, different seam: kit
    resolution happens in the shim before any config cascade exists, the kit path is
    deliberately not a registry field (env-only is the hermeticity guardrail), and it
    touches the fab-kit binary, which Change 1 never does. Rejected within it:
    `fab sync --source` (per-reader skew), in-repo autodetect (invisible dev/user
    divergence), a persistent `kit.path` setting (recreates the stale-kit disease).

## Context: why

- **The per-session pain**: the cascade bottoms out at per-checkout granularity, so
  "these two sessions should dispatch different workers" had no home short of dirtying a
  tracked file — and the A/B-comparison use case (kimi3 vs codex, two worktrees, same
  intake) is exactly that shape. Change 1 gives session-scoped intent a first-class layer.
- **The verb pain**: four subcommands didn't map onto the three intents (check state / print
  one thing / apply a change); "apply a change" was hand-editing YAML — the operation class
  that produced the comment-mashing bug (`setFabVersion`, fixed by yogn #473 then retired by
  the fence engine). `set`/`unset` close the last hand-editing driver.
- **The mode pain**: `dispatch_command`-absence-as-native-signal overloaded capability data
  as mode policy — claude's headless grammar became unshippable, `watchable` was a patch on
  the overload (adding pane eligibility only for command-less providers), and user mode
  intent had no home. The ladder gives intent a home and makes every config+environment
  combination resolve deterministically.
- **The source pain**: five value/schema sources plus a stub copy made "which config comes
  from where" a real question even for the maintainer. After Change 5: values / schema /
  scope, one home each, everything else rendered.
- **The kit-dev pain**: the dev repo's own agents run one-release-behind skills because
  nothing can point kit resolution at the worktree's `src/kit/` — a stale-by-design stamp
  doing its job correctly, with no session-scoped escape hatch. Change 6 adds the escape
  hatch without touching the stamp.
- **Prior art in this folder**: `config-upgrade.md` (2026-07-08, all three changes shipped)
  is the direct predecessor and established the registry/cascade/fence machinery this plan
  builds on.
