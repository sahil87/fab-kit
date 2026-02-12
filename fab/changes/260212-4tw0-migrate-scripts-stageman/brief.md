# Brief: Migrate Scripts to Use Stage Manager

**Change**: 260212-4tw0-migrate-scripts-stageman
**Created**: 2026-02-12
**Status**: Draft

## Origin

> User requested: "Migrate existing scripts to use stageman.sh for stage/state queries"

## Why

Stage and state knowledge is currently hardcoded in multiple bash scripts (`fab-status.sh`, `fab-preflight.sh`, template generation, etc.). We now have a canonical workflow schema (`fab/.kit/schemas/workflow.yaml`) and Stage Manager query utility (`fab/.kit/scripts/stageman.sh`), but existing scripts haven't been migrated to use it yet.

This creates:
- **Maintenance burden**: Changes to stages/states require updates in 7+ locations
- **Inconsistency risk**: Hardcoded stage lists can drift out of sync with schema
- **Missed validation**: Scripts don't leverage `validate_status_file()` and other utilities

The migration guide (`fab/.kit/schemas/MIGRATION.md`) documents the refactoring patterns, but the actual scripts still use hardcoded logic.

## What Changes

**Update fab-status.sh**:
- Source `stageman.sh` at the top
- Replace hardcoded stage loop `for s in brief spec tasks apply review archive` with `for s in $(get_all_stages)`
- Replace hardcoded stage number case statement with `get_stage_number "$stage"`
- Replace hardcoded `symbol()` function with calls to `get_state_symbol "$state"`

**Update fab-preflight.sh**:
- Source `stageman.sh` at the top
- Replace hardcoded stage loop with `get_all_stages`
- Add optional validation: call `validate_status_file "$status_file"` to catch schema violations early

**Update fab-help.sh**:
- Remove hardcoded stage progression from documentation string
- Dynamically generate stage list from `get_all_stages` if needed (or keep as static doc)

**Update template generation** (if applicable):
- Check if any skills dynamically generate `.status.yaml` templates
- Ensure they use `get_initial_state` and `get_allowed_states` from stageman

**Remove hardcoded logic**:
- Delete all hardcoded stage lists, state symbol mappings, and stage number mappings
- Scripts become pure consumers of the schema via stageman

## Affected Docs

### Modified Docs
- `fab-workflow/kit-architecture`: Update script implementation details to reference stageman integration
- `fab-workflow/preflight`: Update preflight script documentation to mention stageman usage and validation

## Impact

**Files affected**:
- `fab/.kit/scripts/fab-status.sh` — primary display script
- `fab/.kit/scripts/fab-preflight.sh` — validation and stage detection
- `fab/.kit/scripts/fab-help.sh` — may reference stage progression
- `fab/.kit/templates/status.yaml` — template may need dynamic generation (check if sourced by bash)

**Benefits**:
- Single source of truth enforced at runtime
- Adding/removing stages becomes schema-only change
- Scripts automatically adapt to schema changes
- Validation catches `.status.yaml` corruption early

**Risks**:
- Minimal: stageman.sh is tested and working
- All changes are in kit scripts (not user-facing config)
- Can be validated by running `fab-status` and `fab-preflight.sh` before/after

## Open Questions

None — migration patterns are documented in `fab/.kit/schemas/MIGRATION.md` and stageman.sh is fully functional and tested.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | All scripts in `fab/.kit/scripts/` should be checked | Standard kit directory structure per constitution |
| 2 | Confident | Hardcoded logic should be completely removed | Single source of truth principle from MIGRATION.md |
| 3 | Confident | `validate_status_file` should be added to preflight | Catches corruption early, aligns with preflight's validation role |

3 assumptions made (3 confident, 0 tentative).
