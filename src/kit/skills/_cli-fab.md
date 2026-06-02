---
name: _cli-fab
description: "Fab CLI command reference — calling conventions, flag details, and commands not covered by the Common fab Commands subsection of _preamble."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Fab CLI Reference

> Loaded selectively via a skill's `helpers: [_cli-fab]` frontmatter. See `_preamble.md` § Common fab Commands for the 6 most-used commands (`preflight`, `score`, `log command`, `change`, `resolve`, `status`). This file documents the remaining commands and exhaustive flag details.

---

## Calling Convention

`fab <command> <subcommand> [args...]`. `fab` is a router dispatching workspace commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`) to `fab-kit` and everything else to the per-version `fab-go` binary resolved from `fab_version` in `fab/project/config.yaml`. `--version`/`-v`/`--help`/`-h`/`help` are handled inline. `fab-go` auto-fetches from GitHub releases on cache miss.

`fab -h` composes help from both binaries. `fab --version` prints the system binary version; inside a fab repo a second line shows the project-pinned version.

### `<change>` Argument

All commands accept the unified `<change>`: 4-char ID (`yobi`), folder substring (`fix-kit`), or full folder name (`260227-yobi-fix-kit-scripts`). Bare directory paths and `.status.yaml` paths are NOT accepted.

### Commands covered in `_preamble` Common fab Commands

`fab preflight`, `fab score`, `fab log command`, `fab change`, `fab resolve`, `fab status` — headline coverage lives there. Sections below document the remaining commands (`fab hook`, `fab pane`, `fab doctor`, `fab kit-path`, `fab fab-help`, `fab operator`, `fab batch`) and extended flag details for the above.

---

## fab change (extended subcommand details)

See `_preamble.md` § Common fab Commands for the headline. Full subcommand table:

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `new` | `new --slug <slug> [--change-id <4char>] [--log-args <desc>]` | Create new change |
| `rename` | `rename --folder <current-folder> --slug <new-slug>` | Rename slug (prefix immutable) |
| `resolve` | `resolve [<override>]` | Passthrough to `fab resolve --folder` |
| `switch` | `switch <name> \| --none` | Switch active change (writes `.fab-status.yaml` symlink) |
| `list` | `list [--archive]` | List changes with stage info |
| `archive` | `archive <change> --description "..."` | Move to `archive/`, update index, clear pointer |
| `restore` | `restore <change> [--switch]` | Move from `archive/`, remove index entry, optionally activate |
| `archive-list` | `archive-list` | List archived folder names |

`archive` and `restore` output structured YAML to stdout — skills parse it for user-facing reports.

---

## fab status (extended subcommand details)

Full subcommand table (headline in `_preamble` § Common fab Commands):

| Subcommand | Usage | Notes |
|------------|-------|-------|
| `finish` | `finish <change> <stage> [driver]` | Done + auto-activate next. Review auto-logs `passed` |
| `start` | `start <change> <stage> [driver] [from] [reason]` | pending/failed → active |
| `advance` | `advance <change> <stage> [driver]` | active → ready |
| `reset` | `reset <change> <stage> [driver] [from] [reason]` | done/ready/skipped → active (cascades downstream to pending) |
| `skip` | `skip <change> <stage> [driver]` | {pending,active} → skipped (cascades pending→skipped downstream) |
| `fail` | `fail <change> <stage> [driver] [rework]` | active → failed (review only). Auto-logs `failed` |
| `set-change-type` | `set-change-type <change> <type>` | |
| `set-acceptance` | `set-acceptance <change> <field> <value>` | Updates `plan:` block. Valid fields: `generated` (bool), `task_count`, `acceptance_count`, `acceptance_completed` (int) |
| `set-checklist` | `set-checklist [args...]` | **Removed** — exits 1 with `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` Use `set-acceptance` |
| `set-confidence` | `set-confidence <change> <counts...> <score> [--indicative]` | Basic confidence block. `--indicative` is a deprecated accepted-but-ignored no-op (1.10.0) — it writes nothing |
| `set-confidence-fuzzy` | `set-confidence-fuzzy <change> <counts...> <score> <dims...> [--indicative]` | With SRAD dimensions. `--indicative` is a deprecated no-op (see above) |
| `add-issue` / `get-issues` | `<change> <id>` / `<change>` | Issue ID array — idempotent / one per line |
| `add-pr` / `get-prs` | `<change> <url>` / `<change>` | PR URL array — idempotent / one per line |
| `progress-line` | `progress-line <change>` | Single-line visual progress |
| `current-stage` | `current-stage <change>` | Detect active stage |

**Side effects of `finish`**: `intake→apply`, `apply→review`, `review→hydrate` (+auto-log `passed`), `hydrate→ship`, `ship→review-pr`. Never call `start` after `finish`. Legacy `tasks` event invocations exit 1 with `"tasks" stage was removed — run "fab status <event> <change> apply" instead. plan.md is now generated at apply entry.` Legacy `spec` event invocations exit 1 with `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`

**Auto-logs**: `finish review`→`passed`; `fail review`→`failed`; every `active` transition is best-effort logged. Skills do NOT manually call `fab log review` or `fab log transition`.

---

## fab score (extended)

See `_preamble.md` § Common fab Commands. Modes:

| Mode | Usage | Behavior |
|------|-------|----------|
| Normal | `fab score <change>` | Parse `intake.md` (the sole scoring source; `--stage` defaults to `intake`), compute, write `.status.yaml`. No `indicative` key is written (retired 1.10.0) |
| Gate | `fab score --check-gate <change>` | Read-only, threshold compare, non-zero below threshold |
| Intake gate | `fab score --check-gate --stage intake <change>` | Flat threshold 3.0 for all types (the single gate) |

---

## fab preflight (extended)

`fab preflight [<change-name>]` — validates config.yaml, constitution.md, active change resolution, `.status.yaml` existence. Outputs YAML with `name`, `change_dir`, `stage`, `progress`, `plan`, `confidence`. Non-zero exit on failure (error on stderr). Pure validation — no side effects.

---

## fab log (extended)

Append-only JSON logging to `.history.jsonl`.

```
fab log command <cmd> [change] [args]
fab log confidence <change> <score> <delta> <trigger>
fab log review <change> <result> [rework]
fab log transition <change> <stage> <action> [from] [reason] [driver]
```

`command` resolves active change from `.fab-status.yaml` when `[change]` omitted; exits 0 silently if resolution fails (dangling/absent symlink). When `[change]` IS provided and doesn't resolve → exits 1.

**Common callers** — skills per `_preamble.md` Context Loading §2 (`fab log command "<skill>" "<change>"`); `finish/fail review` auto-log; `score` auto-logs confidence; `change new`/`change rename` auto-log.

---

## fab resolve (extended)

Pure query, no side effects.

```
fab resolve [--id|--folder|--dir|--status|--pane] [<change>]
```

| Flag | Output |
|------|--------|
| `--id` (default) | 4-char change ID |
| `--folder` | Full folder name |
| `--dir` | Directory path (`fab/changes/.../`) |
| `--status` | `.status.yaml` path |
| `--pane` | Tmux pane ID (requires `$TMUX`; errors if no matching pane) |

---

## fab hook

Claude Code hook handlers. Each subcommand is registered as inline `fab hook <subcommand>` in `.claude/settings.local.json`. **All hook subcommands exit 0** — errors silently swallowed so they never block the agent.

| Subcommand | Event | Purpose |
|------------|-------|---------|
| `session-start` | SessionStart | Delete `_agents[session_id]` entry in `.fab-runtime.yaml` |
| `stop` | Stop | Write `_agents[session_id]` with `idle_since` plus optional tmux/pid/change/transcript fields |
| `user-prompt` | UserPromptSubmit | Remove only `idle_since` from `_agents[session_id]`; other fields preserved |
| `artifact-write` | PostToolUse (Write/Edit) | Per-artifact bookkeeping from stdin JSON |
| `sync` | n/a | Register inline hook entries in `.claude/settings.local.json`; migrates old-style bash scripts; idempotent |

The three session-scoped hooks (`session-start`, `stop`, `user-prompt`) read a JSON payload on stdin with at least a `session_id` field (UUID) and optionally `transcript_path`. Malformed JSON or a missing `session_id` is silently skipped. Each handler also invokes a throttled GC sweep (≤ once per 180 s via `last_run_gc`) that prunes entries whose stored `pid` no longer exists (`kill(pid, 0)` returning ESRCH). `artifact-write` is unchanged — it parses a different payload shape (`tool_input.file_path`) and does not participate in `_agents` writes; it emits `{"additionalContext":"Bookkeeping: ..."}` on stdout.

`sync` output: `Created`, `Updated`, or `.claude/settings.local.json hooks: OK`.

---

## fab pane

Tmux pane operations with fab context enrichment. `fab pane <map|capture|send|process> [flags...]`

**Persistent flag** (all subcommands): `--server <name>` / `-L <name>` (default `""`) — target tmux socket (`tmux -L <name>`). Defaults to `$TMUX` / tmux default. Lets daemons on one tmux server inspect panes on another.

### map — `fab pane map [--json] [--session <name>] [--all-sessions] [--server <name>]`

All tmux panes with pipeline state. Non-git/non-fab panes included with `---` fallbacks.

| Flag | Description |
|------|-------------|
| `--json` | JSON array (snake_case: `session`, `window_index`, `pane`, `tab`, `worktree`, `change`, `stage`, `agent_state`, `agent_idle_duration`) |
| `--session <name>` | Target specific session (skips `$TMUX` check) |
| `--all-sessions` | Query all sessions (skips `$TMUX` check; mutually exclusive with `--session`) |

Without `--session`/`--all-sessions` → current session only (`-s` scope, requires `$TMUX`). Table columns: `Session` (only with `--all-sessions`), `Pane`, `WinIdx`, `Tab`, `Worktree` (relative; `(main)` for main; `basename/` non-git), `Change`, `Stage`, `Agent`. Agent: `active`, `idle ({dur})`, or `—` (em dash). Change: folder name, `(no change)` for fab worktree with no active change, or `—` for non-fab panes. Idle duration: `{N}s`/`{N}m`/`{N}h` floor division. Change and Agent resolve on independent axes: Change comes from `.fab-status.yaml`; Agent comes from `_agents[*].tmux_pane` matching in `.fab-runtime.yaml` — so a pane with a running Claude in discussion mode (no active change) now shows `(no change)` in Change but a populated Agent column. `$TMUX` unset without targeting flag → exit 1. No panes → exit 0 `No tmux panes found.`

### capture — `fab pane capture <pane> [-l N] [--json] [--raw] [--server <name>]`

`<pane>` required (e.g., `%5`). `-l/--lines N` (default 50). `--json` = content + metadata (`worktree`/`change`/`stage`/`agent_state`/`agent_idle_duration`). `--raw` = plain `tmux capture-pane -p`, no enrichment. `--json`/`--raw` mutually exclusive. Pane not found → exit 1.

### send — `fab pane send <pane> <text> [--no-enter] [--force] [--server <name>]`

Validation pipeline: (1) pane exists via `tmux list-panes -a`; (2) agent is idle (rejects `active`/`unknown` unless `--force`); (3) `tmux send-keys`. `--no-enter` skips the trailing Enter. `--force` bypasses idle check only — pane-existence still enforced. Agent resolution matches `_agents[*].tmux_pane` in `.fab-runtime.yaml` at the worktree root; a pane with no matching entry = `unknown` (non-idle). Change state is independent — panes in discussion mode (no active change) now accept sends when idle, instead of being rejected as `unknown`. Success: `Sent to <pane>`.

### process — `fab pane process <pane> [--json] [--server <name>]`

OS-level process tree. Linux: walks `/proc/<pid>/task/<tid>/children`, reads `/proc/<pid>/comm` + `/cmdline`. macOS: `ps -o pid,ppid,comm -ax` PPID traversal, `ps -o args= -p <pid>` for full cmdline. Classification: `claude`/`claude-code` → `agent`, `node` → `node`, `git`/`gh` → `git`, else `other`. JSON: `{pane, pane_pid, processes (tree), has_agent}`. Pane not found → exit 1. `--server` scopes tmux lookup only; `/proc`/`ps` walk is socket-independent.

---

## fab doctor

Prerequisite check. Lives in `fab-kit` so it works before `config.yaml` exists; used as `/fab-setup` Phase 0 gate.

```
fab doctor [--porcelain]
```

**Checks** (7): git, fab, bash, yq (v4+), jq, gh, direnv (with zsh/bash hook detection).

**Output**: `  ✓ {tool} {version}` (pass) / `  ✗ {tool} — not found` + install hint (fail) / summary line. Exit code = failure count.

`--porcelain`: errors only (no passes/hints/summary). Exit code still = failure count. Empty stdout + exit 0 = all good.

---

## fab kit-path

```
fab kit-path
```

Prints absolute path to the resolved kit directory (exe-sibling `kit/` next to `fab-go`). No trailing newline, no decoration. Exit 0 on success; non-zero with stderr error on failure. Used by skills to reference kit content: `$(fab kit-path)/templates/`, `$(fab kit-path)/migrations/`, etc.

---

## fab impact

```
fab impact <base> <head>
```

Computes `git diff --shortstat <base>...<head>` line counts and emits a YAML document on stdout matching the `.status.yaml` `true_impact` block schema (minus `computed_at_stage`):

```yaml
added: 142
deleted: 38
net: 104
excluding:
    added: 87
    deleted: 38
    net: 49
