# Intake: Migrate Kit Scripts to Go Binary

**Change**: 260402-41gc-migrate-kit-scripts
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Backlog item [41gc]: Next step in removing .kit - remove dependency on .kit/scripts folder

Preceded by a detailed `/fab-discuss` session where all 6 user-facing scripts and 2 lib files in `fab/.kit/scripts/` were analyzed individually, with responsibilities mapped and migration targets agreed upon.

## Why

The project is working toward removing the `.kit/` folder from deployed repos. The `fab/.kit/scripts/` directory contains 6 user-facing shell scripts and 2 shared libraries that represent a runtime dependency on `.kit/`. These scripts must be absorbed into the `fab` Go binary (which is already installed system-wide via Homebrew) before `.kit/` can be removed.

If not done: the `.kit/` folder cannot be removed from deployed repos, blocking the larger distribution simplification initiative. Users continue depending on shell scripts that require `yq`, `sed`, and other tools that the Go binary can handle natively.

## What Changes

### 1. `fab doctor` subcommand (from `fab-doctor.sh`)

Absorb the prerequisite checker into the Go binary as `fab doctor`. Checks 7 tools: git, fab, bash, yq v4+, jq, gh, direnv (with shell hook detection). Supports `--porcelain` flag for scripted callers. Exit code = failure count.

Currently called by `/fab-setup` skill as `fab/.kit/scripts/fab-doctor.sh`. After migration, the skill calls `fab doctor` instead.

### 2. Enhanced `fab help` (from `fab-help.sh`)

The current `fab help` shows cobra's default help. Replace it with the dynamic skill discovery from `fab-help.sh`: scan `fab/.kit/skills/*.md` frontmatter for names/descriptions, group them into categories (Start & Navigate, Planning, Completion, Maintenance, Setup, Batch Operations), and render the formatted output including workflow diagrams and typical flow examples.

The group mapping (which skill belongs to which group) can be maintained as a hardcoded map in Go, matching the current shell script's `skill_to_group` associative array.

Frontmatter parsing (`lib/frontmatter.sh`) is absorbed as Go code — extract `name` and `description` fields from YAML frontmatter between `---` markers in markdown files.

### 3. `fab operator` subcommand (from `fab-operator.sh`)

Singleton tmux tab launcher. Creates a tmux window named "operator" running the configured `agent.spawn_command` from `config.yaml` with `/fab-operator`. If the tab already exists, switches to it. Requires active tmux session.

### 4. `fab batch new` subcommand (from `batch-fab-new-backlog.sh`)

Per backlog ID: creates a git worktree (via `wt create`), opens a tmux tab, starts a Claude Code session running `/fab-new <description>`. Parses `fab/backlog.md` to extract item descriptions (handling continuation lines). Supports `--list` (show pending), `--all` (all pending), and positional ID arguments.

### 5. `fab batch switch` subcommand (from `batch-fab-switch-change.sh`)

Per change name/ID: creates a worktree with the correct branch name (reading `branch_prefix` from config), opens a tmux tab, runs `/fab-switch <change>`. Uses `fab change resolve` for name resolution. Supports `--list`, `--all`, and positional arguments.

### 6. `fab batch archive` subcommand (from `batch-fab-archive-change.sh`)

Finds changes with `hydrate: done|skipped` in `.status.yaml`, spawns a single Claude Code session to run `/fab-archive` for each. Supports `--list` (preview), `--all` (default when no args), and positional change arguments. Uses `fab change resolve` for name resolution.

### 7. Shared internals absorbed into Go

- **`lib/spawn.sh`** — `fab_spawn_cmd()` reads `agent.spawn_command` from `config.yaml`, falls back to `claude --dangerously-skip-permissions`. Becomes an internal Go function used by operator and all batch subcommands.
- **`lib/frontmatter.sh`** — YAML and shell-comment frontmatter parsing via `sed`. Becomes Go code for the enhanced `fab help`.

### 8. Script deletion

Once all 6 scripts and 2 lib files are migrated and verified, the entire `fab/.kit/scripts/` directory is deleted. References in skills, memory files, and README are updated.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update scripts/ directory listing, remove shell script descriptions, add new Go subcommand descriptions
- `fab-workflow/execution-skills`: (modify) Update operator launcher references from `fab-operator.sh` to `fab operator`
- `fab-workflow/setup`: (modify) Update `/fab-setup` doctor reference from shell script path to `fab doctor`

## Impact

- **Skills affected**: `/fab-setup` (doctor path), `/fab-help` (script path), `/fab-operator` (launcher path)
- **README.md**: Script reference table needs updating
- **Go binary (`fab-go`)**: 6 new subcommands added (`doctor`, `operator`, `batch new`, `batch switch`, `batch archive`, plus enhanced `help`)
- **`fab` router**: `help` currently routes to cobra default — may need routing adjustment for the enhanced help
- **Dependencies**: All batch scripts depend on `wt` (worktree tool) being on PATH and `tmux` being active — these become Go `exec.Command` calls

## Open Questions

- Should `fab doctor` live in `fab-go` (workflow engine) or `fab-kit` (workspace lifecycle)? Doctor is a setup-time tool, which suggests `fab-kit`, but it's also used at runtime by `/fab-setup`.
- Should `fab help` scan skills from the `.kit/skills/` directory (dynamic, requires .kit to still exist for help) or should the skill list be embedded/configured elsewhere?
<!-- assumed: fab help scans .kit/skills/ — during the transition period .kit still exists for skill content; post-.kit-removal the skill registry moves elsewhere -->

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All 6 scripts migrate to Go subcommands | Discussed — user agreed with all migration targets | S:95 R:80 A:90 D:95 |
| 2 | Certain | Batch scripts grouped under `fab batch` command group | Discussed — natural grouping for `batch new`, `batch switch`, `batch archive` | S:85 R:85 A:85 D:90 |
| 3 | Certain | `lib/spawn.sh` and `lib/frontmatter.sh` absorbed as Go internals | Discussed — only consumers are the migrating scripts | S:90 R:90 A:95 D:95 |
| 4 | Certain | Entire `scripts/` directory deletable after migration | Discussed — user confirmed this is the goal | S:90 R:70 A:90 D:90 |
| 5 | Confident | `fab doctor` becomes a subcommand of the Go binary | Strong signal, but open question about fab-go vs fab-kit placement | S:80 R:75 A:70 D:70 |
| 6 | Confident | `fab help` uses dynamic skill discovery from .kit/skills/ | Current script does this; but post-.kit-removal needs a different registry | S:70 R:60 A:65 D:60 |
| 7 | Confident | Spawn command fallback remains `claude --dangerously-skip-permissions` | Matches current `lib/spawn.sh` behavior | S:80 R:85 A:80 D:85 |
| 8 | Confident | Batch scripts shell out to `wt` and `tmux` via exec.Command | Go binary already uses exec for similar operations | S:75 R:80 A:85 D:80 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
