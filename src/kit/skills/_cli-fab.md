---
name: _scripts
description: "Kit script invocation guide — calling conventions for the fab dispatcher and Go backend."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Kit Script Invocation Guide

> Loaded by every skill via `_preamble.md`. Defines calling conventions for all kit operations.

---

## Calling Convention

`fab` is a router that serves as the sole entry point for all fab CLI operations. It dispatches commands to either `fab-kit` (workspace lifecycle) or `fab-go` (workflow engine) via `syscall.Exec`.

```
fab <command> <subcommand> [args...]
```

### Three-Binary Architecture

| Binary | Role | Installation |
|--------|------|-------------|
| `fab` | Router — dispatches to fab-kit or fab-go | Homebrew (system binary) |
| `fab-kit` | Workspace lifecycle — init, upgrade-repo, sync | Homebrew (system binary) |
| `fab-go` | Workflow engine — resolve, status, preflight, etc. | Per-version cache (~/.fab-kit/versions/) |

### Routing

- **Workspace commands** (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `--version`, `--help`, `help`): routed to `fab-kit`
- **Workflow commands** (everything else): routed to `fab-go` after version resolution

### Backend

1. For workspace commands: the router finds `fab-kit` on PATH and execs it
2. For workflow commands: the router reads `fab_version` from `fab/project/config.yaml`, ensures the matching `fab-go` is cached at `~/.fab-kit/versions/{version}/fab-go`, and execs it
3. If the version is not cached, the router auto-fetches it from GitHub releases

### Help

`fab -h`, `fab --help`, and `fab help` show composed help from both fab-kit and fab-go. `fab-kit -h` and `fab-go -h` show their own help independently.

### Command Reference

| Command | Purpose |
|---------|---------|
| `fab resolve` | Change reference resolution |
| `fab status` | Stage state machine + metadata |
| `fab log` | Append-only history logging |
| `fab preflight` | Validation + structured YAML output |
| `fab change` | Change lifecycle (new, rename, switch, list, archive, restore, archive-list) |
| `fab score` | Confidence scoring |
| `fab runtime` | Runtime state management (.fab-runtime.yaml) |
| `fab hook` | Claude Code hook subcommands (session-start, stop, user-prompt, artifact-write, sync) |
| `fab pane` | Tmux pane operations: map, capture, send, process |
| `fab doctor` | Validate fab-kit prerequisites (lives in fab-kit, works before config.yaml exists) |
| `fab fab-help` | Show fab workflow overview and available commands (dynamic skill discovery) |
| `fab operator` | Launch operator in a dedicated tmux tab (singleton) |
| `fab batch` | Multi-target batch operations (new, switch, archive) |

---

## `<change>` Argument Convention

All commands accept a unified `<change>` argument:

| Form | Example |
|------|---------|
| 4-char change ID | `yobi` |
| Folder name substring | `fix-kit` |
| Full folder name | `260227-yobi-fix-kit-scripts` |

**Not accepted**: bare directory paths or `.status.yaml` paths — use the folder name or change ID instead.

---

# Change Lifecycle

## fab change

Change Manager — manages change folders, naming, and the `.fab-status.yaml` symlink.

```
fab change <subcommand> [flags...]
```

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `new` | `new --slug <slug> [--change-id <4char>] [--log-args <desc>]` | Create new change |
| `rename` | `rename --folder <current-folder> --slug <new-slug>` | Rename change slug |
| `resolve` | `resolve [<override>]` | Passthrough to resolve --folder |
| `switch` | `switch <name> \| --none` | Switch active change |
| `list` | `list [--archive]` | List changes with stage info |
| `archive` | `archive <change> --description "..."` | Clean .pr-done, move to archive/, update index, clear pointer |
| `restore` | `restore <change> [--switch]` | Move from archive/, remove index entry, optionally activate |
| `archive-list` | `archive-list` | List archived folder names (one per line) |

**Resolution**: archive resolves `<change>` via standard resolution (active changes). `restore` uses internal archive-folder resolution. Both support 4-char ID, substring, and full folder name.

**Output**: Both archive and restore output structured YAML to stdout. Skills parse this YAML to construct user-facing reports.

---

# Pipeline & Status

## fab status

Status Manager — manages workflow stages, states, and `.status.yaml`.

```
fab status <subcommand> <change> [args...]
```

