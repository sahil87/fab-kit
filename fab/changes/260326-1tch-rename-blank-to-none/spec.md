# Spec: Rename --blank to --none in fab-switch

**Change**: 260326-1tch-rename-blank-to-none
**Created**: 2026-03-26
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md` (modify)

## CLI: Flag Rename

### Requirement: --none flag replaces --blank

The `fab change switch` command SHALL accept `--none` as the flag to deactivate the current change. The `--blank` flag SHALL be removed. The flag is mutually exclusive with the `<name>` argument.

#### Scenario: Deactivate via --none
- **GIVEN** an active change with `.fab-status.yaml` symlink present
- **WHEN** `fab change switch --none` is executed
- **THEN** the `.fab-status.yaml` symlink is removed
- **AND** stdout outputs `No active change.`

#### Scenario: Deactivate when already inactive
- **GIVEN** no `.fab-status.yaml` symlink exists
- **WHEN** `fab change switch --none` is executed
- **THEN** stdout outputs `No active change (already deactivated).`

#### Scenario: No argument and no flag
- **GIVEN** the user runs `fab change switch` with no argument and no `--none` flag
- **WHEN** the command is parsed
- **THEN** the command exits with error `switch requires <name> or --none`

### Requirement: Go function rename

The `SwitchBlank` function in `internal/change/change.go` SHALL be renamed to `SwitchNone`. All call sites SHALL be updated to use the new name.

#### Scenario: Archive calls SwitchNone
- **GIVEN** the `archive.go` module calls `change.SwitchBlank(fabRoot)` to clear the pointer after archiving
- **WHEN** the rename is applied
- **THEN** the call becomes `change.SwitchNone(fabRoot)`
- **AND** the archive behavior is unchanged

### Requirement: Test updates

All tests referencing `SwitchBlank` or `--blank` SHALL be updated to use `SwitchNone` / `--none`. Test function names, expected output strings, and assertions SHALL reflect the new naming.

#### Scenario: TestSwitchNone replaces TestSwitchBlank
- **GIVEN** `TestSwitchBlank` verifies deactivation behavior
- **WHEN** renamed to `TestSwitchNone`
- **THEN** the test verifies identical behavior with the new function name
- **AND** the expected output still contains `No active change`

#### Scenario: TestSwitchNone_AlreadyDeactivated
- **GIVEN** `TestSwitchBlank_AlreadyBlank` checks the already-blank message
- **WHEN** renamed to `TestSwitchNone_AlreadyDeactivated`
- **THEN** the expected output checks for `already deactivated` instead of `already blank`

## Documentation: Skill and Spec Updates

### Requirement: fab-switch.md skill file update

The `fab/.kit/skills/fab-switch.md` skill file SHALL replace all `--blank` references with `--none`. The heading SHALL become `# /fab-switch [change-name] [--none]`. The deactivation flow section heading SHALL become `Deactivation Flow (--none)`.

#### Scenario: Skill file references updated
- **GIVEN** `fab-switch.md` contains `--blank` in the heading, argument docs, deactivation flow, and output sections
- **WHEN** the rename is applied
- **THEN** all occurrences of `--blank` become `--none`
- **AND** no instances of `--blank` remain in the file

### Requirement: _cli-fab.md reference update

The `fab/.kit/skills/_cli-fab.md` CLI reference SHALL update the switch row from `switch <name> | --blank` to `switch <name> | --none`.

#### Scenario: CLI reference updated
- **GIVEN** the switch row in the command reference table contains `--blank`
- **WHEN** the rename is applied
- **THEN** the row reads `switch <name> \| --none`

### Requirement: SPEC-fab-switch.md update

The `docs/specs/skills/SPEC-fab-switch.md` spec file SHALL replace all `--blank` references with `--none`.

#### Scenario: Spec file references updated
- **GIVEN** `SPEC-fab-switch.md` contains references to `--blank`
- **WHEN** the rename is applied
- **THEN** all occurrences of `--blank` become `--none`

## Memory: Change Lifecycle Update

### Requirement: change-lifecycle.md update

The `docs/memory/fab-workflow/change-lifecycle.md` memory file SHALL replace all `--blank` references with `--none`. The output string references SHALL change from `already blank` to `already deactivated`.

#### Scenario: Memory file references updated
- **GIVEN** `change-lifecycle.md` describes the deactivation lifecycle using `--blank`
- **WHEN** the rename is applied
- **THEN** all occurrences of `--blank` become `--none`
- **AND** references to "already blank" become "already deactivated"

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename to `--none` (not bare `fab-switch`) | Confirmed from intake #1 — bare `fab-switch` already lists changes | S:80 R:90 A:95 D:95 |
| 2 | Certain | Rename Go function `SwitchBlank` → `SwitchNone` | Confirmed from intake #2 — internal consistency | S:90 R:95 A:90 D:95 |
| 3 | Certain | Update output string to `already deactivated` | Confirmed from intake #3 — "blank" should not appear in output | S:85 R:95 A:90 D:90 |
| 4 | Certain | Do not update archive files | Confirmed from intake #4 — archives are frozen historical records | S:90 R:95 A:95 D:95 |
| 5 | Certain | Tests must be updated for CLI changes | Confirmed from intake #5 — constitution VII | S:95 R:90 A:95 D:95 |
| 6 | Certain | `_cli-fab.md` must be updated | Confirmed from intake #6 — constitution additional constraints | S:95 R:90 A:95 D:95 |
| 7 | Certain | `SPEC-fab-switch.md` must be updated | Constitution additional constraints — skill changes require spec update | S:95 R:90 A:95 D:95 |
| 8 | Certain | `fab-switch.md` in `fab/.kit/skills/` is the canonical source | `context.md` — always edit source, not deployed copies | S:95 R:90 A:95 D:95 |

8 assumptions (8 certain, 0 confident, 0 tentative, 0 unresolved).
