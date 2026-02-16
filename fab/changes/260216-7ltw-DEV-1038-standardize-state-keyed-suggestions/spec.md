# Spec: Standardize State-Keyed Next-Step Suggestions

**Change**: 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions
**Created**: 2026-02-16
**Affected memory**:
- `docs/memory/fab-workflow/planning-skills.md`
- `docs/memory/fab-workflow/execution-skills.md`
- `docs/memory/fab-workflow/change-lifecycle.md`
- `docs/memory/fab-workflow/clarify.md`

## Non-Goals

- Changing the 6-stage pipeline or any stage semantics
- Modifying SRAD scoring or confidence gating behavior
- Changing any skill's stage advancement logic (only suggestion output changes)
- Adding new commands or skills

## Suggestion Convention: State Table

### Requirement: Canonical State Table

`_context.md` SHALL define a single state table as the canonical source of truth for all next-step suggestions in the Fab workflow. The table SHALL be keyed by the state reached (not by the skill that ran). Each row SHALL include: the state name, available commands, and the default command.

The state table SHALL contain:

| State | Available commands | Default |
|---|---|---|
| (none) | /fab-setup | /fab-setup |
| initialized | /fab-new, /docs-hydrate-memory | /fab-new |
| intake | /fab-continue, /fab-fff, /fab-clarify | /fab-continue |
| spec | /fab-continue, /fab-ff, /fab-clarify | /fab-continue |
| tasks | /fab-continue, /fab-ff, /fab-clarify | /fab-continue |
| apply | /fab-continue | /fab-continue |
| review (pass) | /fab-continue | /fab-continue |
| review (fail) | (rework menu) | — |
| hydrate | /fab-archive | /fab-archive |

Key constraints encoded in this table:
- `/fab-fff` is available only from `intake` (full pipeline from the start)
- `/fab-ff` is available from `spec` and `tasks` (needs spec to exist)
- `/fab-clarify` is available from `intake`, `spec`, and `tasks` (all planning stages)
- Each state has exactly one default command (except `review (fail)` which uses the rework menu)

State derivation rules:
- **(none)**: `fab/config.yaml` does not exist
- **initialized**: `fab/config.yaml` exists AND no active change (`fab/current` absent or empty)
- **intake** through **apply**: Derived from the active change's `.status.yaml` progress map (the stage with `active` state, per the existing three-tier fallback)
- **review (pass)**: `progress.review == done`
- **review (fail)**: `progress.review == failed`
- **hydrate**: `progress.hydrate == done`

This table SHALL replace the current skill-keyed "After skill | Stage reached | Next line" lookup table in `_context.md`.

#### Scenario: Same state reached by different skills
- **GIVEN** a change where spec generation just completed
- **WHEN** the state reached is `tasks` (tasks stage now active)
- **THEN** the `Next:` line is identical regardless of whether `/fab-continue`, `/fab-ff`, or `/fab-fff` performed the generation
- **AND** the output lists `/fab-continue, /fab-ff, or /fab-clarify`

#### Scenario: State derived from progress map
- **GIVEN** a change with `progress.spec == active` in `.status.yaml`
- **WHEN** any skill determines the current state for suggestion lookup
- **THEN** the state is `spec`
- **AND** the available commands are `/fab-continue, /fab-ff, /fab-clarify`

#### Scenario: No active change after archive
- **GIVEN** `/fab-archive` has completed (change moved, `fab/current` cleared)
- **WHEN** the skill determines the state for suggestions
- **THEN** the state is `initialized` (config exists, no active change)
- **AND** the `Next:` line is `/fab-new <description> or /docs-hydrate-memory <sources>`

### Requirement: Skill Suggestion Lookup Convention

Every skill that outputs a `Next:` line SHALL derive it from the state table:

1. Determine the state reached after the skill's action
2. Look up that state in the canonical table
3. Output `Next:` with the default command listed first, followed by other available commands

Skills MUST NOT hardcode suggestion text. All `Next:` lines SHALL be derivable from the state table.

**Activation preamble**: When a skill creates or restores a change without activating it (no write to `fab/current`), the `Next:` line SHALL include a switch instruction followed by the created/restored change's state-derived commands:

```
Next: /fab-switch {name} to make it active, then {default}, {other commands}
```

