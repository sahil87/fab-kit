---
type: memory
description: "`fab dispatch {start,restart,status,wait,logs,kill,reap,clean}` — process manager for CLI-dispatched pipeline stages in two launch modes, resolved per invocation by a ladder ending in auto (`$TMUX` ⇒ pane, else headless): headless (detached `sh -c` on `dispatch_command`, five states) and pane (`session_command` in a tmux pane, stacked in a column carved at `dispatch.column_width`; three states). `restart` relaunches the persisted prompt; `status`/`wait` observe; `reap` reclaims a done pane."
---
# fab dispatch

**Domain**: runtime

## Overview

`fab dispatch` is the **process manager for CLI-dispatched pipeline stages**, in **two launch modes** that share one state directory, one loader, one concurrency check, one launch path, and one state-string vocabulary:

| Mode | Selection signals (resolved by the § Mode selection ladder, in precedence order) | Worker | Command composed | Completion observed via | tmux |
|------|----------------------------------------------------------------------------------|--------|------------------|-------------------------|------|
| **headless** | `--headless`, `--timeout`, or auto **outside** tmux | a detached `sh -c` process | the provider's `dispatch_command` | `{stage}.exit` + pid liveness + result file | never touched |
| **pane** | `--pane`, `--server`, or auto **inside** tmux | an interactive agent session in a tmux pane the user can watch and steer — **split into the dispatching agent's own window**, or a **new window** when there is no pane to split | the provider's `session_command` | **result file** + pane liveness | **required** — as is a `session_command`; either prerequisite missing is a hard error when explicit, a soft fallback to headless when auto |

Headless mode is **tmux-independent**: it launches the resolved command (persisted in the record as `spawn_cmd`) detached, tracks it via a repo-root state dir, and exposes poll/logs/kill/clean surfaces. Pane mode exists to recover *watch and steer* on the CLI path — a detached headless worker is a black box no one can talk to, whereas an interactive tmux worker is the one universal interface every agent CLI supports natively. **Neither mode is an unconditional default**: `start` resolves the mode per invocation through an explicit-first ladder whose last rung is **auto** — pane when `$TMUX` is set (the run-kit context, where a pane is visible to the caller), headless when it is not (see § Mode selection). Together they are the **two non-native adapters** of the three-adapter cross-harness dispatch catalog (native Agent-tool / headless CLI / interactive pane), whose protocol is fixed by the human-curated spec `docs/specs/harness-adapters.md`. `restart` is the family's **recovery** verb across both modes: it relaunches a non-running dispatch from the prompt `start` persisted, so the caller needs no prompt in context, and it re-derives the mode from the *current* environment rather than the prior attempt's. `status` and `wait` are its two **observation** verbs over a single derivation — `status` the one-shot probe, `wait` its blocking sibling that returns the moment the state leaves `running` — so an orchestrator is *woken by* a state change rather than asking every 30s whether one happened.

Dispatch is the runtime for **cross-harness stage dispatch** ("a codex orchestrator runs `apply` on claude"): a fundamentally launch-and-poll problem, not a pane-observation one. It *runs* the provider command that `fab resolve-agent` names (a role names a provider; the provider carries the commands — see [_shared/configuration.md](/_shared/configuration.md) § `providers` and [runtime/providers-and-profiles.md](/runtime/providers-and-profiles.md)). `fab dispatch` re-resolves the stage itself, so which mode a dispatch lands in is decided by the ladder below and never by the emitted `dispatch=` line — including a line emitted under the `dispatch.watchable` opt-in, where the emitted `session_command` is exactly what pane mode composes anyway (see [runtime/providers-and-profiles.md](/runtime/providers-and-profiles.md) § Watchable pane dispatch). Headless dispatch stays parallel to and independent of the tmux-bound interactive `fab pane` / `fab operator` family (see [pane-commands.md](/runtime/pane-commands.md) and [operator.md](/runtime/operator.md)); pane dispatch **borrows tmux as a launch surface** but does not join the operator's monitored set, and consumes the `_cli-agents` helper's spawn/deliver/peek procedures as its primitive layer (see [agent-primitives.md](/runtime/agent-primitives.md)).

