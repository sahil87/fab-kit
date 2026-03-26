# Intake: Rename --blank to --none in fab-switch

**Change**: 260326-1tch-rename-blank-to-none
**Created**: 2026-03-26
**Status**: Draft

## Origin

> Change fab-switch --blank to fab-switch --none or just bare fab-switch. --blank is not easy to remember

Backlog item `[1tch]` from `fab/backlog.md`. One-shot request — the user finds `--blank` unintuitive and wants a more memorable flag name.

## Why

`--blank` is not a natural word for "deactivate the current change." Users have to pause and remember the flag name. `--none` reads naturally: "switch to none" — i.e., switch to no change. This is a small ergonomic improvement that reduces cognitive friction for a commonly used command.

If not fixed, users continue to fumble or look up the flag name each time they want to deactivate.

## What Changes

### Go CLI: `fab change switch` flag rename

In `src/go/fab/cmd/fab/change.go`:
- Rename the `--blank` flag to `--none`
- Update the flag description
- Update the error message (currently says `switch requires <name> or --blank`)

In `src/go/fab/internal/change/change.go`:
- Rename `SwitchBlank()` → `SwitchNone()`
- Update the "already blank" output string to "already deactivated" or similar

In `src/go/fab/internal/change/change_test.go`:
- Rename test functions and update expected output strings

In `src/go/fab/internal/archive/archive.go`:
- Update the call from `SwitchBlank()` to `SwitchNone()`

### Skill file: `fab/.kit/skills/fab-switch.md`

- Replace all `--blank` references with `--none`
- Update the heading: `# /fab-switch [change-name] [--none]`
- Update the deactivation flow section heading and command

### CLI reference: `fab/.kit/skills/_cli-fab.md`

- Update the `switch` row: `switch <name> | --none`

### Memory file: `docs/memory/fab-workflow/change-lifecycle.md`

- Update all `--blank` references to `--none`

### Spec file: `docs/specs/skills/SPEC-fab-switch.md`

- Update all `--blank` references to `--none`

### Backlog: `fab/backlog.md`

- Mark `[1tch]` as complete

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Update `--blank` → `--none` in the deactivation lifecycle and `/fab-switch` section

## Impact

- **Go source**: 4 files in `src/go/fab/` (cmd, internal/change, internal/archive)
- **Skill files**: `fab-switch.md`, `_cli-fab.md`
- **Docs**: `change-lifecycle.md` memory file, `SPEC-fab-switch.md` spec file
- **Constitution**: Change to `fab` CLI requires `_cli-fab.md` update (covered above) and test updates (covered above)
- **Archive files**: Not updated — they are historical records of past changes

## Open Questions

None — this is a straightforward rename.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename to `--none` (not bare `fab-switch`) | Bare `fab-switch` already lists changes — `--none` is the only option that doesn't conflict with existing behavior | S:80 R:90 A:95 D:95 |
| 2 | Certain | Rename Go function `SwitchBlank` → `SwitchNone` | Internal consistency — function name should match the flag it serves | S:90 R:95 A:90 D:95 |
| 3 | Certain | Update output strings ("already blank" → "already deactivated") | The word "blank" should not appear in user-facing output after the rename | S:85 R:95 A:90 D:90 |
| 4 | Certain | Do not update archive files | Archive is historical record — constitution says memory is post-implementation truth, archives are frozen | S:90 R:95 A:95 D:95 |
| 5 | Certain | Constitution requires test updates for CLI changes | `fab/project/constitution.md` VII — tests must be updated alongside CLI changes | S:95 R:90 A:95 D:95 |
| 6 | Certain | Constitution requires `_cli-fab.md` update for CLI changes | Additional Constraints section explicitly requires this | S:95 R:90 A:95 D:95 |

6 assumptions (6 certain, 0 confident, 0 tentative, 0 unresolved).
