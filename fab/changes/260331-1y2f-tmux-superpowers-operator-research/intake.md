# Intake: Tmux Superpowers — Operator Research

**Change**: 260331-1y2f-tmux-superpowers-operator-research
**Created**: 2026-03-31
**Status**: Draft

## Origin

> Research: investigate tmux superpowers for the operator. Two areas to research online: (1) tmux wait-for — can we use it to block until an agent pane finishes or reaches a state, replacing polling? (2) tmux MCP — is there a tmux MCP server that could give agents direct tmux control? Search online for both topics. Then read _cli-external.md and fab-operator7.md to understand current tmux usage. Produce an intake with concrete recommendations on what we could absorb into _cli-external.md or into the operator skill itself.

One-shot research request, evolved through discussion into a concrete implementation direction: internalize the most valuable tmux capabilities (from both `wait-for` research and the MCP ecosystem) directly into the `fab` Go binary as a `fab pane` command group.

## Why

The operator currently uses a simple polling loop (`/loop 3m`) plus raw `tmux capture-pane`, `tmux send-keys`, and `tmux new-window` for all agent coordination. This works but has limitations:

1. **No safety net on send-keys** — the operator skill manually validates pane existence and agent idle state (§3) before every `tmux send-keys`. This is repeated logic that should be a single validated command.

2. **Heuristic question detection** — §5 captures 20 lines of terminal output and regex-matches for question patterns. This is fragile — false positives on code output, missed prompts from unusual formatting. OS-level process state detection (is the foreground process blocked on read?) is a much stronger signal.

3. **Raw tmux output** — `tmux capture-pane -p` returns unstructured text. `tmux list-panes` requires format string parsing. The operator has to work harder to extract structured data.

4. **Existing precedent** — `fab pane-map` already wraps tmux queries with fab-aware context (change resolution, stage lookup, agent state). Extending this pattern to capture, send, and process detection is natural.

## What Changes

### Research Findings (Background)

Two areas were investigated. Key conclusions from the discussion:

#### `tmux wait-for` — Not Directly Useful for the Operator

`tmux wait-for` is a blocking shell primitive — it blocks the calling shell until a named channel is signaled. The operator is an LLM agent that works via tool calls and `/loop` ticks, not a shell that can block. All four `wait-for` recommendations (blocking scripts, CLI signaling, mutex patterns, hook-based detection) were evaluated and **dropped** because they solve problems for shell-scripted coordinators, not LLM-driven ones.

The `wait-for` research remains valuable as reference material (see Research Archive below) and the process-state detection idea it inspired led to the `fab pane process` recommendation.

#### Tmux MCP Servers — Valuable Capabilities, Wrong Delivery Mechanism

The tmux MCP ecosystem has 10+ projects. The most interesting is `MadAppGang/tmux-mcp` (Go, 20 tools, uses `wait-for` internally, native OS process detection). However, adding an external MCP server:
- Violates Constitution §V (Portability) if made a hard dependency
- Adds an external runtime dependency (violates §I — Pure Prompt Play)
- Creates conditional code paths in the operator ("if MCP available, use MCP tool; else raw tmux")

The better approach: **internalize the valuable capabilities directly into `fab-go`**.

### Recommendation: `fab pane` Command Group

Rename `fab pane-map` to `fab pane map` and add three new subcommands. All live in the existing `fab-go` binary — no new binaries, no external dependencies.

#### `fab pane map` (rename from `pane-map`)

Existing command, moved under the `pane` group for consistency. All current flags and output formats preserved (`--json`, `--session`, `--all-sessions`).

#### `fab pane capture <pane> [-l N] [--json]`

Structured pane content capture.

```
fab/.kit/bin/fab pane capture %3 -l 20
fab/.kit/bin/fab pane capture %3 -l 20 --json
```

**Default output**: raw text (same as `tmux capture-pane -t %3 -p -l 20` today).

**JSON output** (`--json`):
```json
{
  "pane": "%3",
  "lines": 20,
  "content": "...",
  "change": "260331-r3m7-add-retry-logic",
  "stage": "apply",
  "agent_state": "idle"
}
```

Enriches raw capture with fab context (change, stage, agent state) so the operator doesn't need a separate `pane-map` call to correlate.

**Why not just `tmux capture-pane`**: the operator currently runs `fab pane-map` to find which pane has which change, then `tmux capture-pane` on each idle pane. With `fab pane capture --json`, one call gives both the content and the context.

#### `fab pane send <pane> <text> [--force]`

Safe send-keys with built-in validation.

```
fab/.kit/bin/fab pane send %3 "/fab-continue"
fab/.kit/bin/fab pane send %3 "y" --force
```

**Default behavior** (no `--force`):
1. Verify pane exists — if gone, exit 1: `"pane %3 not found"`
2. Check agent is idle (via `.fab-runtime.yaml`) — if active, exit 1: `"agent in %3 is active, use --force to override"`
3. Send the text + Enter via `tmux send-keys`
4. Exit 0