### Key subcommands

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `finish` | `finish <change> <stage> [driver]` | Mark stage done, auto-activate next. Review stage auto-logs "passed" |
| `start` | `start <change> <stage> [driver] [from] [reason]` | pending/failed → active |
| `advance` | `advance <change> <stage> [driver]` | active → ready |
| `reset` | `reset <change> <stage> [driver] [from] [reason]` | done/ready/skipped → active (cascades downstream to pending) |
| `skip` | `skip <change> <stage> [driver]` | {pending,active} → skipped (cascades downstream pending to skipped) |
| `fail` | `fail <change> <stage> [driver] [rework]` | active → failed (review only). Review stage auto-logs "failed" |
| `set-change-type` | `set-change-type <change> <type>` | Set change type |
| `set-checklist` | `set-checklist <change> <field> <value>` | Update checklist field |
| `set-confidence` | `set-confidence <change> <counts...> <score> [--indicative]` | Set confidence block (with optional indicative flag) |
| `set-confidence-fuzzy` | `set-confidence-fuzzy <change> <counts...> <score> <dims...> [--indicative]` | Set confidence with dimensions (with optional indicative flag) |
| `add-issue` | `add-issue <change> <id>` | Append issue ID to issues array (idempotent) |
| `get-issues` | `get-issues <change>` | List issue IDs (one per line) |
| `add-pr` | `add-pr <change> <url>` | Append PR URL to prs array (idempotent) |
| `get-prs` | `get-prs <change>` | List PR URLs (one per line) |
| `progress-line` | `progress-line <change>` | Single-line visual progress |
| `current-stage` | `current-stage <change>` | Detect active stage |

### Stage transition side effects

Each `finish` auto-activates the next pending stage. No separate `start` call needed:

```
finish intake  → spec becomes active
finish spec    → tasks becomes active
finish tasks   → apply becomes active
finish apply   → review becomes active
finish review  → hydrate becomes active (+ auto-logs review "passed")
finish hydrate → pipeline complete
```

**Common mistake**: calling `start <stage>` after `finish <previous-stage>` — this is redundant because `finish` already activated it.

### Auto-logging

- `finish <change> review [driver]` → auto-logs review "passed"
- `fail <change> review [driver] [rework]` → auto-logs review "failed"
- Any event that sets a stage to `active` → auto-logs transition (best-effort)

Skills do NOT need to call `fab log review` or `fab log transition` manually — it's handled by `fab status` internally.

---

## fab score

Confidence scorer — computes SRAD confidence score from Assumptions tables.

```
fab score [--check-gate] [--stage <stage>] <change>
```

| Mode | Usage | Behavior |
|------|-------|----------|
| Normal | `fab score <change>` | Parse spec.md, compute score, write to .status.yaml |
| Intake scoring | `fab score --stage intake <change>` | Parse intake.md, compute score, write to .status.yaml |
| Gate check | `fab score --check-gate <change>` | Parse artifact, compute score, compare threshold. Read-only |
| Intake gate | `fab score --check-gate --stage intake <change>` | Intake gate with fixed threshold 3.0 |

---

## fab preflight

Pre-flight validator — validates project state and outputs structured YAML. Purely validation + structured output — no logging side-effects.

```
fab preflight [<change-name>]
```

- `<change-name>`: Optional change override (resolved via change resolution).

Validates: config.yaml exists, constitution.md exists, active change resolved, `.status.yaml` exists. Outputs YAML with `name`, `change_dir`, `stage`, `progress`, `checklist`, `confidence` fields. Non-zero exit on failure with error message on stderr.

---

# Plumbing

## fab resolve

Change Resolver — pure query, no side effects. Converts any change reference to a canonical output.

```
fab resolve [--id|--folder|--dir|--status|--pane] [<change>]
```

| Flag | Output |
|------|--------|
| `--id` (default) | 4-char change ID (e.g., `9fg2`) |
| `--folder` | Full folder name (e.g., `260228-9fg2-refactor-kit-scripts`) |
| `--dir` | Directory path (e.g., `fab/changes/260228-9fg2-refactor-kit-scripts/`) |
| `--status` | `.status.yaml` path (e.g., `fab/changes/260228-9fg2-refactor-kit-scripts/.status.yaml`) |
| `--pane` | Tmux pane ID (e.g., `%5`). Requires `$TMUX`. Errors if no pane matches the change |

---

## fab log

History Logger — append-only JSON logging to `.history.jsonl`. Skills call `fab log command` directly for command invocation logging.

```
fab log command <cmd> [change] [args]
fab log confidence <change> <score> <delta> <trigger>
fab log review <change> <result> [rework]
fab log transition <change> <stage> <action> [from] [reason] [driver]
```

