# Spec: Add --no-switch Flag to fab-new

**Change**: 260212-r7k3-add-no-switch-flag
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`

## Planning Skills: --no-switch Flag

### Requirement: Optional --no-switch Argument

`/fab-new` SHALL accept an optional `--no-switch` flag. When present, Step 8 (the internal `/fab-switch` invocation) SHALL be skipped entirely. All other steps (folder creation, `.status.yaml` initialization, `brief.md` generation, SRAD scoring, marking brief complete) SHALL execute unchanged.

#### Scenario: Creating a change with --no-switch

- **GIVEN** an initialized fab project with an existing active change
- **WHEN** the user invokes `/fab-new "some description" --no-switch`
- **THEN** the change folder, `.status.yaml`, and `brief.md` are created as normal
- **AND** `fab/current` is NOT modified (retains the previously active change)
- **AND** no git branch is created or checked out
- **AND** the output displays the change summary without a `Branch:` line

#### Scenario: Creating a change without --no-switch (default behavior)

- **GIVEN** an initialized fab project
- **WHEN** the user invokes `/fab-new "some description"` (no `--no-switch` flag)
- **THEN** the existing behavior is unchanged — `/fab-switch` is called internally to activate the change and handle branch integration

### Requirement: Contextual Next Line

When `--no-switch` is used, the output's `Next:` line SHALL use the "not activated" pattern (consistent with `/fab-discuss` when the change is not activated):

```
Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff
```

When `--no-switch` is NOT used, the existing `Next:` line SHALL be preserved:

```
Next: /fab-continue or /fab-ff (fast-forward all planning)
```

#### Scenario: Next line with --no-switch

- **GIVEN** the user has invoked `/fab-new "description" --no-switch`
- **WHEN** the brief is complete and output is displayed
- **THEN** the `Next:` line reads: `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`
- **AND** `{name}` is the generated change folder name

#### Scenario: Next line without --no-switch

- **GIVEN** the user has invoked `/fab-new "description"` without `--no-switch`
- **WHEN** the brief is complete and output is displayed
- **THEN** the `Next:` line reads: `Next: /fab-continue or /fab-ff (fast-forward all planning)`

### Requirement: Skill File Only

This change SHALL modify only the skill file `fab/.kit/skills/fab-new.md`. No code, scripts, templates, or other skill files SHALL be modified. This is a prompt-only change per the Pure Prompt Play principle.

#### Scenario: No changes outside fab-new.md

- **GIVEN** the spec for this change
- **WHEN** implementation is complete
- **THEN** only `fab/.kit/skills/fab-new.md` has been modified
- **AND** no other files in `fab/.kit/` are changed

## Deprecated Requirements

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Scope to fab-new only (not fab-discuss) | fab-discuss already has its own "not activated" flow; brief explicitly scopes to fab-new |

1 assumption made (1 confident, 0 tentative).
