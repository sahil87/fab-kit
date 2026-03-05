# Quality Checklist: Add Pipeline Orchestrator Tests

**Change**: 260221-8bs9-add-pipeline-orchestrator-tests
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Source guards: run.sh has source guard preventing main() execution when sourced
- [ ] CHK-002 Source guards: dispatch.sh has source guard preventing main() execution when sourced
- [ ] CHK-003 Source guards: direct execution (`bash run.sh`) still invokes main()
- [ ] CHK-004 Test scaffold: test.bats exists at `src/scripts/pipeline/test.bats` with setup/teardown
- [ ] CHK-005 validate_manifest: all 7 scenarios from spec covered (valid, missing base, empty changes, missing id, missing depends_on, dangling ref, multi-dep)
- [ ] CHK-006 detect_cycles: all 4 scenarios covered (linear, direct cycle, indirect cycle, independent)
- [ ] CHK-007 is_terminal: terminal and non-terminal stages classified correctly
- [ ] CHK-008 is_dispatchable: all 4 scenarios covered
- [ ] CHK-009 find_next_dispatchable: all 3 scenarios covered
- [ ] CHK-010 get_parent_branch: root and dependent node scenarios covered
- [ ] CHK-011 provision_artifacts: all 3 scenarios covered (create, update stale, missing source)
- [ ] CHK-012 validate_prerequisites: all 3 scenarios covered (missing intake, missing spec, passing)

## Behavioral Correctness
- [ ] CHK-013 Source guards don't change script behavior when invoked directly
- [ ] CHK-014 Tests source run.sh/dispatch.sh without triggering main()

## Scenario Coverage
- [ ] CHK-015 All GIVEN/WHEN/THEN scenarios from spec have corresponding @test entries
- [ ] CHK-016 Test assertions match spec's expected outcomes (exit codes, stderr content)

## Edge Cases & Error Handling
- [ ] CHK-017 validate_manifest: multi-dep marks invalid but returns 0 (not rejection)
- [ ] CHK-018 is_dispatchable: empty stage (no stage set) treated as non-terminal
- [ ] CHK-019 validate_prerequisites: returns 2 (not 1) for prerequisite failures

## Code Quality
- [ ] CHK-020 Pattern consistency: test.bats follows same structure as src/lib/changeman/test.bats (REPO_ROOT, setup/teardown, helper functions, @test naming)
- [ ] CHK-021 No unnecessary duplication: YAML fixture creation uses helper function

## Documentation Accuracy
- [ ] CHK-022 Test location matches spec (`src/scripts/pipeline/test.bats`)

## Cross References
- [ ] CHK-023 Test function names reference the correct run.sh/dispatch.sh function names
