# Spec: Explicit Change Targeting for Workflow Commands

**Change**: 260213-w4k9-explicit-change-targeting
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/context-loading.md`, `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying `fab/current` when a change-name override is provided — the override is transient per invocation
- Adding change-name targeting to `/fab-switch` — it already has its own name resolution
- Adding change-name targeting to `/fab-new`, `/fab-init`, `/fab-hydrate`, or `/fab-hydrate-design` — these skills don't operate on existing changes via preflight

## Preflight: Change Name Override

### Requirement: Preflight Script Accepts Optional Override Argument

`fab/.kit/scripts/fab-preflight.sh` SHALL accept an optional first positional argument (`$1`) as a change name override. When provided, the script SHALL resolve the change using `$1` instead of reading `fab/current`. When `$1` is not provided, the script SHALL fall back to the existing behavior of reading `fab/current`.

The override MUST NOT modify `fab/current` — it is transient for this invocation only.

#### Scenario: Override argument provided with exact match
- **GIVEN** `fab/changes/260213-r3m7-add-conventions-section/` exists with a valid `.status.yaml`
- **WHEN** `fab-preflight.sh 260213-r3m7-add-conventions-section` is executed
- **THEN** the script outputs YAML with `name: 260213-r3m7-add-conventions-section`
- **AND** `fab/current` is not read or modified

#### Scenario: Override argument provided with partial slug match
- **GIVEN** `fab/changes/260213-r3m7-add-conventions-section/` is the only change folder matching "conventions"
- **WHEN** `fab-preflight.sh conventions` is executed
- **THEN** the script resolves to `260213-r3m7-add-conventions-section` and outputs its YAML

#### Scenario: Override argument provided with 4-char ID shorthand
- **GIVEN** `fab/changes/260213-r3m7-add-conventions-section/` exists and is the only folder containing "r3m7"
- **WHEN** `fab-preflight.sh r3m7` is executed
- **THEN** the script resolves to `260213-r3m7-add-conventions-section` and outputs its YAML

#### Scenario: Override argument with ambiguous match
- **GIVEN** multiple change folders match the substring "add" (e.g., `260213-r3m7-add-conventions-section` and `260213-3tyk-add-auth`)
- **WHEN** `fab-preflight.sh add` is executed
- **THEN** the script exits non-zero
- **AND** stderr lists the matching folders: `Multiple changes match "add": {list}. Provide a more specific name.`

#### Scenario: Override argument with no match
- **GIVEN** no change folder contains the substring "xyz"
- **WHEN** `fab-preflight.sh xyz` is executed
- **THEN** the script exits non-zero
- **AND** stderr outputs: `No change matches "xyz".`

#### Scenario: No override argument (backward compatibility)
- **GIVEN** `fab/current` contains `260213-w4k9-explicit-change-targeting`
- **WHEN** `fab-preflight.sh` is executed with no arguments
- **THEN** the script reads `fab/current` and resolves the change as before
- **AND** behavior is identical to the pre-change implementation

### Requirement: Matching Behavior Consistent with fab-switch

The preflight script's override matching SHALL use the same rules as `/fab-switch`:
- Case-insensitive substring matching against folder names in `fab/changes/` (excluding `archive/`)
- Exact match takes priority over partial match
- Single partial match resolves directly
- Multiple partial matches produce an error (not interactive — the script cannot prompt)
- No match produces an error

The 4-character random ID segment (the `XXXX` in `YYMMDD-XXXX-slug`) SHALL be a supported shorthand. Since these IDs are generated to be unique across active changes, providing just the 4-char ID (e.g., `r3m7`) SHOULD resolve to exactly one change via the standard substring matching.

## Skills: Change Name Argument

### Requirement: Workflow Skills Accept Optional Change Name

The following skills SHALL accept an optional `[change-name]` argument: `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-clarify`, `/fab-status`. When provided, the argument SHALL be passed to the preflight script (or status script for `/fab-status`) as `$1`.

#### Scenario: /fab-continue with change-name override
- **GIVEN** `fab/current` points to change A
- **WHEN** the user runs `/fab-continue change-B-name`
- **THEN** the skill passes `change-B-name` to `fab-preflight.sh` as `$1`
- **AND** the skill operates on change B without modifying `fab/current`
- **AND** change A remains the active change in `fab/current`

#### Scenario: /fab-continue with 4-char ID shorthand
- **GIVEN** `fab/current` points to change A
- **AND** change `260213-r3m7-add-conventions-section` exists
- **WHEN** the user runs `/fab-continue r3m7`
- **THEN** the skill passes `r3m7` to `fab-preflight.sh` as `$1`
- **AND** the skill operates on `260213-r3m7-add-conventions-section`

#### Scenario: /fab-continue with both change-name and stage reset
- **GIVEN** change `260213-r3m7-add-conventions-section` exists at stage `apply`
- **WHEN** the user runs `/fab-continue conventions spec`
- **THEN** the skill identifies `conventions` as a change-name (not a stage name) because it doesn't match any of the 6 stage names
- **AND** the skill identifies `spec` as a stage reset target
- **AND** the skill operates on the matched change, resetting to spec stage
<!-- assumed: argument disambiguation — stage names (brief, spec, tasks, apply, review, archive) are checked first since they're a fixed set of 6 known values; any non-stage argument is treated as a change-name override -->

