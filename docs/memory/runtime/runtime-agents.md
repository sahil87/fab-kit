---
type: memory
description: "`.fab-runtime.yaml` schema — `_agents[session_id]` keying, hook write/clear pipeline (stop/session-start/user-prompt) with one flock-serialized `UpdateAgent` call per event (GC folded in, skip-save-when-unchanged, no fsync), throttled GC via `last_run_gc`, grandparent PID walker, pane-map matching rule"
---
# Runtime Agents

**Domain**: runtime

## Overview

`.fab-runtime.yaml` is the ephemeral per-worktree state file that tracks Claude Code agents across tmux panes and worktrees. It lives at the repo root of each git worktree (one file per worktree, gitignored) and is written exclusively by `fab hook` subcommands invoked by Claude Code events (Stop, SessionStart, UserPromptSubmit). It is read by `fab pane map` (to populate the Agent column) and by `fab pane send` (to validate agent idleness before sending keystrokes).

Agents are first-class entries keyed by Claude's `session_id` (a UUID from the hook stdin JSON payload). Change folder name, PID, tmux server, tmux pane, and transcript path are all optional properties on the agent entry — they populate when available and are omitted otherwise. This cleanly separates the three orthogonal axes that `fab pane map` tracks: **Change** (from `.fab-status.yaml`), **Agent** (from `_agents`), and **Process** (opt-in via `fab pane process`).

Agents running in discussion mode (before `/fab-new`, no active change) are tracked the same way as change-associated agents — visibility in `fab pane map` no longer depends on whether a change is active.

## Requirements

### Schema

`.fab-runtime.yaml` contains a single top-level `_agents` map keyed by Claude `session_id` (UUID string), plus a top-level `last_run_gc` timestamp that throttles the GC sweep.

```yaml
_agents:
  "d630bcf0-8820-4dd1-a99c-9bda5ea72c88":       # key: Claude session_id (UUID from hook stdin)
    idle_since: 1729450100                       # unix ts — present when agent is idle
    change: "260417-2fbb-pane-server-flag"       # optional — absent/empty in discussion mode
    pid: 2356168                                 # optional — Claude's PID (for GC liveness)
    tmux_server: "fabKit"                        # optional — basename of $TMUX socket
    tmux_pane: "%15"                             # optional — from $TMUX_PANE (includes %)
    transcript_path: "/home/.../d630bcf0-...jsonl"  # optional — from hook payload
last_run_gc: 1729450200                          # top-level — throttles GC sweeps to every ~3 min
```

**Required fields**: Entry key (`session_id`). All other fields are optional.

**Field semantics**:

| Field | Type | Presence | Meaning |
|-------|------|----------|---------|
| `idle_since` | int (unix ts) | present → idle; absent → active | Set by the Stop hook; removed by the UserPromptSubmit hook |
| `change` | string | optional — empty/absent in discussion mode | Change folder name the agent is working on |
| `pid` | int | optional | Claude's PID (grandparent-walk resolved); used for GC liveness |
| `tmux_server` | string | optional — absent outside tmux | Basename of `$TMUX` socket path |
| `tmux_pane` | string | optional — absent outside tmux | Pane ID from `$TMUX_PANE` (e.g., `%15`) |
| `transcript_path` | string | optional | Absolute path to Claude's transcript (`*.jsonl`) |
| `last_run_gc` | int (unix ts) | top-level, not nested | Most recent GC sweep completion time |

**Missing file semantics**: Read paths treat a missing `.fab-runtime.yaml` as `_agents: {}` — no error, no warning. The file is created on first hook write.

**Schema location invariants**: `.fab-runtime.yaml` lives at the repo root of each git worktree. Each worktree has its own file; there is no cross-worktree state or sharing.

### Hook Pipeline

Claude Code invokes hooks as `claude → sh -c '<command>' → fab hook <event>`. The `sh -c` wrapper means `os.Getppid()` inside the hook returns the `sh` PID, not Claude's. Claude's PID is at depth 2 and is resolved by a grandparent walk in `internal/proc/` (see §Grandparent PID Walker below).

