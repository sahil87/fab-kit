# Intake: Agents Runtime Unified

**Change**: 260419-o5ej-agents-runtime-unified
**Created**: 2026-04-19
**Status**: Draft

## Origin

> Refactor `.fab-runtime.yaml`: make agents first-class entries keyed by Claude `session_id`. `tmux_server` / `tmux_pane` / `pid` / `change` / `transcript_path` become optional properties on the agent entry. Hooks parse stdin JSON for `session_id` and write `_agents[session_id]` regardless of active-change status (fixes discussion-mode visibility gap in `fab pane map`). Add throttled GC (once per ~3 min via `last_run_gc`) that sweeps dead-PID entries via `kill(pid, 0)`. Add grandparent-PID walk (Linux `/proc`, macOS `ps`) so hooks can find Claude's PID past the `sh -c` wrapper. `fab pane map` joins `_agents` to panes via the `tmux_pane` property; agent state column now populates for discussion-mode panes. Three orthogonal axes: change (from `.fab-status.yaml` symlink), agent (from `_agents`), process (opt-in via `fab pane process`).

**Interaction mode**: conversational (iterative). The design emerged from a `/fab-discuss` session that:

1. Started with: *"how are agent states and fab states detected by fab in general (Mostly exposed in the fab pane map command)"*
2. Surfaced a visibility gap: `fab pane map` and the `fab hook stop|session-start|user-prompt` pipeline silently ignore agents running in panes where no change is active — the entire `/fab-discuss` / pre-intake phase is invisible. `fab pane send` rejects these panes (`unknown` state), `fab operator` can't detect them as spawnable-into.
3. Iterated through three schema proposals:
   - **v1 rejected**: pane-scoped `_panes[pane_id]` alongside existing change entries (additive). Rejected because schema bifurcation is permanent and tmux-coupling is unnecessary.
   - **v2 rejected**: PID-keyed `_agents[pid]` with tmux as properties. Rejected in favor of v3 after probing showed session_id is a better identity.
   - **v3 adopted** (this change): `_agents[session_id]` with tmux/change/pid as optional properties.
4. Live-probed Claude Code hook invocation behavior to verify foundational assumptions:
   - `$TMUX=/tmp/tmux-1001/fabKit,8671,0` and `$TMUX_PANE=%15` are preserved into hook subprocesses ✅
   - Hooks are invoked as `claude → sh -c '<command>' → hook` (single shell layer) — `os.Getppid()` returns the `sh` PID, not Claude's. Grandparent walk required.
   - Hook stdin receives a JSON payload with `session_id`, `transcript_path`, `cwd`, `permission_mode`, `hook_event_name`. Same shape across Stop and UserPromptSubmit probes. This is what unlocks session_id as identity.

## Why

**The pain point**: `.fab-runtime.yaml` today keys agent-state entries by change folder name. An agent can only be "seen" after the user creates a change (`/fab-new`). Before that — during `/fab-discuss`, free-form exploration, or any intake-composition work — the pane renders `— — —` in `fab pane map`, is rejected by `fab pane send` without `--force`, and is indistinguishable from a plain shell pane. Multi-agent orchestration (`/fab-operator`) cannot route tasks to "idle discussion panes" because those panes don't register as agents.

**The consequence if we don't fix this**: Every operator-driven workflow has a cold-start gap. The human operator has to manually acknowledge "I spawned an agent here" because the tooling cannot. `fab pane send` becomes unreliable as a general primitive (always needs `--force` for discussion panes, which defeats the idleness gate). As multi-agent coordination grows, this gap compounds.

