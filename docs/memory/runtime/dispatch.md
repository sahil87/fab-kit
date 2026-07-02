---
type: memory
description: "`fab dispatch {start,status,logs,kill,clean}` — the tmux-independent headless process manager for CLI-dispatched pipeline stages (3c): the `SysProcAttr{Setsid:true}` detached-launch model on a plain `sh -c` wrapper, the `.fab-dispatch/{id}/` state layout, the five byte-stable status states (incl. `failed (no-result)`), refuse-if-running/last-attempt-only concurrency, timeout-in-wrapper via POSIX `timeout`, the two deterministic cleanup paths (archive-time deletion + `fab dispatch clean`), no automatic GC, and POSIX-only v1 (Windows errors, compile-time platform split); the `internal/dispatch` package + thin `cmd/fab/dispatch*.go` wiring"
---
# fab dispatch

**Domain**: runtime

## Overview

`fab dispatch` is a **tmux-independent, headless process manager** for launching a pipeline stage's resolved spawn command as a detached CLI worker, tracking it via a repo-root state dir, and exposing poll/logs/kill/clean surfaces. It is the headless pipeline path — the parallel, independent complement to the tmux-bound interactive `fab pane` / `fab operator` family (see [pane-commands.md](/runtime/pane-commands.md) and [operator.md](/runtime/operator.md)), which stays exactly as-is. Dispatch is the runtime for **cross-harness stage dispatch** ("a codex orchestrator runs `apply` on claude"): a fundamentally launch-and-poll problem, not a pane-observation one. It is change 3c of the cross-harness dispatch series; it *runs* the per-tier `spawn_command` that 3b's `fab resolve-agent` emits (see [_shared/configuration.md](/_shared/configuration.md) § `agent.tiers`), and its cross-adapter protocol is fixed by the human-curated spec `docs/specs/harness-adapters.md` (authored by this change), against which the 3d skill wiring conforms.

Source: the testable core lives in `internal/dispatch` (state read/write, wrapper composition, five-state derivation, process signaling); thin cobra wiring lives across `cmd/fab/dispatch.go` (parent) + `dispatch_start.go` / `dispatch_status.go` / `dispatch_logs.go` / `dispatch_kill.go` / `dispatch_clean.go` — mirroring the `internal/pane` + `pane*.go` split precedent.

## Requirements

### Requirement: `fab dispatch` command family

The `fab` binary SHALL expose a top-level command group `fab dispatch` with five subcommands — `start`, `status`, `logs`, `kill`, `clean` — always-routed through the `fab` router. Its top-level name MUST NOT collide with the `fab-kit` `LifecycleCommands` allowlist (pinned by `TestNoTopLevelCommandCollidesWithRouterAllowlist`; `dispatch` is not in the allowlist). It is a new fab-go command group registered via `dispatchCmd()` in `cmd/fab/main.go`'s `newRootCmd()`. See [distribution/kit-architecture.md](/distribution/kit-architecture.md) for its place in the fab-go command inventory.

### Requirement: POSIX-only v1

`fab dispatch start` (and `kill`) SHALL error clearly on non-POSIX platforms rather than half-working — the message names the POSIX-shell requirement (`setsid`/`timeout`). The guard is a **compile-time platform split**, not a runtime `runtime.GOOS` string check: `dispatch_posix.go` (build tag `!windows`) owns the launch/signal syscalls; `dispatch_windows.go` (build tag `windows`) provides the same signatures returning the POSIX-only error (with `Alive` conservatively `false`), so the package compiles on Windows and the error surfaces at the command layer. This mirrors the `proc_{linux,darwin}.go` / `pane_process_{linux,darwin}.go` precedent.

#### Scenario: Windows build errors instead of launching

- **GIVEN** a `GOOS=windows` build
- **WHEN** `fab dispatch start` is invoked
- **THEN** it returns an error naming the POSIX-shell requirement and launches nothing

### Requirement: `.fab-dispatch/{id}/` state layout

Each dispatch's state SHALL live under `.fab-dispatch/{4-char-change-id}/` at the **repository root** (`filepath.Dir(fabRoot)`), keyed by the stable 4-char change ID (not the slug, so it survives `fab change rename`). This sits alongside the `.fab-status.yaml` / `.fab-runtime.yaml` ephemeral-state convention, and each git worktree naturally gets its own dir. **No gitignore/scaffold/migration work is required** — the scaffold `fragment-.gitignore` `.fab-*` pattern already matches `.fab-dispatch/`. The dir name is the `internal/dispatch` named constant `DirName = ".fab-dispatch"`; per-stage filenames derive from named suffix constants (no magic strings).

