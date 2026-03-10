# Spec: Regroup CLI Subcommands

**Change**: 260306-yzxj-regroup-cli-subcommands
**Created**: 2026-03-06
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Renaming the `/fab-archive` skill — only the underlying CLI commands change
- Moving `score`, `status`, or `preflight` under a shared parent command — they stay top-level
- Adding backward-compatibility shims for `fab archive` — skills are the only callers and ship atomically

## CLI: Move Archive Under Change

### Requirement: Archive Subcommand of Change

The `archive` command SHALL be registered as a subcommand of `change` instead of the root command. The CLI signature SHALL be `fab change archive <change> --description "..."`.

#### Scenario: Archive a completed change via new path
- **GIVEN** a completed change `yzxj` with `hydrate: done`
- **WHEN** `fab change archive yzxj --description "Regrouped CLI"` is invoked
- **THEN** the change is archived identically to the old `fab archive` behavior
- **AND** structured YAML is output to stdout

#### Scenario: Old `fab archive` path no longer works
- **GIVEN** the updated binary is installed
- **WHEN** `fab archive yzxj --description "test"` is invoked
- **THEN** Cobra returns an "unknown command" error

### Requirement: Restore Subcommand of Change

The `restore` command SHALL be registered as a direct subcommand of `change` (not nested under `archive`). The CLI signature SHALL be `fab change restore <change> [--switch]`.

#### Scenario: Restore an archived change
- **GIVEN** an archived change `yzxj` in `fab/changes/archive/`
- **WHEN** `fab change restore yzxj --switch` is invoked
- **THEN** the change is restored to `fab/changes/` and `fab/current` is updated
- **AND** structured YAML is output to stdout

### Requirement: Archive-List Subcommand of Change

The `archive-list` command SHALL be registered as a subcommand of `change`. The CLI signature SHALL be `fab change archive-list`. This uses a hyphenated name to avoid ambiguity with `fab change archive <change>`.

#### Scenario: List archived changes
- **GIVEN** archived changes exist in `fab/changes/archive/`
- **WHEN** `fab change archive-list` is invoked
- **THEN** archived folder names are printed one per line (same output as old `fab archive list`)

### Requirement: Remove Top-Level Archive Command

The `archiveCmd()` function SHALL no longer be added to the root command in `main.go`. The root command's `AddCommand` list SHALL NOT include `archiveCmd()`.

#### Scenario: Root command help omits archive
- **GIVEN** the updated binary
- **WHEN** `fab --help` is invoked
- **THEN** `archive` does not appear in the list of available commands

### Requirement: Go Implementation Structure

The archive command functions (`archiveCmd`, `archiveRestoreCmd`, `archiveListCmd`) in `src/go/fab/cmd/fab/archive.go` SHALL be refactored:

1. `archiveCmd()` → `changeArchiveCmd()` — registered under `changeCmd()` as `archive`
2. `archiveRestoreCmd()` → `changeRestoreCmd()` — registered under `changeCmd()` as `restore`
3. `archiveListCmd()` → `changeArchiveListCmd()` — registered under `changeCmd()` as `archive-list`

The internal `archive` package (`internal/archive/`) SHALL NOT change — only the Cobra command wiring in `cmd/fab/` is affected.

#### Scenario: Change command lists new subcommands
- **GIVEN** the updated binary
- **WHEN** `fab change --help` is invoked
- **THEN** the output lists `archive`, `restore`, `archive-list`, `new`, `rename`, `switch`, `list`, `resolve`

## Documentation: Regroup `_scripts.md` by Concern

### Requirement: Three-Section Structure

`fab/.kit/skills/_scripts.md` SHALL reorganize its Command Reference and detailed sections into three concern groups:

1. **Change Lifecycle** — `fab change` (including archive, restore, archive-list subcommands)
2. **Pipeline & Status** — `fab status`, `fab score`, `fab preflight`
3. **Plumbing** — `fab resolve`, `fab log`, `fab runtime`

#### Scenario: Agent reads _scripts.md
- **GIVEN** an agent loading `_scripts.md` for context
- **WHEN** looking for change lifecycle commands
- **THEN** `fab change` (with all subcommands including archive/restore/archive-list) appears under the "Change Lifecycle" heading

#### Scenario: Pipeline commands are grouped
- **GIVEN** an agent reading `_scripts.md`
- **WHEN** looking for pipeline operational commands
- **THEN** `status`, `score`, and `preflight` appear under "Pipeline & Status"

### Requirement: Updated Command Reference Table

The Command Reference table SHALL reflect the new grouping with section headers and updated `fab change` description that includes archive/restore/archive-list.

#### Scenario: Command reference table
- **GIVEN** the updated `_scripts.md`
- **WHEN** reading the Command Reference table
- **THEN** `fab archive` no longer appears as a separate row
- **AND** `fab change` row description mentions archive, restore, archive-list alongside new, rename, switch, list

### Requirement: Updated fab archive Section

The `## fab archive` section SHALL be removed. Its content SHALL be incorporated into the `## fab change` section as additional subcommands in the subcommand table.

#### Scenario: No standalone archive section
- **GIVEN** the updated `_scripts.md`
- **WHEN** searching for `## fab archive`
- **THEN** no such heading exists
- **AND** archive/restore/archive-list appear in the `fab change` subcommand table

## Skill References: Update `fab-archive.md`

### Requirement: Updated CLI Invocations

All `fab archive` CLI invocations in `fab/.kit/skills/fab-archive.md` SHALL be updated to `fab change archive`, `fab change restore`, and `fab change archive-list` respectively.

