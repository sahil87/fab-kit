# Intake: dispatch.mode — the descent ladder + capability delink

**Change**: 260808-yilt-dispatch-mode-descent-ladder
**Created**: 2026-08-08

## Origin

One-shot `/fab-new` dispatch from the operator, implementing **Change 3** of the plan
document `fab/plans/sahil/config-overhaul.md` (all decisions user-confirmed 2026-08-08
in a `/fab-discuss` session; see that plan's § Change 3, § Orchestration, and
§ Resolved decisions — items 4, 5, 6 are this change's charter).

> Implement Change 3 (dispatch.mode: the descent ladder + capability delink) of
> fab/plans/sahil/config-overhaul.md — read that full section plus the Orchestration and
> Resolved Decisions sections for scope, decisions, obligations, and rejected
> alternatives. This is described in the plan as the semantic heart and the largest
> sweep class of the six changes — treat the whole watchable/presence-signal mirror
> class as in-scope up front. Changes 1, 2, and 6 are running in parallel in sibling
> worktrees off the same base commit; this change has zero dependency on them per the
> plan's dependency map, so proceed independently. Note Change 4 (the
> interactive_command/headless_command rename) is a HARD downstream dependency on this
> change and will start only after this one lands, per the plan's critical path
> (C3 → C4 → C5) — that is out of scope for you, just be aware your work here is the
> blocking change for that follow-on. Shared-surface files (_cli-fab.md and its SPEC
> mirror, docs/specs/config.md, _shared/configuration.md, defaults.yaml) will also be
> touched by other changes in different sections/later — stay strictly within this
> change's declared scope in those files and don't try to pre-empt or merge the others'
> work.

**Parallelism constraints (binding on this change):**

- C1 (env layer), C2 (config verbs), C6 (FAB_KIT_PATH) run in parallel off the same
  base commit — zero semantic dependency; do NOT implement or anticipate any of their
  work (no env-layer code, no `set`/`unset` verbs, no `explain` rename, no kit-path
  seam changes).
- C4 (field rename `session_command`/`dispatch_command` → `interactive_command`/
  `headless_command`) lands strictly AFTER this change. **Keep the current field names
  everywhere** — all new code, config, docs, and comments this change writes use
  `session_command`/`dispatch_command`. C4's sweep will rename this change's text.
- On shared-surface files (`_cli-fab.md` + SPEC mirror, `docs/specs/config.md`,
  `_shared/configuration.md`, `defaults.yaml`), edit only the sections this change's
  scope requires (dispatch-mode/watchable/presence-signal content); leave adjacent
  sections untouched.

## Why

Today the dispatch mode — HOW a stage worker is launched (native Agent-tool sub-agent /
tmux pane / headless CLI) — is *f*(dispatch_command presence, `dispatch.watchable`,
`$TMUX`). Capability data doubles as mode policy, in two seams:

- `resolve_agent.go`'s `dispatchLineFor` (emits the `dispatch=` line when the provider
  carries a `dispatch_command`, or under the watchable+tmux opt-in a `session_command`).
- `fab dispatch start`'s auto ladder (`internal/dispatch.SelectMode`: explicit flags,
  then `$TMUX` presence decides pane vs headless).

Consequences of the overload:

1. **claude can never ship its headless grammar as data.** `defaults.yaml` says it
   outright: "claude deliberately carries NO dispatch_command — its absence is what
   selects native Agent-tool dispatch." The moment claude ships a `dispatch_command`,
   every out-of-tmux dispatch for every user flips to headless CLI. So a real,
   documented capability (`claude -p`, prompt on stdin) is structurally unshippable.
2. **A user adding the command themselves silently changes mode** for every dispatch —
   presence-as-policy means describing a capability *activates* it.
3. **`dispatch.watchable` is a patch on the overload** — it adds pane eligibility only
   for command-less providers, entangling a third input ($TMUX) with the presence
   signal. User mode intent still has no home: there is no way to say "always headless"
   for a provider that has both commands, or "let me watch" as a durable preference.

If we don't fix it: C4 (the rename to `interactive_command`/`headless_command`) would
rename fields whose *presence* still carries mode policy, cementing the overload under
honest-sounding names; and the config-overhaul's target state (values/schema/scope each
with one home, preferences with a real knob) stays unreachable. This change is the
plan's critical path (C3 → C4 → C5).

