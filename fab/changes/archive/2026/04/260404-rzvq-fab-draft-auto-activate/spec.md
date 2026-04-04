# Spec: Fab Draft Auto Activate

**Change**: 260404-rzvq-fab-draft-auto-activate
**Created**: 2026-04-05
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/change-lifecycle.md`

## Non-Goals

- Changes to the Go CLI binary (`fab change switch`, `fab change new`) — the CLI already supports the needed commands; this change only updates skill files and docs
- Data migration — no `.status.yaml` schema changes
- Configuration toggles for auto-activation — the opt-out path is `/fab-draft`, not a flag

## `/fab-new` Skill: Auto-Activation

### Requirement: Auto-Activate After Intake

`/fab-new` SHALL activate the newly created change immediately after advancing intake to ready, by calling `fab change switch "{name}"`. The `.fab-status.yaml` symlink SHALL be created as a side effect of this call. The skill's description SHALL read "Start a new change — creates the intake and activates it."

#### Scenario: New change created and activated
- **GIVEN** a user runs `/fab-new <description>` in a project with `fab/project/config.yaml`
- **WHEN** the intake is advanced to `ready` (Step 9)
- **THEN** `/fab-new` calls `fab change switch "{name}"`
- **AND** the command output (switch confirmation) is displayed to the user
- **AND** the `Next:` line reads `/fab-continue, /fab-fff, /fab-ff, or /fab-clarify` (no activation preamble)

#### Scenario: Activated change is immediately usable
- **GIVEN** `/fab-new` has completed successfully
- **WHEN** the user runs `/fab-continue` without a preceding `/fab-switch`
- **THEN** `fab resolve` resolves the newly created change via the `.fab-status.yaml` symlink
- **AND** spec generation proceeds normally

### Requirement: Remove Activation Preamble from Next Line

The `Next:` line in `/fab-new` output SHALL NOT include the activation preamble (`/fab-switch {name} to make it active`). The activation preamble applies only to `/fab-draft` (always) and `/fab-archive restore` (without `--switch`).

#### Scenario: Next line format
- **GIVEN** `/fab-new` completes
- **WHEN** the skill outputs its result
- **THEN** the final `Next:` line is `/fab-continue, /fab-fff, /fab-ff, or /fab-clarify`
- **AND** there is no `/fab-switch` instruction in the output

## `/fab-draft` Skill: Create Without Activate

### Requirement: New Skill File

A new skill file SHALL exist at `src/kit/skills/fab-draft.md`. Its behavior SHALL be identical to the current `/fab-new` Steps 1–9 (create folder, initialize `.status.yaml`, generate intake, advance to ready), but SHALL NOT call `fab change switch` and SHALL NOT create the `.fab-status.yaml` symlink.

#### Scenario: fab-draft creates without activating
- **GIVEN** a user runs `/fab-draft <description>`
- **WHEN** the intake is generated and advanced to ready
- **THEN** no `.fab-status.yaml` symlink is created or modified
- **AND** the `Next:` line includes the activation preamble: `/fab-switch {name} to make it active, then /fab-continue or /fab-fff or /fab-clarify`

#### Scenario: Power user queuing multiple changes
- **GIVEN** a user has an active change `A` and runs `/fab-draft <description B>`
- **WHEN** `/fab-draft` completes
- **THEN** `.fab-status.yaml` still points to change `A`
- **AND** change `B` exists in `fab/changes/` with intake at `ready`

### Requirement: Skill Description and Metadata

The `/fab-draft` skill description SHALL read "Create a change intake without activating it." The skill frontmatter SHALL include `name: fab-draft`. The skill SHALL be added to the `skillToGroupMap` in `fabhelp.go` under the `"Start & Navigate"` group.

#### Scenario: fab help listing
- **GIVEN** a user runs `fab help`
- **WHEN** the skill list renders
- **THEN** `fab-draft` appears under the "Start & Navigate" group alongside `fab-new`

## `/fab-switch` Skill: Updated Error Messages

### Requirement: Reference fab-draft in Empty-State Messages

When no active changes exist, `/fab-switch` error messages SHALL reference both `/fab-new` and `/fab-draft` so users know both creation paths exist.

#### Scenario: No-argument flow with no changes
- **GIVEN** `fab/changes/` is empty (no non-archived changes)
- **WHEN** a user runs `/fab-switch` with no arguments
- **THEN** the output reads: `No active changes found. Run /fab-new <description> to start one, or /fab-draft <description> to create without activating.`

#### Scenario: Error table entry
- **GIVEN** the error handling table in `/fab-switch`
- **WHEN** the "No changes exist" condition triggers
- **THEN** the action reads: `"No active changes found. Run /fab-new or /fab-draft."`

## `/fab-proceed` Skill: Simplified Dispatch Table

### Requirement: Remove fab-switch from Conversation Context Path

Since `/fab-new` now auto-activates, the dispatch table row for "Conversation context (no intake)" SHALL change from `/fab-new → /fab-switch → /git-branch → /fab-fff` to `/fab-new → /git-branch → /fab-fff`. The `/fab-switch` subagent step after `/fab-new` is no longer needed.

#### Scenario: New change from conversation context
- **GIVEN** no active change and no intake exists, but substantive conversation context is present
- **WHEN** `/fab-proceed` runs
- **THEN** it dispatches `/fab-new` subagent, then `/git-branch` subagent, then invokes `/fab-fff`
- **AND** no `/fab-switch` subagent is dispatched between `/fab-new` and `/git-branch`

### Requirement: Retain fab-switch for Unactivated Intakes

The "Unactivated intake (no active change)" path SHALL retain `/fab-switch` in its dispatch chain. These intakes were created by `/fab-draft` and require explicit activation.

#### Scenario: Draft intake activation
- **GIVEN** an intake exists in `fab/changes/` with no `.fab-status.yaml` symlink
- **WHEN** `/fab-proceed` runs
- **THEN** it dispatches `/fab-switch` → `/git-branch` → `/fab-fff`

### Requirement: Updated Error Message

The empty-context error message SHALL reference both `/fab-new` and `/fab-draft`: `Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first.`

## `_preamble.md`: Updated Activation Preamble

### Requirement: Activation Preamble Applies to fab-draft, Not fab-new

The Activation Preamble section in `_preamble.md` SHALL state that the preamble applies to `/fab-draft` (always) and `/fab-archive restore` (without `--switch`). The reference to `/fab-new` as a case that requires the activation preamble SHALL be removed.

#### Scenario: Preamble wording
- **GIVEN** the Activation Preamble section in `_preamble.md`
- **WHEN** a skill generates output for a change that was created without activation
- **THEN** the preamble guidance applies to `/fab-draft` scenarios, not `/fab-new` scenarios

## `README.md`: Updated Diagram

### Requirement: Mermaid Diagram Shows fab-draft and fab-new Distinctly

The command coverage diagram in `README.md` SHALL show `/fab-draft` and `/fab-new` as separate columns. `/fab-draft` SHALL be styled in purple (change lifecycle color) alongside `/fab-switch`. `/fab-new` SHALL be styled in green (automation color) to signal that it handles more steps automatically (intake + activation).

#### Scenario: Diagram column count and structure
- **GIVEN** the mermaid block-beta diagram in `README.md`
- **WHEN** rendered
- **THEN** the diagram has 12 columns (up from 11)
- **AND** `/fab-draft` appears as `header1` (purple, change lifecycle)
- **AND** `/fab-new` appears as `headerN` (green, automation)
- **AND** `/fab-new`'s row shows both `fnew_in["intake"]` and `fnew_act["change active"]`

## `docs/specs/skills.md`: Updated Skill Catalog

### Requirement: Add fab-draft Section and Update fab-new Section

The `docs/specs/skills.md` skill catalog SHALL include:
- A new `## /fab-draft <description>` section documenting the create-only behavior
- The `## /fab-new <description>` section updated to remove the `--switch` flag and reflect auto-activation
- The Next Steps table updated to include `/fab-draft` with its activation-preamble next line

