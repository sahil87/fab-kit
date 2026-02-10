# Spec: Auto-activate after /fab-discuss when no current change

**Change**: 260210-zr1f-discuss-auto-activate-when-no-current
**Created**: 2026-02-10
**Affected docs**: `fab/docs/fab-workflow/change-lifecycle.md`, `fab/docs/fab-workflow/planning-skills.md`

## fab-workflow: Conditional Activation in /fab-discuss

### Requirement: Offer activation when fab/current is empty

After `/fab-discuss` (new change mode) generates the proposal and displays the summary, the skill SHALL check whether `fab/current` exists and is non-empty. If `fab/current` is empty or does not exist, the skill SHALL prompt the user to activate the new change. If the user accepts, the skill SHALL call `/fab-switch` internally (same pattern as `/fab-new`) to write `fab/current` and handle git branch integration.

#### Scenario: New change with no active change — user accepts

- **GIVEN** `/fab-discuss` is in new change mode and `fab/current` does not exist or is empty
- **WHEN** the proposal summary has been displayed
- **THEN** the skill prompts: "No active change — set {name} as active?"
- **AND** the user accepts
- **THEN** the skill calls `/fab-switch` internally to write `fab/current` and handle branch integration
- **AND** the "Next:" line shows `/fab-continue or /fab-ff` (no `/fab-switch` step)

#### Scenario: New change with no active change — user declines

- **GIVEN** `/fab-discuss` is in new change mode and `fab/current` does not exist or is empty
- **WHEN** the proposal summary has been displayed
- **THEN** the skill prompts: "No active change — set {name} as active?"
- **AND** the user declines
- **THEN** `fab/current` is NOT written
- **AND** the "Next:" line shows `/fab-switch {name} to make it active, then /fab-continue or /fab-ff`

#### Scenario: New change with existing active change

- **GIVEN** `/fab-discuss` is in new change mode and `fab/current` points to a valid, different change
- **WHEN** the proposal summary has been displayed
- **THEN** the skill does NOT prompt for activation (preserves current work context)
- **AND** the "Next:" line shows `/fab-switch {name} to make it active, then /fab-continue or /fab-ff`

#### Scenario: Refine mode (active change exists)

- **GIVEN** `/fab-discuss` is in refine mode (working on the active change)
- **WHEN** the proposal is updated
- **THEN** no activation prompt is shown (the change is already active)
- **AND** the "Next:" line shows `/fab-continue or /fab-ff`

#### Scenario: Internal /fab-switch fails

- **GIVEN** the user accepted the activation offer
- **WHEN** `/fab-switch` fails (e.g., git branch creation error)
- **THEN** the proposal is already saved (non-fatal)
- **AND** the skill reports the error and suggests manual `/fab-switch`

### Requirement: Update Key Properties and Key Differences tables

The Key Properties table in the skill file SHALL update "Switches active change?" from "No — never writes to `fab/current`" to "Conditionally — offers when `fab/current` is empty (calls `/fab-switch` internally)".

The Key Differences table SHALL update the "Sets active change" row for `fab-discuss` from "No — must `/fab-switch`" to "Conditionally — offers when no active change".

The "Git integration" row SHALL update from "None" to "Conditionally — via internal `/fab-switch` when activation is accepted".

### Requirement: Update Next Steps table in _context.md

The Next Steps lookup table SHALL add a new row for the activated case:

| `/fab-discuss` (new, activated) | proposal done | `Next: /fab-continue or /fab-ff (fast-forward all planning)` |

The existing `/fab-discuss` (new change) row remains for the declined/existing-active-change case.

### Requirement: Update centralized docs

`fab/docs/fab-workflow/change-lifecycle.md` SHALL update the `fab/current` lifecycle to reflect that `/fab-discuss` conditionally writes via internal `/fab-switch` when no active change exists. The "Not written" bullet SHALL be replaced with a conditional description.

`fab/docs/fab-workflow/planning-skills.md` SHALL update the `/fab-discuss` section to reflect conditional activation in the output description, key differences table, and behavioral description.

Both docs SHALL receive a changelog entry for this change.
