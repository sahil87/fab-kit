# Quality Checklist: PR Review Reply Comments

**Change**: 260317-kwsh-pr-review-replies
**Generated**: 2026-03-17
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Three-Disposition Classification: Each non-informational comment is classified with intent `fix`, `defer`, or `skip`
- [x] CHK-002 Disposition Reply Format: Reply text matches spec format (`Fixed — {desc}. ({sha})`, `Deferred — {reason}.`, `Skipped — {reason}.`)
- [x] CHK-003 Comment ID Capture: Path A and Path B projections include `id` and `node_id`
- [x] CHK-004 Reply Posting: Step 5.5 posts replies via `gh api` REST with `in_reply_to` field
- [x] CHK-005 Reply Deduplication: Existing disposition replies prevent re-posting
- [x] CHK-006 Reply When No Code Changes: defer/skip-only runs still post replies
- [x] CHK-007 Best-Effort Replies: Failed reply POST logs error and continues to next comment
- [x] CHK-008 Reply Summary Output: Prints `Replied to {N} comment(s): {F} fix, {D} defer, {S} skip`
- [x] CHK-009 Expanded Triage Summary: Uses `{F} fix, {D} defer, {S} skip, {I} informational (no reply)` format
- [x] CHK-010 Replying Phase: Phase sub-state `replying` set before posting replies
- [x] CHK-011 Disposition Reference Table: `## Disposition Reference` section exists after `## Rules` in skill file

## Behavioral Correctness

- [x] CHK-012 Informational comments still get no reply (unchanged behavior)
- [x] CHK-013 Existing triage logic (actionable vs informational) unchanged — dispositions layer on top

## Scenario Coverage

- [x] CHK-014 Fix reply includes correct short SHA and description
- [x] CHK-015 Re-run with previous replies skips already-replied comments
- [x] CHK-016 Mixed re-run: skips replied, posts to new comments
- [x] CHK-017 Phase transitions: `replying` set when Step 5.5 begins (with or without prior push)

## Edge Cases & Error Handling

- [x] CHK-018 Reply POST failure for one comment does not abort remaining replies
- [x] CHK-019 All comments informational: no replies posted, no Step 5.5 execution
- [x] CHK-020 Phase `replying` skips `pushed` when no code changes occurred

## Code Quality

- [x] CHK-021 Pattern consistency: New skill sections follow naming and structural patterns of existing steps
- [x] CHK-022 No unnecessary duplication: Reply logic reuses existing comment data from Step 3 fetch

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
