---
type: memory
description: "`fab dispatch {start,open,ready,deliver,restart,status,wait,logs,kill,reap,clean}` manages headless and pane stage workers: pane open/ready/deliver (including resume), headless start, preference-bounded pane → native → headless descent, valid user providers that omit interactive grammar, identity-checked pane liveness (the `pane_pid` restart-alias discriminator), state layout, observation, recovery, placement, and reaping."
---
# fab dispatch

**Domain**: runtime

## Overview

`fab dispatch` is the **process manager for CLI-dispatched pipeline stages**, in **two launch modes** that share one state directory, one loader, one concurrency check, one launch path, and one state-string vocabulary:

| Mode | Requirement | Entry | Worker | Command composed | Completion observed via |
|------|-------------|-------|--------|------------------|-------------------------|
| **pane** | reachable tmux + `interactive_command` | `open` → `ready` → `deliver` | interactive tmux pane | provider `interactive_command`, verbatim | result file + pane liveness |
| **native** | `native: true` | (none — the harness) | native Agent-tool subagent | none in this command family | dispatching harness |
| **headless** | `headless_command` | `start` | detached `sh -c` process | provider `headless_command` | exit file + pid liveness + result file |

Headless mode is tmux-independent; pane mode restores watch-and-steer through the interactive CLI surface. **The two modes have separate entries**, because they hand a worker its prompt in fundamentally different ways: `start` launches a headless worker in one step with the prompt on its stdin, while a pane worker is spawned by `open` and handed its prompt afterwards by `deliver`, with the agent-driven readiness gate (`ready`) in between — a freshly spawned agent TUI may still be booting or parked behind a first-run wall, and answering one is judgment a binary cannot run. `deliver --prompt-file` is also the **pane-arm resume**.

`start`, `open`, and `restart` launch these two non-native adapters, but automatic selection evaluates the full catalog: it starts at `dispatch.mode` and descends pane → native → headless. If native is the first possible rung, the command errors before writing prompt or dispatch state and directs the caller back to native dispatch; if `start`'s selection lands on **pane**, it errors the same way and names `open`. `open` runs no ladder at all — pane is explicit there. `restart` relaunches a non-running attempt from the persisted prompt and re-runs the ladder against current capabilities and environment. `status` and `wait` remain one-shot and blocking views of the same derived state.

Dispatch is the runtime for cross-harness stage dispatch. It re-resolves the stage, provider, profile, preference, and current tmux reachability; the earlier `dispatch=` value is visibility for the skill, not an executable handoff. Provider fields remain pure capabilities, while `dispatch.mode` owns preference. Headless dispatch stays independent of the tmux-bound `fab pane` / `fab operator` family; pane dispatch borrows tmux as a launch surface without joining the operator's monitored set.

The **skill wiring** consumes it: the dispatch-seam skills branch on the resolved `dispatch=` line and, when present, drive this command family. The `dispatch=` line is **unlabelled** — it carries a substituted command and never names the rung that produced it — so the wiring **attempts `start` first and lets its answer be the discriminator**: a headless landing launches, and a pane landing is refused before stdin is read, before the refuse-if-running check, and before any state write, so the identical invocation re-runs as `open` with nothing consumed. From there: `start` (block prompt on stdin) or `open` → `ready` gate → `deliver` → a **blocking `fab dispatch wait <change> <stage> --timeout 300`**, run as a *background* command wherever the harness can re-invoke the agent on exit (foreground blocking is the cross-harness fallback) → the mode's reachable states → read `{stage}-result.yaml` on `done`, then `fab dispatch reap` at the stage-aware moment (§ `fab dispatch reap`). The wiring is **push, not poll**: the orchestrator spends turns only when the wait returns, and a `running` return (the bound expired) is its peek-on-suspicion moment, which is why `--timeout 300` is a *peek cadence* rather than a poll interval. It lives in `_preamble.md` § CLI-Adapter Dispatch + § The pane readiness gate + § Dispatch-Prompt Obligations, where **pane mode is an option inside the `dispatch=`-present arm, never a third branch** (see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership and [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution).

Source: the testable core lives in `internal/dispatch` (state read/write, wrapper composition, both state derivations, the `Wait` control loop in `wait.go`, the reap guard in `reap.go`, process signaling, and the pane placement *policy* in `pane_mode.go` — `SelectMode`, `SelectPaneShape`, `SplitTarget`, `SiblingDispatchPane`); the readiness gate, the verified-delivery choreography, and the tmux pane mechanics live in `internal/pane` (`gate.go`, `create.go` — see [pane-commands.md](/runtime/pane-commands.md) § Shared Pane Package), which the pane entry verbs bind over. Thin cobra wiring lives across `cmd/fab/dispatch.go` (parent) + `dispatch_start.go` / `dispatch_open.go` / `dispatch_ready.go` / `dispatch_deliver.go` / `dispatch_restart.go` / `dispatch_status.go` / `dispatch_wait.go` / `dispatch_logs.go` / `dispatch_kill.go` / `dispatch_reap.go` / `dispatch_clean.go` — mirroring the `internal/pane` + `pane*.go` split precedent. `dispatch_start.go` owns the **shared launch path** (`runDispatchLaunch` + the `promptSource` seam) and the **shared flag surface** (`addLaunchFlags`, which binds a `launchFlags` struct to a per-verb flag set — `start` registers `--timeout`/`--headless` plus hidden `--pane`/`--server` that raise the `open` guidance, `open` registers `--server` alone and forces pane, `restart` keeps the full set — plus its `resolveMode` method carrying the `SelectMode` and `SelectPaneShape` calls, the one place `$TMUX` and `$TMUX_PANE` are read). `open` and `restart` add only their own cobra commands: `open`'s pane-forced flag set, `restart`'s help strings and its `promptFromStateDir` source. The gate's tmux surface is the `PaneIO` interface, whose real implementation delegates to `internal/pane`'s shared `Capture`/`SendLiteral`/`SendKey` helpers — the same helpers the `fab pane` primitives ride.

## Requirements

### Requirement: `fab dispatch` command family

The `fab` binary SHALL expose a top-level command group `fab dispatch` with eleven subcommands — `start`, `open`, `ready`, `deliver`, `restart`, `status`, `wait`, `logs`, `kill`, `reap`, `clean` — always-routed through the `fab` router. Its top-level name MUST NOT collide with the `fab-kit` `LifecycleCommands` allowlist (pinned by `TestNoTopLevelCommandCollidesWithRouterAllowlist`; `dispatch` is not in the allowlist). It is a new fab-go command group registered via `dispatchCmd()` in `cmd/fab/main.go`'s `newRootCmd()`. See [distribution/kit-architecture.md](/distribution/kit-architecture.md) for its place in the fab-go command inventory.

### Requirement: The pane entry verbs are thin bindings over the `fab pane` primitives

`open`, `ready`, and `deliver` SHALL delegate their tmux mechanics to `internal/pane` — the gate classifier (`NewGate`/`Probe`/`Deliver`/`DeriveReadiness`), the creators (`OpenWindow`/`OpenSplitPane`/`OpenPlainPane`/`SplitPlacement`), liveness/kill (`PaneAlive`/`KillPane`), `Tail`, and the pointer composer (`PointerPrompt`) — the same machinery the provider-generic `fab pane open`/`ready`/`deliver` verbs run addressed by pane id (see [pane-commands.md](/runtime/pane-commands.md)). What remains dispatch-owned is the record bookkeeping: record load/save, refuse-if-running, the mid-stage worker guard, the stash/restore of completion signals around delivery, result-file clearing, the `delivered` marker, and the placement *policy* (`SelectMode`/`SelectPaneShape`/`SplitTarget`/`SiblingDispatchPane`). The verbs' external contract — flags, output forms (`opened …` / `delivered …`, readiness reports carrying pane+socket+snippet), and exit behavior — is unaffected by the layering.

### Requirement: POSIX-only v1 (the headless launch/signal syscalls)

The **headless** `fab dispatch start` (and `kill`) SHALL error clearly on non-POSIX platforms rather than half-working — the message names the POSIX-shell requirement (`setsid`/`timeout`). The guard is a **compile-time platform split**, not a runtime `runtime.GOOS` string check: `dispatch_posix.go` (build tag `!windows`) owns the launch/signal syscalls; `dispatch_windows.go` (build tag `windows`) provides the same signatures returning the POSIX-only error (with `Alive` conservatively `false`), so the package compiles on Windows and the error surfaces at the command layer. This mirrors the `proc_{linux,darwin}.go` / `pane_process_{linux,darwin}.go` precedent.

Pane mode's tmux mechanics (`ServerReachable`, `OpenWindow`, `OpenSplitPane`, `PaneAlive`, `KillPane`) and its placement policy (`SiblingDispatchPane`/`SplitTarget`) live in the platform-**independent** core (`internal/pane/create.go` and `internal/dispatch/pane_mode.go` respectively) for the same reason `WrapperArgv` does: they are plain tmux subprocess calls with no syscall dependency, so they compile everywhere. Pane mode is still unusable where tmux is absent, but that surfaces as `ServerReachable`'s actionable error, not as a compile-time platform split.

#### Scenario: Windows build errors instead of launching

- **GIVEN** a `GOOS=windows` build
- **WHEN** headless `fab dispatch start` is invoked
- **THEN** it returns an error naming the POSIX-shell requirement and launches nothing

### Requirement: `.fab-dispatch/{id}/` state layout

Each dispatch's state SHALL live under `.fab-dispatch/{4-char-change-id}/` at the **repository root** (`filepath.Dir(fabRoot)`), keyed by the stable 4-char change ID (not the slug, so it survives `fab change rename`). This sits alongside the `.fab-status.yaml` repo-root ephemeral-state convention (the `.fab-runtime.yaml` sibling that convention once also named is gone (ioku) — agent-state production is divested — see [runtime-agents.md](/runtime/runtime-agents.md)), and each git worktree naturally gets its own dir. **No gitignore/scaffold/migration work is required** — the scaffold `fragment-.gitignore` `.fab-*` pattern already matches `.fab-dispatch/`. The dir name is the `internal/dispatch` named constant `DirName = ".fab-dispatch"`; per-stage filenames derive from named suffix constants (no magic strings). **Both modes share the dir**, the loader, the save path, and the refuse-if-running check.

