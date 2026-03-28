# Intake: Standardize Tmux Tab Naming for Spawned Agents

**Change**: 260328-iqt8-standardize-tmux-tab-naming
**Created**: 2026-03-28
**Status**: Draft

## Origin

> Backlog [iqt8]: fab-operator7: Standardize tmux tab naming for spawned agents — use ⚡ followed by worktree name (e.g. ⚡fix-restart or ⚡ fix-restart if spaces allowed). Currently the skill says fab-<id> but at spawn time the change ID doesn't exist yet. The worktree name is always known and unique.

## Why

The current operator7 skill uses `fab-<id>` as the tmux tab name when spawning agents (`tmux new-window -n "fab-<id>" ...`). This has two problems:

1. **For new changes (from raw text or backlog)**, the change ID doesn't exist yet at spawn time — `fab-new` creates it *after* the agent starts. The skill already works around this inconsistently: the "From raw text" path uses `fab-<wt>` (line 373) while "From backlog" uses `fab-<id>` (line 382), creating an ambiguous naming situation.

2. **The worktree name is always available** at spawn time (it's created in step 1 of every path) and is unique across all panes. It's a more reliable and consistent identifier than the change ID.

Standardizing to `⚡{worktree-name}` gives a visually distinct, consistent naming scheme across all three spawn paths. The ⚡ prefix makes agent tabs immediately distinguishable from other tmux windows (like the operator tab itself).

## What Changes

### Update `fab/.kit/skills/fab-operator7.md`

Four `tmux new-window -n` invocations need their tab name argument updated:

1. **Line ~305** (generic "Open agent tab" reference):
   - From: `tmux new-window -n "fab-<id>" -c <worktree-path> ...`
   - To: `tmux new-window -n "⚡<wt>" -c <worktree-path> ...`

2. **Line ~365** ("From existing change" path):
   - From: `tmux new-window -n "fab-<id>" -c <worktree-path> ...`
   - To: `tmux new-window -n "⚡<wt>" -c <worktree-path> ...`

3. **Line ~382** ("From backlog ID or Linear issue" path):
   - From: `tmux new-window -n "fab-<id>" -c <worktree-path> ...`
   - To: `tmux new-window -n "⚡<wt>" -c <worktree-path> ...`

4. **Line ~373** ("From raw text" path) — already uses `<wt>`, just update the prefix:
   - From: `tmux new-window -n "fab-<wt>" -c <worktree-path> ...`
   - To: `tmux new-window -n "⚡<wt>" -c <worktree-path> ...`

Where `<wt>` is the worktree name (the `--worktree-name` value or auto-generated name from `wt create`).

### Update corresponding spec file

Per constitution: "Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md` file." Update the operator7 spec if it references tab naming.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator7 tab naming convention from `fab-<id>` to `⚡<wt>`

## Impact

- **`fab/.kit/skills/fab-operator7.md`** — primary change target (4 line edits)
- **`docs/specs/skills/`** — corresponding spec update if operator7 has one
- No code changes — this is a docs/skill-only change
- No migration needed — tab naming is ephemeral (tmux state, not persisted)

## Open Questions

- None — the backlog item is specific and the scope is narrow.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `⚡` (zap emoji) as prefix, not a text string | Backlog item explicitly specifies ⚡ | S:95 R:90 A:95 D:95 |
| 2 | Certain | Use worktree name (not change ID or slug) as the identifier | Backlog item explains worktree name is always available at spawn time | S:90 R:90 A:90 D:95 |
| 3 | Confident | No space between ⚡ and worktree name (e.g. `⚡swift-fox`) | Backlog mentions both forms; no-space is more compact and standard for emoji prefixes in terminal tabs | S:60 R:95 A:70 D:60 |
| 4 | Certain | All three spawn paths get the same naming pattern | Standardization is the explicit goal of the change | S:90 R:90 A:90 D:95 |
| 5 | Confident | No changes to operator6 — only operator7 | Backlog item specifically says "fab-operator7"; operator6 is the predecessor and may share the pattern but is not mentioned | S:70 R:85 A:70 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
