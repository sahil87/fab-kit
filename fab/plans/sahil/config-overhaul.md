# Config management overhaul — verbs, dispatch modes, field names, source consolidation

> Backlog detail doc — written 2026-08-08 after a `/fab-discuss` session. All open decisions
> below are **resolved** (user-confirmed in that session). The work is split into four
> changes, each independently shippable, in dependency order — intended to be picked up by
> other agents via `/fab-new`.
>
> **Supersedes change `260807-k0v3-consolidate-config-surface`**: its intake and backlog
> entry are deleted; its implementation branch (worktree nimble-ravine, apply done through
> 5 review-rework cycles, never merged) is **discarded as stale** — the design survives,
> re-derived fresh in Change 1, and the review-cycle findings survive as day-one test cases
> (see Change 1 § Lessons from the discarded branch).

## Goal

Four related cleanups to how fab-kit's configuration is inspected, mutated, and resolved:

1. **Verb surface** — `fab config` becomes six intent-grouped verbs (`show`/`explain`/`set`/
   `unset`/`init`/`upgrade`), with surgical fence-engine writes replacing hand-editing.
2. **Dispatch-mode semantics** — how a stage worker is launched (native sub-agent / tmux
   pane / headless CLI) becomes an explicit, impossible-state-free preference knob
   (`dispatch.mode`), delinked from the presence/absence of provider command fields.
3. **Field naming** — `session_command`/`dispatch_command` renamed to the honest axis:
   `interactive_command`/`headless_command`.
4. **Source consolidation** — one embedded values file, zero stub copies, a `--check` drift
   mode, and a fence that teaches which file each setting belongs in.

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

---

## Change 1 — the six-verb `fab config` surface

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
  `set providers.codex.profiles.default.model …`). `<value>` parsed as a YAML scalar; lists
  in flow syntax. `--system` creates `~/.fab-kit/config.yaml` (scaffold header) when missing
  — git-config precedent.
- **`unset`** removes the live override through the same engine; the fence regains the
  commented reference line. Restores inheritance (system value, else built-in) — semantic
  under presence=intent. Unsetting an unset key = exit-0 no-op with notice (idempotent).
  Deliberately ungated on value type (the repair path).
- **`init`** — bare becomes project mode (today a usage error; error→behavior is the safe
  breaking direction). `--project` retained (fab-kit's `fab init` shells out to it);
  `--system` unchanged; both together still an error; overwrite refusal unchanged.
- **No `edit` verb** ($EDITOR sugar over a fence-managed file invites mangling; `set`/`unset`
  is the nudge). **No `fix`/`doctor` verb** (`upgrade` already hoists/parks/regenerates/
  refuses-on-unparseable — extend it, don't duplicate it; see Change 4's `--check`).

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

## Change 2 — `dispatch.mode`: the descent ladder + capability delink

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

Rung prerequisites: **pane** = tmux reachable ∧ `interactive_command`; **native** =
provider's native flag; **headless** = `headless_command`. A rung whose prerequisite is
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
`_shared/context-loading.md`. This is the largest sweep class of the four changes — treat
the whole watchable/presence-signal mirror class as in-scope up front.

---

## Change 3 — rename `session_command`/`dispatch_command` → `interactive_command`/`headless_command`

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
- **Ordering note**: Changes 2 and 3 touch the same provider blocks. Recommended order is
  2 → 3 (mode semantics first; the rename sweep then also renames Change 2's additions).
  If executed 3 → 2 instead, Change 2 lands directly on the new names — either works; the
  second to land owns the sweep of the first's text.

---

## Change 4 — source consolidation, `upgrade --check`, fence clarity

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
   designed for (deferred in the original Change 3/j0qm): exit non-zero when a run would
   change the file (stale fence stamp, unparked unknown keys, missing fence), zero writes.
   The CI/human probe for "is this repo's config clean?" — this is the cleanup tooling
   story, alongside Change 1's `set`/`unset` (surgical repair) and `show --origin`
   (find typo'd overrides).
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

## Resolved decisions (all user-confirmed 2026-08-08)

1. **Discard the k0v3 branch** (worktree nimble-ravine) — the code it patched is stale;
   design re-derived fresh in Change 1; review findings preserved as test cases. Intake and
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
    census after Change 4: `defaults.yaml` (values) + configref registry (schema/prose) +
    configscope (scope taxonomy) + `src/kit/scaffold/` (non-config files), zero stub copies.

## Context: why

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
  from where" a real question even for the maintainer. After Change 4: values / schema /
  scope, one home each, everything else rendered.
- **Prior art in this folder**: `config-upgrade.md` (2026-07-08, all three changes shipped)
  is the direct predecessor and established the registry/cascade/fence machinery this plan
  builds on.
