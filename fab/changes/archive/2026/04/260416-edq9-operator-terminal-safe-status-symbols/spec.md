# Spec: Operator Terminal-Safe Status Symbols

**Change**: 260416-edq9-operator-terminal-safe-status-symbols
**Created**: 2026-04-16
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the status frame layout or column structure — only the health indicator characters change
- Adding ANSI color codes — symbols MUST be shape-distinct without color dependency
- Modifying `.fab-operator.yaml` schema or any persisted state — symbols are rendering-only

## Operator Skill: Status Symbol Replacement

### Requirement: Replace SMP emoji with BMP single-width symbols

The operator skill (`src/kit/skills/fab-operator.md`) SHALL replace all Supplementary Multilingual Plane emoji and variable-width Unicode status indicators with single-width Basic Multilingual Plane Unicode symbols. The replacement mapping SHALL be:

| Current | Replacement | Unicode | Meaning |
|---------|-------------|---------|---------|
| 🟢 | ● | U+25CF BLACK CIRCLE | active/healthy |
| 🟡 | ◌ | U+25CC DOTTED CIRCLE | idle/new items |
| 🔴 | ✗ | U+2717 BALLOT X | stuck/errored |
| ⏸ | – | U+2013 EN DASH | paused |
| ✓ | ✓ | (unchanged) | complete |

All five replacement characters MUST be in the Basic Multilingual Plane (U+0000–U+FFFF) and MUST render as single-width in standard monospace terminal fonts.

#### Scenario: Status frame renders with new symbols

- **GIVEN** the operator is running and has tracked changes and watches
- **WHEN** a monitoring tick produces the status frame
- **THEN** active/healthy items display `●`, idle/warning items display `◌`, stuck/errored items display `✗`, and complete items display `✓`
- **AND** all columns remain aligned (no double-width character shifts)

#### Scenario: Watch health uses new symbols

- **GIVEN** the operator has active watches
- **WHEN** a watch is healthy, has new items, is errored, or is paused
- **THEN** the watch row displays `●`, `◌`, `✗`, or `–` respectively

### Requirement: Update status frame code block example

The status frame example in the monitoring tick section (§4) SHALL use the new symbols in place of emoji. The example SHALL demonstrate at least one instance of each active symbol (●, ◌, ✗, ✓).

#### Scenario: Code block example uses new symbols

- **GIVEN** a reader views the status frame example in the skill file
- **WHEN** they read the code block
- **THEN** all health indicators use BMP symbols (●, ◌, ✗, ✓), not emoji (🟢, 🟡, 🔴)

### Requirement: Update health legend text

The change health and watch health legend lines SHALL use the new symbols in their definitions. The legend text SHALL read:

- **Change health**: `● active, ◌ idle, ✗ stuck (>15m idle at non-terminal), ✓ complete.`
- **Watch health**: `● healthy (last query succeeded, no new items), ◌ has new unprocessed items, ✗ errored (last_error set), – paused (enabled: false).`

#### Scenario: Health legend matches new symbols

- **GIVEN** a reader views the health legend in the skill file
- **WHEN** they read the change health or watch health definition
- **THEN** the symbols match the replacement mapping (●, ◌, ✗, –) with no emoji present

### Requirement: Update autopilot queue reference

The autopilot queue progress text (§6) that references `🟢/🟡` as current-item indicators SHALL be updated to reference `●/◌`.

#### Scenario: Autopilot reference uses new symbols

- **GIVEN** a reader views the autopilot queue progress description
- **WHEN** they read the indicator reference
- **THEN** it reads `●/◌` instead of `🟢/🟡`

### Requirement: Update Health column description

The status frame column table's Health row currently reads "Status emoji". This SHALL be updated to "Status indicator" to reflect that the symbols are no longer emoji.

#### Scenario: Column table uses updated terminology

- **GIVEN** a reader views the status frame column layout table
- **WHEN** they read the Health column description
- **THEN** it reads "Status indicator — universal position across all types"

## Operator Spec: Corresponding Update

### Requirement: Update SPEC-fab-operator.md

Per constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md` file"), `docs/specs/skills/SPEC-fab-operator.md` SHALL be updated to reflect the new status symbols wherever the old emoji are referenced.

#### Scenario: Spec file reflects new symbols

- **GIVEN** the skill file has been updated with new symbols
- **WHEN** a reader checks the spec file
- **THEN** any references to status indicators use the new BMP symbols, not the old emoji

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace 🟢 with ● (U+25CF) for active/healthy | Confirmed from intake #1 — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 2 | Certain | Replace 🟡 with ◌ (U+25CC) for idle/new items | Confirmed from intake #2 — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 3 | Certain | Replace 🔴 with ✗ (U+2717) for stuck/errored | Confirmed from intake #3 — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 4 | Certain | Replace ⏸ with – (U+2013) for paused | Confirmed from intake #4 — user specified exact mapping | S:95 R:90 A:95 D:95 |
| 5 | Certain | Keep ✓ unchanged | Confirmed from intake #5 — already single-width BMP | S:95 R:95 A:95 D:95 |
| 6 | Certain | Option 2 (distinct BMP shapes) over ANSI-colored or ASCII-only | Confirmed from intake #6 — ANSI stripping risk, ASCII noise | S:95 R:85 A:90 D:95 |
| 7 | Certain | SPEC-fab-operator.md needs corresponding update | Constitution mandates it; confirmed from intake #7 | S:95 R:80 A:95 D:95 |
| 8 | Certain | No migration needed | Confirmed from intake #8 — symbols in skill prose only | S:95 R:85 A:95 D:95 |
| 9 | Certain | Update "Status emoji" column description to "Status indicator" | Terminology should match — no longer emoji | S:90 R:95 A:90 D:90 |

9 assumptions (9 certain, 0 confident, 0 tentative, 0 unresolved).
