# Brief: Add dev folder and tests for _calc-score.sh

**Change**: 260214-mgh5-calc-score-dev-setup
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Check the setup for stageman and resolve-change (fab/.kit/scripts/*, src/*). Similarly create the folder and tests for _calc-score.sh.

## Why

`_stageman.sh` and `_resolve-change.sh` both have `src/` development folders with symlinks, READMEs, smoke tests, and comprehensive test suites. `_calc-score.sh` is the only internal library script without this setup, making it harder to develop and test in isolation.

## What Changes

- **Add** `src/calc-score/` directory following the established convention:
  - `_calc-score.sh` symlink to `../../fab/.kit/scripts/_calc-score.sh`
  - `README.md` documenting API, usage, testing, and changelog
  - `test-simple.sh` quick smoke test
  - `test.sh` comprehensive test suite covering:
    - Grade counting from Assumptions tables
    - Score formula computation
    - Carry-forward of implicit Certain counts
    - .status.yaml confidence block updates
    - Delta computation
    - Error cases (missing change-dir, missing spec.md)

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Add calc-score to the internal scripts inventory

## Impact

- `src/calc-score/` — new directory
- `fab/.kit/scripts/_calc-score.sh` — target of new symlink (no changes to the script itself)
- Follows identical pattern to `src/stageman/`, `src/resolve-change/`, `src/preflight/`

## Open Questions

None — the pattern is fully established by the three existing `src/` folders.
