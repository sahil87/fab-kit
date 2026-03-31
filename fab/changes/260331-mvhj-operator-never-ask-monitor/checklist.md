# Quality Checklist: Operator Never-Ask Monitor Fix

**Change**: 260331-mvhj-operator-never-ask-monitor
**Generated**: 2026-03-31
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Spawn sequence auto-enroll prohibition (operator7): blockquote admonition present after spawn steps in `fab-operator7.md`
- [x] CHK-002 Spawn sequence auto-enroll prohibition (operator7): step 4 annotated with unconditional/silent language
- [x] CHK-003 Spawn section auto-enroll prohibition (operator6): blockquote admonition present in spawn subsection of `fab-operator6.md`
- [x] CHK-004 Spawn section auto-enroll prohibition (operator6): §1 "Automate the routine" includes "Never ask whether to monitor" sentence

## Behavioral Correctness
- [x] CHK-005 Principle-procedure consistency: never-ask language appears in both §1 and §6 of `fab-operator7.md`
- [x] CHK-006 Principle-procedure consistency: never-ask language appears in both §1 and spawn subsection of `fab-operator6.md`

## Scenario Coverage
- [x] CHK-007 Operator spawns agent for existing change: spawn sequence text makes auto-enroll unambiguous
- [x] CHK-008 Operator spawns agent for new backlog item: spawn sequence text makes auto-enroll unambiguous
- [x] CHK-009 LLM reads spawn sequence in isolation: local prohibition text is self-contained

## Code Quality
- [x] CHK-010 Pattern consistency: RFC 2119 keywords (MUST NOT) used consistently with rest of skill files
- [x] CHK-011 No unnecessary duplication: admonition text is concise, not redundantly restating the full principle

## Documentation Accuracy
- [x] CHK-012 Memory file `execution-skills.md` updated to document the never-ask-monitor reinforcement

## Cross References
- [x] CHK-013 Operator6 and operator7 use matching language for the never-ask prohibition

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
