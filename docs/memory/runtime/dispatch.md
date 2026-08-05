---
type: memory
description: "`fab dispatch {start,status,logs,kill,clean}` — process manager for CLI-dispatched pipeline stages in two modes: headless (default, tmux-independent, detached `Setsid` `sh -c` wrapper on `dispatch_command`, all five byte-stable states incl. `failed (no-result)`) and `--pane` (interactive tmux-window worker from `session_command`, prompt-file + pointer delivery, result-file-only detection, three-state subset). Shared state layout, derived-mode record, two cleanup paths, no cross-fallback."
---
# fab dispatch

**Domain**: runtime

## Overview

`fab dispatch` is the **process manager for CLI-dispatched pipeline stages**, in **two launch modes** that share one state directory, one loader, one concurrency check, and one state-string vocabulary:

| Mode | Flag | Worker | Command composed | Completion observed via | tmux |
|------|------|--------|------------------|-------------------------|------|
| **headless** (default) | — | a detached `sh -c` process | the provider's `dispatch_command` | `{stage}.exit` + pid liveness + result file | never touched |
| **pane** | `--pane` | an interactive agent session in a tmux window the user can watch and steer | the provider's `session_command` | **result file** + pane liveness | **required** (hard error without) |

Headless mode is the **tmux-independent default** for unattended pipelines: it launches the resolved command (persisted in the record as `spawn_cmd`) detached, tracks it via a repo-root state dir, and exposes poll/logs/kill/clean surfaces. Pane mode is **opt-in per invocation** and exists to recover *watch and steer* on the CLI path — a detached headless worker is a black box no one can talk to, whereas an interactive tmux worker is the one universal interface every agent CLI supports natively. Together they are the **two non-native adapters** of the three-adapter cross-harness dispatch catalog (native Agent-tool / headless CLI / interactive pane), whose protocol is fixed by the human-curated spec `docs/specs/harness-adapters.md`.

Dispatch is the runtime for **cross-harness stage dispatch** ("a codex orchestrator runs `apply` on claude"): a fundamentally launch-and-poll problem, not a pane-observation one. It *runs* the provider command that `fab resolve-agent` names (a tier names a provider; the provider carries the commands — see [_shared/configuration.md](/_shared/configuration.md) § `providers` and [runtime/providers-and-tiers.md](/runtime/providers-and-tiers.md)). Headless dispatch stays parallel to and independent of the tmux-bound interactive `fab pane` / `fab operator` family (see [pane-commands.md](/runtime/pane-commands.md) and [operator.md](/runtime/operator.md)); pane dispatch **borrows tmux as a launch surface** but does not join the operator's monitored set, and consumes the `_cli-agents` helper's spawn/deliver/peek procedures as its primitive layer (see [agent-primitives.md](/runtime/agent-primitives.md)).

The **skill wiring** consumes it: the dispatch-seam skills branch on the resolved `dispatch=` line and, when present, drive this command family — `fab dispatch start` (block prompt on stdin, `--pane` added only on an explicit user directive for a watchable stage) → `sleep 30` polling of `fab dispatch status` → the mode's reachable states → read `{stage}-result.yaml` on `done`. The wiring lives in `_preamble.md` § CLI-Adapter Dispatch + § Dispatch-Prompt Obligations, where **pane mode is an option inside the `dispatch=`-present arm, never a third branch** (see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership and [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution).

Source: the testable core lives in `internal/dispatch` (state read/write, wrapper composition, both state derivations, process signaling, and the tmux pane primitives in `pane_mode.go`); thin cobra wiring lives across `cmd/fab/dispatch.go` (parent) + `dispatch_start.go` / `dispatch_status.go` / `dispatch_logs.go` / `dispatch_kill.go` / `dispatch_clean.go` — mirroring the `internal/pane` + `pane*.go` split precedent.

## Requirements

### Requirement: `fab dispatch` command family

The `fab` binary SHALL expose a top-level command group `fab dispatch` with five subcommands — `start`, `status`, `logs`, `kill`, `clean` — always-routed through the `fab` router. Its top-level name MUST NOT collide with the `fab-kit` `LifecycleCommands` allowlist (pinned by `TestNoTopLevelCommandCollidesWithRouterAllowlist`; `dispatch` is not in the allowlist). It is a new fab-go command group registered via `dispatchCmd()` in `cmd/fab/main.go`'s `newRootCmd()`. See [distribution/kit-architecture.md](/distribution/kit-architecture.md) for its place in the fab-go command inventory.