**Why this approach over alternatives**: Three orthogonal axes (change, agent, process) exist independently. The old schema conflates change with agent — no change means no agent entry, even if an agent is running. Treating agents as first-class entities keyed by their stable identity (Claude's session_id UUID) cleanly separates the axes. The change is then just an optional property of the agent, like tmux pane or PID. One concept replaces two (change-agents vs pane-agents). No schema bifurcation, no migration of entries between buckets when `/fab-new` runs — the same `_agents[session_id]` entry just gains a `change` field.

**Rejected alternatives**:

- *Merge `fab pane process`'s `has_agent` into `map` by default* — adds `/proc` walk cost to the common path; can't distinguish idle/active without the hook signal. Kept as opt-in.
- *Worktree-scoped runtime key (no pane id)* — multiple discussion panes in the same worktree would collide on `idle_since`.
- *PID-as-key* — PID reuse creates stale-entry collisions across Claude restarts; platform-specific code to fetch Claude's PID; less human-correlatable than session_id. session_id is a UUID, never collides.
- *Cross-server GC enumeration* — tmux has no built-in cross-server listing; would require platform-specific socket-directory walks. `kill(pid, 0)` on each entry's stored PID is cheaper and server-agnostic.

## What Changes

### 1. Schema (`.fab-runtime.yaml` at each worktree's repo root)

**Before**:
```yaml
260417-2fbb-pane-server-flag:
  agent:
    idle_since: 1729450000
```

**After**:
```yaml
_agents:
  "d630bcf0-8820-4dd1-a99c-9bda5ea72c88":       # key: Claude session_id (UUID)
    change: "260417-2fbb-pane-server-flag"       # optional — absent/empty = discussion mode
    idle_since: 1729450100                       # unix ts
    pid: 2356168                                 # optional — Claude's PID, for GC liveness
    tmux_server: "fabKit"                        # optional — parsed from $TMUX
    tmux_pane: "%15"                             # optional — from $TMUX_PANE
    transcript_path: "/home/.../d630bcf0-...jsonl"  # optional — for correlation
last_run_gc: 1729450200                          # top-level — throttles GC sweeps to every ~3 min
```

Single top-level `_agents` bucket replaces the per-folder `agent:` block. `last_run_gc` is a sibling at the top level (not nested inside `_agents`).

### 2. Hook Changes (`src/go/fab/cmd/fab/hook.go` + `src/go/fab/internal/runtime/runtime.go`)

Every hook handler:

1. **Parses stdin JSON** to extract `session_id` (and optionally `transcript_path`). A new helper `hooklib.ParseContext()` (or similar) centralizes JSON parsing.
2. **Resolves Claude's PID** via grandparent walk — new helper in a platform-split file pair (e.g., `internal/proc/ppid_linux.go` / `internal/proc/ppid_darwin.go`) mirroring the existing `pane_process_linux.go` pattern. Linux reads `/proc/$PPID/status`'s `PPid:` line; macOS execs `ps -o ppid= -p $PPID`.
3. **Reads tmux env** — `$TMUX_PANE`, `$TMUX` (parsed to extract socket-label). Absent → omit from written entry.
4. **Writes or clears `_agents[session_id]`** — regardless of whether a change is active.

Hook semantics per event:

| Event | Action on `_agents[session_id]` |
|---|---|
| `stop` | Write entry with `idle_since = now()`, refreshing all properties |
| `session-start` | Delete entry (new session starting; old should be gone) |
| `user-prompt` | Delete entry (agent is about to be active) — OR update: remove `idle_since` only, keep other props |
| `artifact-write` | Unchanged (artifact bookkeeping path; does not touch `_agents`) |

*Decision to confirm in spec*: `user-prompt` behavior — full delete vs. clear-idle-only. Clear-idle-only preserves pane correlation properties, which helps `fab pane map` distinguish "active agent in this pane" from "no agent at all."

### 3. GC (`src/go/fab/internal/runtime/runtime.go`)

New function `GCIfDue(fabRoot string, interval time.Duration)`:

1. Load `.fab-runtime.yaml`. If file missing, no-op.
2. Read `last_run_gc`. If `now - last_run_gc < 180s`, return (throttled).
3. For each `_agents[session_id]`: if entry has `pid` field and `kill(pid, 0)` returns `ESRCH` → delete entry.
4. Update `last_run_gc = now()`, save atomically.

Called at the end of every hook handler (cheap no-op when throttled; sub-100µs when actually running).

### 4. `fab pane map` Changes (`src/go/fab/cmd/fab/panemap.go` + `src/go/fab/internal/pane/pane.go`)

`ResolvePaneContext` currently short-circuits agent resolution on `folderName == ""` (pane.go:116-134). Rewrite:

1. Change resolution (via `.fab-status.yaml` symlink) — **unchanged**, returns optional folder name.
2. Stage resolution (via `.status.yaml` on that folder) — **unchanged**.
3. Agent resolution — **new logic**: scan `_agents` entries looking for one where `tmux_pane == paneID` AND (`tmux_server` unset OR `tmux_server == currentServer`). If found: agent state is `active` (no `idle_since`) or `idle (<duration>)` (with `idle_since`). Independent of change resolution.

Display semantics (table + JSON):

| Scenario | Change | Stage | Agent | Change from today |
|---|---|---|---|---|
| Change active, agent idle | `260417-...` | `spec` | `idle (2m)` | none |
| Change active, agent active | `260417-...` | `spec` | `active` | none |
| **Discussion mode, agent idle** | `—` | `—` | `idle (2m)` | **newly visible** |
| **Discussion mode, agent active** | `—` | `—` | `active` | **newly visible** |
| No agent matched to pane | `—` or `260417-...` | `—` or stage | `—` | unchanged |

JSON field semantics: `agent_state` and `agent_idle_duration` populate whenever an `_agents` entry matches the pane, independent of `change`/`stage` fields.

### 5. `fab pane send` Idle Check (`src/go/fab/cmd/fab/pane_send.go`)

Reuses `ResolvePaneContext`. With discussion-mode visibility, panes that were previously `unknown` (rejected without `--force`) now resolve to `idle` or `active` correctly. No code change expected in `pane_send.go` itself — it inherits the fix.

### 6. Grandparent PID Walker

New internal package `src/go/fab/internal/proc/`:

- `proc_linux.go`: `func ClaudePID() (int, error)` reads `/proc/$PPID/status`, parses `PPid:` field.
- `proc_darwin.go`: same signature, execs `ps -o ppid= -p $PPID`.

Behavior is identical across platforms — returns the grandparent PID of the calling process, which — given Claude's `sh -c` hook invocation pattern — is Claude itself. Probe-verified on Linux; macOS behavior assumed to match (tmux + sh are cross-platform; Claude's hook invocation is platform-independent).

