# Quality Checklist: Add Operator3 — Auto-Nudge for Blocked Agents

**Change**: 260312-ngew-add-operator3-auto-nudge
**Generated**: 2026-03-12
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Question Detection: Skill file defines terminal heuristic using `tmux capture-pane -t <pane> -p -l 10` for idle agents
- [x] CHK-002 Question Patterns: All specified indicator patterns are documented (lines ending `?`, `[Y/n]`/`[y/N]`/`(y/n)`/`(yes/no)`, keywords, Claude Code prompts, phrasing patterns)
- [x] CHK-003 Answer Confidence Model: Two-tier classification (auto-answer vs escalate) is fully defined with examples for each tier
- [x] CHK-004 Classification Heuristic: Binary yes/no where "yes" continues and "no" stalls, excluding destructive/branching → auto-answer; everything else → escalate
- [x] CHK-005 Monitoring Tick: 6-step tick order documented with input-waiting detection as step 5
- [x] CHK-006 Inheritance: Skill file references operator2 for all inherited behavior (UC1–UC8, monitoring, enrollment, configuration)
- [x] CHK-007 Launcher Script: `fab-operator3.sh` exists, mirrors `fab-operator2.sh`, uses singleton `operator` tab, launches `/fab-operator3`
- [x] CHK-008 Per-Skill Spec: `SPEC-fab-operator3.md` documents behavior, primitives, question detection, answer confidence, and relationship to operator2

## Behavioral Correctness
- [x] CHK-009 Auto-answer action: Auto-answer tier sends response via `tmux send-keys` and reports "{change}: auto-answered '{summary}' → {answer}"
- [x] CHK-010 Escalation action: Escalate tier reports "{change}: waiting for input — '{summary}'. Please respond." and does NOT send any answer
- [x] CHK-011 No nudge budget: No cooldown or retry limit; each question evaluated independently
- [x] CHK-012 Bottom-most indicator: When multiple question indicators in capture, operator evaluates the most recent (bottom-most)

## Scenario Coverage
- [x] CHK-013 Agent idle with question → confidence assessment triggered
- [x] CHK-014 Agent idle without question → normal idle (operator2 stuck detection)
- [x] CHK-015 Agent active → terminal heuristic NOT run
- [x] CHK-016 Idle agent with question not flagged as stuck (even past threshold)
- [x] CHK-017 Idle agent without question → stuck detection applies normally
- [x] CHK-018 Standard operator2 operations work identically in operator3

## Edge Cases & Error Handling
- [x] CHK-019 Input-waiting detection runs before stuck detection in tick order
- [x] CHK-020 Multiple sequential auto-answers: each evaluated independently with no cooldown

## Code Quality
- [x] CHK-021 Pattern consistency: Skill file follows naming and structural patterns of operator1/operator2
- [x] CHK-022 No unnecessary duplication: Operator3 references operator2 rather than duplicating its content

## Documentation Accuracy
- [x] CHK-023 Memory file (`execution-skills.md`) updated with operator3 section and changelog entry
- [x] CHK-024 Spec file (`SPEC-fab-operator3.md`) accurately reflects the skill's behavior

## Cross References
- [x] CHK-025 Skill file references operator2 skill path correctly
- [x] CHK-026 Launcher script command matches skill name (`/fab-operator3`)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
