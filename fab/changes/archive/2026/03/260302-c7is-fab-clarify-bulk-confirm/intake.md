# Intake: Add bulk confirm mode to fab-clarify

**Change**: 260302-c7is-fab-clarify-bulk-confirm
**Created**: 2026-03-02
**Status**: Draft

## Origin

> Add a "bulk confirm" mode to /fab-clarify for upgrading Confident assumptions to Certain. When the primary confidence drag is many Confident (not Tentative/Unresolved) assumptions, fab-clarify should detect this and offer a bulk review flow instead of one-at-a-time AskUserQuestion.

This emerged from a retrospective on session `260302-c5vz-tu-watch-mode` (tu --watch mode). After spec generation, confidence was 1.7/5.0 with 9 Certain and 11 Confident assumptions. The `/fab-clarify` skill ran its taxonomy scan, found 3 real structural gaps (interface mismatch, scope omission, edge case), and resolved them — but the score stayed at 1.7 because the scan targets `<!-- assumed: -->` markers and `[NEEDS CLARIFICATION]`, which correspond to Tentative/Unresolved items. Confident assumptions have NO markers — they only exist in the `## Assumptions` table. So `/fab-clarify` literally cannot see them as actionable items.

The user then bypassed the skill entirely, asked "Show me the confident assumptions", and reviewed all 11 in a single conversational message:

> `10. Make 10 seconds. 11. yes 12. yes. 13. Explain pls 14. ok 15. ok 16. Explain pls 17. ok 18 ok. 19. ok 20 ok`

This resolved 9 assumptions instantly (plus 2 after brief explanations). The agent upgraded all 11 from Confident → Certain. Score jumped 1.7 → 5.0. This organic pattern was ~10x faster than individual `AskUserQuestion` round-trips would have been.

## Why

The confidence score formula penalizes Confident assumptions (`-0.3 * confident`). With many Confident items, the score can drop well below `/fab-ff` gate thresholds even when the spec is actually solid — "Confident" means "strong signal, one obvious interpretation," not "uncertain." The only way to raise the score is to upgrade Confident → Certain, which requires user confirmation.

Currently there is no efficient path for this:
- `/fab-clarify` scans for markers (`<!-- assumed: -->`, `[NEEDS CLARIFICATION]`) — Confident assumptions have none
- `/fab-continue` would advance to the next stage, ignoring the low score
- `/fab-ff` gates on the score and refuses to proceed
- The only option is manual: user asks to see assumptions, bulk-confirms conversationally

Without this change, users will either hit the `/fab-ff` gate wall and not know why, or waste time in slow one-at-a-time confirmation loops for decisions that just need a quick "yes."

## What Changes

### New "Bulk Confirm" step in `/fab-clarify` Suggest Mode

Add a new **Step 1.5** (between reading the artifact and the taxonomy scan) that detects when Confident assumptions are the primary confidence drag and offers a bulk review flow.

#### Detection logic

After reading the target artifact's `## Assumptions` table, count assumptions by grade. Trigger bulk confirm when:
- `confident >= 3` (enough to materially affect the score)
- `confident > tentative + unresolved` (Confident is the dominant drag, not real ambiguity)

If triggered, skip straight to the bulk confirm flow. After bulk confirm completes, proceed to the existing taxonomy scan (Step 2) for any remaining gaps.

If not triggered (few Confident items, or Tentative/Unresolved dominate), proceed directly to the taxonomy scan as today.

#### Bulk confirm flow

1. Display all Confident assumptions in a numbered list with their Decision and Rationale columns:
   ```
   ## Confident Assumptions ({N} items — primary confidence drag)

   Review each and respond with: ✓ (confirm), a new value, or ? (explain).

   10. Default poll interval 30 seconds — balances freshness vs API load
   11. --json + --watch incompatible — fundamentally different modes
   12. --sync runs once at start — git push/pull too heavy for 30s
   ...
   ```

