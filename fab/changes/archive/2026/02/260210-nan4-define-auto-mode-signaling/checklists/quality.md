# Quality Checklist: Define auto-mode signaling mechanism for skill-to-skill invocation

**Change**: 260210-nan4-define-auto-mode-signaling
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Skill Invocation Protocol section exists in `_context.md` with `[AUTO-MODE]` prefix definition
- [x] CHK-002 Protocol specifies exact prefix string, placement (first line), called-skill detection behavior, and absent-prefix default behavior
- [x] CHK-003 `fab-clarify.md` Mode Selection references the protocol and documents `[AUTO-MODE]` detection
- [x] CHK-004 `fab-ff.md` auto-clarify invocations use the `[AUTO-MODE]` prefix per the protocol
- [x] CHK-005 `fab-fff.md` audited — delegates to fab-ff (no standalone auto-clarify invocations)

## Behavioral Correctness

- [x] CHK-006 Protocol is backward-compatible — user-invoked `/fab-clarify` still enters suggest mode (no prefix = interactive)
- [x] CHK-007 Protocol is transitive — fab-fff → fab-ff → fab-clarify chain correctly propagates auto mode

## Scenario Coverage

- [x] CHK-008 Scenario: fab-ff invokes fab-clarify with `[AUTO-MODE]` prefix → auto mode activated
- [x] CHK-009 Scenario: User invokes fab-clarify without prefix → suggest mode activated
- [x] CHK-010 Scenario: fab-fff delegates to fab-ff which invokes fab-clarify → auto mode via transitivity

## Documentation Accuracy

- [x] CHK-011 `fab/docs/fab-workflow/planning-skills.md` references the skill invocation protocol
- [x] CHK-012 `fab/docs/fab-workflow/clarify.md` references `[AUTO-MODE]` prefix detection

## Cross References

- [x] CHK-013 All references between `_context.md`, `fab-clarify.md`, `fab-ff.md`, and `fab-fff.md` are consistent and point to the correct section names

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
