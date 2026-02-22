# Spec: wt-create stderr & wt-list flags

**Change**: 260222-s101-wt-create-stderr-wt-list-flags
**Created**: 2026-02-22
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Adding `--porcelain` or `--quiet` as separate flags — `--non-interactive` covers the use case
- Changing interactive mode output behavior — all changes scoped to `--non-interactive` and new flags
- Adding `wt-cd` — that's a downstream consumer of `--path`, not part of this change
- Changing `wt-open` or `wt-delete` output contracts

## wt-create: Non-Interactive Porcelain Output

### Requirement: Stderr Redirect in Non-Interactive Mode

When `--non-interactive` is set, wt-create SHALL write all human-readable messages to stderr and write only the worktree path to stdout. Interactive mode output SHALL remain unchanged.

Specifically:
- `wt_print_success` output (name, path, branch lines) SHALL go to stderr
- `wt_run_worktree_setup` output (init script messages) SHALL go to stderr
- The final `echo "$WT_PATH"` SHALL remain on stdout as the sole stdout output
- The `--reuse` codepath already writes messages to stderr — no changes needed there

#### Scenario: Non-interactive creation captures path cleanly
- **GIVEN** a git repository with no existing worktrees
- **WHEN** `wt-create --non-interactive` is invoked
- **THEN** stdout contains exactly one line: the absolute worktree path
- **AND** stderr contains the human-readable "Created worktree" messages

#### Scenario: Non-interactive with init script
- **GIVEN** a git repository with `fab/.kit/worktree-init.sh` present
- **WHEN** `wt-create --non-interactive` is invoked (init defaults to true)
- **THEN** stdout contains exactly one line: the absolute worktree path
- **AND** stderr contains init script output ("Running worktree init...", script output, "Worktree init complete.")

#### Scenario: Interactive mode unchanged
- **GIVEN** a git repository
- **WHEN** `wt-create` is invoked without `--non-interactive`
- **THEN** all output (success messages, prompts, path) goes to stdout as before

#### Scenario: Reuse path unchanged
- **GIVEN** an existing worktree named "my-wt"
- **WHEN** `wt-create --non-interactive --reuse --worktree-name my-wt` is invoked
- **THEN** stdout contains exactly one line: the absolute worktree path
- **AND** stderr contains "Reusing existing worktree: my-wt"

### Requirement: wt-pr Consistency

wt-pr SHALL apply the same stderr redirect when `--non-interactive` is set. The `wt_print_success` and `wt_run_worktree_setup` calls in wt-pr SHALL redirect to stderr in non-interactive mode, matching wt-create's behavior.

#### Scenario: Non-interactive PR worktree
- **GIVEN** a repository with open PR #42
- **WHEN** `wt-pr --non-interactive 42` is invoked
- **THEN** stdout contains exactly one line: the absolute worktree path
- **AND** stderr contains "Creating worktree for PR #42" and success messages

### Requirement: Batch Caller Simplification

All batch callers SHALL be updated to remove `| tail -1` since `--non-interactive` now guarantees clean stdout. Affected files:
- `batch-fab-new-backlog.sh` line 135
- `batch-fab-switch-change.sh` line 123
- `pipeline/dispatch.sh` line 104

#### Scenario: Batch caller captures path directly
- **GIVEN** a batch script calling `wt-create --non-interactive`
- **WHEN** the caller captures stdout via `$(wt-create --non-interactive ...)`
- **THEN** the captured value is the absolute worktree path with no trailing output

## wt-list: Path Lookup Flag

### Requirement: `--path <name>` Flag

wt-list SHALL support a `--path <name>` flag that outputs the absolute path for a worktree matched by basename. The flag SHALL exit 0 with the path on stdout if found, or exit 1 with an error on stderr if not found. When `--path` is provided, no other output (headers, table, totals) SHALL be written.

#### Scenario: Path lookup for existing worktree
- **GIVEN** a worktree named "swift-fox" exists
- **WHEN** `wt-list --path swift-fox` is invoked
- **THEN** stdout contains exactly one line: the absolute path to the worktree
- **AND** exit code is 0

#### Scenario: Path lookup for nonexistent worktree
- **GIVEN** no worktree named "nonexistent" exists
- **WHEN** `wt-list --path nonexistent` is invoked
- **THEN** stderr contains an error message
- **AND** stdout is empty
- **AND** exit code is 1

#### Scenario: Path lookup for main repo
- **GIVEN** the main repo exists
- **WHEN** `wt-list --path "(main)"` is invoked
- **THEN** it SHALL NOT match — `--path` matches worktree basenames only, not the "(main)" display label

## wt-list: JSON Output Flag

### Requirement: `--json` Flag

wt-list SHALL support a `--json` flag that outputs worktree data as a JSON array to stdout. Each element SHALL be an object with fields: `name` (string — basename or "main"), `branch` (string), `path` (string — absolute), `is_main` (bool), `is_current` (bool), `dirty` (bool — uncommitted changes or untracked files), `unpushed` (int — count of unpushed commits, 0 if no upstream). The output SHALL be valid JSON formatted by `jq`. When `--json` is provided, no human-readable output SHALL be written.