All three session-scoped hooks (`fab hook stop|session-start|user-prompt`) parse stdin as a JSON object and extract `session_id` (required) and `transcript_path` (optional). Malformed JSON or missing `session_id` causes the hook to exit 0 silently — matching the existing swallow-on-error pattern. The fourth hook, `fab hook artifact-write`, parses a different payload shape (file path) and does not participate in `_agents` writes.

Per-event write/clear semantics:

| Hook event | Claude event | Action on `_agents[session_id]` |
|------------|--------------|----------------------------------|
| `fab hook stop` | Stop (turn end, agent now idle) | **Write full entry**: `idle_since = now()`, `change`, `pid`, `tmux_server`, `tmux_pane`, `transcript_path` (each present when available) |
| `fab hook session-start` | SessionStart (fresh session beginning) | **Delete entry entirely** — old session state is gone |
| `fab hook user-prompt` | UserPromptSubmit (agent about to become active) | **Remove only `idle_since`** — preserve `change`, `pid`, `tmux_server`, `tmux_pane`, `transcript_path` |
| `fab hook artifact-write` | PostToolUse (Write/Edit) | No-op on `_agents` — unrelated artifact bookkeeping path |

The `user-prompt` clear-idle-only semantics preserve pane-map correlation properties across the idle → active transition: `fab pane map` can immediately show "active agent here" without waiting for the next Stop event to reconstruct the entry.

Writes are **independent of active-change state**. An agent running in discussion mode (no `.fab-status.yaml` symlink) still produces an `_agents` entry; the `change` field is simply absent or empty.

Each hook handler makes exactly **one** runtime call — `WriteAgent` / `ClearAgent` / `ClearAgentIdle` with `gcInterval = 180*time.Second` — which is a thin typed wrapper over `runtime.UpdateAgent(fabRoot, createIfMissing, mutate, gcInterval)`: load once (under the lock), apply the entry mutation, run the GC sweep inline when due, save once. Passing `gcInterval <= 0` (`runtime.NoGC`) disables the sweep. Before mz4q each handler ran a second, independent `GCIfDue` load-modify-save cycle after its mutation; the sweep is now folded into the same round-trip. When `createIfMissing` is false (clear paths, GC-only) and the file is absent, the call is a complete no-op; the stop-hook write path passes true and creates the file.

**Tmux env handling**: Hooks read `$TMUX_PANE` and `$TMUX` from the environment. `$TMUX_PANE` writes verbatim to `tmux_pane` (including the `%` prefix). `$TMUX` is parsed — the first comma-separated component is a socket path, and its basename writes to `tmux_server`. When either env var is absent, the corresponding field is omitted. Claude Code's `sh -c` wrapper preserves both env vars into hook subprocesses (probe-verified on Linux).

**Concurrency & durability**: All writes use the `SaveFile` path (temp file + rename), so a reader never observes a torn file — but rename atomicity does NOT prevent **lost updates**: two unlocked load-mutate-save cycles are last-writer-wins over the whole document (the realistic race is multiple Claude sessions hooking in the *same* worktree, e.g. a lost `ClearAgentIdle` letting the operator inject keystrokes into a busy agent, or a lost `WriteAgent` blocking `fab pane send`). Since mz4q every mutator runs its full load-mutate-save cycle holding an exclusive advisory flock on the sibling `.fab-runtime.yaml.lock` (shared `internal/lockfile` helper: `O_CREATE` + `LOCK_EX|LOCK_NB` retry, ~10s bounded acquisition; the lock file is gitignored and never deleted — flock state, not file existence, carries the lock). The save is **skipped entirely when nothing changed** (no entry to clear, no `idle_since` to remove, GC throttled), so write-free paths stay write-free. `SaveFile` deliberately does NOT fsync: the file is ephemeral, fully re-derivable state ("re-populates on next hook event") and the fsync sat on every hook event's latency path — durability follows criticality (contrast `statusfile.Save`, which fsyncs because `.status.yaml` is the pipeline's source of truth).

### Garbage Collection

The GC sweep runs **inline inside `runtime.UpdateAgent`** (`gcSweepIfDue` — pure in-memory aside from `kill(pid, 0)` probes; the caller owns load, save, and locking) whenever a hook handler's merged call passes `gcInterval = 180 * time.Second`. `GCIfDue(fabRoot, interval)` survives only as a one-line GC-only wrapper — `UpdateAgent(fabRoot, false, nil, interval)` — with **zero production call sites** (exercised by runtime tests). Behavior:

