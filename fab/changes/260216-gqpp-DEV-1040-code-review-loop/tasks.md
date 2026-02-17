# Tasks: DEV-1040 Code Review Loop

**Change**: 260216-gqpp-DEV-1040-code-review-loop
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill Files

<!-- Update the three skill files that define review behavior. Sequential to ensure
     consistent language across the sub-agent dispatch pattern. fab-continue.md is
     the authoritative Review Behavior definition; fab-ff.md and fab-fff.md reference it. -->

- [x] T001 Update Review Behavior in `fab/.kit/skills/fab-continue.md`: Replace inline review with sub-agent dispatch. Add structured output with three-tier priority scheme (must-fix / should-fix / nice-to-have). Update the On Failure section to present findings with priority annotations. Preserve the existing manual rework options (fix code, revise tasks, revise spec). Add note that sub-agent is non-prescriptive — LLM uses whatever review agent is available.
- [x] T002 Update Review Behavior in `fab/.kit/skills/fab-fff.md`: Update Step 7 (Review) to dispatch review via sub-agent instead of inline. Update autonomous rework to use sub-agent's prioritized findings with comment triage. Each rework cycle re-spawns a fresh sub-agent for re-review. Preserve existing 3-cycle retry cap and escalation rules. Update the description line to mention sub-agent review.
- [x] T003 Update Review Behavior in `fab/.kit/skills/fab-ff.md`: Major rewrite of Step 5 (Review). Replace fully-interactive rework with auto-loop (3 cycles, sub-agent review, prioritized findings, comment triage) + interactive fallback on cap exhaustion. Add escalation rule (same as fab-fff: escalate after 2 consecutive fix-code). Update description line and error handling table. After fallback to interactive, no further retry cap (user is in the loop).

## Phase 2: Documentation — Specs

<!-- Independent spec files, can be done in parallel. -->

- [x] T004 [P] Update `docs/specs/skills.md`: Update Review Behavior section to describe sub-agent dispatch, structured output with priority tiers, and manual rework with priority annotations. Update `/fab-ff` section description and Step 5 to reflect auto-loop with interactive fallback. Update `/fab-fff` section to mention sub-agent in autonomous rework loop.
- [x] T005 [P] Update `docs/specs/overview.md`: Update stage table review row to mention sub-agent validation. Update Quick Reference entries for `/fab-ff` (add "auto-loop with interactive fallback") and `/fab-fff` (mention sub-agent review).
- [x] T006 [P] Update `docs/specs/user-flow.md`: Update Diagram 2 (The Same Flow, With Fab) to show review→apply loop arrow. Update Diagram 3B (Change Flow) to annotate review with sub-agent and auto-loop. Update Diagram 4 (State Diagram) to add review→apply loop transition for auto-rework.

## Phase 3: Documentation — README

- [x] T007 Update `README.md`: Update stage table review row (line 44). Update review description paragraph (line 49) to mention sub-agent and prioritized comments. Update "Code Quality as a Guardrail" section: rework table and `/fab-fff`/`/fab-ff` descriptions to reflect sub-agent review with prioritized findings.

---

## Execution Order

- T001 → T002 → T003 (sequential for consistency across shared sub-agent dispatch pattern)
- T004, T005, T006 are independent (parallel)
- T004-T006 depend on T001-T003 (specs must reflect final skill file wording)
- T007 depends on T004-T006 (README summary should align with detailed specs)
