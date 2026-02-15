# Quality Checklist: Rename "brief" to "intake" + Add Intake Generation Rule

**Change**: 260215-v4n7-DEV-1025-rename-brief-to-intake
**Generated**: 2026-02-15
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Stage Identifier Rename: All pipeline references use `intake` instead of `brief` — zero `\bbrief\b` stage/artifact matches remain in `fab/.kit/skills/`, `fab/.kit/schemas/`, `fab/.kit/templates/`
- [x] CHK-002 Config Updated: `fab/config.yaml` defines `id: intake`, `generates: intake.md`, `requires: [intake]`
- [x] CHK-003 Schema Updated: `fab/.kit/schemas/workflow.yaml` lists `intake` in stage enum, `brief` absent
- [x] CHK-004 Template Renamed: `fab/.kit/templates/intake.md` exists, `fab/.kit/templates/brief.md` does not
- [x] CHK-005 Intake Generation Procedure: `_generation.md` contains `## Intake Generation Procedure` with generation rule, template loading, section guidance
- [x] CHK-006 fab-new References Procedure: `/fab-new` Step 5 references "Intake Generation Procedure (`_generation.md`)" instead of inlining
- [x] CHK-007 Template Strengthened: `intake.md` template comments include explicit detail expectations for What Changes, Origin, Why sections

## Behavioral Correctness

- [x] CHK-008 Stage dispatch: `fab-continue.md` dispatches `intake` stage to spec generation (same behavior as former `brief`)
- [x] CHK-009 Transitions: All `lib/stageman.sh transition` calls reference `intake` (not `brief`) in skill files
- [x] CHK-010 Memory lookup: `_context.md` §3 references "intake's Affected Memory section"

## Scenario Coverage

- [x] CHK-011 New change initialization: `.status.yaml` template initializes with `intake: active`
- [x] CHK-012 Stage display: `fab-switch.md` shows `intake=1` in stage number mapping and `intake (active)` in suggested next table
- [x] CHK-013 Clarify scan: `fab-clarify.md` references scanning `intake.md` for taxonomy
- [x] CHK-014 Spec/Tasks procedures: `_generation.md` Spec and Tasks procedures reference `intake.md` (not `brief.md`)

## Edge Cases & Error Handling

- [x] CHK-015 English adjective preserved: The word "brief" used as an English adjective (e.g., "brief reason", "brief description") is NOT replaced in any file — verified 9 remaining uses across skills, 1 in spec template, 2 in docs, all English adjective
- [x] CHK-016 Historical artifacts: Existing `brief.md` files in `fab/changes/` are untouched — verified `260212-1aag-DEV-1018-build-stageman2/brief.md` exists
- [x] CHK-017 No stale references: Full-text search for `\bbrief\b` in fab/.kit/ returns only English adjective uses

## Code Quality

- [x] CHK-018 Pattern consistency: Rename substitutions follow the same context-sensitive patterns throughout (stage name, artifact name, YAML key, prose)
- [x] CHK-019 No unnecessary duplication: Intake Generation Procedure is defined once in `_generation.md`, referenced by `fab-new.md`

## Documentation Accuracy

- [x] CHK-020 Specs updated: All 8 files in `docs/specs/` have `brief` → `intake` replacements applied (preserving English adjective)
- [x] CHK-021 Memory updated: All 10 files in `docs/memory/fab-workflow/` have `brief` → `intake` replacements applied (preserving English adjective) + 1 bonus file (model-tiers.md)

## Cross References

- [x] CHK-022 Internal links: All cross-references to `brief.md` (in templates, context loading, generation procedures) now point to `intake.md`
- [x] CHK-023 **N/A**: Changelog entries are added during hydrate, not apply — memory files were updated for content (brief→intake) but changelog rows are part of hydrate behavior

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
