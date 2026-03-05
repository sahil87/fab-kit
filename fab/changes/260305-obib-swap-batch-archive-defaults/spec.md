# Spec: Swap batch-fab-archive-change defaults

**Change**: 260305-obib-swap-batch-archive-defaults
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`

## CLI: Default Behavior

### Requirement: No-Argument Default SHALL Archive All

When `batch-fab-archive-change.sh` is invoked with no arguments, the script SHALL behave as if `--all` was passed — archiving all eligible changes (hydrate done or skipped).

#### Scenario: No arguments invokes archive-all
- **GIVEN** there are 3 archivable changes in `fab/changes/`
- **WHEN** the user runs `batch-fab-archive-change.sh` with no arguments
- **THEN** all 3 changes are resolved and passed to the Claude archive prompt
- **AND** the output begins with "Archiving 3 changes..."

#### Scenario: No arguments with no archivable changes
- **GIVEN** there are no changes with hydrate done or skipped
- **WHEN** the user runs `batch-fab-archive-change.sh` with no arguments
- **THEN** the script exits with code 1
- **AND** stderr contains "No archivable changes found."

### Requirement: --list Flag SHALL Show Archivable Changes

The `--list` flag SHALL remain available as an explicit option to preview archivable changes without archiving them.

#### Scenario: Explicit --list shows archivable changes
- **GIVEN** there are 2 archivable changes in `fab/changes/`
- **WHEN** the user runs `batch-fab-archive-change.sh --list`
- **THEN** the output lists both changes under "Archivable changes (hydrate done|skipped):"
- **AND** the script exits with code 0

#### Scenario: --list with no archivable changes
- **GIVEN** there are no changes with hydrate done or skipped
- **WHEN** the user runs `batch-fab-archive-change.sh --list`
- **THEN** the output shows "(none)" under the header
- **AND** the script exits with code 0

### Requirement: Usage Text SHALL Reflect New Default

The usage/help text (shown via `-h` or `--help`) SHALL document that:
1. Running with no arguments archives all eligible changes (same as `--all`)
2. `--list` is available to preview archivable changes first

#### Scenario: Help text documents new default
- **GIVEN** the script is invoked
- **WHEN** the user runs `batch-fab-archive-change.sh --help`
- **THEN** the usage text indicates the default (no-argument) behavior is archive-all
- **AND** `--list` is documented as an explicit preview option

### Requirement: Existing Flags SHALL Continue Working

All other flags and argument modes SHALL remain unchanged: `--all` (explicit archive-all), `--list` (explicit list), `-h`/`--help` (usage), and positional `<change>` arguments (archive specific changes).

#### Scenario: Explicit --all still works
- **GIVEN** there are 2 archivable changes
- **WHEN** the user runs `batch-fab-archive-change.sh --all`
- **THEN** both changes are archived (identical to no-argument behavior)

#### Scenario: Positional arguments still work
- **GIVEN** change `v3rn` is archivable
- **WHEN** the user runs `batch-fab-archive-change.sh v3rn`
- **THEN** only `v3rn` is archived

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only the zero-argument default changes; both `--list` and `--all` flags remain | Confirmed from intake #1 — directly stated in user request | S:90 R:95 A:95 D:95 |
| 2 | Certain | Usage/help text updated to reflect new default | Confirmed from intake #2 — standard practice for CLI default changes | S:85 R:95 A:90 D:95 |
| 3 | Certain | No confirmation prompt needed before archiving with no args | Upgraded from intake #3 (Confident → Certain) — `--all` already runs without confirmation; the script delegates to Claude which handles `/fab-archive` interactively per change | S:80 R:90 A:90 D:90 |
| 4 | Certain | The change is limited to the fallback line and usage text — no structural refactoring | Codebase shows a clean `set -- --list` on line 83 that maps directly to `set -- --all` | S:90 R:95 A:95 D:95 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