The `command` subcommand accepts `<cmd>` (skill name) as the first argument. `[change]` is optional — when omitted, it resolves the active change via `.fab-status.yaml`. If resolution fails (no `.fab-status.yaml` symlink, dangling symlink), exits 0 silently. When `[change]` IS provided and doesn't resolve, exits 1 with an error.

**Callers**:

| Caller | Trigger | Call |
|--------|---------|------|
| Skills (via `_preamble.md` §2) | Skill invocation (preflight-calling skills) | `fab log command "<skill>" "<change>"` |
| Skills (per-skill instructions) | Skill invocation (exempt skills) | `fab log command "<skill>"` |
| `fab status finish review` | Review pass | auto-logs review "passed" |
| `fab status fail review` | Review fail | auto-logs review "failed" |
| `fab score` | Score computation | auto-logs confidence |
| `fab change new` | Change creation | auto-logs command |
| `fab change rename` | Change rename | auto-logs command |

---

## fab runtime

Runtime State Manager — manages `.fab-runtime.yaml` at the repo root. Used by hooks to track agent idle state per change.

```
fab runtime <subcommand> <change>
```

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `set-idle` | `set-idle <change>` | Write `agent.idle_since` Unix timestamp for the resolved change |
| `clear-idle` | `clear-idle <change>` | Delete the `agent` block for the resolved change (no-op if file missing) |
| `is-idle` | `is-idle <change>` | Check agent idle state. Outputs `idle {duration}`, `active`, or `unknown`. Always exits 0 |

All subcommands accept the standard `<change>` argument (4-char ID, substring, or full folder name). The runtime file is `.fab-runtime.yaml` at the repo root, keyed by the change's full folder name.

---

## fab hook

Claude Code Hook Manager — implements hook logic in Go for Claude Code lifecycle events. Each subcommand is registered as an inline `fab hook <subcommand>` command in `.claude/settings.local.json`. All hook subcommands MUST exit 0 always — errors are silently swallowed to never block the agent.

```
fab hook <subcommand>
```

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `session-start` | `hook session-start` | Clear agent idle state for the active change. Fires on `SessionStart` |
| `stop` | `hook stop` | Write `agent.idle_since` Unix timestamp for the active change. Fires on `Stop` |
| `user-prompt` | `hook user-prompt` | Clear agent idle state for the active change. Fires on `UserPromptSubmit` |
| `artifact-write` | `hook artifact-write` | Read PostToolUse JSON from stdin, perform per-artifact bookkeeping (change type, score, checklist). Fires on `PostToolUse` (Write/Edit) |
| `sync` | `hook sync` | Register inline `fab hook <subcommand>` commands in `.claude/settings.local.json`. Migrates old-style bash script hooks. Idempotent |

**Hook subcommands vs `fab runtime`**: Hook subcommands resolve the active change automatically (no `<change>` argument) and swallow all errors. `fab runtime` subcommands require an explicit `<change>` argument and report errors normally. Hook subcommands use `internal/runtime` and other internal packages directly — no subprocesses.

**`artifact-write` stdin**: Expects Claude Code PostToolUse JSON payload on stdin. Extracts `tool_input.file_path`, matches against fab artifact patterns (`fab/changes/*/intake.md`, `spec.md`, `tasks.md`, `checklist.md`), and performs bookkeeping. Outputs `{"additionalContext":"Bookkeeping: ..."}` JSON on success.

**`sync` output**: Reports one of three statuses to stdout:
- `Created: .claude/settings.local.json hooks (N hook entries)` — fresh settings
- `Updated: .claude/settings.local.json hooks (added N hook entries)` — merged new entries
- `.claude/settings.local.json hooks: OK` — already up to date

---

## fab pane

Tmux pane operations — groups all pane-related subcommands under a single parent command.

```
fab pane <subcommand> [flags...]
```

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `map` | `fab pane map [--json] [--session <name>] [--all-sessions]` | Show tmux pane-to-worktree mapping with pipeline state |
| `capture` | `fab pane capture <pane> [-l N] [--json] [--raw]` | Capture terminal content with fab context enrichment |
| `send` | `fab pane send <pane> <text> [--no-enter] [--force]` | Send keystrokes with pane existence and agent idle validation |
| `process` | `fab pane process <pane> [--json]` | Detect the process tree running in a pane |

---

### fab pane map

Show all tmux panes with pipeline state. Includes all panes regardless of whether they are in a git repo or have a `fab/` directory.

