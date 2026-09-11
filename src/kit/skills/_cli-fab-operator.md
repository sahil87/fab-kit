---
name: _cli-fab-operator
description: "Fab CLI reference — the `fab operator` and `fab agent` command families (the tmux operator's state/tick/autopilot verbs; the stage/role agent-resolution query). Split out of _cli-fab so operator consumers load only this slice."
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

Called at start of each operator tick. Increments `tick_count`, writes `last_tick_at` (ISO 8601 UTC) to the **server-keyed** state file (not the old repo-rooted `.fab-operator.yaml`). The flagless form is unchanged — stdout:

```
tick: N
now: HH:MM
```

**`--diff`** additionally snapshots the fleet internally (the same pane-map discovery+resolve pipeline `fab pane map` runs — one enumeration path), diffs it against the monitored baseline, and emits one stdout document: the `tick:`/`now:` header lines, then three YAML blocks in this order:

```yaml
deltas:
    - kind: completion            # completion | pane_death | pane_mismatch | agent_exited | stage_advance | review_fail
      change: r3m7
      pane: "%3"
      # kind-specific fields:
      #   completion    → stage, display_state
      #   pane_death    → (none)
      #   pane_mismatch → found   (the change ID occupying the pane; null when none resolvable)
      #   agent_exited  → command (shell foreground with no live agent in the pane tree, e.g. zsh)
      #   stage_advance → from, to
      #   review_fail   → from (review), to (apply)
candidates:
    - pane: "%7"
      change: k8ds
      agent_state: waiting        # waiting | idle
      idle_duration: null         # non-null only for idle (upstream idle-only semantics)
fleet:
    - change: r3m7
      pane: "%3"
      repo: /home/user/code/foo
      session: work
      stage: review-pr
      display_state: done
      agent_state: idle           # active | waiting | idle | null (unknown)
      idle_duration: 8m           # null unless idle
      pr_url: https://github.com/acme/foo/pull/412   # null when none
fleet: []                          # empty-list form when nothing monitored (likewise deltas/candidates)
```

- **Two delivery classes.** `completion` / `pane_death` / `pane_mismatch` / `agent_exited` are **level-triggered**: stateless predicates over the current snapshot, re-emitted every tick until acted on — `fab operator remove` is the ack, so a crash between diff and action loses nothing. `stage_advance` / `review_fail` are **consumed-on-read** (baseline-diffed, consumed by the same-write baseline update); a lost one costs a missed report only. `review→apply` is the rework reset path and emits `review_fail`, not `stage_advance`.
- **Detection semantics.** `completion` is a display-state/terminal-stage predicate (never a stage diff): with `stop_stage: null` it fires only at the pipeline terminus — `review-pr` with `display_state` done/skipped (hydrate and ship are mid-pipeline and never complete an entry by themselves; a run that deliberately parks earlier expresses that via `stop_stage`); with a `stop_stage` it fires past the stop in stage order, or at the stop with `display_state` done/skipped. `pane_mismatch` fires when the entry's pane now resolves to a **different** change ID (or none — `found: null`): tmux recycles `%N` pane IDs across server restarts, so a recycled pane is never diffed, baseline-updated, or listed as a candidate (its fleet row falls back to baseline identity fields with null observed fields). `agent_exited` fires only when the entry's pane IS present and change-matched, its **foreground command is a shell** (basename ∈ `sh bash zsh fish dash ksh tcsh csh nu`), AND its pane-PID process tree contains no live agent. The shell value triggers a lazy current-socket PID lookup and tree walk; agent names are `claude`/`claude-code` plus every merged provider `interactive_command`'s quote-aware leading-command-word basename after true POSIX `NAME=value` prefixes, with shell names excluded. Matching checks `comm` first and at most the first two cmdline tokens. Only positive live-agent evidence suppresses the delta; PID or walk failure silently fails toward emitting. Agent state is never liveness evidence — its option can retain the agent's last stale value after exit. An `agent_exited` pane gets the same exclusions as `pane_mismatch` (no baseline write, no stage diffs, excluded from `candidates:`, baseline fleet row with null agent_state). Evaluation order per entry is `pane_death` → `pane_mismatch` → `agent_exited` → clean join: a mismatched pane hosting a shell emits only `pane_mismatch`, without a tree walk.
- **`candidates:`** — monitored entries whose snapshot agent state is waiting or idle, waiting first then idle, sorted by change ID within each class; unknown (`—`) and active panes are excluded, so on rk-less servers the block is empty. This is the operator §5 sweep population.
- **`fleet:`** — one row per monitored entry, ordered repo → session → enrolled_at → change ID: the status frame's data source, so the skill never re-fetches the full pane map per tick.
- **`--quiet`** — valid only with `--diff` (`--quiet` alone errors `--quiet requires --diff` before any state read/write, consuming no tick). On a **quiet tick** — `deltas:` empty AND the post-increment `tick_count` not a multiple of the built-in constant 10 (not a flag or config knob) — the `fleet:` block is **replaced** by a five-count `fleet_summary:` mapping (never both keys; block order stays `deltas`, `candidates`, then one of the two):

  ```yaml
  fleet_summary:
      tracked: 8      # one per monitored entry
      waiting: 1      # snapshot agent_state waiting
      idle: 3         # snapshot agent_state idle
      active: 3       # snapshot agent_state active
      unknown: 1      # null/empty/em-dash agent_state
  ```

  `tracked` always equals `waiting + idle + active + unknown` on a quiet tick (dead/mismatched/exited panes emit level-triggered deltas, which force the full document). A tick with non-empty `deltas:`, and every 10th tick, emits the full document — identical to plain `--diff`. `candidates:` is always emitted. The empty-monitored short-circuit still skips the snapshot; under `--quiet` it emits the all-zero `fleet_summary:` (or `fleet: []` on a 10th tick).