Per-stage files under `.fab-dispatch/{id}/`:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start` / `open` (from stdin) | the stage prompt — piped to the dispatched command's stdin (headless) or **pointed at** by the one-line prompt `deliver` types into the pane worker. Written identically in both modes; only the hand-over differs. It is also `restart`'s **input**: `restart` reads it and leaves it byte-identical, never re-writing it |
| `{stage}-continuation.md` *(convention)* | the orchestrator | a rework-cycle continuation prompt for the pane-arm resume, handed over with `deliver --prompt-file`. fab neither writes nor requires it — the path is the wiring's convention (`_preamble.md` § Pane-arm continuation), the record walker ignores non-`.yaml` files, and it is removed with the rest of the dir by the two cleanup paths |
| `{stage}.yaml` | `start` / `open` (via `internal/atomicfile`), plus `deliver`'s marker flip | the `Dispatch` state struct — `spawn_cmd` (resolved) + `started_at`, plus the mode's identity: `pid`/`pgid`/`timeout` (headless) or `pane`/`window`/`server`/`pane_pid`/`delivered` (pane). `pane_pid` is the pane's shell pid recorded at open-time — the restart-alias liveness discriminator (§ Pane liveness is identity-checked). `window` holds the `fab-{id}-{stage}` **identity string** in both pane shapes, so its meaning is *"tmux window name (new-window shape) or tmux pane title (split shape)"* — the string is identical either way, which is why it stayed one field with no schema change and no migration. `delivered` records that the worker has been handed its prompt. Every mode-specific key is `omitempty`, and the mode is **derived** from which keys are present (no stored discriminator). File paths are **derived** from the dir convention, not stored |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command — **headless only** (a pane worker's output is tmux scrollback) |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its **presence** is the "process finished" signal; **headless only** |
| `{stage}-result.yaml` | the **dispatched agent** (contract) | the stage result; dispatch defines only the path + consumes its presence. Presence is required for `done` in both modes and is the **sole** completion signal in pane mode. Its **content schema** is a minimal YAML envelope mirroring each native block's return — common `stage`/`status`/`summary`; apply adds `failed_task`/`reason` on failure; review adds `verdict` (pass\|fail) + `findings{must_fix,should_fix,nice_to_have}`; hydrate carries only the envelope. The **`status` (worker/infra outcome) vs `verdict` (review outcome) split is load-bearing** — a completed review with `verdict: fail` is dispatch-state `done` (result present), and the orchestrator then takes the normal review-fail path; dispatch-state `failed` is reserved for worker/infrastructure failure. Schema documented in `_preamble.md` § Dispatch-Prompt Obligations. |

#### Scenario: a headless record's on-disk shape is unchanged by pane mode's fields

- **GIVEN** a headless dispatch
- **WHEN** `{stage}.yaml` is serialized
- **THEN** it carries `pid`/`pgid`/`spawn_cmd`/`started_at` and **none** of `pane`/`window`/`server`/`pane_pid`/`delivered`
- **AND** GIVEN a pane dispatch, the record carries `pane`/`window`/(`server` when set)/(`pane_pid` when the open-time pid read succeeded)/(`delivered` once true) and no `pid`/`pgid` — every addition is additive on disk, so no migration exists or is needed

### Requirement: `fab dispatch start <change> <stage> [--timeout <secs>] [--headless]` is HEADLESS-ONLY

`start` SHALL launch **only the headless arm**. It runs the **shared prologue**: resolve `<change>` to its 4-char ID (via `internal/resolve` — ID / folder substring / full name), load config, resolve the stage's role → provider profile via `internal/agent` + `internal/spawn.WithProfile` (the same `{model}`/`{effort}` substitution `fab resolve-agent` performs), enforce refuse-if-running, obtain the stage prompt through a **`promptSource` seam** (`start`: stdin, persisted into `{stage}-prompt.md`), and clear stale per-stage files; the tail launches the worker and persists `{stage}.yaml` before returning. The prologue SHALL NOT be duplicated across the verbs that run it — `start`, `open`, and `restart` share it via `runDispatchLaunch`.

**Pane mode's entry is `open`, not `start`**, because a pane worker is spawned and delivered to in separate steps with an agent-driven readiness gate between them — a shape a single-shot launch has nothing to map onto. `start` SHALL therefore refuse a pane landing **before any state write**, in both shapes it can arrive:

| Typed / resolved | Result |
|------------------|--------|
| `--pane` or `--server` | a refusal naming `fab dispatch open`, then `ready` and `deliver`. Both flags stay **registered but hidden** precisely so this guidance is reachable, where a removed flag would give cobra's bare `unknown flag` |
| the ladder LANDS on pane (a `dispatch.mode: pane` preference whose prerequisites hold) | an error naming the stage, the provider, and `fab dispatch open` (plus the `--headless` force), mirroring the automatic-native redirect's shape |

The pane refusal fires **after** the descent has had its chance, which is load-bearing: a stale `$TMUX` makes the ladder pick pane, and descending to headless on the failed reachability probe is exactly what keeps an unattended `start` working there. Only a pane rung that survived validation is a genuine "you wanted a watchable worker" landing.

Output names the launched identity. Automatic success also reports `mode: <rung> (preferred)` or `mode: <rung> (descended: <reasons>)`; forced-mode output carries no automatic-selection suffix.

#### Scenario: a pane landing sends the caller to `open`

- **GIVEN** `dispatch.mode: pane`, a reachable tmux server, and a provider with an `interactive_command`
- **WHEN** `fab dispatch start <change> <stage>` runs with a prompt on stdin
- **THEN** it exits non-zero naming `fab dispatch open`, launches nothing, and writes no record
- **AND** GIVEN `--pane` or `--server <name>` typed on `start`, it exits non-zero with the same `open` guidance

### Requirement: Explicit signals precede preference-bounded automatic descent

The launch verbs SHALL honor explicit signals first: `--pane` selects pane; `--headless` or `--timeout` selects headless; `--server` selects pane. Explicit selections are hard requirements and continue to key on whether a flag was supplied. With no explicit signal, they use `dispatch.mode`, provider capabilities, and tmux availability. Selection starts at the configured rung, never ascends, and returns the first possible rung in pane → native → headless order. The pane-selecting signals are live only on `restart`; on `start` they raise the `open` guidance, and `open` needs none of them because pane is its premise.

`dispatch.SelectMode` is pure: it receives explicit signals, preferred mode, provider capabilities, and a tmux-availability value, and performs no environment read or tmux probe. `fab resolve-agent` supplies environment-derived availability; start/restart supply a real reachability result before any state write.

`$TMUX` remains a cheap availability input for the pure resolver, not a reachability guarantee. The launch verbs use `ServerReachable` and re-run the ladder with tmux marked unreachable when the probe fails. An explicitly requested pane — `restart --pane`, or `open`, whose pane mode is forced — still probes and hard-errors instead of descending.

`$TMUX_PANE` is a **separate signal that does not enter this ladder at all** — it selects the pane *shape* (split vs. new window) only after pane mode has already been chosen (see § A pane worker splits the dispatching agent's window). Keeping the two decisions apart is what lets `SelectMode`'s matrix stay unchanged by the split shape.

#### Scenario: automatic selection descends without ascending

- **GIVEN** no mode flag, `dispatch.mode: pane`, unreachable tmux, and a native-capable provider
- **WHEN** `fab dispatch start <change> <stage>` runs
- **THEN** selection descends to native and the command exits before writes with native-dispatch guidance
- **AND** if native is unavailable but `headless_command` exists, it descends again and launches headless

#### Scenario: an explicit signal always beats auto

- **GIVEN** `$TMUX` set (auto would select pane)
- **WHEN** `--headless` or `--timeout N` is supplied
- **THEN** the mode resolves to headless with no automatic-selection suffix

### Requirement: `--headless` is the explicit opt-out; `--pane` + `--headless` is a usage error

A `--headless` boolean flag SHALL exist as the explicit opt-out from auto pane selection — the escape hatch for an unattended run that happens to live inside a tmux tab. On `restart`, where both pane-selecting flags are live, `--pane` + `--headless` SHALL be a **usage error** (exit 2) enforced by cobra's `MarkFlagsMutuallyExclusive`, which fires during `ValidateFlagGroups` **before any `RunE` work**, so it structurally cannot leave partial state: nothing is launched and no dispatch record is written. `--headless` + `--timeout` SHALL **compose** (both select headless). `--pane` + `--timeout` SHALL likewise be an error on `restart`, raised in `resolveMode` before any launch or file write: `--timeout` is enforced by the headless wrapper, which pane mode never constructs, so accepting it there would advertise a bound nothing enforces. Only the **explicit** `--pane` conflicts — a bare `--timeout` is itself a headless rung of the ladder, so `--timeout` inside tmux selects headless rather than erroring. On `start` neither pairing arises: a pane flag raises the `open` guidance first.

#### Scenario: contradictory mode flags are rejected before anything happens

- **GIVEN** `fab dispatch restart <change> <stage> --pane --headless`
- **WHEN** it runs
- **THEN** it exits non-zero naming both flags, launches nothing, and writes no dispatch record

**Headless tail — detach mechanism — `SysProcAttr{Setsid:true}` on a plain `sh -c`, NOT the `setsid` binary.** The launch runs the wrapper `sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'` via `exec.Command` with `SysProcAttr{Setsid:true}` — Go's syscall attribute puts the child in a **new session/process group** so the dispatch survives the orchestrator dying, with no Go supervisor process in the loop (the shell records the exit code itself, so resumability falls out: a resumed skill reattaches via `fab dispatch status` instead of re-running the stage). The recorded `pid`/`pgid` therefore track the **live worker shell**. The intake's `setsid sh -c` string described the *intent* (new session, survives orchestrator death); the `SysProcAttr` attribute delivers that intent while keeping the tracked pid on the worker (see Design Decisions — an end-to-end smoke test showed the `setsid` **binary** double-forks, leaving the Go-recorded pid pointing at an immediately-exiting process and breaking liveness/refuse-if-running/kill). `WrapperArgv` is therefore always `[sh -c <script>]` with **no `setsid` prefix**.

**Timeout is enforced entirely inside the wrapper** via POSIX `timeout N <cmd>` when `--timeout N` is given — self-contained, no Go timer, no background sweep, no daemon. A timed-out command exits `124` (POSIX `timeout` convention), which surfaces as `failed` via the normal exit-code path.

#### Scenario: detached launch persists tracked state

- **GIVEN** a change/stage whose resolved role's provider carries a `headless_command`
- **WHEN** `fab dispatch start <change> <stage>` runs with a prompt on stdin
- **THEN** the prompt is persisted, the command is launched detached in a new session/process group, and `{stage}.yaml` records the pid/pgid/spawn_cmd/started_at
- **AND** with `--timeout N`, the resolved command is wrapped in POSIX `timeout N` inside the same `sh -c` wrapper

