# Quality Checklist: Standardize State-Keyed Next-Step Suggestions

**Change**: 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Canonical State Table: `_context.md` contains a state-keyed table with all 9 states (none, initialized, intake, spec, tasks, apply, review pass, review fail, hydrate) and correct available/default commands
- [x] CHK-002 Skill-keyed table removed: The old "After skill | Stage reached | Next line" table no longer exists in `_context.md`
- [x] CHK-003 Lookup convention documented: `_context.md` instructs skills to derive Next: from state table with default command listed first
- [x] CHK-004 Activation preamble documented: `_context.md` defines the preamble convention for non-active resources (fab-new, fab-archive restore)
- [x] CHK-005 fab-new --switch removed: `fab-new.md` has no `--switch` argument, no Step 6 conditional activation, no natural language switching detection
- [x] CHK-006 fab-new single suggestion: `fab-new.md` has exactly one Next: path using activation preamble with intake state commands
- [x] CHK-007 fab-switch private table removed: `fab-switch.md` has no inline stage→suggestion table, references canonical state table instead
- [x] CHK-008 fab-clarify stage guard extended: `fab-clarify.md` accepts `intake` as a valid stage alongside `spec` and `tasks`

## Behavioral Correctness
- [x] CHK-009 fab-clarify intake suggestions: From intake state, suggests /fab-continue, /fab-fff, /fab-clarify (NOT /fab-ff)
- [x] CHK-010 fab-clarify spec suggestions: From spec state, suggests /fab-continue, /fab-ff, /fab-clarify (NOT /fab-fff)
- [x] CHK-011 fab-clarify tasks suggestions: From tasks state, suggests /fab-continue, /fab-ff, /fab-clarify
- [x] CHK-012 fab-archive restore with --switch: Derives Next: directly from restored change's state
- [x] CHK-013 fab-archive restore without --switch: Uses activation preamble before state-derived commands

## Removal Verification
- [x] CHK-014 No hardcoded Next: fab-ff: `fab-ff.md` no longer has `Next: /fab-archive` hardcoded
- [x] CHK-015 No hardcoded Next: fab-fff: `fab-fff.md` no longer has `Next: /fab-archive` hardcoded
- [x] CHK-016 No hardcoded Next: fab-archive: `fab-archive.md` no longer has `Next: /fab-new <description>` or `Next: /fab-switch {name}` hardcoded
- [x] CHK-017 No hardcoded Next: fab-setup: `fab-setup.md` no longer has hardcoded suggestion text
- [x] CHK-018 No hardcoded Next: fab-continue: `fab-continue.md` no longer has `Next: /fab-continue` hardcoded in review pass
- [x] CHK-019 No hardcoded Next: docs-hydrate-memory: `docs-hydrate-memory.md` no longer has hardcoded suggestion text
- [x] CHK-020 No hardcoded Next: fab-clarify: `fab-clarify.md` no longer has `Next: /fab-clarify or /fab-continue or /fab-ff` hardcoded

## Scenario Coverage
- [x] CHK-021 Same state same suggestion: Multiple skills reaching the same state (e.g., hydrate) produce identical suggestions
- [x] CHK-022 Default command first: In all state table entries, the default command is listed first in the Next: output format
- [x] CHK-023 fab-status uses state table: `fab-status.md` references the state table for suggested next command

## Edge Cases & Error Handling
- [x] CHK-024 review (fail) state: State table correctly shows (rework menu) with no default command
- [x] CHK-025 (none) state: State table includes pre-init state with /fab-setup as the only command
- [x] CHK-026 No stale references: No skill file references the removed skill-keyed "Lookup Table" or "After skill" table

## Code Quality
- [x] CHK-027 Pattern consistency: All updated skills use consistent language for state table references (e.g., "per state table in `_context.md`" or equivalent)
- [x] CHK-028 No unnecessary duplication: No skill duplicates the state table content — all reference the single canonical table

## Documentation Accuracy
- [x] CHK-029 State derivation rules: The state table includes clear derivation rules mapping progress map values to states
- [x] CHK-030 Convention text complete: The updated Next Steps Convention fully explains the lookup procedure (determine state, look up, output with default first)

## Cross References
- [x] CHK-031 Specs check: `docs/specs/skills.md` and `docs/specs/user-flow.md` reviewed for stale suggestion documentation — flagged if found

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

## Review Notes

- **CHK-031 flag**: `docs/specs/skills.md` contains a stale skill-keyed lookup table (lines ~54-68) that references the old "After skill | Stage reached | Next line" format. This should be updated to reflect the new state-keyed table — flagged for human curation per constitution (specs are human-curated, not auto-modified).
- `docs/specs/user-flow.md` — no stale suggestion documentation found.
