# Spec: Add bulk confirm mode to fab-clarify

**Change**: 260302-c7is-fab-clarify-bulk-confirm
**Created**: 2026-03-02
**Affected memory**: None — modifies fab-kit skill files only

## Non-Goals

- Changes to Auto Mode — bulk confirm is interactive by nature and only applies to Suggest Mode
- Changes to the confidence scoring formula — the formula stays the same; this change adds a faster path to upgrade assumptions
- Changes to how Tentative or Unresolved assumptions are handled — those continue to flow through the taxonomy scan

## Fab-Clarify: Bulk Confirm Detection

### Requirement: Detect Confident-Dominant Confidence Drag

After reading the target artifact (Step 1) and before the taxonomy scan (Step 2), `/fab-clarify` Suggest Mode SHALL parse the `## Assumptions` table and count assumptions by grade. The skill SHALL trigger the bulk confirm flow when BOTH conditions are met:

1. `confident >= 3` (enough Confident items to materially affect the score)
2. `confident > tentative + unresolved` (Confident assumptions are the dominant drag, not real ambiguity)

When triggered, the skill SHALL execute the bulk confirm flow (Step 1.5) before proceeding to the taxonomy scan (Step 2).

When NOT triggered, the skill SHALL skip Step 1.5 and proceed directly to the taxonomy scan as today.

#### Scenario: Many Confident assumptions dominate

- **GIVEN** a spec with 9 Certain, 11 Confident, 0 Tentative, 0 Unresolved assumptions
- **WHEN** the user runs `/fab-clarify`
- **THEN** the detection triggers (`confident=11 >= 3` AND `confident=11 > tentative+unresolved=0`)
- **AND** the bulk confirm flow is presented before the taxonomy scan

#### Scenario: Confident count below threshold

- **GIVEN** a spec with 8 Certain, 2 Confident, 0 Tentative, 0 Unresolved assumptions
- **WHEN** the user runs `/fab-clarify`
- **THEN** the detection does NOT trigger (`confident=2 < 3`)
- **AND** the skill proceeds directly to the taxonomy scan

#### Scenario: Tentative/Unresolved dominate over Confident

- **GIVEN** a spec with 5 Certain, 4 Confident, 3 Tentative, 1 Unresolved assumptions
- **WHEN** the user runs `/fab-clarify`
- **THEN** the detection does NOT trigger (`confident=4` is NOT `> tentative+unresolved=4`)
- **AND** the skill proceeds directly to the taxonomy scan (Tentative/Unresolved are the real drag)

#### Scenario: Equal Confident and Tentative+Unresolved

- **GIVEN** a spec with 5 Certain, 3 Confident, 2 Tentative, 1 Unresolved assumptions
- **WHEN** the user runs `/fab-clarify`
- **THEN** the detection does NOT trigger (`confident=3` is NOT `> tentative+unresolved=3`; must be strictly greater)
- **AND** the taxonomy scan runs first to address the real ambiguity

## Fab-Clarify: Bulk Confirm Display

### Requirement: Present Confident Assumptions as Numbered List

When bulk confirm is triggered, the skill SHALL display all Confident assumptions in a numbered list using the original `#` column from the `## Assumptions` table. Each entry SHALL show the Decision and Rationale columns.

The display format SHALL be:

```
## Confident Assumptions ({N} items — primary confidence drag)

Review each and respond with: ✓ (confirm), a new value, or ? (explain).

{original_#}. {Decision} — {Rationale}
{original_#}. {Decision} — {Rationale}
...
```

The display MUST NOT use `AskUserQuestion` or any per-item tool call. The list SHALL be presented as plain text output, and the user's next conversational message SHALL be read as the response.

#### Scenario: Display format with original numbering

- **GIVEN** a spec with Confident assumptions at rows #6, #7, #10, #11 in the Assumptions table
- **WHEN** bulk confirm is triggered
- **THEN** the display shows items numbered 6, 7, 10, 11 (original table numbers, not renumbered 1–4)
- **AND** each entry includes the Decision and Rationale text

## Fab-Clarify: Bulk Confirm Response Parsing

### Requirement: Parse Conversational Bulk Response