2. Wait for a single conversational response. The user can:
   - Confirm items: `10. ✓` or `10. ok` or `10. yes` or just `10.` (bare number = confirm)
   - Change a value: `10. Make 10 seconds` (free text = override)
   - Request explanation: `10. ?` or `10. explain` (agent explains, then user confirms/changes)
   - Batch shorthand: `11-15. ✓` or `all ✓` for ranges

3. Parse the response:
   - Confirmed items → upgrade to Certain, update Rationale to `Clarified — user confirmed`, update Scores to reflect explicit signal (S:95)
   - Changed items → upgrade to Certain, update Decision with new value, update Rationale to `Clarified — user changed to {value}`
   - Explanation requests → provide a brief explanation inline, then re-prompt for that item only
   - Unmentioned items → leave as Confident (not touched)

4. Update the `## Assumptions` table in the artifact in place. Add entries to the Clarifications audit trail.

#### Interaction style

This MUST NOT use `AskUserQuestion` — the whole point is to avoid per-item tool call round-trips. Instead, display the list as plain text output and read the user's next conversational message as the response. This matches how the pattern worked organically in the motivating session.

### Document bulk confirm in `_preamble.md`

Add a subsection under `## Confidence Scoring` in `fab/.kit/skills/_preamble.md` documenting the bulk confirm pattern as an alternative flow:

```markdown
### Bulk Confirm (Confident Assumptions)

When the confidence score is low primarily due to many Confident (not Tentative/Unresolved)
assumptions, `/fab-clarify` offers a bulk confirm flow. This displays all Confident assumptions
in a numbered list and lets the user confirm, change, or request explanation in a single
conversational turn — typically 10x faster than individual question/answer cycles.

Detection: triggered when `confident >= 3` and `confident > tentative + unresolved`.

This flow runs before the standard taxonomy scan. Items confirmed are upgraded to Certain;
items changed are updated and upgraded; items not mentioned remain Confident.
```

## Affected Memory

None — this change modifies fab-kit skill files, not project memory files.

## Impact

- **`fab/.kit/skills/fab-clarify.md`**: Add Step 1.5 (bulk confirm detection + flow) to Suggest Mode, between Step 1 and Step 2. Update Step 2 to note it runs after bulk confirm (if triggered). No changes to Auto Mode.
- **`fab/.kit/skills/_preamble.md`**: Add `### Bulk Confirm (Confident Assumptions)` subsection under `## Confidence Scoring`.

## Open Questions

None — the design is well-defined from the motivating session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Trigger threshold: `confident >= 3` AND `confident > tentative + unresolved` | Discussed — 3+ Confident items materially drag the score; condition ensures we don't trigger when real ambiguity (Tentative/Unresolved) is the dominant issue | S:85 R:90 A:85 D:80 |
| 2 | Certain | Display as numbered list, not AskUserQuestion | Discussed — the whole point is avoiding per-item tool call round-trips. Organic session proved conversational bulk response is ~10x faster | S:95 R:90 A:90 D:95 |
| 3 | Certain | Runs before taxonomy scan, not after | Discussed — bulk confirm resolves the score drag first, then taxonomy scan catches remaining structural gaps | S:80 R:90 A:85 D:85 |
| 4 | Certain | Unmentioned items stay Confident | Discussed — no forced confirmation. User only touches what they want to. Matches the organic pattern | S:75 R:95 A:85 D:85 |
| 5 | Certain | Document pattern in _preamble.md under Confidence Scoring | User explicitly requested this in the retrospective discussion | S:95 R:95 A:90 D:95 |
| 6 | Confident | Support batch shorthand (`11-15. ✓`, `all ✓`) | Natural extension of the pattern — users often want to confirm a range. Low risk, easily reversible | S:50 R:95 A:75 D:70 |
| 7 | Confident | Bare number = confirm (e.g., `10.` without explicit ✓) | Follows least-effort principle — most items are confirmed, not changed. But could be ambiguous | S:50 R:90 A:70 D:65 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
