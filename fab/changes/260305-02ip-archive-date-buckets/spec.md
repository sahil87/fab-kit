# Spec: Archive Date Buckets

**Change**: 260305-02ip-archive-date-buckets
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`

## Non-Goals

- Changing the `archive/index.md` format — it remains a flat list (no path nesting)
- Adding git archaeology or commit-date parsing — date is always from the folder name prefix
- Creating a separate command — all changes are within `archiveman.sh`

## Archive Path: Date-Bucketed Destination

### Requirement: Date-Bucketed Archive Path

`archiveman.sh archive` SHALL move change folders to `fab/changes/archive/yyyy/mm/{name}` instead of `fab/changes/archive/{name}`.

The `yyyy` and `mm` components SHALL be derived from the folder name's `YYMMDD` prefix:
- Extract characters 1-2 as `YY`, characters 3-4 as `MM`
- Prefix `20` to `YY` for the 4-digit year
- Create intermediate directories as needed (`mkdir -p`)

#### Scenario: Archive a change with standard folder name

- **GIVEN** a change folder `260305-02ip-archive-date-buckets` exists in `fab/changes/`
- **WHEN** `archiveman.sh archive 02ip --description "..."` is run
- **THEN** the folder SHALL be moved to `fab/changes/archive/2026/03/260305-02ip-archive-date-buckets`
- **AND** intermediate directories `archive/2026/03/` SHALL be created if they don't exist

#### Scenario: Archive changes from different months

- **GIVEN** two changes: `260207-09sj-autonomy-framework` and `260305-02ip-archive-date-buckets`
- **WHEN** both are archived
- **THEN** they SHALL be placed in `archive/2026/02/` and `archive/2026/03/` respectively

### Requirement: Date Parsing Helper

A helper function SHALL extract `yyyy` and `mm` from a folder name. The function MUST:
- Accept a folder name as argument
- Return the 4-digit year and 2-digit month (e.g., `2026 03`)
- Use only the first 6 characters of the name (the `YYMMDD` prefix)

#### Scenario: Parse standard folder name

- **GIVEN** folder name `260305-02ip-archive-date-buckets`
- **WHEN** the date parser is invoked
- **THEN** it SHALL return year `2026` and month `03`

#### Scenario: Parse older folder name

- **GIVEN** folder name `251215-a1b2-some-change`
- **WHEN** the date parser is invoked
- **THEN** it SHALL return year `2025` and month `12`

### Requirement: Collision Detection Update

The collision check SHALL use the date-bucketed path. If `archive/yyyy/mm/{name}` already exists, the script SHALL exit with an error.

#### Scenario: Collision in bucketed path

- **GIVEN** `archive/2026/03/260305-02ip-archive-date-buckets` already exists
- **WHEN** archiving `260305-02ip-archive-date-buckets` is attempted
- **THEN** the script SHALL exit with error: `Archive destination already exists: ...`

## Archive Restore: Nested Structure Resolution

### Requirement: Recursive Archive Resolution

`resolve_archive` SHALL scan the nested `archive/yyyy/mm/` structure instead of only `archive/*/`. The function MUST find change folders at any depth within the archive hierarchy.

#### Scenario: Resolve a change in nested archive

- **GIVEN** `archive/2026/02/260207-09sj-autonomy-framework/` exists
- **WHEN** `archiveman.sh restore 09sj` is run
- **THEN** the change SHALL be found and restored to `fab/changes/260207-09sj-autonomy-framework`

#### Scenario: Resolve with partial name match

- **GIVEN** `archive/2026/03/260305-02ip-archive-date-buckets/` exists
- **WHEN** `archiveman.sh restore archive-date` is run
- **THEN** the change SHALL be found via case-insensitive substring match

#### Scenario: Multiple matches in nested archive

- **GIVEN** two archived changes both match the search term
- **WHEN** restore is attempted
- **THEN** the script SHALL exit with `Multiple archives match "..."` error listing the matches

## Archive List: Nested Structure Traversal

### Requirement: List Traverses Nested Structure

`archiveman.sh list` SHALL walk the `archive/yyyy/mm/` hierarchy and output change folder names (without the `yyyy/mm/` prefix path). The output contract MUST remain the same: one folder name per line.

#### Scenario: List with nested archive

- **GIVEN** archive contains `2026/02/260207-09sj-autonomy-framework/` and `2026/03/260305-02ip-archive-date-buckets/`
- **WHEN** `archiveman.sh list` is run
- **THEN** the output SHALL be:
  ```
  260207-09sj-autonomy-framework
  260305-02ip-archive-date-buckets
  ```
- **AND** no `yyyy/mm/` prefix path SHALL appear in the output

## Archive Index: Backfill Update

### Requirement: Backfill Scans Nested Structure

The `backfill_index` function SHALL scan the nested `archive/yyyy/mm/` structure instead of only `archive/*/` when checking for unindexed archived folders.

#### Scenario: Backfill finds nested unindexed folder

- **GIVEN** `archive/2026/02/260207-09sj-autonomy-framework/` exists but is not in `archive/index.md`
- **WHEN** backfill runs during an archive operation
- **THEN** an entry SHALL be appended to `index.md` for the missing folder

## One-Time Migration

### Requirement: Migrate Flat Archive to Date Buckets

A migration function within `archiveman.sh` SHALL restructure existing flat archive entries into the `yyyy/mm/` hierarchy.

The migration SHALL:
1. Scan `fab/changes/archive/` for directories directly under `archive/` that are not `yyyy/` directories and not `index.md`
2. Parse the `YYMMDD` prefix from each folder name
3. Create `archive/yyyy/mm/` and move the folder into the bucket
4. Leave `archive/index.md` untouched

#### Scenario: Migrate flat archive entries

- **GIVEN** `archive/260207-09sj-autonomy-framework/` exists as a flat entry
- **WHEN** `archiveman.sh migrate` is run
- **THEN** it SHALL be moved to `archive/2026/02/260207-09sj-autonomy-framework/`
- **AND** `mkdir -p archive/2026/02/` SHALL be called first

#### Scenario: Migration is idempotent

- **GIVEN** all archive entries are already in `yyyy/mm/` buckets
- **WHEN** `archiveman.sh migrate` is run
- **THEN** no folders SHALL be moved
- **AND** the script SHALL exit successfully

#### Scenario: Mixed flat and bucketed entries

- **GIVEN** some entries are flat and some are already bucketed
- **WHEN** `archiveman.sh migrate` is run
- **THEN** only the flat entries SHALL be moved
- **AND** already-bucketed entries SHALL not be affected

### Requirement: Migration Subcommand

The migration SHALL be exposed as `archiveman.sh migrate` subcommand.

#### Scenario: Invoke migration

- **GIVEN** the archive contains flat entries
- **WHEN** `archiveman.sh migrate` is run
- **THEN** it SHALL output a summary of moved folders
- **AND** exit 0 on success

## Design Decisions

1. **Date parsing from folder name prefix, not git metadata**
   - *Why*: Folder names already encode creation date as `YYMMDD`. Parsing is deterministic and requires no git operations.
   - *Rejected*: Git commit date — requires `git log` on each folder, slow, and the folder name date is the authoritative creation date.

2. **Year prefix hardcoded to `20`**
   - *Why*: Folder names use 2-digit year (`YY`). Prepending `20` is valid through 2099 and avoids over-engineering century detection.
   - *Rejected*: Dynamic century detection — unnecessary complexity for the next 73 years.

3. **Migration as `migrate` subcommand, not auto-migration**
   - *Why*: Explicit invocation lets users choose when to restructure. Idempotent by design. Avoids surprise side effects during normal archive operations.
   - *Rejected*: Auto-migrate on first bucketed archive — surprising behavior, violates Principle III (idempotent operations should not have hidden side effects on first run).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Date source is folder name YYMMDD prefix | Confirmed from intake #1 — user explicitly decided | S:95 R:90 A:95 D:95 |
| 2 | Certain | Archive path is `archive/yyyy/mm/{name}` | Confirmed from intake #2 — user chose this structure | S:95 R:85 A:90 D:95 |
| 3 | Certain | Built into archiveman.sh, not a separate command | Confirmed from intake #3 — user decided against separate lifecycle command | S:95 R:90 A:90 D:90 |
| 4 | Certain | Migration restructures existing flat entries | Confirmed from intake #4 — user requested migration | S:90 R:85 A:85 D:90 |
| 5 | Certain | Year prefix is `20` + 2-digit year | Upgraded from intake Confident #5 — 2-digit year convention is unambiguous in this codebase | S:85 R:95 A:95 D:90 |
| 6 | Confident | `archive/index.md` stays flat (no path changes) | Confirmed from intake #6 — index is logical, not filesystem mirror | S:70 R:90 A:85 D:80 |
| 7 | Confident | Migration exposed as `migrate` subcommand | Codebase pattern: archiveman uses subcommand dispatch. Explicit > implicit | S:70 R:90 A:85 D:80 |
| 8 | Confident | Migration is idempotent and safe to re-run | Constitution Principle III requires idempotent operations | S:70 R:85 A:90 D:85 |
| 9 | Confident | `resolve_archive` uses recursive scan (find or glob) | Necessary to find folders at `archive/yyyy/mm/` depth | S:75 R:90 A:90 D:85 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
