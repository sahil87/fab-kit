# Spec: Add `fab pane-map` Subcommand

**Change**: 260306-bh45-pane-map-subcommand
**Created**: 2026-03-06
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- `--json` flag for machine-readable output — deferred to a future iteration when the conductor skill needs it
- `--watch` flag for dashboard refresh — deferred to a future iteration
- Orphan change discovery — only panes visible in tmux are shown; changes with no pane are not listed
- Process inspection — pane-map does not inspect running processes; agent state comes solely from `.fab-runtime.yaml`

## CLI: Pane Map Subcommand

### Requirement: Subcommand Registration

The Go binary SHALL register `pane-map` as a top-level Cobra subcommand. The command SHALL have `Use: "pane-map"` and a `Short` description. It SHALL accept no positional arguments.

#### Scenario: Command appears in help

- **GIVEN** the fab binary is installed
- **WHEN** the user runs `fab --help`
- **THEN** `pane-map` appears in the list of available commands

#### Scenario: Command runs without arguments

- **GIVEN** the user is inside a tmux session
- **WHEN** the user runs `fab pane-map`
- **THEN** the command outputs the pane map table and exits 0

### Requirement: Tmux Pane Discovery

The command SHALL discover all tmux panes by executing `tmux list-panes -a -F '#{pane_id} #{pane_current_path}'`. Each line of output provides a pane ID (e.g., `%3`) and the pane's current working directory.

#### Scenario: Multiple panes in session

- **GIVEN** a tmux session with 3 panes, each in a different directory
- **WHEN** `fab pane-map` runs
- **THEN** all 3 panes are evaluated for worktree membership

#### Scenario: Pane CWD is a subdirectory of a worktree

- **GIVEN** a pane whose CWD is `/home/user/repo.worktrees/alpha/src/lib`
- **WHEN** `fab pane-map` resolves the pane
- **THEN** it identifies the worktree root at `/home/user/repo.worktrees/alpha/` via `git -C <path> rev-parse --show-toplevel`

### Requirement: Worktree Resolution

For each pane CWD, the command SHALL run `git -C <path> rev-parse --show-toplevel` to identify the git worktree root. Panes where this command fails (not inside a git repo) SHALL be excluded from output. The command SHALL then check `fab/current` in each worktree to find the active change name, and read `.status.yaml` for stage information.

#### Scenario: Pane is a fab worktree with active change

- **GIVEN** a pane whose CWD resolves to worktree root `/home/user/repo.worktrees/alpha/`
- **AND** `fab/current` contains a valid change folder name
- **WHEN** the pane is resolved
- **THEN** the Change column shows the folder name and the Stage column shows the current stage

#### Scenario: Pane is a git worktree without fab/current

- **GIVEN** a pane whose CWD resolves to a git worktree root
- **AND** `fab/current` is missing or empty
- **WHEN** the pane is resolved
- **THEN** the Change column shows `(no change)` and Stage shows `—`

#### Scenario: Pane is not inside a git repo

- **GIVEN** a pane whose CWD is `/tmp/scratch`
- **WHEN** `git rev-parse --show-toplevel` fails
- **THEN** the pane is excluded from the output entirely

### Requirement: Main Worktree Inclusion

The main repo worktree (not a `.worktrees/` child) SHALL appear in the map when a tmux pane is inside it. Its Worktree column SHALL display `(main)`.

#### Scenario: Main worktree with active change

- **GIVEN** a pane whose CWD resolves to the main repo root
- **AND** `fab/current` contains a valid change folder name
- **WHEN** `fab pane-map` runs
- **THEN** the row shows `(main)` in the Worktree column with the change name, stage, and agent state

### Requirement: Runtime State Correlation

The command SHALL read `.fab-runtime.yaml` from each worktree's root directory to determine agent idle state. For each change shown, the command SHALL look up the change's folder name as a key in the runtime file and read `agent.idle_since`. If `idle_since` is present, the Agent column SHALL show `idle ({duration})` where `{duration}` is the elapsed time in human-readable form. If no `agent` block exists for the change, the Agent column SHALL show `active`.

#### Scenario: Agent is idle

- **GIVEN** `.fab-runtime.yaml` contains `{change_folder: {agent: {idle_since: <timestamp>}}}` where timestamp is 300 seconds ago
- **WHEN** the pane is rendered
- **THEN** the Agent column shows `idle (5m)`

#### Scenario: Agent is active (no idle_since)

- **GIVEN** `.fab-runtime.yaml` exists but has no `agent` block for this change
- **WHEN** the pane is rendered
- **THEN** the Agent column shows `active`

#### Scenario: Runtime file missing

- **GIVEN** `.fab-runtime.yaml` does not exist in the worktree root
- **WHEN** the pane is rendered
- **THEN** the Agent column shows `?`

### Requirement: Human-Readable Idle Duration

The idle duration SHALL be formatted as the largest appropriate unit:
- `< 60s` → `{N}s` (e.g., `30s`)
- `60s–59m` → `{N}m` (e.g., `5m`)
- `≥ 60m` → `{N}h` (e.g., `2h`)

Values are rounded down (floor).

#### Scenario: Duration formatting

- **GIVEN** `idle_since` is 45 seconds ago
- **WHEN** rendered
- **THEN** shows `idle (45s)`

- **GIVEN** `idle_since` is 125 seconds ago
- **WHEN** rendered
- **THEN** shows `idle (2m)`

- **GIVEN** `idle_since` is 7500 seconds ago
- **WHEN** rendered
- **THEN** shows `idle (2h)`

### Requirement: Worktree Path Display

