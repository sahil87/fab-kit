# Quality Checklist: Expand fab-init Command Family

**Change**: 260212-h9k3-fab-init-family
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-init-constitution: Skill file exists at `fab/.kit/skills/fab-init-constitution.md` with correct frontmatter
- [x] CHK-002 fab-init-constitution: Create mode — generates constitution when file doesn't exist, requires config.yaml
- [x] CHK-003 fab-init-constitution: Update mode — displays current content, offers amendment menu, applies changes
- [x] CHK-004 fab-init-constitution: Semantic versioning — MAJOR/MINOR/PATCH bumps based on change type
- [x] CHK-005 fab-init-constitution: Multiple amendments per session with highest-precedence version bump
- [x] CHK-006 fab-init-constitution: Governance audit trail in output
- [x] CHK-007 fab-init-constitution: Structural preservation (heading hierarchy, Roman numerals, re-numbering)
- [x] CHK-008 fab-init-config: Skill file exists at `fab/.kit/skills/fab-init-config.md` with correct frontmatter
- [x] CHK-009 fab-init-config: Section menu lists all 8 config sections
- [x] CHK-010 fab-init-config: Optional section argument skips menu (e.g., `/fab-init-config context`)
- [x] CHK-011 fab-init-config: Invalid section argument shows helpful error with valid section names
- [x] CHK-012 fab-init-config: YAML validation after each edit with revert option
- [x] CHK-013 fab-init-config: Comment and formatting preservation via string replacement
- [x] CHK-014 fab-init-validate: Skill file exists at `fab/.kit/skills/fab-init-validate.md` with correct frontmatter
- [x] CHK-015 fab-init-validate: Config validation covers all 8 structural checks
- [x] CHK-016 fab-init-validate: Constitution validation covers all 6 structural checks
- [x] CHK-017 fab-init-validate: Combined report when both files exist
- [x] CHK-018 fab-init-validate: Actionable fix suggestions for every failure
- [x] CHK-019 fab-init-validate: Clear exit status messaging
- [x] CHK-020 fab-init.md: Delegates config creation to `/fab-init-config`
- [x] CHK-021 fab-init.md: Delegates constitution creation to `/fab-init-constitution`
- [x] CHK-022 fab-init.md: "Related Commands" section lists all three family commands

## Behavioral Correctness

- [x] CHK-023 fab-init-constitution: No-op update leaves file unchanged and version unbumped
- [x] CHK-024 fab-init-config: No-op run leaves file unchanged
- [x] CHK-025 fab-init.md: Delegation is skipped when files already exist (idempotent)
- [x] CHK-026 fab-init-config: Multiple sections updatable in a single invocation

## Scenario Coverage

- [x] CHK-027 fab-init-constitution: Create Constitution — Fresh Project scenario
- [x] CHK-028 fab-init-constitution: Create Constitution — No Config scenario (error message)
- [x] CHK-029 fab-init-constitution: Update Constitution — Existing File scenario
- [x] CHK-030 fab-init-constitution: Multiple Amendments — Version Precedence scenario
- [x] CHK-031 fab-init-constitution: Principle Removal Re-numbers scenario
- [x] CHK-032 fab-init-config: Direct Section Access via Argument scenario
- [x] CHK-033 fab-init-config: Invalid Section Argument scenario
- [x] CHK-034 fab-init-config: Invalid YAML After Edit scenario (revert)
- [x] CHK-035 fab-init-config: Broken Stage Reference scenario
- [x] CHK-036 fab-init-validate: Valid Config scenario (8/8 checks)
- [x] CHK-037 fab-init-validate: Missing Required Field scenario (actionable fix)
- [x] CHK-038 fab-init-validate: Circular Stage Dependencies scenario
- [x] CHK-039 fab-init-validate: Missing Governance Section scenario
- [x] CHK-040 fab-init-validate: One File Missing scenario
- [x] CHK-041 fab-init.md: Init Delegates Config Creation scenario
- [x] CHK-042 fab-init.md: Init Skips Delegation When Files Exist scenario

## Edge Cases & Error Handling

- [x] CHK-043 fab-init-constitution: Missing config.yaml produces clear error message
- [x] CHK-044 fab-init-config: Missing config.yaml produces clear error message
- [x] CHK-045 fab-init-validate: Handles one or both files missing gracefully
- [x] CHK-046 fab-init-validate: Circular dependency detection works for transitive cycles

## Documentation Accuracy

- [x] CHK-047 All skill files follow `_context.md` preamble convention
- [x] CHK-048 All skill files have consistent structure with existing skills (Purpose, Arguments, Behavior, Output, Error Handling, Key Properties)
- [x] CHK-049 Skill descriptions match the brief's one-liners

## Cross References

- [x] CHK-050 fab-init.md Related Commands section references correct skill names and descriptions
- [x] CHK-051 Symlinks created by fab-setup.sh point to the correct skill files
- [x] CHK-052 All three new skills reference `_context.md` in their preamble

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