### Requirement: `fab dispatch open <change> <stage> [--server <name>]` spawns a pane and delivers nothing

`open` SHALL be **pane mode's entry**. It runs the same shared prologue `start` runs — same resolution, same `internal/spawn.WithProfile` substitution, same refuse-if-running check, same stale exit/result/log clearing, same stdin prompt persisted to `{stage}-prompt.md` — then composes the resolved provider's **`interactive_command`** (the same string `fab agent` composes) and opens it in a tmux **pane** whose cwd is the repo root, persisting the pane's **pane ID**, the `fab-{id}-{stage}` identity string, and the tmux socket label in `{stage}.yaml`. The composed command reaches tmux **verbatim, with no prompt argument appended**: `deliver` hands the worker its prompt afterwards, which is what decouples pane capability from whether a provider's CLI accepts a positional initial prompt. (WHERE the pane opens — split into the dispatching agent's own window, or a new window — is a second decision; see § A pane worker splits the dispatching agent's window.) Shell expansions the composed command carries (e.g. `$(basename "$(pwd)")` in the built-in claude `interactive_command`) expand at invocation inside the new pane — the `_cli-agents.md` § Spawn Composition contract. The **pane ID**, not the identity string, is the recorded identity: it is server-global, stable for the pane's lifetime, and exempt from tmux's target-grammar prefix/glob resolution, so liveness probes and kills are exact where a name-based target could resolve to a window the user renamed into place. `-P -F '#{pane_id}'` on the creating verb prints it, avoiding a follow-up lookup that could race a fast-exiting worker. `open` also records the pane's shell pid (`#{pane_pid}`) as `pane_pid` in the record — the liveness discriminator against the restart-alias hole (§ Pane liveness is identity-checked); a failed pid read warns on stderr, leaves the key absent, and never fails the launch.

**Pane is EXPLICIT on `open`, never a ladder result**: `open` opens a pane or it errors. An unreachable tmux server or a provider with no `interactive_command` is a hard error that launches nothing and persists nothing, rather than a silent descent to headless — which would be the opposite of what the caller asked for. `open` accepts no `--pane` (redundant), no `--headless`, and no `--timeout` (a headless-wrapper bound pane mode never constructs).

`--server <name>` / `-L <name>` targets a tmux socket (`tmux -L <name>`), mirroring the `fab pane` family's persistent flag, and is persisted so `status`/`kill`/`ready`/`deliver` reach the same server without re-supplying it. Naming a socket also keeps the **new-window** shape, since the caller's own `$TMUX_PANE` id is meaningless on another server.

Output is `opened <id>/<stage> (pane %N, split, title fab-<id>-<stage>)` or `opened <id>/<stage> (pane %N, window fab-<id>-<stage>)`. The verb is **`opened`, not `dispatched`** — the pane exists, but the stage has not been handed over yet.

#### Scenario: pane launch persists pane identity and no pointer

- **GIVEN** a change/stage whose resolved role's provider carries an `interactive_command`, and a reachable tmux server
- **WHEN** `fab dispatch open <change> <stage>` runs with a prompt on stdin
- **THEN** a tmux pane runs the composed interactive command **verbatim** (no trailing pointer argument), `{stage}.yaml` records `pane`/`window`/(`server`)/`spawn_cmd`/`started_at` with `delivered` unset, `{stage}-prompt.md` holds the stdin bytes, and no `{stage}.exit` wrapper is involved

### Requirement: `fab dispatch ready <change> <stage>` is a mechanical, purely echo-based probe

`ready` SHALL answer one question about an opened pane — *can it accept typed input right now?* — and report exactly one of `ready`, `booting`, or `parked`, derived **only** from a literal sentinel send plus pane captures: never from `@rk_pane_agent_state`, never from a pattern table of known dialogs. It sends the sentinel with `send-keys -l` (never submitted), takes the captures, clears the sentinel with `C-u` whether or not it echoed, and presses no other key.

| Report | Condition |
|--------|-----------|
| `ready` | the sentinel echoed — the pane accepts typed input |
| `booting` | no echo, and the screen is blank or **changed** between two captures spaced by an internal stability delay — a TUI still painting itself |
| `parked` | no echo on a **stable, non-blank** screen — a dialog, survey, login wall, or wedged process is holding the input |

The classification is the pure `DeriveReadiness(echoed, first, second)`, matching the package's `SelectMode`/`DerivePaneState` precedent; the blank-screen case precedes the difference check because two identical *empty* captures are stable by the letter of the rule while meaning the opposite of parked. **Both stability captures are taken BEFORE the sentinel is cleared**: `C-u` is itself a keystroke, and a TUI that repaints its input line in response would make every straddling capture pair differ, so a genuinely parked pane would read `booting` forever.

Every non-`ready` report SHALL carry the pane ID, the record's socket when non-empty, and a trailing capture snippet — everything a judgment round needs to answer the wall with `tmux [-L <server>] send-keys` without a second lookup. The snippet is the last 20 lines counted **after** the pane's trailing blank padding is dropped, so a dialog drawn near the top of a tall pane is what a reader sees; an empty snippet prints no header at all.

All three classifications exit **0** — the report string is the sole discriminator, the `fab dispatch wait` precedent. Non-zero is reserved for real errors: no record, a headless record, a dead pane, a mid-stage worker (§ `deliver` refuses a mid-stage worker — the guard is shared because the probe is a sender too), or a tmux failure. The probe is idempotent and safe to re-run (Constitution III).

#### Scenario: the three classifications

- **GIVEN** a live pane whose TUI accepts input
- **WHEN** `fab dispatch ready <change> <stage>` runs
- **THEN** stdout is exactly `ready` and the sentinel has been cleared with `C-u`
- **AND** GIVEN a pane parked at a trust dialog that swallows the sentinel on a stable screen, the report is `parked` followed by `pane:`, an optional `server:`, and the snippet
- **AND** GIVEN a pane whose screen is still changing between the two captures, the report is `booting`

### Requirement: `fab dispatch deliver <change> <stage> [--prompt-file <path>]` is the sole, verified delivery

`deliver` SHALL be the **only** mechanism that hands a pane worker its prompt — for both initial dispatch and rework-cycle continuation — and SHALL verify every step that could silently do nothing. Per attempt: an internal readiness probe → `C-u` → capture the cleared baseline → type the one-line pointer literally → **capture-verify that the pointer newly echoed** → `Enter` → **confirm the screen advanced**. A failed check costs one attempt; there is exactly **one retry** (reported as a `warning: delivery attempt 1 failed (…); retrying` on stderr even when the retry succeeds), and a second failure exits non-zero with the pane's last lines on stderr. A pane that failed verification twice needs eyes, not a loop.

Two properties of the echo check are load-bearing:

- **The baseline is captured after the attempt's OWN `C-u`**, not from the preceding probe. A pointer typed but never submitted is exactly what a busy-check failure leaves on the input line, so baselining against the probe capture would compare one occurrence against one, report `did not echo`, and kill the retry for the very failure class it exists to recover.
- **Echo counting ignores whitespace AND box-drawing runes** (`countWrapped`, which drops U+2500–U+257F alongside every whitespace rune), because tmux hard-wraps a pane's visible lines at the pane width, so a pointer longer than a narrow pane arrives split across lines — and *what* lands between the halves depends on how the TUI frames its input line. Both shapes are probed live: a **borderless** box inserts nothing but whitespace (claude at 50 and at 30 columns, narrow enough to wrap mid-word, and agy at 50 columns, all drawing the box as horizontal rules), while a **side-bordered** box interleaves its own frame — kimi 0.34.0 draws `│` down both sides of its input box, so a wrap lands `││` between the halves (probed 2026-08-10). Dropping both classes from both sides is what keeps the check wrap-independent without knowing the pane's width *or* its frame style. The drop is range-scoped rather than a broader "ignore non-alphanumerics" normalization: a `ReadySentinel` and a prompt-file pointer line never legitimately contain frame runes, so removing them cannot mask a genuine mismatch, and the failure mode of a wrong answer remains a loud double failure into the gate's escalation, never a false success.

With no flag the pointer names `{stage}-prompt.md`; **`--prompt-file <path>` points it at a continuation prompt instead — the pane-arm resume** (see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Apply-worker continuation). A missing prompt file, either spelling, is a refusal before any keystroke: a pointer at a file that is not there would type cleanly, verify cleanly, and leave the worker reading nothing.

**The previous attempt's `{stage}-result.yaml` and `{stage}.exit` are taken out of the way before the first send, and restored if no attempt verifies.** Clearing them is what makes a continuation read `running` again instead of letting the next `wait` return immediately on the last cycle's result; restoring them is what keeps a failed continuation recoverable — a record left at `delivered: true` with no result derives `running`, which both `deliver` (below) and `open` (refuse-if-running) reject, so the pane-arm resume's mandatory fresh-dispatch fallback would need an undocumented `kill` to be executable at all. A partial stash failure returns the entries already removed so the same restore path covers it.

**The `delivered` marker is written only after verification succeeds.** A failed delivery leaves it unset, which is what lets a caller distinguish *"the worker never got its prompt"* from *"the worker got it and failed at the work"* — the distinction a spawn-time argument could not express.

#### Scenario: a verified delivery flips the marker

- **GIVEN** a pane opened by `open` with `delivered` unset
- **WHEN** `deliver` runs, the pointer echoes, and the screen advances after Enter
- **THEN** the record records `delivered: true` and stdout is `delivered <id>/<stage> (pane %N, prompt <repo-relative-path>)`
- **AND** GIVEN the pointer does not echo on the first attempt but the retry verifies, delivery succeeds with the retry warning on stderr
- **AND** GIVEN both attempts fail, it exits non-zero with the capture snippet, leaves `delivered` unset, and restores the stashed result/exit files

#### Scenario: a side-bordered input box does not defeat the echo check

- **GIVEN** a pane whose TUI frames its input box with vertical rules, so a wrapped pointer arrives in the capture with `││` interleaved between the halves
- **WHEN** the echo is counted — for `deliver`'s pointer or for `ready`'s sentinel, which share `countWrapped`
- **THEN** the occurrence is found, exactly as for a borderless box that inserts whitespace alone
- **AND** GIVEN a capture whose ordinary ASCII punctuation differs from the needle's, the drop being range-scoped keeps that punctuation in the comparison, so it yields no false match

### Requirement: `deliver` and `ready` refuse a worker that is mid-stage