**`--force`**: skip the idle check (still validates pane exists).

**Why this matters**: the operator skill's §3 Pre-Send Validation currently has 4 manual steps before every `tmux send-keys`. Steps 1 and 2 (pane exists, agent idle) become a single `fab pane send` call. Steps 3–4 (change active, branch aligned) remain operator-level logic since they require pipeline awareness.

#### `fab pane process <pane> [--json]`

OS-level process state detection for the pane's foreground process.

```
fab/.kit/bin/fab pane process %3
fab/.kit/bin/fab pane process %3 --json
```

**Implementation**: resolve the pane's foreground PID via `tmux display-message -t %3 -p '#{pane_pid}'`, then read `/proc/{pid}/stat` (or walk the process tree to find the actual foreground process in the pane's process group). Detect:

- **`running`** — process is actively executing (R state in `/proc`)
- **`waiting-for-input`** — foreground process is blocked on read (`S` state + `wchan` is `read`/`wait_for_input`/`poll_schedule_timeout` on a tty fd)
- **`sleeping`** — process is sleeping but not on tty read (e.g., `sleep`, network I/O)
- **`stopped`** — process is stopped (SIGSTOP/SIGTSTP)
- **`exited`** — no foreground process (pane shell is the foreground)

**Default output**: single word (`running`, `waiting-for-input`, `sleeping`, `stopped`, `exited`).

**JSON output**:
```json
{
  "pane": "%3",
  "pid": 12345,
  "state": "waiting-for-input",
  "process_name": "claude",
  "change": "260331-r3m7-add-retry-logic"
}
```

**How this improves question detection**: the operator's §5 currently captures terminal output and regex-matches for question indicators. With `fab pane process`:

1. `fab pane map` — find agents marked idle
2. `fab pane process %3` — if `waiting-for-input`, the agent is genuinely waiting for user input
3. Only then `fab pane capture %3 -l 20` — regex-match to determine *what* to answer