#### Scenario: Spec catalog completeness
- **GIVEN** `docs/specs/skills.md`
- **WHEN** a reader looks up `/fab-new`
- **THEN** the section shows: no `--switch` argument, auto-activation behavior in step 8, and a cross-reference to `/fab-draft`
- **AND** a separate `/fab-draft` section exists with its own purpose, examples, and behavior description

## `fab-operator.md` and `_cli-fab.md`: Cross-References

### Requirement: Updated Command Vocabulary

`fab-operator.md` SHALL describe the setup commands as: `/fab-new` (create + activate change), `/fab-draft` (create without activating), `/fab-switch` (activate existing change), `/git-branch` (align branch). The pipeline description SHALL reflect that `/fab-proceed` runs `/fab-new → /git-branch` (not `/fab-new → /fab-switch → /git-branch`).

`_cli-fab.md` error message table SHALL update the "No active changes found" entry to say: `Run /fab-new or /fab-draft first`.

#### Scenario: Operator routing awareness
- **GIVEN** `fab-operator.md` vocabulary section
- **WHEN** an operator agent reads the command vocabulary
- **THEN** it knows `/fab-new` auto-activates and `/fab-draft` creates without activating
- **AND** the `/fab-proceed` pipeline description omits the `/fab-switch` step after `/fab-new`

