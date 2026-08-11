---
type: memory
description: "`fab pane {map,capture,send,process,window-name,open,ready,deliver,kill,await}` reference: provider-generic `open --provider` spawn + `ready`/`deliver` primitives with `--json`, generic `kill` + blocking `await` (idle/file/running/gone), `send`'s warn-and-proceed unknown-state posture, `--server`/`-L`, pane-family exit-code scheme (2 = pane missing / 3 = other tmux failure), shared `internal/pane` helpers + `display-message` validation probe, pane-ID-per-server semantics, window-name rewrites."
---
# Pane Commands

**Domain**: runtime

## Overview

`fab pane` is the parent command grouping ten tmux-pane operations. Six query or manipulate existing panes: `map`, `capture`, `send`, and `process` shell out to `tmux` to query or manipulate panes, combining raw tmux output with fab-specific enrichment (worktree, change, stage, agent state resolved from per-pane CWD), `window-name` is a primitive set for idempotent / guarded rewrites of the tmux window name — used by `/fab-operator` to mark enrolled and done-monitoring windows — and `kill` is the generic pane-removal verb. The other three — `open`, `ready`, `deliver` — are the provider-generic spawn / readiness-probe / verified-delivery primitives, addressed by pane id (or provider name) with no fab context required; `fab dispatch open`/`ready`/`deliver` are thin record-keeping bindings over them (see [dispatch.md](/runtime/dispatch.md)). `await` is the record-free blocking wait over the pane's completion signals (idle state, a contract file).

The command group runs from any directory — including outside a fab-managed repo (scratch tmux tabs, cross-repo orchestration, non-fab daemons). The router routes every non-fab-kit command to `fab-go` regardless of `config.yaml` presence, and `pane` subcommands carry no `resolve.FabRoot()` guard because they resolve state from target pane IDs rather than from the invoker's CWD. See `kit-architecture.md` for the router's always-route policy.

This doc covers the ten subcommands, the `--server` / `-L` persistent flag, and the semantic invariants that govern how pane IDs and server selection interact with tmux's own socket model.

## Requirements

### Parent Command: `fab pane`

`fab pane` is a cobra command group with ten subcommands (`map`, `capture`, `send`, `process`, `window-name`, `open`, `ready`, `deliver`, `kill`, `await`) and one persistent flag (`--server` / `-L`). Invoking `fab pane` with no subcommand prints the standard cobra help listing the ten subcommands. Source: `src/go/fab/cmd/fab/pane.go`.

### Subcommand: `fab pane map`

`fab pane map [--json] [--session <name>] [--all-sessions]` combines tmux pane introspection with worktree/change/runtime state into a unified view. Source: `src/go/fab/cmd/fab/panemap.go`.

**Flags**:

| Flag | Type | Purpose |
|------|------|---------|
| `--json` | bool | Output as JSON array instead of aligned table |
| `--session <name>` | string | Target a specific tmux session by name (skips `$TMUX` check) |
| `--all-sessions` | bool | Query all tmux sessions (skips `$TMUX` check) |
| `--server <name>` | string | Persistent flag — see §`--server` flag below |

`--session` and `--all-sessions` are mutually exclusive. When neither is set, discovery runs against the current tmux session only (`tmux list-panes -s`) and requires `$TMUX` to be set.

**Table columns**: `Session` (only with `--all-sessions`), `Pane`, `WinIdx`, `Tab`, `Worktree`, `Change`, `Stage`, `Agent`. The `Worktree` column displays `(main)` for the main worktree, a relative path from the main repo's parent for other git worktrees, or `basename/` for non-git panes. Non-fab panes render em-dash fallbacks for `Change` and `Stage`. The `Agent` column is **not** a non-fab em-dash case: it resolves from the pane's `@rk_agent_state` option for **every** pane class (including non-git / non-fab panes), so it renders em-dash only when that option is absent or unparseable — not because the pane lacks fab context (see **Agent state resolution** below). The relative `Worktree` path is computed **per repo** — each pane's display path is relative to *its own* repo's main-worktree root, so panes from different repos render correct paths (no human-table `Repo` column is added).

**JSON fields** (snake_case): `session`, `window_index`, `window_id`, `pane`, `tab`, `worktree`, `repo`, `change`, `stage`, `display_state`, `agent_state`, `agent_idle_duration`, `pr_url`, `pr_number`. `repo` is the absolute main-worktree root for the pane's repo, `null` when unresolved (non-git pane); it is exposed in `--json` ONLY (no human-table column) so programmatic callers can group rows by repo without re-deriving. `change` and `stage` are `null` when no active change exists on the pane's worktree. `agent_state` and `agent_idle_duration` populate from the pane's `@rk_agent_state` option — independent of `change` / `stage`. `agent_state` ∈ `{active, waiting, idle, null}` (`waiting` (ioku); `null` when the option is absent/unparseable); `agent_idle_duration` is populated **only** for `idle` (null for `active`/`waiting`/unknown). The JSON is emitted from the **structured** state/duration pair (`pane.AgentDisplayFromOption`), not by re-parsing the human display string, so a display-format tweak cannot silently break the contract run-kit consumes.

`display_state` (`*string`, placed immediately after `Stage` in `paneJSON`) is the state half of `status.DisplayStage` (the `stage` field is the name half). Values: `active`, `ready`, `done`, `failed`, `pending`, `skipped`, or `null`. It is `null` exactly when `stage` is `null` (no resolvable change, no `fab/` dir, or an unloadable `.status.yaml`) — the same em-dash-sentinel → `toNullable()` nullability contract as `repo`/`change`/`stage`. Exposed in `--json` ONLY — no human-table column; the table output is byte-identical to the pre-field rendering. **Why**: the stage name alone cannot distinguish an actively-worked stage from a parked finished change — a fully-shipped change renders `"stage": "review-pr"` indefinitely until archived, byte-identical to a change whose review-pr is actively running; the pair (`review-pr`, `done`) vs (`review-pr`, `active`) disambiguates. **Additive shape**: existing JSON consumers ignore unknown keys, matching the `repo` (h3jk) and `pr_url`/`pr_number` (r7ju) precedent. **Consumer**: the run-kit sidebar (`app/backend/internal/sessions/sessions.go` `paneMapEntry`) opts in separately in its own repo — it gains honest per-row attention states (`failed`/`ready` = needs human, `done` + parked = quiet row) instead of heuristics over `agent_state` / `.fab-runtime.yaml` `idle_since`. See [change-lifecycle.md](/pipeline/change-lifecycle.md) "Deriving display stage" for the tier walk that produces the value (including the `failed` tier added alongside this field).