This applies to:
- `/fab-new` (always — change is never auto-activated after this change)
- `/fab-archive restore` without `--switch`

When `/fab-archive restore --switch` is used, the change is activated and the `Next:` line is derived directly from the restored change's state.

#### Scenario: fab-new creates a change
- **GIVEN** `/fab-new` creates a change named `260216-a1b2-add-feature` at intake stage
- **AND** the change is NOT auto-activated
- **WHEN** the skill outputs its `Next:` line
- **THEN** the output includes the activation preamble: `Next: /fab-switch 260216-a1b2-add-feature to make it active, then /fab-continue or /fab-fff or /fab-clarify`

#### Scenario: fab-archive restore with --switch
- **GIVEN** `/fab-archive restore {name} --switch` restores and activates a change at spec stage
- **WHEN** the skill outputs its `Next:` line
- **THEN** the state is `spec` (derived from the restored change)
- **AND** the output is `Next: /fab-continue, /fab-ff, or /fab-clarify`

#### Scenario: fab-archive restore without --switch
- **GIVEN** `/fab-archive restore {name}` restores a change at spec stage without activating
- **WHEN** the skill outputs its `Next:` line
- **THEN** the output includes the activation preamble: `Next: /fab-switch {name} to make it active, then /fab-continue, /fab-ff, or /fab-clarify`

### Requirement: Default Command Listed First