#### Scenario: Archive mode invocation
- **GIVEN** the updated `fab-archive.md`
- **WHEN** the skill runs the archive command
- **THEN** it invokes `fab/.kit/bin/fab change archive <change> --description "..."`

#### Scenario: Restore mode invocation
- **GIVEN** the updated `fab-archive.md`
- **WHEN** the skill runs the restore command
- **THEN** it invokes `fab/.kit/bin/fab change restore <change-name> [--switch]`

#### Scenario: List invocation in error handling
- **GIVEN** the updated `fab-archive.md`
- **WHEN** the skill needs to list archived changes
- **THEN** it invokes `fab/.kit/bin/fab change archive-list`

## Spec and Doc References: Update Cross-References

### Requirement: Update SPEC-fab-archive.md

`docs/specs/skills/SPEC-fab-archive.md` SHALL update all `fab archive` CLI references to `fab change archive`, `fab change restore`, `fab change archive-list`.

#### Scenario: SPEC flow diagram
- **GIVEN** the updated SPEC
- **WHEN** reading the flow diagram
- **THEN** Bash tool lines show `fab change archive`, `fab change restore`, `fab change archive-list`

### Requirement: Update kit-architecture.md

`docs/memory/fab-workflow/kit-architecture.md` SHALL update the command reference line from `fab archive <change> --description "..." | restore | list` to reflect the new paths under `fab change`.

#### Scenario: Memory command reference
- **GIVEN** the updated `kit-architecture.md`
- **WHEN** reading the command surface
- **THEN** archive operations appear under `fab change` (not as a separate top-level entry)

### Requirement: Update Parity Tests

`src/go/fab/test/parity/archive_test.go` SHALL update `runGo` calls from `"archive", ...` to `"change", "archive", ...` (and similarly for restore/list).

#### Scenario: Parity test invocations
- **GIVEN** the updated test file
- **WHEN** parity tests run
- **THEN** Go backend is invoked via `change archive`, `change restore`, `change archive-list`

## Deprecated Requirements

### Top-Level `fab archive` Command

**Reason**: Archive operations are change lifecycle operations and belong under `fab change`.
**Migration**: `fab archive <args>` → `fab change archive <args>`, `fab archive restore <args>` → `fab change restore <args>`, `fab archive list` → `fab change archive-list`.

## Design Decisions

1. **`restore` as direct child of `change`**: `fab change restore` rather than `fab change archive restore`.
   - *Why*: Flatter hierarchy is more ergonomic. Archive and restore are peer lifecycle operations (one moves to archive, one moves back). Nesting restore under archive adds a level with no benefit.
   - *Rejected*: `fab change archive restore` — deeper nesting for no gain; `restore` is conceptually the inverse of `archive`, not a sub-operation of it.

2. **Hyphenated `archive-list`**: `fab change archive-list` rather than `fab change archive --list`.
   - *Why*: Consistent with Cobra subcommand pattern. `archive` takes a positional `<change>` arg, so adding `--list` creates ambiguity (is `--list` a flag to `archive` or a modifier?). A separate subcommand is unambiguous.
   - *Rejected*: `fab change archive --list` — flag conflicts with positional arg; `fab change list --archive` — that already exists for a different purpose (listing changes with archive info).

3. **Documentation-only grouping (no code changes for `_scripts.md` reorg)**: The three-section structure in `_scripts.md` is a documentation reorganization only — it does not create new Cobra command groups or parent commands for Pipeline & Status or Plumbing.
   - *Why*: The grouping serves discoverability in the doc, not CLI hierarchy. Adding parent commands for `pipeline` or `plumbing` would break all existing skill invocations for no ergonomic gain.
   - *Rejected*: Creating `fab pipeline status`, `fab plumbing resolve` etc. — breaks everything, no user benefit.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Regroup `_scripts.md` into three concern sections (Lifecycle, Pipeline, Plumbing) | Confirmed from intake #1 — user explicitly chose combined A+B approach | S:95 R:90 A:90 D:95 |
| 2 | Certain | Move `fab archive` under `fab change` as subcommands | Confirmed from intake #2 — user confirmed this structural change | S:95 R:70 A:85 D:90 |
| 3 | Certain | Keep `score` as a top-level command, not under `status` | Confirmed from intake #3 — distinct flags/behavior | S:90 R:85 A:80 D:85 |
| 4 | Confident | `restore` as direct `fab change restore` | Confirmed from intake #4 — flatter is more ergonomic | S:70 R:80 A:75 D:65 |
| 5 | Confident | Use `archive-list` hyphenated subcommand | Confirmed from intake #5 — consistent with Cobra patterns | S:65 R:85 A:70 D:60 |
| 6 | Certain | No backward compatibility shim for old `fab archive` path | Confirmed from intake #6 — atomic update with skills | S:85 R:80 A:90 D:90 |
| 7 | Certain | Internal `archive` package unchanged — only Cobra wiring moves | Codebase inspection confirms clean separation between `internal/archive/` (logic) and `cmd/fab/archive.go` (CLI wiring) | S:90 R:95 A:95 D:90 |
| 8 | Certain | Skill name `/fab-archive` stays the same — only CLI commands change | Skill identity is independent of underlying CLI paths; changing skill name would break all pipeline references | S:90 R:60 A:90 D:95 |
| 9 | Confident | Memory files beyond kit-architecture.md are out of scope for code changes | execution-skills.md, change-lifecycle.md mention `fab archive` but as narrative history; these are updated via hydrate, not manual edits | S:70 R:85 A:75 D:70 |

9 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