```
fab pane map [--json] [--session <name>] [--all-sessions]
```

#### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--json` | bool | Output as JSON array instead of aligned table |
| `--session <name>` | string | Target a specific tmux session by name (skips `$TMUX` check) |
| `--all-sessions` | bool | Query all tmux sessions (skips `$TMUX` check) |

`--session` and `--all-sessions` are mutually exclusive. When neither is set, discovers panes in the current tmux session only (`-s` session scope) and requires `$TMUX` to be set.

#### Table Output

Produces an aligned table with columns:

| Column | Content | Conditional |
|--------|---------|-------------|
| Session | Tmux session name | Only with `--all-sessions` |
| Pane | Tmux pane ID (e.g., `%3`) | Always |
| WinIdx | Tmux window index (integer) | Always |
| Tab | Tmux window (tab) name | Always |
| Worktree | Relative path from main repo parent, `(main)` for the main worktree, or `basename/` for non-git panes | Always |
| Change | Active change folder name, `(no change)` if no active change, or `---` if not a fab worktree | Always |
| Stage | Current pipeline stage from `.status.yaml`, or `---` if no change or not a fab worktree | Always |
| Agent | Agent state: `active`, `idle ({duration})`, `?` (runtime file missing), or `---` (no change or not fab) | Always |

Idle duration format: `{N}s` (< 60s), `{N}m` (60s-59m), `{N}h` (>= 60m). Floor division.

**Example table output**:

```
Pane   WinIdx  Tab        Worktree                       Change                              Stage     Agent
%3     0       alpha      myrepo.worktrees/alpha/        260306-r3m7-add-retry-logic         apply     active
%7     1       bravo      myrepo.worktrees/bravo/        260306-k8ds-ship-wt-binary          review    idle (2m)
%12    2       main       (main)                         260306-ab12-refactor-auth           hydrate   idle (8m)
%15    3       scratch    downloads/                     ---                                 ---       ---
```

#### JSON Output

When `--json` is set, output is a JSON array. Each element has these fields (snake_case):

| Field | Type | Description |
|-------|------|-------------|
| `session` | string | Tmux session name |
| `window_index` | int | Tmux window index |
| `pane` | string | Tmux pane ID |
| `tab` | string | Tmux window (tab) name |
| `worktree` | string | Display path |
| `change` | string\|null | Active change folder name; `null` for `---` or `(no change)` |
| `stage` | string\|null | Pipeline stage; `null` for `---` |
| `agent_state` | string\|null | `"active"`, `"idle"`, `"unknown"`, or `null` |
| `agent_idle_duration` | string\|null | Duration string (e.g., `"5m"`) when idle; `null` otherwise |

**Error behavior**: If `$TMUX` is unset and neither `--session` nor `--all-sessions` is provided, prints `Error: not inside a tmux session` to stderr and exits 1. If no tmux panes are found, prints `No tmux panes found.` and exits 0.

---

### fab pane capture

Capture terminal content from a tmux pane with fab context enrichment.

```
fab pane capture <pane> [-l N] [--json] [--raw]
```

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `<pane>` | positional | required | Tmux pane ID (e.g., `%5`) |
| `-l` | int | 50 | Number of lines to capture (passed to `tmux capture-pane -l`) |
| `--json` | bool | false | Output as JSON with metadata |
| `--raw` | bool | false | Output raw captured text only (no enrichment) |

`--json` and `--raw` are mutually exclusive.

#### Default Output (human-readable)

Captured text with a header block showing pane metadata (pane ID, worktree, change, stage, agent state).

#### JSON Output

```json
{
  "pane": "%5",
  "lines": 50,
  "content": "...",
  "worktree": "myrepo.worktrees/alpha/",
  "change": "260306-r3m7-add-retry-logic",
  "stage": "apply",
  "agent_state": "idle",
  "agent_idle_duration": "2m"
}
```

Fields `change`, `stage`, `agent_state`, `agent_idle_duration` are `null` when the pane is not in a fab worktree or has no active change.

#### Raw Output

Plain captured text only, identical to raw `tmux capture-pane -p`. No header, no enrichment.

**Error behavior**: If the pane does not exist, prints `Error: pane <id> not found` to stderr and exits 1.

---

### fab pane send

Send keystrokes to a tmux pane with built-in pane existence and agent idle validation.

```
fab pane send <pane> <text> [--no-enter] [--force]
```

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `<pane>` | positional | required | Tmux pane ID (e.g., `%5`) |
| `<text>` | positional | required | Text to send |
| `--no-enter` | bool | false | Don't append Enter keystroke |
| `--force` | bool | false | Skip idle validation (still validates pane existence) |