The fix (plan-resolved): provider fields become **pure capability grammar** ("here's
how", never "do it"), and mode becomes an **explicit preference knob**
(`dispatch.mode`) resolved per dispatch down a fixed, impossible-state-free descent
ladder. Rejected shapes (recorded in the plan): keeping presence-as-signal (the status
quo pain), an `auto` enum value (`pane` *is* auto — pane-preference-with-fallback), and
a read-time alias for `watchable` (the knob is young, 260806-mnri, and machine-local —
a migration closes it).

## What Changes

### 1. Provider fields become pure capability grammar

`src/go/fab/internal/config/config.go` — `ProviderConfig` gains an explicit
native-capability field:

```go
type ProviderConfig struct {
    SessionCommand  string `yaml:"session_command"`
    DispatchCommand string `yaml:"dispatch_command"`
    Native          bool   `yaml:"native"`      // NEW — native Agent-tool capability
    Profiles        map[string]RoleProfile `yaml:"profiles"`
    // ... (Tiers fallback etc. unchanged)
}
```

Rationale for shipped data (plan decision 6): native dispatch (the Agent tool) only
exists for claude-family models, and provider names are opaque — fab never infers from
names — so the capability must be shipped data, not a presence signal or a name match.
Overridable like every other provider field.

`src/go/fab/internal/agent/defaults.yaml` — the claude block gains both:

```yaml
providers:
  claude:
    native: true
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
    dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'
    profiles: ...   # unchanged
```

The `dispatch_command` grammar (exact flags settled at implementation per the plan):
`claude -p` runs headless and reads the prompt from **stdin** (where `fab dispatch`
pipes it — the codex/gemini convention), `--dangerously-skip-permissions` matches the
unattended posture of all three built-ins, `{model}`/`{effort}` substituted via
`internal/spawn.WithProfile` as today. Verify against the current claude CLI at
implementation (e.g. whether `--effort` is accepted in `-p` mode) before pinning.
codex/gemini blocks: unchanged (they already carry both commands; they get no `native`
flag — they have no native Agent-tool seam).

The defaults.yaml header comment "claude deliberately carries NO dispatch_command —
its absence is what selects native Agent-tool dispatch" is rewritten to the new
doctrine: **after this change, presence means "here's how", never "do it".**

Pinned/drift-guard tests to update: `defaults_test.go`
(`TestDefaultsFileIsWellFormed`), `agent_test.go` (`TestDefaultRoleProfilesArePinned` —
if it pins command fields), `defaultprofiles_mirrors_test.go` /
`stagemodels_doc_test.go` (`TestMirrorDocsMatchDefaultProfiles`,
`TestDocTablesMatchAgentMaps`, `TestCLIFabReferenceListsDefaultRoles` — the
`docs/specs/stage-models.md` inline-YAML sample and `_cli-fab.md` enumeration must
mirror the new claude block). The drift guards name themselves on failure — follow
them.

### 2. The knob: `dispatch.mode: pane | native | headless`

New config field, **default `native`**, **scope `both`** (settable once machine-wide in
`~/.fab-kit/config.yaml`).

`DispatchConfig` (config.go): drop `Watchable bool`, add `Mode string`. New nil-safe
accessor `GetDispatchMode()` returning the validated mode: absent key → `native`; an
unrecognized value → `fab: warning:` + `native` (the fail-open precedent — config must
never brick; mirrors the malformed-system-file and out-of-range column_width handling).
`GetDispatchWatchable()` is deleted.

**Semantics (plan decision 4, verbatim intent):** the setting is a **preference
ceiling, not a command**. Per dispatch, resolution starts at the preferred rung and
**descends the fixed ladder `pane → native → headless`, never ascending**: take the
first rung that is *possible* given provider capability + environment; surface the
chosen rung and reason on the existing `dispatched …` line; error only when no rung is
possible (a provider with no capabilities at all). No `auto` value — `pane` *is* auto
(pane-preference-with-fallback).

Rung prerequisites:

| Rung | Possible when |
|---|---|
| `pane` | tmux reachable (`$TMUX` at the resolver seam; probe at start) ∧ provider `session_command` |
| `native` | provider `native: true` |
| `headless` | provider `dispatch_command` |

A rung whose prerequisite is missing is **skipped, not errored** — this is what kills
the impossible states.

Behavior matrix (spec-bound; the default column reproduces today's
`watchable: false` behavior **byte-for-byte** for the built-ins):

| `dispatch.mode` | claude worker | codex/gemini worker | outside tmux |
|---|---|---|---|
| `pane` ("let me watch") | pane; no tmux → native | pane; no tmux → headless | descends past pane |
| `native` (default, "quiet in-context") | native | headless (no native seam; notice says so) | same — tmux irrelevant |
| `headless` ("detached processes, always") | headless | headless | same |

Properties the spec must state (plan-listed):

- **Never ascend** is the safety property: a descent is always *less* interactive, so
  an unattended run can never be surprised by a pane, and `native` never becomes a pane.
- Mixed setups just work: mode resolves per dispatch against the stage's resolved
  provider (`agent.session=claude` + `agent.workers=codex` → claude session, headless
  codex stage workers under the default).
- The old "no fallback between the two command fields" doctrine is **replaced by the
  ladder between modes**: each mode still requires its own field — the ladder descends
  *modes*, and a mode whose field is missing is simply not possible. No field ever
  substitutes for another.
- `dispatch.mode` is `scope: both`, so via C1's generic env mapping it becomes settable
  per-session as `FAB_DISPATCH_MODE` with **zero additional work in this change** (do
  not implement any env handling here — that is C1's mechanism; this change only needs
  the registry row to exist).

Registry (`internal/configref/configref.go`): drop the `dispatch.watchable` row, add a
`dispatch.mode` row (Default `"native"` — a real typed built-in value, like watchable's
`false` was; Advertise `true`; Scope both via `configscope.ScopeFor("dispatch")`). The
`dispatchSegment()` renderer is rewritten: the mode/ladder explanation replaces the
watchable prose; `dispatch.column_width` and `dispatch.reap_done` stay rendered inline
in the same single `dispatch:` segment (the one-YAML-block collision rule). The Go
constant pattern follows column_width/reap_done: a canonical
`config.DefaultDispatchMode = "native"` symbol read by both the accessor and the
registry row. `internal/configscope` needs **no change** — its taxonomy is keyed by
top-level key and `dispatch` is already `ScopeBoth` (verified in code; the plan's
"configscope rows" obligation is satisfied by the existing top-level entry). Update the
watchable-naming comment on that entry to name `dispatch.mode` instead.

Fence/golden tests in `internal/configupgrade` and `cmd/fab/config_test.go` that pin
the rendered dispatch segment are updated. After the registry change, regenerate this
repo's own `fab/project/config.yaml` managed fence via `fab config upgrade` and commit
it with the change.

### 3. `resolve_agent.go`: the emission matrix reads (mode, capability, $TMUX)

`dispatchLineFor` is rewritten — same pure-function shape (caller reads `$TMUX` and
config, passes them in; table-testable), new inputs: the resolved `dispatch.mode`, the
provider's three capability fields, and `tmuxEnv`. It resolves the ladder and returns
the command for the resolved rung:

- resolved rung `native` → `""` (omit the `dispatch=` line)
- resolved rung `pane` → the provider's `session_command`
- resolved rung `headless` → the provider's `dispatch_command`
- no rung possible → **error** (new: `dispatchLineFor` or its caller returns an error
  naming the provider and the missing capabilities, instead of silently resolving
  native). This covers a defined provider with no commands and no native flag, and an
  unknown provider name reached via a depth knob / profile override (today both
  silently resolve native; under capability-as-data there is nothing they can do). The
  `--provider` unknown-name lookup failure (`unknownProviderError`) is unchanged and
  fires first.

**The `dispatch=` contract is unchanged in shape** (plan-stated): line absent ⇔
resolved mode is native; line present carries the resolved mode's command with
`{model}`/`{effort}` already substituted (full model ID, never aliased). The skills'
branch-on-presence seam needs no restructuring. `--alias`, `--model`, `--effort`,
`--provider` behavior otherwise unchanged.

### 4. `fab dispatch start`/`restart`: the auto ladder re-derives from (mode, capability, env)

`internal/dispatch.SelectMode` (pane_mode.go) is rewritten to take the resolved
`dispatch.mode` and provider capability alongside the existing explicit signals. Kept
as a pure function; the cobra layer passes config + env in.

- **Per-invocation `--pane`/`--headless` survive** as one-shot overrides (mutually
  exclusive at the cobra layer, unchanged), including their existing hard-error
  semantics: forced `--pane` requires reachable tmux + a provider `session_command` and
  hard-errors without either. `--timeout ⇒ headless` and `--server ⇒ pane` remain
  explicit signals (unchanged rungs above auto).
- **The auto rung descends the same ladder as `dispatchLineFor`** from the configured
  `dispatch.mode` — one selection function/table shared or mirrored between the two
  seams so they cannot disagree on the same inputs (the plan's stated invariant).
- **`start` resolving to `native` is an actionable error**, not a launch: `fab
  dispatch` has no native Agent-tool seam. This is reachable only when the environment
  changed between the orchestrator's resolve and `start` (e.g. `mode: pane`, claude:
  tmux died in between → pane impossible → ladder lands on native). The error tells the
  caller to re-resolve (`fab resolve-agent` would now omit the `dispatch=` line, i.e.
  native Agent-tool dispatch). This replaces today's documented soft-fallback edge
  (watchable + tmux died → headless → error on missing dispatch_command) with an
  honest, named outcome.
- **`AutoReason`/`dispatched …` suffixes**: the selection-source strings are reworked
  to surface the chosen rung and why (the preference, plus any descent taken and its
  reason — e.g. no tmux, no native seam). Today's four suffixes (`auto: tmux`,
  `auto: no tmux`, `auto: tmux unreachable`, `auto: no session_command`) and the two
  fallback notices are replaced/extended accordingly; exact strings are an
  implementation decision — the requirement is that every auto selection names rung +
  reason (compliance-visibility principle), and the kit text documents the exact final
  set. The soft-fallback asymmetry survives: auto selections soft-descend with a
  notice; explicit selections hard-error.
  <!-- assumed: suffix string design deferred to apply — plan requires surfacing rung+reason but pins no strings -->
