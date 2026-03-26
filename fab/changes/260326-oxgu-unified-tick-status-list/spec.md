# Spec: Unified Tick Status List

**Change**: 260326-oxgu-unified-tick-status-list
**Created**: 2026-03-26
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Operator: Tick Status Frame

### Requirement: Unified Entry List

The operator tick status frame (§4 Tick Behavior) SHALL render all tracked items — monitored changes and watches — as entries in a single flat list. There SHALL be no visual grouping by type (no blank-line separators, no separate sections). The previous two-block layout (changes block + watches block with 👁 prefix) is replaced entirely.

#### Scenario: Mixed changes and watches

- **GIVEN** the operator monitors 3 changes and 2 watches
- **WHEN** a tick fires
- **THEN** the status frame renders 5 entries in a single list
- **AND** changes appear first (sorted by enrollment time), followed by watches (sorted alphabetically by name)

#### Scenario: Only changes, no watches

- **GIVEN** the operator monitors 2 changes and 0 watches
- **WHEN** a tick fires
- **THEN** the status frame renders 2 entries, all with `[change]` type prefix

#### Scenario: Only watches, no changes

- **GIVEN** the operator monitors 0 changes and 1 watch
- **WHEN** a tick fires
- **THEN** the status frame renders 1 entry with `[watch]` type prefix

### Requirement: Column Structure

Every entry in the status frame SHALL follow a consistent 5-column layout:

| Column | Content |
|--------|---------|
| Type | `[change]` or `[watch]` — bracketed type prefix |
| ID | Change ID (4-char) or watch name |
| Autopilot | `▶` if autopilot-driven, blank otherwise |
| Health | Status emoji — universal position across all types |
| Detail | Type-specific status text |

#### Scenario: Change entry rendering

- **GIVEN** a monitored change `r3m7` is active at the apply stage, driven by autopilot
- **WHEN** the status frame renders
- **THEN** the entry reads: `[change]  r3m7  ▶ 🟢 apply → review`

#### Scenario: Watch entry rendering

- **GIVEN** a watch `linear-bugs` with 2 known items, 1 completed, last checked 3 minutes ago
- **WHEN** the status frame renders
- **THEN** the entry reads: `[watch]  linear-bugs  🟢 2 known · 1 completed · 3m ago`

### Requirement: Type Indicators

Each entry SHALL have a bracketed type prefix as the first column:

- `[change]` — monitored pipeline changes
- `[watch]` — watch sources (Linear, Slack, etc.)

Future tracked-source types SHALL follow the same `[type]` pattern.

#### Scenario: Type prefix format

- **GIVEN** any tracked item
- **WHEN** rendered in the status frame
- **THEN** the entry begins with `[{type}]` where type is lowercase

### Requirement: Autopilot as Per-Change Property

The `▶` symbol SHALL mark changes being driven by the autopilot queue. This replaces the header-level `autopilot 1/3` indicator. Non-autopilot changes (manually enrolled or watch-spawned) SHALL NOT display the `▶` symbol.

#### Scenario: Autopilot-driven change

- **GIVEN** a change `cd34` is in the autopilot queue and currently being worked
- **WHEN** the status frame renders
- **THEN** the entry shows `▶` between the ID and health emoji columns

#### Scenario: Non-autopilot change

- **GIVEN** a change `ab12` was manually enrolled (not in autopilot queue)
- **WHEN** the status frame renders
- **THEN** the entry shows blank space where `▶` would be

#### Scenario: Autopilot queue visible from entries

- **GIVEN** an autopilot queue of [ab12, cd34, ef56] with cd34 current
- **WHEN** the status frame renders
- **THEN** all three entries show `▶`, and their relative positions plus individual health emojis convey queue state (completed entries show ✓, current shows 🟢/🟡, pending show their actual agent state)

### Requirement: Universal Health Emoji

Both changes and watches SHALL display a status emoji in the same column position (Health column).

**Change health emojis** (existing semantics, unchanged):
- 🟢 active
- 🟡 idle
- 🔴 stuck (>15m idle at non-terminal stage)
- ✓ complete

**Watch health emojis** (new):
- 🟢 healthy — last query succeeded, no new items
- 🟡 has new unprocessed items
- 🔴 errored — `last_error` is set
- ⏸ paused — `enabled: false`

#### Scenario: Healthy watch