computed_at: "2026-05-07T14:32:00Z"
```

The `excluding` sub-block is emitted only when `fab/project/config.yaml`'s top-level `true_impact_exclude` list is non-empty; the subcommand applies each entry as a `:(exclude)<pattern>` pathspec when running the second `git diff --shortstat` pass. Three-dot range semantics (`<base>...<head>`) — "changes on this branch only".

Exit codes:
- `0` — success; YAML document on stdout.
- non-zero — `<base>` is empty/invalid or `git diff` failed; actionable message on stderr (e.g., `base ref is empty`). The subcommand does not run `git merge-base` itself — callers must resolve the merge-base upstream and pass the result. The caller decides whether to abort or skip.

Consumers: `/git-pr` Step 3c-impact (PR body `**Impact**` line) and the apply-finish + hydrate-finish hooks (write the result into `.status.yaml` `true_impact`).

---

## fab fab-help

```
fab fab-help
```

Scans skill frontmatter from the cache kit, groups skills by category (Start & Navigate, Planning, Completion, Maintenance, Setup, Batch Operations), renders formatted overview. Excludes `_`-prefix and `internal-` prefix skills. Batch entries read dynamically from `fab batch` cobra subcommands. Unmapped → "Other".

Output: version header, workflow diagram, grouped commands, typical flow, packages section (wt, idea).

(The command name is `fab-help` — not overriding cobra's built-in `help`.)

---

## fab operator

```
fab operator
```

Singleton tmux-tab launcher for `/fab-operator`. Requires `$TMUX`. If window `operator` exists → select it (`Switched to existing operator tab.`); else create one in the repo root running `{spawn_command} '/fab-operator'` (`Launched operator.`).

**Spawn command resolution**: `agent.spawn_command` from `fab/project/config.yaml`; falls back to `claude --dangerously-skip-permissions` if missing/null/empty.

### fab operator tick-start

```
fab operator tick-start
```

Called at start of each operator tick. Increments `tick_count`, writes `last_tick_at` (ISO 8601 UTC) to `.fab-operator.yaml`. Stdout:

```
tick: N
now: HH:MM
```

### fab operator time

```
fab operator time [--interval <duration>]
```

Pure time query (no writes).

- Without `--interval`: `now: HH:MM`
- With `--interval 3m`: `now: HH:MM\nnext: HH:MM` (now + interval)

Duration is Go format (`3m`, `5m`, `2m`). Invalid → exit 1.

---

## fab batch

Multi-target operations: `fab batch <new|switch|archive> [--list] [--all] [targets...]`. Subcommands creating tmux windows require `$TMUX`.

- **`new`** — parse `fab/backlog.md` pending items (`- [ ] [xxxx]`), create worktrees, open tmux windows, start agents with `/fab-new {description}`. No args → `--list`. IDs → one worktree tab each (`wt create --non-interactive --worktree-name {id}`, window `fab-{id}`, `{spawn_command} '/fab-new {description}'`). `--all` → all pending. Handles continuation lines.
- **`switch`** — resolve change names, create worktrees with branch names (applying `branch_prefix` from config), start agents with `/fab-switch {change}`. No args → `--list`. `--all` → all active changes (excludes `archive/`). Branch naming: `{branch_prefix}{folder_name}`.
- **`archive`** — find changes with `hydrate: done|skipped`, spawn one Claude session with `/fab-archive` per change. No args → `--all` (differs from new/switch). `--list` → show archivable only. Session prompt: `Run /fab-archive for each of these changes, one at a time: {changes}`.

---

## Common Error Messages

| Error | Cause | Fix |
|-------|-------|-----|
| `Status file not found: {path}` | Passed a path that doesn't exist | Use change ID or folder name |
| `Cannot resolve change '{arg}'` | ID/name matches no folder in `fab/changes/` | Check `fab change list` |
| `Multiple changes match` | Ambiguous substring matched multiple folders | Use a more specific identifier |
| `No active changes found` | `.fab-status.yaml` symlink absent and no changes exist | Run `/fab-new` or `/fab-draft` |
