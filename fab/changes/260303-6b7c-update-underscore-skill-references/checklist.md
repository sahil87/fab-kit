# Quality Checklist: Update Underscore Skill References

**Change**: 260303-6b7c-update-underscore-skill-references
**Generated**: 2026-03-04
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Pattern A top-of-file: All 11 standard skill files use `fab/.kit/skills/_preamble.md` (no `./`) in line 8
- [x] CHK-002 fab-switch variant: `fab-switch.md` line 8 uses `fab/.kit/skills/_preamble.md` (no `./`)
- [x] CHK-003 _preamble self-reference: `_preamble.md` line 12 example uses `fab/.kit/skills/_preamble.md` (no `./`)
- [x] CHK-004 _preamble _scripts.md ref: `_preamble.md` §1 Always Load `_scripts.md` reference uses no `./` prefix
- [x] CHK-005 internal-skill-optimize: Line 21 `_preamble.md` reference uses no `./` prefix
- [x] CHK-006 Test updated: Stale "skips _preamble.md" test replaced with deployment assertions
- [x] CHK-007 Memory files updated: Full-path refs in context-loading, kit-architecture, planning-skills, execution-skills use no `./` prefix

## Behavioral Correctness

- [x] CHK-008 Pattern B unchanged: Inline shorthand references (`` `_preamble.md` §2 ``, `` (`_generation.md`) ``) in skill files are NOT modified
- [x] CHK-009 Spec files unchanged: `glossary.md` and `skills.md` bare shorthand references are untouched

## Scenario Coverage

- [x] CHK-010 Agent path resolution: Top-of-file instruction resolves correctly from repo root CWD
- [x] CHK-011 Test passes: Updated sync-workspace test assertions pass (underscore files ARE deployed)

## Edge Cases & Error Handling

- [x] CHK-012 No archive changes: Files in `fab/changes/archive/` are not modified

## Code Quality

- [x] CHK-013 Pattern consistency: All updated references follow the same `fab/.kit/skills/_*.md` pattern
- [x] CHK-014 No unnecessary duplication: No new redundant references introduced

## Documentation Accuracy

- [x] CHK-015 kit-architecture: Underscore file deployment is documented in the architecture memory file

## Cross References

- [x] CHK-016 Memory-to-source consistency: Memory file references match the actual paths in skill files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
