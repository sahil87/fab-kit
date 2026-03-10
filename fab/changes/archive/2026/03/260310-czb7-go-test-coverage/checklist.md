# Quality Checklist: Go Test Coverage and Backend Priority

**Change**: 260310-czb7-go-test-coverage
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Justfile targets: `test-go` and `test-go-v` targets exist and `test-go` is in the `test` recipe
- [ ] CHK-002 Backend priority: `fab` dispatcher checks `fab-go` before `fab-rust` in default priority
- [ ] CHK-003 resolve tests: `resolve_test.go` covers ToFolder, ExtractID, FabRoot, ToDir, ToStatus, ToAbsDir, ToAbsStatus
- [ ] CHK-004 log tests: `log_test.go` covers Command, Transition, Review, ConfidenceLog
- [ ] CHK-005 preflight tests: `preflight_test.go` covers Run (valid, missing config, missing constitution, missing change, override) and FormatYAML
- [ ] CHK-006 score tests: `score_test.go` covers Compute and CheckGate
- [ ] CHK-007 archive tests: `archive_test.go` covers Archive, Restore, List
- [ ] CHK-008 change tests: `change_test.go` covers New, Rename, Switch, SwitchBlank, List

## Behavioral Correctness

- [ ] CHK-009 Dispatcher priority: with both backends present and no override, `fab-go` is selected
- [ ] CHK-010 Dispatcher override: `FAB_BACKEND=rust` still selects rust backend
- [ ] CHK-011 Dispatcher version: `--version` detection order matches execution priority (go first)

## Scenario Coverage

- [ ] CHK-012 resolve: symlink resolution, 4-char ID, substring, full name, ambiguous error, no match error
- [ ] CHK-013 log: append-only behavior, optional fields omitted when empty, ISO 8601 timestamps
- [ ] CHK-014 preflight: YAML output structure matches expected format
- [ ] CHK-015 score: zero score when unresolved > 0, cover factor for thin specs, dimension parsing
- [ ] CHK-016 archive: move to archive/yyyy/mm/, index update, pointer clear, restore with switch
- [ ] CHK-017 change: slug validation regex, ID collision detection, rename updates symlink

## Edge Cases & Error Handling

- [ ] CHK-018 resolve: no `fab/changes/` directory returns appropriate error
- [ ] CHK-019 log: graceful handling when change dir doesn't exist
- [ ] CHK-020 score: handles missing Assumptions table gracefully
- [ ] CHK-021 change: rejects invalid slugs (spaces, special chars, leading/trailing hyphens)

## Code Quality

- [ ] CHK-022 Pattern consistency: test files follow existing patterns (t.TempDir, table-driven, t.Run, no assertion libs)
- [ ] CHK-023 No unnecessary duplication: shared fixture setup reused where applicable

## Documentation Accuracy

- [ ] CHK-024 Memory file `kit-architecture.md` updated with Go test strategy

## Cross References

- [ ] CHK-025 All 6 previously-untested packages show `ok` (not `[no test files]`) in `go test ./...` output

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
