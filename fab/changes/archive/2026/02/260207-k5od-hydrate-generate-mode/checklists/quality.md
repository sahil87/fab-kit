# Quality Checklist: Add "generate" mode to fab-hydrate

**Change**: 260207-k5od-hydrate-generate-mode
**Generated**: 2026-02-07
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Unified Argument Routing: Skill determines mode from argument type — URLs and `.md` files → ingest, folders → generate, no args → generate
- [x] CHK-002 No-Args Replaces Usage Error: Invoking `/fab:hydrate` with no arguments enters generate mode, not an error
- [x] CHK-003 Codebase Gap Detection: Generate mode scans for undocumented Modules, APIs, Patterns, Configuration, and Conventions
- [x] CHK-004 Scan Scope: Folder arguments limit scan to those paths; no args scans project root; ignore patterns applied
- [x] CHK-005 Gap Report Presentation: Gap report is categorized, prioritized, and includes name/location/priority for each gap
- [x] CHK-006 Interactive Selection: AskUserQuestion offers "All", "All High priority", "Select by number", "Select by category" strategies
- [x] CHK-007 Structured Doc Output: Generated docs follow centralized format (Overview, Requirements, Design Decisions, Changelog)
- [x] CHK-008 Index Maintenance: Generate mode creates/updates domain indexes and top-level index with relative links
- [x] CHK-009 Idempotent Generation: Re-runs merge into existing docs, preserve manual edits, show new gaps only
- [x] CHK-010 Ingest Behavior Unchanged: URL and `.md` file ingest works identically to before

## Behavioral Correctness

- [x] CHK-011 No-args behavior change: Old "abort with usage message" is fully replaced — no trace of old behavior in skill
- [x] CHK-012 Mixed-mode rejection: Passing both URLs and folders in one invocation produces clear error and no processing

## Scenario Coverage

- [x] CHK-013 No arguments triggers generate mode: Verify `/fab:hydrate` with no args enters generate scanning from project root
- [x] CHK-014 Folder argument triggers generate mode: Verify `/fab:hydrate ./src/` scans only that folder
- [x] CHK-015 URL argument triggers ingest mode: Verify URL still routes to ingest pipeline
- [x] CHK-016 Markdown file triggers ingest mode: Verify `.md` path still routes to ingest pipeline
- [x] CHK-017 Mixed-mode arguments rejected: Verify URL + folder produces error message
- [x] CHK-018 Small number of gaps skips selection: Verify 1-3 gaps skip AskUserQuestion, just confirm
- [x] CHK-019 User selects subset of gaps: Verify only selected gaps are documented
- [x] CHK-020 Ambiguous behavior marked: Verify `[INFERRED]` markers appear on uncertain requirements
- [x] CHK-021 Re-run after manual edits: Verify manually-added content is preserved on re-generation

## Edge Cases & Error Handling

- [x] CHK-022 Zero gaps found: Generate mode reports "No documentation gaps found" and exits cleanly
- [x] CHK-023 Folder path doesn't exist: Skill reports error and aborts
- [x] CHK-024 **N/A**: Spec does not mandate a cap on gap report size; grouping by category and priority sorting provides navigability
- [x] CHK-025 Scan respects ignore patterns: `.git/`, `node_modules/`, etc. are excluded

## Documentation Accuracy

- [x] CHK-026 Skill description frontmatter reflects dual-mode purpose
- [x] CHK-027 Output examples cover both ingest and generate modes
- [x] CHK-028 Error handling table is complete for all new error conditions

## Cross References

- [x] CHK-029 Next line at bottom of skill references generate mode usage
- [x] CHK-030 Existing ingest mode sections are preserved and unmodified (except argument routing)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab:archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
