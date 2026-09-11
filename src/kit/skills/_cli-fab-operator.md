---
name: _cli-fab-operator
description: "Fab CLI reference — the `fab operator` and `fab agent` command families (the tmux operator's state/tick/track verbs; the stage/role agent-resolution query). Split out of _cli-fab so operator consumers load only this slice."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Fab CLI Reference — operator & agent

> Loaded via a skill's `helpers: [_cli-fab-operator]` frontmatter. Core commands (change/status/score/preflight/log/resolve/config/…) are in `_cli-fab`; `fab pane`/`fab dispatch` in `_cli-fab-pane`.

## Contents

- fab operator
- fab agent

---

## fab operator

```
fab operator [--workers <provider>]
```

Singleton tmux-tab launcher for `/fab-operator`.

**rk delegation (checked first)**: when a **capable** run-kit is on PATH, the bare command hands the ENTIRE launch to `rk operator` — fab execs it (process replacement), appending `--workers <v>` iff the flag was supplied. Capability is probed side-effect-free: `exec.LookPath("rk")` + `rk operator --help` exiting 0 with `--workers` in its output (the flag fab passes through is the discriminant — the binary is probed, never the version string). rk then owns the whole launch: its own preconditions (`$TMUX`, fab on PATH — fab's own `$TMUX` check is skipped on this path), the role-marked singleton (`@rk_win_role=operator` wins over a merely-named window), launcher resolution via `fab agent operator --print`, the provider-agnostic **typed** kickoff, and `--workers` validation (rk restricts the value to letters, digits, `_`, `-` — a usage error, exit 2; a **behavior delta** from the fallback's verbatim pass-through). A failed probe (rk absent, or present without a capable `operator`) falls through **silently** to the built-in launcher below; a failure AFTER a passing probe is surfaced, never fallen back on (rk may have mutated tmux state — retrying risks a duplicate operator window). The operator **subcommand family is never delegated** — choreography stays in fab.

**Built-in launcher (the rk-absent fallback)**: Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`). The singleton check is an **exact, server-wide** window-name match: `tmux list-windows -a` enumerated and compared exactly (never tmux target resolution, whose prefix/glob fallback would let e.g. `operator-logs` mask the real check; `-a` enforces the one-operator-per-SERVER invariant across sessions). If a window named exactly `operator` exists anywhere on the server → select it by window ID, switching the client to its session when needed (`Switched to existing operator tab.`); else create the window running `{operator-session-command} {shell-quoted fab-operator prompt}` (`Launched operator.`). The prompt is rendered by `internal/agent.SkillPrompt` for the launched provider (`$` for `codex`, `/` otherwise — including Claude when command resolution falls back to the built-in).

**`--workers <provider>`** on the fallback path is launch sugar for the new-window path: the shell command is prefixed with the safely quoted assignment `FAB_AGENT_WORKERS='<provider>'`. The value passes through verbatim with no provider lookup or validation (on the delegated path it rides argv instead and rk validates — see above). When the singleton already exists, selection is unchanged and no new environment can be injected.

**Launch cwd (no git-repo dependency; fallback path)**: the new window's working directory (`tmux new-window -c <dir>`) is resolved by trying `git rev-parse --show-toplevel` first and falling back to `os.Getwd()` when that fails. The operator launches **inside a git repo** (cwd = repo root) **or from a neutral parent directory** (cwd = current directory), and errors only if both resolutions fail. This matches the per-tmux-server, cross-repo singleton model: the operator's natural launch point is a neutral dir with no `fab/` project.

**Session command resolution (no `fab/`-project dependency) + operator-role profile (fallback path — the delegated path's launcher resolution is rk's, via `fab agent operator --print`)**: the operator resolves the **operator role** in-process (`agent.ResolveRole(cfg, "operator")`) → its provider (the `agent.session` knob, since `operator` is a Tier-1 role) → that provider's `interactive_command`, then injects the role's full `{model, effort}` via `spawn.WithProfile`; it does not consume the YAML `model_alias` seam. When a `fab/` project is resolvable (`resolve.FabRoot()` succeeds) the config supplies the knob + provider; when `resolve.FabRoot()` **fails** — the operator is launched from a neutral directory with no `fab/` project anywhere up the tree (its natural cross-repo home) — this is **non-fatal**: `config.Load` returns an empty config, so `ResolveRole`/`ResolveProvider` degrade to fab-kit's built-in `operator`-role profile (inspect it with `fab agent operator -o yaml`) + built-in claude provider (`spawn.DefaultSpawnCommand`). `WithProfile` is grammar-forgiving: for a **template** `interactive_command` containing `{model}`/`{effort}` — including the built-in claude default, which is templated — it **substitutes** the resolved values in place (all-or-nothing; an empty value drops the placeholder's token and a preceding `-`-flag); for a command carrying **no placeholder** (e.g. a user's plain-form config carried forward by the 2.13.0 migration) it instead **appends** `--model <model> --effort <effort>` to the END (last-wins; each flag omitted when its value is empty, per the `empty ⇒ omit` convention). A provider without an `interactive_command` falls back to `spawn.DefaultSpawnCommand` (the templated claude default, still profile-substituted). So a `fab/`-less launch composes a fully-defaulted command: default session command + the built-in operator `{model, effort}` (byte-identical whether resolved by substitution or, for a plain user command, by append).

### fab operator tick-start

```
fab operator tick-start [--diff [--quiet]]
```

Called at start of each operator tick. Increments `tick_count`, writes `last_tick_at` (RFC3339 UTC) to the **server-keyed** state file (not the old repo-rooted `.fab-operator.yaml`) — with `--diff`, also writes `last_full_at` (RFC3339 UTC) on every tick that emits the full document. The flagless form is unchanged — stdout:

```
tick: N
now: HH:MM
```

**`--diff`** additionally probes the tracked items — joins every pane item against an internal fleet snapshot (the same pane-map discovery+resolve pipeline `fab pane map` runs — one enumeration path) on `scope.pane`, fingerprinted by `scope.pane_pid`, runs every due shell probe, evaluates `done_when`/`depends_on`/staleness — and emits one stdout document: the `tick:`/`now:` header lines, then four YAML blocks in this pinned order — `deltas:`, `candidates:`, `needs_check:`, then `items:` **or** `fleet_summary:` (never both keys):

```yaml
tick: 48
now: 16:36
deltas:
    - kind: changed             # changed | done | stale | probe_error | pane_death | pane_mismatch | agent_exited | stage_advance | review_fail
      id: pr-913
      fields: { state: { from: OPEN, to: MERGED } }   # changed only — one entry per differing declared field
    - kind: changed
      id: tireless-perch
      fields: { change: { from: null, to: 4a8m } }    # a change appearing or switching on a pane item
    - kind: done
      id: pr-913
      then: "spawn n34 in ~/code/hexokit via /fab-fff"   # done only — the item's then, verbatim (present-keyed; null when none)
    - kind: stale
      id: linear-bugs
      age: 11m                    # since checked_at (added_at when never checked)
    - kind: probe_error
      id: deploy-prod
      error: "exit 1: gh: Not Found"
      failures: 3
      paused: true                # true from the tick that trips the 3-failure cap
    - kind: pane_death            # pane deltas, keyed by id
      id: r3m7
      pane: "%3"
    - kind: pane_mismatch         # recycled pane only — the recorded scope.pane_pid fingerprint differs from the pane's current shell pid
      id: r3m7
      pane: "%3"
      found: k8ds                 # the observed change id; null when none resolvable — never part of the mismatch decision
    - kind: agent_exited
      id: r3m7
      pane: "%3"
      command: zsh                # the pane's shell foreground command
    - kind: stage_advance         # review_fail for the review→apply rework reset
      id: k8ds
      pane: "%7"
      from: apply
      to: review