### Requirement: POSIX-only v1 (the headless launch/signal syscalls)

The **headless** `fab dispatch start` (and `kill`) SHALL error clearly on non-POSIX platforms rather than half-working — the message names the POSIX-shell requirement (`setsid`/`timeout`). The guard is a **compile-time platform split**, not a runtime `runtime.GOOS` string check: `dispatch_posix.go` (build tag `!windows`) owns the launch/signal syscalls; `dispatch_windows.go` (build tag `windows`) provides the same signatures returning the POSIX-only error (with `Alive` conservatively `false`), so the package compiles on Windows and the error surfaces at the command layer. This mirrors the `proc_{linux,darwin}.go` / `pane_process_{linux,darwin}.go` precedent.

Pane mode's tmux primitives (`ServerReachable`, `OpenWindow`, `PaneAlive`, `KillPane`) live in the platform-**independent** core (`internal/dispatch/pane_mode.go`) for the same reason `WrapperArgv` does: they are plain tmux subprocess calls with no syscall dependency, so they compile everywhere. Pane mode is still unusable where tmux is absent, but that surfaces as `ServerReachable`'s actionable error, not as a compile-time platform split.

#### Scenario: Windows build errors instead of launching

- **GIVEN** a `GOOS=windows` build
- **WHEN** headless `fab dispatch start` is invoked
- **THEN** it returns an error naming the POSIX-shell requirement and launches nothing

### Requirement: `.fab-dispatch/{id}/` state layout

Each dispatch's state SHALL live under `.fab-dispatch/{4-char-change-id}/` at the **repository root** (`filepath.Dir(fabRoot)`), keyed by the stable 4-char change ID (not the slug, so it survives `fab change rename`). This sits alongside the `.fab-status.yaml` repo-root ephemeral-state convention (the `.fab-runtime.yaml` sibling that convention once also named is gone (ioku) — agent-state production is divested — see [runtime-agents.md](/runtime/runtime-agents.md)), and each git worktree naturally gets its own dir. **No gitignore/scaffold/migration work is required** — the scaffold `fragment-.gitignore` `.fab-*` pattern already matches `.fab-dispatch/`. The dir name is the `internal/dispatch` named constant `DirName = ".fab-dispatch"`; per-stage filenames derive from named suffix constants (no magic strings). **Both modes share the dir**, the loader, the save path, and the refuse-if-running check.