In the `Next:` output, the default command (as specified in the state table's Default column) SHALL appear first. Remaining commands follow in any stable order.

#### Scenario: Default first at spec state
- **GIVEN** the state reached is `spec`
- **WHEN** a skill outputs the `Next:` line
- **THEN** `/fab-continue` appears before `/fab-ff` and `/fab-clarify`

## Suggestion Convention: Per-Skill Updates

### Requirement: fab-switch Removes Private Table

`/fab-switch` SHALL remove its private stage→suggestion table and derive suggestions from the canonical state table. After switching to a change, `/fab-switch` SHALL read the change's state from `.status.yaml` and look up the corresponding `Next:` line.

#### Scenario: Switch to change at intake
- **GIVEN** a change with `progress.intake == active`
- **WHEN** the user runs `/fab-switch {name}`
- **THEN** the `Next:` line is: `/fab-continue, /fab-fff, or /fab-clarify` (intake state)

#### Scenario: Switch to change at apply
- **GIVEN** a change with `progress.apply == active`
- **WHEN** the user runs `/fab-switch {name}`
- **THEN** the `Next:` line is: `/fab-continue` (apply state)

### Requirement: fab-status Uses State Table

`/fab-status` SHALL derive its "suggested next command" from the canonical state table, eliminating any custom suggestion logic.

#### Scenario: Status shows ff at spec stage
- **GIVEN** a change with `progress.spec == active`
- **WHEN** the user runs `/fab-status`
- **THEN** the suggested next command includes `/fab-ff` and `/fab-clarify` (spec state)

#### Scenario: Status shows archive at hydrate done
- **GIVEN** a change with `progress.hydrate == done`
- **WHEN** the user runs `/fab-status`
- **THEN** the suggested next command is `/fab-archive` (hydrate state)

### Requirement: fab-clarify Suggestions Are State-Aware

`/fab-clarify` SHALL derive its `Next:` line from the current state (unchanged, since clarify is non-advancing). The current behavior of always suggesting `/fab-fff` from any planning stage SHALL be removed.

Concrete mappings:
- From intake: `Next: /fab-continue, /fab-fff, or /fab-clarify`
- From spec: `Next: /fab-continue, /fab-ff, or /fab-clarify`
- From tasks: `Next: /fab-continue, /fab-ff, or /fab-clarify`

#### Scenario: Clarify at intake suggests fff
- **GIVEN** a change at intake stage
- **WHEN** `/fab-clarify` completes a suggest-mode session
- **THEN** the `Next:` line includes `/fab-fff` (NOT `/fab-ff`)

#### Scenario: Clarify at spec suggests ff
- **GIVEN** a change at spec stage
- **WHEN** `/fab-clarify` completes a suggest-mode session
- **THEN** the `Next:` line includes `/fab-ff` (NOT `/fab-fff`)

### Requirement: fab-clarify Stage Guard Includes Intake

`/fab-clarify` SHALL extend its stage guard to accept `intake` as a valid stage, in addition to the existing `spec` and `tasks`. This aligns with the state table, which lists `/fab-clarify` as available from all three planning stages.

At the intake stage, the clarify taxonomy scan SHALL include intake artifact refinement: scope boundaries, affected areas, blocking questions, impact completeness, affected memory coverage, Origin section completeness. This taxonomy already exists in the clarify skill's spec-stage scan (which scans both `intake.md` and `spec.md`); at the intake stage, only the intake subset applies.

#### Scenario: Clarify at intake stage
- **GIVEN** a change with `progress.intake == active`
- **WHEN** the user runs `/fab-clarify`
- **THEN** the skill operates normally, scanning `intake.md` for gaps and ambiguities
- **AND** presents structured questions

#### Scenario: Clarify at apply stage still blocked
- **GIVEN** a change with `progress.apply == active`
- **WHEN** the user runs `/fab-clarify`
- **THEN** the skill aborts with a suggestion to use `/fab-continue`

### Requirement: All Hardcoded Next: Lines Replaced

The following skills SHALL replace their hardcoded `Next:` lines with state-table-derived suggestions:

| Skill file | Current behavior | New behavior |
|---|---|---|
| `fab-ff.md` | Hardcoded `Next: /fab-archive` | Derive from state reached |
| `fab-fff.md` | Hardcoded `Next: /fab-archive` | Derive from state reached |
| `fab-archive.md` (archive) | Hardcoded `Next: /fab-new <description>` | Derive from `initialized` state |
| `fab-archive.md` (restore) | Hardcoded `Next: /fab-switch {name}` | Derive from restored change's state (with/without activation preamble) |
| `fab-setup.md` | Hardcoded `Next: /fab-new ... or /docs-hydrate-memory ...` | Derive from `initialized` state |
| `fab-clarify.md` | Hardcoded `Next: /fab-clarify or /fab-continue or /fab-ff` | Derive from current state |
| `fab-continue.md` | Hardcoded per-stage `Next:` lines | Derive from state reached |
| `fab-switch.md` | Private table + hardcoded `Next` | Derive from switched change's state |
| `docs-hydrate-memory.md` | Hardcoded `Next: /fab-new ...` | Derive from `initialized` state |

#### Scenario: fab-ff completes successfully
- **GIVEN** `/fab-ff` completes the pipeline through hydrate
- **WHEN** the skill outputs its `Next:` line
- **THEN** the state is `hydrate` (done) and the output is `Next: /fab-archive`

#### Scenario: fab-continue review pass
- **GIVEN** `/fab-continue` review passes and transitions to `hydrate: active`
- **WHEN** the skill outputs its `Next:` line
- **THEN** the state is `review (pass)` and the output is `Next: /fab-continue`

## fab-new: Remove --switch Flag

### Requirement: Remove --switch Flag and Auto-Switch Logic

`/fab-new` SHALL NOT accept a `--switch` flag. The following SHALL be removed:

- The `--switch` argument from the Arguments section
- The conditional activation step (call to `/fab-switch`)
- The conditional output path (`{if switched: "Branch: {name} (created)\n"}`)
- Natural language switching detection (phrases like "and switch to it", "make it active")

After this change, `/fab-new` SHALL always create the change without activating it. The output SHALL never include a `Branch:` line.

#### Scenario: fab-new always creates without activating
- **GIVEN** a user runs `/fab-new "add new feature"`
- **WHEN** the skill completes
- **THEN** the change folder is created
- **AND** `fab/current` is NOT modified
- **AND** no branch is created or checked out

#### Scenario: Natural language switching ignored
- **GIVEN** a user runs `/fab-new "add new feature and switch to it"`
- **WHEN** the skill processes the description
- **THEN** the switching intent phrase is treated as part of the description, not as a switching directive
- **AND** the change is created but NOT activated

### Requirement: fab-new Single Suggestion Path

After removing `--switch`, `/fab-new` SHALL always output a single `Next:` line using the activation preamble convention:

```
Next: /fab-switch {name} to make it active, then /fab-continue or /fab-fff or /fab-clarify
```

Where the commands after "then" are derived from the `intake` state in the state table. The two-path conditional suggestion SHALL be removed.

#### Scenario: fab-new always suggests switch first
- **GIVEN** `/fab-new` creates a change named `260216-a1b2-add-feature`
- **WHEN** the skill outputs its `Next:` line
- **THEN** the output is `Next: /fab-switch 260216-a1b2-add-feature to make it active, then /fab-continue or /fab-fff or /fab-clarify`

## Deprecated Requirements

### Skill-Keyed Lookup Table
**Reason**: Replaced by the state-keyed table. The skill-keyed table had redundant rows for skills reaching the same state, creating drift risk.
**Migration**: Remove the "After skill | Stage reached | Next line" table from `_context.md`. All skills now look up the state-keyed table.

### fab-switch Private Stage→Suggestion Table
**Reason**: Replaced by the canonical state table lookup. The private table was a third copy of suggestion logic.
**Migration**: `/fab-switch` derives suggestions from the state table in `_context.md`.

### fab-new --switch Flag and Auto-Switch Logic
**Reason**: Eliminated the conditional suggestion branch and simplified `/fab-new` to always-create-without-activating.
**Migration**: Users who want to activate after creation explicitly run `/fab-switch {name}`.

## Design Decisions

1. **State-keyed over skill-keyed table**: The table is keyed by the state reached, not by the skill that ran.
   - *Why*: Same state should always produce the same suggestion, regardless of how you got there. Skill-keyed tables create N rows per state where N is the number of skills that can reach it — redundancy that drifts.
   - *Rejected*: Keeping skill-keyed table — divergence across redundant rows is how the duplication problem started.

2. **Activation preamble for non-active resources**: When a skill creates or restores a change without activating it, the `Next:` line includes a switch instruction before the state-derived commands.
   - *Why*: The user can't run state-dependent commands without first making the change active. The preamble makes this explicit while still deriving the available commands from the state table.
   - *Rejected*: Separate "created" state in the table — conflates project state with change state. Omitting switch instruction — user would try `/fab-continue` without an active change.

3. **Clarify stage guard extended to include intake**: `/fab-clarify` accepts `intake`, `spec`, and `tasks` — all planning stages.
   - *Why*: The state table shows `/fab-clarify` available from intake. Intake artifacts can have gaps worth resolving before spec generation. Not extending clarify would make the table misleading.
   - *Rejected*: Not extending clarify to intake — table would list an unavailable command.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | State table keyed by state, not skill | Confirmed from intake #1 — user explicitly confirmed during analysis discussion | S:95 R:85 A:90 D:95 |
| 2 | Certain | Remove --switch from fab-new | Confirmed from intake #2 — user explicitly proposed and analysis supports it | S:90 R:80 A:85 D:90 |
| 3 | Certain | /fab-fff only from intake, /fab-ff from spec and tasks | Confirmed from intake #3 — discussed and confirmed, eliminates confusing overlap | S:90 R:75 A:90 D:90 |
| 4 | Certain | /fab-clarify available from intake, spec, tasks | Upgraded from intake #4 (Confident→Certain) — the state table explicitly lists clarify at all three planning stages as defining intent of the change | S:85 R:85 A:85 D:85 |
| 5 | Confident | Default command listed first in Next: output | Confirmed from intake #5 — consistent with the table's Default column, though exact formatting (bold, order, etc.) not fully discussed | S:75 R:90 A:80 D:80 |
| 6 | Confident | fab-archive restore uses restored change's state | Upgraded from intake #6 (Tentative→Confident) — follows from the state-keyed principle: restored change has a real state, no need for a "restored" meta-state | S:70 R:85 A:80 D:80 |
| 7 | Confident | Activation preamble convention for non-active resources | New — required since fab-new never activates; consistent with fab-archive restore without --switch | S:75 R:85 A:75 D:75 |
| 8 | Confident | (none) and initialized states explicit in table | New — completeness; pre-init is rare but including it makes the table exhaustive | S:65 R:90 A:80 D:85 |
| 9 | Confident | /fab-clarify stage guard extended to include intake | New — required for state table consistency; the taxonomy for intake refinement already exists within clarify's spec-stage scan | S:75 R:80 A:75 D:80 |
| 10 | Confident | Tasks state includes /fab-clarify (correcting intake table) | New — intake text says clarify at "intake, spec, and tasks" but the table omits it from tasks; resolved in favor of text since clarify already works at tasks | S:80 R:85 A:85 D:85 |

10 assumptions (4 certain, 6 confident, 0 tentative, 0 unresolved).