### 7. Migration (`src/kit/migrations/`)

New migration file following the pattern in `src/kit/migrations/` (see `docs/memory/fab-workflow/migrations.md`). Strategy: **clean-slate** — delete any existing `.fab-runtime.yaml` in the user's worktrees. Runtime state is ephemeral (re-populated on next hook event). Entries in the old schema would not have `session_id` (not available to old hooks), so we cannot faithfully migrate them. Losing "currently idle 3m" information is acceptable — it re-populates within one hook cycle.

### 8. Documentation Updates (per constitution)

- `src/kit/skills/_cli-fab.md` — unchanged (no new fab subcommands). *Confirm during spec: no new `fab runtime gc` or `fab runtime show` needed.*
- `docs/memory/fab-workflow/pane-commands.md` — update agent state section to reflect new resolution logic and three-axis model.
- New: `docs/memory/fab-workflow/runtime-agents.md` — comprehensive doc of `.fab-runtime.yaml` schema, hook write pipeline, GC semantics, pane-map matching rule. Hydrated as part of this change.
- `docs/specs/` updates if pane map observable behavior changes — yes, `skills/SPEC-pane-commands.md` or equivalent needs updating to describe the new column semantics.

## Affected Memory

- `fab-workflow/pane-commands`: (modify) agent state column now populates in discussion mode; matching logic via `_agents[*].tmux_pane` property; three-axis model for Change/Agent/Process columns.
- `fab-workflow/runtime-agents`: (new) authoritative doc for `.fab-runtime.yaml` schema, hook write pipeline, GC throttling, session_id keying.
- `fab-workflow/schemas`: (modify) add cross-reference to runtime-agents; note that `.fab-runtime.yaml` is a separate schema from workflow.yaml.
- `fab-workflow/preflight`: (modify, minor) — preflight currently doesn't touch runtime file; confirm during spec whether it needs to reference new schema for any validation.