Both mechanical senders SHALL refuse when the record is headless (naming `fab dispatch start`), when the pane is dead or restart-aliased (identity-checked liveness — naming `fab dispatch restart`, so no sentinel or pointer is ever typed into an impostor pane), and when the record is `delivered: true` **and** no result file is present — a worker executing its stage. That last refusal is the **code-level expression of the contract's no-input-injection rule**: between `open` and a successful `deliver` the pane holds no stage context and may be typed into, but a delivered worker never may. `delivered: true` **with** a result present is `done` — the worker finished and is sitting at its prompt — which is the sanctioned continuation case and SHALL proceed.

#### Scenario: a mid-stage worker's keyboard is unreachable

- **GIVEN** a pane dispatch that has been delivered and whose result file is absent (state `running`)
- **WHEN** `deliver` or `ready` runs
- **THEN** it exits non-zero naming the mid-stage worker and sends nothing
- **AND** GIVEN the same dispatch after its result file appears (state `done`), `deliver … --prompt-file <continuation>` proceeds

### Requirement: A pane worker splits the dispatching agent's window; a new window is the fallback

**WHERE** the worker pane opens SHALL be a second decision, independent of the mode ladder and made only after pane mode is chosen. It SHALL be resolved by the **pure** `dispatch.SelectPaneShape(paneMode, serverSet, tmuxPane)` — no environment reads, no tmux probe, no I/O, matching `SelectMode`/`DerivePaneState`'s shape in the package — with both env reads (`$TMUX`, `$TMUX_PANE`) performed in the cobra layer and passed down:

| # | Condition | Shape | tmux call | Identity carried by |
|---|-----------|-------|-----------|---------------------|
| 1 | `$TMUX_PANE` non-empty **and** no `--server` | **split** — a pane inside the **dispatching agent's own window** | `tmux split-window {-h -l <n>%\|-v} -t <target> -P -F '#{pane_id}' -c <repo-root> "<resolved-cmd>"`, then `tmux select-pane -t <new-pane> -T fab-{id}-{stage}` | the tmux **pane title** |
| 2 | `--server <name>` supplied | **new window** | `tmux -L <name> new-window -n fab-{id}-{stage} -P -F '#{pane_id}' -c <repo-root> "<resolved-cmd>"` | the tmux **window name** |
| 3 | `$TMUX_PANE` empty | **new window** | `tmux new-window -n fab-{id}-{stage} -P -F '#{pane_id}' -c <repo-root> "<resolved-cmd>"` | the tmux **window name** |

The shell-command argument is the resolved `interactive_command` **alone** in all three shapes — nothing is appended to it (§ Prompt delivery is post-spawn, verified, and send-keys-only). Dispatch pane workers deliberately do **not** carry the interactive spawn's `; exec "$SHELL"` shell fallback (see [agent-primitives.md](/runtime/agent-primitives.md) § Spawn composition): the pane mode's `running`/`done`/`orphaned` subset and `dispatch.reap_done` treat pane death as the worker's terminal event, so a surviving fallback shell would read as a live worker.