candidates:                       # pane items only — waiting first, then idle, item id within each class
    - pane: "%7"
      id: k8ds
      agent_state: waiting        # waiting | idle
      state_duration: 32m         # rk's agent_state_duration verbatim, waiting and idle alike (null when rk reports none) — the input to the §5 30m auto-default
      idle_duration: null         # non-null only for idle (upstream idle-only semantics)
needs_check:                      # due agent items — the LLM runs the instruction and records via track observe
    - id: linear-bugs
      kind: linear
      age: 11m
      instruction: "mcp__claude_ai_Linear__list_issues project=DEV …"
items:                            # one row per item, ordered kind → scope.repo → id — the status frame's data source
    - id: r3m7
      kind: pane
      state: live                 # held | pending | live | watching | stale | paused | done
      pane: "%3"
      change: r3m7                # the observed change id; null (with stage/display_state/pr_url) when none
      repo: /home/user/code/foo
      session: work
      stage: review-pr
      display_state: done
      agent_state: idle
      idle_duration: 8m
      pr_url: https://github.com/acme/foo/pull/412
      checked_at: null            # pane items: null (the snapshot is the state; rendered `live`)
      next: null                  # present-keyed — the item's then, "spawn" for pending, "held: <dep-id>" for held, else null
    - id: pr-913
      kind: github-pr
      state: done
      repo: /home/user/code/hexokit
      last: { state: MERGED, mergedAt: "2026-09-11T16:35:10Z", mergeable: UNKNOWN }
      checked_at: "2026-09-11T16:36:02Z"
      check_every: 2m
      unchanged: 0
      next: "spawn n34 in ~/code/hexokit via /fab-fff"
    - id: n1
      kind: note
      state: watching
      repo: null
      text: "Phase 2 of 4 — …"
      updated_at: "2026-09-11T16:20:00Z"
      next: null
