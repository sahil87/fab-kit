# Spec: Agents Runtime Unified

**Change**: 260419-o5ej-agents-runtime-unified
**Created**: 2026-04-19
**Affected memory**: `docs/memory/fab-workflow/runtime-agents.md`, `docs/memory/fab-workflow/pane-commands.md`, `docs/memory/fab-workflow/schemas.md`

## Non-Goals

- **No new top-level `fab runtime` CLI subcommands** — GC is invoked inline from hooks, no external `fab runtime gc` or `fab runtime show` commands are added.
- **No changes to `.status.yaml` schema** — this change is scoped to `.fab-runtime.yaml` and the hook/pane-map code paths.
- **No faithful migration of existing runtime entries** — clean-slate migration deletes the old file; old entries lack `session_id` and cannot be reconstructed.
- **No merge of `fab pane process` into `fab pane map` by default** — process detection remains opt-in via the separate subcommand.
- **No cross-server pane enumeration or cross-server GC** — GC operates on `.fab-runtime.yaml` at the current worktree only; it does not enumerate tmux sockets.

---

## Runtime Schema: `.fab-runtime.yaml` Structure

### Requirement: Agent entries keyed by session_id

The file SHALL contain a single top-level `_agents` map keyed by Claude Code `session_id` (a UUID string from the hook stdin JSON payload). Each entry SHALL contain `idle_since` (Unix timestamp integer) when the agent is idle. Each entry MAY contain these optional properties: `change` (string — change folder name, absent or empty when the agent is in discussion mode), `pid` (integer — Claude Code process ID for GC liveness), `tmux_server` (string — socket label parsed from `$TMUX`), `tmux_pane` (string — pane ID from `$TMUX_PANE` including the `%` prefix), `transcript_path` (string — from hook payload). The file SHALL contain a top-level `last_run_gc` field (Unix timestamp integer) to throttle GC sweeps.

#### Scenario: Active change with tmux agent

- **GIVEN** a Claude agent running in tmux pane `%15` with session_id `d630bcf0-8820-...`, working on change folder `260417-2fbb-pane-server-flag`
- **WHEN** the Stop hook fires for that session
- **THEN** `.fab-runtime.yaml` contains `_agents["d630bcf0-8820-..."]` with `idle_since`, `change: "260417-2fbb-pane-server-flag"`, `pid`, `tmux_server`, `tmux_pane: "%15"`, `transcript_path`

#### Scenario: Discussion-mode agent without change

- **GIVEN** a Claude agent running in tmux pane `%15` with session_id `d630bcf0-...`, no active change (`.fab-status.yaml` symlink absent)
- **WHEN** the Stop hook fires
- **THEN** `.fab-runtime.yaml` contains `_agents["d630bcf0-..."]` with `idle_since`, `pid`, `tmux_server`, `tmux_pane`, `transcript_path`, AND no `change` key (or empty-string value)

#### Scenario: Non-tmux agent

- **GIVEN** a Claude agent running outside tmux (no `$TMUX` or `$TMUX_PANE` env vars)
- **WHEN** the Stop hook fires
- **THEN** `.fab-runtime.yaml` contains `_agents["<session_id>"]` without `tmux_server` or `tmux_pane` keys (or with those keys empty-string)

### Requirement: Schema location invariants

`.fab-runtime.yaml` SHALL remain at the repo root of each git worktree (one file per worktree). The file SHALL be created on first write and SHALL NOT be required to exist for read paths (missing file is equivalent to `_agents: {}`).

#### Scenario: First hook write in a new worktree

- **GIVEN** a worktree with no `.fab-runtime.yaml`
- **WHEN** any hook writes an entry
- **THEN** the file is created atomically (temp-file + rename) with the new entry as its sole `_agents` member

#### Scenario: Missing file on read

- **GIVEN** a worktree with no `.fab-runtime.yaml`
- **WHEN** `fab pane map` resolves an agent for a pane in that worktree
- **THEN** the agent column displays `—` (no error, no stderr warning)

