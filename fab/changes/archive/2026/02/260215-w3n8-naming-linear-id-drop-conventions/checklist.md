# Quality Checklist: Naming Linear ID & Drop Conventions

**Change**: 260215-w3n8-naming-linear-id-drop-conventions
**Generated**: 2026-02-15
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Naming format extended: `config.yaml` `naming.format` value is `"{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}"`
- [x] CHK-002 Naming comments updated: `config.yaml` inline comments explain the optional `{ISSUE}` component with examples
- [x] CHK-003 Conventions removed: `config.yaml` no longer contains the `conventions` section (comments or YAML)
- [x] CHK-004 fab-new updated: Step 1 references extended format and documents Linear ID insertion
- [x] CHK-005 fab-init updated: Config template uses extended naming format

## Behavioral Correctness

- [x] CHK-006 Backward compatibility: format without `{ISSUE}` is still `{YYMMDD}-{XXXX}-{slug}` (no regression)
- [x] CHK-007 Linear ID casing: `{ISSUE}` is documented as uppercase in all modified files

## Removal Verification

- [x] CHK-008 Conventions section fully removed: no `conventions` key, sub-keys, or associated comments remain in `config.yaml`
- [x] CHK-009 No dangling references: no skill file references `conventions` config section

## Scenario Coverage

- [x] CHK-010 Scenario "Change created from Linear ticket": fab-new Step 1 handles Linear ID from Step 0 parse
- [x] CHK-011 Scenario "Change created without Linear context": format falls back to `{YYMMDD}-{XXXX}-{slug}`
- [x] CHK-012 Scenario "Config reflects extended format": `config.yaml` shows updated format with documentation
- [x] CHK-013 Scenario "fab-init template uses extended format": init template generates correct naming section

## Documentation Accuracy

- [x] CHK-014 All modified files use consistent format string: `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}`
- [x] CHK-015 Examples in config.yaml comments show both with-ID and without-ID variants

## Cross References

- [x] CHK-016 No stale references to old format `{YYMMDD}-{XXXX}-{slug}` as the only format in modified files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
