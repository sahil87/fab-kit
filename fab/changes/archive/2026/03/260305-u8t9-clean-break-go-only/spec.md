# Spec: Clean Break — Go Only

**Change**: 260305-u8t9-clean-break-go-only
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Refactoring Go parity tests from "compare bash vs Go" to "test Go standalone" — deferred to a follow-up change
- Porting `frontmatter.sh` or `env-packages.sh` to Go — these remain as shell scripts
- Renaming or relocating `fab/.kit/bin/fab` dispatcher path — it stays at its current location

## Kit Architecture: Dispatcher

### Requirement: Shell Fallback Removal

The `fab/.kit/bin/fab` dispatcher SHALL remove the shell fallback `case` block (lines 30–49 of current file). When no compiled backend (`fab-rust` or `fab-go`) is found, the dispatcher SHALL print an error message to stderr directing the user to install the Go binary or run `fab-sync.sh`, then exit 1.

#### Scenario: Dispatcher with Go backend present
- **GIVEN** `fab/.kit/bin/fab-go` exists and is executable
- **WHEN** the user runs `fab status current-stage u8t9`
- **THEN** the dispatcher execs `fab-go` with the original arguments
- **AND** no shell fallback is invoked

#### Scenario: Dispatcher with no compiled backend
- **GIVEN** neither `fab-rust` nor `fab-go` exists in `fab/.kit/bin/`
- **WHEN** the user runs `fab status current-stage u8t9`
- **THEN** the dispatcher prints `Error: no fab backend found (expected fab-go or fab-rust in <dir>)` to stderr
- **AND** prints `Fix: run fab-sync.sh or download a platform-specific kit archive` to stderr
- **AND** exits with code 1

### Requirement: Version Handler Preservation

The `--version` handler SHALL remain in the dispatcher. It SHALL report `"none"` as the backend when no compiled binary is found (instead of the current `"shell"`).

#### Scenario: Version check without backend
- **GIVEN** neither `fab-rust` nor `fab-go` exists
- **WHEN** the user runs `fab --version`
- **THEN** the output is `fab <version> (none backend)` (formerly reported `shell`)

### Requirement: LIB_DIR Variable Removal

The dispatcher SHALL NOT define or reference `LIB_DIR`. The `SCRIPT_DIR` and `KIT_DIR` variables remain (used by `--version` handler).

#### Scenario: Dispatcher contains no LIB_DIR
- **GIVEN** the updated dispatcher script
- **WHEN** inspecting the file content
- **THEN** no line references `LIB_DIR`

## Kit Architecture: Shell Script Deletion

### Requirement: Delete Ported Shell Scripts

The following 7 files SHALL be deleted from `fab/.kit/scripts/lib/`:
- `statusman.sh`
- `changeman.sh`
- `archiveman.sh`
- `logman.sh`
- `calc-score.sh`
- `preflight.sh`
- `resolve.sh`

#### Scenario: Ported scripts removed
- **GIVEN** the change is applied
- **WHEN** listing `fab/.kit/scripts/lib/`
- **THEN** only `env-packages.sh` and `frontmatter.sh` remain

### Requirement: Retained Shell Scripts

`env-packages.sh` and `frontmatter.sh` SHALL remain in `fab/.kit/scripts/lib/`. They are not ported to Go and have active consumers (`fab-help.sh`, `2-sync-workspace.sh`, `env-packages.sh` via `.envrc`).

#### Scenario: Retained scripts unmodified
- **GIVEN** the change is applied
- **WHEN** comparing `env-packages.sh` content (excluding the PATH addition)
- **THEN** `frontmatter.sh` is unchanged from before the change

## Kit Architecture: PATH Setup

### Requirement: Add bin/ to PATH

`fab/.kit/scripts/lib/env-packages.sh` SHALL add `$KIT_DIR/bin` to PATH before the packages loop. This makes `fab` callable as a bare command in terminal sessions using direnv.

#### Scenario: fab command available after sourcing env-packages.sh
- **GIVEN** a shell session that sources `env-packages.sh`
- **WHEN** the user types `fab --version`
- **THEN** the command resolves to `fab/.kit/bin/fab` via PATH

## Kit Architecture: wt-status Removal

### Requirement: Delete wt-status Shell Command

`fab/.kit/packages/wt/bin/wt-status` SHALL be deleted. Its functionality is replaced by `fab status show`.

