# Brief: Reorganize src/ into lib/ and scripts/

**Change**: 260214-q7f2-reorganize-src
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Reorganize src/ to separate test infrastructure (lib/) from dev-only scripts (scripts/). Move fab-release.sh out of the shipped kit. Split from docs relocation (260214-m3v8).

## Why

`src/` currently holds test infrastructure for deployed scripts (calc-score, preflight, etc.) at the top level, with no place for dev-only scripts. `fab-release.sh` ships in the tarball even though end users never need it. Creating `src/lib/` and `src/scripts/` gives each concern a clear home.

## What Changes

- Move `src/{calc-score,preflight,resolve-change,stageman}/` to `src/lib/`
- Create `src/scripts/`, move `fab/.kit/scripts/fab-release.sh` into it
- Update `justfile` test glob from `src/*/test.sh` to `src/lib/*/test.sh`
- Fix symlinks in each `src/lib/*/` directory (depth changes from `../../fab/.kit/scripts/` to `../../../fab/.kit/scripts/`)
- Add `src/scripts` to the repo `.envrc` PATH (not the scaffold `.envrc` that ships)
- Update README references

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) src/ directory structure changes
- `fab-workflow/distribution`: (modify) tarball no longer includes fab-release.sh

## Impact

- **Justfile**: test glob changes from `src/*/test.sh` to `src/lib/*/test.sh`
- **Symlinks**: each `src/lib/*/` script symlink needs depth adjustment (one extra `../`)
- **Tarball**: `fab-release.sh` no longer shipped since it moves out of `fab/.kit/scripts/`
- **PATH**: dev-only `.envrc` adds `src/scripts` to PATH; scaffold `.envrc` (shipped) unchanged
- **No dependency on docs relocation** — can be worked in parallel, only README has minor overlap

## Open Questions

None — decisions resolved during planning (batch scripts stay in .kit/scripts/).
