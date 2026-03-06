# Intake: Regroup CLI Subcommands

**Change**: 260306-yzxj-regroup-cli-subcommands
**Created**: 2026-03-06
**Status**: Draft

## Origin

> Reorganize fab CLI commands: (A) regroup `_scripts.md` documentation by concern (Change Lifecycle, Pipeline & Status, Plumbing) and (B) move `fab archive` under `fab change` as subcommands (`fab change archive`, `fab change restore`, `fab change archive list`), updating all skill references and Go implementation accordingly.

This originated from a `/fab-discuss` session analyzing the current `_scripts.md` command surface. Three grouping options were explored (A: doc reorg only, B: consolidate under fewer top-level commands, C: porcelain/plumbing split). The user chose to combine A and B into a single change.

## Why

The current `_scripts.md` presents all 9 top-level commands as a flat list with no conceptual grouping. This makes it harder for agents (and humans reading the docs) to build a mental model of the CLI surface. Specifically:

1. **`archive` is a change lifecycle operation** but sits as a separate top-level command, divorced from `fab change` which owns the rest of the lifecycle (new, rename, switch, list). This is a UX inconsistency — archiving and restoring are the final steps of a change's life.
2. **No visible concern boundaries** — pipeline commands (`status`, `score`, `preflight`), lifecycle commands (`change`, `archive`), and internal plumbing (`resolve`, `log`, `runtime`) are all interleaved without structure.

If we don't fix this, the cognitive load compounds as more commands are added. The flat list becomes increasingly opaque.

## What Changes

### A. Regroup `_scripts.md` by concern

Reorganize the documentation into three sections:

#### Change Lifecycle
- `fab change` — new, rename, switch, list, archive, restore (after B is applied)

#### Pipeline & Status
- `fab status` — stage transitions, metadata, show
- `fab score` — confidence scoring
- `fab preflight` — validation + structured output

#### Plumbing
- `fab resolve` — change reference resolution
- `fab log` — append-only history logging
- `fab runtime` — runtime state management

### B. Move `archive` under `fab change`

Consolidate archive operations as subcommands of `fab change`:

| Current | New |
|---------|-----|
| `fab archive <change> --description "..."` | `fab change archive <change> --description "..."` |
| `fab archive restore <change> [--switch]` | `fab change restore <change> [--switch]` |
| `fab archive list` | `fab change archive-list` |

Go implementation changes:
- Move archive command registration from root to `change` parent command
- `restore` becomes a direct subcommand of `change` (not nested under `archive`) for ergonomics
- `archive-list` becomes a subcommand of `change` (hyphenated to avoid ambiguity with `fab change archive <change>`)
- Remove the top-level `archive` command

Skill and doc updates:
- Update `fab/.kit/skills/fab-archive.md` — all `fab archive` invocations become `fab change archive` / `fab change restore`
- Update `fab/.kit/skills/_scripts.md` — restructure and update command signatures
- Update `fab/.kit/skills/_preamble.md` — update any `fab archive` references in the state table or examples
- Update `docs/specs/skills/SPEC-fab-archive.md`
- Update `docs/memory/fab-workflow/kit-architecture.md` — command reference
- Update any other docs/memory files referencing `fab archive`

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update command reference table to reflect new grouping and `fab change archive`/`restore` subcommands

## Impact

- **Go binary**: `src/` command registration — archive moves under change parent
- **Skills**: `fab-archive.md`, `_scripts.md`, `_preamble.md` — command invocation paths change
- **Docs/specs**: `SPEC-fab-archive.md`, memory files — command references updated
- **Backward compatibility**: Any existing `fab archive` calls will break — this is acceptable since skills are the only callers and they ship in the same kit

## Open Questions

- Should `archive-list` use a hyphen (`fab change archive-list`) or be nested (`fab change archive --list`)? The intake assumes hyphenated subcommand for clarity, but flag-based could also work.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Regroup _scripts.md into three concern sections (Lifecycle, Pipeline, Plumbing) | Discussed — user explicitly chose combined A+B approach | S:95 R:90 A:90 D:95 |
| 2 | Certain | Move `fab archive` under `fab change` as subcommands | Discussed — user confirmed this is the one structural change worth making | S:95 R:70 A:85 D:90 |
| 3 | Certain | Keep `score` as a top-level command, not under `status` | Discussed — score has distinct flags/behavior and status already has 15+ subcommands | S:90 R:85 A:80 D:85 |
| 4 | Confident | `restore` as direct `fab change restore` rather than `fab change archive restore` | Flatter is more ergonomic; archive and restore are peer lifecycle operations | S:70 R:80 A:75 D:65 |
| 5 | Confident | Use `archive-list` hyphenated subcommand rather than flag-based `--list` | Consistent with Cobra subcommand pattern used elsewhere; avoids ambiguity with positional `<change>` arg | S:65 R:85 A:70 D:60 |
| 6 | Certain | No backward compatibility shim for old `fab archive` path | Skills are the only callers and ship in the same kit — atomic update | S:85 R:80 A:90 D:90 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