- **GIVEN** a watch `slack-alerts` with `last_error: null`, `enabled: true`, and 0 new items
- **WHEN** the status frame renders
- **THEN** the health emoji is 🟢

#### Scenario: Watch with new items

- **GIVEN** a watch `gmail-deploys` with 1 new unprocessed item
- **WHEN** the status frame renders
- **THEN** the health emoji is 🟡

#### Scenario: Errored watch

- **GIVEN** a watch `linear-bugs` with `last_error: "API timeout"`
- **WHEN** the status frame renders
- **THEN** the health emoji is 🔴

#### Scenario: Paused watch

- **GIVEN** a watch `linear-bugs` with `enabled: false`
- **WHEN** the status frame renders
- **THEN** the health emoji is ⏸

### Requirement: Header Line

The status frame header SHALL use a single `N tracked` count representing the total number of entries (changes + watches). The header SHALL NOT include per-type counts or autopilot state.

Format: `── Operator ── {HH:MM} ── tick #{N} ── {total} tracked ──`

#### Scenario: Header with mixed types

- **GIVEN** 4 monitored changes and 3 watches
- **WHEN** the status frame renders
- **THEN** the header reads `── Operator ── 17:32 ── tick #47 ── 7 tracked ──`

#### Scenario: Empty monitored set

- **GIVEN** 0 changes and 0 watches
- **WHEN** a tick fires
- **THEN** the status frame shows `── Operator ── 17:32 ── tick #47 ── 0 tracked ──` (or the loop stops per existing lifecycle rules)

### Requirement: Watch Detail Timestamps

Watch entries SHALL use relative timestamps (`{N}m ago`, `{N}h ago`) for the `last_checked` field instead of absolute timestamps (`last check HH:MM`).

Format: `{N}s ago` (< 60s), `{N}m ago` (60s–59m), `{N}h ago` (>= 60m). Floor division, matching the idle duration format used for changes.

#### Scenario: Recent check

- **GIVEN** a watch last checked 3 minutes ago
- **WHEN** the status frame renders
- **THEN** the detail shows `3m ago` (not `last check 17:29`)

### Requirement: Autopilot Section Updates

The §6 Autopilot section's references to `autopilot 1/3` in the header SHALL be updated to describe the `▶` per-entry approach. The autopilot `.fab-operator.yaml` schema is unchanged — only the rendering description changes.

#### Scenario: Autopilot documentation consistency

- **GIVEN** a reader of §6 Autopilot
- **WHEN** they read about autopilot visibility in the status frame
- **THEN** the documentation references `▶` per-entry indicators, not header-level `autopilot 1/3`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Type indicators use `[type]` bracket syntax | Confirmed from intake #1 — user explicitly confirmed bracket prefix approach | S:90 R:90 A:85 D:90 |
| 2 | Certain | Changes ordered before watches in the list | Confirmed from intake #2 — primary items first, natural reading order | S:70 R:95 A:85 D:80 |
| 3 | Certain | Header uses single "N tracked" count, no per-type counts, no autopilot in header | Confirmed from intake #3 — user chose `▶` per-entry over header-level indicator | S:95 R:90 A:90 D:95 |
| 4 | Certain | Autopilot is a per-change property shown via `▶` symbol | Confirmed from intake #4 — user proposed and chose arrow symbol | S:95 R:85 A:90 D:95 |
| 5 | Certain | Universal health emoji column for all entry types | Confirmed from intake #5 — user confirmed unified emoji approach | S:90 R:90 A:85 D:90 |
| 6 | Confident | Watch detail uses relative timestamps (`3m ago`) instead of absolute | Confirmed from intake #6 — follows from unified structure, more scannable | S:65 R:95 A:75 D:70 |
| 7 | Confident | No blank-line separators between types | Confirmed from intake #7 — flat list rendering | S:65 R:95 A:70 D:70 |
| 8 | Certain | Watch health emoji semantics: 🟢 healthy, 🟡 new items, 🔴 errored, ⏸ paused | Derived from intake #5 discussion — maps watch states to existing emoji vocabulary | S:80 R:90 A:85 D:85 |
| 9 | Certain | `.fab-operator.yaml` schema unchanged — only rendering description changes | Intake specifies rendering-only change; autopilot YAML fields persist as-is | S:85 R:95 A:90 D:90 |

9 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved).