---

## Hook Integration

### Requirement: Hook stdin payload parsing

The `fab hook stop`, `fab hook session-start`, `fab hook user-prompt` subcommands SHALL parse stdin as a JSON object and extract `session_id` (required), `transcript_path` (optional). The existing `fab hook artifact-write` subcommand SHALL continue to parse its existing payload shape (file path) and SHALL NOT write to `_agents`. When stdin JSON is malformed or `session_id` is absent, hooks SHALL silently exit success (matching the existing swallow-on-error pattern).

#### Scenario: Hook with well-formed payload

- **GIVEN** `{"session_id": "uuid-1", "transcript_path": "/path/to/transcript.jsonl", "hook_event_name": "Stop"}` on stdin
- **WHEN** `fab hook stop` runs
- **THEN** the hook extracts `session_id: "uuid-1"` and proceeds to write the runtime entry

#### Scenario: Hook with missing session_id

- **GIVEN** stdin JSON `{"hook_event_name": "Stop"}` with no `session_id` field
- **WHEN** `fab hook stop` runs
- **THEN** the hook exits 0 without writing any entry to `.fab-runtime.yaml`

#### Scenario: Hook with malformed stdin

- **GIVEN** stdin containing `not-json` or empty input
- **WHEN** `fab hook stop` runs
- **THEN** the hook exits 0 without writing

### Requirement: Hook write semantics per event

The `fab hook stop` subcommand SHALL write the full entry for `_agents[session_id]` with `idle_since = now()` and all currently-available optional properties. The `fab hook session-start` subcommand SHALL delete the entry at `_agents[session_id]` entirely (fresh session). The `fab hook user-prompt` subcommand SHALL remove only the `idle_since` key from `_agents[session_id]` and SHALL preserve the entry's other properties (`change`, `pid`, `tmux_server`, `tmux_pane`, `transcript_path`). Writes SHALL be independent of whether a change is active.

#### Scenario: Stop event with active change

- **GIVEN** session `uuid-1` in pane `%5` working on change `260417-2fbb-...`
- **WHEN** `fab hook stop` runs
- **THEN** `_agents["uuid-1"]` contains `idle_since`, `change: "260417-2fbb-..."`, `pid`, `tmux_pane: "%5"`, etc.

#### Scenario: Stop event in discussion mode

- **GIVEN** session `uuid-2` in pane `%7` with no active change
- **WHEN** `fab hook stop` runs
- **THEN** `_agents["uuid-2"]` contains `idle_since`, `pid`, `tmux_pane: "%7"`, etc., and no `change` key

#### Scenario: User-prompt after Stop

- **GIVEN** `_agents["uuid-1"]` exists with `idle_since`, `tmux_pane: "%5"`, `change: "260417-..."`
- **WHEN** `fab hook user-prompt` runs for session `uuid-1`
- **THEN** `_agents["uuid-1"]` retains `tmux_pane`, `change`, `pid`, etc., but no `idle_since`

#### Scenario: Session-start clears fully

- **GIVEN** `_agents["uuid-1"]` exists
- **WHEN** `fab hook session-start` runs for session `uuid-1`
- **THEN** `_agents["uuid-1"]` is absent from the file

### Requirement: Hook read of tmux environment

Hooks SHALL read `$TMUX_PANE` and `$TMUX` from environment. If `$TMUX_PANE` is set, its value SHALL be written to `tmux_pane`. If `$TMUX` is set, the first comma-separated component SHALL be interpreted as a socket path; the basename of that path SHALL be written to `tmux_server`. If either env var is absent, the corresponding field SHALL be omitted from the entry.

#### Scenario: Tmux session with named socket

- **GIVEN** `$TMUX=/tmp/tmux-1001/fabKit,8671,0` and `$TMUX_PANE=%15`
- **WHEN** a hook writes an entry
- **THEN** the entry contains `tmux_server: "fabKit"` and `tmux_pane: "%15"`

