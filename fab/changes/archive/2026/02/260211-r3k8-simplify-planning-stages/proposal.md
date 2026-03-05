# Proposal: Simplify Planning Stages

**Change**: 260211-r3k8-simplify-planning-stages
**Created**: 2026-02-11
**Status**: Draft

## Why

The 4-stage planning pipeline (proposal → specs → plan → tasks) has redundancy. Proposal and specs both define "what" at different detail levels — the transition is elaboration, not a phase shift. Plan is optional and almost always skipped; its useful content (architectural decisions) belongs in the spec. Reducing to 3 stages cuts ceremony without losing structure.

## What Changes

- Rename `proposal` stage to `brief` (artifact: `brief.md`)
- Rename `specs` stage to `spec` (artifact: `spec.md`, resolves plural/singular mismatch)
- Drop `plan` stage entirely — spec absorbs architectural decisions via an optional `## Design Decisions` section (agent includes when warranted)
- Rename root folder `fab/specs/` to `fab/design/` to resolve "spec" overload (per-change spec vs. project-level specs)
- `/fab-new` lands on `brief` (lightweight one-shot capture); `/fab-discuss` lands on `spec` (produces both `brief.md` + `spec.md`, marks both stages done)
- Pipeline becomes 6 stages: `brief → spec → tasks → apply → review → archive`

## Affected Docs

### New Docs
(none)

### Modified Docs
- `fab-workflow/planning-skills`: Stage names, generation procedures (remove plan procedure), stage guard logic, `/fab-continue` and `/fab-ff` behavior
- `fab-workflow/change-lifecycle`: Stage progression, `.status.yaml` schema, progress keys
- `fab-workflow/configuration`: `stages:` block in config.yaml, stage IDs
- `fab-workflow/templates`: Remove `plan.md` template, rename `proposal.md` → `brief.md`
- `fab-workflow/kit-architecture`: Directory structure references
- `fab-workflow/specs-index`: Rename to design-index or update references
- `fab-workflow/context-loading`: Update `fab/specs/index.md` reference to `fab/design/index.md` in Always Load layer, update stage names
- `fab-workflow/clarify`: Remove `plan` from stage list, update stage names to brief/spec/tasks

### Removed Docs
(none — content absorbed, not deleted)

## Impact

- **config.yaml**: `stages:` block — rename IDs, remove plan entry, update `requires` chains
- **constitution.md**: References to `fab/specs/` → `fab/design/`
- **All skill files**: Stage name references (proposal → brief, specs → spec, plan references removed)
- **`/fab-new` skill**: Output changes from `proposal.md` to `brief.md`
- **`/fab-discuss` skill**: Output changes to produce both `brief.md` (summary snapshot) and `spec.md` (full structured requirements), marking both stages done
- **Templates**: `proposal.md` → `brief.md`, remove `plan.md`
- **_generation.md**: Remove Plan Generation Procedure, update Spec Generation Procedure to optionally include Design Decisions
- **_context.md**: Update Next Steps Convention table, SRAD skill table, Context Loading references (`fab/specs/index.md` → `fab/design/index.md`), all stage name references
- **fab/specs/ directory**: Rename to `fab/design/`, update all internal cross-references
- **fab/docs/ files**: Update all references to old stage names and `fab/specs/`
- **Glossary**: Update stage terminology

## Open Questions

(none — all decisions resolved during discussion)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Certain | 3-stage planning: brief → spec → tasks | User explicitly chose this structure |
| 2 | Certain | Drop plan stage entirely | User confirmed plan almost always skipped |
| 3 | Certain | Rename fab/specs/ → fab/design/ | User confirmed to resolve "spec" overload |
| 4 | Certain | fab/docs/ unchanged | Agreed — no collision, universally understood |
| 5 | Certain | Spec includes optional Design Decisions section | User agreed — agent decides when to include |
| 6 | Certain | `/fab-discuss` produces both brief + spec | User explicitly chose Option B — each skill lands where its depth naturally reaches |
| 7 | Confident | brief.md as artifact name | Follows convention of stage-name.md |
| 8 | Confident | Update all downstream references in docs, skills, templates | Standard consequence of the rename |
