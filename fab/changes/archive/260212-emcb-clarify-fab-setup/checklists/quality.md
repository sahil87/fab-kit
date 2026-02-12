# Quality Checklist: Clarify fab-setup Responsibilities and Initialize fab/design Folder

**Change**: 260212-emcb-clarify-fab-setup
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-setup.sh creates fab/design/: Script creates `fab/design/` directory and `fab/design/index.md` skeleton when they don't exist
- [x] CHK-002 fab-setup.sh skeleton content: The `fab/design/index.md` skeleton matches the content from `/fab-init` step 1d (blockquote header + empty table)
- [x] CHK-003 fab-setup.sh remains structural: Script does not create config.yaml, constitution.md, or prompt for user input
- [x] CHK-004 fab-init delegation documented: `fab-init.md` skill file documents the delegation pattern to `fab-setup.sh`
- [x] CHK-005 init.md delegation section: `fab/docs/fab-workflow/init.md` includes a section explaining the delegation relationship
- [x] CHK-006 distribution.md updated: `fab/docs/fab-workflow/distribution.md` reflects fab/design/ in bootstrap artifacts

## Behavioral Correctness

- [x] CHK-007 Idempotent re-run: Running fab-setup.sh when `fab/design/index.md` already exists does NOT overwrite it
- [x] CHK-008 Section placement: New fab/design/ section is between docs/index.md (section 3) and skill symlinks (section 5) in fab-setup.sh

## Scenario Coverage

- [x] CHK-009 Fresh bootstrap scenario: On a fresh project with only `fab/.kit/`, fab-setup.sh creates all structural artifacts including design/index.md
- [x] CHK-010 Re-run with existing design/: fab-setup.sh skips design/index.md creation when file already exists

## Edge Cases & Error Handling

- [x] CHK-011 Missing fab/design/ directory: fab-setup.sh creates the directory if only the file is missing (mkdir -p handles this)

## Documentation Accuracy

- [x] CHK-012 Changelog entries: Both init.md and distribution.md have changelog entries for this change

## Cross References

- [x] CHK-013 Responsibility table consistency: The responsibility split documented in fab-init.md matches what fab-setup.sh actually does
- [x] CHK-014 Distribution bootstrap list: The list of artifacts in distribution.md matches what fab-setup.sh actually creates

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