#### Scenario: Default tmux socket

- **GIVEN** `$TMUX=/tmp/tmux-1001/default,8671,0` and `$TMUX_PANE=%3`
- **WHEN** a hook writes an entry
- **THEN** the entry contains `tmux_server: "default"` and `tmux_pane: "%3"`

#### Scenario: Outside tmux

- **GIVEN** `$TMUX` and `$TMUX_PANE` both unset
- **WHEN** a hook writes an entry
- **THEN** the entry omits both `tmux_server` and `tmux_pane`

---

## Claude PID Resolution (Grandparent Walker)

### Requirement: Grandparent PID resolution

Hooks SHALL resolve Claude Code's process ID by reading the PPID of the hook's parent process (the `sh` process that Claude invokes as `sh -c '<command>'`). This produces Claude's PID, which is recorded in the entry's `pid` field. The walker SHALL be implemented as a platform-split package `src/go/fab/internal/proc/` with `proc_linux.go` (reads `/proc/$PPID/status`, parses the `PPid:` line) and `proc_darwin.go` (invokes `ps -o ppid= -p $PPID`). Both files SHALL expose the same function signature. Failure to resolve the grandparent PID SHALL cause the hook to omit the `pid` field (not fail the hook).

#### Scenario: Linux grandparent resolution

- **GIVEN** a hook process on Linux whose parent is `sh` (PID 1000) and whose grandparent is `claude` (PID 500)
- **WHEN** the walker runs
- **THEN** it returns 500 by reading `/proc/1000/status` and parsing `PPid: 500`

#### Scenario: macOS grandparent resolution

- **GIVEN** a hook process on macOS whose parent is `sh` (PID 1000) and whose grandparent is `claude` (PID 500)
- **WHEN** the walker runs
- **THEN** it returns 500 by invoking `ps -o ppid= -p 1000` and parsing `500`

#### Scenario: Grandparent walk failure

- **GIVEN** a hook whose `/proc/$PPID/status` is unreadable (e.g., parent already exited, rare but possible)
- **WHEN** the walker runs
- **THEN** it returns an error, and the hook writes the entry without the `pid` field

---

## Garbage Collection

### Requirement: Throttled GC sweep

The runtime package SHALL expose a `GCIfDue(fabRoot, interval time.Duration) error` function. Each hook handler SHALL call this function with `interval = 180 * time.Second` immediately after its write/clear operation. GC behavior: (a) read `last_run_gc`; if `now - last_run_gc < interval`, return without further work; (b) otherwise, for each `_agents[session_id]` entry with a `pid` field, issue `kill(pid, 0)` — if the call returns ESRCH, delete the entry; (c) write `last_run_gc = now()` and save. Entries without a `pid` field SHALL NOT be pruned by GC.

#### Scenario: GC skipped when recent

- **GIVEN** `last_run_gc = now() - 60s` (within 180s window)
- **WHEN** a hook invokes `GCIfDue`
- **THEN** no entries are pruned and `last_run_gc` is unchanged

#### Scenario: GC prunes dead PID

- **GIVEN** `last_run_gc = now() - 300s`, `_agents["uuid-dead"]` has `pid: 12345`, process 12345 has exited (ESRCH)
- **WHEN** a hook invokes `GCIfDue`
- **THEN** `_agents["uuid-dead"]` is removed AND `last_run_gc` is updated to `now()`

#### Scenario: GC preserves live PID

- **GIVEN** `last_run_gc = now() - 300s`, `_agents["uuid-live"]` has `pid: 54321`, process 54321 is running
- **WHEN** a hook invokes `GCIfDue`
- **THEN** `_agents["uuid-live"]` remains AND `last_run_gc` is updated

#### Scenario: GC skips entry without pid

- **GIVEN** `_agents["uuid-nopid"]` has no `pid` field
- **WHEN** `GCIfDue` runs
- **THEN** the entry is preserved regardless of any other liveness signal