1. Load `.fab-runtime.yaml` (once, shared with the entry mutation). If the file is missing, the call is a no-op.
2. Read `last_run_gc`. If `now - last_run_gc < interval`, skip the sweep (throttled — no GC-driven write).
3. Iterate `_agents`. For each entry with a non-nil `pid` field, send signal 0 via `syscall.Kill(pid, 0)`. If the process is gone (ESRCH; EPERM counts as alive), delete the entry. Entries with live PIDs are preserved.
4. Entries **without** a `pid` field are preserved regardless of any other signal.
5. Update `last_run_gc = now()` and save — in the same single write as the entry mutation.

**GC-on-no-op**: the sweep runs even when the mutation half of the merged call was a no-op (e.g. `ClearAgent` for an absent session while the throttle has expired) — and a due sweep always dirties the map (at minimum `last_run_gc`), so it alone triggers the save.

**180-second throttle**: Matches the "once per 3 mins or so" cadence directed during design. The write-half of each hook invocation already absorbs the common-case cost; the GC sweep piggybacks on the same file read + write round-trip when it's actually due. (Since mz4q the implementation matches this description literally — previously the sweep was a second, independent read + write cycle per hook event.)

**`kill(pid, 0)` liveness**: POSIX-standard; no subprocess spawn; server-agnostic (does not depend on tmux sockets or platform-specific process tables). ESRCH means the PID is definitively gone. PID reuse is an accepted risk — the window (GC interval vs. OS PID reuse horizon) is small enough that the simplicity win dominates.

**Pid-less entries preserved indefinitely**: Entries without a `pid` field (typically non-tmux agents whose grandparent walker failed, or edge cases) are never pruned by this GC. This is a self-limiting leak — low frequency in practice. A secondary mtime-based sweep is possible future work if unbounded growth is ever observed.

### Pane-Map Matching Rule

`fab pane map` resolves a pane's agent state by scanning `_agents` in the worktree's `.fab-runtime.yaml` for an entry where:

- `tmux_pane == <pane_id>` (exact string equality, including the `%` prefix), **AND**
- (`tmux_server` is empty/absent **OR** `tmux_server` equals the basename of the active `$TMUX` socket or the `--server <name>` flag value)

Matched entry:

- Without `idle_since` → Agent column shows `active`
- With `idle_since` → Agent column shows `idle (<duration>)` (e.g., `idle (2m)`)

No match → Agent column shows `—` (em-dash).

**Independent of change**: This resolution path no longer short-circuits on whether the pane has an active change. An agent running in a discussion-mode worktree now populates the Agent column.

**Server disambiguation**: When two worktrees on different tmux servers both use the same pane ID (tmux allocates pane IDs per-server), the `tmux_server` property disambiguates. A `fab pane map --server runKit` invocation matches only entries whose `tmux_server` is empty or equals `"runKit"`.

**Cache discipline**: `ResolveAgentStateWithCache` reads each worktree's `.fab-runtime.yaml` at most once per `fab pane map` invocation, sharing the parsed map across all panes in that worktree.

### Three-Axis Model

`fab pane map` resolves three orthogonal axes independently:

| Axis | Source | Column |
|------|--------|--------|
| **Change** | `.fab-status.yaml` symlink at the worktree root → `fab/changes/<folder>/.status.yaml` | `Change` / `Stage` |
| **Agent** | `_agents` in `.fab-runtime.yaml` at the worktree root, matched by `tmux_pane` | `Agent` |
| **Process** | OS process tree (`/proc` on Linux, `ps` on macOS) — opt-in via `fab pane process` | Not in `map` output |

The axes do not share resolution code and do not gate each other. A pane may have:

- Change but no Agent (change was created by a shell tool, no Claude running) → `260417-... / spec / —`
- Agent but no Change (discussion mode) → `— / — / idle (2m)`
- Neither (plain shell) → `— / — / —`
- Both (normal working state) → `260417-... / spec / idle (2m)`

