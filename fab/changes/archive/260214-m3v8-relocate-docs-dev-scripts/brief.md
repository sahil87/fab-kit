# Brief: Relocate memory and specs to docs/

**Change**: 260214-m3v8-relocate-docs-dev-scripts
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Move memory/ and specs/ from fab/ to docs/. Split from a larger refactor — src/ reorganization is tracked separately.

## Why

`fab/` currently conflates three concerns: shipped kit (`.kit/`), workflow state (changes, config, constitution), and reference documentation (memory, specs). Moving memory and specs to `docs/` makes `fab/` focused on workflow machinery and puts documentation at a standard, discoverable location.

## What Changes

- Move `fab/memory/` to `docs/memory/`
- Move `fab/specs/` to `docs/specs/`
- Update all path references across skills, templates, scaffold, scripts, config, constitution, README, and memory/specs files themselves
- Update `_init_scaffold.sh` to create `docs/` structure instead of `fab/memory/` and `fab/specs/`
- Add a migration entry for existing users
- Archived changes (68 folders) are frozen records and will NOT be updated

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) directory structure changes — docs/ is a new top-level directory
- `fab-workflow/init`: (modify) _init_scaffold.sh creates docs/ instead of fab/memory/ and fab/specs/
- `fab-workflow/context-loading`: (modify) context loading paths change from fab/memory/ to docs/memory/
- `fab-workflow/specs-index`: (modify) path references update
- `fab-workflow/hydrate`: (modify) hydration targets change from fab/memory/ to docs/memory/
- `fab-workflow/hydrate-specs`: (modify) path references update

## Impact

- **Skills**: ~14 skill files reference `fab/memory/` or `fab/specs/` paths — all need updating
- **Templates**: brief.md and spec.md templates reference `fab/memory/` paths
- **Scripts**: _init_scaffold.sh, fab-help.sh, _stageman.sh need path updates
- **Scaffold**: _init_scaffold.sh must create `docs/` structure instead of `fab/memory/` and `fab/specs/`
- **Cross-links preserved**: Both directories move together under `docs/`, so relative links between memory and specs (e.g., `../memory/index.md`) remain valid
- **Constitution**: Principle II and VI reference `fab/memory/` and `fab/specs/` explicitly

## Open Questions

None — all decisions were resolved during planning discussion (docs/ over doc/).