`window_id` (`*string`, placed immediately after `window_index` in `paneJSON`) is the tmux `#{window_id}` (`@N`) — the server-assigned window identifier that is stable for a window's lifetime and travels with the window across `swap-window`/`move-window`. It is `null` only for a legacy/absent input line (a pre-change tmux format string, or a shorter line that omits the field) — a live window always has one, independent of git/fab context (like `agent_state`, it is resolved for every pane class, including non-git rows). It is emitted via the same em-dash/empty-sentinel → `toNullable()` nullability contract as `repo`/`display_state`/`pr_url`, and exposed in `--json` ONLY — no human-table column (the table keeps only `WinIdx`), so table output is byte-identical to the pre-field rendering. **Why**: the positional `(session, window_index)` pair is *not* a stable window identity — a `tmux swap-window` reorder reassigns indexes, so a consumer that joins pane-map rows to its own tmux snapshot by index misattributes one window's fab status to whichever window now occupies that index. `window_id` gives consumers a stable join key that survives reorders, removing the positional coupling. **Consumer**: run-kit's sidebar join saw exactly this bug — it caches the pane map for 5s, so after a swap-window the stale index join lagged the reorder for up to the cache TTL (a sidebar status dot attached to the wrong window); joining on `window_id` fixes it for that and every other index-joining consumer. **Additive shape**: existing JSON consumers ignore unknown keys, matching the `repo` (h3jk), `display_state`, and `pr_url`/`pr_number` (r7ju) `--json`-only precedent; `window_index` is retained unchanged (backward compat). *Introduced by 260713-ueuy.*

`pr_url` (`*string`) is the LAST entry of the pane's active change `.status.yaml` `prs:` list (most recent PR), sourced from the SAME `sf.Load(statusPath)` already performed for `stage` derivation — no second read. It is `null` when the `prs:` list is absent or empty, or when the pane has no fab change / no `fab/` dir / unresolved git. `pr_number` (`*int`) is parsed from the URL's trailing `/pull/<n>` segment (via the `parsePRNumber` helper); it is `null` when there is no URL or the URL is unparseable — a malformed URL keeps `pr_url` set but yields `pr_number: null`. Like `repo`, both fields are exposed in `--json` ONLY — no human-table column is added. **Deliberate non-goal**: fab surfaces only the on-disk URL/number written by `/git-pr`; PR *status* (open/merged/closed, CI state) is NOT fab's job and there is no network / `gh` / `git` call — run-kit fetches live status separately. The `toNullable` helper now also nil-maps `""` (alongside the em-dash and `"(no change)"` sentinels), so an empty `prURL` cleanly maps to JSON `null`; the pre-existing `repo` / `change` / `stage` callers are normalized to the em-dash sentinel and never pass `""`, so the new `""` branch only fires for `pr_url`.

**Per-repo `mainRoot` resolution**: `runPaneMap` computes the main-worktree root **per distinct repo**, cached by the pane's `GitWorktreeRoot` via the `mainRootForPane(cwd, wtRoot, cache)` helper (one `git worktree list` lookup per repo, reused across that repo's panes). Each pane is resolved against its own repo's `mainRoot`, fixing the prior bug where one `mainRoot` derived from the first parsable pane was applied to every row — producing garbage relative paths for panes in other repos. `paneRow` carries a `repo` field set to that absolute root (em dash when unresolved); `paneJSON.Repo` is a nullable `*string` emitted via `toNullable`, matching the existing `change`/`stage` nullable-field pattern. Source: `src/go/fab/cmd/fab/panemap.go`.