#### Scenario: GC with no runtime file

- **GIVEN** `.fab-runtime.yaml` does not exist
- **WHEN** `GCIfDue` runs
- **THEN** it returns `nil` without error (no-op)

---

## Pane Map Agent Resolution

### Requirement: Agent state resolution independent of change

`fab pane map` SHALL resolve a pane's agent state by scanning `_agents` entries in the worktree's `.fab-runtime.yaml` and matching on `tmux_pane` (exact string equality) AND `tmux_server` (if set in the entry, it SHALL equal the current `--server` value or the basename of the current `$TMUX` socket). If multiple entries match, the entry with the most recent `idle_since` (or the first active entry — an entry without `idle_since`) wins. This resolution SHALL be independent of whether the pane is associated with an active change.

#### Scenario: Discussion-mode pane

- **GIVEN** pane `%15` in worktree with no active change, `_agents["uuid-1"]` matches `tmux_pane: "%15"` with `idle_since: now()-120s`, no `change` field
- **WHEN** `fab pane map` renders this pane
- **THEN** the Change column shows `—`, the Stage column shows `—`, the Agent column shows `idle (2m)`

#### Scenario: Change pane

- **GIVEN** pane `%5` with active change `260417-...`, `_agents["uuid-2"]` matches `tmux_pane: "%5"` with `idle_since: now()-30s`, `change: "260417-..."`
- **WHEN** `fab pane map` renders this pane
- **THEN** the Change column shows `260417-...`, the Stage column shows the current stage, the Agent column shows `idle (30s)`

#### Scenario: Active agent (no idle_since)

- **GIVEN** pane `%7`, `_agents["uuid-3"]` matches with `tmux_pane: "%7"` and no `idle_since` field
- **WHEN** `fab pane map` renders this pane
- **THEN** the Agent column shows `active`

#### Scenario: No matching agent entry

- **GIVEN** pane `%9` with no entry in `_agents` whose `tmux_pane` equals `"%9"`
- **WHEN** `fab pane map` renders this pane
- **THEN** the Agent column shows `—` regardless of whether the pane has an active change

#### Scenario: Server disambiguation

- **GIVEN** entries `_agents["uuid-a"]` (tmux_server: "fabKit", tmux_pane: "%3") and `_agents["uuid-b"]` (tmux_server: "runKit", tmux_pane: "%3"), and `fab pane map --server runKit` is invoked
- **WHEN** resolving pane `%3` on server `runKit`
- **THEN** the Agent column reflects `uuid-b`'s state, not `uuid-a`'s

### Requirement: JSON output field semantics

The `fab pane map --json` output SHALL populate `agent_state` and `agent_idle_duration` whenever an `_agents` entry matches the pane, regardless of whether `change` and `stage` are null. The `change` and `stage` fields SHALL remain null when no active change exists on the pane's worktree.

#### Scenario: JSON for discussion-mode pane

- **GIVEN** pane `%15` with discussion-mode agent matched
- **WHEN** `fab pane map --json` is invoked
- **THEN** the JSON entry for `%15` has `"change": null, "stage": null, "agent_state": "idle", "agent_idle_duration": "2m"`

### Requirement: Cache discipline

`ResolveAgentStateWithCache` SHALL cache the parsed `.fab-runtime.yaml` per worktree across a single pane-map invocation, avoiding repeated file reads for panes that share a worktree.

#### Scenario: Multi-pane in same worktree

- **GIVEN** 3 panes in the same worktree
- **WHEN** `fab pane map` resolves all 3 agent columns
- **THEN** `.fab-runtime.yaml` for that worktree is read at most once

---

## Pane Send Validation

### Requirement: Send respects new state resolution

`fab pane send <pane> <text>` SHALL use the same `ResolvePaneContext` as `fab pane map`. A pane whose agent resolves to `idle` SHALL be accepted. A pane whose agent resolves to `active` or `unknown` (no matching entry) SHALL be rejected without `--force`. The behavioral change: panes previously rejected as `unknown` due to discussion mode SHALL now accept sends when the agent is idle.

