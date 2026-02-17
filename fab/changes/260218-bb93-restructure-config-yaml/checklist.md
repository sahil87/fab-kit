# Quality Checklist: Restructure config.yaml

**Change**: 260218-bb93-restructure-config-yaml
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Context extraction: `fab/context.md` exists with project context as free-form markdown
- [ ] CHK-002 Code quality extraction: `fab/code-quality.md` scaffold has `## Principles`, `## Anti-Patterns`, `## Test Strategy` sections
- [ ] CHK-003 Stages removal: No `stages:` section in `fab/config.yaml` or scaffold
- [ ] CHK-004 Model tiers consolidation: `model_tiers:` section present in config.yaml and scaffold with correct defaults
- [ ] CHK-005 Model-tiers.yaml deleted: `fab/.kit/model-tiers.yaml` no longer exists
- [ ] CHK-006 Always-load updated: `_context.md` Layer 1 lists 6 files including context.md and code-quality.md (both optional)
- [ ] CHK-007 fab-setup sections updated: Valid sections list contains `project`, `source_paths`, `rules`, `checklist`, `git`, `naming`, `model_tiers`; excludes `stages`, `code_quality`, `context`
- [ ] CHK-008 fab-setup bootstrap: Steps to create context.md and code-quality.md from scaffold templates
- [ ] CHK-009 Migration file: `fab/.kit/migrations/0.7.0-to-0.8.0.md` exists with pre-checks, changes, and verification

## Behavioral Correctness

- [ ] CHK-010 sync-workspace.sh reads from config.yaml: No references to `model-tiers.yaml` remain; reads `model_tiers.fast.claude` from config.yaml
- [ ] CHK-011 sync-workspace.sh fallback: Hardcoded `haiku` fallback when no `model_tiers` in config or no config.yaml
- [ ] CHK-012 fab-continue code_quality: Apply and review stages reference `fab/code-quality.md`, not `config.yaml` for code_quality
- [ ] CHK-013 _generation.md checklist: Checklist procedure reads from `fab/code-quality.md`

## Removal Verification

- [ ] CHK-014 No `context:` in config.yaml: Section fully removed from both project config and scaffold
- [ ] CHK-015 No `code_quality:` in config.yaml: Section fully removed (including commented-out examples) from both project config and scaffold
- [ ] CHK-016 No `stages:` in config.yaml: Section fully removed from both project config and scaffold
- [ ] CHK-017 No model-tiers.yaml references: No remaining references to `fab/.kit/model-tiers.yaml` in any skill or script
- [ ] CHK-018 No stale Consumed-by comments: Config.yaml comments referencing removed sections are cleaned up

## Scenario Coverage

- [ ] CHK-019 Context optional: Skills don't error when `fab/context.md` is missing
- [ ] CHK-020 Code quality optional: Skills don't error when `fab/code-quality.md` is missing
- [ ] CHK-021 Scaffold includes model_tiers: Scaffold config.yaml has model_tiers with comments
- [ ] CHK-022 Migration idempotent: Migration pre-checks and verification steps are correctly structured

## Edge Cases & Error Handling

- [ ] CHK-023 Sync-workspace.sh no config.yaml: Script runs without error on new projects before config exists
- [ ] CHK-024 Commented code_quality handling: Migration correctly handles projects with only commented-out code_quality

## Code Quality

- [ ] CHK-025 Pattern consistency: New scaffold files follow existing scaffold naming and structure patterns
- [ ] CHK-026 No unnecessary duplication: Config.yaml companion file references are consistent across all skills

## Documentation Accuracy

- [ ] CHK-027 Config.yaml header: Companion files comment block lists all 4 companions (constitution, context, code-quality, memory index)
- [ ] CHK-028 Skill cross-references: All skill files that reference config sections use updated paths

## Cross References

- [ ] CHK-029 Memory files: Affected memory files (configuration, model-tiers, context-loading) accurately listed for hydration
- [ ] CHK-030 VERSION bumped: `fab/.kit/VERSION` updated to 0.8.0

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