The skill SHALL parse the user's conversational response to identify per-item actions. The following response formats SHALL be recognized:

| Format | Meaning | Example |
|--------|---------|---------|
| `{#}. ✓` or `{#}. ok` or `{#}. yes` | Confirm assumption | `10. ✓` |
| `{#}.` (bare number with period) | Confirm assumption | `10.` |
| `{#}. {free text}` (not a keyword) | Change value to free text | `10. Make 10 seconds` |
| `{#}. ?` or `{#}. explain` | Request explanation | `10. ?` |
| `{start}-{end}. ✓` or `{start}-{end}. ok` | Confirm range | `11-15. ✓` |
| `all ✓` or `all ok` or `all yes` | Confirm all items | `all ✓` |

Items not mentioned in the response SHALL remain Confident (unchanged).

Parsing SHALL be case-insensitive for keywords (`ok`, `yes`, `explain`). The `✓` character and the word `ok`/`yes` SHALL be treated identically.

#### Scenario: Mixed response with confirms, changes, and explanations

- **GIVEN** bulk confirm is displayed with items #6, #7, #8, #9, #10
- **WHEN** the user responds: `6. ✓ 7. Make 5 seconds 8. ? 9. ok 10.`
- **THEN** items #6, #9, #10 are confirmed (upgraded to Certain)
- **AND** item #7 is changed to "Make 5 seconds" and upgraded to Certain
- **AND** item #8 triggers an explanation request
- **AND** no other items are modified

#### Scenario: Batch range confirmation

- **GIVEN** bulk confirm is displayed with items #6 through #12
- **WHEN** the user responds: `6-10. ✓ 11. Make it 30s 12. ?`
- **THEN** items #6, #7, #8, #9, #10 are confirmed (upgraded to Certain)
- **AND** item #11 is changed and upgraded to Certain
- **AND** item #12 triggers an explanation request

#### Scenario: Confirm all shorthand

- **GIVEN** bulk confirm is displayed with 8 Confident items
- **WHEN** the user responds: `all ok`
- **THEN** all 8 items are confirmed and upgraded to Certain

#### Scenario: Unmentioned items remain Confident

- **GIVEN** bulk confirm is displayed with items #6, #7, #8, #9
- **WHEN** the user responds: `6. ✓ 8. yes`
- **THEN** items #6 and #8 are upgraded to Certain
- **AND** items #7 and #9 remain Confident (unchanged)

### Requirement: Handle Explanation Requests with Re-prompt

When a user requests explanation for an item (`?` or `explain`), the skill SHALL:

1. Provide a brief inline explanation of the assumption's reasoning and implications
2. Re-prompt for ONLY the unexplained items (not the full list)
3. Accept the same response formats for the re-prompted items

The skill SHALL support at most one round of re-prompting. After the re-prompt response, any still-unresolved items SHALL remain Confident.

#### Scenario: Explanation then confirmation

- **GIVEN** the user responds with `8. ?` for a Confident assumption about poll interval
- **WHEN** the skill explains the reasoning
- **THEN** the skill re-prompts: "Still pending: #8. {Decision} — respond with ✓ or a new value"
- **AND** the user responds `8. ok`
- **AND** item #8 is upgraded to Certain

#### Scenario: Explanation with no follow-up

