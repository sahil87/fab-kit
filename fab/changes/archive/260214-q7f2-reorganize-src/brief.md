# Brief: Reorganize src/ and kit script internals

**Change**: 260214-q7f2-reorganize-src
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Reorganize src/ to separate test infrastructure (lib/) from dev-only scripts (scripts/). Move fab-release.sh out of the shipped kit. Move internal kit library scripts from _ prefix convention to lib/ folder. Split from docs relocation (260214-m3v8).

## Why

Three organizational issues:
1. `src/` holds test infrastructure at the top level with no place for dev-only scripts
2. `fab-release.sh` ships in the tarball even though end users never need it
3. Internal library scripts in `fab/.kit/scripts/` use a `_` prefix convention to distinguish them from user-facing scripts — a `lib/` subfolder is clearer and matches the same pattern we're applying to `src/`

## What Changes

### A. Reorganize `src/`
- Move `src/{calc-score,preflight,resolve-change,stageman}/` to `src/lib/`
- Create `src/scripts/`, move `fab/.kit/scripts/fab-release.sh` into it
- Update `justfile` test glob from `src/*/test.sh` to `src/lib/*/test.sh`
- Fix symlinks in each `src/lib/*/` directory (depth adjustment + new target names)
- Add `src/scripts` to the repo `.envrc` PATH (not the scaffold `.envrc` that ships)

### B. Move kit internal scripts to `fab/.kit/scripts/lib/`
- `fab/.kit/scripts/_calc-score.sh` → `fab/.kit/scripts/lib/calc-score.sh`
- `fab/.kit/scripts/_preflight.sh` → `fab/.kit/scripts/lib/preflight.sh`
- `fab/.kit/scripts/_stageman.sh` → `fab/.kit/scripts/lib/stageman.sh`
- `fab/.kit/scripts/_resolve-change.sh` → `fab/.kit/scripts/lib/resolve-change.sh`
- `fab/.kit/scripts/_init_scaffold.sh` → `fab/.kit/scripts/lib/init-scaffold.sh`
- Drop `_` prefix — the `lib/` folder now signals "internal"

### B1. Update inter-script references
- `lib/preflight.sh` sources `lib/stageman.sh` and `lib/resolve-change.sh`
- `fab-upgrade.sh` calls `lib/init-scaffold.sh`
- `batch-archive-change.sh` sources `lib/resolve-change.sh`
- `batch-switch-change.sh` sources `lib/resolve-change.sh`

### B2. Update skill references
- `_context.md` — references to `_preflight.sh` and `_calc-score.sh` paths
- `fab-init.md` — references to `_init_scaffold.sh`
- `fab-status.md` — references to `_preflight.sh`
- `fab-archive.md` — references to `_preflight.sh`
- `fab-continue.md` — references to `_calc-score.sh`
- `fab-clarify.md` — references to `_calc-score.sh`

### B3. Update src/ symlinks
- Symlinks in `src/lib/*/` point to the renamed scripts in `fab/.kit/scripts/lib/`

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) src/ and .kit/scripts/ directory structure changes
- `fab-workflow/distribution`: (modify) tarball no longer includes fab-release.sh, lib/ subfolder added to .kit/scripts/
- `fab-workflow/preflight`: (modify) script path changes
- `fab-workflow/context-loading`: (modify) preflight invocation path changes
- `fab-workflow/planning-skills`: (modify) calc-score invocation path changes

## Impact

- **Justfile**: test glob changes from `src/*/test.sh` to `src/lib/*/test.sh`
- **Symlinks**: each `src/lib/*/` symlink needs depth adjustment + new target name
- **Tarball**: `fab-release.sh` no longer shipped; `lib/` subfolder added inside `.kit/scripts/`
- **PATH**: dev-only `.envrc` adds `src/scripts`; scaffold `.envrc` (shipped) unchanged
- **Skills**: 6 skill files reference internal script paths — all need updating
- **Scripts**: 4 scripts source/call internal scripts — all need path updates
- **Parallel-safe with docs relocation**: different path strings being changed (`_preflight.sh` vs `fab/memory/`), minor README overlap only
- **README**: update references to internal script paths

## Open Questions

None — decisions resolved during planning (batch scripts stay in .kit/scripts/).