Per-stage files under `.fab-dispatch/{id}/`:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start` (from stdin) | the stage prompt piped to the dispatched command's stdin |
| `{stage}.yaml` | `start` (via `internal/atomicfile`) | the `Dispatch` state struct — `pid`, `pgid`, `spawn_cmd` (resolved), `started_at`, and `timeout` (seconds, `omitempty` when unset). File paths are **derived** from the dir convention, not stored |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its **presence** is the "process finished" signal |
| `{stage}-result.yaml` | the **dispatched agent** (contract) | the stage result; its content is 3d's business — this change defines only the path + consumes its presence for the `done` vs `failed (no-result)` distinction |

### Requirement: `fab dispatch start <change> <stage> [--timeout <secs>]`

`start` SHALL resolve `<change>` to its 4-char ID (via `internal/resolve` — ID / folder substring / full name), read the stage prompt on stdin into `{stage}-prompt.md`, resolve the tier's spawn command via `internal/agent` + `internal/spawn.WithProfile`, launch it **detached** with cwd = the repo root, and persist `{stage}.yaml` before returning.

**Detach mechanism — `SysProcAttr{Setsid:true}` on a plain `sh -c`, NOT the `setsid` binary.** The launch runs the wrapper `sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'` via `exec.Command` with `SysProcAttr{Setsid:true}` — Go's syscall attribute puts the child in a **new session/process group** so the dispatch survives the orchestrator dying, with no Go supervisor process in the loop (the shell records the exit code itself, so resumability falls out: a resumed skill reattaches via `fab dispatch status` instead of re-running the stage). The recorded `pid`/`pgid` therefore track the **live worker shell**. The intake's `setsid sh -c` string described the *intent* (new session, survives orchestrator death); the `SysProcAttr` attribute delivers that intent while keeping the tracked pid on the worker (see Design Decisions — an end-to-end smoke test showed the `setsid` **binary** double-forks, leaving the Go-recorded pid pointing at an immediately-exiting process and breaking liveness/refuse-if-running/kill). `WrapperArgv` is therefore always `[sh -c <script>]` with **no `setsid` prefix**.

**Timeout is enforced entirely inside the wrapper** via POSIX `timeout N <cmd>` when `--timeout N` is given — self-contained, no Go timer, no background sweep, no daemon. A timed-out command exits `124` (POSIX `timeout` convention), which surfaces as `failed` via the normal exit-code path.

#### Scenario: detached launch persists tracked state

- **GIVEN** a change/stage whose resolved tier carries a `spawn_command`
- **WHEN** `fab dispatch start <change> <stage>` runs with a prompt on stdin
- **THEN** the prompt is persisted, the command is launched detached in a new session/process group, and `{stage}.yaml` records the pid/pgid/spawn_cmd/started_at
- **AND** with `--timeout N`, the resolved command is wrapped in POSIX `timeout N` inside the same `sh -c` wrapper

### Requirement: No-spawn_command error (no fallback)

If the resolved tier has no `spawn_command`, `start` SHALL error clearly — naming the stage, the resolved tier, and the `agent.tiers.<tier>.spawn_command` config key — and MUST NOT fall back to the top-level `agent.spawn_command`. This no-cross-fallback rule is the load-bearing dispatch-mode semantic (a fallback would silently flip every project that sets `agent.spawn_command` into CLI dispatch); see [_shared/configuration.md](/_shared/configuration.md) § `agent.tiers`.

### Requirement: Refuse-if-running + last-attempt-only concurrency

`start` SHALL refuse if a dispatch for the exact `(change, stage)` pair is already `running` (reporting the live pid and directing to `fab dispatch kill`), leaving the running dispatch untouched. A `start` over a **completed** prior attempt (done / failed / orphaned) SHALL overwrite its files — there is **no per-attempt history** (last-attempt-only: it removes the stale exit/result/log then re-saves `{stage}.yaml`). Refuse-if-running is scoped per `(change, stage)`: different stages of the same change share `.fab-dispatch/{id}/` via distinct `{stage}.*` filenames and do not collide.

#### Scenario: refuses a live dispatch, overwrites a completed one

- **GIVEN** a `(change, stage)` dispatch whose pid is alive and `{stage}.exit` is absent
- **WHEN** `fab dispatch start` runs again for the same pair
- **THEN** it refuses with a clear error and leaves the running dispatch untouched
- **AND** GIVEN a completed prior attempt, a new `start` overwrites the prior `{stage}.*` files with no history retained

### Requirement: Five-state status machine

`fab dispatch status <change> <stage> [--json]` SHALL read `{stage}.yaml`, `{stage}.exit`, and probe pid liveness, then report exactly one of five **byte-stable** states via the pure `DeriveState`:

| State | Condition | Meaning |
|-------|-----------|---------|
| `running` | pid alive AND `{stage}.exit` absent | still executing |
| `done` | `{stage}.exit` == `0` AND `{stage}-result.yaml` present | finished successfully with a result |
| `failed` | `{stage}.exit` present AND != `0` | non-zero exit (includes `124` timeout) |
| `failed (no-result)` | `{stage}.exit` == `0` BUT `{stage}-result.yaml` absent | **contract violation, NOT done** — exited clean but never wrote its result |
| `orphaned` | pid dead AND `{stage}.exit` absent | reboot / `kill -9` / crash — no exit code was ever recorded |