#### Validation Pipeline

1. **Pane exists**: Checks via `tmux list-panes -a`. If not found, exits 1 with `Error: pane <id> not found` (even with `--force`).
2. **Agent idle**: Resolves pane fab context and checks agent state. Rejects if agent is `active` or `unknown` with `Error: agent in pane <id> is not idle (state: <state>)`. The `--force` flag bypasses this check.
3. **Send keys**: Executes `tmux send-keys -t <pane> "<text>" Enter` (or without `Enter` if `--no-enter`).

**Output on success**: `Sent to <pane>` to stdout.

**Unknown state**: A pane with no active change, no `.fab-runtime.yaml`, or no runtime entry is treated as `unknown` (non-idle). Use `--force` to override.

---

### fab pane process

Detect the process tree running in a tmux pane via OS-level process inspection.

```
fab pane process <pane> [--json]
```

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `<pane>` | positional | required | Tmux pane ID (e.g., `%5`) |
| `--json` | bool | false | Output as JSON |

#### Process Discovery

- **Linux**: Reads `/proc/<pid>/task/<tid>/children` recursively. Reads `/proc/<pid>/comm` and `/proc/<pid>/cmdline` for process details.
- **macOS**: Uses `ps -o pid,ppid,comm -ax` with PPID traversal. Uses `ps -o args= -p <pid>` for full command line.

#### Process Classification

| Comm name | Classification |
|-----------|---------------|
| `claude`, `claude-code` | `agent` |
| `node` | `node` |
| `git`, `gh` | `git` |
| All others | `other` |

#### Default Output (human-readable)

Tree-formatted process listing with PID, command name, and classification.

#### JSON Output

```json
{
  "pane": "%5",
  "pane_pid": 12345,
  "processes": [
    {
      "pid": 12345,
      "ppid": 1,
      "comm": "zsh",
      "cmdline": "/bin/zsh",
      "classification": "other",
      "children": [
        {
          "pid": 12350,
          "ppid": 12345,
          "comm": "claude",
          "cmdline": "claude --dangerously-skip-permissions ...",
          "classification": "agent",
          "children": []
        }
      ]
    }
  ],
  "has_agent": true
}
```

`has_agent` is `true` if any process in the tree is classified as `"agent"`.

**Error behavior**: If the pane does not exist, prints `Error: pane <id> not found` to stderr and exits 1.

---

# Prerequisites & Setup

## fab doctor

Prerequisite Validator — checks that all required tools are installed and configured. Lives in `fab-kit` (not `fab-go`) so it works before `config.yaml` exists. Used by `/fab-setup` as the Phase 0 bootstrap gate.

```
fab doctor [--porcelain]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--porcelain` | bool | Only print errors (no passes, hints, or summary). Useful for scripted callers |

### Checks (7 total)

git, fab, bash, yq (v4+), jq, gh, direnv (with shell hook detection for zsh/bash).

### Output

- Pass: `  ✓ {tool} {version}`
- Fail: `  ✗ {tool} — not found` with install hints
- Summary: `{N}/{total} checks passed. {failures} issues found.`
- Exit code: number of failures (0 = all passed)

### Porcelain mode

When `--porcelain` is set, only error lines are printed. Exit code is the failure count. Empty output + exit 0 means all tools are present.

---

## fab kit-path

Kit Path — prints the absolute path to the resolved kit directory. Used by agents to locate templates, migrations, and other kit content.

```
fab kit-path
```

### Behavior

1. Resolves kit directory from exe-sibling (`kit/` next to the `fab-go` binary)
2. Prints the absolute path to stdout (no trailing newline, no decoration)
3. Exits 0 on success, non-zero with error on stderr if resolution fails

### Usage in Skills

Skills reference kit content via `$(fab kit-path)/templates/`, `$(fab kit-path)/migrations/`, etc. This is agent-agnostic: any agent that can execute a shell command can resolve the kit.

---

## fab fab-help

