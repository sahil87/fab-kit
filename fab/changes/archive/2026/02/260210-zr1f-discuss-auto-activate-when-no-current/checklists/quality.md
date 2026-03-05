# Quality Checklist: Auto-activate after /fab-discuss when no current change

**Change**: 260210-zr1f-discuss-auto-activate-when-no-current
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Conditional activation logic: Skill checks `fab/current` after summary, prompts when empty, calls `/fab-switch` on accept
- [x] CHK-002 Decline path: When user declines, `fab/current` is untouched and Next line shows `/fab-switch`
- [x] CHK-003 Existing active change: No activation prompt shown when `fab/current` already points to a different change
- [x] CHK-004 Refine mode: No activation prompt shown (change already active)

## Behavioral Correctness

- [x] CHK-005 Key Properties table reflects conditional activation (not absolute "No")
- [x] CHK-006 Key Differences table reflects conditional activation and git integration
- [x] CHK-007 Output examples cover both accepted and declined cases
- [x] CHK-008 Next Steps Reference covers activated case

## Scenario Coverage

- [x] CHK-009 New change, no active, user accepts: verified in skill behavior and output section
- [x] CHK-010 New change, no active, user declines: verified in skill behavior and output section
- [x] CHK-011 New change, existing active: no prompt, standard Next line
- [x] CHK-012 /fab-switch failure: error reported, manual switch suggested

## Documentation Accuracy

- [x] CHK-013 `change-lifecycle.md` updated: conditional description replaces "Not written" bullet, changelog entry added
- [x] CHK-014 `planning-skills.md` updated: output, key differences, and changelog entry added
- [x] CHK-015 `_context.md` Next Steps table: new row for activated case

## Cross References

- [x] CHK-016 Backlog item [s3d6] marked done
- [x] CHK-017 No contradictions between skill file, centralized docs, and _context.md regarding fab-discuss activation behavior

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
