# Quality Checklist: Add Code Review Scaffold & 5 Cs of Quality

**Change**: 260218-xkkc-add-code-review-5cs-quality
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Scaffold exists: `fab/.kit/scaffold/code-review.md` created with all 5 sections (Severity Definitions, Review Scope, False Positive Policy, Rework Budget, Project-Specific Review Rules)
- [ ] CHK-002 Context loading: `_context.md` Always Load list contains 7 items including `fab/code-review.md`
- [ ] CHK-003 Setup bootstrap: `fab-setup.md` has step 1b4 for `fab/code-review.md` with correct if-missing/copy/report pattern
- [ ] CHK-004 Setup config menu: `fab-setup.md` includes item 10 (`code-review.md`) in the config editing menu
- [ ] CHK-005 Review context: `fab-continue.md` sub-agent context list includes `fab/code-review.md (if present)`
- [ ] CHK-006 Config scaffold: `fab/.kit/scaffold/config.yaml` companion files comment references `fab/code-review.md`
- [ ] CHK-007 README: "Code Quality as a Guardrail" section contains the 5 Cs table with all 5 entries

## Behavioral Correctness

- [ ] CHK-008 Optional file handling: `_context.md` marks `code-review.md` as optional with `*(optional — no error if missing)*`
- [ ] CHK-009 Fallback behavior: `fab-continue.md` review behavior specifies that hardcoded defaults apply when `code-review.md` is absent

## Scenario Coverage

- [ ] CHK-010 Scaffold populated: All 5 scaffold sections contain populated defaults (not empty headings)
- [ ] CHK-011 Severity defaults match: Scaffold severity definitions match the three-tier scheme in `fab-continue.md` (must-fix, should-fix, nice-to-have)
- [ ] CHK-012 Bootstrap output: `fab-setup.md` bootstrap output example includes "Created: fab/code-review.md"
- [ ] CHK-013 Config menu pattern: Item 10 follows the same editing pattern as items 8 (context.md) and 9 (code-quality.md)

## Edge Cases & Error Handling

- [ ] CHK-014 Idempotent bootstrap: Step 1b4 skips when `fab/code-review.md` already exists (same pattern as 1b2, 1b3)

## Code Quality

- [ ] CHK-015 Pattern consistency: Scaffold follows the same HTML comment + populated defaults pattern as `code-quality.md` scaffold
- [ ] CHK-016 No unnecessary duplication: Severity definitions in scaffold reference the skill's existing three-tier scheme without duplicating implementation details

## Documentation Accuracy

- [ ] CHK-017 5 Cs table accuracy: All 5 entries have correct file paths and meaningful "Question" column values
- [ ] CHK-018 Author-vs-critic distinction: README explains that code-quality.md guides writing (apply) while code-review.md guides validation (review)

## Cross References

- [ ] CHK-019 File reference consistency: All references to `fab/code-review.md` use consistent path format across modified files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