#### Scenario: wt-status no longer exists
- **GIVEN** the change is applied
- **WHEN** listing `fab/.kit/packages/wt/bin/`
- **THEN** `wt-status` is not present

### Requirement: Delete wt-status Test

`src/packages/wt/tests/wt-status.bats` SHALL be deleted.

#### Scenario: wt-status test removed
- **GIVEN** the change is applied
- **WHEN** listing `src/packages/wt/tests/`
- **THEN** `wt-status.bats` is not present

## Go Binary: fab status show

### Requirement: New status show Subcommand

The Go binary SHALL add a `show` subcommand under `fab status` with the interface:

```
fab status show [--all] [--json] [<name>]
```

#### Scenario: Current worktree status (default)
- **GIVEN** the user is in a worktree with an active fab change
- **WHEN** running `fab status show`
- **THEN** output is a single human-readable line: `<wt-name>  <change-folder>  <stage>  <state>`

#### Scenario: Current worktree with no fab directory
- **GIVEN** the user is in a worktree without a `fab/` directory
- **WHEN** running `fab status show`
- **THEN** output shows `<wt-name>  (no fab)`

#### Scenario: Current worktree with no active change
- **GIVEN** the worktree has `fab/` but no `fab/current` or it's empty
- **WHEN** running `fab status show`
- **THEN** output shows `<wt-name>  (no change)`

#### Scenario: Named worktree
- **GIVEN** a worktree named "swift-fox" exists
- **WHEN** running `fab status show swift-fox`
- **THEN** output shows the fab status for that specific worktree

#### Scenario: Named worktree not found
- **GIVEN** no worktree named "nonexistent" exists
- **WHEN** running `fab status show nonexistent`
- **THEN** exits non-zero with error: `Worktree 'nonexistent' not found`

#### Scenario: All worktrees (human-readable)
- **GIVEN** multiple worktrees exist
- **WHEN** running `fab status show --all`
- **THEN** output includes a header with repo name and location, a formatted table of all worktrees, and a total count
- **AND** the current worktree is marked with `*`

#### Scenario: Single worktree JSON output
- **GIVEN** the user is in a worktree with an active fab change
- **WHEN** running `fab status show --json`
- **THEN** output is a JSON object with fields: `name`, `path`, `branch`, `is_main`, `is_current`, `change`, `stage`, `state`

#### Scenario: All worktrees JSON output
- **GIVEN** multiple worktrees exist
- **WHEN** running `fab status show --all --json`
- **THEN** output is a JSON array of worktree status objects

### Requirement: Worktree Discovery

`fab status show` SHALL discover worktrees via `git worktree list --porcelain` and parse the output to extract path, branch, and HEAD information. This is the same approach used by the shell `wt_list_worktrees` function.

#### Scenario: Worktree discovery
- **GIVEN** a git repo with 3 worktrees
- **WHEN** running `fab status show --all`
- **THEN** all 3 worktrees are listed

### Requirement: Fab State Resolution

For each worktree, `fab status show` SHALL resolve fab pipeline state using the existing `internal/resolve` and `internal/statusfile` packages:
1. Check for `<wt-path>/fab/` directory existence
2. Read `<wt-path>/fab/current` for active change name
3. Load `.status.yaml` from the change directory
4. Extract display stage and state via `internal/status`

#### Scenario: Stale fab/current pointer
- **GIVEN** `fab/current` points to a change folder that doesn't exist
- **WHEN** running `fab status show`
- **THEN** output shows `<wt-name>  (stale)`

## Pipeline: dispatch.sh Update

### Requirement: Update calc-score.sh Reference

`fab/.kit/scripts/pipeline/dispatch.sh` SHALL replace the direct `calc-score.sh` invocation with `fab score --check-gate`. The `validate_prerequisites` function SHALL use the `fab` dispatcher via the worktree's `fab/.kit/bin/fab` instead of directly calling `calc-score.sh`.

#### Scenario: Gate check uses fab dispatcher
- **GIVEN** `dispatch.sh` validate_prerequisites runs
- **WHEN** checking the confidence gate
- **THEN** the command invoked is `<wt-path>/fab/.kit/bin/fab score --check-gate <change-id>`
- **AND** the `calc-score.sh` path check is removed

## Documentation: _scripts.md Update

### Requirement: Update _scripts.md

