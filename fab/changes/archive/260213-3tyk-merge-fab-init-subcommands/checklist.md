# Quality Checklist: Merge fab-init Subcommands into Single Skill

**Change**: 260213-3tyk-merge-fab-init-subcommands
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 Argument-Based Routing: `/fab-init` with no args triggers full bootstrap
- [x] CHK-002 Argument-Based Routing: `/fab-init config` triggers config behavior (menu mode)
- [x] CHK-003 Argument-Based Routing: `/fab-init config context` triggers config behavior with section argument
- [x] CHK-004 Argument-Based Routing: `/fab-init constitution` triggers constitution behavior
- [x] CHK-005 Argument-Based Routing: `/fab-init validate` triggers validate behavior
- [x] CHK-006 Argument-Based Routing: `/fab-init https://example.com` outputs hydrate redirect message
- [x] CHK-007 Merged Skill File Structure: all 7 sections present (frontmatter, purpose/args, preflight, bootstrap, config, constitution, validate)
- [x] CHK-008 Frontmatter Update: `name: fab-init`, updated description, `model_tier: fast`

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-009 Bootstrap step 1a references internal Config Behavior section (not `/fab-init-config`)
- [x] CHK-010 Bootstrap step 1b references internal Constitution Behavior section (not `/fab-init-constitution`)
- [x] CHK-011 Config behavior preserves complete logic from original `fab-init-config.md` (create mode, update mode, edit section flow, validation)
- [x] CHK-012 Constitution behavior preserves complete logic from original `fab-init-constitution.md` (create mode, update mode, amendment flow, version bumping)
- [x] CHK-013 Validate behavior preserves all 14 checks from original `fab-init-validate.md` (8 config + 6 constitution)

## Removal Verification
<!-- Every deprecated requirement is actually gone -->
- [x] CHK-014 Variant skill files deleted: `fab-init-config.md`, `fab-init-constitution.md`, `fab-init-validate.md` absent from `fab/.kit/skills/`
- [x] CHK-015 Stale symlink directories removed from `.claude/skills/`
- [x] CHK-016 Stale agent files removed from `.claude/agents/`
- [x] CHK-017 Stale entries removed from `.opencode/commands/` and `.agents/skills/`

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-018 Skill discovery: `fab-setup.sh` creates only `fab-init` symlink/agent (no variant entries)
- [x] CHK-019 fab-setup.sh re-run produces no errors after cleanup

## Cross-References
<!-- All doc references updated to new syntax -->
- [x] CHK-020 All 33 occurrences across 6 doc files updated from `/fab-init-{variant}` to `/fab-init {variant}`
- [x] CHK-021 `_context.md` confirmed: no variant references (0 occurrences)

## Documentation Accuracy
<!-- Content in docs accurately reflects the new reality -->
- [x] CHK-022 `init-family.md` overview accurately describes subcommand structure (not separate commands)
- [x] CHK-023 `init.md` Related Commands section uses new subcommand syntax

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