- `dispatch_restart.go` re-derives mode from the current environment through the same
  path (its existing contract), picking up the new inputs for free.

### 5. Migration: `watchable` dissolves (no read-time alias)

New migration `src/kit/migrations/2.17.2-to-2.18.0.md` (user-data restructure ⇒ ships
as a migration per the constitution; minor version bump for a config-schema change,
following the 2.16.19-to-2.17.0 precedent; `src/kit/VERSION` bumped 2.17.2 → 2.18.0).
Following the established migration format (summary/pre-check/changes/verification,
atomic writes, live-keys-only, fence untouched):

- Sweep **both scopes** (`fab/project/config.yaml` AND `~/.fab-kit/config.yaml` — the
  key is scope `both`; the 2.15.7-to-2.15.8 lesson: a migration touching one location
  strands the other).
- Live `watchable: true` under a top-level `dispatch:` block → rewrite to `mode: pane`.
- Live `watchable: false` → **delete the key** (absent = the `native` default; plan:
  "absent/false → nothing").
- Commented lines and the managed fence are never touched (`fab config upgrade`
  regenerates the fence with the new segment).
- Idempotent sentinel: skip when neither file carries a live `watchable:` key under
  `dispatch:`.
- No read-time alias: the binary stops reading `dispatch.watchable` entirely. A stale
  key left behind (migration not run) becomes an inert unknown key — yaml.v3 ignores
  it; behavior falls back to the `native` default, which is byte-stable with
  `watchable: false` and only differs from a previously-true opt-in by no longer
  opening panes until the migration runs.