#### Scenario: Send to discussion-mode idle pane

- **GIVEN** pane `%15` resolves to `idle (2m)` via discussion-mode matching (previously would have been `unknown`)
- **WHEN** `fab pane send %15 "hello"` runs without `--force`
- **THEN** the command succeeds and sends keystrokes

#### Scenario: Send to active pane rejected

- **GIVEN** pane `%15` resolves to `active` via new matching logic
- **WHEN** `fab pane send %15 "hello"` runs without `--force`
- **THEN** the command exits 1 with `Error: agent in pane %15 is not idle (state: active)`

---

## Migration

### Requirement: Clean-slate migration

A new migration file in `src/kit/migrations/` SHALL delete any existing `.fab-runtime.yaml` at each user worktree's repo root on first application of the migration. The migration SHALL be idempotent (running twice is equivalent to running once). The migration SHALL NOT touch any other file. Runtime state re-populates on the next hook event. The migration SHALL include a user-facing note explaining that transient idle-state display in `fab pane map` may show `—` for up to one hook cycle after the migration.

#### Scenario: Migration on worktree with existing runtime file

- **GIVEN** `.fab-runtime.yaml` exists with old-schema entries (change-folder-keyed `agent:` blocks)
- **WHEN** the migration runs
- **THEN** the file is deleted; no other files are modified

#### Scenario: Migration idempotent

- **GIVEN** `.fab-runtime.yaml` does not exist
- **WHEN** the migration runs
- **THEN** it completes without error (no-op)

---

## Documentation

### Requirement: Memory doc updates

A new memory file `docs/memory/fab-workflow/runtime-agents.md` SHALL document the new schema (all fields, optionality, typical values), the hook write pipeline (per-event write/clear semantics), the GC throttle design, the grandparent PID walker rationale and platform split, and the pane-map matching rule. The existing `docs/memory/fab-workflow/pane-commands.md` SHALL be updated to: (a) reflect the new agent-state resolution logic (matched by `_agents[*].tmux_pane`, independent of change), (b) document the three-axis model (Change / Agent / Process), (c) show the new display scenarios in the output table. The existing `docs/memory/fab-workflow/schemas.md` SHALL add a cross-reference to `runtime-agents.md`. The `docs/memory/fab-workflow/index.md` SHALL list the new `runtime-agents` entry.

#### Scenario: Memory index lists new file

- **GIVEN** the hydrate stage has run
- **WHEN** a reader opens `docs/memory/fab-workflow/index.md`
- **THEN** `runtime-agents` appears in the memory file list with a short description

### Requirement: Spec doc updates

The per-skill specs under `docs/specs/skills/` SHALL be updated if and where they describe pane map observable behavior (specifically agent column semantics). Other spec docs SHALL be updated only if their content conflicts with the new behavior.

---

## Design Decisions

1. **Session_id as identity key**: use Claude's `session_id` UUID (from hook stdin JSON) as the `_agents` map key.
   - *Why*: stable across hook fires within a session, UUID guarantees no collisions, available without platform-specific code (no PID grandparent walk needed for identity), human-correlatable with transcript files.
   - *Rejected*: PID-as-key — PID reuse creates stale collisions across Claude restarts; requires platform-specific lookup; less human-correlatable. Pane-ID-as-key — tmux-coupled, fails for non-tmux agents.

2. **Optional tmux properties, not part of key**: `tmux_server` / `tmux_pane` stored as entry properties, absent when fab runs outside tmux.
   - *Why*: fab doesn't require tmux — agents in IDE terminals, SSH sessions, CI, etc. are still trackable. Keeping these as properties keeps identity uniform.
   - *Rejected*: Compound key `(session_id, tmux_pane)` — forces non-tmux agents to synthesize fake pane IDs; over-specifies identity.

