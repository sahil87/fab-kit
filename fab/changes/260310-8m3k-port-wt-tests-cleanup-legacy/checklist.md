# Quality Checklist: Port wt Tests & Cleanup Legacy

**Change**: 260310-8m3k-port-wt-tests-cleanup-legacy
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Init tests: Script execution, missing script, not-in-repo error all covered
- [ ] CHK-002 Create tests: Exploratory, branch-based, naming, collision, --reuse, porcelain output, branch-off-HEAD all covered
- [ ] CHK-003 Delete tests: By name, branch cleanup, --stash, --delete-all, nonexistent error all covered
- [ ] CHK-004 List tests: Formatted output, --path, --json, mutual exclusivity, dirty/unpushed all covered
- [ ] CHK-005 Open tests: By name, by path, nonexistent error, unknown app error all covered
- [ ] CHK-006 Edge case tests: Corrupted state, invalid branch, detached HEAD, multiple unique names covered
- [ ] CHK-007 Integration tests: Lifecycle, delete-all, automation workflow, git integrity covered

## Behavioral Correctness
- [ ] CHK-008 Tests validate Go CLI behavior (not shell behavior) — adapted for Cobra subcommand syntax
- [ ] CHK-009 Porcelain contract: create stdout outputs only path, messages go to stderr

## Removal Verification
- [ ] CHK-010 src/packages/ completely removed — no directory, no files
- [ ] CHK-011 src/tests/ completely removed
- [ ] CHK-012 .gitmodules removed, git submodule state cleaned up
- [ ] CHK-013 No dangling references to src/packages, src/tests, or bats remain in codebase

## Scenario Coverage
- [ ] CHK-014 `go test ./...` passes in src/wt-go/ with all ported tests
- [ ] CHK-015 wt pr tests explicitly noted as not ported (no Go implementation)

## Edge Cases & Error Handling
- [ ] CHK-016 Tests handle temp directory cleanup to avoid leaking git repos
- [ ] CHK-017 Tests work in CI (no interactive prompts, no real git remotes needed beyond local bare repos)

## Code Quality
- [ ] CHK-018 Pattern consistency: Test files follow naming and structure of existing *_test.go files in src/wt-go/
- [ ] CHK-019 No unnecessary duplication: Shared test utilities in testutil_test.go

## Documentation Accuracy
- [ ] CHK-020 distribution.md updated if it references src/packages

## Cross References
- [ ] CHK-021 No remaining references to removed paths in memory files or specs

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
