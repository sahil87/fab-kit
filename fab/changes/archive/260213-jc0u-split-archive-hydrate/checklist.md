# Quality Checklist: Split Archive into Hydrate Stage and fab-archive Command

**Change**: 260213-jc0u-split-archive-hydrate
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Hydrate Replaces Archive in Progress Map: status.yaml template uses `hydrate: pending`, no `archive` key
- [x] CHK-002 Stage Numbering: hydrate maps to position 6 in workflow.yaml and all display logic
- [x] CHK-003 Hydrate Behavior in fab-continue: dispatches correctly when review is done, performs validation/concurrent-check/hydration/status-update
- [x] CHK-004 Hydrate terminal behavior: change folder stays in `fab/changes/`, `fab/current` NOT cleared after hydrate
- [x] CHK-005 fab-archive skill exists: standalone skill with folder move, index update, backlog marking, pointer clearing
- [x] CHK-006 fab-archive guard: requires `hydrate: done`, rejects when hydrate not complete
- [x] CHK-007 fab-archive conditional pointer clear: only clears `fab/current` when the archived change is the active one
- [x] CHK-008 config.yaml stages: terminal stage is `hydrate`, not `archive`
- [x] CHK-009 workflow.yaml schema: `hydrate` stage definition, progression fallback, completion rule all updated
- [x] CHK-010 fab-ff terminal step: stops at hydrate, suggests `/fab-archive`
- [x] CHK-011 fab-fff terminal step: stops at hydrate, suggests `/fab-archive`
- [x] CHK-012 _context.md next steps table: hydrate entries present, `/fab-archive` entry added
- [x] CHK-013 _generation.md references: archive references replaced with hydrate

## Behavioral Correctness

- [x] CHK-014 fab-continue stage guard: `hydrate` in guard table, dispatches hydrate behavior (not archive)
- [x] CHK-015 fab-continue reset flow: `hydrate` is a valid reset target, `archive` is not
- [x] CHK-016 fab-continue next steps: after hydrate → `Next: /fab-archive` (not `/fab-new`)
- [x] CHK-017 fab-archive fail-safe order: folder move → index → backlog → pointer (recoverable on interruption)

## Removal Verification

- [x] CHK-018 No `archive` in progress map: status.yaml template, workflow.yaml stages, config.yaml stages all lack `archive`
- [x] CHK-019 No archive behavior in fab-continue: Archive Behavior section fully replaced by Hydrate Behavior
- [x] CHK-020 No stale `archive` references in skills: all skill files use `hydrate` for the terminal pipeline stage

## Scenario Coverage

- [x] CHK-021 New change creation scenario: `.status.yaml` from template contains `hydrate: pending`
- [x] CHK-022 Hydrate after review pass scenario: `/fab-continue` dispatches hydrate, folder stays
- [x] CHK-023 fab-archive guard scenarios: blocks when hydrate not done (both `hydrate: pending` and `review: done` cases)
- [x] CHK-024 fab-archive non-active change scenario: archives targeted change, does NOT clear `fab/current` when it points elsewhere

## Edge Cases & Error Handling

- [x] CHK-025 Interrupted archive recovery: fab-archive detects folder already in archive and completes remaining steps
- [x] CHK-026 fab-archive with missing backlog: skips backlog marking gracefully when `fab/backlog.md` absent

## Documentation Accuracy

- [x] CHK-027 **N/A**: Centralized docs haven't been updated yet — will be verified during hydrate stage
- [x] CHK-028 fab-archive skill file documents all behaviors specified in spec (guard, fail-safe order, arguments, conditional pointer clearing)

## Cross References

- [x] CHK-029 Skill cross-references consistent: all skills that mention the pipeline use the same 6-stage sequence (brief→spec→tasks→apply→review→hydrate)
- [x] CHK-030 Script cross-references: stageman.sh, fab-preflight.sh, fab-status.sh all reference `hydrate` as terminal stage

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
