# Tasks: Add `fab/specs/` Index and Clarify Specs vs Docs Distinction

**Change**: 260207-bb1q-add-specs-index
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Implementation

- [x] T001 [P] Create `fab/specs/index.md` with boilerplate that distinguishes specs (pre-implementation, conceptual, human-curated) from docs (post-implementation, authoritative truth). Include empty table for spec entries. Reference `fab/docs/index.md` as the complementary layer.
- [x] T002 [P] Update `fab/docs/index.md` — replace the single-line description with a header block that clearly states docs are post-implementation artifacts (what happened, source of truth for behavior). Reference `fab/specs/index.md` as the complementary pre-implementation layer. Preserve existing table content.
- [x] T003 [P] Update `fab/.kit/skills/fab-init.md` — add step 1d for creating `fab/specs/index.md` (idempotent, skip if exists). Re-letter existing step 1d (changes/) to 1e, and step 1e (symlinks) to 1f, and step 1f (.gitignore) to 1g. Update the re-run output to include specs/index.md check.
- [x] T004 [P] Update `fab/.kit/skills/_context.md` — add `fab/specs/index.md` as a 4th file in the "Always Load" context layer list (Section 1).

## Phase 2: Consistency

- [x] T005 Verify all cross-references are consistent — `fab-init.md` step numbering, `_context.md` list count, and both index.md files reference each other correctly.

---

## Execution Order

- T001–T004 are independent and can run in parallel
- T005 depends on T001–T004 all being complete
