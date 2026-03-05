# Quality Checklist: Smart Copilot Review Detection

**Change**: 260303-n30u-smart-copilot-review-detection
**Generated**: 2026-03-03
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 3-Phase Detection: git-pr-fix Step 2 implements Phase 1 (check existing reviews), Phase 2 (POST request), Phase 3 (mode-specific poll)
- [x] CHK-002 Mode-Specific Behavior: Wait mode polls (30s, 12 attempts); standalone mode does single check after Phase 2
- [x] CHK-003 Login Name Documentation: Inline comment in git-pr-fix Step 2 documents the Copilot bot login name discrepancy across API endpoints

## Behavioral Correctness
- [x] CHK-004 git-pr Step 6: References git-pr-fix Step 2 behavior with wait mode enabled, no longer describes its own polling loop
- [x] CHK-005 git-pr-fix standalone: Prints "Copilot review requested but not yet available — re-run later." when Phase 2 succeeds but Phase 3 single-check finds nothing

## Removal Verification
- [x] CHK-006 Blind polling removed: No residual unconditional polling logic in git-pr Step 6 or git-pr-fix Step 2
- [x] CHK-007 Old standalone bail message: "No Copilot review found — skipping." replaced with appropriate messages per phase outcome

## Scenario Coverage
- [x] CHK-008 Already reviewed: Phase 1 finds existing review → skips to triage
- [x] CHK-009 Copilot available: Phase 2 POST succeeds → Phase 3 polls
- [x] CHK-010 Copilot not available: Phase 2 POST fails → prints "not available" → exits
- [x] CHK-011 Network/API error: Same behavior as "not available" — no polling
- [x] CHK-012 Standalone slow review: Phase 2 succeeds, Phase 3 single-check finds nothing → prints re-run message

## Edge Cases & Error Handling
- [x] CHK-013 Best-effort contract: Any Phase 2 or Phase 3 failure in git-pr Step 6 does NOT affect the "Shipped." output or exit status
- [x] CHK-014 Phase 1 API error: Non-zero exit from GET /reviews on first check → prints error and stops (existing behavior preserved)

## Code Quality
- [x] CHK-015 Pattern consistency: Skill markdown follows existing formatting and section structure of git-pr.md and git-pr-fix.md
- [x] CHK-016 No unnecessary duplication: 3-phase logic described once in git-pr-fix Step 2, referenced from git-pr Step 6

## Documentation Accuracy
- [x] CHK-017 Installed copies match: .claude/skills/git-pr/SKILL.md matches fab/.kit/skills/git-pr.md
- [x] CHK-018 Installed copies match: .claude/skills/git-pr-fix/SKILL.md matches fab/.kit/skills/git-pr-fix.md

## Cross References
- [x] CHK-019 git-pr Rules section: Updated if needed to reflect new detection behavior
- [x] CHK-020 git-pr-fix Rules section: No changes needed (rules are already correct for the new behavior)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