```

- **Due set and probe runner.** `pane` items are probed every tick (the snapshot join). A `shell` item is due when not done, not paused, and `checked_at` is null (always due) or `now - checked_at ≥ check_every`. Due shell probes run **sequentially**: argv executed directly (never a shell string; the operator's environment inherited), 10 s per-probe timeout, stdout parsed as a JSON **object** (a non-JSON or non-object stdout is a probe error). A 60 s per-tick wall-clock budget bounds the fleet; items the budget cuts off emit `probe_error` with `error: "skipped (tick budget)"` and `failures` untouched. Only the declared `probe.fields` (top-level keys, or dotted paths `a.b` into nested objects — fields sharing a prefix such as `a.b` and `a.c` merge under one `a`) are extracted, compared against `last`, and stored back; an absent declared field stores null, so a value→absent transition stays visible and `field == null` predicates work. A successful JSON-object probe resets `failures` to 0 before the unchanged/delta branch — the pause cap counts **consecutive** failures only. `agent` items are never run by the binary — due ones are *listed* under `needs_check:` for the LLM. Done items are never re-probed (the level-triggered `done` delta re-derives from `last`).
- **Two delivery classes.** `done`, `stale`, `probe_error`, `pane_death`, `pane_mismatch`, `agent_exited` are **level-triggered** — re-emitted every tick until acked: `track rm` is the ack for `done` and the pane deltas, `track observe` clears `stale`, `track update --resume` or `track rm` clears `probe_error` — so a crash between diff and action loses nothing. `changed`, `stage_advance`, `review_fail` are **consumed-on-read** (diffed against `last` / the `scope.stage` baseline, consumed by the same-write baseline update); a lost one costs a missed report only. `review→apply` is the rework reset path and emits `review_fail`, not `stage_advance`.
- **`probe_error` re-emission.** A fresh probe failure carries the composed one-line message — `exit N: <first stderr line>`, `timeout after 10s`, or `stdout is not a JSON object`. Once the third consecutive failure trips `paused: true` the item stops being probed, and the delta re-emits every tick as `error: "paused after N consecutive probe failures"` until `track update --resume` (which zeroes `failures`) or `track rm`. An item paused by the user (`track update --pause`) with fewer than 3 failures emits nothing. A `depends_on` id missing from the list emits the same delta class with `error: "unknown dependency <id>"` (the item reads `held`; `failures` is untouched).
- **Item states** (each row's `state`): `held` (any `depends_on` item not done this tick), `pending` (pane item with null `scope.pane` and deps satisfied), `live` (pane item cleanly joined), `watching` (shell/agent item not done — **also a pane item whose pane is dead, mismatched, or exited**: the level-triggered delta carries the truth), `stale` (agent item with `now - checked_at > 2 × check_every`, or `checked_at` null and `added_at` older than 2×), `paused`, `done` (`done_when` true, or the pane built-in fired) — `done` is level-triggered until `track rm`. A pane item's built-in completion is **persisted** as `done_at` (RFC3339, binary-set in the same tick write) so the verdict is durable: a completed item is never re-joined or diffed again, and a pane that vanishes after completing emits `done` (with its `then`), never `pane_death`.
- **Row field sets.** Every `items:` row carries `id, kind, state, next` (`next: null` present-keyed). Probed rows (`shell`/`agent`) add `repo, last, checked_at, check_every, unchanged` — agent rows also `seen`. Pane rows add `pane, change, repo, session, stage, display_state, agent_state, idle_duration, pr_url, checked_at: null`; an unjoined pane row falls back to the item's baseline identity fields with null observed fields. `note` rows add `repo, text, updated_at`; `task` rows add `repo`.
- **Pane built-in completion** applies only when `scope.change` is non-null; a change-less pane item is never done on its own (it leaves the list only via `track rm` or a `then`). With a change, the predicate is a display-state/terminal-stage predicate (never a stage diff): with `stop_stage` null it fires only at the pipeline terminus — `review-pr` with `display_state` done/skipped (hydrate and ship are mid-pipeline and never complete an item by themselves); with a `stop_stage` it fires past the stop in stage order, or at the stop with display_state done/skipped (a finished stop-stage auto-activates the next stage, so equality alone would race the transition).
- **Detection semantics (pane items).** `pane_death` fires when the item's pane is absent from the snapshot. `pane_mismatch` fires when the recorded `scope.pane_pid` is non-null AND the pane's current shell pid differs — pids come from one batched `tmux list-panes -a -F '#{pane_id} #{pane_pid}'` per tick (run only when at least one pane item has a non-null `scope.pane`; a failed batch yields an empty map, never an error): tmux recycles `%N` pane IDs across server restarts while the socket-keyed state file survives, so a differing fingerprint proves the pane is not the one tracked. A null recorded pid or an unreadable current pid joins on the pane id alone; the observed change never participates in the decision (`found:` stays the observed change id or null). A mismatched pane is never diffed, baseline-updated, or listed as a candidate. `agent_exited` fires when the pane IS present and fingerprint-matched but no live agent remains, decided by the pane row's `has_agent` tri-state from `rk mux panes --json`: `false` ⇒ exited, `true` ⇒ alive (no further check); only a **`null`** `has_agent` (an uninstrumented pane — observed: the operator's own pane reports `null`) falls back to the process-tree walk, which fires when the pane's **foreground command is a shell** (basename ∈ `sh bash zsh fish dash ksh tcsh csh nu`) AND its pane-PID process tree contains no live agent. Agent names are `claude`/`claude-code` plus every merged provider `interactive_command`'s quote-aware leading-command-word basename after true POSIX `NAME=value` prefixes, with shell names excluded; matching checks `comm` first and at most the first two cmdline tokens. Only positive live-agent evidence suppresses the delta; PID or walk failure silently fails toward emitting. Agent state is never liveness evidence — its option can retain the agent's last stale value after exit. Evaluation order per item is `pane_death` → `pane_mismatch` → `agent_exited` → clean join: a mismatched pane hosting a shell emits only `pane_mismatch`, without a tree walk; mismatched/exited panes get no baseline write, no stage diffs, and no `candidates:` row.
- **`candidates:`** — pane items whose snapshot agent state is waiting or idle, waiting first then idle, sorted by item id within each class; unknown and active panes are excluded, so on rk-less servers the block is empty. Each row carries `state_duration` — rk's `agent_state_duration` verbatim for `waiting` and `idle` (`null` when rk reports none); `idle_duration` stays idle-only — and `state_duration` is the input to the §5 30m auto-default. This is the operator §5 sweep population.
- **`--quiet`** — valid only with `--diff` (`--quiet` alone errors `--quiet requires --diff` before any state read/write, consuming no tick). On a **quiet tick** — `deltas:` AND `needs_check:` both empty AND the last full document (`last_full_at`) less than 10 minutes old (a built-in constant, not a flag or config knob; a missing or unparseable `last_full_at` counts as stale, so a fresh state file and the first tick after upgrade emit the full document) — the `items:` block is **replaced** by a five-count `fleet_summary:` mapping:

  ```yaml
  fleet_summary:
      tracked: 5      # ALL items not done, any kind
      waiting: 1      # not-done pane items whose snapshot agent_state is waiting
      idle: 1         # … idle
      active: 3       # … active
      unknown: 1      # … unknown — including a pane item with no live reading (dead/mismatched/exited/pending)
  ```

  The invariant is `tracked ≥ waiting + idle + active + unknown`: done items are excluded from every count, and non-pane items count only into `tracked`. A tick with non-empty `deltas:` or `needs_check:`, any tick whose `last_full_at` is at least 10 minutes old, and any tick without `--quiet` emits the full document — and every full document writes `last_full_at` in the same atomic mutation. At the 1m backoff floor that is roughly every 10th tick; at a cadence of 10m or slower every tick is full. The four-block order holds in both shapes — the quiet document still emits `needs_check: []` explicitly.
- **Baseline writer.** `--diff` updates the baseline in the **same atomic mutation** as the tick bookkeeping and the probe bookkeeping: for each cleanly-joined pane item, `scope.stage` ← snapshot stage, `scope.agent` ← the snapshot agent state verbatim (null for unknown), and `scope.change` ← the snapshot change id when resolved (an unresolved snapshot change fabricates no delta and leaves the baseline alone — sticky, mirroring the em-dash stage rule; when the observed change differs from the baseline — null → id, or id → other id — the tick emits a consumed-on-read `changed` delta with `fields: {change: {from, to}}` and consumes it in the same write; on a first observation — baseline null → id — with `branch_map[<change>]` absent, the same mutation also writes `branch_map[<change>] = {branch, repo}`, `repo` from `scope.repo` and `branch` from `git -C <cwd> branch --show-current` on the pane's cwd, skipped when the result is empty or git fails, and retried on a later tick while the entry stays absent); for each probed item, `last`/`checked_at`/`unchanged` (0 on a field delta, +1 otherwise)/`failures`/`paused`. `updated_at` moves only on a stage change or a probe field delta; `checked_at` moves on every completed probe, success or failure. Dead/mismatched/exited items stay untouched; an unresolved (em-dash) snapshot stage fabricates no delta and leaves the baseline stage alone. With an **empty tracked list** the snapshot subprocess is skipped entirely and every block emits `[]` (or the zero `fleet_summary:`) — a no-op tick is first-class. A **legacy-shaped** state file converts on this verb too, and the converted pane items are diffed in the same run — see the **Legacy conversion** paragraph below.
- **Clock reconcile.** `--diff` ends with the level-wise clock reconciles (mute-if-untracked, then the schedule reconcile) — the shared **Clock side effect** paragraph below.

**State path** (server-keyed, XDG): `<XDG_STATE_HOME>/fab/operator/<server-slug>.yaml`, where the base is `$XDG_STATE_HOME` (when set and absolute) else `$HOME/.local/state` — uniform on Linux and macOS (never `~/Library/...`). `<server-slug>` is derived from the tmux socket path (`#{socket_path}`) by escaping literal `-` to `--` then mapping separators to a single `-` (e.g. `/tmp/tmux-1000/default` → `tmp-tmux--1000-default`); the escape keeps the mapping collision-free so distinct sockets never share a state file. One operator-per-tmux-server gets one state file that survives a server restart (same `-L` label → same socket path). Falls back to slug `default` when tmux can't be queried. No migration of old repo-rooted `.fab-operator.yaml` files — they are abandoned in place. The file path and its slug rule are a cross-repo contract: run-kit mirrors the exact slug rule to locate the file for display (◉ watched rows, `⚠ operator stale`), pinned in run-kit's operator-cron spec — renaming the file or changing the slug rule requires a coordinated run-kit change. Adding a top-level field is additive under the tolerant-read contract and needs no coordinated run-kit change — the contract is the file path and slug rule, not the field set. The same holds for `pane_pid` and `change`, new keys **inside** the owned `scope` mapping of `tracked` items: a typed write within the owned section, additive under the tolerant-read contract; the file path and slug rule are unchanged.