The Process axis is intentionally separate (and opt-in) because `/proc` walks cost 5–10ms per pane and are unnecessary for the common `fab pane map` call path.

### Grandparent PID Walker

Claude invokes hooks via `sh -c '<command>'`. The hook process's parent is `sh`; its grandparent is Claude. `os.Getppid()` returns the `sh` PID — one level short. The `internal/proc/` package resolves Claude's PID by one additional step:

- **Linux** (`proc_linux.go`, `//go:build linux`): reads `/proc/$PPID/status` and parses the `PPid:` line. Cheap; no subprocess.
- **macOS** (`proc_darwin.go`, `//go:build darwin`): execs `ps -o ppid= -p $PPID` and parses the trimmed stdout. Same function signature as Linux.

Both files export `ClaudePID() (int, error)`. Failure (e.g., parent already exited) causes the hook to write its entry without the `pid` field — the hook never fails on walker error.

This mirrors the established `internal/pane/pane_process_{linux,darwin}.go` pattern: platform-split via Go build tags, identical signatures, isolation of platform-specific code.

## Design Decisions

### Session_id as identity key
**Decision**: The `_agents` map is keyed by Claude's `session_id` UUID (from the hook stdin JSON payload).
**Why**: `session_id` is stable across all hook fires within a session, is provided directly by Claude Code (no platform-specific lookup), is a UUID (zero collision risk across Claude restarts), and is human-correlatable with the transcript file. Probe-verified across Stop and UserPromptSubmit events.
**Rejected**: PID-as-key — PID reuse creates stale-entry collisions across Claude restarts; requires platform-specific lookup for identity. Pane-ID-as-key — tmux-coupled, fails for non-tmux agents (IDE terminals, SSH, CI). Compound key `(session_id, tmux_pane)` — forces non-tmux agents to synthesize fake pane IDs; over-specifies identity.
*Source*: 260419-o5ej-agents-runtime-unified

### Optional tmux properties, not part of key
**Decision**: `tmux_server` and `tmux_pane` are entry properties, absent when fab runs outside tmux.
**Why**: Fab does not require tmux — agents in IDE terminals, SSH sessions, CI jobs are still trackable. Storing these as properties keeps identity uniform across tmux and non-tmux contexts.
**Rejected**: Requiring tmux for tracking — would lose coverage of major agent-running environments.
*Source*: 260419-o5ej-agents-runtime-unified

