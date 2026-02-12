# Quality Checklist: Provider-Agnostic Model Tiers for Fab Skills

**Change**: 260212-k8m3-skill-model-tiers
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Two-Tier Classification: `model-tiers.yaml` defines exactly `fast` and `capable` tiers with correct structure
- [x] CHK-002 Tier Selection Criteria: criteria table documented in model-tiers doc matches spec
- [x] CHK-003 Audit and Tag: exactly 4 skills (fab-help, fab-status, fab-switch, fab-init) have `model_tier: fast`; remaining 12 have no `model_tier` field
- [x] CHK-004 Frontmatter Format: `model_tier` placed after `description` in YAML frontmatter for all 4 fast skills
- [x] CHK-005 Mapping File: `fab/.kit/model-tiers.yaml` exists with `tiers.fast.claude: haiku` and `tiers.capable.claude: null`
- [x] CHK-006 Per-Project Override: `config.yaml` `model_tiers:` section overrides `.kit/` defaults when present
- [x] CHK-007 Dual Deployment: `fab-setup.sh` creates skill symlinks AND agent files for fast-tier skills
- [x] CHK-008 Agent File Content: generated agent files contain `model: haiku` (not `model_tier: fast`) and full skill content
- [x] CHK-009 Deployment Error Handling: `fab-setup.sh` exits non-zero with descriptive errors for all specified failure cases

## Behavioral Correctness

- [x] CHK-010 Existing capable skill symlinks unchanged after running `fab-setup.sh`
- [x] CHK-011 Fast-tier skill symlinks still work for user invocation (symlink valid, content accessible)
- [x] CHK-012 No agent files generated for capable-tier skills

## Scenario Coverage

- [x] CHK-013 config.yaml override: `model_tiers.fast.claude: sonnet` in config produces `model: sonnet` in agent file
- [x] CHK-014 No model_tiers section: `.kit/` defaults used when config.yaml has no `model_tiers:` key
- [x] CHK-015 .kit/ update: re-running `fab-setup.sh` regenerates agent files and repairs symlinks

## Edge Cases & Error Handling

- [x] CHK-016 Missing model-tiers.yaml: script exits non-zero with `ERROR: fab/.kit/model-tiers.yaml not found`
- [x] CHK-017 Invalid model_tier value: `model_tier: medium` causes non-zero exit with descriptive error
- [x] CHK-018 Missing platform mapping: tier exists but no entry for platform causes non-zero exit
- [x] CHK-019 Malformed frontmatter: unparseable YAML causes non-zero exit with `ERROR: Cannot parse frontmatter`

## Documentation Accuracy

- [x] CHK-020 model-tiers.md contains tier naming, selection criteria, full audit, mapping format, provider extension guide, new-skill guidance
- [x] CHK-021 kit-architecture.md directory listing includes `model-tiers.yaml`; dual deployment documented
- [x] CHK-022 templates.md documents `model_tier` field with valid values and default behavior

## Cross References

- [x] CHK-023 docs/fab-workflow/index.md includes model-tiers entry
- [x] CHK-024 model-tiers.md cross-references kit-architecture and templates docs where appropriate

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
