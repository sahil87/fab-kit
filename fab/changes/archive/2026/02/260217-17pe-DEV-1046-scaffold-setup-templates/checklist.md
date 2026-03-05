# Quality Checklist: Scaffold Setup Templates

**Change**: 260217-17pe-DEV-1046-scaffold-setup-templates
**Generated**: 2026-02-17
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Config template file: `fab/.kit/scaffold/config.yaml` exists with all config sections and `{PLACEHOLDER}` syntax
- [x] CHK-002 Constitution template file: `fab/.kit/scaffold/constitution.md` exists with minimal skeleton structure
- [x] CHK-003 Config create mode: `fab-setup.md` references scaffold file instead of inline YAML
- [x] CHK-004 Constitution create mode: `fab-setup.md` references scaffold file instead of inline markdown
- [x] CHK-005 Memory index reference: `fab-setup.md` step 1c references `fab/.kit/scaffold/memory-index.md`
- [x] CHK-006 Specs index reference: `fab-setup.md` step 1d references `fab/.kit/scaffold/specs-index.md`

## Behavioral Correctness
- [x] CHK-007 Config template content match: scaffold `config.yaml` contains every section, key, default value, and comment from the former inline template
- [x] CHK-008 Constitution skeleton is minimal: scaffold contains structural headings and placeholder principle only, no populated examples

## Scenario Coverage
- [x] CHK-009 Placeholder syntax verified: scaffold config uses `{PROJECT_NAME}`, `{PROJECT_DESCRIPTION}`, `{TECH_STACK_AND_CONVENTIONS}`, `{SOURCE_PATHS}`
- [x] CHK-010 Non-placeholder defaults preserved: stage definitions, naming format, git defaults, rules, checklist defaults are actual values (not placeholders)
- [x] CHK-011 Inline template divergence eliminated: no inline templates remain in `fab-setup.md` for memory-index or specs-index

## Code Quality
- [x] CHK-012 Pattern consistency: scaffold files follow naming and comment style of existing scaffold files (`memory-index.md`, `specs-index.md`, `envrc`, `gitignore-entries`)
- [x] CHK-013 No unnecessary duplication: no template content duplicated between scaffold files and `fab-setup.md`

## Documentation Accuracy
- [x] CHK-014 Scaffold file references: all instructions in `fab-setup.md` that reference scaffold files use correct relative paths

## Cross References
- [x] CHK-015 Config schema alignment: scaffold `config.yaml` matches the schema documented in `docs/memory/fab-workflow/configuration.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