This realizes the **two-tier tmux hierarchy**: an **operator** opens worktree agents as tmux **windows** (that path is untouched), and each **worktree agent**'s stage workers appear as **panes beside it** — so a stage worker does not consume a window in the operator's (and run-kit's) window list. `--server` may name a socket other than the one the caller's pane lives on, where the caller's `$TMUX_PANE` id is meaningless (pane ids are server-global, not global); an empty `$TMUX_PANE` means the dispatcher — a headless orchestrator calling `open` — has no pane of its own to split.

**The identity string is shape-independent.** `WindowName(id, stage)` composes the same `fab-{id}-{stage}` string for both shapes and it is stored in the record's same `window` field, so there is **no schema change and no migration**. In the split shape it rides the **pane title** (`select-pane -T`), because a split pane has no window name of its own — its window is the dispatcher's. A **failed title set is non-fatal** (a stderr warning at most): the worker is already running and its pane ID — the real identity — is already in hand, so refusing the dispatch over a cosmetic label would be strictly worse.

**Everything downstream is pane-ID keyed and therefore shape-blind.** `PaneAlive`, `KillPane`, `fab pane capture`, refuse-if-running, and `DerivePaneState` all target the pane ID and needed **no change**. Killing a split worker's pane leaves the dispatching agent's window — and any sibling worker — intact, by plain tmux `kill-pane` semantics.

`restart` inherits the whole decision from the shared launch path (`resolveMode` resolves mode *and* shape), so it needs no restart-specific branch: a restart issued from inside a tmux pane splits that pane's window even when the prior attempt had opened a window.

#### Scenario: a dispatching agent's worker lands in its own window

- **GIVEN** a dispatching process whose `$TMUX_PANE` names a live pane, and no `--server`
- **WHEN** `fab dispatch open <change> <stage>` runs
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

**Sibling detection SHALL key on dispatch RECORDS, never on pane titles.** The pane launch collects the `pane:` field of every `.fab-dispatch/*/{stage}.yaml` record in the checkout whose `server:` **equals the socket being probed**, intersects that set with `tmux list-panes -t "$TMUX_PANE" -F '#{pane_id}'` (a `-t` pane target resolves to that pane's window), and keeps the **last** pane present in both — list-panes order is pane-index order, so the last match is the newest worker. A pane ID is the correct identity for the same reason `status`/`kill`/`capture` key on it: it is server-global and stable for the pane's lifetime. A pane **title** is not — a harness running inside the worker pane rewrites it via terminal escapes within seconds of spawn — so titles are **set** at spawn for identification only, and **no code path reads `#{pane_title}` for placement**. `{stage}-result.yaml` is not a record, and records with an empty `pane:` (every headless dispatch) contribute nothing.

The **server filter is exact equality**, because a pane ID is per-**socket** rather than global: a `%17` recorded by a `--server work` dispatch names a different pane from the `%17` on the default socket, so an unfiltered set could stack a worker onto an unrelated pane. Default-socket dispatches record `server: ""` and are matched by a default-socket probe under that same equality test — no special case. Enumeration scope is **every** record dir in the checkout, not only the active change's, since nothing stops one window from hosting two changes' workers; over-collecting is safe because the intersection with one window's live pane list **is** the liveness *and* same-window filter — a dead pane, or a live pane in another window, simply never matches, so no separate `PaneAlive` probe or window lookup is needed.

**The column invariant.** The first worker splits the dispatcher's own pane `-h`, **carving** the Left/Right column at `dispatch.column_width` percent (default 35, so the agent the user is watching keeps the rest); every later worker splits the last live recorded worker `-v`, stacking **inside** that column, unsized. Only the carving split is ever sized — sizing a stacking split would fight the user's own resizes within the column. fab issues **no `select-layout`**, never rearranges user-made panes, and never re-touches the vertical Left/Right separator once carved. This is a **creation-time rule, not an enforcement loop**: an already-mangled window is left alone until its panes die.

The decision is a `pane.SplitPlacement{Target, Direction, SizePercent}` returned by `SplitTarget(server, dispatcherPane, repoRoot, columnWidth)`, whose pure halves — `lastRecordedPane` (the intersection), `splitPlacement` (the decision) — plus `internal/pane`'s `splitArgs` (the argv renderer) are table-testable without a tmux server or a record tree, matching `SelectMode`/`DerivePaneState`'s shape in the package. The width is read from config in the cobra layer (`cfg.GetDispatchColumnWidth()`; see [_shared/configuration.md](/_shared/configuration.md) § `dispatch`) and rides the placement, so the "size the carve, never a stack" rule exists in exactly one place.

`SplitTarget` is `internal/dispatch`'s whole exported placement surface — the decision returns `internal/pane`'s exported `SplitPlacement`, whose tmux flags (`splitRight`/`splitBelow`/`sizeFlag`) and argv composer (`splitArgs`) stay package-scope in `internal/pane`, as does the sibling probe in `internal/dispatch`. The cobra layer reads a placement only through `SplitPlacement.Describe()` — the stacked-column wording its degraded-probe warning prints — so no caller handles a raw `-h`/`-v`.

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

### Requirement: Prompt delivery is post-spawn, verified, and send-keys-only in pane mode

In **both** modes the full stage prompt arrives on **stdin** and is persisted to `{stage}-prompt.md`. Headless mode pipes that file into the dispatched command's stdin at launch; pane mode SHALL hand the worker a **one-line pointer** naming the repo-relative prompt path (`pane.PointerPrompt` — *"Read <path> and execute it."*), **typed into the pane by `deliver` after the pane is open and the readiness gate has passed**. No code path appends a prompt argument to `interactive_command`. The prompt **content** is composed identically for every adapter — nothing about the block prompt is written differently for pane mode; only the hand-over differs.

The pointer is rendered **repo-relative** (a path outside the repo root, or an unresolvable root, falls back to the path as given, which still reads correctly from the pane's cwd), so the typed line stays short and portable across worktrees. It is typed **literally** through `send-keys -l`, so no shell escaping applies to it: it is never a word in a shell command. The resolved `interactive_command` reaches tmux **verbatim**: its shell expansions are deliberate and must expand inside the new pane (the pass-through philosophy — the command's own quoting is the resolver's/user's concern), and `shellQuote` remains in use for the headless `sh -c` wrapper's paths.

#### Scenario: a multi-thousand-token prompt reaches an interactive worker

- **GIVEN** a full stage prompt on stdin
- **WHEN** `fab dispatch open <change> <stage>` runs and `fab dispatch deliver <change> <stage>` follows
- **THEN** the full prompt lands in `{stage}-prompt.md`, the pane's command carries no prompt at all, and the worker receives only the typed one-line pointer to that path, readable from its cwd (the repo root)

### Requirement: Pane prerequisites hard-error when explicit and trigger re-descent when automatic

Pane requires a reachable tmux server and `interactive_command`. `open` — and an explicit `restart --pane`/`--server` — hard-errors on either missing prerequisite, launches nothing, and writes no state. Automatic selection instead records the failed pane reason and continues down the same ladder: `pane unavailable: no tmux`, `pane unavailable: tmux unreachable`, or `pane unavailable: no interactive_command`. Every built-in provider has an interactive command; the last shape remains valid for a wholly user-defined provider that deliberately omits one.

The first possible lower rung wins. A native-capable provider therefore redirects to native before any write; a non-native provider with `headless_command` launches headless. If neither lower rung is available, the launch verbs return the shared no-reachable-capability error. Command composition occurs only after final selection, so each rung reads only its own capability field.

Reachability is established by `tmux [-L <server>] list-sessions` via `ServerReachable`, not inferred solely from `$TMUX`. Headless and native selections perform no tmux launch work.

#### Scenario: unreachable tmux leaves no trace under `open`

- **GIVEN** no reachable tmux server (or an unreachable `--server` socket)
- **WHEN** `fab dispatch open <change> <stage>` runs
- **THEN** it exits non-zero naming tmux reachability and the `--server` option, and creates no `{stage}.yaml`

#### Scenario: a stale `$TMUX` re-runs the ladder

- **GIVEN** `$TMUX` set to a dead socket, no mode flag, and `dispatch.mode: pane`
- **WHEN** `fab dispatch restart <change> <stage>` runs
- **THEN** pane is marked unreachable and selection continues to native or headless according to the provider's lower capabilities

### Requirement: Each selected rung consumes only its own capability

Pane composes only `interactive_command`; native uses only `native: true`; headless composes only `headless_command`. Automatic descent skips unavailable rungs rather than substituting fields. Explicit pane/headless errors name the required provider key. Automatic selection with no reachable rung returns one actionable error and writes nothing.

#### Scenario: an interactive-command-only provider opens a pane and errors without it

- **GIVEN** a role whose provider carries an `interactive_command` but no `headless_command`
- **WHEN** `fab dispatch open <change> <stage>` runs with a reachable tmux server
- **THEN** the pane opens on the composed `interactive_command`
- **AND** GIVEN the same role, explicit `--headless` errors with the `headless_command` key hint
- **AND** GIVEN a provider with no `interactive_command`, `open` errors while automatic selection continues to the next lower capability

### Requirement: Refuse-if-running + last-attempt-only concurrency

Every launch verb — `start`, `open`, and `restart` — SHALL refuse if a dispatch for the exact `(change, stage)` pair is already `running`: reporting the live identity (`pid N` or `pane %N`), directing to `fab dispatch kill`, and leaving the running dispatch untouched (they run one shared check, so they cannot diverge). The check SHALL apply the **prior record's own mode's finished signal** — the *same* signal `status` derives that mode's state from, so a launch verb and `status` can never disagree about whether an attempt is still going:

| Prior record's mode | Still running when | Finished when |
|---|---|---|
| headless | `{stage}.exit` absent **and** pid alive | `{stage}.exit` present (the shell recorded a code) |
| pane | `{stage}-result.yaml` absent **and** pane alive (identity-checked) | `{stage}-result.yaml` present — **result presence wins over pane liveness** |

The pane row's "pane alive" is the identity-checked `paneWorkerAlive` (§ Pane liveness is identity-checked): a restart-aliased prior pane — the ID exists but its shell pid is not the recorded one — is NOT the worker, so the attempt is not running and a fresh launch overwrites it rather than refusing against an impostor.

The pane row's result-presence precedence mirrors `DerivePaneState` and is load-bearing: an interactive worker never exits on task completion, it sits at its prompt, so a liveness-only refusal would fire forever after a successful pane run and make a `done` attempt permanently un-overwritable — `status` reporting `done` while the launch verb insisted it was still running.

A launch verb over a **completed** prior attempt (done / failed / orphaned) SHALL overwrite its files — there is **no per-attempt history** (last-attempt-only: it removes the stale exit/result/log then re-saves `{stage}.yaml`), and the new attempt MAY use either mode regardless of the prior one's. Refuse-if-running is scoped per `(change, stage)`: different stages of the same change share `.fab-dispatch/{id}/` via distinct `{stage}.*` filenames and do not collide.

#### Scenario: refuses a live dispatch, overwrites a completed one

- **GIVEN** a `(change, stage)` dispatch whose pid is alive and `{stage}.exit` is absent
- **WHEN** `fab dispatch start` runs again for the same pair
- **THEN** it refuses with a clear error and leaves the running dispatch untouched
- **AND** GIVEN a completed prior attempt, a new `start` overwrites the prior `{stage}.*` files with no history retained

#### Scenario: a finished-but-still-alive pane worker is overwritable

- **GIVEN** a pane dispatch whose pane is still alive (the worker is sitting at its prompt) and whose `{stage}-result.yaml` is absent
- **WHEN** `fab dispatch open` runs again for the same pair
- **THEN** it refuses — the worker is genuinely still executing
- **AND** GIVEN that worker then writes `{stage}-result.yaml` while its pane remains alive
- **THEN** `status` reports `done` **and** a new `open` overwrites the attempt rather than refusing

### Requirement: `fab dispatch restart <change> <stage> [--timeout <secs>] [--pane] [--headless] [--server <name>]`

`restart` SHALL relaunch a **non-running** dispatch from the prompt `start`/`open` persisted at `{stage}-prompt.md`, so the caller does not need the block prompt in context (an orchestrator may have lost a multi-thousand-token prompt to compaction). It differs from the entry verbs in **the prompt's source** — all three run the same Go launch path (`runDispatchLaunch`, parameterized by a `promptSource`: `promptFromStdin` for `start`/`open`, `promptFromStateDir` for `restart`) — and in being the **one launch verb that still accepts pane**. Consequently `restart` SHALL carry:

- the **same prologue** — change resolution, config + role→provider re-resolution (**config only**: like the entry verbs, it exposes no `--provider`/`--model`/`--effort`), pane validation, refuse-if-running, and stale `{stage}.exit`/`-result.yaml`/`.log` clearing;
- the **same mode selector** and the full flag set with its exclusions; forced modes hard-error, while automatic mode re-descends against current capabilities and tmux reachability;
- the **same record shape**, and the same preferred/descended automatic reason text.

**A pane landing is HALF a relaunch.** When `restart` lands on headless it relaunches fully, as before. When it lands on **pane** it SHALL perform the `open` step alone: the pane is spawned, `delivered` stays unset, and stdout reports `opened …` rather than `dispatched …`. Go cannot run the readiness gate's judgment, so the missing half is handed back on **stderr** — a note naming `fab dispatch ready` and `fab dispatch deliver`. (`open` prints no such note; its own name says it.)

The launch **mode — and the pane shape — SHALL be re-derived from the current environment**, never inherited from the prior attempt (the record carries no mode or shape discriminator to inherit from), so a restart issued from inside a tmux pane splits that pane's window even when the prior attempt opened a window. A restart **is** a fresh attempt under last-attempt-only, so it SHALL introduce **no new state string, no attempt counter, no attempt history, and no `restarted:` marker** — `status` cannot distinguish a restart from an initial launch, by design. The prompt file is the **input**: `restart` reads it and leaves it **byte-identical** (re-writing it with its own bytes would only risk corruption on a partial write), while the *prior attempt's* exit/result/log are still cleared. Refuse-if-running SHALL **precede** the prompt read, and an absent prompt SHALL be a clear error — `no persisted prompt at <path> — nothing to relaunch; run \`fab dispatch start\` (headless) or \`fab dispatch open\` (pane) with the prompt on stdin` — that launches nothing and leaves any prior record intact.

The observation policy that *spends* a restart (one automatic restart on `orphaned`, peek-on-suspicion, escalation) is **skill-side**, not a CLI concern — see `_preamble.md` § CLI-Adapter Dispatch → *Recovery policy* and [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution.

#### Scenario: an orphaned attempt is relaunched from the persisted prompt

- **GIVEN** an `orphaned` dispatch for `abcd/apply` whose `apply-prompt.md` is on disk
- **WHEN** `fab dispatch restart abcd apply` runs
- **THEN** a new worker is launched with those persisted bytes as its input, the record is overwritten, and the prior attempt's stale exit/result/log are cleared
- **AND** the prompt file itself is left byte-identical

#### Scenario: the mode is re-derived, so a dead tmux server does not re-fail the restart

- **GIVEN** an `orphaned` **pane** dispatch and no reachable tmux server (`$TMUX` unset)
- **WHEN** `fab dispatch restart` runs with no mode flag
- **THEN** pane is marked unreachable and the selector chooses the first possible lower rung; native returns guidance before writes, while headless creates a headless-shaped record
- **AND** the prior pane identity does not leak into the new record
- **AND** GIVEN a configuration that still resolves to pane, the restart opens a fresh pane with nothing delivered, prints the `opened …` line on stdout, and names `fab dispatch ready` + `fab dispatch deliver` on stderr

#### Scenario: a live dispatch refuses, even with no prompt on disk

- **GIVEN** a genuinely running dispatch for `abcd/apply`
- **WHEN** `fab dispatch restart abcd apply` runs
- **THEN** it refuses with the `already running (…); run fab dispatch kill first` message and leaves the record untouched
- **AND** it does so even when `apply-prompt.md` is absent — the refusal check precedes the prompt read

#### Scenario: a missing prompt writes nothing

- **GIVEN** no `{stage}-prompt.md` for the pair
- **WHEN** `fab dispatch restart` runs
- **THEN** it errors naming the path and both entry remedies (`fab dispatch start` for headless, `fab dispatch open` for pane), launches no worker, and writes or overwrites no `{stage}.yaml`

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

**Pane** — reads result-file presence and **pane** liveness (identity-checked — the shared `paneWorkerAlive`, § Pane liveness is identity-checked); no exit file is ever written or consulted. It reports a **subset of three**:

| State | Condition | Meaning |
|-------|-----------|---------|
| `done` | `{stage}-result.yaml` present | the worker honored the result contract |
| `running` | result absent AND the pane is alive | still working (or sitting idle mid-task) |
| `orphaned` | result absent AND the pane is dead (killed / crashed / tmux server gone) or restart-aliased | no result will arrive |

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

### Requirement: Pane liveness is identity-checked (the restart-alias discriminator)

A tmux server's `%N` pane-id space resets on server restart while `{stage}.yaml` persists `pane: %17`, so after a restart the recorded ID can **alias onto an unrelated new pane** — a bare existence probe would then read a dead dispatch as `running`, and a targeting verb could aim a keystroke or a `kill-pane` at the impostor. Against that, every record-keyed liveness observation SHALL be identity-checked: the worker is alive iff the pane exists AND (no `pane_pid` was recorded — a legacy or degraded record — OR the pid read fails right now — an unreadable pid is not a mismatch, so the check degrades to existence-only — OR the pane's current `#{pane_pid}` equals the recorded one). The decision is the pure `PaneWorkerAlive(paneExists, recordedPID, currentPID, pidReadOK)` in `internal/dispatch` (no I/O, exhaustively table-testable — the `DeriveState`/`DecideReap` precedent), composed once by the cmd-layer `paneWorkerAlive` helper from `pane.PaneAlive` + `pane.GetPanePID`: the single shared liveness read for every record-keyed consumer, so the call sites cannot drift on what "the worker is alive" means.

A **mismatch means the pane is an impostor and the worker is gone**, and each consumer routes that to its existing gone-worker path: `status`/`wait` (via `observeDispatch`) derive `orphaned` — the existing recovery path — instead of `running`; refuse-if-running (`priorRunning`) does not refuse, so a fresh `open`/`restart` overwrites the orphaned attempt; `kill` and `reap` report already-gone and send no `tmux kill-pane`; `ready`/`deliver` refuse naming `fab dispatch restart` and type nothing — no sentinel or pointer ever lands in an unrelated pane; and `logs` reports the gone worker instead of printing a `fab pane capture` hint aimed at the impostor.

`DerivePaneState` itself is **unchanged** — it is a documented byte-stable cross-adapter contract, so the identity check lives in the computation of its `paneAlive` input, one seam up. Records written by older binaries carry no `pane_pid` (`omitempty`, zero value) and behave exactly as before (existence-only) at every consumer; the field is additive on transient per-change state (archive-time deletion + `fab dispatch clean`), so **no migration** ships.

#### Scenario: a restart-aliased pane reads orphaned, not running

- **GIVEN** a pane dispatch whose tmux server restarted mid-flight, so the recorded `%17` now names an unrelated pane and `{stage}-result.yaml` is absent
- **WHEN** `fab dispatch status` or `wait` observes it
- **THEN** the state is `orphaned`, not `running`
- **AND** WHEN `fab dispatch open` or `restart` runs for the same pair, it overwrites the orphaned attempt rather than refusing "already running"
- **AND** WHEN `kill` runs, it reports already-dead and sends no `kill-pane` at the impostor

### Requirement: `status --json` carries a `mode` discriminator plus the mode's identity

`--json` SHALL emit `{change, stage, state, mode, …}` where `mode` is `headless` or `pane`, followed by that mode's identity keys — `pid`, `pgid`, `exit?` (headless) or `pane`, `window`, `server?`, `pane_pid?`, `delivered` (pane). The other mode's keys are **omitted**, so a headless object is unchanged apart from the added `mode`, and `exit` stays absent for a pane dispatch (no exit file exists). `delivered` is reported **even when `false`** — a pane is opened and delivered to in two steps, so "opened but holding no prompt yet" is a case a consumer must be able to see — and it is bookkeeping, never a state: `state` is derived without it. The `mode` discriminator is what tells a consumer which state subset to expect. Keys evolve additively with no `schema_version`.

`server` (`omitempty`) carries the record's tmux socket label for a socket-scoped pane dispatch, so a consumer assembles `fab pane capture -L <server> <pane>` from `--json` alone; `fab dispatch logs` still prints the complete command (below). The key is absent for default-socket and headless dispatches.

`pane_pid` (`omitempty`) mirrors the record's open-time liveness discriminator (§ Pane liveness is identity-checked) — pane-only, absent when the record has none (an older binary's record, or a failed open-time pid read).

#### Scenario: the JSON shape names its mode

- **GIVEN** a pane dispatch
- **WHEN** `fab dispatch status --json` runs
- **THEN** the object carries `mode: "pane"` with `pane`/`window`/`delivered` populated and no `pid`/`pgid`/`exit`
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

**A pane dispatch keeps no log file** — an interactive worker's output is tmux scrollback, not a redirected stream — so `logs` SHALL report that fact and name the equivalent read (`fab pane capture <pane>`) instead of the generic missing-log message. When the record carries a `server`, the printed command SHALL carry the socket too (`fab pane capture -L <server> <pane>`), since a socket-scoped pane is unreachable from a default-socket capture. This report remains one copy-pasteable source for the capture command; the other is `status --json`, whose `server` key carries the same socket. The hint names a TARGET, so one identity guard applies: a record whose pane ID exists but whose `pane_pid` no longer matches (the restart-alias) gets a gone-worker report naming `fab dispatch restart`, with **no** capture command aimed at the impostor (§ Pane liveness is identity-checked).

#### Scenario: pane-mode logs points at the pane

- **GIVEN** a pane dispatch started with `--server work`
- **WHEN** `fab dispatch logs <change> <stage>` runs
- **THEN** it reports the no-log fact and prints `fab pane capture -L work <pane>`

### Requirement: `fab dispatch kill <change> <stage>`

`kill` SHALL terminate the dispatch by the mechanism its **derived mode** implies, and SHALL be **idempotent** in both cases — an already-dead target is a benign no-op with a clear report, and a missing dispatch record is a clear error rather than a panic or a signal to pid 0:

- **Headless** — `SIGTERM` to the **process group** (`pgid` from `{stage}.yaml`, via `syscall.Kill(-pgid, SIGTERM)`) so the detached command and its children die together. SIGTERM (graceful), not SIGKILL, matches "die together". Already dead (ESRCH) → an already-dead report.
- **Pane** — kills the **tmux pane** (`tmux kill-pane -t <pane>`), taking the interactive worker down with it. Already gone → an already-dead report naming the pane. The liveness probe is identity-checked (§ Pane liveness is identity-checked): a restart-aliased pane — the ID exists but its `pane_pid` mismatches — is the SAME already-dead no-op, so no `kill-pane` is ever sent at the impostor.

**No marker file is written in either mode**: with no result file present, a killed dispatch simply reads `orphaned` on the next `status`.

`kill` is the family's **recovery** verb — valid in **any** state, ungated by config, and the verb the pipeline's Recovery policy spends on a parked worker. It is distinct from `reap` (below), the **hygiene** verb, which fires only on `done`, only for a pane record, and only when `dispatch.reap_done` allows it.

#### Scenario: killing a pane leaves the dispatch orphaned

- **GIVEN** a live pane dispatch with no result file
- **WHEN** `fab dispatch kill <change> <stage>` runs
- **THEN** the pane is killed, a killed report is printed, and a subsequent `status` reports `orphaned`
- **AND** GIVEN the pane is already gone, `kill` exits 0 with an already-dead report

### Requirement: `fab dispatch reap <change> <stage>`

`reap` SHALL reclaim the tmux pane of a **finished pane-mode worker** — the pane-hygiene counterpart to `kill`. A pane worker never exits on completion (it writes `{stage}-result.yaml` and sits at its prompt, deliberately, so it can still be steered), so across a multi-stage pipeline every finished stage keeps its slice of the carved worker column and the panes the user actually watches shrink with each completed stage.

**WHEN the orchestrator calls it is stage-aware wiring, not a flag or a guard condition.** Every stage but apply is reaped immediately after its `done` result is read. The **apply** pane is not: it is the pane arm's resume target across rework cycles, so it survives until **review passes** (or the run stops or escalates past apply for good) — see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Apply-worker continuation and `_preamble.md` § CLI-Adapter Dispatch step 3, which own that rule. The Go guard below is unaffected either way: it fires on pane + `done` + knob whenever the call is made, which is why a call made against an existing record needs no skill-side mode or config check. The deferred apply reap is the one call the wiring gates on the arm, because it fires at a moment the pipeline reaches on every arm — including the native and headless arms, which wrote no pane record for reap to find.

It takes exactly two positional arguments and **no flags** — no `--json`, and no `--server`, because the socket comes from the record, so a `--server`-started dispatch is reaped on the right socket with nothing extra passed.

`reap` SHALL own the **whole guard** and kill the pane only when **all three** conditions hold:

1. the record is **pane-mode** (`IsPane()` — a `pane:`-bearing record), **and**
2. the **derived state is `done`** (`DerivePaneState`: `{stage}-result.yaml` present — pane liveness is irrelevant to the state), **and**
3. **`dispatch.reap_done` resolves `true`** (default `true`) through the four-tier config cascade.

Every other case SHALL be a **no-op with a one-line report naming its reason**, exiting 0:

| Case | Behavior |
|------|----------|
| record is **headless** | no-op — the detached process already exited; there is nothing visual to reclaim |
| state is **not `done`** (`running` / `orphaned`, or any headless state) | no-op — reap is NOT kill; it must never terminate a live or failed dispatch. The report points at `fab dispatch kill` |
| **`dispatch.reap_done: false`** | no-op — the user opted to keep done-worker panes and their scrollback |
| **pane already gone** (killed by hand, tmux server died) — or **restart-aliased** (pane ID exists, `pane_pid` mismatches) | benign already-gone report — mirroring `kill`'s idempotence, so a re-reap is safe, and no `kill-pane` is ever aimed at the impostor |
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

- **GIVEN** `fab dispatch open abcd apply --server work` (the new-window shape)
- **WHEN** the window is created
- **THEN** its name is `fab-abcd-apply` and carries no `»`/`›` prefix

#### Scenario: a split worker's pane title is identifiable and equally unmarked

- **GIVEN** `fab dispatch open abcd apply` from inside a tmux pane (the split shape)
- **WHEN** the worker's pane is created
- **THEN** its pane title is `fab-abcd-apply` and carries no `»`/`›` prefix, and no window is renamed

### Requirement: Steering a pane worker is contract-neutral

A user MAY converse with a running pane worker mid-stage. This changes **no** contract: the worker still owes `{stage}-result.yaml`, still ends with the terminal `fab status refresh <change>` epilogue, and still runs no `fab status` **transition** command — **the orchestrator owns every transition** (see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership). Steering is human input into the worker's context, exactly like answering a native sub-agent's question. A worker steered away from producing a result needs no new state: it never reaches `done` and surfaces through the never-`done` escalation the orchestrator already owns. This is **documentation only** — no code detects, gates, or reports a steered worker.

Steering by a *human* is unrestricted; the *pipeline*'s access to a worker's keyboard is not. It may type into a pane only between `open` and a successful `deliver`, where the pane holds no stage context to corrupt, and the two mechanical senders enforce that boundary themselves (§ `deliver` and `ready` refuse a worker that is mid-stage). `docs/specs/harness-adapters.md` § 3 owns the rule and its carve-out.

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

**Why**: `kill` is a **recovery** verb, valid in any state and already in the pipeline's sanctioned verb set with a documented never-kill-a-worker-awaiting-input rule; reap is **hygiene**, fires only on `done`, and is policy-gated. Folding them would put a config-gated no-op inside a verb whose contract is "terminate this now". Putting the guard in Go is what lets the call site stay dumb: the knob resolves through the four-tier cascade (environment > system `~/.fab-kit/config.yaml` > project > defaults), and a skill reading `fab/project/config.yaml` directly would miss both process-local and machine-wide `both`-scope preferences. One dumb call site also keeps the wiring identical across adapters and modes, since a headless record is a reported no-op inside the command.

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

**Why**: Capabilities and reachability may have changed when a worker died, so inheriting the prior attempt's mode would reproduce the failure. Re-running the configured descent ladder can land on native or headless after a pane server dies. A restart is still a fresh last-attempt-only run; counters and history remain outside the state model.

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
**Decision**: Extract state read/write, wrapper composition (`WrapperArgv`), both state derivations (`DeriveState`, `DerivePaneState`), process signaling, and the pane placement policy (`pane_mode.go`) into `internal/dispatch`, with thin `cmd/fab/dispatch*.go` cobra wiring. The gate, the delivery choreography, and the tmux pane mechanics live in `internal/pane` (`gate.go`, `create.go`), shared with the `fab pane` primitives.
**Why**: The status-state machines and wrapper composition are the testable core; the `internal/pane` / `internal/archive` precedent and the need to table-test the pure state machines independent of a launched process or a live tmux server make extraction the clear default. The tmux primitives sit in the platform-independent core (not the build-tagged split) because they are plain subprocess calls with no syscall dependency.
**Rejected**: Inline in `cmd/fab` (harder to table-test the state derivations without launching a real process).
*Introduced by*: 260702-6sgj-fab-dispatch-command; *Updated by*: 260810-1lah-provider-generic-pane-verbs

### Primitives in `internal/pane`, Policy in `internal/dispatch`
**Decision**: The gate, the verified-delivery choreography, the tmux pane creators/liveness/kill helpers, `Tail`, and `PointerPrompt` live in `internal/pane`; mode/shape/placement *decisions* and all record bookkeeping stay in `internal/dispatch`.
**Why**: The provider-generic `fab pane open`/`ready`/`deliver` verbs run the same mechanics addressed by pane id with no dispatch record; the import graph allows the split cleanly (`internal/pane` has no dispatch dependency and the gate needs none), so no sibling package is needed. One home for the tmux mechanics means the echo-verify cannot grow a divergent second copy, and the dispatch verbs keep their byte-identical external contract as record-keeping bindings.
**Rejected**: Ad-hoc `--no-record`/`--pane-id` flags on the dispatch verbs (muddies the five-state contract); a new sibling package (no import cycle exists to force one); moving `SplitTarget`/`SelectPaneShape` (they read dispatch records and config — pipeline policy).
*Introduced by*: 260810-1lah-provider-generic-pane-verbs

### Pane and Headless Share One Process-Manager Family
**Decision**: Interactive pane dispatch and detached headless dispatch share `fab dispatch`, state layout, concurrency, observation, and cleanup. Pane composes `interactive_command`; headless composes `headless_command`; `dispatch.mode` controls automatic preference and flags force a single invocation. What the two modes do **not** share is the entry: headless launches in one step (`start`), pane in three (`open` → `ready` → `deliver`).
**Why**: Both non-native adapters owe the same result artifact and lifecycle state. Sharing the command family avoids duplicate machinery while keeping adapter capability grammar independent. The entries diverge because the hand-over does: a headless worker is handed its prompt on stdin at launch, while a pane worker must be typed into once its TUI is ready — an interval no single-shot verb can bound.
**Rejected**: A parallel pane-dispatch family, a duplicate `pane_command`, or provider-command presence as mode policy. One shared entry verb for both modes (it would have to either skip the readiness gate or block on a human).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch; *Updated by*: 260808-yilt-dispatch-mode-descent-ladder, 260809-3oz7-pane-readiness-gate-sendkeys-delivery

### One Pure Selector Owns Preference-Bounded Adapter Descent
**Decision**: `dispatch.SelectMode` handles forced signals first, then starts at `dispatch.mode` and descends pane → native → headless against independent provider/environment capabilities. `fab resolve-agent`, `fab dispatch start`, and `fab dispatch restart` share it; `open` short-circuits it with pane forced, since pane is its premise rather than a result.
**Why**: A single pure matrix keeps resolver output and runtime launch aligned while allowing the launch verbs to contribute a real tmux reachability probe. Preference as a ceiling prevents the runtime from choosing a more interactive or native adapter than the user requested.
**Rejected**: Command-presence policy, `$TMUX` as the entire default, ascending fallback, skill-side selection, and separate resolver/runtime matrices.
*Introduced by*: 260808-yilt-dispatch-mode-descent-ladder

### Forced Failures Are Hard; Automatic Failures Continue the Ladder
**Decision**: Forced pane/headless selections hard-error on missing prerequisites. Automatic pane failures become named reasons and re-enter the selector with the failed capability removed; native or headless may then win. No command or state is written until a non-native launch rung is final.
**Why**: Forced flags express an exact request, while automatic mode expresses a preference ceiling. Re-descent preserves that distinction and lets a native-capable provider avoid an unnecessary headless launch.
**Rejected**: Silent downgrade of a forced mode, direct pane-to-headless fallback that skips native, and command composition before selection is final.
*Introduced by*: 260808-yilt-dispatch-mode-descent-ladder

### An Unlaunchable Landing Is a Pre-Write Redirect Boundary
**Decision**: When a launch verb's automatic selection lands on a rung it cannot execute, it returns actionable guidance before prompt persistence, stale-file clearing, process launch, or record writes. Two rungs qualify: **native** for every launch verb (redirect to the harness's Agent-tool arm), and **pane** for `start` (redirect to `open`).
**Why**: Native execution belongs to the harness, and a pane launch belongs to the three-step entry; treating each as a redirect keeps the shared ladder honest without creating a fake record or a half-delivered pane. Redirecting before any write is what makes the probe free — the caller re-runs the identical invocation on the right verb with nothing consumed, which is exactly what lets the wiring use `start` as its rung discriminator.
**Rejected**: Persisting a placeholder record, silently jumping past native to headless, attempting to launch the Agent tool from Go, or having `start` quietly perform the `open` half of a pane dispatch.
*Introduced by*: 260808-yilt-dispatch-mode-descent-ladder; *Updated by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

