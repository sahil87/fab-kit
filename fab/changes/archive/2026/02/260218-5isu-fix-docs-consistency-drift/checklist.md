# Quality Checklist: Fix Documentation Consistency Drift

**Change**: 260218-5isu-fix-docs-consistency-drift
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 `/fab-init` replacement: All `/fab-init` references in `docs/specs/` replaced with `/fab-setup`
- [x] CHK-002 `/fab-init` replacement: All `/fab-init` references in `docs/memory/` replaced with `/fab-setup`
- [x] CHK-003 skills.md rewrite: `/fab-init` section replaced with `/fab-setup` documenting `config`, `constitution`, `migrations` subcommands
- [x] CHK-004 "briefs" replacement: `docs/specs/architecture.md` uses "intakes" not "briefs"
- [x] CHK-005 `_init_scaffold.sh` removal: No references in `docs/specs/architecture.md`
- [x] CHK-006 `/fab-update` removal: No `/fab-update` node in `docs/specs/user-flow.md` diagram
- [x] CHK-007 `/fab-update` prose: All prose references replaced with `/fab-setup migrations`
- [x] CHK-008 `archive:` → `hydrate:`: Template spec uses `hydrate` stage name in progress map
- [x] CHK-009 Missing fields added: `change_type`, `confidence` block, `stage_metrics` documented in template spec
- [x] CHK-010 `lib/sync-workspace.sh` replacement: Stale references corrected in memory files
- [x] CHK-011 Kit architecture: `model-tiers.yaml` removed from directory tree
- [x] CHK-012 Kit architecture: `fab-fff.md` added after `fab-ff.md` in skills listing

## Behavioral Correctness

- [x] CHK-013 `/fab-setup` rewrite accuracy: skills.md section matches `docs/memory/fab-workflow/setup.md` as source of truth
- [x] CHK-014 Template spec accuracy: `.status.yaml` documentation matches actual `fab/.kit/templates/status.yaml`

## Scenario Coverage

- [x] CHK-015 Zero-match grep: `/fab-init` returns 0 hits in `docs/specs/` and `docs/memory/`
- [x] CHK-016 Zero-match grep: `_init_scaffold.sh` returns 0 hits in `docs/specs/`
- [x] CHK-017 Zero-match grep: `/fab-update` returns 0 hits in `docs/specs/` and `docs/memory/` (excluding intentional deprecated references)
- [x] CHK-018 Zero-match grep: stale `lib/sync-workspace.sh` returns 0 hits in `docs/memory/`

## Edge Cases & Error Handling

- [x] CHK-019 Deprecated references preserved: `/fab-init` and `/fab-update` entries in `docs/memory/fab-workflow/setup.md` Deprecated Requirements section remain unchanged
- [x] CHK-020 No false positives: Only genuinely stale references replaced — legitimate uses of similar strings (e.g., `init` in other contexts) preserved

## Code Quality

- [x] CHK-021 Pattern consistency: Replacement text follows existing naming and formatting conventions in each file
- [x] CHK-022 No unnecessary duplication: No redundant or duplicated content introduced by rewrites

## Documentation Accuracy

- [x] CHK-023 Cross-references valid: All updated references point to correct targets (file paths, command names)
- [x] CHK-024 Structural integrity: No broken markdown formatting, no orphaned sections from removals

## Cross References

- [x] CHK-025 Spec-memory alignment: Updated spec files and memory files are consistent with each other for the corrected terms