### 6. Kit-text sweep — the watchable/presence-signal mirror class (in-scope up front)

Grep-verified inventory (sweep both `watchable` and the presence-doctrine claims —
"absence of dispatch_command signals native", "deliberately carries NO
dispatch_command", "a second reason dispatch= may be present", the four `auto:`
suffixes; per the ioku lesson, include user-facing **string literals** — cobra `Short`
texts, warning/notice strings, flag help — in the sweep, and re-sweep after every
behavior-changing rework):

**Skills (`src/kit/skills/`)** — each touched skill's SPEC mirror
(`docs/specs/skills/SPEC-*.md`) updates with it, per the constitution:

- `_preamble.md` — the always-load config.yaml enumeration (`dispatch.watchable` →
  `dispatch.mode`); § Subagent Dispatch / CLI-Adapter Dispatch: the "dispatch.watchable
  is a second reason dispatch= may be present, never a third branch" paragraph is
  rewritten to the mode-ladder framing (dispatch sites still branch only on presence
  and never execute the value — that sentence survives); the `auto:` suffix enumeration
  in the pane-mode bullet.
- `_cli-fab.md` — § fab resolve-agent: the `dispatch=` emission rules and the entire
  `dispatch.watchable` bullet become the mode-ladder contract (emission =
  f(mode, capability, $TMUX); no-rung error); § fab dispatch: the auto ladder
  description, selection suffixes, forced-flag semantics; § fab config
  reference/upgrade: schema-coverage lines naming `dispatch.watchable` →
  `dispatch.mode`, plus the claude provider now carrying `dispatch_command` + `native`.