### Inline GC via throttle field, no CLI surface
**Decision**: GC runs from every hook handler, throttled by a top-level `last_run_gc` timestamp with a 180s interval. No `fab runtime gc` subcommand is exposed.
**Why**: Hooks already hold the "something happened" signal that justifies a sweep; piggybacking on their file I/O is free. 180s matches the ≈3-minute cadence directed in design. `kill(pid, 0)` is cheap and server-agnostic. No operational surface to document, cron, or monitor.
**Rejected**: Separate `fab runtime gc` subcommand + cron/systemd — adds operational complexity. Per-entry TTL — requires per-entry timestamps and wall-clock comparison without liveness signal. Cross-server tmux enumeration GC — platform-specific, brittle, doesn't cover non-tmux entries.
*Source*: 260419-o5ej-agents-runtime-unified; *Updated by*: 260612-mz4q-shared-state-concurrency-hook-hot-path (sweep folded into the mutator's own `UpdateAgent` load/save round-trip; `GCIfDue` reduced to a test-facing one-line wrapper with zero production call sites)

### Lost-update protection via sibling flock, not rename alone
**Decision**: Every `.fab-runtime.yaml` load-mutate-save cycle runs while holding an exclusive advisory flock on the sibling `.fab-runtime.yaml.lock`, taken inside `runtime.UpdateAgent` via the shared `internal/lockfile` helper (`O_CREATE` + `LOCK_EX|LOCK_NB` retry loop, bounded ~10s acquisition; lock files are gitignored and never deleted). The save is skipped when neither the mutation nor GC changed anything.
**Why**: Temp+rename only prevents torn files; it does not serialize concurrent load-mutate-save cycles, which are last-writer-wins over the whole document. The realistic race surface is multiple Claude sessions hooking in the same worktree plus the cross-process GC clobber — concrete failure modes were a lost `ClearAgentIdle` (operator injects keystrokes into a busy agent) and a lost `WriteAgent` (idle agent unmatched, `pane send` blocked until the next hook fire). One helper wrapping the whole cycle fixes the class at its root. Bounded (non-blocking + retry) acquisition converts a pathological holder into a clear error instead of an indefinite deadlock; the uncontended path is a single syscall. `flock` works on both supported GOOS (linux + darwin) — no build-tag split.
**Rejected**: Per-call-site patches (fixes instances, not the class). Unbounded blocking `LOCK_EX` (silent deadlock risk in unattended pipelines). Deleting lock files after release (racy — flock state, not file existence, carries the lock).
*Source*: 260612-mz4q-shared-state-concurrency-hook-hot-path

### Durability follows criticality — runtime fsync dropped
**Decision**: `runtime.SaveFile` no longer fsyncs the temp file before rename; `statusfile.Save` gained the fsync in the same change.
**Why**: The runtime file is ephemeral, fully re-derivable state ("re-populates on next hook event"), and the per-write fsync sat on every hook event's latency path — exactly the hot path the merged `UpdateAgent` exists to thin out. `.status.yaml`, by contrast, is the pipeline state machine's source of truth, where a crash leaving an empty/torn file is unacceptable. A one-line revert if the posture proves wrong.
**Rejected**: fsync both files (taxes every hook event for state that self-heals). fsync neither (risks the pipeline's source of truth).
*Source*: 260612-mz4q-shared-state-concurrency-hook-hot-path

### Clean-slate migration, not faithful conversion
**Decision**: The 1.4.0 → 1.5.0 migration deletes any existing `.fab-runtime.yaml` at each user worktree's repo root.
**Why**: Old entries have no `session_id` (hooks didn't capture it before this change) and no correlation path to the current session. Runtime state is ephemeral — it self-heals within one hook cycle. The transient display cost (Agent column shows `—` for up to one hook cycle post-migration) is bounded and acceptable.
**Rejected**: Synthesize `session_id` for legacy entries — no stable identity exists; fabricated UUIDs would go stale immediately. Dual-read (consume both schemas) — prolongs bifurcation forever.
*Source*: 260419-o5ej-agents-runtime-unified

### Platform-split grandparent walker under `internal/proc/`
**Decision**: Platform-specific `proc_linux.go` (reads `/proc/$PPID/status`) and `proc_darwin.go` (execs `ps -o ppid= -p $PPID`) with Go build tags selecting at compile time. Both files export the same `ClaudePID() (int, error)` signature.
**Why**: Mirrors the existing `internal/pane/pane_process_{linux,darwin}.go` convention. `/proc` is strictly cheaper on Linux (no subprocess). Build tags keep platform-specific code isolated and testable.
**Rejected**: Shell-out to `ps` on both platforms — slower on Linux; no reason to avoid `/proc` where it's available.
*Source*: 260419-o5ej-agents-runtime-unified

### user-prompt clears only `idle_since`, preserves other entry properties
**Decision**: `fab hook user-prompt` removes the `idle_since` key from `_agents[session_id]` and leaves `change`, `pid`, `tmux_server`, `tmux_pane`, `transcript_path` intact.
**Why**: Preserves pane-map correlation across the idle → active transition, so `fab pane map` can show "active agent here" immediately. A full delete would create a brief window where the pane appears agentless until the next Stop event reconstructs the entry.
**Rejected**: Full delete on user-prompt — wasteful (Stop will re-write the same properties) and creates a display gap.
*Source*: 260419-o5ej-agents-runtime-unified

### Pid-less entries preserved by GC
**Decision**: GC skips entries without a `pid` field. Only entries with a `pid` are subject to `kill(pid, 0)` liveness checks.
**Why**: Pid-less entries are low-frequency (non-tmux agents where the grandparent walker failed). Pruning them without a liveness signal would require an alternate signal (mtime, explicit TTL). The simpler invariant — "no pid → no pruning" — keeps GC focused on the dominant case.
**Deferred work**: If pid-less entries grow unboundedly in practice, a secondary mtime-based sweep can be added later. Not required for the first cut.
*Source*: 260419-o5ej-agents-runtime-unified
