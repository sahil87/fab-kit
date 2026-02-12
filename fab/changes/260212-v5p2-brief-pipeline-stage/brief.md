# Brief: Add brief as a formal pipeline stage

**Change**: 260212-v5p2-brief-pipeline-stage
**Created**: 2026-02-12
**Status**: Draft

## Origin

User requested:
> Add brief as a formal pipeline stage in config.yaml and update all skills to handle brief stage consistently with the template

## Why

Currently there's an inconsistency between the `.status.yaml` template (which has `brief: active`) and the actual pipeline stages defined in `config.yaml` (which starts with `spec`). This creates confusion:
- The template includes `brief: active` as the first stage
- But `config.yaml` stages list doesn't include `brief`
- `fab-new.md` instructions contradict the template by specifying `spec: active`
- Skills that traverse stages don't recognize `brief` as a valid stage

Making `brief` a formal stage aligns the system with the template and makes the pipeline more consistent - every artifact (brief, spec, tasks) has a corresponding stage.

## What Changes

1. **Add `brief` stage to `config.yaml`**:
   - Insert as first stage in the pipeline
   - Set `generates: brief.md`
   - Mark as `required: true` (always created by fab-new)
   - No prerequisites (it's the entry point)

2. **Update `fab-new.md`** to remove the hardcoded YAML and actually use the template, or update the hardcoded YAML to match the template (`brief: active` not `spec: active`)

3. **Update skills that reference stages**:
   - `fab-continue.md` - handle brief stage traversal (though brief is created by fab-new, not fab-continue)
   - `fab-ff.md` - skip brief stage (already completed by fab-new)
   - `fab-fff.md` - skip brief stage (already completed by fab-new)
   - `fab-status.md` - display brief stage correctly
   - `fab-clarify.md` - support clarifying the brief artifact
   - Any scripts that parse `.status.yaml` progress map

4. **Fix existing changes** that were created with the old format (`spec: active` without `brief`)

## Affected Docs

### New Docs
None - this is an internal workflow change.

### Modified Docs
- `fab-workflow/configuration`: Document the brief stage in the stages section
- `fab-workflow/change-lifecycle`: Update stage progression diagram to include brief
- `fab-workflow/fab-new`: Update to reflect that it sets brief: active
- `fab-workflow/planning-skills`: Update stage descriptions to include brief

### Removed Docs
None

## Impact

**Affected Files**:
- `fab/config.yaml` — add brief stage definition
- `fab/.kit/skills/fab-new.md` — fix .status.yaml initialization
- `fab/.kit/skills/fab-continue.md` — handle brief stage (likely skip since fab-new creates it)
- `fab/.kit/skills/fab-ff.md` — skip brief stage in fast-forward logic
- `fab/.kit/skills/fab-fff.md` — skip brief stage in full pipeline
- `fab/.kit/skills/fab-status.md` — ensure brief stage displays correctly
- `fab/.kit/skills/fab-clarify.md` — support clarifying brief.md
- `fab/.kit/scripts/fab-preflight.sh` — may need updates if it validates stages

**Migration Concern**: Existing changes with `spec: active` (no brief entry) will need handling. Options:
- Retroactively add `brief: done` to all existing .status.yaml files
- Make stage traversal logic tolerant of missing brief stage
- Add a migration script

**User Experience**:
- More consistent behavior between template and actual usage
- Clearer pipeline progression (every artifact is a stage)
- `/fab-status` output more accurate

## Open Questions

None — all decision points resolved via SRAD analysis.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Mark brief as required: true | Always created by fab-new, never optional |
| 2 | Confident | Skills should skip brief stage | Brief is always created by fab-new before other skills run |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
