# Spec: Broaden Archive Backlog Scanning

**Change**: 260213-wloh-archive-backlog-scan
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/change-lifecycle.md`

## Non-Goals

- Fuzzy or semantic matching — keyword overlap is sufficient for a markdown-only workflow
- Backlog scanning outside of archive — this only runs during `/fab-continue` archive Step 7
- Auto-closing without confirmation — all keyword matches require user approval

## Archive Backlog Scanning: Keyword-Based Candidate Matching

### Requirement: Preserve Exact-ID Auto-Marking

The existing archive Step 7 behavior SHALL remain unchanged: when the brief's Origin section contains a backlog ID (e.g., `[qnzx]`), the system SHALL automatically find and mark the corresponding backlog item as done without user interaction.

#### Scenario: Brief With Backlog ID

- **GIVEN** the brief's Origin section contains a backlog ID `[qnzx]`
- **WHEN** archive Step 7 executes
- **THEN** the item with ID `[qnzx]` in `fab/backlog.md` SHALL be marked done and moved to the Done section
- **AND** the secondary keyword scan SHALL still execute (may surface additional related items)

### Requirement: Secondary Keyword Scan

After the exact-ID check (whether it matched or not), archive Step 7 SHALL perform a secondary keyword scan against `fab/backlog.md` to surface candidate matches. The keyword scan SHALL only execute in interactive mode — when archive runs via `/fab-ff` or `/fab-fff` (auto mode), the keyword scan SHALL be skipped entirely and only the exact-ID check runs.
<!-- clarified: keyword scan skipped in auto mode — only exact-ID runs during /fab-ff and /fab-fff -->

The scan SHALL:

1. Extract significant keywords from the brief's title (the `# Brief: {title}` heading) and the Why section content
2. Filter out common stop words (articles, prepositions, conjunctions, common verbs like "is", "are", "has", "should", "must", "the", "a", "an", "in", "of", "to", "for", "and", "or", "with", "from", "by", "on", "at", "this", "that", "it", "be", "not", "no")
3. Normalize keywords to lowercase for case-insensitive comparison
4. Compare extracted keywords against each unchecked (`- [ ]`) backlog item's description text
5. A backlog item is a candidate match when at least 2 significant keywords overlap with the item's description
<!-- clarified: 2-keyword minimum threshold confirmed by user -->

#### Scenario: Keywords Match a Backlog Item

- **GIVEN** the brief title is "Broaden Archive Backlog Scanning" and the Why section mentions "backlog items", "archive", and "marked done"
- **WHEN** archive Step 7 performs the keyword scan
- **AND** a backlog item reads "No skill in the pipeline checks or closes backlog items. `/fab-archive` should scan `fab/backlog.md` for related items and offer to mark them done."
- **THEN** the item SHALL be surfaced as a candidate match (keywords "backlog", "archive", "scan" overlap)

#### Scenario: No Keyword Matches

- **GIVEN** the brief's keywords do not overlap (at 2+ keywords) with any unchecked backlog item
- **WHEN** archive Step 7 performs the keyword scan
- **THEN** no candidates are surfaced
- **AND** the archive proceeds silently (no output for this sub-step)

#### Scenario: Auto Mode Skips Keyword Scan

- **GIVEN** archive runs via `/fab-ff` or `/fab-fff` (auto mode)
- **WHEN** archive Step 7 executes
- **THEN** the exact-ID check SHALL run as normal
- **AND** the keyword scan SHALL be skipped entirely (no candidates surfaced, no prompt)

#### Scenario: Item Already Marked by Exact-ID Check

- **GIVEN** the exact-ID check already marked item `[qnzx]` as done
- **WHEN** the keyword scan also matches item `[qnzx]`
- **THEN** the item SHALL be excluded from the keyword scan candidates (already handled)

### Requirement: Interactive Confirmation Per Candidate

For each candidate match surfaced by the keyword scan, the system SHALL present an interactive confirmation prompt to the user. The system MUST NOT auto-close items matched only by keywords.