Per-stage files under `.fab-dispatch/{id}/`:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start` (from stdin) | the stage prompt — piped to the dispatched command's stdin (headless) or **pointed at** by the one-line prompt the pane worker receives (pane). Written identically in both modes; only the hand-over differs |
| `{stage}.yaml` | `start` (via `internal/atomicfile`) | the `Dispatch` state struct — `spawn_cmd` (resolved) + `started_at`, plus the mode's identity: `pid`/`pgid`/`timeout` (headless) or `pane`/`window`/`server` (pane). Every mode-specific key is `omitempty`, and the mode is **derived** from which keys are present (no stored discriminator). File paths are **derived** from the dir convention, not stored |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command — **headless only** (a pane worker's output is tmux scrollback) |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its **presence** is the "process finished" signal; **headless only** |
| `{stage}-result.yaml` | the **dispatched agent** (contract) | the stage result; dispatch defines only the path + consumes its presence. Presence is required for `done` in both modes and is the **sole** completion signal in pane mode. Its **content schema** is a minimal YAML envelope mirroring each native block's return — common `stage`/`status`/`summary`; apply adds `failed_task`/`reason` on failure; review adds `verdict` (pass\|fail) + `findings{must_fix,should_fix,nice_to_have}`; hydrate carries only the envelope. The **`status` (worker/infra outcome) vs `verdict` (review outcome) split is load-bearing** — a completed review with `verdict: fail` is dispatch-state `done` (result present), and the orchestrator then takes the normal review-fail path; dispatch-state `failed` is reserved for worker/infrastructure failure. Schema documented in `_preamble.md` § Dispatch-Prompt Obligations. |

#### Scenario: a headless record's on-disk shape is unchanged by pane mode's fields

- **GIVEN** a headless dispatch
- **WHEN** `{stage}.yaml` is serialized
- **THEN** it carries `pid`/`pgid`/`spawn_cmd`/`started_at` and **none** of `pane`/`window`/`server`
- **AND** GIVEN a pane dispatch, the record carries `pane`/`window`/(`server` when set) and no `pid`/`pgid` — the addition is additive on disk, so no migration exists or is needed

### Requirement: `fab dispatch start <change> <stage> [--timeout <secs>] [--pane] [--server <name>]`

`start` SHALL run a **shared prologue** followed by one of **two mode-specific launch tails**. The prologue resolves `<change>` to its 4-char ID (via `internal/resolve` — ID / folder substring / full name), loads config, resolves the stage's tier → provider profile via `internal/agent` + `internal/spawn.WithProfile` (the same `{model}`/`{effort}` substitution `fab resolve-agent` performs), enforces refuse-if-running, persists the stage prompt read from stdin into `{stage}-prompt.md`, and clears stale per-stage files. The tail launches the worker and persists `{stage}.yaml` before returning. The prologue SHALL NOT be duplicated across the tails.

Output names the mode's identity: `dispatched <id>/<stage> (pid N, pgid N)` (headless) or `dispatched <id>/<stage> (pane %N, window fab-<id>-<stage>)` (pane).

**Headless tail — detach mechanism — `SysProcAttr{Setsid:true}` on a plain `sh -c`, NOT the `setsid` binary.** The launch runs the wrapper `sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'` via `exec.Command` with `SysProcAttr{Setsid:true}` — Go's syscall attribute puts the child in a **new session/process group** so the dispatch survives the orchestrator dying, with no Go supervisor process in the loop (the shell records the exit code itself, so resumability falls out: a resumed skill reattaches via `fab dispatch status` instead of re-running the stage). The recorded `pid`/`pgid` therefore track the **live worker shell**. The intake's `setsid sh -c` string described the *intent* (new session, survives orchestrator death); the `SysProcAttr` attribute delivers that intent while keeping the tracked pid on the worker (see Design Decisions — an end-to-end smoke test showed the `setsid` **binary** double-forks, leaving the Go-recorded pid pointing at an immediately-exiting process and breaking liveness/refuse-if-running/kill). `WrapperArgv` is therefore always `[sh -c <script>]` with **no `setsid` prefix**.

**Timeout is enforced entirely inside the wrapper** via POSIX `timeout N <cmd>` when `--timeout N` is given — self-contained, no Go timer, no background sweep, no daemon. A timed-out command exits `124` (POSIX `timeout` convention), which surfaces as `failed` via the normal exit-code path.

#### Scenario: detached launch persists tracked state

- **GIVEN** a change/stage whose resolved tier's provider carries a `dispatch_command`
- **WHEN** `fab dispatch start <change> <stage>` runs with a prompt on stdin
- **THEN** the prompt is persisted, the command is launched detached in a new session/process group, and `{stage}.yaml` records the pid/pgid/spawn_cmd/started_at
- **AND** with `--timeout N`, the resolved command is wrapped in POSIX `timeout N` inside the same `sh -c` wrapper

**`--pane` tail — an interactive tmux window.** With `--pane`, `start` SHALL compose the resolved provider's **`session_command`** (the same string `fab agent` composes) and open it as `tmux new-window -n fab-{id}-{stage} -c <repo-root> "<resolved-cmd> <shell-quoted-pointer>"`, persisting the new window's **pane ID**, window name, and tmux socket label in `{stage}.yaml`. The composed command is passed as `new-window`'s shell-command argument, so shell expansions it carries (e.g. `$(basename "$(pwd)")` in the built-in claude `session_command`) expand at invocation inside the new window — the `_cli-agents.md` § Spawn Composition contract. The **pane ID**, not the window name, is the recorded identity: it is server-global, stable for the pane's lifetime, and exempt from tmux's target-grammar prefix/glob resolution, so liveness probes and kills are exact where a name-based target could resolve to a window the user renamed into place. `new-window -P -F '#{pane_id}'` prints it, avoiding a follow-up lookup that could race a fast-exiting worker.

