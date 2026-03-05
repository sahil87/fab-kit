# Brief: Broaden Archive Backlog Scanning

**Change**: 260213-wloh-archive-backlog-scan
**Created**: 2026-02-13
**Status**: Draft

## Origin

> No skill in the pipeline checks or closes backlog items. `/fab-archive` should scan `fab/backlog.md` for related items and offer to mark them done.

User intent expanded during gap analysis: existing Step 7 already handles exact backlog ID matches. This change adds broader keyword-based scanning so that changes created from natural language descriptions (without a backlog ID) can still surface and close related backlog items at archive time.

## Why

Currently, backlog items are only marked done during archive when the change was created directly from a backlog ID (captured in the brief's Origin section by `/fab-new`). Changes created from natural language descriptions or Linear tickets without a backlog reference silently skip Step 7, leaving related backlog items open even though the work is complete. This creates backlog drift — items that are effectively done but never closed.

## What Changes

- **Extend archive Step 7** in `fab-continue.md` to add a secondary scan after the exact-ID check
- The secondary scan searches `fab/backlog.md` for items whose description keywords overlap with the change's brief title and Why section
<!-- assumed: Match against brief title and Why section — minimal useful scope without over-matching -->
- Surface candidate matches to the user with an interactive confirmation prompt ("Mark as done? [y/n]" per item)
- Preserve the existing exact-ID auto-marking behavior (no regression to current Step 7)

## Affected Docs

### New Docs

(none)

### Modified Docs

- `fab-workflow/change-lifecycle`: Update archive stage documentation to reflect the broader backlog scanning behavior

### Removed Docs

(none)

## Impact

- **`fab/.kit/skills/fab-continue.md`** — primary change target (archive behavior, Step 7)
- **`fab/docs/fab-workflow/change-lifecycle.md`** — doc update to reflect new behavior
- No new dependencies or scripts required — this is a skill prompt change only
- Affects `/fab-ff` and `/fab-fff` indirectly since they invoke archive through `/fab-continue`

## Open Questions

(none — all decisions resolved via SRAD)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Supplement existing Step 7 rather than replace it | Exact-ID matching is reliable and should remain; broader scan is additive |
| 2 | Confident | Use keyword matching from brief content against backlog descriptions | Straightforward text overlap is the obvious approach for a markdown-only workflow |
| 3 | Tentative | Match against brief title and Why section only (not full artifact set) | Minimal useful scope — avoids false positives from spec/tasks implementation details; easily expanded later |

3 assumptions made (2 confident, 1 tentative). Run /fab-clarify to review.
