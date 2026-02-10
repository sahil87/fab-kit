# Spec: Fix collision handling in fab-new to regenerate 4-char component

**Change**: 260210-3uej-fix-collision-handling-regenerate
**Created**: 2026-02-10
**Affected docs**: `fab/docs/fab-workflow/change-lifecycle.md`

## Fab Workflow: Change Name Collision Handling

### Requirement: Collision Retry SHALL Regenerate Random Component

When `/fab-new` creates a change folder and detects that a folder with the generated name already exists, it SHALL regenerate the entire 4-character random component (`{XXXX}`) with a new random value and retry folder creation.

The system SHALL NOT append characters to the existing random component — doing so would produce a 5-character component that violates the `{YYMMDD}-{XXXX}-{slug}` format.

#### Scenario: Collision During Folder Creation

- **GIVEN** `/fab-new` has generated the name `260210-a7k2-fix-auth` for a new change
- **AND** a folder named `fab/changes/260210-a7k2-fix-auth/` already exists
- **WHEN** the system attempts to create the folder and detects the collision
- **THEN** the system SHALL generate a new 4-character random component (e.g., `b3m9`)
- **AND** retry folder creation with the new name `260210-b3m9-fix-auth`
- **AND** the final folder name SHALL match the format `{YYMMDD}-{XXXX}-{slug}` where `{XXXX}` is exactly 4 characters

#### Scenario: Format Invariant Preserved

- **GIVEN** `/fab-new` encounters multiple consecutive collisions (extremely unlikely)
- **WHEN** the system regenerates the random component on each collision
- **THEN** every retry attempt SHALL produce a name with exactly 4 random characters
- **AND** no generated name SHALL have a 5-character or longer random component

### Requirement: Documentation SHALL Reflect Regeneration Behavior

The `/fab-new` skill documentation in `fab/.kit/skills/fab-new.md` Step 2 (Create Change Directory) SHALL describe the collision handling as "regenerate the 4-character random component (`{XXXX}`) and retry."

#### Scenario: Skill Documentation Accuracy

- **GIVEN** a developer or agent reads `fab/.kit/skills/fab-new.md` Step 2
- **WHEN** they reach the collision handling instruction (item 3)
- **THEN** the instruction SHALL state "regenerate the 4-character random component (`{XXXX}`) and retry"
- **AND** the instruction SHALL NOT state "append an additional random character"

## Deprecated Requirements

### Collision Retry MAY Append Character

**Reason**: The "append" behavior produces folder names with 5-character random components (e.g., `260210-a7k2x-fix-auth`), violating the `{YYMMDD}-{XXXX}-{slug}` format defined in `fab/config.yaml` and `fab/docs/fab-workflow/change-lifecycle.md`.

**Migration**: Use regeneration instead — generate a new 4-character random value on each collision, preserving the format invariant.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Certain | Update only `fab/.kit/skills/fab-new.md` | The `.agents/skills/fab-new/SKILL.md` is a symlink to this file — one update covers both |
| 2 | Certain | `change-lifecycle.md` requires no update | It mentions collision avoidance conceptually but does not document the collision handling mechanism |

2 assumptions made (2 certain, 0 tentative).