- **Baseline writer.** `--diff` updates the baseline in the **same atomic mutation** as the tick bookkeeping: for each cleanly-joined entry, `stage` ← snapshot stage (touching `last_transition` **iff** the stage changed — `fab operator update`'s semantics) and `agent` ← the snapshot agent state verbatim. Dead/mismatched entries stay untouched; an unresolved (em-dash) snapshot stage fabricates no delta and leaves the baseline stage alone. With an **empty monitored set** the snapshot subprocess is skipped entirely and all three blocks emit `[]` — a no-op tick is first-class.
- **Clock reconcile.** `--diff` also mutes the operator-tick cron entry when the post-tick state is untracked — the shared **Clock side effect** paragraph below.

**State path** (server-keyed, XDG): `<XDG_STATE_HOME>/fab/operator/<server-slug>.yaml`, where the base is `$XDG_STATE_HOME` (when set and absolute) else `$HOME/.local/state` — uniform on Linux and macOS (never `~/Library/...`). `<server-slug>` is derived from the tmux socket path (`#{socket_path}`) by escaping literal `-` to `--` then mapping separators to a single `-` (e.g. `/tmp/tmux-1000/default` → `tmp-tmux--1000-default`); the escape keeps the mapping collision-free so distinct sockets never share a state file. One operator-per-tmux-server gets one state file that survives a server restart (same `-L` label → same socket path). Falls back to slug `default` when tmux can't be queried. No migration of old repo-rooted `.fab-operator.yaml` files — they are abandoned in place. The file path and its slug rule are a cross-repo contract: run-kit mirrors the exact slug rule to locate the file for display (◉ watched rows, `⚠ operator stale`), pinned in run-kit's operator-cron spec — renaming the file or changing the slug rule requires a coordinated run-kit change.

**Shared state-verb mechanics** (apply to every `fab operator` state verb below): the same server-keyed path derivation; atomic temp+rename writes; a tolerant-read/typed-write posture — unknown **top-level** keys survive any read-modify-write, while the five owned sections (`monitored`, `autopilot`, `branch_map`, `watches`, `notes`) are re-marshaled from typed structs on mutation, so an invented field inside an owned section can neither be introduced nor survive a mutation of that section. All timestamps (`enrolled_at`, `last_transition`, `last_checked`, `created_at`, `updated_at`, `resolved_at`) are computed by the binary (RFC3339 UTC) — no verb accepts a timestamp flag. Stage-valued flags validate against the six stage names; unknown change-ids/watch-names/note-ids and no-active-queue calls exit non-zero with a one-line error. Schema is byte-compatible with the `fab-operator.md` §4 shape; no migration.

**Clock side effect** (apply to every `fab operator` state verb below): each verb evaluates the **tracked predicate** — `monitored` non-empty, an `autopilot` block with non-null `state` (an exhausted block with `state: null` counts as empty), any `watches` entry (enabled or disabled), or any unresolved `kind: coordination` note — before and after its mutation, and only when the boolean flips, after the mutated state has been saved, issues exactly one rk call: tracked→untracked ⇒ `rk cron mute <id>` (indefinite mute); untracked→tracked ⇒ `rk cron mute <id> --off` (clears both a mute and a lease). The entry id is resolved per call from `rk cron list --json` — the row whose `target` is `"role:operator"`, with `name == "operator tick"` as the tiebreak; zero candidates, an unresolved tie, or unparseable output is a silent no-op. Every rk call is `exec.LookPath`-gated, argv-only (never a shell string, no `-L` — rk's own `$TMUX` derivation addresses the server), bounded by a 5s timeout, and fail-silent: rk absent, a non-zero exit, a timeout, or a parse failure never changes the verb's exit code, stdout, or the already-saved state. `tick-start --diff` adds a reconcile: after its baseline write it issues one indefinite mute when the post-tick state is untracked, skipping an already-muted entry. The skill-side lease policy is owned by `fab-operator.md` §4 Mute and Lease.

### fab operator state

```
fab operator state [--all] [--json]
```

Prints the server-keyed state file — YAML verbatim by default, JSON conversion with `--json`. When the file is missing it first persists the empty skeleton (`monitored: {}`, `autopilot: null`, `branch_map: {}`, `watches: {}`, `notes: []`), then prints it — the binary owns the "create if missing" init step, and a pure read of an existing file never rewrites it. In human mode (no `--json`) an OPEN NOTES header — one `# `-prefixed comment line per open note (`id · kind · age · first line of text`) — prints before the dump, keeping stdout parseable YAML for yq consumers; the header is omitted when there are no open notes and never appears in `--json` output. Resolved notes are excluded from the printed `notes:` list unless `--all` (the exclusion re-marshals the parsed state; with nothing to filter the raw bytes print verbatim).

### fab operator enroll / update / remove

```
fab operator enroll <change-id> --pane <pane-id> --repo <abs-path> --session <name> --branch <branch> \
    [--stage <stage>] [--agent <state>] [--stop-stage <stage>] [--spawned-by <watch>] [--depends-on <id,id,...>]
fab operator update <change-id> [--stage <stage>] [--agent <state>] [--stop-stage <stage>]
fab operator remove <change-id>
```

- `enroll` creates (or wholesale-replaces, with a fresh `enrolled_at`) the monitored entry, sets `enrolled_at` + `last_transition` to now, defaults `stop_stage: null` / `spawned_by: null` / `depends_on: []`, **and** records the `branch_map` entry `{ branch, repo }` — enrollment is the documented moment `branch_map` gains its pair, so one command owns both writes. `--pane`, `--repo`, `--session`, `--branch` are required. The `»` window-name rename stays a separate `fab pane window-name ensure-prefix` call.
- `update` mutates only the passed fields of an existing entry, touching `last_transition` **iff** `--stage` changes the stored value. `--agent` passes through verbatim (fab is a consumer of run-kit's `@rk_pane_agent_state` convention and does not enumerate its states). `--stop-stage ""` clears to null. Unknown change-id → exit non-zero.
- `remove` deletes the monitored entry and **retains** the `branch_map` entry (the documented persistence policy — downstream dependency resolution needs it; the explicit clear is `branch-map rm`). Unknown change-id → exit non-zero. The `»`→`›` rename stays a separate `fab pane window-name replace-prefix` call.
- All three verbs carry the operator-clock mute/unmute side effect — see the shared **Clock side effect** paragraph above.

### fab operator note

```
fab operator note add --kind <dependency_wait|phase_plan|coordination|correction> [--ref <r>]... <text>
fab operator note resolve <id>
fab operator note update <id> <text>
fab operator note list [--open|--all] [--json]
```

- Notes are the operator's narrative-state surface (routing doctrine: `fab-operator.md` §4 Notes). Each note carries `{ id, kind, text, refs?, created_at, updated_at, resolved, resolved_at }` in a top-level `notes:` **list** (creation order); ids are `n<N>` from the persisted top-level `notes_seq` counter that only increments — **ids are never reused after prune**. Unknown top-level keys (e.g. a legacy hand-written `plan_queue:`) survive every verb.
- `add` creates the note with binary-set timestamps and `resolved: false`, then prints the id to stdout. Unknown `--kind` or text over the **500-character cap** → exit 1 with a one-line error and no state written. Repeated `--ref` flags accumulate into `refs`. Duplicate text is allowed (dedupe is operator judgment).
- `resolve` sets `resolved: true` + `resolved_at` (re-resolving is **idempotent** — exit 0, no field changes), then prunes **resolved** notes past a **50-entry cap, oldest-first in list order**; open notes are never pruned. Unknown id → exit 1.
- `update` replaces the text in place and refreshes `updated_at` (age/staleness render from it); the 500-cap applies. Unknown id → exit 1.
- `list` defaults to `--open` (open notes only; `--all` includes resolved). Human output is one line per note: `id · kind · age-from-updated_at · first line of text`; a note older than **14 days** carries a display-only `⚠ <age>` staleness flag, and more than **25** open notes produces a warning on **stderr** (stdout stays clean). `--json` emits the filtered notes as JSON with no decoration.
- `note add --kind coordination` and `note resolve` carry the operator-clock mute/unmute side effect — see the shared **Clock side effect** paragraph above.

### fab operator watch

```
fab operator watch add <name> --source <linear|slack> --target-repo <abs-path> \
    [--query <json>] [--stop-stage <stage>] [--instructions <text>]
fab operator watch rm <name>
fab operator watch toggle <name> [--on|--off]
fab operator watch update <name> [--target-repo <path>] [--stop-stage <stage>] [--instructions <text>] [--query <json>]
fab operator watch checked <name> [--error <msg>]
fab operator watch seen <name> <item-id>
fab operator watch complete <name> <item-id>
```

- `add` creates the watch with `enabled: true`, empty `known`/`completed`, null `last_checked`/`last_error`. `--query` takes a JSON object string (nested lists/maps like `{"status":["Backlog","Todo"]}` need it) stored as the YAML `query` map; invalid JSON or a non-object → exit non-zero. Duplicate name → exit non-zero.
- `rm` deletes the watch; `toggle` flips `enabled` (or forces with `--on`/`--off`, mutually exclusive); `update` mutates only the passed fields (`--stop-stage ""` clears to null). Unknown name → exit non-zero.
- `checked` sets `last_checked` to now and sets `last_error` to `--error` (flag present) or clears it to null (flag absent) — the per-tick query bookkeeping.
- `seen` appends the item to `known` (idempotent — no duplicates) and enforces the **200-entry cap, oldest pruned first, in the binary**.
- `complete` moves the item from `known` to `completed` (an item absent from `known` is still added to `completed` — a late completion is never lost — and never duplicated).
- `watch add` and `watch rm` carry the operator-clock mute/unmute side effect — see the shared **Clock side effect** paragraph above.

### fab operator autopilot

```
fab operator autopilot start --queue <id,id,...> [--mode <cherry-pick-ladder|merge-auto|stacked-prs>]
fab operator autopilot pause
fab operator autopilot resume
fab operator autopilot advance [--skip]
fab operator autopilot stop
```

- `start` resolves the merge mode by the ladder **`--mode` flag (explicitly passed) > config `autopilot.merge_mode` > built-in default `cherry-pick-ladder`**, then sets `{ queue, current: <first>, completed: [], state: running, mode: <resolved> }` and prints `mode: <name> (<source>)` where source is `flag` / `config` / `default`. Flag-absent is detected via flag-changed state, so an explicit `--mode cherry-pick-ladder` still reports source `flag`. The config lookup works from a neutral (fab-less) cwd: with no fab project up the tree, the system tier (`~/.fab-kit/config.yaml`) and env (`FAB_AUTOPILOT_MERGE_MODE`) still compose; a config load error fails soft to the built-in default. An unknown `--mode` value exits non-zero with a one-line error naming the valid modes and writes no state; an invalid **config** value errors the same way but names the key — `invalid autopilot.merge_mode "<value>" in config (valid: cherry-pick-ladder, merge-auto, stacked-prs)` — with no state written (merging is destructive-tier, so there is no silent fallback to a different topology than the one configured).
- `pause`/`resume` flip `state` between `paused`/`running`; `mode` is retained.
- `advance` appends `current` to `completed` (unless `--skip`) and promotes the next queue entry; on exhaustion it sets `current: null, state: null` **while retaining `queue`/`completed`/`mode`** so the queue-completion summary can still read them.
- `stop` clears the whole block to `autopilot: null`.
- An `autopilot` block lacking `mode` (a pre-existing state file) reads as the built-in default `cherry-pick-ladder` — never config-resolved (the mode was fixed at queue start and persisted); the typed re-marshal writes the field on the next mutation. The binary only stores/validates/prints the mode (via `fab operator state`) — all merge choreography stays in `fab-operator.md` prose.
- Verbs other than `start` exit non-zero when no queue is active (`stop` tolerates an exhausted-but-retained block).
- `autopilot start`/`stop` and `advance`-to-exhaustion carry the operator-clock mute/unmute side effect — see the shared **Clock side effect** paragraph above.

### fab operator branch-map rm

```
fab operator branch-map rm <change-id>
fab operator branch-map rm --all
```

Entries are *written* by `enroll` and retained by `remove`; this verb is the documented "user explicitly clears them" path — one entry, or the whole map with `--all`. Unknown change-id → exit non-zero.

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

