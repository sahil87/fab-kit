# Spec: Worktree Status Command

**Change**: 260305-7zq4-worktree-status-command
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Modifying `wt-list` behavior — it remains a generic git worktree tool
- JSON output — can be added later; this change focuses on human-readable output
- Showing git status (dirty/unpushed) — `wt-list` already covers that

## wt Package: wt-status Command

### Requirement: Single Worktree Status (Atomic Unit)

The `wt-status` command SHALL implement a core function `wt_get_fab_status` that resolves the fab pipeline status for a single worktree path. This function is the composable building block — the `--all` mode loops it over every worktree.

The function SHALL:
1. Check if `{wt_path}/fab/current` exists and is non-empty
2. If yes, read line 2 (folder name) and construct the `.status.yaml` path: `{wt_path}/fab/changes/{folder}/.status.yaml`
3. If the `.status.yaml` file exists, extract stage and state via `statusman.sh display-stage`
4. Return the resolved status (change name, stage, state) or an appropriate fallback label

#### Scenario: Worktree with active fab change
- **GIVEN** a worktree at path `/repo.worktrees/7zq4/`
- **AND** `fab/current` contains `7zq4\n260305-7zq4-worktree-status-command`
- **AND** `fab/changes/260305-7zq4-worktree-status-command/.status.yaml` exists
- **WHEN** `wt_get_fab_status` is called with the worktree path
- **THEN** it returns the change folder name, display stage, and display state

#### Scenario: Worktree with no fab directory
- **GIVEN** a worktree at path `/repo.worktrees/swift-fox/`
- **AND** `fab/` directory does not exist in the worktree
- **WHEN** `wt_get_fab_status` is called with the worktree path
- **THEN** it returns a fallback indicating no fab project (e.g., "no fab")

#### Scenario: Worktree with fab but no active change
- **GIVEN** a worktree with `fab/project/config.yaml` present
- **AND** `fab/current` does not exist or is empty
- **WHEN** `wt_get_fab_status` is called
- **THEN** it returns a fallback indicating no active change (e.g., "no change")

#### Scenario: Worktree with stale fab/current pointer
- **GIVEN** `fab/current` references a change folder that no longer exists
- **WHEN** `wt_get_fab_status` is called
- **THEN** it returns a fallback indicating a stale pointer (e.g., "stale")

### Requirement: Default Invocation (Current Worktree)

When invoked with no arguments, `wt-status` SHALL display the fab status of the current worktree only.

The output SHALL include:
- The worktree name (basename of `pwd -P`, or `(main)` for the main repo)
- The active change name (full folder name)
- The display stage and state from `statusman.sh display-stage`

#### Scenario: No arguments from within a worktree
- **GIVEN** the user is in worktree `7zq4`
- **AND** the worktree has an active fab change at intake:ready
- **WHEN** `wt-status` is run with no arguments
- **THEN** the output shows the worktree name, change name, stage, and state for that single worktree

#### Scenario: No arguments from main repo
- **GIVEN** the user is in the main repository (not a worktree)
- **AND** the main repo has an active fab change
- **WHEN** `wt-status` is run
- **THEN** it displays status for the main repo, labeled as `(main)`

### Requirement: Named Worktree Status

When invoked with a positional argument `<name>`, `wt-status` SHALL display the fab status of the named worktree. The name is matched against worktree basenames using `wt_get_worktree_path_by_name` from `wt-common.sh`.

#### Scenario: Valid worktree name
- **GIVEN** worktree `swift-fox` exists
- **WHEN** `wt-status swift-fox` is run
- **THEN** the output shows the fab status for `swift-fox`

#### Scenario: Invalid worktree name
- **GIVEN** no worktree named `nonexistent` exists
- **WHEN** `wt-status nonexistent` is run
- **THEN** an error is displayed: "Worktree 'nonexistent' not found."

### Requirement: All Worktrees Mode

When invoked with `--all`, `wt-status` SHALL iterate over all worktrees (including the main repo) and display fab status for each one in a formatted table.

The output SHALL:
- Show a header with the repo name and worktrees directory
- Mark the current worktree with a green `*` prefix
- Show `(main)` for the main repository entry
- Align columns for readability
- Show a total count at the bottom

#### Scenario: Multiple worktrees with mixed states
- **GIVEN** 3 worktrees exist plus the main repo
- **AND** worktree A has intake:ready, worktree B has review:active, main has no active change
- **WHEN** `wt-status --all` is run
- **THEN** all 4 entries are displayed with their respective fab statuses

#### Scenario: No worktrees exist (main repo only)
- **GIVEN** only the main repo exists (no worktrees)
- **WHEN** `wt-status --all` is run
- **THEN** only the main repo entry is shown

### Requirement: Output Format

Single worktree output (default and `<name>` modes) SHALL use a compact format:

```
{name}  {change_name}  {stage}  {state}
```

