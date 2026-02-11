# Proposal: Add Non-Goals and Design Decisions to Spec Template

**Change**: 260211-r4w8-spec-template-sections
**Created**: 2026-02-11
**Status**: Draft

## Why

When the plan stage was absorbed into the spec stage, two valuable sections were lost: Non-Goals (explicit scope boundaries) and Design Decisions (rationale and rejected alternatives). The current spec template captures behavioral requirements (what) but not design rationale (why) or scope exclusions. Without these, agents hitting ambiguity during `/fab-apply` have no record of why a particular approach was chosen.

## What Changes

- Add optional `## Non-Goals` section to `fab/.kit/templates/spec.md`, placed before requirement sections
- Add optional `## Design Decisions` section to `fab/.kit/templates/spec.md`, placed after requirements and before Deprecated Requirements
- Update skill prompts that generate spec content (`fab-continue`, `fab-ff`, `fab-discuss`) to populate the new sections when relevant

## Affected Docs

### New Docs
_None_

### Modified Docs
- `fab-workflow/templates`: spec.md template gains two optional sections

### Removed Docs
_None_

## Impact

- `fab/.kit/templates/spec.md` — template file itself
- `fab/.kit/skills/fab-continue.md` — generates spec content during spec stage
- `fab/.kit/skills/fab-ff.md` — fast-forwards through spec stage
- `fab/.kit/skills/fab-discuss.md` — generates spec in new change mode

## Open Questions

_None — scope is clear from discussion._

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Non-Goals before requirements, Design Decisions after | Natural reading flow: scope bounds -> requirements -> rationale |
| 2 | Confident | Decision/Why/Rejected format for Design Decisions | Proven structure from the old plan template |
| 3 | Confident | Bullet list format for Non-Goals | Simple, consistent with brief's bullet style |
| 4 | Confident | fab-continue, fab-ff, fab-discuss are the skills to update | These are the skills that generate spec content |
| 5 | Confident | Guidance via HTML comments in template | Consistent with existing template patterns |