### Real-tmux dispatch tests isolate by a scrubbed environment plus a verified private socket, never by `-L` alone
**Decision**: Every `cmd/fab` and `internal/pane` test that starts a real tmux server runs against a **private socket** under a per-test `TMUX_TMPDIR`, with `$TMUX`/`$TMUX_PANE` **scrubbed by `tmuxSocketDir`** — a documented contract carried by both per-package copies of that helper (`cmd/fab` and `internal/pane`, deliberately not unified) — so opting into private-socket isolation implies the scrub (empirically, tmux treats an empty `$TMUX` as unset, so `t.Setenv("TMUX", "")` suffices). A test that must issue **unscoped** `tmux new-session` / `kill-server` (the default-socket tests, which prove behavior reachable only without `-L`) additionally **verifies the server actually bound the private socket** (`os.Stat`) before registering its cleanup, and scopes every destructive call — `kill-server` included — with an explicit `-S <verified-socket>`. A test that simulates a dispatcher inside a pane sets `$TMUX` itself after the scrub, once its destructive cleanup is socket-scoped.
**Why**: fab's own tmux tests run on whatever server the developer — or an agent worker, which lives in a tmux pane by design — is attached to. `$TMUX` outranks `TMUX_TMPDIR` in tmux's socket resolution (`-L`/`-S` > `$TMUX` > `TMUX_TMPDIR`), so an inherited `$TMUX` silently redirects unscoped tmux calls onto the attached server, and an unscoped `kill-server` cleanup then kills a live server — real data loss from a test cleanup. Scrubbing inside `tmuxSocketDir` removes the hazard for the whole class (a future private-socket test cannot reintroduce it), keeps the suite runnable — not merely safe — from inside panes, and the socket verification remains the checked-precondition backstop: no destructive call is registered until the private socket is proven, and the verified path (not a name or a label) is what every destructive call targets. The same discipline is why the pane tests that *can* be scoped use `-L` with a private, empty `TMUX_TMPDIR` — an unreachable-server assertion then rests on the socket genuinely having no server rather than on the host happening to run none.
**Rejected**: Relying on `-L <label>` alone (a label under the shared default `TMUX_TMPDIR` can collide with a real server). A refuse-if-`$TMUX`-set assertion in each default-socket test (hard-fail up front) — safe where present, but it turns every in-pane suite run into a red test instead of a passing one, and per-test guards apply non-uniformly: the one default-socket test lacking the guard killed a live multi-agent server when the suite ran from a pane (the 260812-kgam incident). Skipping the tmux tests when a server is present (the auto-inside-tmux path would go unasserted on exactly the hosts it ships for — all pane integration tests must RUN, not skip into a false pass). Giving the environment-dependent `interactive_command` test its own ephemeral server (a second `kill-server` cleanup for coverage the tmux-isolated sibling already asserts — the assertions were folded in there instead).
*Introduced by*: 260805-l9ng-auto-pane-dispatch-in-tmux; *Updated by*: 260812-kgam-pane-ready-test-host-kill

