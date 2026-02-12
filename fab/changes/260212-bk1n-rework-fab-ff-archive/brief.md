# Brief: Rework fab-ff to go all the way to archive

**Change**: 260212-bk1n-rework-fab-ff-archive
**Created**: 2026-02-12
**Status**: Draft

## Origin

**Backlog**: [bk1n]
**Linear**: DEV-1002
**Milestone**: M5: Trial Fixes — Correctness & Ergonomics

User description from Linear:
> Modify fab-ff: fab-ff takes you all the way to archive but can stop at clarifications. fab-fff takes you to archive with auto clarification (doesn't stop, a bit unsafe).

## Why

Currently `/fab-ff` fast-forwards through planning stages (spec, tasks) but stops before implementation. This requires users to manually run `/fab-apply`, `/fab-review`, and `/fab-archive` separately. The goal is to extend `/fab-ff` to be a complete end-to-end pipeline while preserving interactive clarification stops.

This differentiates `/fab-ff` from `/fab-fff`:
- **fab-ff** (after this change): Full pipeline, stops for user clarification when needed
- **fab-fff**: Full pipeline, auto-clarifies without stopping (autonomous, requires confidence >= 3.0)

## What Changes

- Extend `/fab-ff` to invoke `/fab-apply`, `/fab-review`, and `/fab-archive` after completing planning stages
- Preserve existing resumability — re-running after a bail picks up from incomplete stages
- Preserve existing frontloaded questions behavior for planning ambiguities
- Stop and bail if clarifications are needed during execution stages (apply/review) — surface to user for interactive resolution
- Update the skill's output format to reflect completion through archive (or bail point)
- Remove or adjust confidence gate if any (current fab-ff has none; keep it that way)

## Affected Docs

### New Docs
None — this is a modification to existing workflow behavior.

### Modified Docs
- `fab-workflow/fab-ff`: Update to reflect extended pipeline scope (spec + tasks + apply + review + archive)
- `fab-workflow/fab-fff`: Update comparison table to clarify difference (fab-ff stops for clarification, fab-fff auto-clarifies)
- `fab-workflow/planning-skills`: Update fab-ff description to note it's now a full-pipeline command
- `fab-workflow/execution-skills`: Note that fab-ff can now invoke apply/review/archive internally

### Removed Docs
None

## Impact

**Affected Files**:
- `fab/.kit/skills/fab-ff.md` — primary implementation (major rewrite of Behavior section)
- `fab/.kit/skills/fab-fff.md` — update comparison table and description
- `fab/docs/fab-workflow/planning-skills.md` — reclassify fab-ff description
- `fab/docs/fab-workflow/execution-skills.md` — note fab-ff's extended scope

**User Experience**:
- Single command (`/fab-ff`) takes users from brief to archive
- Users who want full autonomy still use `/fab-fff` (with confidence gate)
- Users who want control with speed use `/fab-ff` (stops for clarification, no confidence gate)

**Workflow Impact**:
- Reduces command count for typical changes (1 command vs 4-5)
- Maintains safety through clarification stops
- Preserves resumability on failures

## Open Questions

None — all decision points resolved via SRAD analysis with Certain or Confident grades.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Invoke apply/review/archive skills internally | Consistent with fab-fff pattern; cleaner than inlining |
| 2 | Confident | Preserve frontloaded questions behavior | Not mentioned in requirement; existing valuable feature |
| 3 | Confident | Preserve resumability | Not mentioned in requirement; existing valuable feature |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.
