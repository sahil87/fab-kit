# Quality Checklist: Batch Pipeline Series & Rename

**Change**: 260222-bcfy-batch-pipeline-series-rename
**Generated**: 2026-02-22
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Rename: `batch-fab-pipeline.sh` no longer exists; `batch-pipeline.sh` exists with identical behavior
- [ ] CHK-002 Finite exit: `run.sh` exits 0 when all changes are terminal and `watch` is not `true`
- [ ] CHK-003 Watch mode: `run.sh` loops indefinitely when `watch: true` is in the manifest
- [ ] CHK-004 Local branch refs: `dispatch.sh` uses `git show-ref --verify --quiet refs/heads/` not `git ls-remote origin`
- [ ] CHK-005 Series script: `batch-pipeline-series.sh` generates valid manifest and delegates to `run.sh`
- [ ] CHK-006 Series base: `--base` flag overrides default; default is current branch
- [ ] CHK-007 fab-help.sh: `batch_to_group` contains `batch-pipeline` and `batch-pipeline-series`, not `batch-fab-pipeline`
- [ ] CHK-008 example.yaml: `watch` field documented in comments
- [ ] CHK-009 .gitignore: `fab/pipelines/.series-*.yaml` pattern present

## Behavioral Correctness
- [ ] CHK-010 Finite exit only when ALL changes terminal: pending/blocked changes prevent exit
- [ ] CHK-011 Watch field absent treated as false: manifests without `watch` field use finite mode
- [ ] CHK-012 Series minimum 2 changes: single argument prints usage and exits 1
- [ ] CHK-013 Series manifest not cleaned up: file persists after run.sh completes

## Removal Verification
- [ ] CHK-014 Deprecated `batch-fab-pipeline.sh` file does not exist after rename
- [ ] CHK-015 No `origin/` references remain in `dispatch.sh` `create_worktree()` for parent branch lookup

## Scenario Coverage
- [ ] CHK-016 Manifest with `watch: true` + all terminal → continues polling
- [ ] CHK-017 Manifest without `watch` + all terminal → exits 0 with summary
- [ ] CHK-018 Manifest without `watch` + mixed terminal/pending → continues polling
- [ ] CHK-019 Series with 3 changes → correct sequential dependency chain in manifest
- [ ] CHK-020 Series with `--base feat/setup` → manifest base field is `feat/setup`

## Edge Cases & Error Handling
- [ ] CHK-021 Series with 0 arguments → usage printed, exit 1
- [ ] CHK-022 Series with 1 argument → usage printed, exit 1
- [ ] CHK-023 Series `--help` → usage printed, exit 0
- [ ] CHK-024 Parent branch not local in dispatch → falls through to wt-create default (no error)

## Code Quality
- [ ] CHK-025 Pattern consistency: new code follows naming and structural patterns of surrounding pipeline scripts
- [ ] CHK-026 No unnecessary duplication: series delegates to run.sh instead of duplicating logic

## Documentation Accuracy
- [ ] CHK-027 pipeline-orchestrator.md: all `batch-fab-pipeline` references updated to `batch-pipeline`
- [ ] CHK-028 pipeline-orchestrator.md: `watch` field and finite-exit documented
- [ ] CHK-029 pipeline-orchestrator.md: `batch-pipeline-series.sh` documented
- [ ] CHK-030 kit-architecture.md: directory tree and section descriptions updated

## Cross References
- [ ] CHK-031 No stale `batch-fab-pipeline` references remain anywhere in the codebase (scripts, memory, specs)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
