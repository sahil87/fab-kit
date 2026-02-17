# Quality Checklist: DEV-1040 Code Review Loop

**Change**: 260216-gqpp-DEV-1040-code-review-loop
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [ ] CHK-001 Review via Sub-Agent: All three skills (fab-continue, fab-ff, fab-fff) dispatch review to a sub-agent, not inline
- [ ] CHK-002 Sub-Agent Non-Prescriptive: No skill file hardcodes a specific agent name or tool for review
- [ ] CHK-003 Structured Review Output: Review sub-agent produces findings with severity tier, description, and file:line reference
- [ ] CHK-004 Three-Tier Priority: Findings classified as must-fix, should-fix, or nice-to-have with correct semantics
- [ ] CHK-005 Comment Triage: Applying agent triages by priority; must-fix always addressed, nice-to-have may be skipped
- [ ] CHK-006 fab-continue Manual Rework: Sub-agent review with manual rework options preserved (fix code, revise tasks, revise spec)
- [ ] CHK-007 fab-fff Auto-Loop: Autonomous rework uses sub-agent findings with comment triage, 3-cycle cap, escalation rules
- [ ] CHK-008 fab-ff Auto-Loop with Fallback: Auto-loop (3 cycles) + interactive fallback on cap exhaustion
- [ ] CHK-009 fab-ff Escalation Rule: Escalate after 2 consecutive fix-code (same as fab-fff)
- [ ] CHK-010 Retry Cap Alignment: Both fab-ff and fab-fff use 3-cycle cap

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [ ] CHK-011 fab-ff behavioral change: No longer fully interactive on first failure — auto-loops first, then falls back to interactive
- [ ] CHK-012 fab-fff review dispatch: Inline review replaced by sub-agent, autonomous rework heuristics preserved
- [ ] CHK-013 fab-continue review dispatch: Inline review replaced by sub-agent, manual rework flow unchanged

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [ ] CHK-014 Scenario: fab-continue review failure presents findings with priority annotations
- [ ] CHK-015 Scenario: fab-fff auto-rework spawns fresh sub-agent for re-review
- [ ] CHK-016 Scenario: fab-fff retry cap exhaustion bails with cycle summary
- [ ] CHK-017 Scenario: fab-fff escalation after 2 consecutive fix-code
- [ ] CHK-018 Scenario: fab-ff auto-rework on first failure
- [ ] CHK-019 Scenario: fab-ff fallback to interactive after 3 auto-rework cycles
- [ ] CHK-020 Scenario: Nice-to-have-only findings may count as pass

## Edge Cases & Error Handling
- [ ] CHK-021 fab-ff interactive fallback has no further retry cap (user is in the loop)
- [ ] CHK-022 Each rework cycle spawns a NEW sub-agent instance (fresh context)

## Code Quality
- [ ] CHK-023 Pattern consistency: Sub-agent dispatch language consistent across all three skill files
- [ ] CHK-024 No unnecessary duplication: Shared review behavior defined once in fab-continue.md, referenced by fab-ff.md and fab-fff.md

## Documentation Accuracy
- [ ] CHK-025 README review description matches skill file behavior
- [ ] CHK-026 docs/specs/skills.md Review Behavior section matches fab-continue.md
- [ ] CHK-027 docs/specs/skills.md /fab-ff section reflects auto-loop with interactive fallback
- [ ] CHK-028 docs/specs/skills.md /fab-fff section reflects sub-agent in autonomous rework
- [ ] CHK-029 docs/specs/overview.md stage table and quick reference updated
- [ ] CHK-030 docs/specs/user-flow.md diagrams show apply-review loop

## Cross References
- [ ] CHK-031 fab-ff.md and fab-fff.md both reference "review behavior per /fab-continue" consistently
- [ ] CHK-032 Model tier implication: Review sub-agent described as capable-tier work (deep reasoning, code analysis)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
