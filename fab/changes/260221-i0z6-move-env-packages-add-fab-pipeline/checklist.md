# Quality Checklist: Move env-packages.sh to lib & Add fab-pipeline.sh Entry Point

**Change**: 260221-i0z6-move-env-packages-add-fab-pipeline
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 env-packages.sh relocation: File exists at `fab/.kit/scripts/lib/env-packages.sh` and is removed from `fab/.kit/scripts/env-packages.sh`
- [ ] CHK-002 KIT_DIR resolution: `KIT_DIR` resolves to `fab/.kit/` from the new `scripts/lib/` location
- [ ] CHK-003 Source references updated: `fragment-.envrc` and `rc-init.sh` reference the new path
- [ ] CHK-004 fab-pipeline.sh exists: Executable wrapper at `fab/.kit/scripts/fab-pipeline.sh`
- [ ] CHK-005 fab-pipeline.sh no-args listing: Lists pipelines from `fab/pipelines/*.yaml` excluding `example.yaml`
- [ ] CHK-006 fab-pipeline.sh help: `-h` and `--help` print usage to stdout, exit 0
- [ ] CHK-007 fab-pipeline.sh partial matching: Case-insensitive substring matching with ambiguity error
- [ ] CHK-008 fab-pipeline.sh explicit path bypass: Arguments with `/` or `.yaml` pass through unchanged
- [ ] CHK-009 fab-pipeline.sh delegation: Uses `exec` to delegate to `pipeline/run.sh`
- [ ] CHK-010 changeman resolve in run.sh: Manifest change IDs resolved through `changeman resolve` before dispatch
- [ ] CHK-011 --worktree-name in dispatch.sh: `wt-create` call includes `--worktree-name "$CHANGE_ID"`
- [ ] CHK-012 stageman stage detection: `dispatch.sh` uses `stageman display-stage` instead of raw yq for stage determination

## Behavioral Correctness
- [ ] CHK-013 PATH not affected: `env-packages.sh` no longer appears as a callable command from PATH
- [ ] CHK-014 Package PATH setup preserved: Packages still added to PATH via the same mechanism from new location
- [ ] CHK-015 No-args listing excludes example: `example.yaml` is filtered from pipeline listing
- [ ] CHK-016 No pipelines found message: Proper error when `fab/pipelines/` has only `example.yaml` or is empty

## Scenario Coverage
- [ ] CHK-017 Scenario: env-packages.sh moved and no longer on PATH
- [ ] CHK-018 Scenario: KIT_DIR resolves correctly from new location
- [ ] CHK-019 Scenario: scaffold fragment sources from new path
- [ ] CHK-020 Scenario: rc-init.sh sources from new path
- [ ] CHK-021 Scenario: User invokes fab-pipeline.sh with manifest path
- [ ] CHK-022 Scenario: No arguments lists pipelines
- [ ] CHK-023 Scenario: Exact and partial match resolves
- [ ] CHK-024 Scenario: Ambiguous partial match errors
- [ ] CHK-025 Scenario: Short ID resolves to full change name via changeman

## Edge Cases & Error Handling
- [ ] CHK-026 fab-pipeline.sh no match: Prints error to stderr, exit 1
- [ ] CHK-027 fab-pipeline.sh ambiguous match: Lists matches to stderr, exit 1
- [ ] CHK-028 Resolution failure in run.sh: Change marked `invalid` in manifest on changeman resolve failure
- [ ] CHK-029 stageman unavailable: Graceful fallback or clear error in dispatch.sh

## Code Quality
- [ ] CHK-030 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [ ] CHK-031 No unnecessary duplication: Existing utilities (changeman, stageman) reused where applicable

## Documentation Accuracy
- [ ] CHK-032 kit-architecture.md: Directory tree and description sections reflect new layout
- [ ] CHK-033 distribution.md: All env-packages.sh path references point to new location
- [ ] CHK-034 README.md: Packages section references new env-packages.sh path
- [ ] CHK-035 pipeline-orchestrator.md: Documents fab-pipeline.sh, changeman resolve, --worktree-name, stageman detection

## Cross References
- [ ] CHK-036 Memory changelog entries: All modified memory files have changelog entries for this change
- [ ] CHK-037 No stale references: No remaining references to `fab/.kit/scripts/env-packages.sh` (old path) in codebase

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
