# Spec: Scriptify Fab-Archive

**Change**: 260303-hcq9-scriptify-fab-archive
**Created**: 2026-03-03
**Affected memory**: None — internal tooling optimization

## Non-Goals

- Changing backlog matching logic (Step 4) — stays agent-driven for keyword extraction and interactive confirmation
- Adding new features to `/fab-archive` — this is a pure performance refactoring
- Modifying `resolve.sh` — archiveman handles archive-folder resolution internally

## Script: archiveman.sh

### Requirement: Script Location and Convention

`archiveman.sh` SHALL be created at `fab/.kit/scripts/lib/archiveman.sh` following the established kit script pattern (changeman, statusman, resolve, logman, calc-score).

The script SHALL use `set -euo pipefail`, resolve `LIB_DIR` and `FAB_ROOT` using the same `readlink -f` pattern as other kit scripts, and invoke `resolve.sh` directly from `LIB_DIR` for active change resolution.

#### Scenario: Script exists and is executable
- **GIVEN** fab-kit is installed
- **WHEN** `bash fab/.kit/scripts/lib/archiveman.sh --help` is run
- **THEN** usage information is printed to stdout
- **AND** exit code is 0

### Requirement: CLI Dispatch

`archiveman.sh` SHALL support four subcommands: `archive`, `restore`, `list`, and `--help`. Unknown subcommands SHALL print an error to stderr and exit 1.

```
archiveman.sh archive <change> [--description "..."]
archiveman.sh restore <change> [--switch]
archiveman.sh list
archiveman.sh --help
```

#### Scenario: Unknown subcommand
- **GIVEN** archiveman.sh exists
- **WHEN** `archiveman.sh unknown` is run
- **THEN** stderr contains "Unknown subcommand 'unknown'"
- **AND** exit code is 1

### Requirement: Archive Subcommand

`archiveman.sh archive <change>` SHALL perform four sequential operations in one invocation:

1. **Clean**: Delete `.pr-done` from the change folder if present
2. **Move**: Move `fab/changes/{name}/` to `fab/changes/archive/{name}/` (create `archive/` if needed)
3. **Index**: Update `fab/changes/archive/index.md` with a new entry (create file with header if missing)
4. **Pointer**: If the active change matches (via `changeman.sh resolve`), clear via `changeman.sh switch --blank`

The `<change>` argument SHALL be resolved using `resolve.sh --folder` against the active changes folder (`fab/changes/`, excluding `archive/`). This is the standard resolution — no special logic needed since the change hasn't been archived yet when this runs.

The `--description` flag is REQUIRED for archive. If omitted, the script SHALL exit 1 with an error message.

#### Scenario: Full archive of active change
- **GIVEN** change `260303-hcq9-scriptify-fab-archive` exists in `fab/changes/`
- **AND** it is the active change in `fab/current`
- **AND** `.pr-done` exists in the change folder
- **WHEN** `archiveman.sh archive hcq9 --description "Offloaded archive operations to shell script"` is run
- **THEN** `.pr-done` is deleted
- **AND** the folder is moved to `fab/changes/archive/260303-hcq9-scriptify-fab-archive/`
- **AND** `fab/changes/archive/index.md` is prepended with the entry
- **AND** `fab/current` is cleared
- **AND** stdout contains structured YAML (see Output Format)
- **AND** exit code is 0

#### Scenario: Archive when already in archive
- **GIVEN** change folder already exists at `fab/changes/archive/{name}/`
- **WHEN** `archiveman.sh archive <change>` is run
- **THEN** resolution against `fab/changes/` fails (folder not found)
- **AND** exit code is 1 with "No change matches" error

#### Scenario: Archive when not the active change
- **GIVEN** change exists in `fab/changes/` but is NOT the active change
- **WHEN** `archiveman.sh archive <change> --description "..."` is run
- **THEN** clean, move, and index steps execute normally
- **AND** pointer step outputs `skipped`
- **AND** exit code is 0

#### Scenario: Missing --description flag
- **GIVEN** a valid change exists
- **WHEN** `archiveman.sh archive <change>` is run without `--description`
- **THEN** stderr contains "ERROR: --description is required for archive"
- **AND** exit code is 1

### Requirement: Archive Index Format

When creating a new index file, the script SHALL write a `# Archive Index` header followed by a blank line. When updating an existing index, the entry SHALL be prepended after the header line (most-recent-first order).

Entry format: `- **{folder-name}** — {description}`

Where `{description}` is the value passed via `--description`.

The script SHALL also perform a one-time backfill: if archived folders exist that have no corresponding entry in the index, the script SHALL append entries for them (with description `(no description — pre-index archive)`). Backfill entries go after the new entry, maintaining recent-first order for new entries. Backfill runs on every invocation but is a no-op when all folders are indexed.
<!-- assumed: Backfill in script — deterministic folder scanning is mechanical, not reasoning -->

#### Scenario: Index file doesn't exist
- **GIVEN** `fab/changes/archive/index.md` does not exist
- **AND** two folders already exist in `archive/` without index entries
- **WHEN** `archiveman.sh archive <change> --description "New feature"` is run
- **THEN** `index.md` is created with header
- **AND** the new entry is first
- **AND** two backfill entries follow

#### Scenario: Index file exists with existing entries
- **GIVEN** `fab/changes/archive/index.md` exists with 5 entries
- **WHEN** `archiveman.sh archive <change> --description "Bug fix"` is run
- **THEN** the new entry is prepended after the header
- **AND** existing entries follow unchanged

### Requirement: Archive Output Format

On success, `archiveman.sh archive` SHALL output structured YAML to stdout:

```yaml
action: archive
name: {full-folder-name}
clean: removed        # or: not_present
move: moved           # or: already_archived (shouldn't happen — resolve would fail)
index: updated        # or: created
pointer: cleared      # or: skipped
```

#### Scenario: YAML output is parseable
- **GIVEN** a successful archive operation
- **WHEN** the output is piped through `yq`
- **THEN** all six fields are present and contain valid values

### Requirement: Restore Subcommand

`archiveman.sh restore <change>` SHALL perform three sequential operations:

1. **Move**: Move `fab/changes/archive/{name}/` to `fab/changes/{name}/`
2. **Index**: Remove the matching entry from `fab/changes/archive/index.md`
3. **Pointer**: If `--switch` flag is provided, run `changeman.sh switch {name}`

The `<change>` argument SHALL be resolved against `fab/changes/archive/` folder names using the same case-insensitive substring matching logic as `resolve.sh`, but scanning `archive/` instead of `changes/`. This resolution logic SHALL be implemented within `archiveman.sh` (not by modifying `resolve.sh`).

#### Scenario: Full restore with --switch
- **GIVEN** `260303-hcq9-scriptify-fab-archive` exists in `fab/changes/archive/`
- **AND** an entry exists in `archive/index.md`
- **WHEN** `archiveman.sh restore hcq9 --switch` is run
- **THEN** folder is moved to `fab/changes/260303-hcq9-scriptify-fab-archive/`
- **AND** index entry is removed
- **AND** `fab/current` is set to the restored change
- **AND** stdout contains structured YAML
- **AND** exit code is 0

#### Scenario: Restore without --switch
- **GIVEN** an archived change exists
- **WHEN** `archiveman.sh restore <change>` is run without `--switch`
- **THEN** folder is moved and index entry is removed
- **AND** pointer step outputs `skipped`

#### Scenario: Restore when already in changes folder
- **GIVEN** folder already exists at `fab/changes/{name}/` (not in archive)
- **WHEN** `archiveman.sh restore <change>` is run
- **THEN** move step outputs `already_in_changes`
- **AND** remaining steps (index, pointer) still execute
- **AND** exit code is 0

#### Scenario: Multiple archive matches
- **GIVEN** two archived folders match the substring "fix"
- **WHEN** `archiveman.sh restore fix` is run
- **THEN** stderr contains "Multiple archives match" with folder names
- **AND** exit code is 1

#### Scenario: No archive match
- **GIVEN** no archived folder matches "nonexistent"
- **WHEN** `archiveman.sh restore nonexistent` is run
- **THEN** stderr contains "No archive matches"
- **AND** exit code is 1

### Requirement: Restore Output Format

On success, `archiveman.sh restore` SHALL output structured YAML to stdout:

```yaml
action: restore
name: {full-folder-name}
move: restored        # or: already_in_changes
index: removed        # or: not_found
pointer: switched     # or: skipped
```

#### Scenario: Restore output fields
- **GIVEN** a successful restore operation
- **WHEN** the output is piped through `yq`
- **THEN** all fields are present and contain valid values

### Requirement: List Subcommand

`archiveman.sh list` SHALL output one archived folder name per line to stdout. If `fab/changes/archive/` doesn't exist or is empty, output nothing (exit 0).

#### Scenario: List with archived changes
- **GIVEN** 3 folders exist in `fab/changes/archive/`
- **WHEN** `archiveman.sh list` is run
- **THEN** stdout contains 3 lines, one folder name each
- **AND** exit code is 0

#### Scenario: List with empty archive
- **GIVEN** `fab/changes/archive/` exists but contains no folders
- **WHEN** `archiveman.sh list` is run
- **THEN** stdout is empty
- **AND** exit code is 0

## Skill: fab-archive Orchestrator

### Requirement: Slim Archive Mode

The `/fab-archive` skill's archive mode SHALL be reduced to an orchestrator that:

1. Runs preflight (existing — validates hydrate done)
2. Extracts a 1-2 sentence description from the intake's Why section (agent intelligence)
3. Calls `archiveman.sh archive <change> --description "..."` (single shell call replaces Steps 1, 2, 3, 5)
4. Runs backlog matching — Step 4 (agent intelligence: keyword scan + interactive confirmation)
5. Parses YAML output and formats the user-facing report

Steps 1–3 of the current skill (clean, move, index) and Step 5 (clear pointer) SHALL be removed from the skill and fully delegated to `archiveman.sh`.

#### Scenario: Archive mode with backlog match
- **GIVEN** a completed change with intake containing backlog ID
- **AND** `fab/backlog.md` has a matching unchecked item
- **WHEN** `/fab-archive` is invoked
- **THEN** agent extracts description from intake
- **AND** `archiveman.sh archive` is called once
- **AND** backlog matching runs (agent-driven)
- **AND** user-facing report is displayed using script's YAML output

#### Scenario: Archive mode without backlog
- **GIVEN** a completed change
- **AND** `fab/backlog.md` does not exist
- **WHEN** `/fab-archive` is invoked
- **THEN** `archiveman.sh archive` is called once
- **AND** backlog step is skipped silently

### Requirement: Slim Restore Mode

The `/fab-archive restore` skill SHALL be reduced to:

1. Call `archiveman.sh list` to get archived folder names
2. Fuzzy match the user's argument (resolve in script via `archiveman.sh restore`)
3. Call `archiveman.sh restore <match> [--switch]` (single shell call replaces all 3 restore steps)
4. Parse YAML output and format the user-facing report

All three restore steps (move, index, pointer) SHALL be fully delegated to `archiveman.sh`.

#### Scenario: Restore mode with --switch
- **GIVEN** an archived change matching the user's argument
- **WHEN** `/fab-archive restore <name> --switch` is invoked
- **THEN** `archiveman.sh restore <name> --switch` is called once
- **AND** user-facing report shows restore complete with pointer updated

### Requirement: Output Format Unchanged

The user-facing output format of `/fab-archive` (both archive and restore modes) SHALL remain identical to the current format. The report is constructed from the YAML output fields, not displayed raw.

#### Scenario: Archive output matches current format
- **GIVEN** a successful archive via the new orchestrator
- **WHEN** the user reads the output
- **THEN** it matches the existing format:
  ```
  Archive: {change name}

  Cleaned:  ✓ .pr-done removed                    (or: — not present)
  Moved:    ✓ fab/changes/archive/{name}/
  Index:    ✓ fab/changes/archive/index.md updated
  Backlog:  ...
  Scan:     ...
  Pointer:  ✓ fab/current cleared                 (or: — skipped, not active)

  Archive complete.
  ```

## Tests: archiveman.sh

### Requirement: Test Infrastructure

Tests SHALL be created at `src/lib/archiveman/test.bats` following the two-file convention used by all other kit scripts (changeman, resolve, logman, etc.).

The test setup SHALL:
- Create a temp directory with a minimal `fab/` structure (`changes/`, `changes/archive/`, `.kit/scripts/lib/`)
- Copy the real `archiveman.sh` into the kit location
- Copy `resolve.sh` (used by archive resolution)
- Stub `changeman.sh` to record calls (for pointer clear/switch verification) — the stub SHALL handle `resolve` (returning the folder from `fab/current`), `switch --blank` (deleting `fab/current`), and `switch <name>` (writing `fab/current`)
- Clean up the temp directory in teardown

#### Scenario: Tests pass in CI
- **GIVEN** `bats` is installed
- **WHEN** `bats src/lib/archiveman/test.bats` is run
- **THEN** all tests pass
- **AND** no temp files are left behind

### Requirement: Archive Subcommand Tests

The test suite SHALL cover the following archive scenarios:

**Happy path:**
- Archive moves folder to `archive/`, creates index entry, outputs YAML with correct fields
- Archive with `.pr-done` present deletes it (clean: removed)
- Archive without `.pr-done` reports clean: not_present
- Archive creates `archive/` directory if it doesn't exist
- Archive clears `fab/current` when change is active (pointer: cleared)
- Archive skips pointer clear when change is not active (pointer: skipped)

**Index management:**
- Archive creates `index.md` with header when it doesn't exist
- Archive prepends new entry after header in existing index
- New entry uses `--description` text verbatim
- Entry format matches `- **{folder-name}** — {description}`

**Backfill:**
- Backfill adds entries for archived folders missing from index
- Backfill is a no-op when all folders are already indexed
- Backfill entries use placeholder description `(no description — pre-index archive)`

**Error cases:**
- Missing `--description` flag exits 1 with error message
- No `<change>` argument exits 1
- Change not found exits 1 (resolve fails)

**YAML output:**
- Output contains all five fields: action, name, clean, move, index, pointer
- Output is valid YAML parseable by `yq`

#### Scenario: Archive happy path test
- **GIVEN** test environment with a change folder and `fab/current` pointing to it
- **WHEN** `archiveman.sh archive <change> --description "Test archive"` is run
- **THEN** folder is in `archive/`, index entry exists, `fab/current` is cleared
- **AND** YAML output has `action: archive`, `move: moved`, `pointer: cleared`

### Requirement: Restore Subcommand Tests

The test suite SHALL cover the following restore scenarios:

**Happy path:**
- Restore moves folder from `archive/` back to `changes/`
- Restore removes matching entry from `index.md`
- Restore with `--switch` calls `changeman.sh switch {name}` (pointer: switched)
- Restore without `--switch` skips pointer (pointer: skipped)

**Resolution:**
- Exact match resolves correctly
- Substring match (case-insensitive) resolves single match
- 4-char ID match resolves correctly
- Multiple matches exits 1 with "Multiple archives match" listing
- No match exits 1 with "No archive matches"

**Resumability:**
- If folder already exists in `changes/` (not archive), move outputs `already_in_changes`
- Index removal and pointer steps still execute even when move is skipped

**Index cleanup:**
- Entry is removed from index on restore
- Missing entry in index is a no-op (index: not_found)
- Index file is preserved even if it becomes header-only after removal

**YAML output:**
- Output contains all five fields: action, name, move, index, pointer
- Output is valid YAML parseable by `yq`

#### Scenario: Restore happy path test
- **GIVEN** an archived change in `fab/changes/archive/`
- **AND** a matching entry in `archive/index.md`
- **WHEN** `archiveman.sh restore <change> --switch` is run
- **THEN** folder is in `changes/`, index entry is removed, pointer is set
- **AND** YAML output has `action: restore`, `move: restored`, `pointer: switched`

### Requirement: List Subcommand Tests

The test suite SHALL cover:

- List outputs one folder name per line for each archived folder
- List excludes `index.md` from output
- List returns empty output (exit 0) when archive is empty
- List returns empty output (exit 0) when `archive/` directory doesn't exist

#### Scenario: List outputs folder names
- **GIVEN** 3 folders in `fab/changes/archive/`
- **WHEN** `archiveman.sh list` is run
- **THEN** stdout has 3 lines, each a folder name
- **AND** `index.md` is not in the output

### Requirement: CLI Edge Cases

The test suite SHALL cover:

- `--help` prints usage information (exit 0)
- No subcommand prints error (exit 1)
- Unknown subcommand prints "Unknown subcommand" error (exit 1)

#### Scenario: Help output
- **GIVEN** archiveman.sh exists
- **WHEN** `archiveman.sh --help` is run
- **THEN** output contains "USAGE" and subcommand descriptions

## Design Decisions

1. **Archive resolution uses standard `resolve.sh`**: archiveman's `archive` subcommand resolves the `<change>` argument through the normal `resolve.sh --folder` path (scanning `fab/changes/`, excluding `archive/`). This works because the change hasn't been archived yet when archive runs. No modifications to `resolve.sh` needed.
   - *Why*: Reuses existing resolution infrastructure, avoids touching a well-tested utility.
   - *Rejected*: Adding `--archive` flag to `resolve.sh` — unnecessary complexity for one caller.

2. **Restore resolution is self-contained in archiveman**: The `restore` subcommand implements its own case-insensitive substring matching against `fab/changes/archive/` folder names, following the same matching logic as `resolve.sh` but scanning a different directory.
   - *Why*: Keeps `resolve.sh` focused on active changes. Restore is the only consumer of archive resolution.
   - *Rejected*: Adding dual-directory support to `resolve.sh` — violates single responsibility, adds complexity for one use case.

3. **`--description` is required, not optional**: The archive subcommand requires `--description` rather than generating one or leaving it blank. The agent computes the description from intake before calling the script.
   - *Why*: Clean separation — agent handles intelligence (summarization), script handles mechanics (file operations). Avoids the script needing to parse markdown.
   - *Rejected*: Script reads intake and extracts description — would require markdown parsing in bash, fragile and outside script's responsibility.

4. **Index backfill in script**: Backfill (scanning folders missing from the index) runs in the script on every archive invocation. It's deterministic and mechanical — list folders, check index, add missing entries.
   - *Why*: No agent reasoning needed. Running on every invocation is cheap and keeps the index self-healing.
   - *Rejected*: Backfill in skill — adds round-trips for a purely mechanical operation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | archiveman.sh at `fab/.kit/scripts/lib/` following existing pattern | Confirmed from intake #1 — consistent with changeman/statusman/preflight pattern; constitution requires shell scripts | S:90 R:85 A:95 D:90 |
| 2 | Certain | Backlog matching stays agent-driven in skill | Confirmed from intake #2 — keyword extraction and interactive confirmation need agent reasoning | S:85 R:85 A:90 D:90 |
| 3 | Certain | YAML output format for script results | Confirmed from intake #3 — follows preflight.sh YAML output pattern; consistent and parseable | S:85 R:85 A:90 D:85 |
| 4 | Certain | Restore fuzzy matching in script (substring match) | Confirmed from intake #4 — mechanical case-insensitive substring matching, same logic as resolve.sh | S:80 R:85 A:85 D:80 |
| 5 | Certain | Description passed as `--description` arg, computed by agent | Confirmed from intake #5 — clean separation of intelligence (agent) and mechanics (script) | S:85 R:85 A:85 D:80 |
| 6 | Confident | Index backfill in script, runs on every archive invocation | Upgraded from intake #6 Tentative — deterministic folder scanning is clearly mechanical; per-invocation run is cheap and self-healing | S:70 R:80 A:75 D:70 |
| 7 | Certain | Archive resolution uses standard resolve.sh, no modifications needed | Change hasn't been archived yet when archive command runs — standard resolution works | S:90 R:90 A:90 D:90 |
| 8 | Certain | User-facing output format unchanged | Pure internal refactoring — report constructed from YAML fields to match existing format | S:85 R:90 A:90 D:90 |
| 9 | Certain | BATS tests at `src/lib/archiveman/test.bats` following two-file convention | Matches changeman, resolve, logman test patterns; BATS is the established kit test framework | S:90 R:90 A:95 D:90 |
| 10 | Certain | Stub changeman.sh in tests (not real script) for pointer operations | Test isolation — avoids testing changeman behavior, focuses on archiveman logic; matches changeman test pattern of stubbing statusman | S:85 R:85 A:90 D:85 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).
