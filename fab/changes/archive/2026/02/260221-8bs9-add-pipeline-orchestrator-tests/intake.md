# Intake: Add Pipeline Orchestrator Tests

**Change**: 260221-8bs9-add-pipeline-orchestrator-tests
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Add BATS test suite for the pipeline orchestrator scripts (run.sh, dispatch.sh). Tests should live in src/scripts/pipeline/ following the same pattern as existing lib tests in src/lib/stageman/test.bats and src/lib/changeman/test.bats. Start with unit tests for the pure-logic functions: validate_manifest, detect_cycles, is_dispatchable, find_next_dispatchable, get_parent_branch, poll_change state machine. Use YAML fixtures, mock external commands (tmux, claude, gh, stageman) via PATH override stubs, and temp directories for isolation.

Identified during debugging session where several orchestrator bugs were found (stdout pollution, stale worktree artifacts, false review-failure detection). No existing tests caught these.

## Why

The pipeline orchestrator (`run.sh` + `dispatch.sh`) has zero test coverage. Recent debugging found three bugs in a single session:

1. `dispatch.sh` `log()` writing to stdout, polluting the worktree-path + pane-id protocol
2. `provision_artifacts()` skipping copy when target dir exists (stale worktrees)
3. `poll_change()` treating `review:failed` as terminal (killing fab-ff rework loop)

All three would have been caught by basic unit tests. Meanwhile, the lib scripts (`stageman.sh`, `changeman.sh`, `calc-score.sh`, `preflight.sh`) all have comprehensive BATS suites. The pipeline scripts are the only untested shell code in the kit.

## What Changes

### New test directory: `src/scripts/pipeline/`

Create `src/scripts/pipeline/` following the established pattern where `src/` mirrors `fab/.kit/scripts/` structure:
- `src/lib/stageman/test.bats` tests `fab/.kit/scripts/lib/stageman.sh`
- `src/lib/changeman/test.bats` tests `fab/.kit/scripts/lib/changeman.sh`
- `src/scripts/pipeline/test.bats` will test `fab/.kit/scripts/pipeline/run.sh` and `dispatch.sh`

### Test file: `src/scripts/pipeline/test.bats`

#### Setup and fixtures

Follow the changeman pattern:
- `setup()` creates `TEST_DIR` with temp dirs, copies scripts under test, stubs external commands
- `teardown()` removes `TEST_DIR`
- YAML manifest fixtures created inline per test
- External command stubs (`tmux`, `claude`, `gh`, `wt-create`, `changeman`, `stageman`) in `$TEST_DIR/bin/` prepended to `$PATH`

#### Test categories

**1. `validate_manifest()` — pure YAML validation logic**

Tests with fixture manifests for:
- Valid manifest (base + changes with deps) passes
- Missing `base` field fails
- Missing `changes` array fails
- Empty changes array fails
- Circular dependency detected and rejected (A→B→A)
- Self-referencing dependency rejected
- Reference to nonexistent ID rejected
- Multi-dependency rejected (v1 single-dep constraint)
- Duplicate IDs rejected

**2. `detect_cycles()` / `dfs_visit()` — dependency graph**

- Simple chain (A→B→C) has no cycles
- Diamond (A→B, A→C, B→D, C→D) has no cycles
- Direct cycle (A→B→A) detected
- Indirect cycle (A→B→C→A) detected
- Isolated nodes (no deps) have no cycles

**3. `is_terminal()` / `is_dispatchable()` — stage predicates**

- `done`, `failed`, `invalid` are terminal
- `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate` are not terminal
- Absent stage is not terminal
- Dispatchable: non-terminal self + all deps `done`
- Not dispatchable: deps not `done`
- Not dispatchable: self already terminal

**4. `find_next_dispatchable()` — dispatch ordering**

- First non-terminal change with all deps done is selected
- List order preserved (deterministic serial dispatch)
- No dispatchable when all terminal
- No dispatchable when deps not met
- Skips terminal changes, finds first dispatchable

**5. `get_parent_branch()` — branch resolution**

- Root node (no deps) returns manifest `base`
- Dependent node returns parent's branch (with prefix)
- Branch prefix applied correctly

**6. `dispatch.sh` functions**

- `provision_artifacts()`: copies source to target, always syncs even if target exists
- `validate_prerequisites()`: fails without intake.md, fails without spec.md, passes with both + passing confidence gate

**7. `poll_change()` state machine** (mock-heavy)

- `hydrate:done` in progress map → marks `done`, sends ship command
- `review:failed` in progress map → does NOT mark failed (continues polling)
- Pane death → marks `failed`
- Timeout → marks `failed`
- Progress rendering outputs change ID and elapsed time

### Source functions for testability

`run.sh` and `dispatch.sh` define functions that are currently only callable by sourcing the entire script. The test file will source the scripts to access individual functions, using stubs to prevent `main()` from executing (the scripts check `[[ "${BASH_SOURCE[0]}" == "${0}" ]]` or similar, or the tests source with guards).

If the scripts don't have source guards, add minimal ones:
```bash
# At bottom of run.sh / dispatch.sh
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
```

This is a standard pattern that allows sourcing for testing without executing.

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Add "Testing" section documenting test location, coverage, and fixture patterns

## Impact

- New files: `src/scripts/pipeline/test.bats`
- Modified files: `fab/.kit/scripts/pipeline/run.sh` (source guard if needed), `fab/.kit/scripts/pipeline/dispatch.sh` (source guard if needed)
- No behavioral changes to the scripts themselves — only adding testability hooks and tests

## Open Questions

None — the test patterns are well-established in the codebase and the functions to test are clearly identified.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Test location at `src/scripts/pipeline/test.bats` | User explicitly specified `src/scripts/*` pattern; mirrors existing `src/lib/*` structure | S:95 R:95 A:95 D:95 |
| 2 | Certain | Use BATS framework with bats-assert/bats-support | All existing test suites use this; infrastructure already in place | S:90 R:95 A:95 D:95 |
| 3 | Certain | Mock external commands via PATH override stubs | Established pattern in changeman tests; proven approach for isolating shell scripts | S:90 R:90 A:95 D:90 |
| 4 | Confident | Add source guards to run.sh and dispatch.sh for testability | Standard bash testing pattern; minimal change; only if not already present | S:75 R:90 A:85 D:80 |
| 5 | Confident | Start with unit tests for pure-logic functions, defer full integration tests | User specified these functions; integration tests for main loops are high-effort, low-priority | S:85 R:85 A:80 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