Workflow Help — scans skill frontmatter from the cache kit, groups commands by category, and renders a formatted overview. The command name is `fab-help` (not overriding cobra's built-in `help`).

```
fab fab-help
```

### Behavior

1. Reads `VERSION` from `.kit/` directory
2. Scans skill files from the cache kit for `name` and `description` frontmatter fields
3. Excludes partials (`_` prefix) and internal skills (`internal-` prefix)
4. Groups skills by a hardcoded mapping into: Start & Navigate, Planning, Completion, Maintenance, Setup, Batch Operations
5. Batch command entries are read dynamically from `fab batch` cobra subcommands
6. Unmapped skills appear under "Other"

### Output sections

- Version header
- Workflow diagram
- Grouped commands with descriptions
- Typical flow examples
- Packages section (wt, idea)

---

## fab operator

Singleton Tmux Tab Launcher — creates a tmux window named "operator" running the configured agent spawn command with `/fab-operator`. If a window named "operator" already exists, switches to it instead.

```
fab operator
```

### Behavior

1. Requires `$TMUX` to be set (exits 1 with error if not in tmux)
2. If a tmux window named "operator" exists, selects it and prints `Switched to existing operator tab.`
3. Otherwise, creates a new tmux window in the repo root running `{spawn_command} '/fab-operator'` and prints `Launched operator.`

### Spawn command resolution

Reads `agent.spawn_command` from `fab/project/config.yaml`. Falls back to `claude --dangerously-skip-permissions` if the key is missing, null, or empty.

---

# Batch Operations

## fab batch

Multi-target batch operations — groups `new`, `switch`, and `archive` subcommands that operate on multiple changes or backlog items at once. All subcommands that create tmux windows require `$TMUX` to be set.

```
fab batch <subcommand> [flags...]
```

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `new` | `new [--list] [--all] [backlog-id...]` | Create worktree tabs from backlog items |
| `switch` | `switch [--list] [--all] [change...]` | Open tmux tabs in worktrees for changes |
| `archive` | `archive [--list] [--all] [change...]` | Archive completed changes in one session |

---

## fab batch new

Create Worktree Tabs from Backlog — parses `fab/backlog.md` for pending items (`- [ ] [xxxx]`), creates git worktrees, opens tmux windows, and starts Claude Code sessions with `/fab-new {description}`.

```
fab batch new [--list] [--all] [backlog-id...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--list` | bool | Show pending backlog items and their IDs |
| `--all` | bool | Open tabs for all pending backlog items |

### Behavior

- No arguments: defaults to `--list`
- With IDs: creates a worktree tab for each specified backlog ID
- With `--all`: creates worktree tabs for all pending items
- Requires `$TMUX` when creating tabs (not for `--list`)
- For each ID: runs `wt create --non-interactive --worktree-name {id}`, opens a tmux window named `fab-{id}`, starts `{spawn_command} '/fab-new {description}'`
- Handles continuation lines in backlog entries (lines starting with whitespace that aren't new list items)

---

## fab batch switch

Open Worktree Tabs for Changes — resolves change names, creates worktrees with branch names (using `branch_prefix` from config if present), and starts Claude Code sessions with `/fab-switch {change}`.

```
fab batch switch [--list] [--all] [change...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--list` | bool | Show available changes |
| `--all` | bool | Open tabs for all changes |

### Behavior

- No arguments: defaults to `--list`
- With change names: resolves each via `fab change resolve`, creates a worktree tab
- With `--all`: opens tabs for all active changes (excludes `archive/`)
- Requires `$TMUX` when creating tabs (not for `--list`)
- Branch naming: `{branch_prefix}{folder_name}` (reads `branch_prefix` from config.yaml)

---

## fab batch archive

Archive Completed Changes — finds changes with `hydrate: done|skipped` in `.status.yaml`, then spawns a single Claude Code session with a prompt to run `/fab-archive` for each eligible change.

```
fab batch archive [--list] [--all] [change...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--list` | bool | Show archivable changes without archiving |
| `--all` | bool | Archive all archivable changes |

### Behavior

- No arguments: defaults to `--all` (unlike new/switch which default to `--list`)
- With change names: resolves each via `fab change resolve`, validates archivability, then archives
- With `--list`: shows changes where hydrate is done or skipped
- Spawns a single Claude session with prompt: `Run /fab-archive for each of these changes, one at a time: {changes}`

---

## Common Error Messages

| Error | Cause | Fix |
|-------|-------|-----|
| "Status file not found: {path}" | Passed a path that doesn't exist as a file | Use a change ID or folder name instead |
| "Cannot resolve change '{arg}'" | Change ID/name doesn't match any folder in `fab/changes/` | Check `fab change list` for available changes |
| "Multiple changes match" | Ambiguous substring matched multiple folders | Use a more specific identifier |
| "No active changes found" | `.fab-status.yaml` symlink is absent and no changes exist | Run `/fab-new` or `/fab-draft` first |
