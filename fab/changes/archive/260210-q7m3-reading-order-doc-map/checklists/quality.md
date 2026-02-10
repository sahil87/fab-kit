# Quality Checklist: Add Reading Order Guide and Documentation Map

**Change**: 260210-q7m3-reading-order-doc-map
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Audience reading paths: README contains three distinct reading paths (New User, Contributor, Spec Reader) each with ordered document list and descriptions
- [x] CHK-002 Grouped inventory: README contains document inventory grouped into four categories (Getting Started, Concepts, Reference, Internals)
- [x] CHK-003 Glossary linking: README, fab/specs/index.md, and fab/docs/index.md each contain at least one prominent link to fab/specs/glossary.md
- [x] CHK-004 Index orientation notes: both fab/specs/index.md and fab/docs/index.md contain a newcomer orientation note pointing to README Documentation Map

## Behavioral Correctness

- [x] CHK-005 References exclusion: document inventory does NOT include files from references/speckit/ or references/openspec/
- [x] CHK-006 Existing content preserved: README "What is Fab Kit?", "Get Started", "Repository Structure", and "References" sections remain intact

## Scenario Coverage

- [x] CHK-007 New user path: following the New User reading path leads through a logical progression from overview to first-change workflow
- [x] CHK-008 Contributor path: following the Contributor path progresses from concepts to internals with clear prerequisite ordering
- [x] CHK-009 Spec reader path: following the Spec Reader path covers specs directory in logical order with glossary as prerequisite

## Edge Cases & Error Handling

- [x] CHK-010 All markdown links resolve: every link in README.md, fab/specs/index.md, and fab/docs/index.md points to an existing file

## Documentation Accuracy

- [x] CHK-011 Document descriptions match actual content: each document entry's brief description accurately reflects the file's contents
- [x] CHK-012 CommonMark compliance: all modified files use standard CommonMark syntax (constitution requirement)

## Cross References

- [x] CHK-013 Glossary link consistency: glossary links in all three files point to the same path (fab/specs/glossary.md)
- [x] CHK-014 Bidirectional navigation: index files point to README, README points back to index files within the inventory

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