## Impact

### Code

- `src/go/fab/internal/runtime/runtime.go` — major refactor: new API surface (`SetAgentIdle`, `ClearAgent`, `GCIfDue`), legacy `SetIdle`/`ClearIdle` removed or wrapped
- `src/go/fab/internal/pane/pane.go` — `ResolvePaneContext`, `ResolveAgentState`, `ResolveAgentStateWithCache` rewritten for new schema
- `src/go/fab/cmd/fab/hook.go` — all hook handlers parse stdin JSON and write to new bucket; GC call added
- `src/go/fab/cmd/fab/panemap.go` — matching logic against `_agents` by `tmux_pane` property
- `src/go/fab/cmd/fab/pane_capture.go` — inherits fix via `ResolvePaneContext`
- `src/go/fab/cmd/fab/pane_send.go` — inherits fix via `ResolvePaneContext`
- New: `src/go/fab/internal/proc/proc_{linux,darwin}.go` — grandparent PID walker
- New helper (probably in `internal/hooklib/`): stdin JSON payload parser exposing `SessionID()`, `TranscriptPath()`, etc.

### Data / Config

- `.fab-runtime.yaml` schema change (breaking, migrated)
- New migration file in `src/kit/migrations/`

### Tests

- `internal/runtime/*_test.go` — cover new `SetAgentIdle`, `ClearAgent`, `GCIfDue` (including throttle behavior, dead-PID sweep)
- `internal/pane/*_test.go` — cover new matching logic, cache behavior
- `cmd/fab/hook_test.go` — JSON payload parsing, tmux env handling, absent-tmux fallback
- `cmd/fab/panemap_test.go` — discussion-mode visibility, three-axis independence
- New: `internal/proc/*_test.go` — platform-specific grandparent lookup

### External consumers

- **run-kit** (`~/code/sahil87/run-kit`): The motivating upstream for `--server`/`-L`; consumes `fab pane map --json --all-sessions`. Impact analysis runs in parallel via a separate subagent; any required run-kit change will be queued as a draft intake there.

## Open Questions

