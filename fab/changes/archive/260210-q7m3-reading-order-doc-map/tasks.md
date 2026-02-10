# Tasks: Add Reading Order Guide and Documentation Map

**Change**: 260210-q7m3-reading-order-doc-map
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Content

- [x] T001 Rewrite `README.md` Documentation section: replace the existing "Documentation" and "References" sections with a full Documentation Map containing three audience-specific reading paths (New User, Contributor, Spec Reader), a grouped document inventory (Getting Started, Concepts, Reference, Internals), and prominent glossary link. Preserve the existing "What is Fab Kit?", "Get Started", and "Repository Structure" sections. Keep the References section unchanged at the bottom.

## Phase 2: Cross-References

- [x] T002 [P] Add orientation note to `fab/specs/index.md`: insert a one-line callout after the existing blockquote directing newcomers to the README Documentation Map and glossary.
- [x] T003 [P] Add orientation note to `fab/docs/index.md`: insert a one-line callout after the existing blockquote directing newcomers to the README Documentation Map and glossary.

## Phase 3: Verification

- [x] T004 Verify all links in modified files resolve correctly: check that every markdown link in `README.md`, `fab/specs/index.md`, and `fab/docs/index.md` points to an existing file.

---

## Execution Order

- T001 must complete before T004 (T004 verifies T001's links)
- T002 and T003 are independent of each other and of T001
- T004 depends on T001, T002, T003 all being complete