Worktree paths SHALL be displayed relative to the main repo's parent directory. For the main worktree, `(main)` is used instead. For child worktrees, show the path relative to the parent (e.g., `.worktrees/alpha/` or `repo.worktrees/alpha/`).

#### Scenario: Relative path display

- **GIVEN** the main worktree is at `/home/user/myrepo`
- **AND** a child worktree is at `/home/user/myrepo.worktrees/alpha`
- **WHEN** the table is rendered
- **THEN** the child's Worktree column shows `myrepo.worktrees/alpha/`

### Requirement: Output Format

The command SHALL produce an aligned table with columns: `Pane`, `Worktree`, `Change`, `Stage`, `Agent`. Column widths SHALL be dynamically computed from the widest value in each column. A header row SHALL be printed first.

```
Pane   Worktree                       Change                              Stage     Agent
%3     myrepo.worktrees/alpha/        260306-r3m7-add-retry-logic         apply     active
%7     myrepo.worktrees/bravo/        260306-k8ds-ship-wt-binary          review    idle (2m)
%12    (main)                         260306-ab12-refactor-auth           hydrate   idle (8m)
```

#### Scenario: Empty result (no fab worktrees in tmux)

- **GIVEN** all tmux panes are in non-git or non-fab directories
- **WHEN** `fab pane-map` runs
- **THEN** the command prints `No fab worktrees found in tmux panes.` and exits 0

### Requirement: Non-Fab Pane Exclusion

Panes that are not inside a git repository, or that are in a git repo without a `fab/` directory, SHALL be excluded from the output.

#### Scenario: Plain shell pane excluded

- **GIVEN** a pane in `/home/user`
- **WHEN** `fab pane-map` runs
- **THEN** this pane does not appear in the output

### Requirement: Tmux Session Guard

If the `$TMUX` environment variable is not set, the command SHALL print `Error: not inside a tmux session` to stderr and exit with code 1.

#### Scenario: Not in tmux

- **GIVEN** the user is not inside a tmux session (`$TMUX` is unset)
- **WHEN** `fab pane-map` runs
- **THEN** stderr shows `Error: not inside a tmux session` and exit code is 1

### Requirement: Deduplication

When multiple panes share the same worktree (e.g., two panes both `cd`'d into the same worktree), each pane SHALL appear as a separate row. The same worktree MAY appear in multiple rows.

#### Scenario: Two panes in same worktree

- **GIVEN** panes `%3` and `%5` both have CWDs inside the same worktree
- **WHEN** `fab pane-map` runs
- **THEN** both panes appear as separate rows with the same Worktree and Change values

## Design Decisions

1. **Runtime file only — no process inspection**: Agent state comes from `.fab-runtime.yaml` set by Claude Code hooks, not from inspecting processes in panes.
   - *Why*: Runtime file is the existing mechanism, already maintained by hooks. Process inspection would be fragile (different shell wrappers, no standard process name).
   - *Rejected*: Checking for `claude` process in pane TTY — unreliable and platform-dependent.

2. **No orphan discovery**: Only tmux-visible panes are shown. Changes with no pane are not listed.
   - *Why*: Simplifies implementation. The command answers "what's running in my tmux?" not "what changes exist?". `fab status show --all` already covers the latter.
   - *Rejected*: Scanning all worktree `fab/changes/` — requires cross-worktree traversal, mixes observation with inventory.

3. **Command file with inline logic**: The pane-map logic lives in a single command file (`src/go/fab/cmd/fab/panemap.go`) rather than an internal package.
   - *Why*: The logic is self-contained and specific to this command. No other command needs tmux pane discovery. Follows the pattern of `runtime.go`.
   - *Rejected*: Creating `internal/panemap/` package — over-engineering for a single consumer.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Subcommand lives in Go binary as `fab pane-map` | Confirmed from intake #1 — consistent with existing subcommands | S:95 R:90 A:95 D:95 |
| 2 | Certain | Output is a formatted table with Pane, Worktree, Change, Stage, Agent columns | Confirmed from intake #2 — specific format agreed | S:95 R:85 A:90 D:95 |
| 3 | Certain | No orphan discovery — only tmux panes shown | Confirmed from intake #3 (clarified) — user chose option B | S:95 R:90 A:90 D:95 |
| 4 | Certain | Worktree paths shown relative to repo parent | Confirmed from intake #4 | S:95 R:85 A:80 D:80 |
| 5 | Certain | Idle duration as human-readable relative time (Ns/Nm/Nh) | Confirmed from intake #5 | S:95 R:90 A:85 D:80 |
| 6 | Certain | Non-fab panes excluded from output | Confirmed from intake #6 | S:95 R:85 A:75 D:70 |
| 7 | Certain | Tmux required — graceful error if not in tmux session | Confirmed from intake #7 | S:95 R:90 A:70 D:85 |
| 8 | Certain | `--json` and `--watch` deferred to future iteration | Confirmed from intake #8 — upgraded from Tentative since scope is locked | S:95 R:85 A:65 D:60 |
| 9 | Certain | Agent state from `.fab-runtime.yaml` only — no process inspection | Clarified in intake — user confirmed recommended approach | S:95 R:90 A:85 D:90 |
| 10 | Certain | Main worktree included, shown as `(main)` | Clarified in intake — user confirmed inclusion | S:95 R:85 A:90 D:90 |
| 11 | Certain | Multiple panes in same worktree shown as separate rows | New — natural consequence of pane-centric view, no dedup needed | S:85 R:90 A:90 D:95 |
| 12 | Certain | Implementation in single command file, no internal package | Follows runtime.go pattern — self-contained, single consumer | S:80 R:90 A:85 D:80 |

12 assumptions (12 certain, 0 confident, 0 tentative, 0 unresolved).
