# Brief: Add Conventions Section to config.yaml

**Change**: 260213-r3m7-add-conventions-section
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Add a conventions section to config.yaml for storing project-wide conventions (branch naming, PR naming, backlog location) that all fab skills can reference

## Why

Skills frequently need to reference project-wide conventions — branch naming patterns, PR title formats, backlog location — but these live nowhere structured today. The `naming` and `git` sections cover folder naming and branch prefix, but PR naming, backlog URL, and commit style have no home. The `context` field is free-form and unreliable for skills to parse programmatically. A dedicated `conventions` section in `config.yaml` gives every skill a single, structured place to look.

## What Changes

- Add a top-level `conventions:` key to `config.yaml` with structured fields for:
  - `branch_naming` — pattern or description of branch naming convention
  - `pr_title` — PR title format pattern
  - `backlog` — URL or location of the project backlog
  - Extensible for future conventions (commit style, issue labeling, etc.)
- Document the new section in the `config.yaml` template comments (inline documentation)
- Update `fab/design/architecture.md` Configuration section to document `conventions`
- Update `fab/docs/` with the new configuration capability

## Affected Docs

### New Docs
<!-- None — this extends existing configuration documentation -->

### Modified Docs
- `fab-workflow/configuration`: Add `conventions` section documentation to the configuration reference

### Removed Docs
<!-- None -->

## Impact

- **`fab/config.yaml`** — new top-level section added
- **`fab/design/architecture.md`** — Configuration section updated with `conventions` documentation
- **All skills via `_context.md` context loading** — no loading changes needed; skills already read `config.yaml` in full. The new section is automatically available.
- **Existing `naming` and `git` sections** — remain unchanged. `conventions` complements rather than replaces them (different scope: `naming` = folder names, `git` = git integration toggles, `conventions` = human/workflow conventions).

## Open Questions

<!-- No blocking questions — all decision points resolved via SRAD evaluation. -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Include branch_naming, pr_title, and backlog as initial keys | User explicitly listed these three; they cover the stated need |
| 2 | Confident | Use flat key-value pairs within the section | Consistent with existing config.yaml patterns (naming, git sections) |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