### Pane completion keys on the result file, with liveness only separating running from orphaned
**Decision**: A pane dispatch is `done` when `{stage}-result.yaml` exists, `running` when it does not and the pane lives, and `orphaned` when it does not and the pane is gone. Result presence wins over liveness, and `failed` / `failed (no-result)` are simply unreachable rather than remapped or renamed.
**Why**: An interactive worker never exits on task completion — it finishes and sits at its prompt — so no exit-code channel exists to key on, and a liveness-first rule would report `running` forever. The result file is already the contract's success token for the other adapters, so reusing it keeps **one** success definition across all three. Leaving the two exit-code-derived states unreachable keeps the state strings byte-stable, so every existing consumer of `fab dispatch status` reads a pane dispatch without changes; a mode-specific string would have forked the cross-adapter contract.
**Rejected**: Screen-pattern detection via `tmux capture-pane` (scrollback-dependent and ambiguous — the `_cli-agents` § Await guidance explicitly prefers an artifact over a screen pattern). Requiring the worker to exit after writing its result (throws away the steer-after-finish property that motivates the adapter). New pane-only state strings (forks the byte-stable contract).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

### The liveness discriminator is the pane's shell pid, not a start time
**Decision**: `open` records the pane's shell pid (`#{pane_pid}`) as `pane_pid` on the dispatch record at launch, and identity-checked liveness compares it against the pane's current pid at observation time.
**Why**: tmux exposes no `pane_start_time` format variable (verified against the tmux 3.7c man page — only `pane_pid`/`pane_start_command`/`pane_start_path` exist); the shell pid is tmux-native, stable for the pane's lifetime, and already readable through the existing `pane.GetPanePID`. A recorded pid that no longer matches proves the pane the ID now names is not the pane the dispatch opened.
**Rejected**: `pane_start_time` (does not exist in tmux). Recording the spawn timestamp fab-side (proves nothing about which pane the recycled ID currently names).
*Introduced by*: 260822-m461-pane-identity-keying-contract

