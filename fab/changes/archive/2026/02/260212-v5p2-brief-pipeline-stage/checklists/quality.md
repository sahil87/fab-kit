# Quality Checklist: Add brief as a formal pipeline stage

**Change**: 260212-v5p2-brief-pipeline-stage
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Config brief stage: `fab/config.yaml` has `brief` as first stage with `id: brief`, `generates: brief.md`, `required: true`
- [x] CHK-002 Config spec prerequisite: `spec` stage entry has `requires: [brief]`
- [x] CHK-003 fab-new consistency: `fab-new.md` no longer shows hardcoded YAML with `spec: active`; references template correctly
- [x] CHK-004 fab-continue error handling: Error Handling table does not reject brief reset
- [x] CHK-005 fab-switch stage mapping: Stage Number Mapping table shows 6 stages with brief=1
- [x] CHK-006 configuration.md: Documents 6 stages with brief as valid stage ID
- [x] CHK-007 change-lifecycle.md: Migration Note no longer instructs removal of `brief:` from progress map

## Behavioral Correctness

- [x] CHK-008 fab-preflight.sh outputs `stage: brief` when `.status.yaml` has `progress.brief: active`
- [x] CHK-009 fab-preflight.sh infers `brief: done` when `.status.yaml` has no `brief:` entry (backward compat)
- [x] CHK-010 fab-status.sh displays `brief (1/6)` for changes at brief stage

## Scenario Coverage

- [x] CHK-011 Config stages list: first entry is brief, spec entry has `requires: [brief]`
- [x] CHK-012 Normal forward from brief: `/fab-continue` from `brief: active` would generate spec
- [x] CHK-013 Reset to brief: `/fab-continue brief` resets to `brief: active` and marks downstream pending
- [x] CHK-014 Fast-forward from brief: `/fab-ff` from `brief: active` sets `brief: done` and generates spec
- [x] CHK-015 Legacy status file: preflight handles missing `brief:` entry gracefully

## Edge Cases & Error Handling

- [x] CHK-016 fab-new Step 8 text no longer says "brief is an input artifact, not a pipeline stage"
- [x] CHK-017 No contradictions between Normal Flow, Reset Flow, and Error Handling in fab-continue.md

## Documentation Accuracy

- [x] CHK-018 configuration.md stage count matches actual config.yaml (6 stages)
- [x] CHK-019 change-lifecycle.md migration note is consistent with brief being a formal stage

## Cross References

- [x] CHK-020 All files that already handle brief correctly (preflight.sh, status.sh, fab-ff.md, fab-fff.md, fab-clarify.md, planning-skills.md) were verified and left unchanged

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
