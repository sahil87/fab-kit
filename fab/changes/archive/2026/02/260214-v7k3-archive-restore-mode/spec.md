# Spec: Archive Restore Mode

**Change**: 260214-v7k3-archive-restore-mode
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/execution-skills.md`, `fab/memory/fab-workflow/change-lifecycle.md`

## Non-Goals

- Reset or modify `.status.yaml` progress during restore — artifacts are preserved as-is
- Restore changes that were never archived (i.e., changes still in `fab/changes/`)
- Modify git branches — restore is purely a folder + index operation

## Execution Skills: Restore Behavior

### Requirement: Restore Subcommand

`/fab-archive` SHALL support a `restore` subcommand with the syntax `/fab-archive restore <change-name>`. The `<change-name>` argument is required and SHALL be resolved via case-insensitive substring matching against folder names in `fab/changes/archive/` (excluding `index.md`).

#### Scenario: Successful Restore

- **GIVEN** a change `260210-a1b2-add-oauth` exists in `fab/changes/archive/`
- **WHEN** the user runs `/fab-archive restore a1b2`
- **THEN** the folder `fab/changes/archive/260210-a1b2-add-oauth/` SHALL be moved to `fab/changes/260210-a1b2-add-oauth/`
- **AND** the entry for `260210-a1b2-add-oauth` SHALL be removed from `fab/changes/archive/index.md`
- **AND** `fab/current` SHALL NOT be modified

#### Scenario: Restore with --switch Flag

- **GIVEN** a change `260210-a1b2-add-oauth` exists in `fab/changes/archive/`
- **WHEN** the user runs `/fab-archive restore a1b2 --switch`
- **THEN** the folder SHALL be moved to `fab/changes/260210-a1b2-add-oauth/`
- **AND** the index entry SHALL be removed
- **AND** `fab/current` SHALL be written with `260210-a1b2-add-oauth`

#### Scenario: No Archived Changes Exist

- **GIVEN** `fab/changes/archive/` is empty or contains only `index.md`
- **WHEN** the user runs `/fab-archive restore anything`
- **THEN** the skill SHALL output: `No archived changes found.`

#### Scenario: No Match Found

- **GIVEN** archived changes exist but none match the argument
- **WHEN** the user runs `/fab-archive restore nonexistent`
- **THEN** the skill SHALL list all archived changes and inform the user no match was found

#### Scenario: Ambiguous Match

- **GIVEN** multiple archived changes match the substring (e.g., `oauth` matches `260210-a1b2-add-oauth` and `260211-c3d4-fix-oauth-redirect`)
- **WHEN** the user runs `/fab-archive restore oauth`
- **THEN** the skill SHALL list all matching changes and ask the user to pick one

### Requirement: Artifact Preservation

The restore operation SHALL preserve all artifacts in the change folder (`.status.yaml`, `brief.md`, `spec.md`, `tasks.md`, `checklist.md`, etc.) without modification. No status reset, no artifact regeneration.

#### Scenario: Restored Change Retains State

- **GIVEN** an archived change with `progress.review: done` and `progress.hydrate: done`
- **WHEN** the change is restored
- **THEN** `.status.yaml` SHALL contain the same progress values as before archiving

### Requirement: Archive Index Cleanup

The restore operation SHALL remove the corresponding entry from `fab/changes/archive/index.md`. If the index file becomes empty (header only, no entries), it SHALL be preserved (not deleted).

#### Scenario: Index Entry Removed

- **GIVEN** `archive/index.md` contains entries for three changes including `260210-a1b2-add-oauth`
- **WHEN** `260210-a1b2-add-oauth` is restored
- **THEN** the index SHALL contain only the remaining two entries
- **AND** the file header SHALL be preserved

### Requirement: Idempotent Restore

The restore operation SHALL be safe to re-run. If the folder already exists in `fab/changes/` (not in archive), the skill SHALL detect this and skip the move, completing only remaining steps (index cleanup).

#### Scenario: Re-run After Partial Restore

- **GIVEN** `260210-a1b2-add-oauth/` already exists in `fab/changes/` (moved previously)
- **AND** the entry still exists in `archive/index.md`
- **WHEN** the user re-runs `/fab-archive restore a1b2`
- **THEN** the move SHALL be skipped
- **AND** the index entry SHALL be removed
- **AND** output SHALL indicate the folder was already restored

### Requirement: Restore Output Format

The restore operation SHALL display a structured summary following the existing `/fab-archive` output conventions.

#### Scenario: Standard Restore Output

- **GIVEN** a successful restore of `260210-a1b2-add-oauth`
- **WHEN** the restore completes
- **THEN** the output SHALL follow this format:
  ```
  Restore: 260210-a1b2-add-oauth

  Moved:    ✓ fab/changes/260210-a1b2-add-oauth/   (or: ✓ already in changes)
  Index:    ✓ entry removed from archive/index.md   (or: — entry not found)
  Pointer:  ✓ fab/current updated                   (or: — not requested)

  Restore complete.

  Next: /fab-switch 260210-a1b2-add-oauth
  ```

## Change Lifecycle: Restore Transition

### Requirement: Lifecycle Transition

The change lifecycle SHALL support an `archived → active` transition via `/fab-archive restore`. This is the inverse of the archive operation. The restored change returns to `fab/changes/` with its original state intact.

#### Scenario: Lifecycle Round-Trip

- **GIVEN** a change that was archived after completing hydrate
- **WHEN** the change is restored
- **THEN** the change SHALL appear in `fab/changes/` listings (e.g., `/fab-switch` with no argument)
- **AND** `/fab-status` SHALL show the change's preserved stage and progress

## Design Decisions

1. **Restore as subcommand of `/fab-archive`, not a new skill**: The restore operation is the logical inverse of archive. Grouping them under the same skill keeps related operations together and avoids skill proliferation.
   - *Why*: Conceptual cohesion — archive and restore are paired operations. Users naturally look for restore under the archive command.
   - *Rejected*: Separate `/fab-restore` skill — adds a new top-level command for a narrow, complementary operation.

2. **`<change-name>` is required for restore (no "restore most recent")**: Restore is a deliberate action on a specific change. There is no "undo last archive" convenience.
   - *Why*: Avoids accidental restores. The user must name what they want back, consistent with the "No Dedicated Abandon Skill" philosophy of deliberate manual actions.
   - *Rejected*: Default to most-recent archive — error-prone, assumes LIFO intent.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Preserve all artifacts without status reset | Brief explicitly states "no status reset"; user wants to resume work, not restart |
| 2 | Confident | Suggest `/fab-switch` as next step (not auto-switch) | Brief says "do not switch by default"; `--switch` is opt-in. Suggesting switch as next step keeps the user oriented |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