- Should `user-prompt` hook fully delete the `_agents[session_id]` entry or just clear `idle_since` (preserving `tmux_pane` / `tmux_server` for correlation)? *Lean: clear-idle-only — gives `fab pane map` better signal for "active agent here" vs "no agent here."*
- Does `artifact-write` need to participate in `_agents` writes? *Lean: no — its job is artifact bookkeeping, not agent state.*
- What if `session_id` is absent from stdin payload (older Claude Code versions)? *Lean: skip silently — same as current swallow-on-error behavior; log at debug level.*
- Confirm macOS probe: `$TMUX_PANE` / `$TMUX` preservation and `sh -c` wrapper behavior. *Probed Linux only; high confidence macOS matches but deserves a manual verification before ship.*
- Is the 180s GC throttle the right value? (User suggested "once per 3 mins or so.") Too-frequent GC wastes syscalls; too-infrequent grows the file unboundedly. *180s is a reasonable default; configurable would be over-engineering.*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Schema uses `_agents[session_id]` bucket with `last_run_gc` as sibling top-level field | Discussed and user confirmed across iterations; captured in Origin as "v3 adopted" | S:98 R:70 A:90 D:95 |
| 2 | Certain | session_id (Claude UUID from hook stdin) is the agent identity key | Probe-verified: payload contains `session_id` stable across all probed events (Stop + UserPromptSubmit); user confirmed this direction replaces PID-as-key | S:98 R:70 A:95 D:95 |
| 3 | Certain | `tmux_server`, `tmux_pane`, `pid`, `change`, `transcript_path` are OPTIONAL properties on the agent entry | User: *"Instead of making the tmux server and pane id as keys, can we store them as agent properties. Why? Because they are optional - no necessary fab starts in a tmux session"* | S:98 R:70 A:90 D:95 |
| 4 | Certain | GC throttle interval = 180 seconds (~3 min) via `last_run_gc` field | User: *"Have a GC. But it can happen less frequently - using a last_run_gc: <timestamp> field. Run it only once per 3 mins or so."* | S:98 R:85 A:90 D:90 |
| 5 | Certain | Hooks are invoked as `claude → sh -c → hook`; grandparent PID walk required | Probe-verified on Linux for both Stop and UserPromptSubmit events; `os.Getppid()` returns sh, Claude is at depth 2 | S:98 R:80 A:95 D:95 |
| 6 | Certain | `$TMUX_PANE` and `$TMUX` are preserved through hook invocation | Probe-verified on Linux; present in captured env dumps for both events | S:95 R:75 A:90 D:95 |
| 7 | Certain | Three orthogonal axes in `fab pane map`: Change (from `.fab-status.yaml`), Agent (from `_agents`), Process (opt-in via `fab pane process`) | User: *"Change is a different axis, Agent running + its state is a different axis, The running process is a different axis"* | S:98 R:70 A:85 D:90 |
| 8 | Certain | `fab pane process` remains opt-in — no default merge into `fab pane map` | Discussed; rejected merge-by-default to avoid /proc-walk cost on common path | S:90 R:85 A:85 D:90 |
| 9 | Certain | GC liveness via `kill(pid, 0)` — ESRCH means entry prunable | POSIX-standard; user accepted PID-reuse risk in design discussion ("over-engineering. We will live with the risk."); cross-platform; zero subprocess cost | S:90 R:85 A:92 D:92 |
| 10 | Certain | Platform-split grandparent PID walker under `internal/proc/` mirrors existing `internal/pane/pane_process_{linux,darwin}.go` pattern | Established codebase convention; user named this pattern during design as the model to follow | S:92 R:88 A:95 D:88 |
| 11 | Confident | Migration strategy is clean-slate: delete old `.fab-runtime.yaml`; state re-populates via next hook | Runtime state is ephemeral; old entries lack session_id so faithful migration impossible; constitution allows migrations for data restructuring | S:72 R:86 A:82 D:72 | <!-- assumed: clean-slate chosen over synthesize-entries — state is ephemeral so loss is bounded -->
| 12 | Confident | New memory file `fab-workflow/runtime-agents.md` (vs. appending to pane-commands) | pane-commands is already substantial; runtime-agents is a coherent topic; memory organizes by topic per `docs-reorg-memory` skill conventions | S:70 R:90 A:80 D:75 |
| 13 | Confident | `user-prompt` hook clears `idle_since` only, preserves other properties (tmux_pane, tmux_server, pid) | Preserves pane-map correlation for "active agent here" signal; full-delete forces reconstruction on next Stop. Scope is one hook handler — easily revised at spec stage if review flips it. | S:65 R:78 A:72 D:65 | <!-- assumed: clear-idle-only over full-delete — spec stage will re-examine -->
| 14 | Confident | `session_id` absent from stdin payload → hook skips silently (logs at debug level) | Matches existing swallow-on-error pattern in hooks; alternative (synthesize from PID) adds complexity for a rare edge case (older Claude Code versions) | S:65 R:85 A:78 D:75 |
| 15 | Tentative | macOS preserves `$TMUX_PANE`/`$TMUX` identically to Linux; `sh -c` wrapper behavior matches | Probed Linux only; tmux + sh are cross-platform and Claude's hook invocation is platform-independent, but empirically unverified | S:55 R:80 A:70 D:75 | <!-- assumed: macOS matches Linux hook env behavior; verify before ship -->

15 assumptions (10 certain, 4 confident, 1 tentative, 0 unresolved).