### Identity checking computes the `paneAlive` input; the state machine is untouched
**Decision**: `DerivePaneState` keeps its signature and derivation table byte-stable; identity-checked liveness replaces the bare `PaneAlive` call inside one shared cmd-layer helper (`paneWorkerAlive`) that every record-keyed consumer — `status`/`wait`, refuse-if-running, `reap`, `kill`, `ready`/`deliver`, `logs` — calls in its place.
**Why**: The three-state derivation is a documented cross-adapter contract with in-code contract comments; changing its signature would ripple through the harness-adapters spec for zero semantic gain. One shared helper keeps the six call sites from drifting on what "the worker is alive" means.
**Rejected**: A fourth parameter on `DerivePaneState` (couples identity into a documented byte-stable machine). Per-call-site inline checks (six sites — a drift liability).
*Introduced by*: 260822-m461-pane-identity-keying-contract

### A failed pid read degrades to existence-only, never to orphaned
**Decision**: At record time (open) and at check time, a pid-read failure downgrades that observation to existence-only liveness — with a stderr warning at record time, and never a failed launch over it.
**Why**: An unreadable pid is not a mismatch — treating it as one would false-orphan live workers on transient tmux errors and burn the one-restart recovery budget. The degraded mode is exactly the legacy-record mode, so it adds no new behavior class.
**Rejected**: Conservative not-alive on read failure (false orphans). Failing the open (a cosmetic-adjacent read must not kill a launch that already succeeded).
*Introduced by*: 260822-m461-pane-identity-keying-contract

### No migration for the `pane_pid` record field
**Decision**: `pane_pid` ships as an additive `omitempty` field with no `src/kit/migrations/` file.
**Why**: `.fab-dispatch/` is transient per-change runtime state (archive-deleted, `clean`-able), not user data being restructured; an absent field keeps existence-only behavior by construction, so records written by older binaries behave byte-identically at every consumer.
**Rejected**: A migration stamping pids into existing records (a stored pid for a pane opened before the migration is unverifiable and could itself alias).
*Introduced by*: 260822-m461-pane-identity-keying-contract

### Prompt file plus a one-line pointer, typed in after spawn and verified
**Decision**: The full stage prompt is persisted to `{stage}-prompt.md` — the path the headless launch already writes — and the pane worker receives a one-line pointer to it, **typed into the pane by `fab dispatch deliver` after the pane is open**, with every step verified against the screen (echo, then submission). The composed `interactive_command` reaches tmux verbatim, carrying no prompt argument, and one delivery engine serves both the initial dispatch and a rework-cycle continuation. Prompt *content* is composed identically for every adapter.
**Why**: A spawn-time positional pointer is **fire-and-forget and unverifiable** — a CLI that silently drops it (observed with agy) leaves a worker at an empty prompt while the dispatch reads `running`, undetectable short of the stage orphaning — and requiring one made pane capability hostage to whether a provider's CLI happens to accept a positional initial prompt, an implicit requirement validated nowhere. Typing the pointer through tmux and *checking* that it landed makes delivery observable, decouples the pane rung from CLI grammar, and gives resume for free: a continuation is the same choreography pointed at a different file, so the one engine is hardened by every dispatch. The prompt file still doubles as a debugging artifact and rides the existing cleanup paths, so `.fab-dispatch/` gains no new file type and no GC change.
**Rejected**: A hybrid preferring the positional one-shot where supported (it needs a per-provider "accepts positional prompt" capability bit — exactly the presence-as-policy coupling the dispatch-mode ladder eliminated — and keeps an unverifiable arm). Sending the whole prompt via send-keys rather than a pointer (send-keys length limits, and a multi-thousand-token prompt cannot ride argv reliably either). Passing the prompt on the interactive command's stdin (an interactive TUI reads stdin as keystrokes, not as a prompt). Composing a shorter prompt for pane mode (would fork the dispatch-prompt obligations that bind all three adapters).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch; *Updated by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

### The echo check normalizes away frame runes by range, not by character class
**Decision**: `countWrapped`'s `squeeze` drops runes in the Unicode box-drawing block (U+2500–U+257F) alongside every whitespace rune, on both the capture and the needle — one predicate covering both shapes a wrap can insert, at both call sites (`ready`'s sentinel echo and `deliver`'s pointer echo).
**Why**: What a tmux hard-wrap inserts between the halves of a pointer is decided by the TUI's own frame, so a whitespace-only normalization silently makes pane capability depend on a provider drawing its input box without side rules — a boxed TUI reads 0 for a pointer plainly on screen and escalates a working delivery. The needle is always a `ReadySentinel` or a prompt-file pointer line, neither of which legitimately contains a frame rune, so dropping the block from both sides cannot mask a genuine mismatch; the range is the narrowest normalization that admits a side-bordered box.
**Rejected**: A broader "ignore all non-alphanumerics" normalization (grows the false-positive surface with no probed need). Verifying submission instead of echo (a larger contract change that gives up the pre-Enter safety check). Per-provider delivery choreography (provider-specific and unautomatable, and it is the workaround the range drop retires).
*Introduced by*: 260810-ki9v-kimi-pane-enablement

### First-run walls are classified mechanically and answered by agent judgment
**Decision**: `fab dispatch ready` classifies a pane by echo and screen stability only — `ready` / `booting` / `parked` — and reports a capture snippet with every non-`ready` answer. It never presses Enter, never answers anything, and carries no table of known dialogs; deciding what a parked screen wants is the orchestrator's judgment, bounded by the wiring's 2-round budget with login walls escalating immediately.
**Why**: Dialog text is a version treadmill and provider-specific, and a half-matched pattern pressing Enter into an unknown screen is worse than stalling. An agent already reads screens for a living, and the snippet gives it everything it needs in the same call. Keeping the binary's half purely mechanical is also what keeps it provider-neutral: nothing in Go knows what a trust prompt looks like.
**Rejected**: A Go pattern table of known dialogs (version treadmill, blind keypresses). Provider-specific trust-store pre-seeding — probed and working for agy, but undocumented-format provider machinery inside a provider-neutral binary. Consulting `@rk_pane_agent_state` (couples dispatch to the operator's agent-state reader for an answer a sentinel already gives).
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

### Two verbs for the pane entry, not one flag
**Decision**: Pane mode enters at `open` (spawn only) with `ready` and `deliver` as separate verbs, rather than `start --no-deliver` plus a delivery flag; `start` is narrowed to headless and refuses a pane landing with `open` guidance, keeping `--pane`/`--server` registered but hidden so the guidance is reachable.
**Why**: The gate between spawn and delivery is an agent-driven loop of unknown length, which a single-shot launch verb has no shape for; naming the three steps makes the sequence executable and inspectable one call at a time, and makes `restart`'s pane arm expressible as "the `open` half, with the rest handed back". Hiding rather than deleting the retired flags is what turns a bare cobra `unknown flag` into an actionable route.
**Rejected**: `start --no-deliver` plus `start --deliver-only` (one verb with two contradictory contracts, and no name for the probe). Leaving `--pane` on `start` as an alias for the whole sequence (Go cannot run the gate's judgment, so the alias would either skip the gate or block on a human). Deleting the flags outright (loses the guidance at the exact moment a caller needs it).
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

### The delivery marker is bookkeeping, never a state
**Decision**: `delivered` is an additive `omitempty` bool on the existing `Dispatch` record, written by `deliver` only after verification succeeds. The five-state machine and the pane three-state derivation are untouched; `status --json` exposes it for pane records only.
**Why**: The distinction it carries — "the worker never got its prompt" vs "the worker got it and failed at the work" — is what makes a failed delivery diagnosable, and it is precisely what a fire-and-forget spawn argument cannot express. But it is not a lifecycle stage: an undelivered pane is genuinely `running` by the same result-presence-plus-liveness rule everything else uses, so promoting it to a state would fork the byte-stable cross-adapter string set for a bookkeeping fact.
**Rejected**: A `delivering`/`undelivered` state string (forks the cross-adapter contract every consumer reads). Inferring delivery from the screen at read time (a capture is not durable, and `status` performs no tmux work beyond a liveness probe). Writing the marker before the choreography (a failed delivery would then read as a mid-stage worker, which every recovery verb refuses).
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

### Stage-aware reap timing is wiring, not a Go guard change
**Decision**: `DecideReap` is untouched; only the moment the orchestrator calls reap moves — apply after review passes, every other stage on done-read.
**Why**: Reap is orchestrator-invoked by design, so *when* is already a wiring concern, and the apply pane has to outlive its own `done` result to be a resume target at all. Touching the guard would put pipeline policy inside the binary and give the command a stage list it has no business knowing.
**Rejected**: A `--keep-for-resume` flag or a stage list in Go (both duplicate a decision the caller already owns). Reaping apply on done-read and re-opening a pane per rework cycle (pays the cold start the pane-arm resume exists to avoid).
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

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

### A pane dispatch takes no worker timeout
**Decision**: `--timeout` exists only where it is enforceable — as POSIX `timeout N` inside the headless `sh -c` wrapper. The pane entry `open` does not register it at all; `restart`, the one verb where a pane flag and `--timeout` are both live, rejects the pair as a usage error before any launch or file write.
**Why**: Pane mode constructs no wrapper to enforce a bound in, and accepting the flag anyway would let an orchestrator believe a bound is enforced when nothing enforces it — precisely the class of silent non-enforcement the `failed (no-result)` state exists to prevent elsewhere. Omitting the flag from a pane-only verb is a stronger guarantee than rejecting a combination, so the rejection is kept only where omission is impossible.
**Rejected**: Accepting and ignoring `--timeout` on the pane path (a false guarantee). Implementing a pane-side timer (re-introduces the supervisor process the dispatch design deliberately has none of).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch; *Updated by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

### `status --json` carries `server` (omitempty); `logs` remains the copy-pasteable capture source
**Decision**: The `--json` surface carries `mode` plus `pane`/`window`/`server`/`delivered` for pane dispatches — `server` sourced from the record, omitted when empty. The complete, socket-aware `fab pane capture [-L <server>] <pane>` command is ALSO printed by `fab dispatch logs` on a pane dispatch.
**Why**: The socket is now first-class in the JSON surface (the additive-field precedent — existing consumers ignore unknown keys), so programmatic consumers assemble a socket-scoped capture from `--json` alone; `logs` keeps printing the exact command because it is where a reader lands at the moment of need, and it cannot be assembled wrong.
**Rejected**: Omitting `server` and pointing readers at `logs` (the original shape — it forced every programmatic consumer through a second command for a value the record already holds). Telling readers to hand-assemble the capture command from `--json` without the socket (they would silently omit `-L` and get an empty capture against the wrong socket).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch; *Updated by*: 260811-yxyi-pane-dispatch-surface-completion