The **skill wiring** consumes it: the dispatch-seam skills branch on the resolved `dispatch=` line and, when present, drive this command family — `fab dispatch start` (block prompt on stdin, no mode flag by default — the worker's mode auto-resolves, and `--pane`/`--headless` are passed only to force one) → a **blocking `fab dispatch wait <change> <stage> --timeout 300`**, run as a *background* command wherever the harness can re-invoke the agent on exit (foreground blocking is the cross-harness fallback) → the mode's reachable states → read `{stage}-result.yaml` on `done`, then `fab dispatch reap` to reclaim a finished pane worker's pane. The wiring is **push, not poll**: the orchestrator spends turns only when the wait returns, and a `running` return (the bound expired) is its peek-on-suspicion moment, which is why `--timeout 300` is a *peek cadence* rather than a poll interval. It lives in `_preamble.md` § CLI-Adapter Dispatch + § Dispatch-Prompt Obligations, where **pane mode is an option inside the `dispatch=`-present arm, never a third branch** (see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership and [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution).

Source: the testable core lives in `internal/dispatch` (state read/write, wrapper composition, both state derivations, the `Wait` control loop in `wait.go`, the reap guard in `reap.go`, process signaling, and the tmux pane primitives in `pane_mode.go`); thin cobra wiring lives across `cmd/fab/dispatch.go` (parent) + `dispatch_start.go` / `dispatch_restart.go` / `dispatch_status.go` / `dispatch_wait.go` / `dispatch_logs.go` / `dispatch_kill.go` / `dispatch_reap.go` / `dispatch_clean.go` — mirroring the `internal/pane` + `pane*.go` split precedent. `dispatch_start.go` owns the **shared launch path** (`runDispatchLaunch` + the `promptSource` seam) and the **shared flag surface** (`addLaunchFlags` binding a `launchFlags` struct, plus its `resolveMode` method carrying the `--pane`+`--timeout` guard, the `SelectMode` call, and the `SelectPaneShape` call — the one place `$TMUX` and `$TMUX_PANE` are read) that both `start` and `restart` run; `restart` adds only its cobra command — its own help strings and its `promptFromStateDir` source.

## Requirements

### Requirement: `fab dispatch` command family

The `fab` binary SHALL expose a top-level command group `fab dispatch` with eight subcommands — `start`, `restart`, `status`, `wait`, `logs`, `kill`, `reap`, `clean` — always-routed through the `fab` router. Its top-level name MUST NOT collide with the `fab-kit` `LifecycleCommands` allowlist (pinned by `TestNoTopLevelCommandCollidesWithRouterAllowlist`; `dispatch` is not in the allowlist). It is a new fab-go command group registered via `dispatchCmd()` in `cmd/fab/main.go`'s `newRootCmd()`. See [distribution/kit-architecture.md](/distribution/kit-architecture.md) for its place in the fab-go command inventory.

### Requirement: POSIX-only v1 (the headless launch/signal syscalls)

The **headless** `fab dispatch start` (and `kill`) SHALL error clearly on non-POSIX platforms rather than half-working — the message names the POSIX-shell requirement (`setsid`/`timeout`). The guard is a **compile-time platform split**, not a runtime `runtime.GOOS` string check: `dispatch_posix.go` (build tag `!windows`) owns the launch/signal syscalls; `dispatch_windows.go` (build tag `windows`) provides the same signatures returning the POSIX-only error (with `Alive` conservatively `false`), so the package compiles on Windows and the error surfaces at the command layer. This mirrors the `proc_{linux,darwin}.go` / `pane_process_{linux,darwin}.go` precedent.

Pane mode's tmux primitives (`ServerReachable`, `OpenWindow`, `OpenSplitPane`, `SiblingDispatchPane`/`SplitTarget`, `PaneAlive`, `KillPane`) live in the platform-**independent** core (`internal/dispatch/pane_mode.go`) for the same reason `WrapperArgv` does: they are plain tmux subprocess calls with no syscall dependency, so they compile everywhere. Pane mode is still unusable where tmux is absent, but that surfaces as `ServerReachable`'s actionable error, not as a compile-time platform split.

#### Scenario: Windows build errors instead of launching

- **GIVEN** a `GOOS=windows` build
- **WHEN** headless `fab dispatch start` is invoked
- **THEN** it returns an error naming the POSIX-shell requirement and launches nothing

### Requirement: `.fab-dispatch/{id}/` state layout

Each dispatch's state SHALL live under `.fab-dispatch/{4-char-change-id}/` at the **repository root** (`filepath.Dir(fabRoot)`), keyed by the stable 4-char change ID (not the slug, so it survives `fab change rename`). This sits alongside the `.fab-status.yaml` repo-root ephemeral-state convention (the `.fab-runtime.yaml` sibling that convention once also named is gone (ioku) — agent-state production is divested — see [runtime-agents.md](/runtime/runtime-agents.md)), and each git worktree naturally gets its own dir. **No gitignore/scaffold/migration work is required** — the scaffold `fragment-.gitignore` `.fab-*` pattern already matches `.fab-dispatch/`. The dir name is the `internal/dispatch` named constant `DirName = ".fab-dispatch"`; per-stage filenames derive from named suffix constants (no magic strings). **Both modes share the dir**, the loader, the save path, and the refuse-if-running check.

Per-stage files under `.fab-dispatch/{id}/`:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start` (from stdin) | the stage prompt — piped to the dispatched command's stdin (headless) or **pointed at** by the one-line prompt the pane worker receives (pane). Written identically in both modes; only the hand-over differs. It is also `restart`'s **input**: `restart` reads it and leaves it byte-identical, never re-writing it |
| `{stage}.yaml` | `start` (via `internal/atomicfile`) | the `Dispatch` state struct — `spawn_cmd` (resolved) + `started_at`, plus the mode's identity: `pid`/`pgid`/`timeout` (headless) or `pane`/`window`/`server` (pane). `window` holds the `fab-{id}-{stage}` **identity string** in both pane shapes, so its meaning is *"tmux window name (new-window shape) or tmux pane title (split shape)"* — the string is identical either way, which is why it stayed one field with no schema change and no migration. Every mode-specific key is `omitempty`, and the mode is **derived** from which keys are present (no stored discriminator). File paths are **derived** from the dir convention, not stored |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command — **headless only** (a pane worker's output is tmux scrollback) |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its **presence** is the "process finished" signal; **headless only** |
| `{stage}-result.yaml` | the **dispatched agent** (contract) | the stage result; dispatch defines only the path + consumes its presence. Presence is required for `done` in both modes and is the **sole** completion signal in pane mode. Its **content schema** is a minimal YAML envelope mirroring each native block's return — common `stage`/`status`/`summary`; apply adds `failed_task`/`reason` on failure; review adds `verdict` (pass\|fail) + `findings{must_fix,should_fix,nice_to_have}`; hydrate carries only the envelope. The **`status` (worker/infra outcome) vs `verdict` (review outcome) split is load-bearing** — a completed review with `verdict: fail` is dispatch-state `done` (result present), and the orchestrator then takes the normal review-fail path; dispatch-state `failed` is reserved for worker/infrastructure failure. Schema documented in `_preamble.md` § Dispatch-Prompt Obligations. |

#### Scenario: a headless record's on-disk shape is unchanged by pane mode's fields

- **GIVEN** a headless dispatch
- **WHEN** `{stage}.yaml` is serialized
- **THEN** it carries `pid`/`pgid`/`spawn_cmd`/`started_at` and **none** of `pane`/`window`/`server`
- **AND** GIVEN a pane dispatch, the record carries `pane`/`window`/(`server` when set) and no `pid`/`pgid` — the addition is additive on disk, so no migration exists or is needed

### Requirement: `fab dispatch start <change> <stage> [--timeout <secs>] [--pane] [--headless] [--server <name>]`

`start` SHALL run a **shared prologue** followed by one of **two mode-specific launch tails**. The prologue resolves `<change>` to its 4-char ID (via `internal/resolve` — ID / folder substring / full name), loads config, resolves the stage's role → provider profile via `internal/agent` + `internal/spawn.WithProfile` (the same `{model}`/`{effort}` substitution `fab resolve-agent` performs), enforces refuse-if-running, obtains the stage prompt through a **`promptSource` seam** (`start`: stdin, persisted into `{stage}-prompt.md`), and clears stale per-stage files. The tail launches the worker and persists `{stage}.yaml` before returning. The prologue SHALL NOT be duplicated across the tails — nor across the two subcommands that run it (`start` and `restart` share it via `runDispatchLaunch`).

Output names the mode's identity — and, in pane mode, its **shape**: `dispatched <id>/<stage> (pid N, pgid N)` (headless), `dispatched <id>/<stage> (pane %N, split, title fab-<id>-<stage>)` (pane, split shape), or `dispatched <id>/<stage> (pane %N, window fab-<id>-<stage>)` (pane, new-window shape — byte-identical to before the split shape existed). When the mode was chosen by **auto**, the report appends the selection **source** — `, auto: tmux` / `, auto: no tmux` / `, auto: tmux unreachable` / `, auto: no session_command` — so a surprising mode is explainable from output alone (compliance visibility); an explicitly-selected mode's line carries no suffix and is byte-identical to before auto selection existed.

### Requirement: Mode selection is an explicit-first ladder ending in auto

`start` SHALL resolve its launch mode per invocation from an **explicit-signal-first ladder**, evaluated in order, the first match winning: `--pane` ⇒ pane; `--headless` ⇒ headless; `--timeout` ⇒ headless (the bound exists only in the headless wrapper); `--server` ⇒ pane (the flag exists solely to target a pane's socket); otherwise **auto** — `$TMUX` non-empty ⇒ pane, else headless. Each explicit rung SHALL key on whether the flag was **supplied** (`Flags().Changed`), not its value, so `--timeout 0` / `--server ""` still count as signals. An **empty** `$TMUX` reads as unset (Go cannot distinguish the two, and tmux never exports an empty value).

The ladder SHALL be a **pure function** — `dispatch.SelectMode(paneFlag, headlessFlag, timeoutSet, serverSet bool, tmuxEnv string) (Mode, AutoReason)` in `internal/dispatch/pane_mode.go` — performing no environment read and no tmux probe, so the whole matrix is table-testable without launching a process or a tmux server (matching `DeriveState`/`DerivePaneState`'s shape in the same package). It is called once per invocation from `launchFlags.resolveMode` — the shared flag-surface helper both `start`'s and `restart`'s cobra `RunE` run — which hands the resolved mode (never the raw `--pane` bool) to `runDispatchLaunch`, whence it reaches the command composition, the reachability probe, and the launch branch.

`$TMUX` is the **defaulting** signal only — it means "a pane opened without `-L` lands on the server the caller is attached to". It never replaces `ServerReachable`, which stays the **validation** step once pane mode is chosen and is a real tmux query precisely so an explicit `--pane` works from a headless orchestrator where `$TMUX` is unset. An auto-selected pane targets the **current** server: no `-L` is passed unless `--server` was given.

`$TMUX_PANE` is a **separate signal that does not enter this ladder at all** — it selects the pane *shape* (split vs. new window) only after pane mode has already been chosen (see § A pane worker splits the dispatching agent's window). Keeping the two decisions apart is what lets `SelectMode`'s matrix stay unchanged by the split shape.

#### Scenario: auto resolves from the environment

- **GIVEN** no mode-related flag
- **WHEN** `fab dispatch start <change> <stage>` runs with `$TMUX` set
- **THEN** the mode resolves to pane against the current server, and the output line carries `auto: tmux`
- **AND** with `$TMUX` unset (or empty), the mode resolves to headless — byte-identical to the pre-auto default apart from the `auto: no tmux` suffix

#### Scenario: an explicit signal always beats auto

- **GIVEN** `$TMUX` set (auto would select pane)
- **WHEN** `--headless` or `--timeout N` is supplied
- **THEN** the mode resolves to headless with no `auto:` suffix

### Requirement: `--headless` is the explicit opt-out; `--pane` + `--headless` is a usage error

A `--headless` boolean flag SHALL exist as the explicit opt-out from auto pane selection — the escape hatch for an unattended run that happens to live inside a tmux tab. `--pane` + `--headless` SHALL be a **usage error** (exit 2) enforced by cobra's `MarkFlagsMutuallyExclusive`, which fires during `ValidateFlagGroups` **before any `RunE` work**, so it structurally cannot leave partial state: nothing is launched and no dispatch record is written. `--headless` + `--timeout` SHALL **compose** (both select headless).

#### Scenario: contradictory mode flags are rejected before anything happens

- **GIVEN** `--pane --headless`
- **WHEN** `fab dispatch start` runs
- **THEN** it exits non-zero naming both flags, launches nothing, and writes no dispatch record

**Headless tail — detach mechanism — `SysProcAttr{Setsid:true}` on a plain `sh -c`, NOT the `setsid` binary.** The launch runs the wrapper `sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'` via `exec.Command` with `SysProcAttr{Setsid:true}` — Go's syscall attribute puts the child in a **new session/process group** so the dispatch survives the orchestrator dying, with no Go supervisor process in the loop (the shell records the exit code itself, so resumability falls out: a resumed skill reattaches via `fab dispatch status` instead of re-running the stage). The recorded `pid`/`pgid` therefore track the **live worker shell**. The intake's `setsid sh -c` string described the *intent* (new session, survives orchestrator death); the `SysProcAttr` attribute delivers that intent while keeping the tracked pid on the worker (see Design Decisions — an end-to-end smoke test showed the `setsid` **binary** double-forks, leaving the Go-recorded pid pointing at an immediately-exiting process and breaking liveness/refuse-if-running/kill). `WrapperArgv` is therefore always `[sh -c <script>]` with **no `setsid` prefix**.

**Timeout is enforced entirely inside the wrapper** via POSIX `timeout N <cmd>` when `--timeout N` is given — self-contained, no Go timer, no background sweep, no daemon. A timed-out command exits `124` (POSIX `timeout` convention), which surfaces as `failed` via the normal exit-code path.

#### Scenario: detached launch persists tracked state

- **GIVEN** a change/stage whose resolved role's provider carries a `dispatch_command`
- **WHEN** `fab dispatch start <change> <stage>` runs with a prompt on stdin
- **THEN** the prompt is persisted, the command is launched detached in a new session/process group, and `{stage}.yaml` records the pid/pgid/spawn_cmd/started_at
- **AND** with `--timeout N`, the resolved command is wrapped in POSIX `timeout N` inside the same `sh -c` wrapper

**`--pane` tail — an interactive tmux pane.** With `--pane`, `start` SHALL compose the resolved provider's **`session_command`** (the same string `fab agent` composes) and open it in a tmux **pane** whose cwd is the repo root, persisting the pane's **pane ID**, the `fab-{id}-{stage}` identity string, and the tmux socket label in `{stage}.yaml`. (WHERE that pane opens — split into the dispatching agent's own window, or a new window — is a second decision; see § A pane worker splits the dispatching agent's window.) The composed command is passed as the creating verb's shell-command argument, so shell expansions it carries (e.g. `$(basename "$(pwd)")` in the built-in claude `session_command`) expand at invocation inside the new pane — the `_cli-agents.md` § Spawn Composition contract. The **pane ID**, not the identity string, is the recorded identity: it is server-global, stable for the pane's lifetime, and exempt from tmux's target-grammar prefix/glob resolution, so liveness probes and kills are exact where a name-based target could resolve to a window the user renamed into place. `-P -F '#{pane_id}'` on the creating verb prints it, avoiding a follow-up lookup that could race a fast-exiting worker.

`--server <name>` / `-L <name>` targets a tmux socket (`tmux -L <name>`), mirroring the `fab pane` family's persistent flag, and is persisted so `status`/`kill` reach the same server without re-supplying it. It **implies pane mode** under auto (naming a socket while meaning headless is incoherent) and is **ignored in headless mode** (headless touches no tmux).

#### Scenario: pane launch persists pane identity

- **GIVEN** a change/stage whose resolved role's provider carries a `session_command`, and a reachable tmux server
- **WHEN** `fab dispatch start <change> <stage> --pane` runs with a prompt on stdin
- **THEN** a tmux pane runs the composed interactive command, `{stage}.yaml` records `pane`/`window`/(`server`)/`spawn_cmd`/`started_at`, and no `{stage}.exit` wrapper is involved

### Requirement: A pane worker splits the dispatching agent's window; a new window is the fallback

**WHERE** the worker pane opens SHALL be a second decision, independent of the mode ladder and made only after pane mode is chosen. It SHALL be resolved by the **pure** `dispatch.SelectPaneShape(paneMode, serverSet, tmuxPane)` — no environment reads, no tmux probe, no I/O, matching `SelectMode`/`DerivePaneState`'s shape in the package — with both env reads (`$TMUX`, `$TMUX_PANE`) performed in the cobra layer and passed down:

| # | Condition | Shape | tmux call | Identity carried by |
|---|-----------|-------|-----------|---------------------|
| 1 | `$TMUX_PANE` non-empty **and** no `--server` | **split** — a pane inside the **dispatching agent's own window** | `tmux split-window {-h -l <n>%\|-v} -t <target> -P -F '#{pane_id}' -c <repo-root> "<resolved-cmd> <shell-quoted-pointer>"`, then `tmux select-pane -t <new-pane> -T fab-{id}-{stage}` | the tmux **pane title** |
| 2 | `--server <name>` supplied | **new window** | `tmux -L <name> new-window -n fab-{id}-{stage} -P -F '#{pane_id}' -c <repo-root> "…"` | the tmux **window name** |
| 3 | `$TMUX_PANE` empty | **new window** | `tmux new-window -n fab-{id}-{stage} -P -F '#{pane_id}' -c <repo-root> "…"` | the tmux **window name** |

This realizes the **two-tier tmux hierarchy**: an **operator** opens worktree agents as tmux **windows** (that path is untouched), and each **worktree agent**'s stage workers appear as **panes beside it** — so a stage worker does not consume a window in the operator's (and run-kit's) window list. Rows 2 and 3 reproduce the pre-split behavior **byte-identically**, which is what makes the change additive: `--server` may name a socket other than the one the caller's pane lives on, where the caller's `$TMUX_PANE` id is meaningless (pane ids are server-global, not global); an empty `$TMUX_PANE` means the dispatcher — a headless orchestrator passing an explicit `--pane` — has no pane of its own to split.

**The identity string is shape-independent.** `WindowName(id, stage)` composes the same `fab-{id}-{stage}` string for both shapes and it is stored in the record's same `window` field, so there is **no schema change and no migration**. In the split shape it rides the **pane title** (`select-pane -T`), because a split pane has no window name of its own — its window is the dispatcher's. A **failed title set is non-fatal** (a stderr warning at most): the worker is already running and its pane ID — the real identity — is already in hand, so refusing the dispatch over a cosmetic label would be strictly worse.

**Everything downstream is pane-ID keyed and therefore shape-blind.** `PaneAlive`, `KillPane`, `fab pane capture`, refuse-if-running, and `DerivePaneState` all target the pane ID and needed **no change**. Killing a split worker's pane leaves the dispatching agent's window — and any sibling worker — intact, by plain tmux `kill-pane` semantics.

`restart` inherits the whole decision from the shared launch path (`resolveMode` resolves mode *and* shape), so it needs no restart-specific branch: a restart issued from inside a tmux pane splits that pane's window even when the prior attempt had opened a window.

#### Scenario: a dispatching agent's worker lands in its own window

- **GIVEN** a dispatching process whose `$TMUX_PANE` names a live pane, and no `--server`
- **WHEN** `fab dispatch start <change> <stage>` auto-selects pane mode
- **THEN** the worker's pane shares the dispatcher's `#{window_id}`, no new window is created, the pane's title is `fab-{id}-{stage}`, and the report reads `pane %N, split, title fab-{id}-{stage}`

#### Scenario: `--server` and an absent `$TMUX_PANE` keep the new-window shape

- **GIVEN** either `--server <name>` supplied (whatever `$TMUX_PANE` holds) or an empty `$TMUX_PANE`
- **WHEN** a pane dispatch runs
- **THEN** a tmux **window** named `fab-{id}-{stage}` is created and the report reads `pane %N, window fab-{id}-{stage}` — byte-identical to before the split shape existed

#### Scenario: killing a split worker leaves the dispatcher's window intact

- **GIVEN** a split worker pane in the dispatching agent's window
- **WHEN** `fab dispatch kill <change> <stage>` runs
- **THEN** only that pane dies; the dispatcher's pane, its window, and any sibling worker pane all survive, and the dispatch reads `orphaned`

### Requirement: Split placement is a record-keyed stacked column, carved once at `dispatch.column_width`

**Sibling detection SHALL key on dispatch RECORDS, never on pane titles.** `start` collects the `pane:` field of every `.fab-dispatch/*/{stage}.yaml` record in the checkout whose `server:` **equals the socket being probed**, intersects that set with `tmux list-panes -t "$TMUX_PANE" -F '#{pane_id}'` (a `-t` pane target resolves to that pane's window), and keeps the **last** pane present in both — list-panes order is pane-index order, so the last match is the newest worker. A pane ID is the correct identity for the same reason `status`/`kill`/`capture` key on it: it is server-global and stable for the pane's lifetime. A pane **title** is not — a harness running inside the worker pane rewrites it via terminal escapes within seconds of spawn — so titles are **set** at spawn for identification only, and **no code path reads `#{pane_title}` for placement**. `{stage}-result.yaml` is not a record, and records with an empty `pane:` (every headless dispatch) contribute nothing.

The **server filter is exact equality**, because a pane ID is per-**socket** rather than global: a `%17` recorded by a `--server work` dispatch names a different pane from the `%17` on the default socket, so an unfiltered set could stack a worker onto an unrelated pane. Default-socket dispatches record `server: ""` and are matched by a default-socket probe under that same equality test — no special case. Enumeration scope is **every** record dir in the checkout, not only the active change's, since nothing stops one window from hosting two changes' workers; over-collecting is safe because the intersection with one window's live pane list **is** the liveness *and* same-window filter — a dead pane, or a live pane in another window, simply never matches, so no separate `PaneAlive` probe or window lookup is needed.

**The column invariant.** The first worker splits the dispatcher's own pane `-h`, **carving** the Left/Right column at `dispatch.column_width` percent (default 35, so the agent the user is watching keeps the rest); every later worker splits the last live recorded worker `-v`, stacking **inside** that column, unsized. Only the carving split is ever sized — sizing a stacking split would fight the user's own resizes within the column. fab issues **no `select-layout`**, never rearranges user-made panes, and never re-touches the vertical Left/Right separator once carved. This is a **creation-time rule, not an enforcement loop**: an already-mangled window is left alone until its panes die.

The decision is a `SplitPlacement{Target, Direction, SizePercent}` returned by `SplitTarget(server, dispatcherPane, repoRoot, columnWidth)`, whose pure halves — `lastRecordedPane` (the intersection), `splitPlacement` (the decision), `splitArgs` (the argv) — are table-testable without a tmux server or a record tree, matching `SelectMode`/`DerivePaneState`'s shape in the package. The width is read from config in the cobra layer (`cfg.GetDispatchColumnWidth()`; see [_shared/configuration.md](/_shared/configuration.md) § `dispatch`) and rides the placement, so the "size the carve, never a stack" rule exists in exactly one place.

`SplitPlacement` and `SplitTarget` are the package's whole exported placement surface: the tmux flags (`splitRight`/`splitBelow`/`sizeFlag`), the argv composer (`splitArgs`), and the sibling probe are package-scope, and the cobra layer reads a placement only through `SplitPlacement.Describe()` — the stacked-column wording its degraded-probe warning prints — so no caller outside `internal/dispatch` handles a raw `-h`/`-v`.

**Placement is cosmetic, so every failure degrades warn-only** and never fails a dispatch that would otherwise launch. Each warning names both what failed and where the worker actually landed (`worker-column placement probe failed (…); carving a new worker column off pane %N` / `… stacking the worker under pane %N`):

| Failure | Outcome |
|---------|---------|
| `tmux list-panes` fails | no window to intersect ⇒ the **sized** carve off the dispatcher (a fallback column that halved the dispatcher would reintroduce the squeeze the width exists to prevent) |
| an unreadable dispatch dir / corrupt `{stage}.yaml` | the records that **did** read are kept, so a partial failure still stacks when a live sibling is among them — an unread record can only fail to *find* a sibling, never invent one |
| an absent `.fab-dispatch/` tree | the ordinary first-dispatch case, **not** a failure: empty set, no warning |
| tmux rejects `-l <n>%` (every tmux before 3.1, or a window too narrow) | the identical split is retried **unsized**; the first failure becomes the warning |
| `select-pane -T` fails | non-fatal, as for the new-window shape's label |

One consequence is deliberate: when the **dispatcher is itself a pane worker** (a stage worker dispatching a stage of its own), its own pane is in the record set, so it can be its own sibling and the new worker stacks under it rather than carving a second column. That is the wanted outcome — the dispatcher already lives in a worker column — and it is self-limiting, since the next dispatch finds the child below it and stacks under that.

`restart` reaches this placement through the shared launch path with no restart-specific branch, so a relaunched worker stacks in the right column exactly as a fresh `start` would.

#### Scenario: a clobbered pane title does not misplace the next worker

- **GIVEN** a live worker pane recorded in `.fab-dispatch/abcd/apply.yaml` whose pane **title has been rewritten** by the harness running inside it
- **WHEN** a second pane dispatch starts from the same dispatcher pane
- **THEN** the probe still finds that worker, the new pane splits it `-v` (not the dispatcher), and all three panes share one window
- **AND** GIVEN a recorded pane that is dead, or live in another window, or recorded against a different `server:`, it never matches and the dispatch carves a fresh sized column instead

#### Scenario: the carving split is sized and the stacking split is not

- **GIVEN** `dispatch.column_width` resolving to 35
- **WHEN** the first pane worker of a window is dispatched
- **THEN** the split argv carries `-h -l 35%` and the dispatcher keeps ~65% of the window width
- **AND** GIVEN a second worker stacking under it, that argv carries `-v` and no `-l`
- **AND** GIVEN a tmux that rejects the size, the split is retried unsized, the worker still launches, and stderr carries the one-line warning

### Requirement: Prompt delivery is a file plus a one-line pointer in pane mode

In **both** modes the full stage prompt arrives on **stdin** and is persisted to `{stage}-prompt.md`. Headless mode pipes that file into the dispatched command's stdin; pane mode SHALL instead hand the worker a **one-line pointer** naming the repo-relative prompt path, embedded at spawn as the interactive command's single prompt argument. The prompt **content** is composed identically for every adapter — nothing about the block prompt is written differently for `--pane`; only the hand-over differs. No `send-keys` delivery and no printed-prompt probe is required for the initial delivery.

**The pointer SHALL be shell-quoted; the resolved command SHALL stay verbatim.** The two halves of the pane command are quoted differently on purpose. The pointer names a *repo-derived* path, so a checkout under a directory containing a single quote (`/home/me/sahil's-repo/…`) would terminate a naively-single-quoted argument early — breaking the `new-window`/`split-window` command and handing the path's remainder to the new pane's shell. It therefore rides through the package's `shellQuote` (the `'\''` idiom the headless wrapper's paths already use), composed in one place by `dispatch.WindowCommand` for **both** pane shapes, honoring `_cli-agents.md` § Spawn Composition's "shell-escape any user-supplied text before embedding it". The resolved `session_command` is inserted **verbatim** — its shell expansions are deliberate and must expand inside the new pane (per the pass-through philosophy: the command's own quoting is the resolver's/user's concern).

#### Scenario: a multi-thousand-token prompt reaches an interactive worker

- **GIVEN** a full stage prompt on stdin
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** the full prompt lands in `{stage}-prompt.md` and the window's command carries only the one-line pointer to that path, readable from the window's cwd (the repo root)

#### Scenario: a repo path containing a single quote does not break the window command

- **GIVEN** a repository whose path contains a `'` character, so the repo-relative pointer inherits it
- **WHEN** `fab dispatch start <change> <stage> --pane` composes the window command
- **THEN** the pointer is shell-escaped and parses as exactly one shell word, arriving at the worker byte-identical to the composed pointer

### Requirement: The pane path has two prerequisites — hard error when explicit, soft fallback when auto

Pane mode SHALL require **both** a reachable tmux server **and** a `session_command` on the resolved provider, and **both** failure shapes SHALL be **asymmetric by selection source**. An **explicitly** selected pane (`--pane`, or `--server` acting as the pane signal) SHALL fail with a non-zero exit and an actionable stderr message — launching nothing and persisting **no** dispatch record — because a caller who asked for watchability must not be silently downgraded: an unreachable server yields the reachability error, a missing `session_command` the `providers.<name>.session_command` config-key hint. An **auto**-selected pane SHALL instead **soft-fall-back to headless**, printing the single stderr line for its shape (each a named constant) and then proceeding with a normal headless launch producing a headless-shaped record (`pid`/`pgid`, no `pane`/`window`/`server`):

| Shape | Trigger | stderr notice (constant) | `auto:` reason |
|-------|---------|--------------------------|----------------|
| (a) | `ServerReachable` fails — a stale `$TMUX` inherited from a killed server, or no tmux | `pane auto-selection: tmux unreachable, falling back to headless` (`dispatch.FallbackNotice`) | `auto: tmux unreachable` |
| (b) | tmux answered, but the provider carries no `session_command` (a `dispatch_command`-only provider) | `pane auto-selection: provider has no session_command, falling back to headless` (`dispatch.FallbackNoticeNoSessionCommand`) | `auto: no session_command` |

Neither shape may break a dispatch that never asked for a pane: a stale `$TMUX` must not break an unattended run, and a `dispatch_command`-only provider must dispatch identically inside and outside tmux.

**Pane-command composition is therefore deferred until the mode is validated** (probe passed AND `session_command` present) — composing it first would hard-fail shape (b) before the fallback decision point, turning a previously-working headless dispatch into an error demanding a `session_command` it never needed. Validation writes no state and composes no command, so both outcomes are safe before anything is persisted.

The fallback **re-composes the provider command from `dispatch_command`**, so the no-cross-fallback rule survives the mode change: a provider carrying *neither* field still errors with the usual `dispatch_command` config-key hint, after the notice has explained why headless was attempted. The soft fallback is a **mode** change, not a cross-field fallback.

- **GIVEN** a provider carrying `dispatch_command` but no `session_command`, a **reachable** tmux server via `$TMUX`, and no mode flag
- **WHEN** `fab dispatch start <change> <stage>` runs
- **THEN** stderr carries the shape-(b) notice, the launch is headless from `dispatch_command`, no tmux window is opened, and stdout's `dispatched …` line ends `auto: no session_command`

- **GIVEN** the same provider and an explicit `--pane` against a reachable server
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** it exits non-zero with the `providers.<name>.session_command` hint and persists no dispatch record

Reachability SHALL be established by a **real tmux query** (`tmux [-L <server>] list-sessions` via `ServerReachable`), **not** an `$TMUX` environment read: a dispatching orchestrator may itself be headless — exactly the cross-harness caller pane mode exists for — so an `$TMUX`-only gate would make `--pane` unusable from those callers. This mirrors `fab resolve --pane`, where `--server` likewise replaces the `$TMUX` guard with socket-scoped discovery. In headless mode, **no tmux probe occurs at all**, preserving the tmux-independence guarantee.

The probe distinguishes "a server answered" from "nothing answered", not "has sessions" from "has none": under tmux's default `exit-empty on` a server exits with its last session, so a zero-session server never persists to be probed and `--pane` correctly errors; under `exit-empty off` a sessionless server does persist and the probe passes, and a subsequent `new-window` that tmux cannot satisfy surfaces the child's own stderr via `internal/pane`'s `StderrError`. The probe is a reachability gate, not a launch guarantee — the launch carries its own actionable error.

#### Scenario: unreachable tmux leaves no trace under an explicit `--pane`

- **GIVEN** no reachable tmux server (or an unreachable `--server` socket)
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** it exits non-zero naming tmux reachability, the `--server` option, and the `--headless` alternative, and creates no `{stage}.yaml`

#### Scenario: a stale `$TMUX` degrades to headless instead of failing

- **GIVEN** `$TMUX` set to a dead socket and no mode flag (auto selects pane)
- **WHEN** `fab dispatch start <change> <stage>` runs
- **THEN** stderr carries the one-line fallback notice, the launch is headless, and `{stage}.yaml` records `pid`/`pgid` with no `pane`/`window`/`server`

### Requirement: `--pane` and `--timeout` are mutually exclusive

Supplying both flags SHALL be a usage error (non-zero exit) naming the exclusion, enforced before any launch or file write — never a silently ignored `--timeout`. `--timeout` is implemented as POSIX `timeout N` inside the headless `sh -c` wrapper, which pane mode never constructs. Only the **explicit** `--pane` conflicts: a bare `--timeout` is itself a headless rung of the selection ladder, so `--timeout` inside tmux selects headless rather than erroring — scripted invocations that never mention panes keep working unchanged.

#### Scenario: the exclusion is enforced before anything happens

- **GIVEN** `fab dispatch start <change> <stage> --pane --timeout 600`
- **WHEN** it runs
- **THEN** it exits non-zero naming the `--pane`/`--timeout` exclusion, launching nothing and writing nothing

### Requirement: Missing command field → error, no cross-fallback in either direction

If the resolved role's provider lacks the field the **finally-selected** mode needs, `start` SHALL error clearly — naming the stage, the resolved role, the provider, and the config key (`providers.<name>.dispatch_command` for headless, `providers.<name>.session_command` for pane) — and MUST NOT fall back to the other field. The **no-cross-fallback rule holds in both directions**: headless never falls back to `session_command`, and pane mode never falls back to `dispatch_command`. This is the load-bearing dispatch-mode semantic (a fallback would silently flip a session-command-only provider into headless CLI dispatch, or the reverse); see [_shared/configuration.md](/_shared/configuration.md) § `providers` and [runtime/providers-and-profiles.md](/runtime/providers-and-profiles.md). Pane mode reading the provider table for `session_command` is **not** a fallback and **not** a resolver change: mode selection is per-invocation, `fab resolve-agent`'s output is identical either way, and pane mode reads the table itself exactly as `fab agent` does.

The **auto-only soft fallback is a MODE change, not a cross-field fallback**, and does not weaken this rule: it re-selects headless and re-composes from `dispatch_command`, so a provider carrying neither field still errors with the `dispatch_command` hint (§ the pane path's two prerequisites). Because the mode may change during validation, the error a provider gets names the field of the mode that was actually launched.

#### Scenario: a session-command-only provider dispatches under pane mode and errors without it

- **GIVEN** a role whose provider carries a `session_command` but no `dispatch_command`
- **WHEN** `fab dispatch start <change> <stage> --pane` runs with a reachable tmux server
- **THEN** the dispatch succeeds using the composed `session_command`
- **AND** GIVEN the same role, an explicitly headless `start` (`--headless`, or auto outside tmux) errors with the `dispatch_command` config-key hint
- **AND** GIVEN a provider with no `session_command`, an explicit `--pane` errors with the `providers.<name>.session_command` hint (under **auto** the same provider soft-falls-back to headless instead)

### Requirement: Refuse-if-running + last-attempt-only concurrency

`start` **and `restart`** SHALL refuse if a dispatch for the exact `(change, stage)` pair is already `running` — reporting the live identity (`pid N` or `pane %N`) and directing to `fab dispatch kill` — leaving the running dispatch untouched (they run one shared check, so they cannot diverge). The check SHALL apply the **prior record's own mode's finished signal** — the *same* signal `status` derives that mode's state from, so `start` and `status` can never disagree about whether an attempt is still going:

| Prior record's mode | Still running when | Finished when |
|---|---|---|
| headless | `{stage}.exit` absent **and** pid alive | `{stage}.exit` present (the shell recorded a code) |
| pane | `{stage}-result.yaml` absent **and** pane alive | `{stage}-result.yaml` present — **result presence wins over pane liveness** |

The pane row's result-presence precedence mirrors `DerivePaneState` and is load-bearing: an interactive worker never exits on task completion, it sits at its prompt, so a liveness-only refusal would fire forever after a successful pane run and make a `done` attempt permanently un-overwritable — `status` reporting `done` while `start` insisted it was still running.

A `start` **or `restart`** over a **completed** prior attempt (done / failed / orphaned) SHALL overwrite its files — there is **no per-attempt history** (last-attempt-only: it removes the stale exit/result/log then re-saves `{stage}.yaml`), and the new attempt MAY use either mode regardless of the prior one's. Refuse-if-running is scoped per `(change, stage)`: different stages of the same change share `.fab-dispatch/{id}/` via distinct `{stage}.*` filenames and do not collide.

#### Scenario: refuses a live dispatch, overwrites a completed one

- **GIVEN** a `(change, stage)` dispatch whose pid is alive and `{stage}.exit` is absent
- **WHEN** `fab dispatch start` runs again for the same pair
- **THEN** it refuses with a clear error and leaves the running dispatch untouched
- **AND** GIVEN a completed prior attempt, a new `start` overwrites the prior `{stage}.*` files with no history retained

#### Scenario: a finished-but-still-alive pane worker is overwritable

- **GIVEN** a pane dispatch whose pane is still alive (the worker is sitting at its prompt) and whose `{stage}-result.yaml` is absent
- **WHEN** `fab dispatch start` runs again for the same pair
- **THEN** it refuses — the worker is genuinely still executing
- **AND** GIVEN that worker then writes `{stage}-result.yaml` while its pane remains alive
- **THEN** `status` reports `done` **and** a new `start` overwrites the attempt rather than refusing

### Requirement: `fab dispatch restart <change> <stage> [--timeout <secs>] [--pane] [--headless] [--server <name>]`

`restart` SHALL relaunch a **non-running** dispatch from the prompt `start` persisted at `{stage}-prompt.md`, so the caller does not need the block prompt in context (an orchestrator may have lost a multi-thousand-token prompt to compaction). It is `start` with **exactly one difference — the prompt's source** — because both run the same Go launch path (`runDispatchLaunch`, parameterized by a `promptSource`: `promptFromStdin` for `start`, `promptFromStateDir` for `restart`). Consequently `restart` SHALL carry:

- the **same prologue** — change resolution, config + role→provider re-resolution (**config only**: like `start`, it exposes no `--provider`/`--model`/`--effort`), pane validation, refuse-if-running, and stale `{stage}.exit`/`-result.yaml`/`.log` clearing;
- the **same mode-selection ladder** and the same flags with the same exclusions (`--pane`+`--headless` and `--pane`+`--timeout` are usage errors; `--headless`+`--timeout` composes), including the explicit-hard / auto-soft asymmetry on the pane path's two prerequisites — all single-sourced in the shared `addLaunchFlags` / `launchFlags.resolveMode` helper pair, so the flag surface cannot drift any more than the launch tail can;
- the **same output and record byte-shape** — the `dispatched <id>/<stage> (<report>)` line including the `, auto: …` selection-source suffix rules, and the same `{stage}.yaml` keys.

The launch **mode — and the pane shape — SHALL be re-derived from the current environment**, never inherited from the prior attempt (the record carries no mode or shape discriminator to inherit from), so a restart issued from inside a tmux pane splits that pane's window even when the prior attempt opened a window. A restart **is** a fresh attempt under last-attempt-only, so it SHALL introduce **no new state string, no attempt counter, no attempt history, and no `restarted:` marker** — `status` cannot distinguish a restart from a `start`, by design. The prompt file is the **input**: `restart` reads it and leaves it **byte-identical** (re-writing it with its own bytes would only risk corruption on a partial write), while the *prior attempt's* exit/result/log are still cleared. Refuse-if-running SHALL **precede** the prompt read, and an absent prompt SHALL be a clear error — `no persisted prompt at <path> — nothing to relaunch; run \`fab dispatch start\` with the prompt on stdin` — that launches nothing and leaves any prior record intact.

The observation policy that *spends* a restart (one automatic restart on `orphaned`, peek-on-suspicion, escalation) is **skill-side**, not a CLI concern — see `_preamble.md` § CLI-Adapter Dispatch → *Recovery policy* and [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution.

#### Scenario: an orphaned attempt is relaunched from the persisted prompt

- **GIVEN** an `orphaned` dispatch for `abcd/apply` whose `apply-prompt.md` is on disk
- **WHEN** `fab dispatch restart abcd apply` runs
- **THEN** a new worker is launched with those persisted bytes as its input, the record is overwritten, and the prior attempt's stale exit/result/log are cleared
- **AND** the prompt file itself is left byte-identical

#### Scenario: the mode is re-derived, so a dead tmux server does not re-fail the restart

- **GIVEN** an `orphaned` **pane** dispatch and no reachable tmux server (`$TMUX` unset)
- **WHEN** `fab dispatch restart` runs with no mode flag
- **THEN** auto selects **headless** and the new record is headless-shaped (`pid`/`pgid`, no pane identity), composed from `dispatch_command`
- **AND** the prior pane identity does not leak into the new record

#### Scenario: a live dispatch refuses, even with no prompt on disk

- **GIVEN** a genuinely running dispatch for `abcd/apply`
- **WHEN** `fab dispatch restart abcd apply` runs
- **THEN** it refuses with the `already running (…); run fab dispatch kill first` message and leaves the record untouched
- **AND** it does so even when `apply-prompt.md` is absent — the refusal check precedes the prompt read

#### Scenario: a missing prompt writes nothing

- **GIVEN** no `{stage}-prompt.md` for the pair
- **WHEN** `fab dispatch restart` runs
- **THEN** it errors naming the path and the `fab dispatch start` remedy, launches no worker, and writes or overwrites no `{stage}.yaml`

### Requirement: Five byte-stable states, derived per mode

`fab dispatch status <change> <stage> [--json]` SHALL read `{stage}.yaml` and derive the state by the record's **derived mode**. The five state **strings are the cross-adapter contract** and are identical in both modes; what differs is which are reachable. Each mode's derivation is its own **pure function**, so both are independently table-testable: `DeriveState` (headless five-state) and `DerivePaneState` (pane three-state).

**Headless** — reads `{stage}.exit` and probes `pid` liveness (the POSIX-standard `syscall.Kill(pid, 0)` EPERM/ESRCH probe), reporting one of all five:

| State | Condition | Meaning |
|-------|-----------|---------|
| `running` | pid alive AND `{stage}.exit` absent | still executing |
| `done` | `{stage}.exit` == `0` AND `{stage}-result.yaml` present | finished successfully with a result |
| `failed` | `{stage}.exit` present AND != `0` | non-zero exit (includes `124` timeout) |
| `failed (no-result)` | `{stage}.exit` == `0` BUT `{stage}-result.yaml` absent | **contract violation, NOT done** — exited clean but never wrote its result |
| `orphaned` | pid dead AND `{stage}.exit` absent | reboot / `kill -9` / crash — no exit code was ever recorded |

The `failed (no-result)` state is the crux: a clean exit is necessary but **not sufficient** for `done`; the result file must exist. This distinguishes a well-behaved success from an agent that exited 0 without honoring the result contract.

**Pane** — reads result-file presence and **pane** liveness (`PaneAlive`); no exit file is ever written or consulted. It reports a **subset of three**:

| State | Condition | Meaning |
|-------|-----------|---------|
| `done` | `{stage}-result.yaml` present | the worker honored the result contract |
| `running` | result absent AND the pane is alive | still working (or sitting idle mid-task) |
| `orphaned` | result absent AND the pane is dead (killed / crashed / tmux server gone) | no result will arrive |

**Result presence WINS over pane liveness.** An interactive worker never exits on task completion — it finishes and sits at its prompt — so a liveness-first rule would report `running` forever and never terminate. **`failed` and `failed (no-result)` are UNREACHABLE in pane mode**: there is no exit-code channel, so a crashed or killed worker collapses into `orphaned`, and a clean-exit-without-result has no pane analogue. A pane dispatch simply never emits those two strings; no string changed and no new state exists.

#### Scenario: clean exit without a result is not done

- **GIVEN** a headless dispatch that exited `0` with **no** `{stage}-result.yaml`
- **WHEN** `fab dispatch status` runs
- **THEN** it reports `failed (no-result)`, never `done`

#### Scenario: a finished pane worker still at its prompt reads done

- **GIVEN** a pane dispatch whose window is alive with no result file
- **WHEN** `fab dispatch status` runs
- **THEN** it reports `running`
- **AND** GIVEN the result file is then written while the pane still lives, it reports `done`
- **AND** GIVEN the pane (or the whole tmux server) is gone with no result file, it reports `orphaned` rather than erroring out of `status`

### Requirement: `status --json` carries a `mode` discriminator plus the mode's identity

`--json` SHALL emit `{change, stage, state, mode, …}` where `mode` is `headless` or `pane`, followed by that mode's identity keys — `pid`, `pgid`, `exit?` (headless) or `pane`, `window` (pane). The other mode's keys are **omitted**, so a headless object is unchanged apart from the added `mode`, and `exit` stays absent for a pane dispatch (no exit file exists). The `mode` discriminator is what tells a consumer which state subset to expect. Keys evolve additively with no `schema_version`.

**`server` is deliberately absent from the JSON surface**, so `--json` alone is not enough to assemble a socket-scoped `fab pane capture` command; `fab dispatch logs` prints the complete command instead (below).

#### Scenario: the JSON shape names its mode

- **GIVEN** a pane dispatch
- **WHEN** `fab dispatch status --json` runs
- **THEN** the object carries `mode: "pane"` with `pane`/`window` populated and no `pid`/`pgid`/`exit`
- **AND** GIVEN a headless dispatch, it carries `mode: "headless"` with `pid`/`pgid` as before

### Requirement: `fab dispatch wait <change> <stage> [--timeout <secs>] [--json]`

`wait` SHALL be `status`'s **blocking sibling**: it blocks while the derived state is `running` and returns the moment that state becomes anything else, printing it — and, under `--json`, emitting the same object — **exactly as `status` does**, through the same render path. It SHALL re-derive state on a fixed in-process tick (the named constant `dispatch.TickInterval = 2s`, with no flag and no config field) using the **same loader and the same per-mode derivation** `status` uses (`DeriveState` / `DerivePaneState`, selected by `IsPane()`), so the two verbs **cannot disagree** about state by construction. It SHALL be purely additive — no existing subcommand, record schema, or state string changes — and it SHALL have **no side effects**: nothing is written and no signal is sent.

The per-observation logic lives in `internal/dispatch` as a pure control loop (`Wait(ctx, observe Observer, tick, timeout)`), where `Observer` is a function value the cobra layer fills with the same `observeDispatch` composition `status` runs — so the loop itself performs no I/O and is table-testable in milliseconds. The record is loaded **once** (`{stage}.yaml` is immutable for the lifetime of an attempt); only the derived signals — exit file, result file, pid/pane liveness — are re-read per tick.

- **Already-terminal returns immediately.** The first observation precedes any sleep, so a non-`running` state at entry costs no tick. This makes the verb idempotent and safe to re-arm after a `restart` or re-run after an interruption.
- **`--timeout <secs>` bounds the block; expiry is not an error.** On expiry `wait` SHALL observe **once more** and print that still-current state — necessarily `running`, since any other would already have ended the wait — exiting **0**. Re-observing at the bound (rather than reporting the last tick's cached state) is what keeps a transition landing in the sub-tick window from going unreported. Absent, or `0`, waits indefinitely.
- **The state string is the sole discriminator.** A consumer reading `running` knows the bound expired; every other string is a terminal state reached. There is no timeout-specific exit code and no timeout error.
- **Errors mirror `status`.** An unresolvable change or no dispatch record for the pair exits non-zero with `status`'s own message; only real errors do. An observe error aborts the wait rather than being re-polled, and ctx cancellation returns the last observed state with `ctx.Err()`, so a cancelled wait stays distinguishable from a timeout.
- **`wait --timeout` bounds the OBSERVER, never the worker** — unlike `start`/`restart`'s `--timeout`, which is a POSIX `timeout` inside the headless wrapper that kills the dispatched command.

Both modes are observed identically: a pane dispatch's state comes from the same result-presence-plus-pane-liveness rule, so the wiring is byte-identical across modes and `failed`/`failed (no-result)` are simply never returned there. The observation *policy* built on `wait` — the `--timeout 300` peek cadence, the single restart budget, escalation — is **skill-side**, not a CLI concern: see `_preamble.md` § CLI-Adapter Dispatch → *Recovery policy*.

#### Scenario: a blocking wait wakes on the transition

- **GIVEN** a `running` dispatch for `abcd/apply`
- **WHEN** `fab dispatch wait abcd apply` is blocking and the worker writes its exit + result files
- **THEN** the command returns within a tick, prints `done`, and exits 0

#### Scenario: timeout expiry reports `running` and succeeds

- **GIVEN** a dispatch that stays `running`
- **WHEN** `fab dispatch wait <change> <stage> --timeout 1` runs
- **THEN** it returns at the bound, prints `running`, and exits 0
- **AND** GIVEN a dispatch already `done` (or `failed` / `failed (no-result)` / `orphaned`), it prints that state and exits 0 without consuming a tick

#### Scenario: a worker's death surfaces within a tick

- **GIVEN** a headless dispatch whose worker dies without writing `{stage}.exit`
- **WHEN** `fab dispatch wait` is blocking on it
- **THEN** the next liveness probe derives `orphaned` and the command returns — the same state `status` would report at that instant

### Requirement: `fab dispatch logs <change> <stage> [--tail N]`

`logs` SHALL print `.fab-dispatch/{id}/{stage}.log`; `--tail N` prints the last N lines (implemented in Go via the `Tail` helper, no external `tail`). A missing log SHALL produce a clear "no dispatch log" message rather than erroring opaquely.

**A pane dispatch keeps no log file** — an interactive worker's output is tmux scrollback, not a redirected stream — so `logs` SHALL report that fact and name the equivalent read (`fab pane capture <pane>`) instead of the generic missing-log message. When the record carries a `server`, the printed command SHALL carry the socket too (`fab pane capture -L <server> <pane>`), since a socket-scoped pane is unreachable from a default-socket capture. This report is therefore the **copy-pasteable source** for the capture command, and the reason readers are pointed at it rather than at `status --json`.

#### Scenario: pane-mode logs points at the pane

- **GIVEN** a pane dispatch started with `--server work`
- **WHEN** `fab dispatch logs <change> <stage>` runs
- **THEN** it reports the no-log fact and prints `fab pane capture -L work <pane>`

### Requirement: `fab dispatch kill <change> <stage>`

`kill` SHALL terminate the dispatch by the mechanism its **derived mode** implies, and SHALL be **idempotent** in both cases — an already-dead target is a benign no-op with a clear report, and a missing dispatch record is a clear error rather than a panic or a signal to pid 0:

- **Headless** — `SIGTERM` to the **process group** (`pgid` from `{stage}.yaml`, via `syscall.Kill(-pgid, SIGTERM)`) so the detached command and its children die together. SIGTERM (graceful), not SIGKILL, matches "die together". Already dead (ESRCH) → an already-dead report.
- **Pane** — kills the **tmux pane** (`tmux kill-pane -t <pane>`), taking the interactive worker down with it. Already gone → an already-dead report naming the pane.

**No marker file is written in either mode**: with no result file present, a killed dispatch simply reads `orphaned` on the next `status`.

`kill` is the family's **recovery** verb — valid in **any** state, ungated by config, and the verb the pipeline's Recovery policy spends on a parked worker. It is distinct from `reap` (below), the **hygiene** verb, which fires only on `done`, only for a pane record, and only when `dispatch.reap_done` allows it.

#### Scenario: killing a pane leaves the dispatch orphaned

- **GIVEN** a live pane dispatch with no result file
- **WHEN** `fab dispatch kill <change> <stage>` runs
- **THEN** the pane is killed, a killed report is printed, and a subsequent `status` reports `orphaned`
- **AND** GIVEN the pane is already gone, `kill` exits 0 with an already-dead report

### Requirement: `fab dispatch reap <change> <stage>`

`reap` SHALL reclaim the tmux pane of a **finished pane-mode worker** — the pane-hygiene counterpart to `kill`. A pane worker never exits on completion (it writes `{stage}-result.yaml` and sits at its prompt, deliberately, so it can still be steered), so across a multi-stage pipeline every finished stage keeps its slice of the carved worker column and the panes the user actually watches shrink with each completed stage. The orchestrator calls `reap` at the one deterministic moment that already exists: immediately after it reads a `done` result.

It takes exactly two positional arguments and **no flags** — no `--json`, and no `--server`, because the socket comes from the record, so a `--server`-started dispatch is reaped on the right socket with nothing extra passed.

`reap` SHALL own the **whole guard** and kill the pane only when **all three** conditions hold:

1. the record is **pane-mode** (`IsPane()` — a `pane:`-bearing record), **and**
2. the **derived state is `done`** (`DerivePaneState`: `{stage}-result.yaml` present — pane liveness is irrelevant to the state), **and**
3. **`dispatch.reap_done` resolves `true`** (default `true`) through the four-layer config cascade.

Every other case SHALL be a **no-op with a one-line report naming its reason**, exiting 0:

| Case | Behavior |
|------|----------|
| record is **headless** | no-op — the detached process already exited; there is nothing visual to reclaim |
| state is **not `done`** (`running` / `orphaned`, or any headless state) | no-op — reap is NOT kill; it must never terminate a live or failed dispatch. The report points at `fab dispatch kill` |
| **`dispatch.reap_done: false`** | no-op — the user opted to keep done-worker panes and their scrollback |
| **pane already gone** (killed by hand, tmux server died) | benign already-gone report — mirroring `kill`'s idempotence, so a re-reap is safe |
| all three hold | `KillPane(rec.Pane, rec.Server)` — `tmux kill-pane` on the record's own socket, reporting `reaped pane %N for <change>/<stage>` |

**Exit codes**: 0 for the reap and for every no-op above; non-zero **only** for real errors — no dispatch record for the pair, or an unresolvable change — sharing `status`/`wait`'s message surface via the same `loadDispatchRecord` loader. An **unreadable `fab/project/config.yaml` is deliberately not in that set**, because the orchestrator calls reap unconditionally after every `done` and a broken config must not turn pane hygiene into a pipeline failure: the knob is resolved **lazily**, only once the mode and state conditions already hold (a headless or non-`done` no-op reads no config at all), and where it is read an unparseable file **warns on stderr and fails open** to `DefaultDispatchReapDone` — the same value an absent key resolves to.

**Reap kills the pane only; it cleans no state.** The record (`{stage}.yaml`), the result (`{stage}-result.yaml`), the prompt file, and the log all remain. That is exactly why a reaped dispatch still derives `done` forever (result presence wins over pane liveness) and why reap is pane **hygiene**, not state cleanup — the no-automatic-GC posture and its two deterministic cleanup moments are untouched (§ Two cleanup paths). `reap` introduces **no new state string, no record-schema field, and no migration**, and `restart` after a reaped attempt needs no special handling: last-attempt-only overwrite already covers a completed attempt.

Because `KillPane` is pane-ID keyed, reap is **shape-blind**: killing a **split**-shape worker's pane leaves the dispatching agent's pane, its window, and any sibling worker intact, while killing the only pane of a **new-window**-shape worker takes the window with it (plain tmux semantics).

The decision itself is the **pure** `DecideReap(isPane bool, state State, reapDone bool) ReapVerdict` in `internal/dispatch/reap.go` — no I/O, no config read, no tmux probe — so the mode × state × knob matrix is exhaustively table-testable, matching `SelectMode`/`SelectPaneShape`/`DerivePaneState` in the same package. Each skip is a **named verdict constant** (`ReapSkipHeadless` / `ReapSkipNotDone` / `ReapSkipDisabled` alongside `ReapPane`), so the command layer never recomposes a reason from raw booleans. The check order — mode, then state, then policy — is deliberate and only the *reported* reason depends on it: putting state ahead of the knob keeps "reap never terminates a live or failed dispatch" independent of configuration, so no value of `dispatch.reap_done` can reach a non-`done` dispatch. The cobra wiring (`cmd/fab/dispatch_reap.go`) reuses `resolveDispatchDir` / `loadDispatchRecord` / `KillPane`, and reads the knob through `resolve.FabRoot()` + `config.Load` in a small local helper — a second cheap upward walk that leaves `resolveDispatchDir`'s signature and its other call sites untouched. That helper **cannot fail**: it warns and falls back to `DefaultDispatchReapDone`, and the command calls it only when mode and state have already passed, so the config read sits behind the same short-circuit the pure guard applies (the placeholder value passed otherwise is never consulted). Both halves are what keep the exit contract above true: an eager, error-returning read would fail even a headless no-op whenever the project config will not parse.

#### Scenario: a done pane worker's pane is reclaimed and the dispatch stays done

- **GIVEN** a pane dispatch whose `{stage}-result.yaml` is present and `dispatch.reap_done` unset
- **WHEN** `fab dispatch reap <change> <stage>` runs
- **THEN** the worker's pane is killed on the record's own socket, the report names the pane, and the exit code is 0
- **AND** a subsequent `fab dispatch status` still reports `done`, with every `.fab-dispatch/{id}/` file it held before still present

#### Scenario: reap never terminates a live dispatch

- **GIVEN** a pane dispatch that is still `running`
- **WHEN** `reap` runs
- **THEN** nothing is killed, the report names the state and points at `fab dispatch kill`, and the exit code is 0
- **AND** the same holds for a headless record, for `dispatch.reap_done: false`, and for a pane already gone

#### Scenario: reaping a split worker leaves the dispatcher and its siblings intact

- **GIVEN** a dispatcher pane with two stacked worker panes, one of them `done`
- **WHEN** the `done` worker is reaped
- **THEN** only that worker's pane disappears; the dispatcher pane and the sibling worker survive

### Requirement: A pane dispatch's identity is `fab-{id}-{stage}` and carries no operator marker

A pane dispatch SHALL take its identity from its own convention — `fab-{4-char-change-id}-{stage}`, composed by `WindowName` — and that string MUST NOT carry the operator's `»` (U+00BB) enrollment prefix or its `›` (U+203A) done marker, in **either** pane shape. The string's **carrier** varies by shape (the tmux **window name** in the new-window shape, the tmux **pane title** in the split shape) but the string does not. Those markers assert that a window is in the operator's monitored set and that the operator owns its lifecycle, neither of which a pipeline dispatch has. An operator that genuinely enrolls a window adds the marker itself through its own idempotent `fab pane window-name ensure-prefix` primitive (see [operator.md](/runtime/operator.md) § monitored-set enrollment and [pane-commands.md](/runtime/pane-commands.md)).

#### Scenario: a dispatch window is identifiable without claiming operator ownership

- **GIVEN** `fab dispatch start abcd apply --pane --server work` (the new-window shape)
- **WHEN** the window is created
- **THEN** its name is `fab-abcd-apply` and carries no `»`/`›` prefix

#### Scenario: a split worker's pane title is identifiable and equally unmarked

- **GIVEN** `fab dispatch start abcd apply` from inside a tmux pane (the split shape)
- **WHEN** the worker's pane is created
- **THEN** its pane title is `fab-abcd-apply` and carries no `»`/`›` prefix, and no window is renamed

### Requirement: Steering a pane worker is contract-neutral

A user MAY converse with a running pane worker mid-stage. This changes **no** contract: the worker still owes `{stage}-result.yaml`, still ends with the terminal `fab status refresh` epilogue, and still runs no `fab status` **transition** command — **the orchestrator owns every transition** (see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership). Steering is human input into the worker's context, exactly like answering a native sub-agent's question. A worker steered away from producing a result needs no new state: it never reaches `done` and surfaces through the never-`done` escalation the orchestrator already owns. This is **documentation only** — no code detects, gates, or reports a steered worker.

#### Scenario: a steered worker owes the same artifacts

- **GIVEN** a user typing into a running pane worker's window
- **WHEN** the worker finishes
- **THEN** the same result-file and refresh-epilogue obligations apply and the orchestrator's transition ownership is unchanged
- **AND** GIVEN the worker never produces a result, the dispatch stays `running`/`orphaned` and the orchestrator escalates by its existing never-`done` path

### Requirement: Two cleanup paths, no automatic GC

**State** cleanup SHALL happen at exactly **two deterministic moments** and never on a timer (throttled/timer sweeps were explicitly rejected — matching fab's no-magic-background-work posture). `fab dispatch reap` is **not** a third moment: it kills a finished pane worker's tmux pane and removes **no files at all**, so it reclaims visual space without touching `.fab-dispatch/` state (§ `fab dispatch reap`).

1. **Archive-time deletion.** `fab change archive` deletes `.fab-dispatch/{id}/` as part of the archive move — dispatch artifacts are transient comms, not history — so `fab change restore` does **NOT** recreate them. The deletion lives in `internal/archive.Archive()` (best-effort, immediately after the folder move, computing the repo root as `filepath.Dir(fabRoot)`); an absent dir is a no-op and a removal error never undoes the completed move. See [pipeline/change-lifecycle.md](/pipeline/change-lifecycle.md) § archive/restore and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § `/fab-archive`.
2. **Manual `fab dispatch clean [<change>] [--orphans]`.** `clean <change>` removes that change's dir; `clean` (no arg) removes all `.fab-dispatch/*/` dirs; `clean --orphans` prunes any `.fab-dispatch/{id}/` whose ID does not resolve to a **non-archived** change (via `resolve.ToFolder`, which excludes `archive/`), covering the case where a change was archived upstream and a local `git pull` left the state dir orphaned.

`clean` is **mode-blind**: it removes state dirs and never inspects a record's mode, so a pane dispatch's dir (prompt file included) is cleaned exactly like a headless one's. As with a live headless process, cleaning a **live** pane dispatch removes the state without killing the worker — `kill` is the verb for that.

#### Scenario: `--orphans` prunes only unresolvable IDs

- **GIVEN** several `.fab-dispatch/*/` dirs, one whose ID does not resolve to an active change
- **WHEN** `fab dispatch clean --orphans` runs
- **THEN** only the orphaned dir is pruned; live dirs are left intact

## Design Decisions

### A done pane worker's column space is reclaimed by default, with the knob as the opt-out

**Decision**: The default posture for a finished pane worker is **space-reclaimed** — `dispatch.reap_done` defaults to `true`, so the orchestrator's post-`done` `fab dispatch reap` kills the pane. Anyone who wants a done worker's scrollback sets the knob `false` and keeps the pane untouched.

**Why**: The pane-worker column exists to keep the dispatching agent readable, and a pipeline that leaves one dead pane per completed stage erodes exactly that: after three stages the panes the user actually needs to watch have been squeezed by three panes nobody is reading. The evidence a reaped pane holds is not lost — the result file, prompt, log, and record all survive, and they are what an orchestrator and a human actually read after the fact. Reaping at the post-`done` moment also needs no new machinery: it is an orchestrator-invoked call at a point the wiring already reaches, not a timer or a watcher, so it stays inside the no-magic-background-work posture.

**Rejected**: **Zoom / shrink-in-place** (`resize-pane -y 2`) — a two-line stub still holds a slot in the column, so the squeeze returns with enough stages. **`break-pane` parking** to a held window — it preserves scrollback but re-clutters the window list the two-tier hierarchy was built to clean up, trading one clutter surface for another. **Defaulting to `false`** (evidence-preserved) — it makes the common case the broken one and requires every user to opt into hygiene they already expected.

*Introduced by*: 260807-zfl7-dispatch-reap-done-panes

### Reap is a distinct verb from kill, and owns its whole guard in Go

**Decision**: Pane hygiene is a **new verb** (`fab dispatch reap`) rather than a flag on `kill`, and it owns all three guard conditions — pane-mode record, derived state `done`, `dispatch.reap_done` true — so the skill wiring calls it **unconditionally and dumbly** after reading a `done` result.

**Why**: `kill` is a **recovery** verb, valid in any state and already in the pipeline's sanctioned verb set with a documented never-kill-a-worker-awaiting-input rule; reap is **hygiene**, fires only on `done`, and is policy-gated. Folding them would put a config-gated no-op inside a verb whose contract is "terminate this now". Putting the guard in Go is what lets the call site stay dumb: the knob resolves through the four-layer cascade (environment > project > system `~/.fab-kit/config.yaml` > defaults), and a skill reading `fab/project/config.yaml` directly would miss both process-local and machine-wide `both`-scope preferences. One dumb call site also keeps the wiring identical across adapters and modes, since a headless record is a reported no-op inside the command.

**Rejected**: `kill --if-done` (one verb with two contracts, and a config knob silently modulating a recovery command). A skill-side conditional `if pane && done && knob` (three chances to drift from the runtime, and a wrong config read on the layer that matters most). Making reap a *recovery* verb the Recovery policy may spend (it would blur the never-kill-a-live-worker invariant the policy rests on).

*Introduced by*: 260807-zfl7-dispatch-reap-done-panes

### Worker placement keys on the record's pane ID, and the live `list-panes` intersection is the whole filter

**Decision**: `SiblingDispatchPane` collects the `pane:` field of every `.fab-dispatch/*/{stage}.yaml` record in the checkout that was recorded against the socket being probed, then keeps the **last** pane in `tmux list-panes -t <dispatcherPane> -F '#{pane_id}'` that appears in that set.

**Why**: Pane IDs are server-global and stable for the pane's lifetime — the same reason `status`/`kill`/`capture` key on them — whereas a pane title is rewritten by the harness running inside the worker. That is what makes a title-keyed probe unusable: it finds no sibling, so every worker takes the no-sibling branch and re-splits the dispatcher, degrading the window into N equal-width columns with the session agent squeezed to a sliver. Intersecting with the window's own live pane list collapses three filters into one probe — liveness (a dead pane is absent), same-window scoping (a `-t <pane>` target resolves to that pane's window), and the geometric "last" ordering the stacked column is built in — which is also what makes the all-record-dirs enumeration scope safe: a pane recorded by another change in another window never matches. Filtering on `Server` equality is required because pane IDs are per-socket, so a `--server`-recorded `%N` would otherwise false-match an unrelated default-socket pane.

**Rejected**: Per-record `PaneAlive` probes plus a separate window lookup (N tmux calls for the answer one `list-panes` already gives, and it would still need the list order to define "last"). Scanning only the active change's record dir (a window can host two changes' workers). Keeping the title scan as a fallback — it is the broken signal, so a fallback would silently resurrect the bug. Enforcing the layout with `select-layout main-vertical` after every pane event (rearranges the user's hand-made panes and resets manual resizes on every dispatch). A repair pass over an already-mangled window (fab only stops creating new mess; existing columns stay until their panes die). Workers in a separate `fab-{id}-workers` window (loses side-by-side glanceability).

*Introduced by*: 260807-g4a5-pane-worker-column-invariant

### The size rides the resolved placement, so one rule also covers the degraded path

**Decision**: `SplitTarget` returns a `SplitPlacement{Target, Direction, SizePercent}` and sets `SizePercent` only on the column-carving `splitRight` decision; `splitArgs` renders `-l {n}%` from it, and `OpenSplitPane` merely executes the placement.

**Why**: "Size the carving split, never a stacking split" then exists once — in the decision — rather than at each call site, and the degraded branch (probe failed ⇒ carve off the dispatcher) is the *same* `splitRight` decision, so it inherits the size for free instead of needing its own copy of the rule. That matters because an unsized fallback column would halve the dispatcher and reintroduce exactly the squeeze the width exists to prevent. Bundling also keeps `OpenSplitPane`'s parameter list from growing another argument, and the percentage form (rather than a cell count) keeps the column proportional across window resizes.

**Rejected**: A `sizePercent` parameter threaded separately through `SplitTarget` and `OpenSplitPane` (two places to remember the direction condition). Sizing inside `OpenSplitPane` by re-deriving the direction (a second copy of the rule, free to drift from `SplitTarget`).

*Introduced by*: 260807-g4a5-pane-worker-column-invariant

### A rejected size retries unsized rather than probing the tmux version

**Decision**: `OpenSplitPane` runs the sized split and, when tmux rejects it while a size was requested, retries the identical split with the size dropped, returning the first failure as a non-fatal warning.

**Why**: It covers the whole class of "this tmux will not take this size" — pre-3.1 with no `-l N%` syntax, a window too narrow for the requested percentage — with no version string to parse (`3.1a`, `next-3.4`, distro forks) and no extra tmux round-trip on the happy path. It is also exactly the existing degradation contract: placement is cosmetic, so it warns and proceeds.

**Rejected**: A `tmux -V` version probe (a second call on every dispatch, plus version-string parsing that rots). Failing the dispatch (contradicts placement-is-cosmetic). Silently dropping the size (the user set a knob; a silent no-op is unexplainable from output).

*Introduced by*: 260807-g4a5-pane-worker-column-invariant

### A record-read failure returns its partial set alongside the error

**Decision**: `recordedPanes` returns `(map[string]bool, error)`. An absent `.fab-dispatch/` tree is the benign empty-set/nil-error case; every real read or parse failure is joined into `SiblingDispatchPane`'s error and surfaced by the cobra layer's warning, which names both the failure and the resolved placement — while the records that *did* read are still returned.

**Why**: Discarding the partial set on any failure would turn one corrupt record into a forced carve, whereas keeping it is strictly safer: a record that could not be read can only fail to *find* a sibling, never invent one, because the caller still intersects with the window's live pane list. So the degraded answer is either the clean answer or the sized carve — never a misplacement. Distinguishing `os.IsNotExist` on the tree root from a real failure is what keeps the ordinary first dispatch silent instead of warning on every run.

**Rejected**: Returning only the set and swallowing errors (a corrupt record then degrades placement invisibly, against the warn-only contract). Returning only the error and dropping the set (loses a usable answer for a partial failure). Warning on an absent tree (the common case is not a problem).

*Introduced by*: 260807-g4a5-pane-worker-column-invariant

### Observation is a blocking `wait` over an internal derivation tick, not an fsnotify watch

**Decision**: `fab dispatch wait` blocks by re-deriving state on a fixed ~2s in-process tick (`dispatch.TickInterval`) over the existing loader plus `DeriveState`/`DerivePaneState`, with the loop expressed as a pure control structure (`Wait(ctx, observe, tick, timeout)`) whose `Observer` the cobra layer fills with `status`'s own composition.

**Why**: The cost being eliminated is **inference turns** — an orchestrator waking every 30s to run `fab dispatch status` burns a full turn per poll, so a 90-minute stage costs ~180 turns observing that nothing happened — not filesystem syscalls, and a 2s stat-plus-liveness probe inside one Go process is free. Reusing the derivation `status` already owns is what makes the two verbs structurally incapable of disagreeing, and injecting the observer keeps the loop table-testable in milliseconds without launching a process or a tmux server. The 2s value also buys orphan latency: a dead worker surfaces in ~2s instead of at the next poll, so the recovery policy's single automatic restart fires almost immediately.

**Rejected**: An fsnotify watch on `{stage}-result.yaml` — a watcher can see a worker *finish* but never see one *die*, and `orphaned` is derived from pid/pane liveness, so a periodic probe would still be needed alongside it: a new dependency that removes nothing. A configurable tick (the user-visible knob is `--timeout`, which bounds the block; the tick is an implementation detail of the blocking contract, not a tuning surface). Returning the last tick's cached state at the bound (a transition in the sub-tick window would go unreported, letting `wait` print `running` while `status` printed `done` at the same instant).

*Introduced by*: 260806-mkfj-dispatch-wait-event-driven

### Timeout expiry exits 0, discriminated by the printed state string

**Decision**: On `--timeout` expiry `wait` prints the still-current state (`running`) and exits 0; non-zero exits stay reserved for the same real errors `status` raises.

**Why**: The consuming skill reads stdout either way and the state string already carries the fact, so a second channel would be redundant. It also keeps the common "timed out, go peek" path from looking like a failure to any shell-level `set -e` wrapper — and that path is the *normal* one, since the wiring's `--timeout 300` exists precisely to produce periodic peek moments.

**Rejected**: A dedicated timeout exit code such as POSIX `timeout`'s `124` — in this family `124` already means "the **worker** was killed by its own `start --timeout` wrapper", so reusing it for the **observer**'s bound would invert the meaning of the one exit code the family already assigns. A timeout error return (turns an expected outcome into an exceptional one).

*Introduced by*: 260806-mkfj-dispatch-wait-event-driven

### `restart` is a prompt-source seam on `start`'s launch path, not a second command implementation

**Decision**: `fab dispatch restart` shares `start`'s entire launch path (`runDispatchLaunch`) and differs only through a `promptSource` function — `promptFromStdin` vs `promptFromStateDir`. The source also reports whether the shared path should persist the bytes: `start` does, `restart` does not (the file it read *is* the input).

**Why**: Every semantic a restart needs is already `start`'s — prompt persistence, refuse-if-running, last-attempt-only overwrite, the mode ladder, the output shape. Copying that tail would have created two places to change every one of them, and the byte-shape parity the design promises would have been an assertion rather than a structural fact. Threading **bytes** (not an `io.Reader` or a strategy interface) keeps the seam minimal: both sources are fully read into memory before the launch anyway, since the file must exist on disk by the time a pane worker follows its pointer.

**Rejected**: A `--from-prompt-file` flag on `start` (overloads one command with two contradictory input contracts and makes the recovery verb unnameable in a poll loop); a duplicated `runDispatchRestart` tail (the drift the shared prologue was extracted to prevent).

*Introduced by*: 260806-mnri-dispatch-worker-lifecycle-supervision

### A restart re-derives its mode and leaves no trace of itself

**Decision**: A restart resolves its launch mode from the *current* environment through the same ladder, and adds nothing to the record — no attempt counter, no history, no `restarted:` marker. `status` cannot tell a restart from a `start`.

**Why**: The environment is exactly what changed when a worker died, so inheriting the prior attempt's mode would reproduce the failure — restarting a pane dispatch orphaned *by* a dead tmux server must land headless, and the auto ladder already does that for free. And a restart genuinely *is* a fresh attempt under last-attempt-only (a shipped design decision): a marker would be a second source of truth the state machine has no use for, and an on-disk counter would need reset semantics fab does not want to own. The recovery budget that bounds restarts therefore lives in the orchestrator's context, where the worst case after a context loss is one extra restart.

**Rejected**: Persisting the prior mode and reusing it (re-fails on the condition that killed the worker); a `restarts:` key in `{stage}.yaml` (breaks last-attempt-only and invites a supervisor-shaped design); a `--same-mode` flag (no demonstrated need, and no stored discriminator to read).

*Introduced by*: 260806-mnri-dispatch-worker-lifecycle-supervision

### Setsid syscall attribute, not the `setsid` binary
**Decision**: Detach via Go's `exec.Command(...).SysProcAttr = &syscall.SysProcAttr{Setsid: true}` on a plain `sh -c '...'` wrapper — a single detach mechanism — rather than prefixing the `setsid` binary as the intake's `setsid sh -c` string literally suggested.
**Why**: An end-to-end smoke test showed the `setsid` **binary** double-forks (its caller is already a process-group leader under Setsid), so the Go-recorded pid pointed at an immediately-exiting `setsid` process — breaking liveness, refuse-if-running, and kill. One trackable detach mechanism is the correctness fix; the observable behavior (detached, survives orchestrator death, resumable) matches the intake exactly.
**Rejected**: The literal `setsid` binary prefix (untrackable pid); a long-lived Go supervisor process that waits on the child (re-introduces a process that must itself survive the orchestrator — defeats the point; the shell wrapper's `echo $? > exit` makes the shell the supervisor with no Go process in the loop).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### Parallel family, not a headless mode on `fab pane`
**Decision**: `fab dispatch` is a command family independent of `fab pane` / `fab operator`; the `fab pane` command surface and the operator's monitored-set machinery carry no dispatch concerns. Pane-mode dispatch consumes `internal/pane`'s tmux helpers (`RunCmd`/`WithServer`/`StderrError`) as a library and borrows tmux as a launch surface, without joining the operator's monitored set or adding a headless mode to `fab pane`.
**Why**: Pane *observation* (tmux capture, operator ownership) and *stage dispatch* (a state dir, a result-file contract, an observation loop over derived state) are different models with different owners; conflating the command surfaces would burden the interactive-operator path with pipeline concerns. Sharing the tmux argv builder at the library level gets the reuse without the conflation — one tmux invocation convention in the binary, two independent command families.
**Rejected**: Extending `fab pane` with a headless mode (model conflation — the inverse of the one `fab dispatch` was created to avoid). Re-implementing tmux invocation inside `internal/dispatch` (a second argv builder and a second stderr-enrichment convention). Automatic GC of state dirs on a timer (cleanup is exactly two deterministic moments).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### `internal/dispatch` package with thin cmd wiring
**Decision**: Extract state read/write, wrapper composition (`WrapperArgv`), both state derivations (`DeriveState`, `DerivePaneState`), process signaling, and the tmux pane primitives (`pane_mode.go`) into `internal/dispatch`, with thin `cmd/fab/dispatch*.go` cobra wiring.
**Why**: The status-state machines and wrapper composition are the testable core; the `internal/pane` / `internal/archive` precedent and the need to table-test the pure state machines independent of a launched process or a live tmux server make extraction the clear default. The tmux primitives sit in the platform-independent core (not the build-tagged split) because they are plain subprocess calls with no syscall dependency.
**Rejected**: Inline in `cmd/fab` (harder to table-test the state derivations without launching a real process).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### Pane mode is a flag on `fab dispatch start`, not a new command family
**Decision**: Interactive dispatch is `fab dispatch start <change> <stage> --pane`, sharing the role resolution, state directory, refuse-if-running concurrency, and `status`/`kill`/`logs`/`clean` surfaces with the headless path. Mode selection is **per-invocation** — resolved by the explicit-first ladder ending in the `$TMUX`-driven auto default, so a skill normally passes no mode flag at all — with **no provider config field**: pane mode composes the resolved provider's existing `session_command`, so `fab resolve-agent`'s output is identical either way.
**Why**: The sequencer wiring already branches at `fab dispatch`, and the state-string vocabulary plus the `.fab-dispatch/{id}/` layout are exactly the machinery a pane dispatch needs; a parallel command family would duplicate all of it and force every dispatch site to learn a second grammar. Keeping the interactive invocation derivable from `session_command` — the same string `fab agent` composes — means no schema change and no new default behavior; a provider whose interactive grammar genuinely diverges from its session grammar is the trigger to add a field later, as a data-only config addition.
**Rejected**: A new `fab pane-dispatch` command family (duplicates the state machine, the loader, and the concurrency check). A third per-provider command field (`pane_command`) in v1 (schema churn for a string already in the config). Declaring the mode in config rather than per invocation (watchability is a property of *this run*, not of a provider).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

### `$TMUX` presence is the mode default, resolved in Go by a pure ladder function
**Decision**: The launch mode is resolved inside `fab dispatch start` by the pure, table-tested `dispatch.SelectMode(paneFlag, headlessFlag, timeoutSet, serverSet bool, tmuxEnv string) (Mode, AutoReason)` — an explicit-first ladder (`--pane` / `--headless` / `--timeout` ⇒ headless / `--server` ⇒ pane) whose last rung defaults from `$TMUX` presence. Skills get documentation changes only; the dispatch seam is untouched.
**Why**: Resolving in Go is a **single enforcement point** covering manual and skill-driven invocations identically, so no dispatch site has to remember an environment check. `$TMUX` is the right *defaulting* signal because it means precisely "a window opened without `-L` lands on the server the caller is attached to" — the condition under which an un-targeted pane is visible to a human. A pure function (no env read, no tmux probe) makes the seven-input matrix table-testable without launching a process or a tmux server, matching `DeriveState`/`DerivePaneState` in the same package. Each explicit rung keys on `Flags().Changed` rather than value, so `--timeout 0` / `--server ""` remain signals; an empty `$TMUX` reads as unset because Go cannot distinguish the two and tmux never exports an empty value.
**Rejected**: An inline `if` chain in `runDispatchStart` (untestable without real launches, and the file already delegates every derivable rule to `internal/dispatch`). A `dispatch.default_mode` config knob in v1 (the env default plus two explicit flags cover the matrix; a knob stays additive later). Detecting run-kit specifically rather than tmux (rk presence adds nothing — the pane lands in tmux either way). Reading `$TMUX` at the dispatch seam in every skill (N enforcement points, each able to drift).
*Introduced by*: 260805-l9ng-auto-pane-dispatch-in-tmux

### Every pane-path failure is asymmetric: soft under auto, hard under explicit
**Decision**: Both pane-path prerequisites — a reachable tmux server and a `session_command` on the resolved provider — are validated *before* any command is composed, and both failures take the same asymmetry: an **auto**-selected pane degrades to headless with a one-line stderr notice per shape (`dispatch.FallbackNotice` / `dispatch.FallbackNoticeNoSessionCommand`), re-composing from `dispatch_command`, while an **explicitly** selected pane (`--pane`, or `--server` as the pane signal) keeps its hard error — non-zero exit, nothing launched, nothing persisted. The `dispatched …` line names the selection source whenever auto fired (`auto: tmux` / `auto: no tmux` / `auto: tmux unreachable` / `auto: no session_command`), and stays byte-identical for an explicit selection.
**Why**: The two selection sources carry different caller intent. A stale `$TMUX`, or a provider that only ever dispatched headless, must never break a dispatch that *never asked* for a pane — that is a defaulting heuristic missing, not a request failing. Conversely, a caller who typed `--pane` asked for watchability, and a silent downgrade would defeat the request. Deferring composition until the mode is validated is what makes the `session_command` shape reachable at all: composing the pane command first hard-fails a `dispatch_command`-only provider before the fallback decision, so every CLI-dispatched stage of such a provider would regress inside tmux — the exact byte-preservation the auto default promises. Naming the selection source on the output line is the compliance-visibility seam: a surprising mode (or a fallback) is explainable from output alone, while keeping the explicit report byte-identical protects existing output assertions. The fallback re-composes from `dispatch_command` so the load-bearing no-cross-fallback rule survives the mode change.
**Rejected**: A hard error under auto (a stale env var breaks unattended runs; a `dispatch_command`-only provider regresses outright inside run-kit). A soft fallback under explicit `--pane` (silently discards the caller's stated intent). Composing the pane command before mode validation (makes the `session_command` shape unreachable). Suffixing every dispatch line with its mode source (churns byte-stable output for explicitly-selected modes, where the source is already obvious from the flag).
*Introduced by*: 260805-l9ng-auto-pane-dispatch-in-tmux

### One `validatePane` helper bundles each shape's three values, so the asymmetry is one branch
**Decision**: Both pane prerequisites are checked by a single `validatePane(prov, server, stage, providerName) *paneFallback` helper in `cmd/fab/dispatch_start.go`, returning `nil` when the pane path can proceed and otherwise a `paneFallback` struct bundling that shape's three per-shape values — the stderr `notice`, the `AutoReason`, and the plain `error` an explicit selection propagates. The call site then branches **once** on `reason != dispatch.ReasonAutoTmux`. The **probe runs before** the `session_command` read, so a provider failing *both* prerequisites reports shape (a). The shape-(b) error text is the shared `missingCommandError(stage, providerName, field)`, the same constructor `modeCommand` raises, and `modeCommand` keeps its own pane-branch absent-field check.
**Why**: R3 requires identical asymmetric handling for two independent failure shapes. Bundling each shape's three values keeps exactly one auto-vs-explicit `if` in the command, where a per-shape branch would have grown a second copy of the same rule — the duplication the no-duplication acceptance criterion forbids. Validation is probe-plus-a-config-field-read only: it writes no state and composes no command, so **both** outcomes are safe before anything is persisted, which is what lets the explicit path guarantee "nothing launched, nothing persisted". Reachability is checked first because it is the environment-level precondition and it owns the pinned `pane mode requires a reachable tmux server` message, so reporting the environment problem first matches the ladder's outside-in ordering; the ordering is deliberately left an implementation detail rather than a pinned contract, so the both-shapes-failing regression test asserts the generic "falling back to headless" substring. Sharing `missingCommandError` between the pre-composition diagnosis and composition itself means the two sentences cannot drift, while retaining `modeCommand`'s own check keeps its independent contract that no mode ever composes an empty command.
**Rejected**: A separate auto-vs-explicit branch per failure shape (two copies of one rule, free to diverge). Closures on the fallback struct for the hard error (`func(string, string) error` values that discarded both parameters — passing `stage`/`providerName` into `validatePane` makes the field a plain `error`). Reusing one generic notice for both shapes (they need different fixes — start tmux vs. configure a `session_command` — and the output suffix is the only explanation a caller gets). Diagnosing the missing `session_command` at composition time only (makes the auto fallback unreachable). Dropping `modeCommand`'s pane-branch check as now-redundant (it would let the function compose an empty command).
*Introduced by*: 260805-l9ng-auto-pane-dispatch-in-tmux

### Real-tmux dispatch tests isolate by a verified private socket, never by `-L` alone
**Decision**: Every `cmd/fab` test that starts a real tmux server runs against a **private socket** under a per-test `TMUX_TMPDIR`, and each such test hard-**fails up front** unless `$TMUX` is empty. A test that must issue **unscoped** `tmux new-session` / `kill-server` (the auto-inside-tmux integration tests, which prove auto passes no `-L`) additionally **verifies the server actually bound the private socket** before registering its cleanup, and scopes every later call — `kill-server` included — with an explicit `-S <verified-socket>`.
**Why**: fab's own tmux tests run on whatever server the developer is attached to. A **set** `$TMUX` makes tmux ignore `TMUX_TMPDIR` and target the attached server, so an unscoped `kill-server` cleanup would kill a live development or run-kit server — real data loss from a test cleanup. The refuse-if-`$TMUX`-set assertion plus socket verification turns that from an implicit dependency on helper ordering into a checked precondition: no destructive call is registered until the private socket is proven, and the verified path (not a name or a label) is what every destructive call targets. The same discipline is why the pane tests that *can* be scoped use `-L` with a private, empty `TMUX_TMPDIR` — an unreachable-server assertion then rests on the socket genuinely having no server rather than on the host happening to run none.
**Rejected**: Relying on `-L <label>` alone (a label under the shared default `TMUX_TMPDIR` can collide with a real server). Relying on a helper's incidental `TMUX=""` for safety (an ordering change silently re-arms the hazard). Skipping the tmux tests when a server is present (the auto-inside-tmux path would go unasserted on exactly the hosts it ships for — all pane integration tests must RUN, not skip into a false pass). Giving the environment-dependent `session_command` test its own ephemeral server (a second `kill-server` cleanup for coverage the tmux-isolated sibling already asserts — the assertions were folded in there instead).
*Introduced by*: 260805-l9ng-auto-pane-dispatch-in-tmux

### Pane completion keys on the result file, with liveness only separating running from orphaned
**Decision**: A pane dispatch is `done` when `{stage}-result.yaml` exists, `running` when it does not and the pane lives, and `orphaned` when it does not and the pane is gone. Result presence wins over liveness, and `failed` / `failed (no-result)` are simply unreachable rather than remapped or renamed.
**Why**: An interactive worker never exits on task completion — it finishes and sits at its prompt — so no exit-code channel exists to key on, and a liveness-first rule would report `running` forever. The result file is already the contract's success token for the other adapters, so reusing it keeps **one** success definition across all three. Leaving the two exit-code-derived states unreachable keeps the state strings byte-stable, so every existing consumer of `fab dispatch status` reads a pane dispatch without changes; a mode-specific string would have forked the cross-adapter contract.
**Rejected**: Screen-pattern detection via `tmux capture-pane` (scrollback-dependent and ambiguous — the `_cli-agents` § Await guidance explicitly prefers an artifact over a screen pattern). Requiring the worker to exit after writing its result (throws away the steer-after-finish property that motivates the adapter). New pane-only state strings (forks the byte-stable contract).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

### Prompt file plus a one-line pointer, embedded at spawn
**Decision**: The full stage prompt is persisted to `{stage}-prompt.md` — the path the headless path already writes — and the pane worker receives a one-line pointer to it, embedded as the interactive command's single **shell-quoted** prompt argument at window creation (composed by `dispatch.WindowCommand`, which reuses the package's `shellQuote`; the resolved command itself stays verbatim). Prompt *content* is composed identically for every adapter.
**Why**: A multi-thousand-token stage prompt cannot ride `send-keys` or argv reliably, and embedding the pointer at spawn sidesteps the printed-prompt trap entirely — there is no pre-existing buffer to probe when the window is created with its prompt already attached. The file doubles as a debugging artifact and rides the existing cleanup paths, so `.fab-dispatch/` gains no new file type and no GC change. Quoting the pointer (rather than wrapping it in bare `'…'`) is what keeps the asymmetry honest: the pointer is repo-path-derived text fab composes, so it gets escaped per § Spawn Composition, while the `session_command` is the user's own string whose expansions must survive — one composer holds both rules so neither drifts.
**Rejected**: Sending the whole prompt via `fab pane send` after spawn (the printed-prompt trap plus send-keys length limits). Passing the prompt on the interactive command's stdin (an interactive TUI reads stdin as keystrokes, not as a prompt). Composing a shorter prompt for pane mode (would fork the dispatch-prompt obligations that bind all three adapters).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

### Pane identity extends the existing record; the mode is derived, never stored
**Decision**: `pane`, `window`, and `server` are `omitempty` fields on the existing `Dispatch` record, `pid`/`pgid` become `omitempty` too, and the mode is **derived** from `Pane` being non-empty (`IsPane()`/`Mode()`, with named `ModeHeadless`/`ModePane` constants) rather than persisted as a discriminator.
**Why**: Extending the record keeps one loader, one save path, and one refuse-if-running check. `omitempty` on both sides makes a headless record's on-disk bytes identical to before, so the change is purely additive and **no migration exists or is needed** — and a record written before pane mode existed reads as `ModeHeadless` with no backfill. A stored discriminator would be a second source of truth that could disagree with the identity fields.
**Rejected**: A second state file for pane dispatches (two loaders, two concurrency checks, two things to clean). A persisted `mode:` key (redundant with the identity fields and able to drift from them; would also require a migration to stamp onto existing records).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

### Pane dispatches get a `fab-{id}-{stage}` identity, not the operator's `»` marker
**Decision**: A pane dispatch's identity string is `fab-{id}-{stage}` and carries no `»`/`›` prefix — in both pane shapes, whether the string rides a window name or a pane title.
**Why**: The `»` prefix is the operator's enrollment marker — it asserts the window is in the operator's monitored set and that the operator owns its lifecycle. A pipeline dispatch has neither property, so pre-marking would make the operator's tab bar lie about what it tracks. A distinct, greppable name convention gives the same at-a-glance identification without the false claim, and an operator that genuinely enrolls a window still adds the marker through its own idempotent primitive.
**Rejected**: Prefixing `»` at creation (falsely signals operator ownership). Leaving the window/pane unlabelled (indistinguishable from an ad-hoc shell tab, and the string is what makes a worker greppable at a glance).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

### Pane workers split the dispatching agent's window, with the new window as fallback
**Decision**: A pane-mode worker opens as a **pane split into the dispatching agent's own window** whenever `$TMUX_PANE` is non-empty and no `--server` was supplied; otherwise it keeps opening as a **new window** named `fab-{id}-{stage}`. The decision is the pure `SelectPaneShape`; both env reads stay in the cobra layer. WHERE inside that window the pane lands is a separate decision — the stacked right column of § Split placement.
**Why**: The tmux hierarchy is genuinely **two-tier** — an operator opens worktree agents as windows, and each agent dispatches its own stage workers — but pane dispatch collapsed both tiers onto windows, so every stage worker surfaced as another window in the operator's and run-kit's window list, drowning the tier that actually maps to worktrees. Splitting the dispatcher's window puts a worker exactly where its dispatcher is, which is also the layout the native agent-team UI uses, so it reads as intended rather than as clutter. The new-window shape is kept — not replaced — because the two conditions that select it are the two where a split is *impossible*, not merely undesirable: with `--server` the caller's pane id is meaningless on the named socket (pane ids are server-global), and with `$TMUX_PANE` unset there is no pane to split at all. Keeping it as the fallback makes the change additive: every pre-existing caller's output and layout is byte-identical. The stacked column, rather than repeated splits of the dispatcher's own pane, keeps the dispatcher's pane from halving with every worker. Placement degrades rather than fails (a failed sibling probe falls back to `-h`, a failed title set only warns) because both are cosmetic and the worker is already running by then.
**Rejected**: Replacing the new-window shape outright (breaks headless orchestrators and `--server` callers, which have no pane to split). A `--split`/`--window` flag (the environment already answers the question, and a flag would have to be threaded through every dispatching skill for no added expressiveness). A second record field for the pane title (the string is identical to the window name, so a second field carries no information and would break every existing record's schema for nothing). Splitting `$TMUX_PANE` every time (each worker would halve the dispatcher's pane again, so the third worker leaves the dispatcher unreadable).
*Introduced by*: off-pipeline follow-up to 260806-mnri-dispatch-worker-lifecycle-supervision (pane dispatch splits the dispatching agent's window)

### `--pane` is mutually exclusive with `--timeout`
**Decision**: Supplying both flags is a usage error, enforced before any launch or file write.
**Why**: `--timeout` is implemented as POSIX `timeout N` inside the headless `sh -c` wrapper, which pane mode does not construct. Silently ignoring the flag would let an orchestrator believe a bound is enforced when nothing enforces it — precisely the class of silent non-enforcement the `failed (no-result)` state exists to prevent elsewhere.
**Rejected**: Silently ignoring `--timeout` under `--pane` (a false guarantee). Implementing a pane-side timer (re-introduces the supervisor process the dispatch design deliberately has none of).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

### `status --json` omits `server`; `logs` is the copy-pasteable capture source
**Decision**: The `--json` surface carries `mode` plus `pane`/`window`, but **not** `server`. The complete, socket-aware `fab pane capture [-L <server>] <pane>` command is printed by `fab dispatch logs` on a pane dispatch, and readers are pointed there rather than at `--json`.
**Why**: `logs` is where a reader lands when they want to see a pane worker's output, so printing the exact command — socket included when the record carries one — is the actionable answer at the moment of need, and it cannot be assembled wrong. Adding `server` to the JSON is an additive, backward-compatible change available whenever a programmatic consumer needs it; documenting the gap is cheaper than shipping a field with no consumer.
**Rejected**: Telling readers to hand-assemble the capture command from `--json` (they would silently omit `-L` and get an empty capture against the wrong socket). Adding `server` to the JSON speculatively (a surface with no consumer).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch
