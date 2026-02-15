# Spec: Batch Script Rename and Default List Behavior

**Change**: 260215-g4r2-DEV-1023-batch-rename-default-list
**Created**: 2026-02-15
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Scripts: Default No-Arg Behavior

### Requirement: No-arg invocation SHALL show list output

All three batch scripts (`batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh`, `batch-fab-archive-change.sh`) SHALL display their `--list` output when invoked with no arguments, instead of showing usage text.

The implementation SHALL use `set -- --list` to rewrite positional parameters, allowing fallthrough into the existing `--list` case in the `case` statement.

#### Scenario: No-arg shows list (new-backlog)
- **GIVEN** `batch-fab-new-backlog.sh` exists and `fab/backlog.md` has pending items
- **WHEN** the script is invoked with no arguments
- **THEN** it SHALL display the pending backlog items list (same output as `--list`)
- **AND** exit with code 0

#### Scenario: No-arg shows list (switch-change)
- **GIVEN** `batch-fab-switch-change.sh` exists and `fab/changes/` contains active changes
- **WHEN** the script is invoked with no arguments
- **THEN** it SHALL display the available changes list (same output as `--list`)
- **AND** exit with code 0

#### Scenario: No-arg shows list (archive-change)
- **GIVEN** `batch-fab-archive-change.sh` exists and `fab/changes/` may contain archivable changes
- **WHEN** the script is invoked with no arguments
- **THEN** it SHALL display the archivable changes list (same output as `--list`)
- **AND** exit with code 0

### Requirement: Help SHALL remain available via flags

The `-h` and `--help` flags SHALL continue to display usage text. No behavioral change to the help output, except that script names in usage text and examples SHALL reflect the new `batch-fab-*` filenames.

#### Scenario: Help flag still works
- **GIVEN** any of the three renamed batch scripts
- **WHEN** invoked with `-h` or `--help`
- **THEN** it SHALL display usage text with the updated script name
- **AND** examples SHALL reference the new `batch-fab-*` name

## Scripts: Rename

### Requirement: Batch scripts SHALL be renamed to `batch-fab-*`

The three batch scripts SHALL be renamed from `batch-{verb}-{entity}.sh` to `batch-fab-{verb}-{entity}.sh`:

| Current name | New name |
|---|---|
| `batch-new-backlog.sh` | `batch-fab-new-backlog.sh` |
| `batch-switch-change.sh` | `batch-fab-switch-change.sh` |
| `batch-archive-change.sh` | `batch-fab-archive-change.sh` |

The rename SHALL be a `git mv` to preserve history.

#### Scenario: Scripts renamed
- **GIVEN** the three batch scripts exist at their current names in `fab/.kit/scripts/`
- **WHEN** the rename is applied
- **THEN** each script SHALL exist at its new `batch-fab-*` name
- **AND** the old names SHALL no longer exist

### Requirement: Usage text SHALL reflect new filenames

Each script's `usage()` function SHALL reference the new filename in the `Usage:` line and all `Examples:` lines.

#### Scenario: Usage text updated (new-backlog)
- **GIVEN** `batch-fab-new-backlog.sh`
- **WHEN** `-h` or `--help` is passed
- **THEN** the usage header SHALL read `Usage: batch-fab-new-backlog <backlog-id> [<backlog-id>...]`
- **AND** examples SHALL use `batch-fab-new-backlog` as the command name

#### Scenario: Usage text updated (switch-change)
- **GIVEN** `batch-fab-switch-change.sh`
- **WHEN** `-h` or `--help` is passed
- **THEN** the usage header SHALL read `Usage: batch-fab-switch-change <change> [<change>...]`
- **AND** examples SHALL use `batch-fab-switch-change` as the command name

#### Scenario: Usage text updated (archive-change)
- **GIVEN** `batch-fab-archive-change.sh`
- **WHEN** `-h` or `--help` is passed
- **THEN** the usage header SHALL read `Usage: batch-fab-archive-change <change> [<change>...]`
- **AND** examples SHALL use `batch-fab-archive-change` as the command name

## Documentation: Memory Update

### Requirement: Kit architecture memory SHALL reflect new names and pattern

`docs/memory/fab-workflow/kit-architecture.md` SHALL be updated:

1. **Directory tree**: The three batch script entries SHALL use the new `batch-fab-*` names
2. **Batch Scripts section**: The naming pattern description SHALL change from `batch-{verb}-{entity}.sh` to `batch-fab-{verb}-{entity}.sh`
3. **Script names in descriptions**: All references to the old names SHALL use the new names

#### Scenario: Directory tree updated
- **GIVEN** the kit-architecture memory file
- **WHEN** the hydrate stage runs
- **THEN** the directory tree listing SHALL show `batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh`, and `batch-fab-archive-change.sh`

#### Scenario: Naming pattern updated
- **GIVEN** the Batch Scripts section in kit-architecture memory
- **WHEN** the hydrate stage runs
- **THEN** the naming pattern SHALL read `batch-fab-{verb}-{entity}.sh`

## Documentation: Spec Update

### Requirement: Architecture spec SHALL reflect new names and pattern

`docs/specs/architecture.md` SHALL be updated:

1. **Script Naming Convention table**: The `batch-` prefix row SHALL be updated to `batch-fab-` with the example updated accordingly
2. **Batch Scripts table**: Script names SHALL use the new `batch-fab-*` names

#### Scenario: Prefix convention table updated
- **GIVEN** the Script Naming Convention table in `docs/specs/architecture.md`
- **WHEN** the apply stage runs
- **THEN** the `batch-` row SHALL show prefix `batch-fab-`, role "Batch orchestration", and example `batch-fab-new-backlog.sh`

#### Scenario: Batch scripts table updated
- **GIVEN** the Batch Scripts table in `docs/specs/architecture.md`
- **WHEN** the apply stage runs
- **THEN** all three script names SHALL use the `batch-fab-*` format

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-arg uses `set -- --list` fallthrough | Confirmed from intake #1 — the `--list` case exists in each script's `case` statement; `set --` is standard bash for rewriting positional params. Clean single-line change. | S:95 R:95 A:95 D:95 |
| 2 | Certain | Rename pattern is `batch-fab-{verb}-{entity}.sh` | Confirmed from intake #2 — user explicitly specified the target pattern | S:95 R:90 A:90 D:95 |
| 3 | Certain | `batch-fab-` is a separate row in prefix convention table | Confirmed from intake #3 — user explicitly chose this when asked | S:95 R:95 A:95 D:95 |
| 4 | Confident | Archive change records not updated | Confirmed from intake #4 — archive records are historical snapshots; updating them would rewrite history | S:70 R:90 A:85 D:80 |
| 5 | Certain | `git mv` for rename to preserve history | Standard practice for renaming tracked files; config shows `git.enabled: true` | S:90 R:95 A:95 D:95 |
| 6 | Certain | Comment headers in scripts updated with new names | Each script has a comment header referencing its own name; these must match the filename | S:90 R:95 A:95 D:95 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
