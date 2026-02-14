# Quality Checklist: Relocate memory and specs to docs/

**Change**: 260214-m3v8-relocate-docs-dev-scripts
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 docs/ as canonical root: `docs/memory/` and `docs/specs/` exist with all content preserved from `fab/memory/` and `fab/specs/`
- [x] CHK-002 Always-load layer: `_context.md` references `docs/memory/index.md` and `docs/specs/index.md`
- [x] CHK-003 Selective domain loading: `_context.md` memory file lookup uses `docs/memory/{domain}/` paths
- [x] CHK-004 Init scaffold: `_init_scaffold.sh` creates `docs/memory/` and `docs/specs/` (not `fab/memory/` or `fab/specs/`)
- [x] CHK-005 Scaffold templates: `scaffold/memory-index.md` and `scaffold/specs-index.md` reference `docs/` paths
- [x] CHK-006 Skill files: No remaining `fab/memory/` or `fab/specs/` references in any `.kit/skills/*.md` file
- [x] CHK-007 Templates: `templates/brief.md` and `templates/spec.md` use `docs/memory/` paths
- [x] CHK-008 Scripts: `fab-help.sh` and `_stageman.sh` use `docs/` paths
- [x] CHK-009 Constitution: Principles II and VI reference `docs/memory/` and `docs/specs/`
- [x] CHK-010 Migration file: `fab/.kit/migrations/0.2.0-to-0.3.0.md` exists with correct instructions

## Behavioral Correctness

- [x] CHK-011 Cross-links preserved: Relative links between `docs/memory/` and `docs/specs/` files (e.g., `../specs/index.md`) resolve correctly
- [x] CHK-012 README links: All `fab/memory/` and `fab/specs/` links in README.md updated to `docs/` paths

## Scenario Coverage

- [x] CHK-013 New project bootstrap: Scaffold creates `docs/memory/index.md` and `docs/specs/index.md` (no `fab/memory/` or `fab/specs/` created)
- [x] CHK-014 Existing project migration: Migration instructions correctly describe moving `fab/memory/` → `docs/memory/` and `fab/specs/` → `docs/specs/`

## Edge Cases & Error Handling

- [x] CHK-015 No stale references: `grep -r 'fab/memory/\|fab/specs/' fab/.kit/` returns zero matches (excluding migration file which references old paths in instructions)
- [x] CHK-016 Archived changes untouched: No files under `fab/changes/archive/` are modified

## Documentation Accuracy

- [x] CHK-017 Memory files: All memory files under `docs/memory/` that describe directory paths reference `docs/` not `fab/` for memory/specs locations
- [x] CHK-018 Memory index: `docs/memory/index.md` boilerplate describes `docs/memory/` as the canonical location

## Cross References

- [x] CHK-019 Memory-to-specs links: `docs/memory/index.md` cross-reference to specs resolves to `docs/specs/index.md`
- [x] CHK-020 Specs-to-memory links: `docs/specs/index.md` cross-reference to memory resolves to `docs/memory/index.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
