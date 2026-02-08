# Tasks: Fix broken template links in fab-hydrate

**Change**: 260208-4wg3-fix-hydrate-links
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Implementation

- [x] T001 [P] Update Domain Index format link on line 97 of `fab/.kit/skills/fab-hydrate.md`: change `../../doc/fab-spec/TEMPLATES.md#domain-index-fabdocsdomainindexmd` to `../../specs/templates.md#domain-index-fabdocsdomainindexmd`
- [x] T002 [P] Update Centralized Doc format link on line 104 of `fab/.kit/skills/fab-hydrate.md`: change `../../doc/fab-spec/TEMPLATES.md#individual-doc-fabdomainnamemd` to `../../specs/templates.md#individual-doc-fabdomainnamemd`
- [x] T003 [P] Update Top-Level Index format link on line 124 of `fab/.kit/skills/fab-hydrate.md`: change `../../doc/fab-spec/TEMPLATES.md#top-level-index-fabdocsindexmd` to `../../specs/templates.md#top-level-index-fabdocsindexmd`

## Phase 2: Verification

- [x] T004 Verify no remaining references to `doc/fab-spec/TEMPLATES.md` exist in `fab/.kit/skills/fab-hydrate.md`

---

## Execution Order

- T001, T002, T003 are independent (`[P]`) — can be applied in any order
- T004 depends on T001-T003 completing