The prompt format SHALL be:

```
Backlog matches found:
  1. [ID] {description (truncated to ~80 chars)}
  2. [ID] {description (truncated to ~80 chars)}

Mark as done? (comma-separated numbers, or "none")
```
<!-- assumed: batch prompt rather than per-item y/n — fewer interruptions for multiple matches -->

#### Scenario: User Confirms One Match

- **GIVEN** the keyword scan surfaces 2 candidate matches
- **WHEN** the user responds with "1"
- **THEN** only the first item SHALL be marked done and moved to the Done section in `fab/backlog.md`
- **AND** the second item SHALL remain unchecked

#### Scenario: User Confirms Multiple Matches

- **GIVEN** the keyword scan surfaces 3 candidate matches
- **WHEN** the user responds with "1,3"
- **THEN** items 1 and 3 SHALL be marked done and moved to the Done section
- **AND** item 2 SHALL remain unchecked

#### Scenario: User Declines All

- **GIVEN** the keyword scan surfaces candidates
- **WHEN** the user responds with "none"
- **THEN** no items are marked done
- **AND** the archive proceeds normally

### Requirement: Backlog File Mutation

When an item is marked done (whether by exact-ID or keyword scan), the mutation SHALL:

1. Change the item's checkbox from `- [ ]` to `- [x]`
2. Move the item line from its current location in the Backlog section to the Done section
3. Prepend the done item to the Done section (most-recent-first ordering, consistent with archive index)

This matches the existing exact-ID behavior and ensures consistency.

#### Scenario: Item Moved to Done Section

- **GIVEN** backlog item `[qnzx]` is in the "Cleanup & Hardening" subsection of the Backlog section
- **WHEN** the item is marked done
- **THEN** the item SHALL be moved to the Done section as `- [x] [qnzx] {date}: {description}`
- **AND** the item SHALL be removed from its original location in the Backlog section

## Design Decisions

1. **Batch prompt instead of per-item y/n**: Present all keyword matches at once with a comma-separated selection rather than individual yes/no prompts per item.
   - *Why*: Fewer interruptions. Keyword matches may surface 2-5 items; individual prompts would be tedious. Users can review all candidates and decide at once.
   - *Rejected*: Per-item y/n — more interruptions, slower UX for multiple matches.

2. **2-keyword minimum overlap threshold**: Require at least 2 significant keywords to match for an item to be a candidate.
   - *Why*: Single-keyword matches would produce too many false positives (e.g., "add" or "fix" appearing in many items). Two keywords provide reasonable precision while still catching related items.
   - *Rejected*: Single-keyword match — too noisy. Three or more — too strict, would miss legitimate matches.

3. **Keywords from title + Why section only (not full artifact set)**: Extract keywords only from the brief's title and Why section, not from spec, tasks, or other artifacts.
   - *Why*: The brief captures the user's original intent in natural language. Spec and tasks contain implementation details that would produce false-positive matches (e.g., matching every item that mentions "file" or "script"). This aligns with the brief's tentative assumption.
   - *Rejected*: Full artifact keyword extraction — too many false positives from implementation language.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Batch prompt for keyword matches instead of per-item y/n | Standard UX for multi-item selection; fewer interruptions |
| 2 | Confident | List all keyword matches without ranking/scoring | User reviews and decides; ranking adds complexity for little gain in a typically small list |
| 3 | Certain | 2-keyword minimum overlap threshold | Confirmed by user during clarification |
| 4 | Confident | Keywords from title + Why only (carried from brief) | Avoids implementation-language false positives; matches user intent |

4 assumptions made (3 confident, 1 certain). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-13

- **Q**: What should happen with keyword scan candidates when archive runs autonomously via `/fab-ff` or `/fab-fff`?
  **A**: Skip keyword scan in auto mode — only exact-ID matching runs. Keyword matches are interactive-only.
- **Q**: Is the 2-keyword minimum overlap threshold the right threshold for candidate matching?
  **A**: Accepted recommendation: 2-keyword minimum confirmed.