3. **Inline GC via throttle field, no CLI surface**: GC runs from every hook handler, throttled by a top-level `last_run_gc` timestamp with a 180s interval.
   - *Why*: no new CLI surface needed; hooks already hold the "something happened" signal that justifies a sweep; 180s matches the ≈3-minute cadence user requested; `kill(pid, 0)` is cheap and server-agnostic.
   - *Rejected*: Separate `fab runtime gc` subcommand + cron/systemd — adds operational complexity. Per-entry TTL — requires per-entry timestamps and wall-clock comparison without liveness signal. Cross-server tmux enumeration GC — platform-specific, brittle, doesn't cover non-tmux entries.

4. **Clean-slate migration, not faithful conversion**: delete old `.fab-runtime.yaml` on migration.
   - *Why*: old entries have no `session_id` (hooks didn't capture it) and no correlation path to the current session; runtime state is ephemeral and self-healing within one hook cycle.
   - *Rejected*: Synthesize `session_id` for legacy entries — no stable identity exists; fabricated UUIDs would go stale immediately. Dual-read (consume both schemas) — prolongs bifurcation forever.

5. **user-prompt clears only `idle_since`, preserves other entry properties**:
   - *Why*: preserves pane-map correlation properties (tmux_pane, tmux_server) across the idle-active transition, so `fab pane map` can show "active agent here" immediately without waiting for the next Stop event. Full delete would require reconstruction from env on next Stop.
   - *Rejected*: Full delete on user-prompt — wasteful since Stop will re-write the same properties; creates a brief window where the pane appears agentless.

6. **Platform-split grandparent walker under `internal/proc/`**:
   - *Why*: mirrors existing `internal/pane/pane_process_{linux,darwin}.go` convention; Go build tags handle selection; keeps platform-specific code isolated and testable.
   - *Rejected*: Shell-out to `ps` on both platforms — slower; `/proc` read is strictly cheaper on Linux and there's no reason not to use it.

7. **New memory file vs. appending to `pane-commands.md`**:
   - *Why*: `.fab-runtime.yaml` schema is a coherent standalone topic (write pipeline, GC, matching rule); `pane-commands.md` already documents four subcommands and is substantial. Separating avoids bloat and lets consumers cite the right doc.
   - *Rejected*: Append to `pane-commands.md` — topic overloading; future edits to either subject become harder to locate.

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Schema uses `_agents[session_id]` bucket + top-level `last_run_gc` | Confirmed from intake #1; user-directed across all design iterations; no alternative surfaced at spec stage | S:98 R:75 A:92 D:95 |
| 2 | Certain | session_id from hook stdin JSON is the identity key | Confirmed from intake #2; probe-verified payload availability; design doc references the hook transcript_path → session_id mapping | S:98 R:75 A:95 D:95 |
| 3 | Certain | tmux_server / tmux_pane / pid / change / transcript_path are optional entry properties | Confirmed from intake #3; user directive; spec Non-Goals explicitly excludes making these part of the key | S:98 R:75 A:92 D:95 |
| 4 | Certain | GC interval = 180 seconds via top-level `last_run_gc` field | Confirmed from intake #4; user's "once per 3 mins or so" translated to 180s; documented in spec §GC Requirement | S:98 R:88 A:92 D:92 |
| 5 | Certain | Hooks invoked as `claude → sh -c → hook`; grandparent walk required for Claude PID | Confirmed from intake #5; probe-verified on Linux for Stop + UserPromptSubmit events | S:98 R:82 A:95 D:95 |
| 6 | Certain | `$TMUX_PANE` and `$TMUX` are preserved through hook invocation env | Confirmed from intake #6; probe-verified on Linux; hooks populate `tmux_pane` and `tmux_server` from these | S:95 R:78 A:92 D:95 |
| 7 | Certain | Three orthogonal axes in pane map — Change / Agent / Process, resolved independently | Confirmed from intake #7; spec's pane-map resolution requirement codifies it | S:98 R:75 A:88 D:92 |
| 8 | Certain | `fab pane process` remains opt-in — no default merge into pane map | Confirmed from intake #8; spec Non-Goals explicitly excludes merge | S:95 R:88 A:88 D:92 |
| 9 | Certain | GC liveness via `kill(pid, 0)` sending signal 0 — ESRCH means entry prunable | Confirmed from intake #9; POSIX-standard; spec §GC Requirement specifies ESRCH-based pruning | S:92 R:88 A:95 D:95 |
| 10 | Certain | Grandparent walker under `internal/proc/` uses build-tag-split files (Linux /proc, macOS ps) | Confirmed from intake #10; mirrors existing `internal/pane/pane_process_{linux,darwin}.go`; spec requirement specifies the exact file structure | S:92 R:90 A:95 D:90 |
| 11 | Certain | user-prompt hook clears only `idle_since`, preserves other entry properties | Upgraded from intake #13 Confident — spec's §Hook write semantics codifies the exact field-level behavior; design decision #5 captures the rationale and rejected alternative | S:88 R:82 A:85 D:82 |
| 12 | Certain | `session_id` absent from stdin payload → hook exits 0 silently | Upgraded from intake #14 Confident — spec's §Hook stdin parsing scenario codifies it; matches existing swallow-on-error pattern in `hook.go:57-63` | S:88 R:88 A:85 D:88 |
| 13 | Certain | Clean-slate migration deletes old `.fab-runtime.yaml`; state re-populates on next hook | Only feasible strategy — old entries have no `session_id` (hooks didn't capture it before this change) and cannot be faithfully migrated; runtime state is ephemeral, so loss is bounded to one hook cycle. Per spec §Migration requirement and design decision #4. | S:88 R:90 A:88 D:90 |
| 14 | Confident | New memory file `fab-workflow/runtime-agents.md` + updates to `pane-commands.md`, `schemas.md`, index | Confirmed from intake #12; design decision #7 explains separation rationale; honest design choice (could arguably co-locate in pane-commands, but spec §Documentation codifies the split) | S:80 R:92 A:85 D:80 |
| 15 | Certain | `artifact-write` hook unchanged — it parses a different payload shape (file path) and doesn't participate in `_agents` writes | Explicit scope exclusion; spec §Hook stdin parsing separates artifact-write from the three session-scoped hooks; no design ambiguity | S:92 R:90 A:92 D:92 |
| 16 | Certain | Pane-server disambiguation uses basename of `$TMUX` socket path (first comma-separated component) | Tmux's own convention; the `--server -L` flag accepts the same basename form (documented in `pane-commands.md`); no alternative interpretation | S:90 R:85 A:92 D:90 |
| 17 | Certain | Cache discipline: `.fab-runtime.yaml` read once per worktree per pane-map invocation | Preserves existing `ResolveAgentStateWithCache` behavior from `pane.go:252`; no new decision — just carrying forward the established pattern | S:92 R:90 A:95 D:92 |
| 18 | Tentative | macOS preserves `$TMUX_PANE`/`$TMUX` identically to Linux; `sh -c` wrapper behavior matches | Confirmed from intake #15; probed Linux only — tmux + sh are cross-platform and Claude's hook invocation is platform-independent but empirically unverified | S:55 R:80 A:70 D:75 | <!-- assumed: macOS matches Linux hook env behavior — CI test on darwin recommended before ship -->
| 19 | Confident | GC entries without `pid` field are preserved indefinitely (never pruned) | Spec §GC Requirement explicitly skips pid-less entries. Edge case: non-tmux hooks where grandparent walker failed could leave pid-less entries. Low frequency; addressable via a secondary mtime-based GC if we observe growth. | S:70 R:82 A:80 D:75 | <!-- assumed: pid-less entry bloat is bounded and self-limiting; revisit if we observe growth -->

19 assumptions (16 certain, 2 confident, 1 tentative, 0 unresolved).