**Subprocess economy (hot path — the operator's per-tick snapshot)**: each pane's git worktree root is resolved at most **once per distinct pane cwd** via the cwd-keyed `worktreeRootForPane(cwd, cache)` cache, with `""` as the cached non-git sentinel that `resolvePane`'s non-git branch keys off; the resolved root is threaded into both `mainRootForPane` and `resolvePane` (one `git rev-parse --show-toplevel` per pane cwd, not two). `--all-sessions` discovery is a **single `tmux list-panes -a -F <tmuxPaneFormat>`** call (the format carries `#{session_name}`) instead of `list-sessions` plus one `list-panes -s -t <session>` per session — `-a` enumeration also side-steps the prefix/glob target resolution of `-t <session>`.

**Three-axis model**: The map resolves three orthogonal axes independently — **Change** (from `.fab-status.yaml`), **Agent** (from the `@rk_agent_state` tmux pane option), and **Process** (opt-in via `fab pane process`, not in `map` output). See [runtime-agents.md](/runtime/runtime-agents.md) for the full model.

**Agent state resolution**: The Agent column reads the `@rk_agent_state` pane option — carried on the **existing** `list-panes -F` format string as the 6th tab-delimited field `#{@rk_agent_state}` (a possibly-empty MIDDLE field since 260713-ueuy appended `#{window_id}` as the never-empty 7th/trailing field — see `window_id` above), so agent state costs **zero extra subprocesses**, and the tmux-server disambiguation problem does not arise because a pane option lives on exactly one server's pane. The raw option value is parsed by the shared `internal/pane` helper (see [runtime-agents.md](/runtime/runtime-agents.md)): `active` → `active`; `waiting` → `waiting`; a well-formed `idle:<epoch>` → `idle (<duration>)` (duration from the epoch via `FormatIdleDuration`); absent/unparseable/unknown-token → `—`. This resolution is **independent of whether the pane has an active change**, and it is computed for **every** pane class (including non-git rows), so `map` agrees with `send`/`capture` on non-fab panes. See [runtime-agents.md](/runtime/runtime-agents.md) for the read contract, the three states + unknown, and epoch-derived duration.

**Display scenarios**:

| Scenario | Change | Stage | Agent |
|----------|--------|-------|-------|
| Change active, agent idle | `260417-...` | `apply` | `idle (2m)` |
| Change active, agent active | `260417-...` | `apply` | `active` |
| Change active, agent waiting on a human | `260417-...` | `apply` | `waiting` |
| Discussion mode (fab worktree), agent idle | `(no change)` | `—` | `idle (2m)` |
| Change active, no instrumented agent | `260417-...` | `apply` | `—` |
| Fab worktree, no change, no agent | `(no change)` | `—` | `—` |
| Non-fab pane (no `fab/` dir) | `—` | `—` | `—` |

**Error behavior**: Unset `$TMUX` with neither session flag → `ERROR: not inside a tmux session` (exit 1, returned through RunE to main's single formatter). No panes found → `No tmux panes found.` (exit 0).

### Subcommand: `fab pane capture`

`fab pane capture <pane> [-l N] [--json] [--raw]` captures terminal content from a tmux pane with fab context enrichment. Source: `src/go/fab/cmd/fab/pane_capture.go`.

**Flags**: `<pane>` (required tmux pane ID, e.g. `%5`); `-l`/`--lines` (int, default 50); `--json` (structured output with pane metadata); `--raw` (plain captured text, no header, no enrichment). `--json` and `--raw` are mutually exclusive.

**Default output**: Header block (pane ID, worktree, change, stage, agent state) followed by the captured text. The agent line reads `@rk_agent_state` (same read as `send`) and shows `agent: <state>` — e.g. `agent: waiting`, or `agent: idle (<dur>)` for a completed turn.

**JSON output**: `pane`, `lines`, `content`, `worktree`, `change`, `stage`, `agent_state`, `agent_idle_duration` — `agent_state` ∈ `{active, waiting, idle, null}` (same structured-source semantics as `map --json`; `waiting` (ioku)), `agent_idle_duration` only for `idle`. The `worktree`/`change`/`stage` fields are `null` when the pane is not in a fab worktree or has no active change.

**Error behavior**: Pane not found → `Error: pane <id> not found` (**exit 2**); any other tmux validation failure (dead server, bad socket) → **exit 3** — the same scheme as `window-name`, so operator scripts can branch on cause uniformly across the pane family (260612-ye8r). `--lines < 1` → `ERROR: --lines must be >= 1` (exit 1, via RunE). Pane existence is validated via the targeted `display-message` probe (see §Shared Pane Package `ValidatePane`); a failed `tmux capture-pane` surfaces the child's trimmed stderr alongside the pane ID. `--raw` output is byte-identical to tmux's stdout (never trimmed).

### Subcommand: `fab pane send`

`fab pane send <pane> <text> [--no-enter] [--force]` sends keystrokes to a tmux pane with built-in pane-existence and agent-idle validation. Source: `src/go/fab/cmd/fab/pane_send.go`.

**Flags**: `<pane>` (required); `<text>` (required); `--no-enter` (don't append Enter); `--force` (skip idle validation — still validates pane existence).

**Validation pipeline**:

1. Pane exists: a single targeted probe — `tmux display-message -t <pane> -p '#{pane_id}'`, output compared to the argument for ID-exactness (see §Shared Pane Package `ValidatePane`). If not found → **exit 2** with `Error: pane <id> not found` (even with `--force`); any other tmux validation failure → **exit 3** — the `window-name` scheme, unified across the family in 260612-ye8r.
2. Agent idle (three-state gate) (ioku): reads `@rk_agent_state` and gates on the resolved state. `idle` → send. `active`/`waiting` → refuse with `ERROR: agent in pane <id> is not idle (state: <state>)` (exit 1, returned through RunE — three-state-aware, the state name appears in the message). Absent/unparseable (**unknown**) → warn and send anyway: `warning: agent state unknown — sending anyway` on stderr, exit 0 — an unknown state carries nothing to gate on, so refusal is reserved for a *parseable* non-idle state. `--force` bypasses **only** this check (pane existence in step 1 is still enforced, so a missing pane exits 2 even with `--force`). The gate is a pure decision helper (`idleGate`), unit-tested for all five cases + both pinned message contracts, with an integration test driving the full command against a real tmux server via the `tmux set-option -p` writer simulation.
3. Send keys: `tmux send-keys -t <pane> -l <text>` (literal text), optionally followed by a separate `tmux send-keys -t <pane> Enter`. A failed send surfaces tmux's trimmed stderr and names the pane (e.g. `tmux send-keys to %5: exit status 1: can't find pane: %5`).

**Why two send-keys invocations**: The `-l` flag sends `<text>` literally so tmux does not interpret key names like `"Enter"`, `"Space"`, `"C-c"` embedded in the text itself. The trailing Enter keystroke is sent as a separate non-literal command.

**Unknown state**: A pane with no `@rk_agent_state` option — or a value with an unknown token or a missing/bad epoch — resolves to **unknown** and sends with the stderr warning above (no `--force` needed). The convention's only writer is run-kit's `rk agent-setup` harness hooks (Claude, codex, copilot, gemini, opencode), so a foreign-agent or uninstrumented pane always reads unknown — refusing there would force `--force` on precisely the multi-provider panes the gate exists for. An instrumented agent that has flipped to `idle` is accepted without ceremony; `active`/`waiting` refuse with the three-state-aware message. `--force` remains the skip-everything override. See [runtime-agents.md](/runtime/runtime-agents.md) for the read contract.

### Subcommand: `fab pane process`

`fab pane process <pane> [--json]` detects the process tree running in a tmux pane via OS-level process inspection. Source: `src/go/fab/cmd/fab/pane_process.go` (plus platform-specific `pane_process_linux.go` / `pane_process_darwin.go`).

**Discovery**: Linux reads `/proc/<pid>/task/<tid>/children` recursively; macOS uses `ps -o pid,ppid,comm -ax` with PPID traversal, plus ONE batched `ps -axo pid=,args=` pass parsed into a PID→args map (pure `parsePSCmdlines` parser: pid is numeric-first, remainder is args — robust against comm-with-spaces) and joined by PID for full cmdlines — exactly two `ps` spawns total, no per-node lookups. A process exiting between the two passes degrades to cmdline `""` (the same value as a per-PID failure). Platform selection via Go build tags.

**Classification** (based on process comm name): `claude`/`claude-code` → `agent`; `node` → `node`; `git`/`gh` → `git`; all others → `other`.

**Default output**: Tree-formatted process listing with PID, command name, and classification.

**JSON output**: `pane`, `pane_pid`, `processes` (tree of `{pid, ppid, comm, cmdline, classification, children}`), `has_agent` (true if any process classified as `agent`).

Platform-specific process discovery is tmux-server-independent — once the pane's shell PID has been resolved via `GetPanePID`, the `/proc` walk or `ps` traversal operates on the OS process table, not tmux.

**Error behavior**: Pane not found → `Error: pane <id> not found` (**exit 2**, in-handler via the typed `*PaneNotFoundError` branch); any other tmux validation failure (dead server, bad socket) → **exit 3** — the pane-family scheme, shared with `capture`/`send`/`ready`/`deliver`.

### Subcommand: `fab pane window-name`

`fab pane window-name` is a cobra subgroup with two verbs (`ensure-prefix`, `replace-prefix`) that perform guarded rewrites of the tmux window name. Both verbs read the current name via `tmux display-message -p -t <pane> '#W'`, compare it against a literal prefix, and conditionally call `tmux rename-window`. Both honor the parent `--server` / `-L` flag via the existing `WithServer` argv builder. Source: `src/go/fab/cmd/fab/pane_window_name.go`.

**Motivating use case**: `/fab-operator` enrolls monitored windows with a `»` (U+00BB) prefix and transitions them to `›` (U+203A) on removal to keep the tmux tab bar an honest at-a-glance map of active vs. done monitoring.
#### Verb: `ensure-prefix <pane> <char>`

Idempotent prepend. Reads the current name; if it begins with the literal string `<char>`, no-ops. Otherwise runs `tmux rename-window -t <pane> "<char><current-name>"`. Exits 0 on both rename and no-op, with stdout `renamed: <old> -> <new>` on rename and empty stdout on no-op.

`<char>` is any non-empty string — no width / BMP / codepoint validation is performed. The caller owns the single-width convention (the operator skill enforces it via its choice of `»` and `›`).

#### Verb: `replace-prefix <pane> <from> <to>`

Atomic guarded swap. Reads the current name; if it begins with the literal string `<from>`, runs `tmux rename-window -t <pane> "<to><name-without-from-prefix>"`. Otherwise, no-ops with exit 0 — this is the user-retitle-mid-monitoring guard: if the user retitled the window so it does not start with `<from>`, the swap is silently skipped.

`<to>` MAY be empty, in which case the `<from>` prefix is stripped (removal). `<from>` MUST be non-empty; an empty `<from>` exits 3 with a usage message on stderr.

#### Exit codes (both verbs)

| Exit | Meaning |
|------|---------|
| 0 | Rename succeeded OR operation was a no-op |
| 2 | Pane does not exist — tmux stderr contains `can't find pane` (or `no such pane`). Stderr is propagated to the caller. |
| 3 | Any other tmux error: tmux not running / socket unreachable / rename failed / argument usage error (e.g., empty `<char>` or `<from>`). Stderr is propagated when tmux supplied it. |

The primitives do not gate on `$TMUX`; they rely on tmux's own exec failure to surface "tmux not running" as exit 3, which lets callers run them via `--server` targeting outside a tmux client. The distinct 2 vs. 3 split lets `/fab-operator`'s removal path discriminate "pane gone" (exit 2 → treat as successful removal, window is gone anyway) from "pane alive but rename failed" (exit 3 → log warning and continue). Stderr mapping uses case-insensitive substring matching.

### Subcommand: `fab pane open`

`fab pane open --provider <name> [--role <role>] [-c <dir>] [--json]` resolves the named provider's `interactive_command` — project config per-field merged over the built-in table, exactly as `fab agent` resolves it; outside a fab repo the built-in table alone applies — substitutes the `{model}`/`{effort}` fills via the standard precedence with the **provider pinned at invocation time** (`agent.ResolveRoleWith` with the provider override; `--role` selects whose fills apply, the `default` role otherwise — the opposite of `fab agent --provider`'s deliberate fill bypass), and spawns the composed command in a tmux pane. It writes **no dispatch record and no `.fab-dispatch/` state**. Source: `src/go/fab/cmd/fab/pane_open.go`.

**Spawn shape**: a **plain split** of the current window when the invoker is itself a tmux pane on the target server (`$TMUX_PANE` set, no `--server`), an **unnamed new window** otherwise — no size, no title, no worker-column placement (placement and the `fab-{id}-{stage}` identity are dispatch policy — see [dispatch.md](/runtime/dispatch.md)).

**Output**: `opened pane %N (provider <name>)`, plus a `server: <name>` line when the socket is non-default. `--json` instead emits a single JSON object `{"pane": "%N", "provider": "<name>", "server": null}` — `server` is `null` for the default socket via the `toNullable` contract. Plain-text output is byte-identical without `--json`.

**Error behavior**: unknown provider → the shared lookup error naming the available providers (exit 1); a provider with no `interactive_command` → a hard error naming it (`configure providers.<name>.interactive_command`, exit 1 — the explicit-pane posture: no descent); unreachable tmux server or a failed spawn → **exit 3** (reachability is probed via `pane.ServerReachable` before anything is spawned).

### Subcommand: `fab pane ready`

`fab pane ready <pane>` runs the readiness classifier against the named pane id — the same gate `fab dispatch ready` binds over, with no dispatch record to load. Source: `src/go/fab/cmd/fab/pane_ready.go`.

The probe types the `FAB-READY-PROBE` sentinel literally (never submitted), checks the echo against two screen-stability captures, clears with `C-u`, and reports exactly one of `ready` / `booting` / `parked` — the classification table the two verbs share is documented under [dispatch.md](/runtime/dispatch.md) § `fab dispatch ready`. Non-`ready` reports add `pane: %N`, a `server: <name>` line when set, and the trailing capture snippet under `--- last 20 lines ---` (no header over an empty snippet). **All three classifications exit 0** — the report string is the sole discriminator. `--json` instead emits a single JSON object `{"state": "<ready|booting|parked>", "pane": "%N", "server": null, "snippet": "…"}` — `snippet` is the same trailing-blank-trimmed 20-line capture the text report carries (`""` when empty), `server` is `null` when empty, and all three classifications still exit 0. Plain-text output is byte-identical without `--json`.

**Side effect**: the probe TYPES into the target pane (the sentinel, `C-u`-cleared before return), so run it only against panes you own — never one an agent or a human is actively working in.

**Error behavior**: pane missing → **exit 2** (`Error: pane <id> not found` via `ValidatePane`); any other tmux failure (dead server, bad socket, failed probe) → **exit 3** — the pane-family scheme.

### Subcommand: `fab pane deliver`

`fab pane deliver <pane> (--prompt-file <path> | --text <string>)` runs the verified typed-delivery choreography against the named pane — the same gate `fab dispatch deliver` binds over, with no dispatch record and no completion signals to stash. Source: `src/go/fab/cmd/fab/pane_deliver.go`.

Per attempt: readiness probe → `C-u` → capture the cleared baseline → type the payload literally → wrap-tolerant echo-verify (`countWrapped`, whitespace + box-drawing runes dropped) → `Enter` → confirm the screen advanced, with exactly **one retry** (a retry warns on stderr even when it succeeds; a second failure prints the pane's last 20 lines to stderr and exits 1). `--prompt-file` checks the file exists first (missing → exit 1, nothing typed) and types the dispatch-parity pointer line `Read <path> and execute it.` (`pane.PointerPrompt`) — the path is typed **as supplied** (a generic pane's cwd is unknown to fab), so make it meaningful from the pane's own cwd; `--text` types its argument literally. The two flags are mutually exclusive and exactly one is required (usage error otherwise).

**Output**: `delivered <pane> (prompt <path>)` or `delivered <pane> (text)`. `--json` instead emits a single JSON object on verified delivery — `{"pane": "%N", "source": "prompt", "path": "<path>"}` or `{"pane": "%N", "source": "text"}` (`path` is present only for `prompt`). Failures keep the stderr + non-zero contract and emit no JSON. Plain-text output is byte-identical without `--json`.

**Error behavior**: pane missing → **exit 2**; other tmux failure → **exit 3**; verification exhausted or a missing prompt file → exit 1 through RunE.

### Subcommand: `fab pane kill`

`fab pane kill <pane>` kills a tmux pane — the generic, record-free exposure of the `pane.KillPane` helper, giving removal paths and probe cleanups the family's validated exit-code contract instead of raw `tmux kill-pane`. Source: `src/go/fab/cmd/fab/pane_kill.go`.

**Validation**: `pane.ValidatePane` first (the targeted `display-message` probe) — a missing pane prints `Error: pane <id> not found` and exits **2** in-handler via the `*PaneNotFoundError` branch; any other tmux failure (dead server, bad socket) exits **3** — the family scheme.

**Output**: `killed <pane>` on success, plus a `server: <name>` line when the socket is non-default (matching `open`'s form).

**Scope**: generic and record-free — no dispatch-record interaction, no `.fab-dispatch/` state. `fab dispatch kill` (record-keyed, ungated recovery) is unaffected and remains the pipeline's kill.

### Subcommand: `fab pane await`

`fab pane await <pane> [--file <path>] [--timeout <secs>]` is the record-free blocking wait — the completion primitive for the provider-generic layer. It blocks until **any** waitable signal fires, prints a one-word report, and (except for `gone`) exits 0 — the report string is the discriminator (the `fab dispatch wait` precedent). Source: `src/go/fab/cmd/fab/pane_await.go`.

| Report | Fired by | Exit |
|--------|----------|------|
| `idle` | the pane's `@rk_agent_state` resolves to `idle` (the shared parser, same read as `map`/`send`/`capture`) | 0 |
| `file` | the `--file <path>` file exists | 0 |
| `running` | `--timeout` expired with neither signal — the timeout bounds the observer, not the worker | 0 |
| `gone` | the pane died mid-wait — the wait cannot complete; the caller branches on cause | 2 |

**Signals compose as OR**: with both `--file` and an instrumented pane, whichever fires first wins. An uninstrumented pane (agent state unknown/absent) with **no `--file`** is an immediate error (exit 1 via RunE) — there is nothing observable to wait on, and blocking to timeout would report `running` while meaning "never watching anything".

**Cadence/bounds**: an internal ~2s re-derive tick (`pane.AwaitTick`, no flag, no config) and `--timeout` default 300 (0 or negative = unbounded), mirroring `dispatch wait`'s observer conventions; the first observation happens before any sleep. The control loop lives in `internal/pane` as the pure, observer-injected `Await(ctx, observe, tick, timeout)` (see §Shared Pane Package).

**Error behavior**: pane missing at entry → **exit 2**; other tmux failure → **exit 3**; unwaitable pane (above) → exit 1 via RunE.

### Usage-Error Coexistence with the Binary-Wide Exit-2 Convention (swon)

The pane-family 2/3 scheme above (`2` = pane missing, `3` = other tmux failure, across `capture`/`send`/`process`/`window-name`/`ready`/`deliver`/`kill`/`await`) is set via an **in-handler `os.Exit`** from inside each verb's `RunE`. The `fab`/`fab-go` binary also has (swon) a **binary-wide usage-error convention** — `0` success / `1` operational failure / **`2` usage error** — where a *usage* error (an unknown/malformed flag, or a cobra arg-count violation on any pane verb) is caught at **parse/validation time, before the handler runs**, and exits `2` via the classifier in `main()`'s testable `run()` helper (execution-phase classification, no string matching — see [kit-architecture.md](/distribution/kit-architecture.md) § Binary-Wide Exit-Code Convention).

The two `2`s **coexist without renumbering**: the in-handler pane `os.Exit(2|3)` calls **bypass `run()`'s usage/operational mapping entirely**, so the pane 2/3 scheme is **unchanged** — a missing pane still exits `2` in-handler, any other tmux failure still exits `3`, and `pane_exitcode_test.go` stays green unmodified. On a pane verb, exit `2` is therefore intentionally ambiguous between "usage error" (at parse time) and "pane missing" (in-handler); the codes are not renumbered because downstream consumers (operator/run-kit scripts) branch on the pane codes. Disambiguate on stderr wording (`Error: pane <id> not found` for the in-handler case vs. cobra's usage/flag error text for the parse-time case). `map` is the sole pane verb using plain `ERROR:`-formatted exit `1` for its in-handler errors (multi-pane discovery has no single target pane to be "missing") and is unaffected — a *usage* error on it likewise exits `2` at parse time under the binary-wide convention.

#### Output modes

Plain text is the default: `renamed: <old> -> <new>\n` on a rename, empty stdout on a no-op. The `--json` flag emits a single JSON object on stdout with the shape `{"pane", "old", "new", "action"}` where `action` is `"renamed"` or `"noop"`. JSON output always emits an object (including for no-ops), unlike plain output which is empty on no-op. Matches the plain/`--json` pattern used by `map` and `capture`.

#### Operator skill consumption

`src/kit/skills/fab-operator.md` §4 Enrollment invokes `fab pane window-name ensure-prefix <pane> »` after writing the monitored entry to `.fab-operator.yaml`. §4 Removal invokes `fab pane window-name replace-prefix <pane> » ›` on every removal path (terminal stage, `stop_stage` reached, pane death, explicit stop) — exit 2 is treated as successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."` and continue.

### `--server` / `-L` Flag

**Registration**: `paneCmd` registers a persistent string flag `--server` (short `-L`) with default `""`. Because it is a persistent flag on the parent, it is automatically visible on all ten subcommands' `--help`. Source: `src/go/fab/cmd/fab/pane.go:14`.

**Help text**: `Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.`

**Behavior**:

- When the flag is **absent or empty**, every `exec.Command("tmux", ...)` invocation in the pane call tree runs with no `-L` argument. Tmux inherits socket selection from `$TMUX` (when set) or falls back to its default socket. This is byte-for-byte identical to pre-flag behavior.
- When the flag is **non-empty**, every `exec.Command("tmux", ...)` invocation in the pane call tree is prepended with `-L <value>`. The flag is passed to tmux verbatim — fab does not inspect, validate, or normalize the server name. Tmux owns the semantics; any error (e.g., `no server running on /tmp/tmux-1001/nonexistent`) is propagated to stderr.

**Short form**: `fab pane map -L runKit` is identical to `fab pane map --server runKit`.

**Motivating use case**: The run-kit daemon runs inside a tmux session named `rk-daemon` (so its `$TMUX` points to one socket) while the user's sessions it is inspecting live on a different socket (`runKit`). Without `--server`, `fab pane map --json --all-sessions` invoked by `rk serve` enumerates panes from the wrong socket — the one in its own `$TMUX` — and every key lookup misses. With `fab pane map --json --all-sessions --server runKit`, every internal tmux invocation runs with `-L runKit` and the correct pane set is returned. More generally, the flag enables any programmatic caller that needs to inspect a tmux server different from the one it inherits.

**Workarounds that don't work** (and why the flag is the right fix): Setting `$TMUX` as a socket selector is incorrect — `$TMUX` means `socket,pid,pane_id`, not a socket path, and some tmux code paths behave differently when `$TMUX` is set (e.g., refusing nested `attach`). `$TMUX_TMPDIR` only helps for default-named sockets in a dedicated tmpdir. Unsetting `$TMUX` and relying on the default socket only works when the target is in fact the default.

### Semantic Invariants

**Pane IDs are per-server.** Tmux allocates pane IDs (e.g., `%3`, `%5`) within each tmux server's own scope. The same `%3` can exist on two different servers and refer to unrelated panes. When `--server <S>` is passed with a pane ID argument, the ID is interpreted in the context of server `<S>`. Callers are responsible for pairing the correct pane ID with the correct server.

**`--server` takes precedence over `$TMUX`.** When both are set, the explicit CLI flag wins. This matches tmux's own behavior — `tmux -L <label>` explicitly selects a socket, overriding any inherited selection.

**Non-tmux operations are unaffected by `--server`.** File reads (`.status.yaml`), git-worktree detection (`git rev-parse --show-toplevel`, `git worktree list`), and OS-level process discovery (`/proc` on Linux, `ps` on macOS) key off the pane's CWD or the resolved folder name, not the tmux server. The `--server` value is never used as a filesystem lookup key. (Agent state is a tmux pane option, not a file read — so it *does* flow through the `--server`-prefixed tmux calls.)

### Shared Pane Package (`internal/pane`)

Shared pane machinery lives in `src/go/fab/internal/pane/` — the pane-resolution helpers in `pane.go`:

- `RunCmd(name string, args ...string) (stdout string, stderr []byte, err error)` — the single subprocess-capture implementation for any child command (tmux, git, wt): captures stdout and stderr separately, returning stdout **untrimmed** so capture-style output is never altered
- `StderrError(err error, stderr []byte) error` — appends the trimmed child stderr to an exec error when present (`%w: <stderr>`; returns `err` unchanged when stderr is empty, the original error stays unwrappable via `errors.Is/As`), so failures surface the child's diagnostic — the agent self-correction signal — instead of a bare `exit status 1`
- `IsPaneMissing(stderr []byte) bool` — case-insensitive substring matcher for tmux's missing-pane stderr ("can't find pane" / "no such pane" / pane…not found); shared by `ValidatePane` and the `window-name` verbs' exit-code mapping (`tmuxExitCode` in `pane_window_name.go` is unified onto it)
- `ValidatePane(paneID, server string) error` — a single **targeted probe** `tmux display-message -t <pane> -p '#{pane_id}'`, comparing the trimmed output to the argument (ID-exact: window-name / target-grammar args resolve to a *different* pane ID and are rejected). Version-robust via two detection branches: on tmux ≥3.6 a missing pane exits 0 with **empty output** (caught by the output==arg comparison — the load-bearing check, verified empirically on 3.6a); older tmux errors with "can't find pane" stderr (caught by `IsPaneMissing`). Missing pane → the typed `*PaneNotFoundError` (message `pane <id> not found`, byte-identical to the historical string; detectable via `errors.As`, which is how capture/send map it to exit 2 vs. 3 — no string matching); other tmux failures (dead server, bad socket) surface stderr via `StderrError`. Pure decision half extracted as `validatePaneResult` for tmux-free tests
- `GetPanePID(paneID, server string) (int, error)` — resolves shell PID via `tmux display-message`
- `ReadWindowName(paneID, server string) (string, []byte, error)` — reads the tmux window name via `tmux display-message -p -t <pane> '#W'`, trimmed; delegates to `RunCmd`. Returns (name, tmux stderr bytes, exec error) — callers use the stderr bytes to map tmux's "can't find pane" message to exit 2 vs. other tmux failures to exit 3. Used by the `window-name` subcommand group.
- `ResolvePaneContext(paneID, mainRoot, server string) (*PaneContext, error)` — resolves worktree, change, stage, and agent state from the pane's CWD
- `FindMainWorktreeRoot(cwds []string) string` — derives the main worktree root from pane CWDs via `git worktree list --porcelain`
- `WithServer(server string, args ...string) []string` — the canonical argv-building helper (see Design Decisions)
- The **readiness gate and verified-delivery choreography** (`gate.go`): `Gate`/`NewGate`, `Probe`, `Deliver`, `DeriveReadiness`, the `Readiness` constants (`ReadyReady`/`ReadyBooting`/`ReadyParked` — the exact report strings `fab pane ready` and `fab dispatch ready` print), `ReadySentinel`, `SnippetLines`, `Snippet`, `PaneIO`, the wrap-tolerant echo counter (`countWrapped`/`squeeze`, dropping whitespace and U+2500–U+257F box-drawing runes), and `Tail`. One copy of the classifier and choreography serves both the `fab pane` primitives and the `fab dispatch` bindings
- The **pane creators and lifecycle mechanics** (`create.go`): `OpenWindow`, `OpenSplitPane`, `OpenPlainPane` (the unsized, untitled split/window behind `fab pane open`), `SplitPlacement` (+`Describe`) with its package-scope `splitArgs` argv renderer, `ServerReachable`, `PaneAlive`, `KillPane`, and `PointerPrompt` (the `Read <path> and execute it.` pointer-line composer `fab pane deliver --prompt-file` and `fab dispatch deliver` share). The placement *decision* half (`SplitTarget`/`SelectPaneShape`/`SiblingDispatchPane`) stays in `internal/dispatch` — it reads dispatch records and config
- The **blocking-wait control loop** (`await.go`): `Await(ctx, observe, tick, timeout)` plus the `AwaitReport` constants (`idle`/`file`/`running`/`gone`) and the `AwaitTick` (2s) cadence constant — a pure, observer-injected loop mirroring `internal/dispatch`'s `Wait`/`TickInterval` semantics (first observation before any sleep; timeout bounds the observer, not the worker), so it is unit-testable without tmux. The cobra layer (`fab pane await`) fills the observer with pane liveness, `--file` existence, and the `@rk_agent_state` idle read

All tmux-invoking functions accept a trailing `server string` parameter and build their argv via `WithServer`. Callers in `cmd/fab/pane*.go` read the flag via `cmd.Flags().GetString("server")` and thread the value through. The `RunCmd`/`StderrError` pair is applied at the capture (`capturePaneContent`), send (both `send-keys` sites), operator (`tmux new-window`, `gitRepoRoot`), batch-new (`wt create`, `tmux new-window`), and batch-switch (`wt create` — added by 260717-otol when `batch switch` moved off `.Output()` to surface wt's typed exit-2 error in its warn-and-skip line) subprocess sites — errors include the trimmed child stderr and the relevant identifier (pane ID / target).

## Design Decisions

### Targeted `display-message` Probe with Two Detection Branches (ValidatePane)
**Decision**: `ValidatePane` is a single `tmux display-message -t <pane> -p '#{pane_id}'` probe whose trimmed output must equal the argument, with two detection branches for a missing pane: the output==arg comparison (load-bearing on tmux ≥3.6, where `display-message` exits **0 with empty output** for a missing pane) and the `IsPaneMissing` stderr mapping (older tmux, which errors with "can't find pane").
**Why**: The previous `tmux list-panes -a` pre-check enumerated every pane on the server before each `capture`/`send`/`process` invocation and was TOCTOU-ineffective anyway. The probe keeps both contracts the enumeration provided — existence checking and **ID-exactness** (`-t` alone accepts the full tmux target grammar: window names, `session:win.pane` — a behavioral loosening) — at O(1) subprocess cost. Empirical verification on tmux 3.6a contradicted the originally assumed stderr-only error path, so both branches are required for version robustness; error-path equivalence (at the time: missing pane → `Error: pane <id> not found` exit 1; dead server → exit 1 with stderr detail now included) was re-verified before the old path was removed. (The flat exit-1 codes described here were subsequently split into the pane-family 2/3 scheme for capture/send by 260612-ye8r — see the Error behavior sections above.)
**Rejected**: Bare `-t` targeting without the output comparison (accepts the full target grammar — loosens ID-exactness). Keeping the `list-panes -a` pre-check (O(server) per invocation, race-prone). New helpers in a fresh `internal/tmuxutil` package (over-engineering for three helpers; `internal/pane` is the documented home for cross-package tmux helpers per the `WithServer` decision below).
*Introduced by*: 260612-pw3k-operator-pane-perf-error-surfacing

### Persistent Flag on the Parent, Not Per-Subcommand
**Decision**: `--server` is registered as a persistent flag on `paneCmd` via `cmd.PersistentFlags().StringP("server", "L", "", "...")`, visible on all ten subcommands. Each subcommand reads the value via `cmd.Flags().GetString("server")`.
**Why**: Cobra idiom for a flag that applies uniformly across a command group. Single registration point, single help-text location, zero chance of per-subcommand drift.
**Rejected**: Per-subcommand registration — one copy of the same flag per subcommand, as many places to update if the description changes.
*Introduced by*: 260417-2fbb-pane-server-flag

### `WithServer` Helper in `internal/pane/pane.go`
**Decision**: A single argv-building helper `WithServer(server string, args ...string) []string` lives in `src/go/fab/internal/pane/pane.go`. It returns `args` unchanged when `server == ""` and `append([]string{"-L", server}, args...)` otherwise. Every `exec.Command("tmux", ...)` site in the pane call tree builds its argv via this helper.
**Why**: `WithServer` is a short pure function that eliminates per-file conditional logic and ensures the `-L` prepend is identical at every call site. Scope is exactly one helper for one flag; introducing an `internal/tmuxutil/` package or a `TmuxClient` struct type would be over-engineering for a single-flag change and can be promoted later if tmux-helper surface grows.
**Exported** (rather than unexported as drafted in the spec): the helper is used from the `cmd/fab` package (e.g., inside `sendTextArgs`, `listPanesArgs`, `capturePaneArgs`) to keep a single canonical argv builder across packages. Cross-package argv builders in this codebase are exported from `internal/pane` when consumed outside the pane package — future tmux-helper additions should follow the same pattern.
*Introduced by*: 260417-2fbb-pane-server-flag

### Helper Named `WithServer`, Not `tmuxArgs`
**Decision**: The helper is named `WithServer`. The pre-existing local variable `tmuxArgs` in `pane_send.go:58` is preserved.
**Why**: `tmuxArgs` was already a local variable name; a free function of the same name would shadow or collide. Renaming the local variable creates churn outside the flag's scope. `WithServer` also reads naturally at call sites: `exec.Command("tmux", WithServer(server, "list-panes", "-a")...)`.
*Introduced by*: 260417-2fbb-pane-server-flag

### Pass the Server Name Verbatim to Tmux
**Decision**: The `--server` value is passed to tmux without fab-side validation, escaping, or normalization.
**Why**: Tmux owns the semantics of socket labels. Any pre-validation in fab (e.g., `tmux -L <server> has-session`) would duplicate tmux's own error handling and introduce race conditions (socket created/destroyed between check and use). Propagating tmux's native error is simpler and more accurate.
**Rejected**: Pre-check via `tmux has-session` — extra subprocess, and fab would still need to handle the real tmux error from the actual command anyway.
*Introduced by*: 260417-2fbb-pane-server-flag

### `-L <name>` Only — No `-S <path>` in First Cut
**Decision**: Only `--server <name>` (maps to `tmux -L <name>`) is exposed. A `--socket-path` / `-S` equivalent is a non-goal for the first cut.
**Why**: `-L` covers the motivating run-kit case and every named-socket scenario. Callers that truly need a full path rather than a label are rare; adding `-S` later is cheap and non-breaking.
**Rejected**: Env-var alternative (`FAB_TMUX_SERVER`) — adds hidden env coupling; CLI flag is more discoverable via `--help` and easier to plumb through subprocess-style callers that already build argv slices.
*Introduced by*: 260417-2fbb-pane-server-flag

### Provider-Generic Primitives in the Pane Family, Dispatch as the Record-Keeping Binding
**Decision**: The readiness classifier, the verified-delivery choreography, the tmux pane creators/liveness/kill helpers, `Tail`, and `PointerPrompt` live in `internal/pane` and are exposed as the record-free `fab pane open`/`ready`/`deliver` verbs; `fab dispatch open`/`ready`/`deliver` are thin bindings over them that add only dispatch-record bookkeeping. Placement *policy* and all record state stay in `internal/dispatch`.
**Why**: Provider probes, warm-ups, and operator interactions need the gate and verified delivery addressed by pane id with no active change — before this split every piece of that choreography was reachable only through a change+stage dispatch record, so probes were hand-rolled with raw tmux. The import graph allows the move cleanly (`internal/pane` has no dispatch dependency and the gate needs none), and one home for the tmux mechanics means the echo-verify cannot grow a divergent second copy.
**Rejected**: Ad-hoc `--no-record`/`--pane-id` flags on the dispatch verbs (muddies the five-state dispatch contract); a new sibling package (no import cycle exists to force one); worker-column placement on `fab pane open` (placement is pipeline policy and stays dispatch-owned — the primitive does a plain split).
*Introduced by*: 260810-1lah-provider-generic-pane-verbs

### `fab pane open` Pins the Provider and Applies the Standard Fill Precedence
**Decision**: `--provider` pins the provider at the top of the standard fill precedence (`agent.ResolveRoleWith` with the provider override set); `--role` selects whose `{model}`/`{effort}` fills apply, defaulting to `default`.
**Why**: A probe spawn wants the same resolved command a pipeline worker would get; `fab agent --provider`'s deliberate fill bypass composes a profile-free invocation the probe would then have to hand-tune.
**Rejected**: Reusing `fab agent --provider`'s bypass semantics (contradicts the probe use case).
*Introduced by*: 260810-1lah-provider-generic-pane-verbs

### Unknown Agent State Warns and Proceeds; Refusal Is Reserved for a Parseable Non-Idle State
**Decision**: `fab pane send` treats an absent/unparseable `@rk_agent_state` as warn-and-send (`warning: agent state unknown — sending anyway`, exit 0); only a parseable `active`/`waiting` state refuses. `--force` retains its skip-everything meaning.
**Why**: Only run-kit-instrumented panes set the option, so against any foreign-agent pane (kimi, agy, codex) the unknown case fired unconditionally and every send needed `--force` — the validation never validated in precisely the multi-provider scenarios where validation matters. Reserving refusal for a parseable non-idle state keeps the guard where the convention actually reports state.
**Rejected**: Keeping unknown as a refusal (the always-`--force` ritual); auto-force on unknown (silently drops the one signal an instrumented pane might still carry).
*Introduced by*: 260810-1lah-provider-generic-pane-verbs

### Await's Control Loop Mirrors `dispatch.Wait`
**Decision**: `fab pane await`'s tick loop lives in `internal/pane` as a pure, observer-injected control structure (`Await(ctx, observe, tick, timeout)`) with a package `AwaitTick` constant — the same shape as `internal/dispatch`'s `Wait`/`TickInterval`.
**Why**: The observer conventions (first observation before any sleep, timeout bounds the observer not the worker, observe-once-more at the bound) are already the family's documented blocking-wait contract; reusing the shape keeps the two waits incapable of drifting on semantics, and observer injection makes the loop unit-testable without tmux.
**Rejected**: A hand-rolled `for`/`sleep` poll in the cobra layer (untestable, re-derives solved edge cases); a filesystem watcher (cannot see a pane die — liveness needs a periodic probe regardless).
*Introduced by*: 260811-yxyi-pane-dispatch-surface-completion