`--server <name>` / `-L <name>` targets a tmux socket (`tmux -L <name>`), mirroring the `fab pane` family's persistent flag, and is persisted so `status`/`kill` reach the same server without re-supplying it. It is **ignored without `--pane`** (headless touches no tmux).

#### Scenario: pane launch persists pane identity

- **GIVEN** a change/stage whose resolved tier's provider carries a `session_command`, and a reachable tmux server
- **WHEN** `fab dispatch start <change> <stage> --pane` runs with a prompt on stdin
- **THEN** a tmux window runs the composed interactive command, `{stage}.yaml` records `pane`/`window`/(`server`)/`spawn_cmd`/`started_at`, and no `{stage}.exit` wrapper is involved

### Requirement: Prompt delivery is a file plus a one-line pointer in pane mode

In **both** modes the full stage prompt arrives on **stdin** and is persisted to `{stage}-prompt.md`. Headless mode pipes that file into the dispatched command's stdin; pane mode SHALL instead hand the worker a **one-line pointer** naming the repo-relative prompt path, embedded at spawn as the interactive command's single prompt argument. The prompt **content** is composed identically for every adapter — nothing about the block prompt is written differently for `--pane`; only the hand-over differs. No `send-keys` delivery and no printed-prompt probe is required for the initial delivery.

**The pointer SHALL be shell-quoted; the resolved command SHALL stay verbatim.** The two halves of the window command are quoted differently on purpose. The pointer names a *repo-derived* path, so a checkout under a directory containing a single quote (`/home/me/sahil's-repo/…`) would terminate a naively-single-quoted argument early — breaking the `new-window` command and handing the path's remainder to the window's shell. It therefore rides through the package's `shellQuote` (the `'\''` idiom the headless wrapper's paths already use), composed in one place by `dispatch.WindowCommand`, honoring `_cli-agents.md` § Spawn Composition's "shell-escape any user-supplied text before embedding it". The resolved `session_command` is inserted **verbatim** — its shell expansions are deliberate and must expand inside the new window (per the pass-through philosophy: the command's own quoting is the resolver's/user's concern).

#### Scenario: a multi-thousand-token prompt reaches an interactive worker

- **GIVEN** a full stage prompt on stdin
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** the full prompt lands in `{stage}-prompt.md` and the window's command carries only the one-line pointer to that path, readable from the window's cwd (the repo root)

#### Scenario: a repo path containing a single quote does not break the window command

- **GIVEN** a repository whose path contains a `'` character, so the repo-relative pointer inherits it
- **WHEN** `fab dispatch start <change> <stage> --pane` composes the window command
- **THEN** the pointer is shell-escaped and parses as exactly one shell word, arriving at the worker byte-identical to the composed pointer

### Requirement: `--pane` requires a reachable tmux server — hard error, nothing written

Pane mode SHALL require a reachable tmux server, failing with a non-zero exit and an actionable stderr message when none is reachable — launching nothing and persisting **no** dispatch record. Reachability SHALL be established by a **real tmux query** (`tmux [-L <server>] list-sessions` via `ServerReachable`), **not** an `$TMUX` environment read: a dispatching orchestrator may itself be headless — exactly the cross-harness caller pane mode exists for — so an `$TMUX`-only gate would make `--pane` unusable from those callers. This mirrors `fab resolve --pane`, where `--server` likewise replaces the `$TMUX` guard with socket-scoped discovery. Without `--pane`, **no tmux probe occurs at all**, preserving the headless path's tmux-independence guarantee.

The probe distinguishes "a server answered" from "nothing answered", not "has sessions" from "has none": under tmux's default `exit-empty on` a server exits with its last session, so a zero-session server never persists to be probed and `--pane` correctly errors; under `exit-empty off` a sessionless server does persist and the probe passes, and a subsequent `new-window` that tmux cannot satisfy surfaces the child's own stderr via `internal/pane`'s `StderrError`. The probe is a reachability gate, not a launch guarantee — the launch carries its own actionable error.

#### Scenario: unreachable tmux leaves no trace

- **GIVEN** no reachable tmux server (or an unreachable `--server` socket)
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** it exits non-zero naming tmux reachability, the `--server` option, and the headless alternative, and creates no `{stage}.yaml`