#### Scenario: JSON output with multiple worktrees
- **GIVEN** a main repo and two worktrees ("swift-fox" on wt/swift-fox, "calm-owl" on feature/auth)
- **WHEN** `wt-list --json` is invoked
- **THEN** stdout is a valid JSON array with 3 elements (main + 2 worktrees)
- **AND** each element has all required fields with correct types

#### Scenario: JSON dirty and unpushed detection
- **GIVEN** a worktree "swift-fox" with uncommitted changes and 2 unpushed commits
- **WHEN** `wt-list --json` is invoked
- **THEN** the "swift-fox" element has `"dirty": true` and `"unpushed": 2`

#### Scenario: JSON with no worktrees
- **GIVEN** only the main repo exists (no worktrees)
- **WHEN** `wt-list --json` is invoked
- **THEN** stdout is a JSON array with 1 element (the main repo)

### Requirement: Flag Mutual Exclusivity

`--path` and `--json` SHALL be mutually exclusive. If both are provided, wt-list SHALL exit with an error.

#### Scenario: Conflicting flags
- **GIVEN** any git repository
- **WHEN** `wt-list --path foo --json` is invoked
- **THEN** stderr contains an error about mutually exclusive flags
- **AND** exit code is 2

## wt-list: Status Column in Default View

### Requirement: Status Indicators

The default formatted output of wt-list SHALL include a status column between the branch and path columns. The status column SHALL show:
- `*` if the worktree has uncommitted changes (staged, unstaged, or untracked files)
- `↑N` if the worktree's branch has N unpushed commits (where N > 0)
- Both indicators when both conditions are true
- Empty when the worktree is clean with no unpushed commits

Status checks SHALL be performed by cd-ing into each worktree path and using `wt_has_uncommitted_changes`, `wt_has_untracked_files`, and `wt_get_unpushed_count` from `wt-common.sh`.

#### Scenario: Dirty worktree display
- **GIVEN** a worktree "swift-fox" with uncommitted changes
- **WHEN** `wt-list` is invoked (default formatted output)
- **THEN** the swift-fox row shows `*` in the status column

#### Scenario: Unpushed commits display
- **GIVEN** a worktree "calm-owl" with 3 unpushed commits and no dirty files
- **WHEN** `wt-list` is invoked
- **THEN** the calm-owl row shows `↑3` in the status column

#### Scenario: Both dirty and unpushed
- **GIVEN** a worktree with both uncommitted changes and 2 unpushed commits
- **WHEN** `wt-list` is invoked
- **THEN** the row shows `* ↑2` in the status column

#### Scenario: Clean worktree
- **GIVEN** a worktree with no uncommitted changes and no unpushed commits
- **WHEN** `wt-list` is invoked
- **THEN** the status column is empty for that row

#### Scenario: Main repo status
- **GIVEN** the main repo has uncommitted changes
- **WHEN** `wt-list` is invoked
- **THEN** the (main) row also shows status indicators

## Design Decisions

1. **Redirect mechanism via subshell redirection, not function modification**:
   - *Chosen*: Redirect at the call sites in `main()` (e.g., `wt_print_success ... >&2`) rather than adding an `$output_fd` parameter to shared library functions
   - *Why*: Shared functions in `wt-common.sh` are used by multiple commands. Adding conditional fd logic to library functions couples them to the non-interactive concept. Call-site redirection is explicit, grep-able, and doesn't change function signatures
   - *Rejected*: Global fd variable (`exec 3>&2` / `exec 3>&1`) — harder to reason about, invisible redirection

2. **Status checks via subshell cd**:
   - *Chosen*: `(cd "$path" && wt_has_uncommitted_changes)` for each worktree
   - *Why*: Git commands operate on the current directory. Subshell cd is safe (doesn't affect parent), uses existing helpers, and is the pattern already used in wt-delete for stash checks
   - *Rejected*: `git -C "$path"` prefix — would require rewriting all helper functions to accept a path argument

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `--non-interactive` implies porcelain (no separate flag) | Discussed — user explicitly chose this over `--porcelain` and `--quiet` alternatives | S:95 R:90 A:95 D:95 |
| 2 | Certain | Apply same stderr redirect to `wt-pr --non-interactive` | Consistency — wt-pr shares the same flag and output pattern | S:80 R:95 A:90 D:95 |
| 3 | Certain | Use `jq` for `--json` formatting | Already a project prerequisite (validated by `sync/1-prerequisites.sh`) | S:85 R:95 A:95 D:95 |
| 4 | Confident | Status column uses `*` for dirty and `↑N` for unpushed | Common git convention (git prompt, lazygit). Could use other symbols but these are well-established | S:70 R:95 A:80 D:75 |
| 5 | Confident | `--path` exits non-zero when worktree not found | Standard CLI convention for lookup commands | S:75 R:90 A:85 D:80 |
| 6 | Confident | `--json` includes dirty/unpushed fields | Essential for monitoring use case; slight perf cost acceptable | S:70 R:85 A:80 D:80 |
| 7 | Certain | Redirect at call sites, not in shared library functions | Preserves wt-common.sh API; call-site `>&2` is explicit and grep-able | S:85 R:90 A:90 D:90 |
| 8 | Confident | `--path` and `--json` are mutually exclusive | Different output contracts — combining them is nonsensical | S:80 R:95 A:90 D:85 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
