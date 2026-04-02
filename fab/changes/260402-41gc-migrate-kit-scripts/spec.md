# Spec: Migrate Kit Scripts to Go Binary

**Change**: 260402-41gc-migrate-kit-scripts
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/setup.md`

## Non-Goals

- Removing the `.kit/` folder entirely — this change only eliminates the `scripts/` subdirectory dependency
- Changing any script behavior — all subcommands SHALL be behavioral clones of their shell predecessors
- Modifying the `fab` router's dispatch architecture — `fab-kit` allowlist gets `doctor` added; all other new commands route to `fab-go` as normal

## Routing: Command Placement

### Requirement: `doctor` SHALL be a `fab-kit` subcommand

`fab doctor` MUST work before `config.yaml` exists (it's the Phase 0 gate in `/fab-setup` bootstrap). Since the `fab` router requires `config.yaml` to resolve the `fab-go` version, `doctor` cannot live in `fab-go`. It SHALL be added to `fab-kit` and the router's `fabKitArgs` allowlist.

#### Scenario: Doctor runs before project initialization
- **GIVEN** a repo with no `fab/project/config.yaml`
- **WHEN** the user runs `fab doctor`
- **THEN** the router dispatches to `fab-kit`
- **AND** doctor checks all 7 prerequisites and reports results

#### Scenario: Doctor called by /fab-setup as early gate
- **GIVEN** a fresh repo being bootstrapped
- **WHEN** `/fab-setup` runs `fab doctor`
- **THEN** exit code = number of failures
- **AND** if non-zero, `/fab-setup` stops before creating artifacts

### Requirement: `help`, `operator`, and `batch` SHALL be `fab-go` subcommands

Enhanced `help`, `operator`, and all `batch` subcommands SHALL live in `fab-go`. They require the project to be initialized (config.yaml exists) and are workflow-adjacent commands. The router dispatches them to `fab-go` via the default route (no allowlist change needed).

#### Scenario: Enhanced help inside an initialized repo
- **GIVEN** a repo with `fab/project/config.yaml` and `fab/.kit/skills/`
- **WHEN** the user runs `fab fab-help` (or the `/fab-help` skill calls it)
- **THEN** `fab-go` scans `.kit/skills/` frontmatter, groups commands, and renders formatted output

#### Scenario: Batch command inside an initialized repo
- **GIVEN** a repo with `fab/project/config.yaml`
- **WHEN** the user runs `fab batch switch --all`
- **THEN** `fab-go` handles the command

## Doctor: Prerequisite Validation

### Requirement: `fab doctor` SHALL check 7 prerequisites

The command SHALL check: git, fab, bash, yq (v4+), jq, gh, direnv (with shell hook detection). Each check reports pass/fail with version info. The output format SHALL match the current `fab-doctor.sh` output.

#### Scenario: All prerequisites present
- **GIVEN** all 7 tools are installed and configured
- **WHEN** `fab doctor` runs
- **THEN** each tool shows `✓ {tool} {version}`
- **AND** summary shows `7/7 checks passed.`
- **AND** exit code is 0

#### Scenario: Some prerequisites missing
- **GIVEN** `yq` is not installed and `direnv` hook is not configured
- **WHEN** `fab doctor` runs
- **THEN** missing tools show `✗ {tool} — not found` with install hints
- **AND** summary shows `5/7 checks passed. 2 issues found.`
- **AND** exit code is 2

### Requirement: `fab doctor` SHALL support `--porcelain` flag

When `--porcelain` is set, only error lines are printed (no passes, no hints, no summary). Useful for scripted callers like worktree creation hooks.

#### Scenario: Porcelain mode with failures
- **GIVEN** `jq` is not installed
- **WHEN** `fab doctor --porcelain` runs
- **THEN** output is only `jq — not found`
- **AND** exit code is 1

#### Scenario: Porcelain mode with no failures
- **GIVEN** all tools are present
- **WHEN** `fab doctor --porcelain` runs
- **THEN** output is empty
- **AND** exit code is 0

## Help: Dynamic Skill Discovery

### Requirement: `fab-go` SHALL provide a `fab-help` subcommand

A new `fab-help` subcommand in `fab-go` SHALL scan `fab/.kit/skills/*.md` files, extract `name` and `description` from YAML frontmatter, group them by category, and render formatted output. The command name is `fab-help` (not overriding cobra's built-in `help`).

The group mapping SHALL be a hardcoded map in Go matching the current categories: Start & Navigate, Planning, Completion, Maintenance, Setup, Batch Operations.

Skills with `_` prefix (partials) and `internal-` prefix SHALL be excluded from the listing.

#### Scenario: Skill discovery with grouped output
- **GIVEN** `fab/.kit/skills/` contains `fab-new.md`, `fab-continue.md`, `fab-archive.md`, etc.
- **WHEN** `fab fab-help` runs
- **THEN** output shows version header, workflow diagram, grouped commands with descriptions, typical flow, and packages section
- **AND** skills without a group mapping appear under "Other"

### Requirement: `fab-help` SHALL discover batch commands from `fab batch`

Instead of scanning `batch-*.sh` scripts (which will be deleted), the batch commands section SHALL list `fab batch new`, `fab batch switch`, `fab batch archive` with their descriptions from cobra command metadata.

#### Scenario: Batch commands in help output
- **GIVEN** `fab batch` subcommands are registered
- **WHEN** `fab fab-help` runs
- **THEN** the "Batch Operations" group shows `fab batch new`, `fab batch switch`, `fab batch archive` with descriptions

### Requirement: Frontmatter parsing SHALL handle markdown YAML frontmatter

The Go implementation SHALL extract fields from YAML frontmatter (content between `---` markers at the start of a file). It SHALL handle quoted and unquoted values, strip inline comments, and return empty string for missing fields.

#### Scenario: Parse frontmatter from a skill file
- **GIVEN** a file starting with `---\nname: fab-new\ndescription: "Start a new change"\n---`
- **WHEN** the frontmatter parser extracts `name` and `description`
- **THEN** it returns `fab-new` and `Start a new change`

## Operator: Singleton Tmux Launcher

### Requirement: `fab operator` SHALL launch a singleton tmux tab

The command SHALL create a tmux window named "operator" running the configured `agent.spawn_command` with `/fab-operator`. If a window named "operator" already exists in the current session, it SHALL switch to it instead.

#### Scenario: First launch
- **GIVEN** no tmux window named "operator" exists
- **WHEN** `fab operator` runs
- **THEN** a new tmux window is created named "operator" in the repo root
- **AND** the window runs `{spawn_command} '/fab-operator'`
- **AND** output is `Launched operator.`

#### Scenario: Already running
- **GIVEN** a tmux window named "operator" exists
- **WHEN** `fab operator` runs
- **THEN** the existing window is selected
- **AND** output is `Switched to existing operator tab.`

#### Scenario: Not in tmux
- **GIVEN** `$TMUX` is not set
- **WHEN** `fab operator` runs
- **THEN** it prints `Error: not inside a tmux session.` to stderr and exits 1

### Requirement: Spawn command SHALL be read from config.yaml

The command SHALL read `agent.spawn_command` from `fab/project/config.yaml`. If the key is missing, null, or empty, it SHALL fall back to `claude --dangerously-skip-permissions`.

#### Scenario: Custom spawn command
- **GIVEN** `config.yaml` contains `agent: { spawn_command: "claude --model opus" }`
- **WHEN** `fab operator` launches
- **THEN** the tmux window runs `claude --model opus '/fab-operator'`

#### Scenario: No spawn command configured
- **GIVEN** `config.yaml` has no `agent.spawn_command` key
- **WHEN** `fab operator` launches
- **THEN** the tmux window runs `claude --dangerously-skip-permissions '/fab-operator'`

## Batch: Multi-Target Operations

### Requirement: `fab batch new` SHALL create worktree tabs from backlog items

Per backlog ID: creates a git worktree via `wt create --non-interactive --worktree-name {id}`, opens a tmux window, starts a Claude Code session running `/fab-new {description}`. Backlog parsing SHALL handle continuation lines (lines starting with whitespace that aren't new list items).

#### Scenario: Create tabs for specific backlog IDs
- **GIVEN** `fab/backlog.md` contains pending items `[90g5]` and `[jgt6]`
- **WHEN** `fab batch new 90g5 jgt6` runs
- **THEN** for each ID: worktree is created, tmux window opens, Claude session starts with `/fab-new {extracted description}`

#### Scenario: List pending items
- **GIVEN** `fab/backlog.md` has 3 pending (`[ ]`) and 2 done (`[x]`) items
- **WHEN** `fab batch new --list` runs
- **THEN** output shows only the 3 pending items with IDs and descriptions

#### Scenario: All pending items
- **GIVEN** `fab/backlog.md` has 3 pending items
- **WHEN** `fab batch new --all` runs
- **THEN** all 3 pending items get worktree tabs

#### Scenario: No arguments shows list
- **GIVEN** `fab/backlog.md` exists
- **WHEN** `fab batch new` runs with no arguments
- **THEN** output is the same as `--list`

#### Scenario: Not in tmux
- **GIVEN** `$TMUX` is not set
- **WHEN** `fab batch new 90g5` runs
- **THEN** it prints `Error: not inside a tmux session` to stderr and exits 1

### Requirement: `fab batch switch` SHALL open worktree tabs for changes

Per change: resolves via `fab change resolve`, creates a worktree with the correct branch name (using `branch_prefix` from config if present), opens a tmux window, starts a Claude session running `/fab-switch {change}`.

#### Scenario: Switch specific changes
- **GIVEN** changes `r7k3` and `ab12` exist
- **WHEN** `fab batch switch r7k3 ab12` runs
- **THEN** for each: worktree is created with branch `{branch_prefix}{folder_name}`, tmux window opens, Claude session starts

#### Scenario: List available changes
- **GIVEN** 3 active changes exist (excluding archive/)
- **WHEN** `fab batch switch --list` runs
- **THEN** output lists all 3 change folder names

#### Scenario: All changes
- **GIVEN** 3 active changes exist
- **WHEN** `fab batch switch --all` runs
- **THEN** all 3 get worktree tabs

### Requirement: `fab batch archive` SHALL archive completed changes

Finds changes with `hydrate: done|skipped` in `.status.yaml`, spawns a single Claude Code session with a prompt to run `/fab-archive` for each. Default (no args) archives all eligible.

#### Scenario: Archive all eligible
- **GIVEN** 2 changes have `hydrate: done` and 1 has `hydrate: pending`
- **WHEN** `fab batch archive` runs (or `fab batch archive --all`)
- **THEN** a single Claude session is spawned with prompt to archive the 2 eligible changes

#### Scenario: Preview eligible changes
- **GIVEN** 2 changes are archivable
- **WHEN** `fab batch archive --list` runs
- **THEN** output shows the 2 archivable change names

#### Scenario: Specific changes
- **GIVEN** change `v3rn` has `hydrate: done`
- **WHEN** `fab batch archive v3rn` runs
- **THEN** `v3rn` is resolved via `fab change resolve`, validated as archivable, and archived

## Shared Internals

### Requirement: Spawn command resolution SHALL be a shared Go function

An internal Go function SHALL read `agent.spawn_command` from `fab/project/config.yaml` and fall back to `claude --dangerously-skip-permissions`. This replaces `lib/spawn.sh` and is used by `operator` and all `batch` subcommands.

#### Scenario: Config with spawn command
- **GIVEN** `config.yaml` has `agent: { spawn_command: "custom-claude" }`
- **WHEN** the spawn command function is called
- **THEN** it returns `custom-claude`

#### Scenario: Config without spawn command
- **GIVEN** `config.yaml` exists but has no `agent` section
- **WHEN** the spawn command function is called
- **THEN** it returns `claude --dangerously-skip-permissions`

## Script Deletion and Reference Updates

### Requirement: `fab/.kit/scripts/` directory SHALL be deleted after migration

Once all commands are implemented and verified, the entire `fab/.kit/scripts/` directory (6 scripts + `lib/` with 2 files) SHALL be deleted.

#### Scenario: Clean deletion
- **GIVEN** all 6 Go subcommands are implemented and tested
- **WHEN** the scripts directory is deleted
- **THEN** no remaining code references `fab/.kit/scripts/`

### Requirement: Skill references SHALL be updated

- `/fab-setup` skill: `fab/.kit/scripts/fab-doctor.sh` → `fab doctor`
- `/fab-help` skill: `bash fab/.kit/scripts/fab-help.sh` → `fab fab-help`
- `/fab-operator` skill: `fab/.kit/scripts/fab-operator.sh` → `fab operator`

#### Scenario: /fab-setup calls fab doctor
- **GIVEN** the `/fab-setup` skill has been updated
- **WHEN** it runs the doctor early gate
- **THEN** it calls `fab doctor` (not a shell script path)

## Deprecated Requirements

### Shell Script Execution of Kit Scripts
**Reason**: All scripts absorbed into compiled Go binaries (`fab-kit` for doctor, `fab-go` for all others). Shell scripts with `yq`/`sed` dependencies replaced by native Go implementations.
**Migration**: Replace script paths with corresponding `fab` subcommands.

## Design Decisions

1. **`doctor` in `fab-kit`, everything else in `fab-go`**
   - *Why*: Doctor must work before `config.yaml` exists (Phase 0 bootstrap gate). The `fab` router requires `config.yaml` to resolve the `fab-go` version, so commands that run pre-initialization must live in `fab-kit`.
   - *Rejected*: Doctor in `fab-go` — would fail on fresh repos where `/fab-setup` hasn't run yet.

2. **`fab-help` as distinct subcommand, not overriding cobra `help`**
   - *Why*: Cobra's built-in `help` command is deeply integrated and overriding it creates conflicts with `--help` flag handling. A separate `fab-help` subcommand is clean and matches the skill name.
   - *Rejected*: Overriding cobra's `help` — fragile, interferes with cobra internals. Also rejected: enhancing the router's `printHelp()` — would require duplicating skill scanning logic in `fab-kit`.

3. **Dynamic skill scanning from `.kit/skills/` for help**
   - *Why*: Skills change over time; hardcoding descriptions couples help to binary releases. Dynamic scanning matches the current `fab-help.sh` approach and keeps help accurate without binary updates.
   - *Rejected*: Embedded skill list — requires binary update on every skill change.

4. **`fab batch` command group**
   - *Why*: Three batch operations share common patterns (tmux tab creation, spawn command, `--list`/`--all` flags). Grouping under `batch` provides clean namespace and discoverability.
   - *Rejected*: Flat namespace (`fab batch-new`, `fab batch-switch`) — less organized, clutters top-level.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All 6 scripts migrate to Go subcommands | Confirmed from intake #1 — user agreed | S:95 R:80 A:90 D:95 |
| 2 | Certain | Batch scripts grouped under `fab batch` | Confirmed from intake #2 — natural grouping | S:85 R:85 A:85 D:90 |
| 3 | Certain | `lib/spawn.sh` and `lib/frontmatter.sh` absorbed as Go internals | Confirmed from intake #3 — sole consumers are migrating scripts | S:90 R:90 A:95 D:95 |
| 4 | Certain | Entire `scripts/` directory deleted after migration | Confirmed from intake #4 — explicit goal | S:90 R:70 A:90 D:90 |
| 5 | Certain | `fab doctor` lives in `fab-kit` (not `fab-go`) | Resolved — must work before config.yaml exists for /fab-setup Phase 0 | S:95 R:75 A:95 D:95 |
| 6 | Certain | Enhanced help is `fab-help` subcommand in `fab-go` with dynamic skill scanning | Resolved — cobra `help` override is fragile; dynamic scanning matches current behavior | S:90 R:65 A:85 D:85 |
| 7 | Certain | Spawn command fallback is `claude --dangerously-skip-permissions` | Confirmed from intake #7 — matches current behavior | S:80 R:85 A:80 D:85 |
| 8 | Certain | Batch/operator commands shell out to `wt` and `tmux` via exec.Command | Confirmed from intake #8 — pattern already used in panemap.go | S:75 R:80 A:85 D:80 |

8 assumptions (8 certain, 0 confident, 0 tentative, 0 unresolved).
