# Quality Checklist: Separate Doc Hydration from Init, Add Smart Context Loading, and Index fab/docs

**Change**: 260207-q7m3-separate-hydrate-smart-context
**Generated**: 2026-02-07
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Standalone Hydrate Skill: `fab/.kit/skills/fab-hydrate.md` exists with complete hydration logic (fetch, analyze, map, create/merge, index update)
- [x] CHK-002 Hydrate Skill File: `fab-hydrate.md` has correct frontmatter and references `_context.md`; matches the `fab-*.md` glob for auto-discovery
- [x] CHK-003 Idempotent Hydration: Skill instructions specify merge-not-overwrite behavior for re-hydrating same source
- [x] CHK-004 Init Structural Bootstrap Only: `fab-init.md` contains no Phase 2 hydration logic and no `[sources...]` arguments
- [x] CHK-005 Init Output Updated: `fab-init.md` has no "With Sources" output section; first-run output suggests `/fab:hydrate`
- [x] CHK-006 Always Load Docs Index: `_context.md` "Always Load" section includes `fab/docs/index.md` as a third item
- [x] CHK-007 Selective Domain Loading: `_context.md` "Centralized Doc Lookup" applies to all skills on an active change, not just spec-writing
- [x] CHK-008 Context Loading in _context.md: `/fab:hydrate` is listed in the exceptions (alongside init, switch, status)
- [x] CHK-009 Top-Level Index Maintenance: Both hydrate and archive skills instruct the agent to update `fab/docs/index.md` when domains or docs are added
- [x] CHK-010 Domain Index Maintenance: Both hydrate and archive skills instruct the agent to create/update `fab/docs/{domain}/index.md`
- [x] CHK-011 Index Format Consistency: Both skills reference relative links and the format defined in `TEMPLATES.md`
- [x] CHK-012 Spec Docs — SKILLS.md: Contains `/fab:hydrate` section; `/fab:init` section has no source hydration
- [x] CHK-013 Spec Docs — README.md: Quick Reference has `/fab:hydrate` row; `/fab:init` row updated; "Hydrating Docs" section uses `/fab:hydrate`
- [x] CHK-014 Spec Docs — ARCHITECTURE.md: Bootstrap sequence includes `/fab:hydrate` as step 4; no hydration in init description

## Behavioral Correctness

- [x] CHK-015 Init With Arguments Rejected: `fab-init.md` outputs redirect message when arguments are passed ("Did you mean /fab:hydrate?")
- [x] CHK-016 Hydrate Without fab/docs/ Aborts: `fab-hydrate.md` pre-flight check verifies `fab/docs/` exists and aborts with guidance if not
- [x] CHK-017 Archive Updates Top-Level Index: `fab-archive.md` Step 3b item 4 and Step 3c item 6 include instructions to update `fab/docs/index.md` doc-list column

## Removal Verification

- [x] CHK-018 Init Source Hydration Removed: No trace of Phase 2 (source hydration), source types, fetch logic, or hydration output in `fab-init.md`
- [x] CHK-019 Init [sources...] Removed: Title, arguments, examples in `fab-init.md` do not mention `[sources...]`

## Scenario Coverage

- [x] CHK-020 Scenario — Hydrate from Notion URL: `fab-hydrate.md` Step 1 covers Notion URL fetching; Steps 2-5 cover domain mapping and index updates
- [x] CHK-021 Scenario — Hydrate from local directory: `fab-hydrate.md` Step 1 covers recursive local file reading
- [x] CHK-022 Scenario — Hydrate without fab/docs/: `fab-hydrate.md` pre-flight check aborts with "fab/docs/ not found. Run /fab:init first to create the docs directory."
- [x] CHK-023 Scenario — Multiple sources: `fab-hydrate.md` output section shows "Multiple Sources" example with 3 sources
- [x] CHK-024 Scenario — Init with no arguments: `fab-init.md` runs structural bootstrap only (Phase 1)
- [x] CHK-025 Scenario — Init with arguments rejected: `fab-init.md` pre-flight check outputs redirect to `/fab:hydrate`
- [x] CHK-026 Scenario — Skill loads docs index: `_context.md` "Always Load" includes `fab/docs/index.md` as third bullet
- [x] CHK-027 Scenario — Domain added during hydration: `fab-hydrate.md` Step 3 creates domain folder and domain index; Step 5 updates top-level index
- [x] CHK-028 Scenario — Domain added during archive: `fab-archive.md` Step 3b creates domain folder, domain index, and updates top-level index
- [x] CHK-029 Scenario — Symlink auto-discovery: `fab-hydrate.md` filename matches `fab-*.md` glob; `fab-setup.sh` line 56 confirms `fab-*.md` pattern
- [x] CHK-030 Scenario — Index uses relative links: Both hydrate (Step 4-5) and archive (Step 3b-3c) use relative paths in index links

## Edge Cases & Error Handling

- [x] CHK-031 Re-hydrate same source: `fab-hydrate.md` Idempotency Guarantee section and Step 3 item 4 specify merge behavior — new requirements added, existing updated, manual content preserved
- [x] CHK-032 No relevant docs exist yet: `_context.md` section 3 item 4 handles missing domain docs gracefully ("note this and proceed without error")

## Documentation Accuracy

- [x] CHK-033 All three fab-spec docs (README, SKILLS, ARCHITECTURE) consistently reference `/fab:hydrate` for doc ingestion
- [x] CHK-034 No stale references to `/fab:init [sources...]` in any skill file or fab-spec doc (verified via grep — zero matches in fab/.kit/skills/ and doc/fab-spec/)

## Cross References

- [x] CHK-035 `fab-hydrate.md` references `TEMPLATES.md` for centralized doc format and hydration rules (Step 3 items 2-3, Step 5 item 5)
- [x] CHK-036 SKILLS.md Context Loading Convention matches `_context.md` content (both describe same layers with same exceptions list and same doc lookup scope)
- [x] CHK-037 Next Steps table in `_context.md` and SKILLS.md are consistent (both include `/fab:init` → hydrate, `/fab:hydrate` → new/hydrate rows)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab:archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