Where `{change_name}` is the full folder name from `fab/current` line 2, or a fallback label (`(no fab)`, `(no change)`, `(stale)`).

The `--all` mode SHALL use the same column format with a header and footer:

```
Worktrees for: {repo_name}
Location: {worktrees_dir}

* {name}       {change_name}                    {stage}    {state}
  {name}       {change_name}                    {stage}    {state}
  (main)       (no change)

Total: {N} worktree(s)
```

#### Scenario: Fallback labels render correctly
- **GIVEN** three worktrees: one with active change, one with no fab, one with no active change
- **WHEN** `wt-status --all` is run
- **THEN** active change shows `{folder} {stage} {state}`, no-fab shows `(no fab)`, no-change shows `(no change)`

### Requirement: Script Infrastructure

`wt-status` SHALL:
- Be placed at `fab/.kit/packages/wt/bin/wt-status`
- Source `wt-common.sh` for shared helpers (`wt_get_repo_context`, `wt_list_worktrees`, `wt_get_worktree_path_by_name`, color constants)
- Use `statusman.sh display-stage` for stage/state extraction (passing the `.status.yaml` path directly)
- Use `set -euo pipefail`
- Include a `help` subcommand following the pattern of other `wt-*` commands
- Use `wt_validate_git_repo` as the first validation step

#### Scenario: Help output
- **GIVEN** the user runs `wt-status help`
- **WHEN** the command processes the argument
- **THEN** usage information is displayed including examples for no-args, `<name>`, and `--all` modes

### Requirement: statusman.sh Invocation from Worktree Context

When extracting stage/state for a worktree, `wt-status` SHALL invoke `statusman.sh display-stage` with the resolved `.status.yaml` path. Since `statusman.sh display-stage` accepts a `<change>` argument resolved by `resolve.sh`, and `resolve.sh` operates relative to `fab/changes/` in the current directory, the script MUST either:

- (A) `cd` into the worktree before calling statusman, or
- (B) Pass the `.status.yaml` path directly if statusman accepts it

Since `resolve_to_status` in `statusman.sh` accepts absolute file paths directly (backward-compat codepath: `if [ -f "$arg" ]`), the command SHALL pass the absolute `.status.yaml` path to `statusman.sh display-stage` — no `cd` into the worktree is needed.

## Design Decisions

1. **Standalone command vs. wt-list flag**: Standalone `wt-status`
   - *Why*: `wt-list` is generic git tooling that works in any repo. `wt-status` depends on fab (`fab/current`, `statusman.sh`). Mixing them violates single-responsibility and makes `wt-list` depend on fab infrastructure.
   - *Rejected*: `--status` flag on `wt-list` — would couple generic and fab-specific concerns.

2. **Composable architecture**: Single-worktree function as building block
   - *Why*: Discussed with user — "this command should be composed from a more atomic command that gives us status for just one worktree. All worktrees is just the same thing in a loop." This keeps the code DRY and testable.
   - *Rejected*: Monolithic `--all`-only command — less reusable, harder to test.

3. **Default behavior**: No args = current worktree
   - *Why*: Discussed with user — explicitly chose option A (no args = current worktree, `--all` for all). More unix-conventional, and the user specifically requested this default.
   - *Rejected*: No args = all worktrees (like `wt-list`). User preferred the focused default.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Standalone `wt-status` command | Discussed — user confirmed. `wt-list` is generic; fab-specific status is separate. | S:90 R:90 A:90 D:90 |
| 2 | Certain | Composable: atomic single-worktree function + `--all` loop | Discussed — user explicitly requested this architecture. | S:95 R:90 A:95 D:95 |
| 3 | Certain | Default = current worktree, `--all` for all | Discussed — user chose option A explicitly. | S:95 R:90 A:95 D:95 |
| 4 | Certain | Use `statusman.sh display-stage` for stage/state | Authoritative API used by `changeman.sh list`. Confirmed from intake #2. | S:80 R:95 A:95 D:95 |
| 5 | Certain | Source `wt-common.sh` for shared infrastructure | All `wt-*` commands follow this pattern. Confirmed from intake #5. | S:90 R:95 A:95 D:95 |
| 6 | Certain | Read `fab/current` two-line format | Documented format used by all fab scripts. Confirmed from intake #3. | S:90 R:95 A:95 D:95 |
| 7 | Confident | Show full change folder name (not truncated) | Intake assumed truncation, but full names are more informative and consistent with `changeman.sh list`. Column alignment handles length. | S:65 R:90 A:80 D:70 |
| 8 | Certain | Handle edge cases (no fab, no change, stale pointer) | Robustness requirement. Confirmed from intake #7. | S:75 R:85 A:90 D:90 |
| 9 | Certain | Pass absolute .status.yaml path to statusman | resolve_to_status accepts absolute file paths directly — no cd needed. Simpler and avoids subshell overhead. | S:85 R:90 A:95 D:90 |

9 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved).
