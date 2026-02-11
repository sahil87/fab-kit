# Proposal: Add `created_by` Attribution to Changes

**Change**: 260211-endg-add-created-by-field
**Created**: 2026-02-11
**Status**: Draft

## Why

`.status.yaml` tracks when a change was created but not by whom. For projects with multiple contributors (human or agent), there's no way to trace a change back to its initiator. Adding a `created_by` field provides lightweight, zero-friction attribution.

## What Changes

- Add a `created_by` field to the `.status.yaml` template, populated automatically from `git config user.name` at change creation time
- Update `/fab-new` and `/fab-discuss` to set `created_by` when initializing `.status.yaml`
- Update `/fab-status` to display `created_by` in its output
- Update the `.status.yaml` template spec in `fab/specs/templates.md`
- Fallback value is `"unknown"` when git config is not set

## Affected Docs

### New Docs

(none)

### Modified Docs

- `fab-workflow/templates`: Update `.status.yaml` template to include `created_by` field and document its behavior
- `fab-workflow/planning-skills`: Note `created_by` population in `/fab-new` and `/fab-discuss` behavior
- `fab-workflow/execution-skills`: Note `created_by` display in `/fab-status` behavior

### Removed Docs

(none)

## Impact

- **Skills**: `/fab-new`, `/fab-discuss` (write `created_by`), `/fab-status` (read and display)
- **Templates**: `.status.yaml` template gains one new field
- **Backward compatibility**: Existing archived changes won't have the field — skills reading `created_by` must tolerate its absence (treat as omitted, not error)
- **Dependencies**: Requires `git config user.name` to be set for meaningful attribution; falls back to `"unknown"`

## Open Questions

(none — all decisions resolved through discussion)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Fallback to `"unknown"` when git config unset | Standard defensive default; avoids blocking change creation |
| 2 | Confident | `/fab-new`, `/fab-discuss`, `/fab-status` are the only skills affected | These are the only skills that create `.status.yaml` or display its summary |
| 3 | Confident | Existing archived changes not backfilled | Disruptive and low value; field is optional for backward compat |
