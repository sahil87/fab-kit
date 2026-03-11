# Spec: Drop runtime is-idle from Operator

**Change**: 260311-ftrh-drop-runtime-idle-from-operator
**Created**: 2026-03-11
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying `fab runtime` CLI commands — they remain as-is for CWD-local use by hooks
- Changing `fab pane-map` behavior — it already works correctly
- Altering the operator's idle-checking policy — only the data source changes

## Operator Skill: Remove `fab runtime is-idle` References

### Requirement: Operator SHALL use pane-map Agent column for idle detection

The operator skill (`fab/.kit/skills/fab-operator1.md`) SHALL use the Agent column from `fab pane-map` output as its sole mechanism for determining agent idle state. All references to `fab runtime is-idle` SHALL be removed from the skill file.

#### Scenario: State re-derivation uses pane-map only

- **GIVEN** the operator skill's State Re-derivation section
- **WHEN** an agent reads the section to understand how to re-derive state before an action
- **THEN** the only listed mechanism is `fab pane-map`
- **AND** there is no mention of `fab runtime is-idle`

#### Scenario: Pre-send validation uses pane-map Agent column

- **GIVEN** the operator skill's Pre-Send Validation section
- **WHEN** an agent reads the idle-check step
- **THEN** the instruction is to read the Agent column from the pane map
- **AND** there is no alternative `fab runtime is-idle` path

#### Scenario: Broadcast filters idle agents via pane-map

- **GIVEN** UC1 (Broadcast command to all idle agents) in the operator skill
- **WHEN** an agent reads how to filter for idle agents
- **THEN** the filtering mechanism references the Agent column in the pane map
- **AND** there is no reference to `fab runtime` state

#### Scenario: Unstick uses pane-map for idle confirmation

- **GIVEN** UC6 (Unstick a stuck agent) in the operator skill
- **WHEN** an agent reads how to confirm the target agent is idle
- **THEN** the instruction uses the pane map Agent column
- **AND** there is no `fab runtime` reference

#### Scenario: Autopilot monitoring uses pane-map only

- **GIVEN** the Autopilot per-change loop in the operator skill
- **WHEN** an agent reads the monitoring step (step 4)
- **THEN** the polling mechanism is `fab pane-map` only
- **AND** there is no `fab runtime is-idle` alongside it

#### Scenario: Purpose statement reflects pane-map-only observation

- **GIVEN** the Purpose section of the operator skill
- **WHEN** an agent reads how the operator observes agents
- **THEN** the observation mechanism is `fab pane-map` only
- **AND** `fab runtime` is not listed as an observation primitive

### Requirement: Six specific edits SHALL be applied to the operator skill

The following edits SHALL be applied to `fab/.kit/skills/fab-operator1.md`:

1. **Purpose (line ~14)**: Replace "via `fab pane-map` and `fab runtime`" with "via `fab pane-map`"
2. **State Re-derivation (line ~80)**: Remove the `fab runtime is-idle` bullet; keep only `fab pane-map`
3. **UC1 Broadcast (line ~93)**: Change "via `fab runtime` state in the pane map" to "via the Agent column in the pane map"
4. **UC6 Unstick (line ~125)**: Replace "Confirm the target agent is idle via `fab/.kit/bin/fab runtime`" with "Confirm the target agent is idle via the Agent column in the pane map"
5. **Pre-Send Validation (line ~164)**: Replace "Run `fab/.kit/bin/fab runtime is-idle <change>` or read the Agent column from the pane map" with "Read the Agent column from the pane map"
6. **Autopilot per-change loop (line ~249)**: Replace "poll `fab pane-map` + `fab runtime is-idle`" with "poll `fab pane-map`"

#### Scenario: All six edits applied

- **GIVEN** the operator skill file after edits
- **WHEN** searching for the string `runtime is-idle` or `fab runtime`
- **THEN** zero matches are found
- **AND** all six locations reference pane-map or the Agent column instead

## Operator Spec: Remove `fab runtime is-idle` References

### Requirement: Operator spec SHALL mirror skill changes

The operator spec (`docs/specs/skills/SPEC-fab-operator1.md`) SHALL be updated to remove all `fab runtime is-idle` references, matching the skill changes.

#### Scenario: Spec primitives table has no runtime is-idle row

- **GIVEN** the Primitives table in the operator spec
- **WHEN** an agent reads the table
- **THEN** there is no row for `fab runtime is-idle`
- **AND** the `fab pane-map` row remains as the primary observation mechanism

#### Scenario: Spec relationship table has no runtime is-idle row

- **GIVEN** the Relationship to Existing Components table in the operator spec
- **WHEN** an agent reads the table
- **THEN** there is no row for `fab runtime is-idle`

### Requirement: Seven specific edits SHALL be applied to the operator spec

The following edits SHALL be applied to `docs/specs/skills/SPEC-fab-operator1.md`:

1. **Summary (line ~5)**: Remove `fab runtime` from the observation primitives list and replace "via `fab pane-map` and `fab runtime`" with "via `fab pane-map`"
2. **Primitives table (line ~18)**: Remove the `fab runtime is-idle` row entirely
3. **Per-change loop (line ~151)**: Remove `fab runtime is-idle` from monitoring step
4. **Pre-send validation (line ~223)**: Replace `runtime is-idle` with pane-map Agent column
5. **Always re-derive state (line ~243)**: Remove `fab runtime is-idle` from the list
6. **Agent busy detection (line ~314)**: Replace `fab runtime is-idle` with pane-map Agent column
7. **Relationship table (line ~329)**: Remove the `fab runtime is-idle` row

#### Scenario: All seven edits applied

- **GIVEN** the operator spec file after edits
- **WHEN** searching for the string `runtime is-idle` or `fab runtime`
- **THEN** zero matches are found
- **AND** all locations reference pane-map or the Agent column instead

## Deprecated Requirements

### `fab runtime is-idle` as operator observation primitive

**Reason**: `fab runtime is-idle` reads `.fab-runtime.yaml` from the CWD, which in the operator's case is its own worktree — not the target agent's worktree. The result is always wrong. `fab pane-map` already resolves each worktree's runtime file correctly via tmux pane discovery.

**Migration**: Replaced by the Agent column in `fab pane-map` output, which the operator already refreshes before every action.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use pane-map Agent column instead of runtime is-idle | Confirmed from intake #1 — user chose Option A, pane-map already has the data | S:95 R:90 A:95 D:95 |
| 2 | Certain | No changes to fab runtime CLI | Confirmed from intake #2 — YAGNI, hooks use CWD-local runtime correctly | S:90 R:95 A:90 D:90 |
| 3 | Certain | No behavioral change to operator policy | Confirmed from intake #3 — idle checking still happens, data source changes | S:90 R:95 A:95 D:95 |
| 4 | Certain | Update spec alongside skill per constitution | Confirmed from intake #4 — constitution mandates parallel spec updates | S:95 R:80 A:95 D:95 |
| 5 | Certain | No grep residue in operator skill/spec after edits | Search of operator skill and spec confirms all `runtime is-idle` references removed; remaining matches are only in archived/architecture docs | S:90 R:90 A:90 D:95 |

5 assumptions (5 certain, 0 confident, 0 tentative, 0 unresolved).
