# Tasks: PR Review Reply Comments

**Change**: 260317-kwsh-pr-review-replies
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Expand `--jq` projections in `fab/.kit/skills/git-pr-review.md` Step 3 Path A and Path B to include `id` and `node_id` fields

## Phase 2: Core Implementation

- [x] T002 Update Step 4 triage in `fab/.kit/skills/git-pr-review.md` to assign disposition intents (`fix`, `defer`, `skip`) to each non-informational comment and expand the triage summary line
- [x] T003 Add Step 5.5 (Post Replies) to `fab/.kit/skills/git-pr-review.md` — post reply comments via REST API after push, with best-effort error handling and summary output
- [x] T004 Add reply deduplication logic to Step 5.5 in `fab/.kit/skills/git-pr-review.md` — check existing replies for disposition prefixes before posting

## Phase 3: Integration & Edge Cases

- [x] T005 Update Step 5 flow in `fab/.kit/skills/git-pr-review.md` to proceed to Step 5.5 when no code changes (all defer/skip), instead of stopping at "No changes needed."
- [x] T006 Add `replying` phase to Phase Sub-State Tracking table in `fab/.kit/skills/git-pr-review.md` and document when it is set

## Phase 4: Polish

- [x] T007 [P] Add `## Disposition Reference` table at the bottom of `fab/.kit/skills/git-pr-review.md` after `## Rules`
- [x] T008 [P] Update `docs/specs/skills/SPEC-git-pr-review.md` to reflect new Step 5.5, disposition taxonomy, reply deduplication, and `replying` phase

---

## Execution Order

- T001 blocks T002 (dispositions need IDs from extended projections)
- T002 blocks T003 (replies need dispositions from triage)
- T003 blocks T004 (deduplication is part of Step 5.5)
- T003 blocks T005 (no-code-changes flow routes to Step 5.5)
- T007 and T008 are independent, can run alongside each other
