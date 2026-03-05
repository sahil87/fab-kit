# Brief: Re-evaluate Checklist Folder Location

**Change**: 260212-ipoe-checklist-folder-location
**Created**: 2026-02-12
**Status**: Draft

## Origin

**Backlog**: [ipoe] | **Linear**: DEV-1015

User invoked: `/fab-new ipoe`

Linear issue: "Re-evaluate storing checklists in separate folder - Question the design decision: Why are checklists (and only checklists) saved in a separate folder in the changes directory? Should they be co-located with other change artifacts?"

**Scope decision**: User confirmed this change should analyze alternatives and implement refactoring if moving checklists to root is justified.

## Why

The current structure stores quality checklists in `fab/changes/{name}/checklists/quality.md` while all other artifacts (brief.md, spec.md, tasks.md, .status.yaml) live at the change root. This subfolder contains only one file across all archived changes, raising questions about:

1. **Consistency**: Why treat checklists differently from other artifacts?
2. **YAGNI**: Is a subfolder justified for a single file?
3. **Discoverability**: Does the extra directory depth help or hinder?

This change evaluates the design rationale and executes refactoring if co-location is more appropriate.

## What Changes

1. **Analysis**: Document the historical rationale for separate folder (if discoverable), evaluate alternatives (keep, move to root, expand with additional checklist types)
2. **Decision**: Determine recommended approach based on consistency, extensibility, and workflow clarity
3. **Implementation** (if moving is justified):
   - Move `checklists/quality.md` → `quality.md` at change root
   - Update all skill references (fab-new, fab-continue, fab-ff, fab-review, fab-archive)
   - Update templates and documentation
   - Update `.status.yaml` checklist.path field handling

## Affected Docs

### Modified Docs
- `fab-workflow/templates`: Update checklist template section and path references
- `fab-workflow/planning-skills`: Update fab-continue and fab-ff checklist generation paths
- `fab-workflow/execution-skills`: Update fab-review and fab-archive checklist validation paths
- `fab-workflow/change-lifecycle`: Update .status.yaml checklist.path field documentation
- `fab-workflow/kit-architecture`: Update change folder structure diagram (if it includes checklist location)

## Impact

**Skills affected** (if moving):
- `/fab-new`: checklists/ directory creation (line 121)
- `/fab-continue`: checklist generation path (line 198, 244)
- `/fab-ff`: checklist generation path (line 156, 206)
- `/fab-review`: checklist reading and validation (lines 30, 40, 54, 83, 90, 290)
- `/fab-archive`: checklist verification (lines 30, 40, 67)
- `_generation.md`: shared checklist generation logic (lines 57-58)

**Templates affected**:
- `fab/.kit/templates/status.yaml`: checklist.path default value
- `fab/.kit/templates/brief.md`: No change (doesn't reference checklists)
- `fab/.kit/templates/spec.md`: No change
- `fab/.kit/templates/tasks.md`: No change

**Backward compatibility**: All existing archived changes have checklists/ subfolder - this change only affects new changes going forward.

## Open Questions

None - scope clarified via user confirmation.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Moving to root likely preferred | Consistency with other artifacts (brief.md, spec.md, tasks.md at root), YAGNI principle (no evidence of multiple checklist types), simpler paths |
| 2 | Tentative | Evaluation should include expansion alternative | Even though backlog says "can be moved", should consider whether separate folder is justified if planning multiple checklist types (security.md, performance.md, etc.) |
| 3 | Certain | Single atomic change | Straightforward file move + path updates, no need for multi-phase approach |
