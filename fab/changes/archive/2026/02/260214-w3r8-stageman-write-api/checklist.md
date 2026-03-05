# Quality Checklist: Stageman Write API

**Change**: 260214-w3r8-stageman-write-api
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 set_stage_state: function exists, validates stage/state/file, writes atomically, updates last_updated
- [x] CHK-002 transition_stages: function exists, validates adjacency + from_stage is active, writes both changes atomically
- [x] CHK-003 set_checklist_field: function exists, validates field name and value type, writes atomically
- [x] CHK-004 set_confidence_block: function exists, validates counts/score, replaces entire block atomically
- [x] CHK-005 CLI dispatch: all 4 write subcommands (set-state, transition, set-checklist, set-confidence) work when _stageman.sh is executed directly
- [x] CHK-006 _calc-score.sh refactor: sources _stageman.sh, calls set_confidence_block, stdout output unchanged
- [x] CHK-007 Skill prompts updated: fab-continue.md, fab-ff.md, fab-fff.md, _generation.md reference CLI commands instead of ad-hoc editing

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-008 set_stage_state rejects invalid stage names (exit 1, stderr message, file untouched)
- [x] CHK-009 set_stage_state rejects states not in allowed_states for the stage (exit 1, file untouched)
- [x] CHK-010 transition_stages rejects non-adjacent stages (exit 1, file untouched)
- [x] CHK-011 transition_stages rejects when from_stage is not currently active (exit 1, file untouched)
- [x] CHK-012 All write functions reject nonexistent status file (exit 1, stderr "file not found")
- [x] CHK-013 set_checklist_field rejects invalid field names and wrong value types
- [x] CHK-014 set_confidence_block rejects negative counts and non-numeric scores
- [x] CHK-015 _calc-score.sh now refreshes last_updated (intentional behavioral change)

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-016 Scenario: valid state change (set_stage_state spec pending→active)
- [x] CHK-017 Scenario: normal forward transition (brief→spec)
- [x] CHK-018 Scenario: last pipeline transition (review→hydrate)
- [x] CHK-019 Scenario: set checklist generated true/false
- [x] CHK-020 Scenario: write confidence block with specific values
- [x] CHK-021 Scenario: CLI set-state invocation
- [x] CHK-022 Scenario: CLI missing arguments (usage help, exit 1)
- [x] CHK-023 Scenario: identical _calc-score.sh output after refactor

## Edge Cases & Error Handling
- [x] CHK-024 Temp file created in same directory as target (not /tmp) for atomic mv
- [x] CHK-025 last_updated format matches ISO 8601 with timezone offset
- [x] CHK-026 Existing read accessors unchanged (get_progress_map, get_checklist, get_confidence, get_current_stage)
- [x] CHK-027 --help lists all write commands under a "Write commands" section

## Documentation Accuracy
- [x] CHK-028 fab-continue.md Step 4 references _stageman.sh CLI, no remaining ad-hoc edit instructions for progress/checklist/confidence
- [x] CHK-029 fab-ff.md status transitions reference _stageman.sh CLI
- [x] CHK-030 _generation.md Checklist Generation Procedure Step 6 references _stageman.sh set-checklist

## Cross References
- [x] CHK-031 Existing test suite still passes (src/stageman/test.sh read function tests)
- [x] CHK-032 _calc-score.sh existing tests still pass (src/calc-score/ test suite if present)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