- `_cli-agents.md` — provider-table/spawn-composition text restating the presence
  doctrine or claude's command-less shape (grep hit on dispatch_command; rewrite only
  what states mode policy).
- `_pipeline.md`, `fab-continue.md`, `fab-proceed.md` — dispatch-seam restatements of
  the presence doctrine (branch-on-presence survives; any "absence = native because no
  dispatch_command" *reasoning* text updates to "absence = resolved mode is native").
- `fab/project/config.yaml` fence (this repo) — regenerated via `fab config upgrade`.

**Specs (`docs/specs/`)**: `harness-adapters.md` (adapter-selection contract),
`stage-models.md` (resolver contract + the inline defaults.yaml sample + `dispatch=`
emission conditions), `config.md` (field table: drop watchable, add mode; dispatch
section), `architecture.md` (watchable mention), `glossary.md` (watchable entry →
dispatch.mode/descent-ladder entries). `docs/specs/index.md` row descriptions only if
they name watchable (verify at apply).

**Memory** is hydrate's job — see Affected Memory below. `docs/memory/*/index.md`
regenerate via `fab memory-index`.

**Constitution / standards note**: the change alters CLI output text (the
`dispatched …` line) and the config surface — check `shll standards` for the governed
surfaces before finalizing user-facing strings (constitution § Toolkit Standards).

### 7. Tests (new + updated)

- Table tests for the full ladder matrix: mode × {claude, codex/gemini,
  custom-no-capability, unknown-provider} × {tmux, no-tmux} — for BOTH seams
  (`dispatchLineFor` and `SelectMode`/start's auto rung), asserting they agree.
- Byte-stability regression: default `native` with the shipped built-ins reproduces
  today's `watchable: false` outputs exactly (claude → no `dispatch=` line;
  codex/gemini → their `dispatch_command`); existing watchable test cases are rewritten
  as `mode: pane` cases with identical expectations.
- Accessor tests: absent → native; invalid value → warn + native; each valid value.
- No-rung error: defined provider with no commands/flag, and unknown provider via a
  depth knob, both error naming what's missing.
- `start`-lands-on-native error path.
- Fence/golden/idempotence tests (configupgrade, config_test.go) for the new segment;
  registry lint (paired-entry rules) passes.
- Migration file carries its own Verification section (yq probes, idempotent re-run) —
  markdown instruction file, no Go test.

## Affected Memory

- `_shared/configuration.md`: (modify) § dispatch — the field table and watchable
  prose (18 mentions) become the `dispatch.mode` ladder; provider capability fields
  (claude's new `dispatch_command` + `native`).
- `_shared/context-loading.md`: (modify) 7 watchable mentions in the always-load /
  dispatch-seam description.
- `runtime/providers-and-profiles.md`: (modify) 17 mentions — provider grammar
  purity, the emission matrix, claude's shipped headless grammar.
- `runtime/dispatch.md`: (modify) mode selection (SelectMode ladder, auto reasons,
  start-native error), watchable mention.
- `distribution/kit-architecture.md`: (modify) small — watchable mention.
- `distribution/migrations.md`: (modify) register the 2.17.2-to-2.18.0 migration in
  the catalog.

## Impact

- **Go binary (`src/go/fab/`)**: `internal/config/config.go` (DispatchConfig,
  ProviderConfig, accessors, DefaultDispatchMode), `internal/configref/configref.go`
  (row swap + dispatchSegment), `internal/configscope/configscope.go` (comment only),
  `cmd/fab/resolve_agent.go` (+`_test`), `cmd/fab/dispatch_start.go` /
  `dispatch_restart.go` (+`_test`s), `internal/dispatch/pane_mode.go` (SelectMode,
  AutoReason set, notices) (+`dispatch_test.go`), `internal/agent/defaults.yaml`
  (+ pinned tests), `cmd/fab/config_test.go`, `internal/configupgrade` tests.
  `resolve_agent.go` + `internal/dispatch` are **exclusive to this change** per the
  plan's conflict table.
- **Kit (`src/kit/`)**: `skills/_preamble.md`, `skills/_cli-fab.md`,
  `skills/_cli-agents.md`, `skills/_pipeline.md`, `skills/fab-continue.md`,
  `skills/fab-proceed.md` (presence-doctrine text only), `migrations/2.17.2-to-2.18.0.md`
  (new), `VERSION` (2.17.2 → 2.18.0).
- **Specs (`docs/specs/`)**: `harness-adapters.md`, `stage-models.md`, `config.md`,
  `architecture.md`, `glossary.md`, `skills/SPEC-*.md` mirrors of every touched skill.
- **Repo config**: `fab/project/config.yaml` fence regenerated.
- **Behavior**: unconfigured repos see **zero behavior change** (default `native` ≡
  today's `watchable: false`); `watchable: true` users are migrated to `mode: pane`
  (same behavior via the ladder); the only new failure mode is the honest no-rung /
  start-lands-on-native error where today silently resolves native or dead-ends.
- **Downstream**: C4 renames the two command fields after this lands; C5 folds
  `dispatch.mode: native` into defaults.yaml as part of source consolidation. Neither
  is anticipated here.

## Open Questions

- None blocking — the plan resolves every design decision (its § Resolved decisions
  items 4–6). Implementation-detail choices are recorded as graded assumptions below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ladder semantics exactly per plan decision 4: `dispatch.mode` enum of pane/native/headless, default `native`, scope `both`, preference ceiling descending `pane → native → headless`, never ascending, first possible rung wins, rung+reason surfaced, error only when no rung possible, no `auto` value | Plan-resolved, user-confirmed 2026-08-08 | S:95 R:75 A:95 D:95 |
| 2 | Certain | `dispatch.watchable` dissolves via migration (`true` → `mode: pane`, absent/`false` → nothing), no read-time alias | Plan decision 5, explicit | S:95 R:70 A:95 D:95 |
| 3 | Certain | Provider capability ships as data: claude gets a real `dispatch_command` + an explicit native flag; presence never selects mode again | Plan decision 6, explicit | S:95 R:70 A:95 D:95 |
| 4 | Certain | Field names `session_command`/`dispatch_command` are kept verbatim everywhere in this change; C4 owns the rename sweep | Dispatch text explicit; plan critical path C3 → C4 | S:95 R:85 A:95 D:95 |
| 5 | Certain | `internal/configscope` needs no row change — the taxonomy is keyed by top-level key and `dispatch` is already `ScopeBoth`; only its watchable-naming comment updates | Code-verified (configscope.go keyScopes); satisfies the plan's "configscope rows" obligation as written | S:85 R:90 A:95 D:90 |
| 6 | Confident | claude headless grammar: `claude -p --dangerously-skip-permissions --model {model} --effort {effort}`, prompt on stdin — verify exact flag validity against the current claude CLI at implementation | Plan defers exact flags to implementation; mirrors the shipped session_command posture and the codex/gemini stdin convention | S:70 R:85 A:75 D:70 |
| 7 | Confident | The native-capability flag is spelled `native: true` on the provider block | The plan's own example spelling; no competing candidate | S:70 R:80 A:80 D:75 |
| 8 | Confident | Invalid `dispatch.mode` value → `fab: warning:` + resolve to `native`; absent → `native` | Fail-open config precedent (config must never brick: malformed-system-file, out-of-range column_width) | S:60 R:85 A:80 D:75 |
| 9 | Confident | Migration file is `2.17.2-to-2.18.0.md` with a minor `src/kit/VERSION` bump (2.17.2 released → 2.18.0 next) | Config-schema-change precedent: 2.16.19-to-2.17.0 was minor for the profiles schema; FROM=released, TO=next convention | S:65 R:80 A:80 D:75 |
| 10 | Confident | A provider with no possible rung — including an unknown provider name reached via a depth knob/profile (today silently native) — errors at dispatch resolution, naming the missing capabilities | Plan: "error only when no rung is possible"; under capability-as-data an unknown provider has no capabilities; `--provider` lookup failure unchanged | S:65 R:75 A:80 D:70 |
| 11 | Confident | `fab dispatch start` re-deriving to `native` (environment changed between the two seams) is an actionable error directing re-resolution, replacing the documented tmux-died soft-fallback edge | start has no native Agent-tool seam; plan invariant "the two seams cannot disagree" on same inputs; honest error beats dead-end fallback | S:60 R:80 A:75 D:70 |
| 12 | Confident | Exact wording/format of the new `dispatched …` rung+reason suffixes and fallback notices (replacing the four `auto:` suffixes) is settled at apply; requirement is every auto selection names rung + descent reason, kit text documents the final set, and `shll standards` is checked for CLI-output surfaces | Plan pins the requirement, not the strings; low risk, easily reworked in review | S:55 R:70 A:60 D:40 |

12 assumptions (5 certain, 7 confident, 0 tentative, 0 unresolved).