### Requirement: `--pane` and `--timeout` are mutually exclusive

Supplying both flags SHALL be a usage error (non-zero exit) naming the exclusion, enforced before any launch or file write — never a silently ignored `--timeout`. `--timeout` is implemented as POSIX `timeout N` inside the headless `sh -c` wrapper, which pane mode never constructs.

#### Scenario: the exclusion is enforced before anything happens

- **GIVEN** `fab dispatch start <change> <stage> --pane --timeout 600`
- **WHEN** it runs
- **THEN** it exits non-zero naming the `--pane`/`--timeout` exclusion, launching nothing and writing nothing

### Requirement: Missing command field → error, no cross-fallback in either direction

If the resolved tier's provider lacks the field the mode needs, `start` SHALL error clearly — naming the stage, the resolved tier, the provider, and the config key (`providers.<name>.dispatch_command` for headless, `providers.<name>.session_command` for `--pane`) — and MUST NOT fall back to the other field. The **no-cross-fallback rule holds in both directions**: headless never falls back to `session_command`, and pane mode never falls back to `dispatch_command`. This is the load-bearing dispatch-mode semantic (a fallback would silently flip a session-command-only provider into headless CLI dispatch, or the reverse); see [_shared/configuration.md](/_shared/configuration.md) § `providers` and [runtime/providers-and-tiers.md](/runtime/providers-and-tiers.md). Pane mode reading the provider table for `session_command` is **not** a fallback and **not** a resolver change: mode selection is per-invocation, `fab resolve-agent`'s output is identical either way, and pane mode reads the table itself exactly as `fab agent` does.

#### Scenario: a session-command-only provider dispatches under `--pane` and errors without it

- **GIVEN** a tier whose provider carries a `session_command` but no `dispatch_command`
- **WHEN** `fab dispatch start <change> <stage> --pane` runs with a reachable tmux server
- **THEN** the dispatch succeeds using the composed `session_command`
- **AND** GIVEN the same tier, a `--pane`-less `start` errors with the `dispatch_command` config-key hint
- **AND** GIVEN a provider with no `session_command`, `--pane` errors with the `providers.<name>.session_command` hint

### Requirement: Refuse-if-running + last-attempt-only concurrency

`start` SHALL refuse if a dispatch for the exact `(change, stage)` pair is already `running` — reporting the live identity (`pid N` or `pane %N`) and directing to `fab dispatch kill` — leaving the running dispatch untouched. The check SHALL apply the **prior record's own mode's finished signal** — the *same* signal `status` derives that mode's state from, so `start` and `status` can never disagree about whether an attempt is still going:

| Prior record's mode | Still running when | Finished when |
|---|---|---|
| headless | `{stage}.exit` absent **and** pid alive | `{stage}.exit` present (the shell recorded a code) |
| pane | `{stage}-result.yaml` absent **and** pane alive | `{stage}-result.yaml` present — **result presence wins over pane liveness** |

The pane row's result-presence precedence mirrors `DerivePaneState` and is load-bearing: an interactive worker never exits on task completion, it sits at its prompt, so a liveness-only refusal would fire forever after a successful pane run and make a `done` attempt permanently un-overwritable — `status` reporting `done` while `start` insisted it was still running.

A `start` over a **completed** prior attempt (done / failed / orphaned) SHALL overwrite its files — there is **no per-attempt history** (last-attempt-only: it removes the stale exit/result/log then re-saves `{stage}.yaml`), and the new attempt MAY use either mode regardless of the prior one's. Refuse-if-running is scoped per `(change, stage)`: different stages of the same change share `.fab-dispatch/{id}/` via distinct `{stage}.*` filenames and do not collide.

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

#### Scenario: killing a pane leaves the dispatch orphaned

- **GIVEN** a live pane dispatch with no result file
- **WHEN** `fab dispatch kill <change> <stage>` runs
- **THEN** the pane is killed, a killed report is printed, and a subsequent `status` reports `orphaned`
- **AND** GIVEN the pane is already gone, `kill` exits 0 with an already-dead report

### Requirement: A dispatch window is named `fab-{id}-{stage}` and carries no operator marker