**Shared state-verb mechanics** (apply to every `fab operator` state verb below): the same server-keyed path derivation; atomic temp+rename writes; a tolerant-read/typed-write posture — unknown **top-level** keys survive any read-modify-write, while the owned sections (`tracked`, `branch_map`, `clock_override`, plus the `tick_count`/`last_tick_at`/`last_full_at` scalars) are re-marshaled from typed structs on mutation, so an invented field inside an owned section can neither be introduced nor survive a mutation of that section. All timestamps (`added_at`, `updated_at`, `checked_at`, `last_tick_at`, `last_full_at`, the override's `until`) are computed by the binary (RFC3339 UTC) — no verb accepts a timestamp flag. Stage-valued flags validate against the six stage names. Every validation failure exits non-zero with a one-line error and no state written.

**Clock side effect** (apply to every `fab operator` state verb below): each verb evaluates the **tracked predicate** — any `tracked` item whose state is not `done` (a list holding only `done` items counts as untracked) — before and after its mutation, and only when the boolean flips, after the mutated state has been saved, issues exactly one rk call: tracked→untracked ⇒ `rk cron mute <id>` (indefinite mute); untracked→tracked ⇒ `rk cron mute <id> --off` (clears both a mute and a lease). After any successful save the verb then runs the **schedule reconcile**: the schedule is derived from the tracked set —

| Tracked set (items not `done`) | Derived schedule | Derived deliver |
|---|---|---|
| empty | *(muted — no edit)* | — |
| only `pane`/`none` items | `--backoff --min 1m --max 30m` | `immediate` |
| any `shell`/`agent` item, and the operator pane has an agent-state epoch | `--idle-every <min(check_every) over the shell+agent items>` | `skip-if-busy` |
| any `shell`/`agent` item, no epoch | `--every <min(check_every)>` | `skip-if-busy` |

— compared against the structured `schedule`/`deliver` fields of the resolved `rk cron list --json` entry (durations compare as durations, so `2m0s` equals `2m`), and exactly one `rk cron edit <id> <schedule flags> --deliver <policy>` is issued **only when they differ**. A live `clock_override` (written by `track clock --for`) applies instead of the derived value until its `until` passes; an expired override is removed from the file in the next mutation's own atomic write and the derived schedule resumes. The epoch signal is the operator pane's `rk mux panes --json` row carrying a non-null `agent_state` (the operator pane is the pane whose window carries `@rk_win_role=operator`, else the current `$TMUX_PANE`). A muted or leased entry is still edited — the new schedule takes effect when the lease expires; the lease itself is untouched. The entry id is resolved per call from `rk cron list --json` — the row whose `target` is `"role:operator"`, with `name == "operator tick"` as the tiebreak; zero candidates, an unresolved tie, or unparseable output is a silent no-op. Every rk call is `exec.LookPath`-gated, argv-only (never a shell string, no `-L` — rk's own `$TMUX` derivation addresses the server), bounded by a 5s timeout, and fail-silent: rk absent, a non-zero exit, a timeout, or a parse failure never changes the verb's exit code, stdout, or the already-saved state. `tick-start --diff` disables the edge trigger and runs the level-wise form instead — after its baseline write it issues one indefinite mute when the post-tick state is untracked (skipping an already-muted entry), then the same schedule reconcile — so one tick never issues two mutes. The skill-side lease policy is owned by `fab-operator.md` §4 Mute and Lease.

**Legacy conversion** (applies to every `fab operator` verb): a legacy-shaped state file — any of `monitored`/`watches`/`autopilot`/`notes` present with `tracked` absent — converts on the first read-modify-write by any verb (including `tick-start`; the read verbs `state` and `track list` convert-and-save, then read), landing in the **same atomic write** as the verb's own mutation: `monitored.<id>` → `pane` items (scope from the entry fields, `scope.change` seeded from the id, `checked_at` = `last_transition`); `watches.<name>` → `linear`/`slack` items with `probe: agent`, `probe.instruction` composed from `source` + `query`, `scope.{repo, stop_stage, query}`, `seen` = `known ∪ completed` (200-capped), `then` = `instructions`, `check_every: 5m`, a disabled watch converting `paused: true`; `autopilot.queue` entries not yet in `completed` → pane-less `pane` items chained by `depends_on` (nearest same-repo predecessor; a cross-repo entry chains to its immediate predecessor) with `scope.merge_mode` = the queue's `mode`; open `notes` → `kind: note` items keeping their `n<N>` ids; resolved notes are dropped. The legacy keys (including `notes_seq`) are deleted by the conversion; a file with `tracked` already present is never re-converted — a stray legacy key then survives as an unknown top-level key. A second, idempotent pass fires on any verb's read-modify-write when a `tracked`-present file still holds an item carrying the retired 2.25.x name of the pane kind (the literal retired name is spelled out in the 2.25.1 → 2.26.0 migration file): the item is rewritten to `kind: pane` with `scope.change` seeded from the item's id (that kind's item id was the change id by construction) and `scope.pane_pid: null` — same atomic write, and a second run is a no-op. **Refusal**: when `autopilot.state == running`, the verb exits non-zero with `operator state file has a running autopilot queue — finish or stop it (fab operator autopilot stop on fab ≤2.24) before upgrading` and writes nothing. The user-facing walkthroughs are the 2.24.9 → 2.25.0 and 2.25.1 → 2.26.0 migration files (under `$(fab kit-path)/migrations/`); they perform no file edits themselves — the binary owns the conversion.

### fab operator state

```
fab operator state [--all] [--json]
```

Prints the server-keyed state file — YAML verbatim by default, JSON conversion with `--json`. When the file is missing it first persists the empty skeleton (`tracked: []`, `branch_map: {}`), then prints it — the binary owns the "create if missing" init step, and a pure read of an existing file never rewrites it (a legacy-shaped file converts first — see **Legacy conversion** above). In human mode (no `--json`) an OPEN NOTES header — one `# `-prefixed comment line per `kind: note` item (`id · note · age-from-updated_at · first line of text`, a note older than 14 days carrying a display-only `⚠ <age>` staleness flag) — prints before the dump, keeping stdout parseable YAML for yq consumers; the header is omitted when there are no note items and never appears in `--json` output. `--all` is a deprecated no-op (the notes section is gone; note items always print).

### fab operator track

```
fab operator track add <id> --kind <kind> [--probe <mode>]
    [--argv <tok> [--argv <tok> …]] [--fields <a,b>] [--instruction <text>]
    [--check-every <dur>] [--done-when <pred>] [--then <text>] [--depends-on <id,id,…>]
    [--scope '<json-object>'] [--text <note-text>] [--mode <merge-mode>]
    [pane sugar: --pane --repo --session --branch --stage --agent --stop-stage --spawned-by --change]
fab operator track update <id> [--check-every <dur>] [--done-when <pred>] [--then <text>]
    [--depends-on <id,id,…>] [--scope '<json-object>'] [--text <t>] [--pause | --resume]
fab operator track observe <id> (--json '<object>' [--seen <item-id>]… | --error <msg>)
fab operator track rm <id>
fab operator track list [--kind <k>] [--json]
fab operator track clock (--every <dur> | --idle-every <dur>) --for <dur> | --off
```

One verb family over the state file's `tracked` list — the replacement for the removed `enroll`/`update`/`remove`, `watch *`, `autopilot *`, and `note *` verbs (removed outright, **no aliases**; legacy state files auto-convert — see **Legacy conversion** above). The binary owns the schema, the timestamps, and the caps; the operator states intent through flags and never hand-writes the YAML.

**Kinds and defaults** — `--kind` is required; the kind fills its defaults at `add` and flags override them:

| Kind | Default probe | Default `done_when` | Notes |
|---|---|---|---|
| `pane` | `pane` | null — built-in when `scope.change` is set (review-pr done/skipped, or at/past `scope.stop_stage`); never done on its own when null | `--pane` records `scope.pane_pid` (the pane's shell-pid fingerprint) in the same mutation, null on lookup failure; `--change` pre-declares the expected change (existence not validated); with branch+repo set, the same mutation writes `branch_map[<change or id>] = {branch, repo}` |
| `github-pr` | `shell`: `gh pr view <n> [--repo <owner/repo>] --json state,mergedAt,mergeable`, fields `state, mergedAt, mergeable` | `state == "MERGED"` | Requires `scope.pr` (or explicit `--argv`); `--repo` is derived from `scope.repo` via `gh repo view` at add time when reachable, else omitted (gh infers from cwd); `--argv`/`--fields` override the default probe per part |
| `linear` / `slack` | `agent` (`--instruction` names the MCP call) | null (standing item — removed by the user) | Carries a binary-capped `seen` list (200 entries, oldest pruned) fed by `observe --seen` |
| `shell` | `shell` (`--argv` and `--fields` required) | required at add | Generic: deploys, CI runs, any JSON-emitting command |
| `task` | `none` | null | A held action with `depends_on` and `then` but nothing to probe — the operator runs `then` when the deps are done |
| `note` | `none` | null | `--text` required (500-char cap); rendered in the frame and the `state` OPEN NOTES header |

**Probe modes** — each kind allows exactly its default mode (`--probe` naming any other mode, or an unknown mode, exits non-zero):

- `pane` — the binary snapshots the pane every tick and joins on `scope.pane`; `check_every` is ignored (forced null).
- `shell` — the binary runs `probe.argv` on the cadence (see the tick-start probe runner above): argv tokens, never a shell string; only `probe.fields` are extracted and stored into `last`.
- `agent` — the LLM runs `probe.instruction` when the tick lists the item under `needs_check:` and records the result with `track observe`; the binary never runs it.
- `none` — never probed (`note`, `task`); `check_every` forced null.

**`--done-when` grammar**: one or more clauses joined by ` and `; each clause is `<path> <op> <literal>` where `<path>` is a field name or dotted path (a leading `.` is accepted and stripped), `<op>` ∈ `==`, `!=`, and `<literal>` is a JSON scalar (`"MERGED"`, `42`, `true`, `null`). A path absent from `last` compares as `null`. No `or`, no comparison operators, no functions, no regex — a malformed predicate exits non-zero naming the offending clause. The binary evaluates it at read time against `last` (level-triggered: no verdict is stored).

**`--check-every`**: a Go duration with a **1m floor** (below → exit non-zero) and a **5m default** for `shell`/`agent` items when omitted.

- **`add`** creates the item with binary-set `added_at`/`updated_at`, `last: {}`, `depends_on: []`, `failures: 0`, `paused: false`. Duplicate id → exit non-zero (`tracked item <id> already exists`). `--scope` takes a JSON object (invalid JSON or a non-object errors). For `--kind pane` the scope starts from eleven pinned keys (`pane, pane_pid, change, repo, session, branch, stage, agent, stop_stage, spawned_by, merge_mode` — null until set), the sugar flags write straight into it (an explicitly-empty value nulls the key; `--stage`/`--stop-stage` validate against the six stage names; `--change` writes `scope.change` — non-empty, existence not validated; setting `--pane` records the pane's shell pid in `scope.pane_pid`, null on lookup failure), and when both branch and repo resolve non-empty the same mutation writes `branch_map[<change or id>] = {branch, repo}`; the sugar flags and `--mode` on any other kind exit non-zero (`--<flag> applies only to kind pane`). A **chained** pane item (added with `--depends-on` or `--mode`) resolves its merge mode by the ladder `--mode` flag > config `autopilot.merge_mode` (the key keeps its historical name) > built-in `cherry-pick-ladder`, stamps it into `scope.merge_mode`, and prints `mode: <name> (<source>)` with source `flag`/`config`/`default`; an unknown `--mode` value — or an invalid config value, which errors naming the key — exits non-zero with no state written. Every `--depends-on` id must already exist and not name the item itself. `--text` is note-only.
- **`update`** mutates only the passed fields of an existing item and refreshes `updated_at`: `--check-every` (floor applies; nulled for pane/none items), `--done-when` (re-validated; empty string clears to null), `--then` (empty string clears), `--depends-on` (replaces the list; ids re-validated), `--scope` (**merged per key** over the existing scope; a value carrying `pane` re-records `pane_pid`, and clearing `pane` nulls it), `--text` (note-only; the 500-cap applies), `--pause`/`--resume` (mutually exclusive; `--resume` also zeroes `failures` — the ack that clears a paused-at-cap `probe_error`). Unknown id → exit non-zero (`no tracked item <id>`).
- **`observe`** records an agent-probe result: exactly one of `--json` / `--error` (both or neither → exit non-zero). `--json` must decode to a JSON object; the declared `probe.fields` (or the whole object when `fields` is empty) are stored into `last` — an equal payload leaves `last` untouched and increments `unchanged`, a differing payload replaces `last` and resets `unchanged` to 0. `checked_at` and `updated_at` move on every observe; `failures` resets to 0. Repeated `--seen` flags append handled item ids to `seen` (deduped; the 200-entry cap prunes oldest first). `--error <msg>` instead increments `failures`; the third consecutive failure sets `paused: true`.
- **`rm`** deletes the item — the ack for `done` and the pane deltas. Removing a **done** item drops its id from every other item's `depends_on` in the same write (a satisfied edge is inert — this is what lets a chain advance after the ack); removing a not-done item keeps the edges, so its dependents stay `held` and the tick names the missing id (`probe_error: unknown dependency <id>`) until `track update --depends-on` repairs them. A pane item's `branch_map` entry is **retained** (downstream dependency resolution needs it; the explicit clear is `branch-map rm`). Unknown id → exit non-zero.
- **`list`** prints one line per item — `id · kind · state · checked-age · next` — or the items array with `--json`; `--kind` filters to one kind (an unknown kind errors). The state is derived from `done_when` and the stored fields only — no pane snapshot (a pane item with a pane reads `live`; the tick's join is the authoritative liveness check), so a dependent of a pane dep that completed via the built-in predicate still shows `held` in `list` until the next tick. Checked-age renders `—` for held items, `live` for pane items, the age since `checked_at` for probed items, and the age since `updated_at` for notes.
- **`clock`** writes the bounded cadence override — top-level `clock_override: { schedule: { kind: every|idle-every, every: <dur> }, deliver: skip-if-busy, until: <RFC3339> }` — which the schedule reconcile applies instead of the derived value until `until` passes (the expired override is then removed and the derived schedule resumes). `--for` is **required** — an override without it exits non-zero (unbounded overrides are forbidden; use `--off` to clear); `--for` must be positive; the cadence obeys the 1m floor; exactly one of `--every`/`--idle-every`. `track clock --off` deletes the override early and takes no other flags.

All six verbs carry the operator-clock side effect (the mute/unmute flip plus the schedule reconcile) — see the shared **Clock side effect** paragraph above.

### fab operator branch-map rm

```
fab operator branch-map rm <change-id>
fab operator branch-map rm --all
```

Entries are *written* by `track add` (branch+repo) or at first change observation, and retained by `track rm`; this verb is the documented "user explicitly clears them" path — one entry, or the whole map with `--all`. Unknown change-id → exit non-zero.

### fab operator time

```
fab operator time [--interval <duration>]
```

Pure time query (no writes).

- Without `--interval`: `now: HH:MM`
- With `--interval 3m`: `now: HH:MM\nnext: HH:MM` (now + interval)

Duration is Go format (`3m`, `5m`, `2m`). Invalid → exit 1.

---

## fab agent

```
fab agent [role|stage] [--provider <name>] [--model <id>] [--effort <level>] [--headless] [-t|--template] [-o yaml] [--workers <provider>] [-p|--print] [--repo <path>] [-- <agent-args>...]
```

Launch (or `--print`) the profile-resolved agent **session** command in the current shell, with model and effort substituted.

Two **addressing forms** compose the command, plus their combination:

- **Selector-addressed** (the `[role|stage]` positional) — resolves the role profile (`default` when the positional is omitted; the six role names — `default`, `operator`, `doing`, `review`, `hydrate`, `fast` — plus the six stage names — `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr` — are all accepted; a stage name maps through the fixed `stageRoles` table to its role, and the `review`/`hydrate` collisions are fixed points, so either name resolves identically), then composes `providers.<profile.provider>.interactive_command` with the role's `{model}`/`{effort}` substituted (or Claude-style `--model`/`--effort` appended for a non-templated command) via `internal/spawn.WithProfile` — the same substitution the YAML `command`/`dispatch.command` values and the operator launcher use. Which provider the role lands on is the role's depth knob (`agent.session` for `default`/`operator`, `agent.workers` for the rest) unless `agent.profiles.<role>.provider` names one.
- **Provider-addressed** (`--provider <name>`, no selector) — **bypasses role resolution entirely**: looks up `providers.<name>` directly (project config per-field merged over fab-kit's built-in provider table, exactly as the selector path's provider lookup does) and composes its `interactive_command` with the `--model`/`--effort` values through the same `WithProfile`. This is the "spawn a codex session right here" form — no knob or role need name the provider first, and no `providers:` block either: `codex`, `agy`, and `kimi` are **built-in** providers, so each works on a fresh project. It works for every built-in and for any user-defined provider that has an `interactive_command`.
- **Selector + `--provider`** — re-resolves the selector's role from the named provider's own fills (`agent.ResolveRoleWith` with the provider pinned): the provider pin refills from `providers.<p>.profiles.<role>`, then that provider's `.default`, then empty — while an explicit `agent.profiles.<role>.model` pin still wins. So `fab agent apply --provider kimi --print` prints kimi's own command shape with kimi's fills, not claude's model patched in.

Common to all forms:

- **Default (exec)**: replaces this process with the composed command via `sh -c` (so shell expansions like `$(basename "$(pwd)")` expand at invocation). `fab agent` starts the default-role agent right here; `fab agent operator` starts the coordinator profile. **No TTY guard** — exec-and-let-the-agent-CLI-handle-it (document-don't-validate).
- **`-p, --print`**: prints the fully-resolved command instead of executing. Lets the operator compose a worker spawn from a real profile. `-p` is a pure shorthand — byte-identical to `--print`.
- **`-t, --template`**: prints the selected provider's command template **unsubstituted** — a tap BEFORE the fill step, `{model}`/`{effort}` placeholders intact. Implies print. Combines with any selector, `--provider`, and `--headless` (they pick which template). `--model`/`--effort` are rejected with a usage error — they feed the substitution step that `-t` skips.
- **`-o, --output yaml`**: prints the full resolution as structured YAML. The original seven keys keep their order and the remaining keys follow:

  | Key | Semantics |
  |-----|-----------|
  | `selector` | Requested role or stage; empty for a bare-provider invocation. |
  | `kind` | `role`, `stage`, or `provider`. |
  | `role` | Resolved role; empty for `kind: provider`. |
  | `provider` | Resolved provider name. |
  | `model` | Full model ID; empty preserves the provider-inherit signal. |
  | `effort` | Resolved effort; empty when inherited or unsupported. |
  | `command` | Selected interactive or headless template after profile substitution and passthrough composition. |
  | `model_alias` | Agent-tool short alias for a Claude model; empty for non-Claude IDs. |
  | `template` | Raw selected provider command slot with placeholders intact. |
  | `fill_mode` | `template` when placeholders are substituted, otherwise `append`. |
  | `source` | Nested `provider`, `model`, and `effort` provenance using the actual precedence-rung names; an empty value denotes inherit/no supplying rung. |
  | `dispatch` | Omitted exactly when the native rung is selected; otherwise contains labelled `rung: pane\|headless` and its fully substituted `command`. |
  | `skill_prefix` | Explicit-skill-invocation prefix for the resolved provider: `$` for `codex`, `/` for every other provider (built-in, custom, or unknown). Always present and always the last key (`dispatch` is omitted for native). Skill consumers compose `<skill_prefix><skill>[ <args>]`; `_cli-agents.md` § Skill Prompts owns receiver selection and quoting, `internal/agent.SkillPrefix` owns the rule. |

  YAML output alone derives dispatch. It implies print, accepts only `yaml`, and is mutually exclusive with `--print` and `-t`; `--print`, exec, and `-t` remain unchanged.
- **`--headless`**: resolves the provider's `headless_command` instead of `interactive_command`. Valid only with the three print sinks (`--print`, `-t`, `-o yaml`) — exec of a headless command is a usage error. A provider with no `headless_command` hard-errors naming the config key (`configure providers.<name>.headless_command`).
- **`--model` / `--effort`** are **general final overrides**, valid with every addressing form (bare, selector, selector+`--provider`, provider-alone): they apply verbatim AFTER role resolution / provider refill, keyed on the flag being SUPPLIED (cobra's `Flag.Changed`), so an explicitly-empty `--model=` CLEARS the field rather than being ignored. No validation, no enum, no pair correction — verbatim pass-through. This matches `fab resolve-agent`'s bare-override semantics; the former launcher-side restriction is erased.
- **`--repo <path>`**: reads `<path>/fab/project/config.yaml` instead of the current repo. Composes with any addressing form.

**Passthrough args (`-- <agent-args>...`)**: everything after `--` is forwarded to the launched agent CLI, so `fab agent -- --resume` runs the resolved claude command with `--resume` appended, and `fab agent doing -- -p "two words"` does the same on the `doing` role's provider. The split is cobra's `ArgsLenAtDash()` — the arg BEFORE the dash is still the role, and arity is validated on that half only (`fab agent a b -- x` errors; so does `fab agent a b`, with a hint pointing at the `--` form). Flags after the dash belong to the agent CLI, never to fab: `fab agent -- --repo /x` forwards `--repo /x` rather than re-pointing fab's config lookup. Each token is **shell-quoted** on the way in, because the composed line is executed via `sh -c` and printed verbatim by `--print` — that is what keeps `-p "two words"` one argument, and what makes a token containing `$(…)`, `;` or `>` inert data instead of script. Passthrough applies to BOTH addressing modes (role and `--provider`), and an invocation with no `--` is byte-identical to before.

**Runs OUTSIDE a fab project.** `fab agent` is a **config-only** command — it resolves a `{provider, model, effort}` and launches a session; it never reads or writes change state. So when no `fab/` directory exists anywhere up the tree it does **not** fail closed: it degrades to the **project-free cascade**, `env > system (~/.fab-kit/config.yaml) > built-in defaults`, dropping only the project tier. A machine-wide preference therefore takes effect in any directory, which is what makes `fab agent` usable as a plain session launcher (`eval "$(fab agent --print)"` from anywhere). `Config.FabVersion` stays empty outside a project — `.fab-version` is a per-project pin — and preflight's staleness check already skips an empty value. `fab resolve-agent` and `fab config show` degrade the same way. System-targeted config writers (`init/set/unset/upgrade --system`) are repo-free too; bare/project writes and `upgrade --all` still require a fab root.
- **`--workers <provider>`**: sets `FAB_AGENT_WORKERS=<provider>` in the exec environment for the launched session, replacing any value inherited from the parent environment rather than appending a second entry. It is pure pass-through launch sugar: no provider lookup or validation, and `--print` remains exactly the resolved session command with no assignment added.

Provider-form specifics (the bare `--provider <name>` form — selector forms' overrides are the "Common to all forms" bullets above):

- **Omitted `--model`/`--effort`** leave the value empty, which follows the existing `WithProfile` empty-value rule: in **template** mode the placeholder's whitespace-delimited token is dropped along with a preceding `-`-flag; in **append** mode the flag is simply not appended. So `fab agent --provider codex --print` against `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` prints `codex --dangerously-bypass-approvals-and-sandbox`: the installed CLI's own default model applies while fixed, non-profile flags remain. This is how you spawn a provider whose current model IDs you do not know.
- **The bare provider form deliberately bypasses the fills**: `fab agent --provider <name>` does NOT read `providers.<name>.profiles` — the provider form's profile is exactly the flags. The fill rungs apply to *selector* resolution (including selector-with-`--provider`, which re-resolves them) and to `fab resolve-agent` (see its § Fill precedence); here an omitted value stays empty and the CLI's own default applies.
- **Unknown provider name** → non-zero exit listing the **available** provider names (the project's `providers:` keys ∪ fab-kit's built-in table, sorted). This is a **lookup** failure, not validation of the command's content — resolved command strings still pass through verbatim (document-don't-validate; fab never infers a provider from a model string).
- **Error**: a resolved provider with no `interactive_command` errors with a config-key hint (`configure providers.<name>.interactive_command`) on either path — reachable for a wholly user-defined, headless-only `providers:` entry. Every built-in ships one. A partial built-in override cannot reach this error: `ResolveProvider` per-field merges over the built-in row, so `providers.codex: {headless_command: …}` still inherits the built-in `interactive_command` and still launches a session. An unknown selector errors naming both valid sets (roles and stages).

The procedural knowledge for *using* the composed command — opening it in a tmux window, delivering a prompt reliably, peeking, awaiting — plus the per-provider invocation grammar and model-discovery recipes live in the `_cli-agents` helper (`helpers: [_cli-agents]`).

---