`fab/.kit/skills/_scripts.md` SHALL be updated to:
1. Remove the `[fab] using shell backend` stderr message reference
2. Update Backend Priority to show `rust > go > error` (no shell fallback)
3. Note that `fab -h`, `fab --help`, and `fab <subcommand> --help` work via Cobra
4. Remove references to shell scripts as the fallback mechanism

#### Scenario: _scripts.md reflects Go-only backend
- **GIVEN** the updated `_scripts.md`
- **WHEN** reading the Backend Priority section
- **THEN** it shows `fab-rust` → `fab-go` → error (no shell fallback)
- **AND** no mention of `[fab] using shell backend`

## Deprecated Requirements

### wt-status Shell Command
**Reason**: Replaced by `fab status show` in the Go binary. The Go implementation provides the same human-readable output plus JSON output for orchestrator consumption.
**Migration**: Use `fab status show [--all] [--json]` instead of `wt-status [--all]`.

### Shell Script Fallback in Dispatcher
**Reason**: All 7 CLI shell scripts have been fully ported to Go with parity tests. The shell fallback adds maintenance burden with no benefit.
**Migration**: Install the Go binary (via platform-specific kit archive or `go build`). The generic `kit.tar.gz` archive (shell-only) no longer provides a working `fab` command.

## Design Decisions

1. **Keep dispatcher as thin shell script**: The dispatcher remains `#!/usr/bin/env sh` (not replaced by Go binary directly) to preserve the `fab-rust` > `fab-go` priority chain and the `--version` handler that works even without a backend.
   - *Why*: The dispatcher is the stable entry point. Backend selection logic belongs here, not in either backend.
   - *Rejected*: Making `fab-go` the direct entry point — loses the rust priority slot and version reporting.

2. **`fab status show` in Go, not as a separate binary**: The worktree status functionality is added as a subcommand of the existing `fab` Go binary rather than creating a new `fab-wt` binary.
   - *Why*: Consolidates into the single binary that's already distributed. Reuses existing resolve/statusfile packages.
   - *Rejected*: Separate `fab-wt` binary — adds another binary to distribute and maintain.

3. **dispatch.sh uses change ID, not directory path**: The `dispatch.sh` update changes from passing `$change_dir` (directory path) to passing `$CHANGE_ID` (4-char ID) to `fab score --check-gate`.
   - *Why*: The Go binary's `fab score` command accepts change references (ID, substring, folder name), not raw directory paths. This aligns with the unified `<change>` argument convention.
   - *Rejected*: Adding directory path support to `fab score` — violates the convention established in `_scripts.md`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keep `fab/.kit/bin/fab` path unchanged | Confirmed from intake #1 — discussed, Option A chosen, 320 references stay | S:95 R:90 A:95 D:95 |
| 2 | Certain | Dispatcher keeps `fab-rust` > `fab-go` priority | Confirmed from intake #2 — user explicitly requested rust support | S:90 R:85 A:90 D:95 |
| 3 | Certain | Keep `frontmatter.sh` and `env-packages.sh` | Confirmed from intake #3/#4 — active consumers verified | S:95 R:90 A:95 D:95 |
| 4 | Certain | `fab status show` interface: `[--all] [--json] [<name>]` | Confirmed from intake #5 — discussed, `--all --json` is orchestrator endpoint | S:90 R:85 A:85 D:85 |
| 5 | Certain | Remove wt-status completely (no deprecation) | Confirmed from intake #6 — clean break, `fab status show` replaces it | S:85 R:80 A:85 D:90 |
| 6 | Certain | Delete all 7 ported lib/ scripts | Parity tests confirm Go binary equivalence; shell scripts are dead code | S:95 R:80 A:95 D:95 |
| 7 | Confident | Keep Go parity tests as-is, skip bash side gracefully | Confirmed from intake #7 — full refactor deferred. Tests will skip bash comparison when scripts missing | S:70 R:85 A:75 D:70 |
| 8 | Confident | `yq` still needed by dispatch.sh and fab-doctor.sh | Confirmed from intake #8 — dispatch.sh uses `yq -i` for manifest YAML writes | S:75 R:80 A:80 D:75 |
| 9 | Certain | dispatch.sh uses change ID for fab score invocation | Go binary accepts unified `<change>` references per _scripts.md convention | S:85 R:85 A:90 D:90 |

9 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved).