#### Scenario: /fab-ff with change-name override
- **GIVEN** `fab/current` is empty or points to a different change
- **WHEN** the user runs `/fab-ff conventions`
- **THEN** the skill passes the argument to preflight and runs the full pipeline on the matched change

#### Scenario: /fab-clarify with change-name override
- **GIVEN** change `260213-r3m7-add-conventions-section` is at stage `spec`
- **WHEN** the user runs `/fab-clarify conventions`
- **THEN** the skill clarifies the spec artifact of the matched change

#### Scenario: /fab-status with change-name override
- **GIVEN** `fab/current` points to change A
- **WHEN** the user runs `/fab-status conventions`
- **THEN** the skill displays status for `260213-r3m7-add-conventions-section` without modifying `fab/current`

### Requirement: Argument Disambiguation for /fab-continue

`/fab-continue` SHALL disambiguate its arguments as follows: arguments matching one of the 6 stage names (`brief`, `spec`, `tasks`, `apply`, `review`, `archive`) are treated as stage reset targets. All other arguments are treated as change-name overrides. Both MAY be provided in the same invocation in any order.

#### Scenario: Ambiguous-looking argument that is a stage name
- **GIVEN** a change folder named `260213-xxxx-apply-fixes` exists
- **WHEN** the user runs `/fab-continue apply`
- **THEN** `apply` is treated as a stage reset target (stage names take priority)
- **AND** the change is resolved from `fab/current` as normal

#### Scenario: Two arguments — change name and stage
- **GIVEN** change `260213-r3m7-add-conventions-section` exists
- **WHEN** the user runs `/fab-continue conventions spec`
- **THEN** `conventions` is treated as a change-name (doesn't match any stage name)
- **AND** `spec` is treated as a stage reset target
- **AND** the spec stage is reset on the targeted change

### Requirement: Parallel Safety

When a change-name override is used, the invocation MUST NOT write to `fab/current` or any shared pointer file. This ensures multiple terminal sessions or Claude Code tabs can operate on different changes concurrently without racing on the pointer file.

#### Scenario: Two concurrent invocations on different changes
- **GIVEN** `fab/current` points to change A
- **AND** tab 1 runs `/fab-continue change-B`
- **AND** tab 2 runs `/fab-continue change-C`
- **WHEN** both invocations complete
- **THEN** `fab/current` still points to change A
- **AND** changes B and C were operated on independently

## Context Documentation: Preflight Override

### Requirement: Update _context.md Preflight Invocation

`fab/.kit/skills/_context.md` SHALL document the optional override argument in the "Change Context" section. The preflight invocation pattern SHALL be updated to show: `fab/.kit/scripts/fab-preflight.sh [change-name]`.

#### Scenario: Agent reads updated context
- **GIVEN** an agent loads `_context.md` before executing a skill
- **WHEN** the agent reads the "Change Context" section
- **THEN** it sees that preflight accepts an optional change-name argument
- **AND** understands that the argument overrides `fab/current` resolution transiently

### Requirement: Update Centralized Docs References

The affected centralized docs (`fab-workflow/context-loading`, `fab-workflow/planning-skills`, `fab-workflow/execution-skills`) SHALL be updated during archive to reflect the new override capability.

#### Scenario: Context-loading doc reflects override
- **GIVEN** the archive stage runs for this change
- **WHEN** `fab/docs/fab-workflow/context-loading.md` is hydrated
- **THEN** the "Preflight Script for Change Context" section documents the optional `$1` override argument

## Design Decisions

1. **Centralized matching in preflight script**: Matching logic lives in `fab-preflight.sh`, not in individual skills.
   - *Why*: All affected skills already go through preflight. Centralizing avoids duplicating matching logic in 6+ skill files and ensures consistent behavior.
   - *Rejected*: Per-skill matching — duplicates logic, risks inconsistency.

2. **Non-interactive matching errors**: When the override argument produces an ambiguous or no match, the preflight script exits non-zero with a descriptive error rather than prompting interactively.
   - *Why*: The preflight script is a Bash utility invoked by skills. Interactive prompts in the script would conflict with the skill's own interaction model. The skill surfaces the error to the user, who can provide a more specific name.
   - *Rejected*: Interactive prompts in preflight — Bash scripts shouldn't own user interaction in the Fab architecture.

3. **Stage names take priority in /fab-continue disambiguation**: When an argument matches both a stage name and could be a change-name substring, it is treated as a stage name.
   - *Why*: Stage names are a fixed set of 6 known values. Change names follow a `YYMMDD-XXXX-slug` format that is extremely unlikely to collide. Stage reset is the existing behavior and should not break.
   - *Rejected*: Change-name priority — would break existing `/fab-continue spec` behavior.

## Deprecated Requirements

None — this change is purely additive. Existing behavior (no argument → read `fab/current`) is fully preserved.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Stage names take priority over change-name matching in `/fab-continue` | Stage names are a fixed set of 6; change names follow YYMMDD-XXXX-slug format — collision is near-impossible. Preserves existing reset behavior |
| 2 | Confident | Preflight errors on ambiguous match (non-interactive) | Preflight is a Bash utility; interactive prompting belongs in skills. Consistent with Fab architecture where scripts validate and skills interact |
| 3 | Confident | Both change-name and stage arguments can coexist in `/fab-continue` | Natural extension — users may want to reset a non-active change's stage. Arguments are trivially disambiguatable by format |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.