## `src/go/fab/cmd/fab/fabhelp.go`: Skill Group Registration

### Requirement: fab-draft in Start & Navigate Group

`src/go/fab/cmd/fab/fabhelp.go` SHALL include `"fab-draft": "Start & Navigate"` in `skillToGroupMap`. The corresponding test in `fabhelp_test.go` SHALL include `"fab-draft"` in the `expectedMapped` slice.

#### Scenario: fab help renders fab-draft
- **GIVEN** `skillToGroupMap` in `fabhelp.go`
- **WHEN** `fab help` renders skill groups
- **THEN** `fab-draft` appears under "Start & Navigate" alongside `fab-new` and `fab-switch`

## Deprecated Requirements

### `/fab-new --switch` Flag

**Reason**: The `--switch` flag existed in `docs/specs/skills.md` but was never implemented in the actual skill source. With `/fab-new` now always auto-activating, the flag concept is superseded entirely.

**Migration**: Use `/fab-new` (always activates) or `/fab-draft` (never activates). No migration needed — the flag was never shipped.

## Design Decisions

1. **Separate skill files, not a flag**: `/fab-draft` is a distinct skill file (`fab-draft.md`), not a `--draft` flag on `/fab-new`.
   - *Why*: User explicitly chose separate files for single-purpose clarity. Each skill has one job; flags create two behaviors per command.
   - *Rejected*: `--draft` flag on `/fab-new` — rejected because it makes the default less obvious and burdens newcomers with flag awareness.

2. **Always-on activation for `/fab-new`**: `/fab-new` always auto-activates with no opt-out flag.
   - *Why*: The opt-out path is `/fab-draft`. Adding both `--draft` and default activation creates a third configuration that doesn't exist.
   - *Rejected*: `--no-switch` opt-out flag — rejected as unnecessary indirection.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `/fab-draft` is a separate skill file, not a `--draft` flag | Confirmed from intake #1 — user explicitly chose separate files | S:95 R:85 A:95 D:95 |
| 2 | Certain | `/fab-new` always auto-activates, no opt-out flag | Confirmed from intake #2 — opt-out path is `/fab-draft` | S:95 R:80 A:90 D:95 |
| 3 | Certain | No data migration needed | Confirmed from intake #3 — skill behavior change only, no schema changes | S:90 R:95 A:95 D:95 |
| 4 | Certain | No CLI binary changes needed | `fab change switch` already exists; only skill files and docs change | S:90 R:95 A:95 D:95 |
| 5 | Confident | `/fab-draft` copies Steps 1-9 from `/fab-new` by duplication | Confirmed from intake #4 — constitution's "pure prompt play" principle means no include/import mechanism; duplication is the only option for two separate skill files | S:70 R:90 A:80 D:75 |
| 6 | Confident | `/fab-proceed` retains `/fab-switch` in "unactivated intake" path | Confirmed from intake #5 — unactivated intakes created by `/fab-draft` still need explicit activation | S:75 R:80 A:85 D:80 |
| 7 | Certain | `--switch` flag removed from `docs/specs/skills.md` | Confirmed from intake #6 — flag was never implemented in skill source, now superseded | S:90 R:90 A:90 D:95 |
| 8 | Confident | Operator routing unchanged for batch new | Confirmed from intake #7 — `/fab-batch new` spawns agents with `/fab-new`, auto-activation is correct behavior in that context | S:70 R:75 A:80 D:80 |
| 9 | Certain | `fab-draft` added to `skillToGroupMap` under "Start & Navigate" | Required for `fab help` to list the new skill; group mirrors `fab-new`'s placement | S:90 R:90 A:90 D:95 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