A pane dispatch's tmux window SHALL be named from its own convention — `fab-{4-char-change-id}-{stage}`, composed by `WindowName` — and MUST NOT carry the operator's `»` (U+00BB) enrollment prefix or its `›` (U+203A) done marker. Those markers assert that the window is in the operator's monitored set and that the operator owns its lifecycle, neither of which a pipeline dispatch has. An operator that genuinely enrolls the window adds the marker itself through its own idempotent `fab pane window-name ensure-prefix` primitive (see [operator.md](/runtime/operator.md) § monitored-set enrollment and [pane-commands.md](/runtime/pane-commands.md)).

#### Scenario: the window is identifiable without claiming operator ownership

- **GIVEN** `fab dispatch start abcd apply --pane`
- **WHEN** the window is created
- **THEN** its name is `fab-abcd-apply` and carries no `»`/`›` prefix

### Requirement: Steering a pane worker is contract-neutral

A user MAY converse with a running pane worker mid-stage. This changes **no** contract: the worker still owes `{stage}-result.yaml`, still ends with the terminal `fab status refresh` epilogue, and still runs no `fab status` **transition** command — **the orchestrator owns every transition** (see [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership). Steering is human input into the worker's context, exactly like answering a native sub-agent's question. A worker steered away from producing a result needs no new state: it never reaches `done` and surfaces through the never-`done` escalation the orchestrator already owns. This is **documentation only** — no code detects, gates, or reports a steered worker.

#### Scenario: a steered worker owes the same artifacts

- **GIVEN** a user typing into a running pane worker's window
- **WHEN** the worker finishes
- **THEN** the same result-file and refresh-epilogue obligations apply and the orchestrator's transition ownership is unchanged
- **AND** GIVEN the worker never produces a result, the dispatch stays `running`/`orphaned` and the orchestrator escalates by its existing never-`done` path

### Requirement: Two cleanup paths, no automatic GC

Cleanup SHALL happen at exactly **two deterministic moments** and never on a timer (throttled/timer sweeps were explicitly rejected — matching fab's no-magic-background-work posture):

1. **Archive-time deletion.** `fab change archive` deletes `.fab-dispatch/{id}/` as part of the archive move — dispatch artifacts are transient comms, not history — so `fab change restore` does **NOT** recreate them. The deletion lives in `internal/archive.Archive()` (best-effort, immediately after the folder move, computing the repo root as `filepath.Dir(fabRoot)`); an absent dir is a no-op and a removal error never undoes the completed move. See [pipeline/change-lifecycle.md](/pipeline/change-lifecycle.md) § archive/restore and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § `/fab-archive`.
2. **Manual `fab dispatch clean [<change>] [--orphans]`.** `clean <change>` removes that change's dir; `clean` (no arg) removes all `.fab-dispatch/*/` dirs; `clean --orphans` prunes any `.fab-dispatch/{id}/` whose ID does not resolve to a **non-archived** change (via `resolve.ToFolder`, which excludes `archive/`), covering the case where a change was archived upstream and a local `git pull` left the state dir orphaned.

`clean` is **mode-blind**: it removes state dirs and never inspects a record's mode, so a pane dispatch's dir (prompt file included) is cleaned exactly like a headless one's. As with a live headless process, cleaning a **live** pane dispatch removes the state without killing the worker — `kill` is the verb for that.

#### Scenario: `--orphans` prunes only unresolvable IDs

- **GIVEN** several `.fab-dispatch/*/` dirs, one whose ID does not resolve to an active change
- **WHEN** `fab dispatch clean --orphans` runs
- **THEN** only the orphaned dir is pruned; live dirs are left intact

## Design Decisions

### Setsid syscall attribute, not the `setsid` binary
**Decision**: Detach via Go's `exec.Command(...).SysProcAttr = &syscall.SysProcAttr{Setsid: true}` on a plain `sh -c '...'` wrapper — a single detach mechanism — rather than prefixing the `setsid` binary as the intake's `setsid sh -c` string literally suggested.
**Why**: An end-to-end smoke test showed the `setsid` **binary** double-forks (its caller is already a process-group leader under Setsid), so the Go-recorded pid pointed at an immediately-exiting `setsid` process — breaking liveness, refuse-if-running, and kill. One trackable detach mechanism is the correctness fix; the observable behavior (detached, survives orchestrator death, resumable) matches the intake exactly.
**Rejected**: The literal `setsid` binary prefix (untrackable pid); a long-lived Go supervisor process that waits on the child (re-introduces a process that must itself survive the orchestrator — defeats the point; the shell wrapper's `echo $? > exit` makes the shell the supervisor with no Go process in the loop).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### Parallel family, not a headless mode on `fab pane`
**Decision**: `fab dispatch` is a command family independent of `fab pane` / `fab operator`; the `fab pane` command surface and the operator's monitored-set machinery carry no dispatch concerns. Pane-mode dispatch consumes `internal/pane`'s tmux helpers (`RunCmd`/`WithServer`/`StderrError`) as a library and borrows tmux as a launch surface, without joining the operator's monitored set or adding a headless mode to `fab pane`.
**Why**: Pane *observation* (tmux capture, operator ownership) and *stage dispatch* (a state dir, a result-file contract, a poll loop) are different models with different owners; conflating the command surfaces would burden the interactive-operator path with pipeline concerns. Sharing the tmux argv builder at the library level gets the reuse without the conflation — one tmux invocation convention in the binary, two independent command families.
**Rejected**: Extending `fab pane` with a headless mode (model conflation — the inverse of the one `fab dispatch` was created to avoid). Re-implementing tmux invocation inside `internal/dispatch` (a second argv builder and a second stderr-enrichment convention). Automatic GC of state dirs on a timer (cleanup is exactly two deterministic moments).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### `internal/dispatch` package with thin cmd wiring
**Decision**: Extract state read/write, wrapper composition (`WrapperArgv`), both state derivations (`DeriveState`, `DerivePaneState`), process signaling, and the tmux pane primitives (`pane_mode.go`) into `internal/dispatch`, with thin `cmd/fab/dispatch*.go` cobra wiring.
**Why**: The status-state machines and wrapper composition are the testable core; the `internal/pane` / `internal/archive` precedent and the need to table-test the pure state machines independent of a launched process or a live tmux server make extraction the clear default. The tmux primitives sit in the platform-independent core (not the build-tagged split) because they are plain subprocess calls with no syscall dependency.
**Rejected**: Inline in `cmd/fab` (harder to table-test the state derivations without launching a real process).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### Pane mode is a flag on `fab dispatch start`, not a new command family
**Decision**: Interactive dispatch is `fab dispatch start <change> <stage> --pane`, sharing the tier resolution, state directory, refuse-if-running concurrency, and `status`/`kill`/`logs`/`clean` surfaces with the headless path. Mode selection is **per-invocation** — a flag a skill passes on an explicit user directive — with **no provider config field**: pane mode composes the resolved provider's existing `session_command`, so `fab resolve-agent`'s output is identical either way.
**Why**: The sequencer wiring already branches at `fab dispatch`, and the state-string vocabulary plus the `.fab-dispatch/{id}/` layout are exactly the machinery a pane dispatch needs; a parallel command family would duplicate all of it and force every dispatch site to learn a second grammar. Keeping the interactive invocation derivable from `session_command` — the same string `fab agent` composes — means no schema change and no new default behavior; a provider whose interactive grammar genuinely diverges from its session grammar is the trigger to add a field later, as a data-only config addition.
**Rejected**: A new `fab pane-dispatch` command family (duplicates the state machine, the loader, and the concurrency check). A third per-provider command field (`pane_command`) in v1 (schema churn for a string already in the config). Declaring the mode in config rather than per invocation (watchability is a property of *this run*, not of a provider).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

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

### Pane windows get a `fab-{id}-{stage}` name, not the operator's `»` marker
**Decision**: A dispatch window is named `fab-{id}-{stage}` and carries no `»`/`›` prefix.
**Why**: The `»` prefix is the operator's enrollment marker — it asserts the window is in the operator's monitored set and that the operator owns its lifecycle. A pipeline dispatch has neither property, so pre-marking would make the operator's tab bar lie about what it tracks. A distinct, greppable name convention gives the same at-a-glance identification without the false claim, and an operator that genuinely enrolls the window still adds the marker through its own idempotent primitive.
**Rejected**: Prefixing `»` at creation (falsely signals operator ownership). Leaving the window unnamed (indistinguishable from an ad-hoc shell tab).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

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
