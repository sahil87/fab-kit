# Spec: Add changeman.sh list Subcommand

**Change**: 260226-w4fw-add-changeman-list-subcommand
**Created**: 2026-02-26
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`

## Change Management: list Subcommand

### Requirement: Enumerate Active Changes

`changeman.sh list` SHALL scan `fab/changes/` for directories, exclude `archive/`, and output one structured line per change to stdout. Each line SHALL follow the format `name:display_stage:display_state`.

- `name` — the change folder name (e.g., `260226-tnr8-coverage-scoring`)
- `display_stage` — derived from `stageman.sh display-stage <status_file>` (the "where you are" stage)
- `display_state` — the state of that display stage (`active`, `done`, `pending`)

Lines SHALL be output in directory enumeration order (glob order).

#### Scenario: Multiple Active Changes

- **GIVEN** `fab/changes/` contains `260226-a1b2-change-one/` and `260226-c3d4-change-two/`, each with valid `.status.yaml`
- **AND** `fab/changes/archive/` exists (and is excluded)
- **WHEN** `changeman.sh list` is executed
- **THEN** stdout contains exactly two lines, one per change, in format `name:display_stage:display_state`
- **AND** the exit code is 0

#### Scenario: Single Active Change

- **GIVEN** `fab/changes/` contains one change directory with a valid `.status.yaml` at `spec:active`
- **WHEN** `changeman.sh list` is executed
- **THEN** stdout contains exactly one line: `{name}:spec:active`
- **AND** the exit code is 0

### Requirement: Empty Result on No Changes

When `fab/changes/` contains no change directories (or only `archive/`), `changeman.sh list` SHALL exit 0 with no stdout output.

#### Scenario: No Changes Exist

- **GIVEN** `fab/changes/` is empty (or contains only `archive/`)
- **WHEN** `changeman.sh list` is executed
- **THEN** stdout is empty
- **AND** the exit code is 0

#### Scenario: Changes Directory Does Not Exist

- **GIVEN** `fab/changes/` does not exist
- **WHEN** `changeman.sh list` is executed
- **THEN** stderr contains `fab/changes/ not found.`
- **AND** the exit code is 1

### Requirement: Graceful Handling of Missing `.status.yaml`

When a change directory lacks `.status.yaml`, `changeman.sh list` SHALL output `name:unknown:unknown` for that entry and emit a warning to stderr. It SHALL NOT fail the entire list.

#### Scenario: One Change Missing Status

- **GIVEN** `fab/changes/` contains `260226-a1b2-good/` (with `.status.yaml`) and `260226-c3d4-bad/` (without `.status.yaml`)
- **WHEN** `changeman.sh list` is executed
- **THEN** stdout contains two lines: one with derived stage/state for `good`, one with `260226-c3d4-bad:unknown:unknown`
- **AND** stderr contains a warning mentioning `260226-c3d4-bad`
- **AND** the exit code is 0

### Requirement: Archive Listing via `--archive` Flag

`changeman.sh list --archive` SHALL scan `fab/changes/archive/` instead of `fab/changes/` (excluding `archive/`). The output format and error handling are identical to the default mode.

#### Scenario: List Archived Changes

- **GIVEN** `fab/changes/archive/` contains `260220-x1y2-old-change/` with a valid `.status.yaml`
- **WHEN** `changeman.sh list --archive` is executed
- **THEN** stdout contains one line: `260220-x1y2-old-change:hydrate:done`
- **AND** the exit code is 0

#### Scenario: No Archived Changes

- **GIVEN** `fab/changes/archive/` is empty or does not exist
- **WHEN** `changeman.sh list --archive` is executed
- **THEN** stdout is empty
- **AND** the exit code is 0

### Requirement: Stage Derivation via stageman.sh

Stage and state values MUST be derived by calling `stageman.sh display-stage <status_file>`, which returns the `stage:state` pair. `changeman.sh list` SHALL NOT implement its own stage derivation logic.

#### Scenario: Display-Stage Integration

- **GIVEN** a change directory contains `.status.yaml` with `progress.spec: active`
- **WHEN** `changeman.sh list` processes this change
- **THEN** `stageman.sh display-stage` is invoked with the `.status.yaml` path
- **AND** the output uses the stage:state returned by stageman (e.g., `spec:active`)

### Requirement: Updated Help Text

`changeman.sh --help` SHALL include `list` in the USAGE section and SUBCOMMANDS section.

#### Scenario: Help Includes list

- **GIVEN** the user runs `changeman.sh --help`
- **WHEN** the output is displayed
- **THEN** the USAGE section includes `changeman.sh list [--archive]`
- **AND** the SUBCOMMANDS section includes a `list` entry with a brief description

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Output format `name:display_stage:display_state` one per line | Confirmed from intake #1 — discussed, matches changeman's existing stdout conventions | S:90 R:90 A:90 D:95 |
| 2 | Certain | Exclude archive by default, `--archive` flag for archived | Confirmed from intake #2 — primary use case is active changes | S:85 R:90 A:90 D:90 |
| 3 | Certain | Use `stageman.sh display-stage` for stage derivation | Confirmed from intake #3 — display-stage already encapsulates "where you are" logic, returns `stage:state` format | S:85 R:85 A:95 D:90 |
| 4 | Confident | Exit 0 with empty stdout when no changes exist | Confirmed from intake #4 — consistent with CLI conventions; callers check for empty stdout | S:75 R:90 A:80 D:80 |
| 5 | Confident | Missing `.status.yaml` outputs `name:unknown:unknown` with stderr warning | Confirmed from intake #5 — defensive; don't fail entire list for one corrupted change | S:70 R:85 A:85 D:75 |
| 6 | Confident | `fab/changes/` not found exits 1 (not silent empty) | New — unlike "no changes" (valid empty state), missing directory indicates setup issue; error is appropriate | S:70 R:85 A:80 D:80 |
| 7 | Certain | Output order follows glob enumeration order | New — no sorting requirement; glob order is chronological by construction (YYMMDD prefix) | S:80 R:95 A:90 D:95 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