The `failed (no-result)` state is the crux: a clean exit is necessary but **not sufficient** for `done`; the result file must exist. This distinguishes a well-behaved success from an agent that exited 0 without honoring the result contract (whose result-body schema 3d owns). Liveness reuses the POSIX-standard `syscall.Kill(pid, 0)` EPERM/ESRCH probe.

#### Scenario: clean exit without a result is not done

- **GIVEN** a dispatch that exited `0` with **no** `{stage}-result.yaml`
- **WHEN** `fab dispatch status` runs
- **THEN** it reports `failed (no-result)`, never `done`

### Requirement: `fab dispatch logs <change> <stage> [--tail N]`

`logs` SHALL print `.fab-dispatch/{id}/{stage}.log`; `--tail N` prints the last N lines (implemented in Go via the `Tail` helper, no external `tail`). A missing log SHALL produce a clear "no dispatch log" message rather than erroring opaquely.

### Requirement: `fab dispatch kill <change> <stage>`

`kill` SHALL terminate the **process group** (`pgid` from `{stage}.yaml`, via `syscall.Kill(-pgid, SIGTERM)`) so the detached command and its children die together. It SHALL be **idempotent**: killing an already-dead dispatch (ESRCH) is a benign no-op with a clear report; a missing dispatch gives a clear "no dispatch" error. SIGTERM (graceful), not SIGKILL, matches "die together".

### Requirement: Two cleanup paths, no automatic GC

Cleanup SHALL happen at exactly **two deterministic moments** and never on a timer (throttled/timer sweeps were explicitly rejected — matching fab's no-magic-background-work posture):

1. **Archive-time deletion.** `fab change archive` deletes `.fab-dispatch/{id}/` as part of the archive move — dispatch artifacts are transient comms, not history — so `fab change restore` does **NOT** recreate them. The deletion lives in `internal/archive.Archive()` (best-effort, immediately after the folder move, computing the repo root as `filepath.Dir(fabRoot)`); an absent dir is a no-op and a removal error never undoes the completed move. See [pipeline/change-lifecycle.md](/pipeline/change-lifecycle.md) § archive/restore and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § `/fab-archive`.
2. **Manual `fab dispatch clean [<change>] [--orphans]`.** `clean <change>` removes that change's dir; `clean` (no arg) removes all `.fab-dispatch/*/` dirs; `clean --orphans` prunes any `.fab-dispatch/{id}/` whose ID no longer resolves to a **non-archived** change (via `resolve.ToFolder`, which excludes `archive/`), covering the case where a change was archived upstream and a local `git pull` left the state dir orphaned.

#### Scenario: `--orphans` prunes only unresolvable IDs

- **GIVEN** several `.fab-dispatch/*/` dirs, one whose ID no longer resolves to an active change
- **WHEN** `fab dispatch clean --orphans` runs
- **THEN** only the orphaned dir is pruned; live dirs are left intact

## Design Decisions

### Setsid syscall attribute, not the `setsid` binary
**Decision**: Detach via Go's `exec.Command(...).SysProcAttr = &syscall.SysProcAttr{Setsid: true}` on a plain `sh -c '...'` wrapper — a single detach mechanism — rather than prefixing the `setsid` binary as the intake's `setsid sh -c` string literally suggested.
**Why**: An end-to-end smoke test showed the `setsid` **binary** double-forks (its caller is already a process-group leader under Setsid), so the Go-recorded pid pointed at an immediately-exiting `setsid` process — breaking liveness, refuse-if-running, and kill. One trackable detach mechanism is the correctness fix; the observable behavior (detached, survives orchestrator death, resumable) matches the intake exactly.
**Rejected**: The literal `setsid` binary prefix (untrackable pid); a long-lived Go supervisor process that waits on the child (re-introduces a process that must itself survive the orchestrator — defeats the point; the shell wrapper's `echo $? > exit` makes the shell the supervisor with no Go process in the loop).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### Parallel family, not a headless mode on `fab pane`
**Decision**: `fab dispatch` is a new command family independent of `fab pane` / `fab operator`; the interactive pane machinery is untouched.
**Why**: Pane observation (tmux capture) and headless dispatch (file polling) are fundamentally different models; conflating them would burden the interactive path with headless concerns. `fab pane`/`fab operator` remain the interactive-operator-visibility surface; `fab dispatch` is the headless pipeline path.
**Rejected**: Extending `fab pane` with a headless mode (model conflation). Automatic GC of state dirs on a timer (rejected by the user — cleanup is exactly two deterministic moments).
*Introduced by*: 260702-6sgj-fab-dispatch-command

### `internal/dispatch` package with thin cmd wiring
**Decision**: Extract state read/write, wrapper composition (`WrapperArgv`), five-state derivation (`DeriveState`), and process signaling into `internal/dispatch`, with thin `cmd/fab/dispatch*.go` cobra wiring.
**Why**: The status-state machine and wrapper composition are the testable core; the `internal/pane` / `internal/archive` precedent and the need to table-test the pure state machine independent of a launched process make extraction the clear default.
**Rejected**: Inline in `cmd/fab` (harder to table-test the state derivation without launching a real process).
*Introduced by*: 260702-6sgj-fab-dispatch-command