- **GIVEN** the user responds with `8. ?` and `9. ?`
- **WHEN** the skill explains both items and re-prompts
- **AND** the user responds with only `8. ✓` (ignoring #9)
- **THEN** item #8 is upgraded to Certain
- **AND** item #9 remains Confident

## Fab-Clarify: Artifact Update After Bulk Confirm

### Requirement: Update Assumptions Table In Place

For each confirmed or changed item, the skill SHALL update the `## Assumptions` table in the target artifact:

| Action | Grade update | Rationale update | Scores update |
|--------|-------------|-----------------|---------------|
| Confirmed | Confident → Certain | `Clarified — user confirmed` | S → 95 (explicit signal) |
| Changed | Confident → Certain | `Clarified — user changed to {value}` | S → 95 |
| Explained then confirmed | Confident → Certain | `Clarified — user confirmed after explanation` | S → 95 |
| Unmentioned | No change | No change | No change |

The Decision column SHALL be updated for changed items to reflect the user's new value.

#### Scenario: Confirmed item update

- **GIVEN** assumption #7 is Confident with Decision "Bare number = confirm" and Scores `S:50 R:90 A:70 D:65`
- **WHEN** the user confirms item #7
- **THEN** the row updates to: Grade=Certain, Rationale=`Clarified — user confirmed`, Scores=`S:95 R:90 A:70 D:65`

#### Scenario: Changed item update

- **GIVEN** assumption #6 is Confident with Decision "Support batch shorthand" and Scores `S:50 R:95 A:75 D:70`
- **WHEN** the user responds `6. Only support 'all' shorthand, not ranges`
- **THEN** the row updates to: Grade=Certain, Decision=`Only support 'all' shorthand, not ranges`, Rationale=`Clarified — user changed to: Only support 'all' shorthand, not ranges`, Scores=`S:95 R:95 A:75 D:70`

### Requirement: Append to Clarifications Audit Trail

Bulk confirm resolutions SHALL be appended to the `## Clarifications` section using the existing session format. Bulk confirms SHALL use a subsection heading that distinguishes them from individual clarifications:

```markdown
## Clarifications

### Session {YYYY-MM-DD} (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 6 | Confirmed | — |
| 7 | Changed | "Only support 'all' shorthand, not ranges" |
| 8 | Confirmed | After explanation |
```

If a `## Clarifications` section already exists, the new session SHALL be appended. If not, it SHALL be created before the `## Assumptions` section.

#### Scenario: Audit trail with mixed actions

- **GIVEN** a bulk confirm where 3 items are confirmed, 1 changed, and 1 explained-then-confirmed
- **WHEN** the artifact is updated
- **THEN** a `### Session {date} (bulk confirm)` subsection is appended to `## Clarifications`
- **AND** each resolved item appears as a row with its action and detail

## Fab-Clarify: Integration with Existing Flow

### Requirement: Bulk Confirm Precedes Taxonomy Scan

Step 1.5 (bulk confirm) SHALL execute between Step 1 (read artifact) and Step 2 (taxonomy scan). After bulk confirm completes (including any re-prompts), the skill SHALL proceed to Step 2 as normal.

The taxonomy scan (Step 2) MAY find fewer gaps after bulk confirm has resolved Confident assumptions, but it operates independently on the updated artifact.

#### Scenario: Bulk confirm then taxonomy scan finds remaining gaps

- **GIVEN** a spec where bulk confirm resolves 8 Confident assumptions
- **WHEN** bulk confirm completes
- **THEN** the taxonomy scan runs on the now-updated artifact
- **AND** the taxonomy scan may identify structural gaps (Tentative markers, missing scenarios, etc.)

#### Scenario: Bulk confirm resolves all issues

- **GIVEN** a spec where the only confidence drag was Confident assumptions (no Tentative, no markers)
- **WHEN** bulk confirm resolves all Confident items
- **THEN** the taxonomy scan runs and finds no gaps
- **AND** the skill outputs "No gaps found — artifact looks solid."

### Requirement: No Changes to Auto Mode

The bulk confirm flow SHALL NOT be added to Auto Mode. Auto Mode is for autonomous gap resolution where no user interaction is possible. Bulk confirm is inherently interactive.

#### Scenario: Auto Mode skips bulk confirm

- **GIVEN** `/fab-ff` invokes `/fab-clarify` with `[AUTO-MODE]` prefix
- **WHEN** the artifact has many Confident assumptions
- **THEN** the bulk confirm flow is NOT triggered
- **AND** Auto Mode proceeds with its existing autonomous gap resolution

## Preamble Documentation

### Requirement: Document Bulk Confirm in _preamble.md

A `### Bulk Confirm (Confident Assumptions)` subsection SHALL be added under `## Confidence Scoring` in `fab/.kit/skills/_preamble.md`. The subsection SHALL document:

1. What the bulk confirm flow does (upgrade Confident → Certain via conversational bulk response)
2. When it triggers (`confident >= 3` AND `confident > tentative + unresolved`)
3. Where it runs (before taxonomy scan in Suggest Mode only)
4. How confirmed items are updated (Grade, Rationale, Scores changes)

The documentation SHALL be concise (no more than ~10 lines of prose) and reference `/fab-clarify` as the implementing skill.

#### Scenario: User reads preamble for bulk confirm info

- **GIVEN** a user or agent reads `_preamble.md`
- **WHEN** they look under `## Confidence Scoring`
- **THEN** they find a `### Bulk Confirm (Confident Assumptions)` subsection
- **AND** it explains the trigger conditions, flow location, and update behavior

## Design Decisions

1. **Conversational response over AskUserQuestion**: The bulk confirm flow uses plain text display + conversational message parsing instead of per-item `AskUserQuestion` tool calls.
   - *Why*: The motivating session proved this pattern is ~10x faster. `AskUserQuestion` forces per-item round-trips that defeat the purpose of bulk confirmation.
   - *Rejected*: Multi-select `AskUserQuestion` with `multiSelect: true` — this caps at 4 options per question and still requires structured tool-call interaction.

2. **Before taxonomy scan, not after**: Bulk confirm runs as Step 1.5, before the taxonomy scan in Step 2.
   - *Why*: Resolving the score drag first means the taxonomy scan operates on a more accurate artifact. Running after would mean the user sees structural gaps before the quick bulk-confirm opportunity.
   - *Rejected*: After taxonomy scan — the natural pattern from the motivating session was "fix the easy stuff first," and the score drag is the most actionable.

3. **Original table numbering, not sequential renumbering**: Items are displayed with their original `#` column from the Assumptions table.
   - *Why*: Preserves cross-referencing between the displayed list and the artifact's table. Users can look at the spec's Assumptions section and see the same numbers.
   - *Rejected*: Renumbering 1–N — simpler display but breaks the mental link to the source table.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Trigger threshold: `confident >= 3` AND `confident > tentative + unresolved` | Confirmed from intake #1 — explicit design, discussed with concrete score examples | S:85 R:90 A:85 D:80 |
| 2 | Certain | Display as numbered list, not AskUserQuestion | Confirmed from intake #2 — core design principle of this change | S:95 R:90 A:90 D:95 |
| 3 | Certain | Runs before taxonomy scan as Step 1.5 | Confirmed from intake #3 — resolve score drag first, then structural gaps | S:80 R:90 A:85 D:85 |
| 4 | Certain | Unmentioned items stay Confident | Confirmed from intake #4 — no forced confirmation, matches organic pattern | S:75 R:95 A:85 D:85 |
| 5 | Certain | Document pattern in _preamble.md under Confidence Scoring | Confirmed from intake #5 — explicit user request | S:95 R:95 A:90 D:95 |
| 6 | Confident | Support batch shorthand (`11-15. ✓`, `all ✓`) | Carried from intake #6 — natural extension, low risk. Parsing adds minor complexity but users expect range shortcuts | S:50 R:95 A:75 D:70 |
| 7 | Confident | Bare number = confirm (e.g., `10.` without explicit ✓) | Carried from intake #7 — least-effort default. Could be ambiguous if user types number then forgets to add response, but period delimiter helps | S:50 R:90 A:70 D:65 |
| 8 | Certain | Use original Assumptions table # column for item numbering | Codebase signal: Assumptions table already has numbered rows; reusing preserves cross-reference | S:75 R:95 A:90 D:90 |
| 9 | Certain | One round of re-prompting for explanation requests | Natural flow: explain → re-ask → accept outcome. More rounds would defeat the bulk speed advantage | S:70 R:90 A:80 D:85 |
| 10 | Certain | Only S dimension updated to 95 on confirmation (R, A, D unchanged) | S reflects signal strength which genuinely increases with explicit user confirmation. Other dimensions are about the decision's nature, not the user's endorsement | S:80 R:95 A:85 D:90 |
| 11 | Certain | No changes to Auto Mode | Confirmed from intake — bulk confirm is inherently interactive, Auto Mode has no user to confirm with | S:90 R:95 A:95 D:95 |

11 assumptions (9 certain, 2 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