Step 2 eliminates false positives (agent outputting text that looks like a question but isn't waiting). It's a strong pre-filter that makes the heuristic capture-and-match step more reliable.

**Platform note**: `/proc` parsing is Linux-only. On macOS, equivalent info comes from `ps -o stat= -p {pid}` and `lsof`. The Go implementation should support both.

### Summary of Commands

| Command | Purpose | Replaces |
|---------|---------|----------|
| `fab pane map` | Pane-to-change mapping with pipeline state | `fab pane-map` (renamed) |
| `fab pane capture <pane>` | Structured pane content capture with fab context | `tmux capture-pane -t <pane> -p` |
| `fab pane send <pane> <text>` | Safe send-keys with built-in idle/existence checks | `tmux send-keys -t <pane>` + manual §3 validation |
| `fab pane process <pane>` | OS-level process state detection | Heuristic regex question detection (§5 pre-filter) |

### Impact on Operator Skill

After `fab pane` is implemented, `fab-operator7.md` changes:

- **§3 Pre-Send Validation**: steps 1–2 collapse into `fab pane send` (pane exists + idle check are built in). Steps 3–4 (change active, branch aligned) remain.
- **§5 Question Detection**: gains a `fab pane process` pre-filter step before capture-and-regex. Reduces false positives.
- **`_cli-external.md` tmux section**: raw `tmux capture-pane` and `tmux send-keys` entries replaced by `fab pane capture` and `fab pane send`. `tmux new-window` remains (no `fab pane` equivalent needed — spawning is a one-shot operation with complex argument construction).

### Research Archive

The full `tmux wait-for` and tmux MCP ecosystem research is preserved below for future reference.

<details>
<summary>tmux wait-for deep dive</summary>

`tmux wait-for [-L|-S|-U] <channel>` — named-channel signaling between tmux clients and shell scripts.

| Flag | Mode | Behavior |
|------|------|----------|
| *(none)* | Wait | Blocks the calling tmux client until the channel is signaled |
| `-S` | Signal | Wakes **all** clients currently waiting on that channel |
| `-L` | Lock | Acquires a mutex on the channel; blocks if already locked (FIFO queue) |
| `-U` | Unlock | Releases the mutex; wakes the next queued locker |

**Internal model** (from `cmd-wait-for.c`): Each channel is a struct in a red-black tree. Key behaviors:
- Signal wakes ALL waiters (iterates the entire queue), not just one
- **Signal-before-wait is safe**: if `-S` fires before any waiter, a `woken` flag is set — a subsequent `wait-for` returns immediately (prevents the classic race condition)
- After signaling, the channel is **destroyed** (one-shot). For repeated use, need a new channel name.
- Lock mode is NOT one-shot — the channel persists and can be locked/unlocked repeatedly.

**Canonical pattern**:
```bash
tmux send-keys -t %3 'my-command; tmux wait-for -S cmd-done' Enter
tmux wait-for cmd-done
```

**Limitations**: no timeout support (blocks indefinitely, tmux issue #832), one-shot channels, global namespace (must namespace channel names), no passive observation (can't wait for pane output to match regex), trap handlers don't fire during wait-for.

**Why dropped for the operator**: `wait-for` is a blocking primitive. The operator is an LLM agent that works via tool calls and `/loop` ticks — it can't block. Helper scripts that block don't help because the operator would still need to poll to check if the helper finished. The process-state detection idea was absorbed into `fab pane process` instead.

</details>

<details>
<summary>tmux MCP ecosystem survey</summary>

10+ projects exist across npm, PyPI, and standalone Go binaries.

| Project | Lang | Stars | Tools | Key Differentiator |
|---------|------|-------|-------|--------------------|
| `nickgnd/tmux-mcp` | JS/TS | 248 | 13 | Pioneer, battle-tested, npm installable |
| `MadAppGang/tmux-mcp` | Go | 18 | 20 | Synchronous execution via `wait-for`, native process detection |
| `PsychArch/tmux-mcp-tools` | Python | 7 | 5 | Lean, uvx installable |
| `memextech/ht-mcp` | Rust | 211 | 6 | Headless terminal (not tmux), self-contained |
| `Martian-Engineering/maniple` | Python | 39 | — | Higher-level agent orchestration via tmux |

`MadAppGang/tmux-mcp` was the most architecturally interesting — Go binary (aligns with Constitution §I), uses `wait-for` internally for synchronous command execution, native OS process detection via `/proc`. Its key capabilities (structured capture, safe send, process detection) were absorbed into the `fab pane` command group recommendation rather than taken as an external dependency.

**Why not adopt an MCP server**: adds external runtime dependency (violates §I), creates conditional operator code paths, and the most valuable capabilities can be internalized into `fab-go` with tighter integration to fab state (change resolution, agent idle detection).

</details>

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document `fab pane` command group and its impact on operator §3/§5

## Impact

- `fab/.kit/bin/fab-go` — new `pane` command group with `map`, `capture`, `send`, `process` subcommands
- `fab/.kit/skills/_cli-fab.md` — document the new `fab pane` commands (per constitution: CLI changes MUST update `_cli-fab.md`)
- `fab/.kit/skills/_cli-external.md` — tmux section shrinks (raw commands replaced by `fab pane` equivalents, only `tmux new-window` remains)
- `fab/.kit/skills/fab-operator7.md` — §3 Pre-Send Validation simplifies, §5 Question Detection gains process-state pre-filter

## Open Questions

None — both original questions (process tree walking, macOS `/proc` equivalent) were resolved as implementation details:
- **Process tree**: walk to the foreground process group leader via the pane's tty (solved problem, same approach as MadAppGang's tmux-mcp)
- **macOS**: `ps -o stat=` for coarse state + `lsof` for tty-read detection; `/proc/stat` + `/proc/wchan` on Linux. Platform abstraction in Go.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Internalize into `fab-go` rather than adopt external MCP server | Constitution §I (no runtime frameworks) and §V (portability) — discussed and confirmed by user | S:95 R:90 A:95 D:95 |
| 2 | Certain | `wait-for` recommendations (R1–R4) dropped — not useful for LLM-driven operator | Discussed — blocking primitives don't help agents that work via tool calls and polling ticks | S:95 R:95 A:90 D:95 |
| 3 | Certain | `fab pane-map` renamed to `fab pane map`, no alias for old name | User explicitly confirmed: "No need of pane-map alias" | S:95 R:85 A:95 D:95 |
| 4 | Certain | All pane commands live in existing `fab-go` binary, no separate binary | Discussed — pane commands need shared plumbing (runtime state, change resolution) that lives in fab-go | S:95 R:90 A:95 D:90 |
| 5 | Confident | `fab pane send` defaults to safe mode (idle + exists check), `--force` to skip idle check | Mirrors operator §3 behavior — safe by default is the obvious design | S:75 R:85 A:80 D:75 |
| 6 | Confident | `fab pane process` uses `/proc` on Linux, `ps`/`lsof` on macOS | Standard approach for process introspection — no other viable method | S:70 R:80 A:85 D:80 |
| 7 | Confident | `fab pane process` can reliably distinguish "waiting for input" from other sleep states | Clarified — binary distinction (foreground process on tty read vs not) is straightforward; edge case failure degrades gracefully to existing capture-and-regex fallback | S:70 R:85 A:70 D:70 |
| 8 | Confident | `tmux new-window` stays as raw tmux (no `fab pane` equivalent needed) | Clarified — tmux call is the last step of a multi-step spawn sequence; wrapping it moves complexity without reducing it. A `fab pane spawn` would need to absorb the entire sequence (worktree + deps + tmux), which is a different change | S:75 R:80 A:75 D:70 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
